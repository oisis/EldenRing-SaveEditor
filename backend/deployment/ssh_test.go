package deployment

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// The tests in this file use generated keys, an in-process SFTP server over a
// pair of pipes, and temporary directories only. No network connection, no real
// host, no user key and no real save takes part in any of them.

func generatedHostKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	key, err := ssh.NewPublicKey(public)
	if err != nil {
		t.Fatalf("wrap key: %v", err)
	}
	return key
}

// TestHostKeyTrustOnFirstUseBindsApprovalToTheObservedKey walks the whole TOFU
// rule: the first handshake is refused and records what it saw, only that
// observation can be approved, an approved key connects, and a changed key is
// refused again without ever being trusted automatically.
func TestHostKeyTrustOnFirstUseBindsApprovalToTheObservedKey(t *testing.T) {
	store := NewStore(t.TempDir())
	target := Target{
		Name: "Remote", Kind: KindSSH, SavePath: "/home/deck/ER0000.sl2",
		Host: "192.0.2.1", Port: 2222, User: "deck", KeyPath: "/does/not/matter",
	}
	driver := &sshDriver{target: target, keys: store}
	callback := driver.hostKeyCallback()
	first := generatedHostKey(t)

	if err := callback("192.0.2.1:2222", nil, first); err != ErrHostKeyNotApproved {
		t.Fatalf("first handshake = %v, want the unapproved refusal", err)
	}
	observed, seen := store.ObservedHostKey(target.Address())
	if !seen || observed != ssh.FingerprintSHA256(first) {
		t.Fatalf("observed = %q, %v; want the fingerprint the host presented", observed, seen)
	}

	// Nothing but the observation can be approved, and only for this address.
	if err := store.TrustHostKey(target.Address(), "SHA256:invented"); err == nil {
		t.Fatal("an invented fingerprint was approved")
	}
	if err := store.TrustHostKey("192.0.2.9:2222", observed); err == nil {
		t.Fatal("a fingerprint was approved for an address it was never seen on")
	}
	if err := store.TrustHostKey(target.Address(), observed); err != nil {
		t.Fatalf("TrustHostKey: %v", err)
	}
	if err := callback("192.0.2.1:2222", nil, first); err != nil {
		t.Fatalf("handshake with the approved key = %v, want it accepted", err)
	}

	second := generatedHostKey(t)
	if err := callback("192.0.2.1:2222", nil, second); err != ErrHostKeyChanged {
		t.Fatalf("handshake with a changed key = %v, want the changed-key refusal", err)
	}
	stored, known, err := store.TrustedHostKey(target.Address())
	if err != nil || !known || stored != observed {
		t.Fatalf("approved key after the change = %q, %v, %v; want the original one still", stored, known, err)
	}
}

// TestIPv6TargetsAreAddressedWithBrackets: the address is both what is dialled
// and the key the approved fingerprint is remembered under, so an IPv6 literal
// has to be written unambiguously.
func TestIPv6TargetsAreAddressedWithBrackets(t *testing.T) {
	target := Target{Kind: KindSSH, Host: "2001:db8::1", Port: 2222}
	if got := target.Address(); got != "[2001:db8::1]:2222" {
		t.Fatalf("Address() = %q, want the bracketed IPv6 address", got)
	}
	if got := (Target{Kind: KindSSH, Host: "deck.local"}).Address(); got != "deck.local:22" {
		t.Fatalf("Address() = %q, want the default port", got)
	}
}

// TestSSHTargetRefusesBeforeAnyNetworkWhenTheKeyIsUnusable: authentication is
// key-only, so a missing or unusable key stops the operation at configuration
// level rather than falling back to an agent or a password.
func TestSSHTargetRefusesBeforeAnyNetworkWhenTheKeyIsUnusable(t *testing.T) {
	directory := t.TempDir()
	missing := filepath.Join(directory, "absent-key")
	driver := &sshDriver{
		target: Target{
			Kind: KindSSH, SavePath: "/home/deck/ER0000.sl2",
			Host: "192.0.2.1", Port: 22, User: "deck", KeyPath: missing,
		},
		keys: NewStore(directory),
	}
	if _, err := driver.connect(context.Background()); err == nil {
		t.Fatal("a target whose key file does not exist opened a connection")
	}

	garbage := filepath.Join(directory, "not-a-key")
	if err := os.WriteFile(garbage, []byte("this is not a private key"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	driver.target.KeyPath = garbage
	if _, err := driver.connect(context.Background()); err == nil {
		t.Fatal("a target whose key file is not a key opened a connection")
	}
}

// sftpPair connects an SFTP client to an in-process server over two pipes. The
// server serves the real filesystem, so the test drives the same client code
// production uses without any network in between.
func sftpPair(t *testing.T) *sftp.Client {
	t.Helper()
	clientReader, serverWriter := io.Pipe()
	serverReader, clientWriter := io.Pipe()
	server, err := sftp.NewServer(struct {
		io.Reader
		io.WriteCloser
	}{serverReader, serverWriter})
	if err != nil {
		t.Fatalf("sftp.NewServer: %v", err)
	}
	go func() { _ = server.Serve() }()
	client, err := sftp.NewClientPipe(clientReader, clientWriter)
	if err != nil {
		t.Fatalf("sftp.NewClientPipe: %v", err)
	}
	// Teardown order matters: both pipe writers are closed first so the reader on
	// each side sees EOF, otherwise the client's receive goroutine blocks for ever
	// and Close never returns.
	t.Cleanup(func() {
		_ = clientWriter.Close()
		_ = serverWriter.Close()
		_ = client.Close()
		_ = server.Close()
	})
	return client
}

// TestSSHReplaceStagesVerifiesAndRenamesAtomically is the whole replacement
// contract of section 6 over the real SFTP client: the prepared file is staged
// beside the target, verified, renamed atomically, verified again, and no
// staging file is left behind. A failure before the rename leaves the existing
// target exactly as it was.
func TestSSHReplaceStagesVerifiesAndRenamesAtomically(t *testing.T) {
	directory := t.TempDir()
	targetPath := filepath.Join(directory, "ER0000.sl2")
	if err := os.WriteFile(targetPath, []byte("the existing target save"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	prepared := filepath.Join(t.TempDir(), "prepared.sl2")
	if err := os.WriteFile(prepared, []byte("the prepared save"), 0o600); err != nil {
		t.Fatalf("write prepared: %v", err)
	}
	driver := &sshDriver{
		target: Target{Kind: KindSSH, SavePath: targetPath},
		files:  sftpPair(t),
	}

	result, err := driver.ReplaceFromLocal(context.Background(), prepared, targetPath)
	if err != nil {
		t.Fatalf("ReplaceFromLocal: %v", err)
	}
	if result.Outcome != ReplacementPerformed || !result.Verified {
		t.Fatalf("result = %+v, want a performed and verified replacement", result)
	}
	written, err := os.ReadFile(targetPath)
	if err != nil || string(written) != "the prepared save" {
		t.Fatalf("target = %q, %v; want the prepared save", written, err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != "ER0000.sl2" {
			t.Fatalf("the target directory also holds %q; the staging file was not cleaned up", entry.Name())
		}
	}

	// A failure before the rename: the staging file cannot be created because the
	// directory is not writable, and the existing target is untouched.
	if err := os.Chmod(directory, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(directory, 0o700) })
	blocked, err := driver.ReplaceFromLocal(context.Background(), prepared, targetPath)
	if err == nil {
		t.Fatal("ReplaceFromLocal succeeded although the target directory is not writable")
	}
	if blocked.Outcome != ReplacementNotPerformed || blocked.Verified {
		t.Fatalf("result = %+v, want nothing performed before the replacement point", blocked)
	}
	after, err := os.ReadFile(targetPath)
	if err != nil || string(after) != "the prepared save" {
		t.Fatalf("target = %q, %v; a failed replacement changed it", after, err)
	}
}

// TestSSHGameStatusMapsTheConfiguredCommandExitCode: the exit code is the whole
// contract, and everything outside it is unknown rather than a guess.
func TestSSHGameStatusMapsTheConfiguredCommandExitCode(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		command string
		code    int
		failure error
		want    GameStatus
	}{
		{"no command configured", "", 0, nil, GameUnknown},
		{"exit zero is running", "pgrep game", 0, nil, GameRunning},
		{"exit one is stopped", "pgrep game", 1, nil, GameStopped},
		{"any other exit code is unknown", "pgrep game", 7, nil, GameUnknown},
		{"a command that cannot run is unknown", "pgrep game", 0, io.ErrUnexpectedEOF, GameUnknown},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			driver := &sshDriver{
				target: Target{Kind: KindSSH, StatusCommand: testCase.command},
				exec: func(context.Context, string) (int, error) {
					return testCase.code, testCase.failure
				},
			}
			status, err := driver.GameStatus(context.Background())
			if err != nil {
				t.Fatalf("GameStatus: %v", err)
			}
			if status != testCase.want {
				t.Fatalf("status = %q, want %q", status, testCase.want)
			}
		})
	}
}

// TestReplacementWhoseAnswerIsLostIsUndetermined is the case that a lost answer
// is not evidence of a target left alone: the server performs the rename and
// the reply never arrives.
//
// A server that answers the request with a refusal is the opposite case and
// stays a certain negative, so the two are asserted together.
func TestReplacementWhoseAnswerIsLostIsUndetermined(t *testing.T) {
	setUp := func(t *testing.T) (*sshDriver, string, string) {
		t.Helper()
		directory := t.TempDir()
		targetPath := filepath.Join(directory, "ER0000.sl2")
		if err := os.WriteFile(targetPath, []byte("the existing target save"), 0o600); err != nil {
			t.Fatalf("write target: %v", err)
		}
		prepared := filepath.Join(t.TempDir(), "prepared.sl2")
		if err := os.WriteFile(prepared, []byte("the prepared save"), 0o600); err != nil {
			t.Fatalf("write prepared: %v", err)
		}
		return &sshDriver{
			target: Target{Kind: KindSSH, SavePath: targetPath},
			files:  sftpPair(t),
		}, prepared, targetPath
	}

	t.Run("the server renamed and the answer was lost", func(t *testing.T) {
		driver, prepared, targetPath := setUp(t)
		driver.rename = func(_ *sftp.Client, from, to string) error {
			// The server did the work; only its reply is gone.
			if err := os.Rename(from, to); err != nil {
				t.Fatalf("rename: %v", err)
			}
			return io.ErrUnexpectedEOF
		}

		result, err := driver.ReplaceFromLocal(context.Background(), prepared, targetPath)
		if err == nil {
			t.Fatal("a lost answer was reported as a successful replacement")
		}
		if result.Outcome != ReplacementUndetermined {
			t.Fatalf("outcome = %q, want the undetermined replacement", result.Outcome)
		}
		if result.Verified {
			t.Fatal("an undetermined replacement was reported as verified")
		}
		// The target really did change, which is exactly why this outcome may not
		// be reported as "the target was left unchanged".
		if got := readFile(t, targetPath); got != "the prepared save" {
			t.Fatalf("target = %q, want the prepared save the server actually wrote", got)
		}
	})

	t.Run("the server refused the rename", func(t *testing.T) {
		driver, prepared, targetPath := setUp(t)
		driver.rename = func(*sftp.Client, string, string) error {
			// SSH_FX_FAILURE: the server answered, and its answer was no.
			return &sftp.StatusError{Code: 4}
		}

		result, err := driver.ReplaceFromLocal(context.Background(), prepared, targetPath)
		if err == nil {
			t.Fatal("a refused rename was reported as a successful replacement")
		}
		if result.Outcome != ReplacementNotPerformed {
			t.Fatalf("outcome = %q, want the certain negative", result.Outcome)
		}
		if got := readFile(t, targetPath); got != "the existing target save" {
			t.Fatalf("target = %q, want the existing save a refused rename leaves alone", got)
		}
	})
}

// mutedConn is a transport whose writes stop arriving after the handshake. It
// is what a server that goes silent mid-operation looks like from this side:
// the request is sent and no answer ever comes.
type mutedConn struct {
	net.Conn
	muted    chan struct{}
	released chan struct{}
}

func (conn *mutedConn) Write(payload []byte) (int, error) {
	select {
	case <-conn.muted:
		<-conn.released
		return 0, io.ErrClosedPipe
	default:
	}
	return conn.Conn.Write(payload)
}

// TestBlockedSFTPCallEndsWhenTheContextIsCancelled: checking ctx.Err() before a
// blocking call proves nothing, because the call blocks afterwards. Cancelling
// has to reach the transport itself.
func TestBlockedSFTPCallEndsWhenTheContextIsCancelled(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	muted := &mutedConn{Conn: serverSide, muted: make(chan struct{}), released: make(chan struct{})}
	server, err := sftp.NewServer(muted)
	if err != nil {
		t.Fatalf("sftp.NewServer: %v", err)
	}
	served := make(chan struct{})
	go func() {
		defer close(served)
		_ = server.Serve()
	}()
	client, err := sftp.NewClientPipe(clientSide, clientSide)
	if err != nil {
		t.Fatalf("sftp.NewClientPipe: %v", err)
	}
	// The handshake is done; from here the server receives requests and answers
	// none of them.
	close(muted.muted)
	t.Cleanup(func() {
		close(muted.released)
		_ = client.Close()
		_ = server.Close()
		_ = clientSide.Close()
		<-served
	})

	driver := &sshDriver{
		target:     Target{Kind: KindSSH, SavePath: "/home/deck/ER0000.sl2"},
		files:      client,
		connection: clientSide,
	}
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan error, 1)
	go func() {
		_, existsErr := driver.Exists(ctx, "/home/deck/ER0000.sl2")
		finished <- existsErr
	}()
	// The call is blocked on an answer that never comes. Cancelling is the only
	// thing that can end it.
	cancel()
	select {
	case err := <-finished:
		if err == nil {
			t.Fatal("a call the server never answered reported success")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the cancelled call is still blocked; cancellation did not reach the transport")
	}
}

// TestReleasingABoundOperationLeavesTheTransportOpen: a normal release disarms
// the watch instead of racing it. The loop is not a search for a rare timing:
// each release makes both the signal and the cancelled context ready at once,
// which is exactly the state a select cannot be trusted to resolve.
func TestReleasingABoundOperationLeavesTheTransportOpen(t *testing.T) {
	directory := t.TempDir()
	targetPath := filepath.Join(directory, "ER0000.sl2")
	if err := os.WriteFile(targetPath, []byte("the existing target save"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	clientSide, serverSide := net.Pipe()
	server, err := sftp.NewServer(serverSide)
	if err != nil {
		t.Fatalf("sftp.NewServer: %v", err)
	}
	served := make(chan struct{})
	go func() {
		defer close(served)
		_ = server.Serve()
	}()
	client, err := sftp.NewClientPipe(clientSide, clientSide)
	if err != nil {
		t.Fatalf("sftp.NewClientPipe: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
		_ = clientSide.Close()
		<-served
	})
	driver := &sshDriver{
		target:     Target{Kind: KindSSH, SavePath: targetPath},
		files:      client,
		connection: clientSide,
	}

	for attempt := 0; attempt < 20; attempt++ {
		_, release := driver.bind(context.Background(), operationTimeout)
		release()
	}

	exists, err := driver.Exists(context.Background(), targetPath)
	if err != nil || !exists {
		t.Fatalf("Exists = %v, %v; a completed operation closed the transport it released", exists, err)
	}
}
