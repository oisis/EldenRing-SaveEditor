package deployment

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// The SSH transport of the deployment package.
//
// It authenticates with the configured private key only, verifies the host key
// under Trust On First Use, and replaces a target save through a staging file
// and the atomic POSIX rename extension. There is no password authentication,
// no agent fallback, no InsecureIgnoreHostKey and no fallback to a direct
// overwrite or to a remove-and-rename.

// The reasons a connection was refused before it was used for anything. They
// are sentinels rather than sentences: the interface decides what to ask, and
// the trust decision is always the user's explicit one.
var (
	// ErrHostKeyNotApproved reports a host whose key this configuration has
	// never approved. The observed fingerprint is recorded for the approval
	// dialog; nothing is trusted by making the connection.
	ErrHostKeyNotApproved = errors.New("the SSH host key of this target has not been approved yet")
	// ErrHostKeyChanged reports a host presenting a different key than the one
	// approved for its address. The connection is refused outright.
	ErrHostKeyChanged = errors.New("the SSH host key of this target changed since it was approved")
	// ErrAtomicReplaceUnsupported reports a server without the POSIX rename
	// extension. Section 6 of deployment.md has no fallback for it.
	ErrAtomicReplaceUnsupported = errors.New(
		"this SSH server cannot replace a file atomically, so the target save was left unchanged")
)

// posixRenameExtension is the only mechanism this driver replaces a file with.
const posixRenameExtension = "posix-rename@openssh.com"

// The clocks of this transport. Every one of them is a real limit: SFTP has no
// context-aware API, so a call this driver does not bound itself is a call that
// waits for ever on a server that stopped answering.
const (
	// connectTimeout bounds the handshake and the opening of the SFTP session.
	connectTimeout = 20 * time.Second
	// operationTimeout bounds one blocking transfer, metadata call or command
	// session when the caller supplied no deadline of its own.
	operationTimeout = 2 * time.Minute
	// statusCommandTimeout bounds the configured status command. It is short on
	// purpose: the status of the game is polled, and a command that hangs must
	// not hold an operation open.
	statusCommandTimeout = 15 * time.Second
	// verifyTimeout bounds the verification that follows a confirmed
	// replacement. That verification runs on a context of its own, so a caller
	// who cancelled earlier cannot skip it.
	verifyTimeout = 30 * time.Second
)

// HostKeys is the half of the deployment store the SSH transport needs: the
// approved fingerprint of an address, and somewhere to record the fingerprint a
// handshake actually presented.
//
// The driver never trusts a key itself. It only reports what it saw, so the
// approval the user gives afterwards is bound to a real observation instead of
// to a value some caller supplied.
type HostKeys interface {
	TrustedHostKey(address string) (string, bool, error)
	ObserveHostKey(address string, fingerprint string) error
}

// sshExec runs one configured command on the target and reports its exit code.
type sshExec func(ctx context.Context, command string) (int, error)

// sshDriver implements Driver over SSH and SFTP.
//
// It connects lazily: constructing a driver contacts nothing, so listing or
// editing a target never opens a connection in the background.
type sshDriver struct {
	target Target
	keys   HostKeys

	mutex  sync.Mutex
	client *ssh.Client
	files  *sftp.Client
	// connection is the raw transport under client and files. Closing it is the
	// only way to unblock an SFTP call the server never answers, which is what
	// makes cancelling a context actually cancel anything here.
	connection net.Conn
	// exec is replaced by a test. Production leaves it nil and runs the command
	// in a real SSH session.
	exec sshExec
	// rename is replaced by a test. Production leaves it nil and uses the SFTP
	// POSIX rename extension.
	rename func(files *sftp.Client, from string, to string) error
}

// watchConnection closes connection once ctx is done, until the returned stop
// disarms it.
//
// The disarm is what makes a normal release safe. Signalling the goroutine and
// cancelling the context both leave two ready channels behind, and a select
// between two ready channels picks either one; the flag below decides instead,
// and stop does not return until the goroutine is gone. After stop, nothing is
// left that could close a healthy transport.
func watchConnection(ctx context.Context, connection net.Conn) (stop func()) {
	var guard sync.Mutex
	disarmed := false
	finished := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		select {
		case <-ctx.Done():
			guard.Lock()
			defer guard.Unlock()
			if disarmed {
				return
			}
			_ = connection.Close()
		case <-finished:
		}
	}()
	return func() {
		guard.Lock()
		disarmed = true
		guard.Unlock()
		close(finished)
		<-stopped
	}
}

// bind bounds one blocking operation and ties its cancellation to the transport.
//
// The SFTP client offers no context-aware call, so a read, a write or a
// metadata request whose answer never arrives only returns when the connection
// under it is closed. Cancelling the returned context closes it. That ends this
// driver for good, which is correct: every operation builds its own driver and
// closes it when it is done.
//
// The limit always applies: a derived deadline expires at whichever of the
// caller's own deadline and this limit comes first.
//
// The returned release disarms the watch and waits for it to be gone before it
// returns, so a completed operation never damages the connection the next step
// needs and no goroutine outlives the call it was started for.
func (driver *sshDriver) bind(ctx context.Context, limit time.Duration) (context.Context, func()) {
	bounded, cancel := context.WithTimeout(ctx, limit)
	driver.mutex.Lock()
	connection := driver.connection
	driver.mutex.Unlock()
	if connection == nil {
		return bounded, cancel
	}
	stop := watchConnection(bounded, connection)
	return bounded, func() {
		stop()
		cancel()
	}
}

// renameOnTarget is the one atomic replacement mechanism, behind the seam a
// test replaces to reproduce a server that performed the rename and whose
// answer was then lost.
func (driver *sshDriver) renameOnTarget(files *sftp.Client, from string, to string) error {
	if driver.rename != nil {
		return driver.rename(files, from, to)
	}
	return files.PosixRename(from, to)
}

func (driver *sshDriver) Kind() Kind { return KindSSH }

// address is the "host:port" of this target, written so an IPv6 literal is
// bracketed. It is also the key the approved fingerprint is remembered under.
func (driver *sshDriver) address() string { return driver.target.Address() }

// connect opens the SSH connection and the SFTP session, once.
func (driver *sshDriver) connect(ctx context.Context) (*sftp.Client, error) {
	driver.mutex.Lock()
	defer driver.mutex.Unlock()
	if driver.files != nil {
		return driver.files, nil
	}
	if driver.keys == nil {
		return nil, errors.New("the deployment host key store is not available")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	material, err := os.ReadFile(driver.target.KeyPath)
	if err != nil {
		// The path is the user's own configuration, but the message stays free of
		// it: section 12 keeps host paths and key material out of every report.
		return nil, errors.New("the configured SSH private key could not be read")
	}
	signer, err := ssh.ParsePrivateKey(material)
	// The key material never leaves this function.
	for index := range material {
		material[index] = 0
	}
	if err != nil {
		return nil, errors.New(
			"the configured SSH private key could not be used; encrypted keys are not supported")
	}
	config := &ssh.ClientConfig{
		User:            driver.target.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: driver.hostKeyCallback(),
		Timeout:         connectTimeout,
	}
	dialer := net.Dialer{Timeout: connectTimeout}
	connection, err := dialer.DialContext(ctx, "tcp", driver.address())
	if err != nil {
		return nil, fmt.Errorf("the SSH target could not be reached: %w", err)
	}
	// The deadline covers the handshake and the opening of the SFTP session
	// alike. Both block on the server, and neither may wait for ever; it is only
	// cleared once the session is open, after which bind takes over.
	deadline := time.Now().Add(connectTimeout)
	if given, ok := ctx.Deadline(); ok && given.Before(deadline) {
		deadline = given
	}
	_ = connection.SetDeadline(deadline)
	// A cancellation during the handshake has to reach the transport too. The
	// driver does not hold the connection yet, so the watch is taken directly
	// here rather than waiting the deadline out.
	stopWatch := watchConnection(ctx, connection)
	handshake, channels, requests, err := ssh.NewClientConn(connection, driver.address(), config)
	if err != nil {
		stopWatch()
		_ = connection.Close()
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		// The host key callback's own refusal is the useful one; it is preserved
		// so the interface can offer the approval dialog.
		if errors.Is(err, ErrHostKeyNotApproved) || errors.Is(err, ErrHostKeyChanged) {
			return nil, err
		}
		return nil, errors.New("the SSH connection could not be established")
	}
	client := ssh.NewClient(handshake, channels, requests)
	files, err := sftp.NewClient(client)
	if err != nil {
		stopWatch()
		_ = connection.Close()
		_ = client.Close()
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, errors.New("the SSH server did not provide an SFTP session")
	}
	stopWatch()
	_ = connection.SetDeadline(time.Time{})
	driver.client = client
	driver.files = files
	driver.connection = connection
	return files, nil
}

// hostKeyCallback is the whole Trust On First Use rule.
//
// It records what the host actually presented, then compares it with what the
// user approved for this exact address. An unapproved host and a changed key
// are both refused; neither is ever trusted automatically.
func (driver *sshDriver) hostKeyCallback() ssh.HostKeyCallback {
	address := driver.address()
	keys := driver.keys
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		observed := ssh.FingerprintSHA256(key)
		if err := keys.ObserveHostKey(address, observed); err != nil {
			return err
		}
		approved, known, err := keys.TrustedHostKey(address)
		if err != nil {
			return err
		}
		if !known {
			return ErrHostKeyNotApproved
		}
		if approved != observed {
			return ErrHostKeyChanged
		}
		return nil
	}
}

func (driver *sshDriver) Test(ctx context.Context) error {
	files, err := driver.connect(ctx)
	if err != nil {
		return err
	}
	_, release := driver.bind(ctx, operationTimeout)
	defer release()
	directory := path.Dir(driver.target.SavePath)
	info, err := files.Stat(directory)
	if err != nil {
		return errors.New("the target directory is not reachable")
	}
	if !info.IsDir() {
		return errors.New("the target save path does not sit inside a directory")
	}
	probe, name, err := createUniqueRemoteFile(files, directory, ".saveforge-probe-")
	if err != nil {
		return errors.New("the target directory is not writable")
	}
	_ = probe.Close()
	return files.Remove(name)
}

func (driver *sshDriver) Exists(ctx context.Context, target string) (bool, error) {
	files, err := driver.connect(ctx)
	if err != nil {
		return false, err
	}
	_, release := driver.bind(ctx, operationTimeout)
	defer release()
	info, err := files.Lstat(target)
	switch {
	case err == nil:
		if !info.Mode().IsRegular() {
			return false, errors.New("the target path is not a regular file")
		}
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, errors.New("cannot inspect the target path")
	}
}

// CopyOnTarget duplicates a file inside the target. SFTP has no server-side
// copy this driver can rely on, so the bytes travel through this machine; they
// are never written to a durable local file and the result is verified by the
// caller before it is accepted as a backup.
func (driver *sshDriver) CopyOnTarget(ctx context.Context, source, destination string) error {
	files, err := driver.connect(ctx)
	if err != nil {
		return err
	}
	_, release := driver.bind(ctx, operationTimeout)
	defer release()
	reader, err := files.Open(source)
	if err != nil {
		return errors.New("cannot read the target file")
	}
	defer reader.Close()
	writer, err := files.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return errors.New("cannot create the backup on the target")
	}
	if _, err := io.Copy(writer, reader); err != nil {
		_ = writer.Close()
		_ = files.Remove(destination)
		return errors.New("cannot copy the file on the target")
	}
	if err := writer.Close(); err != nil {
		_ = files.Remove(destination)
		return errors.New("cannot close the backup on the target")
	}
	return nil
}

func (driver *sshDriver) FilesEqual(ctx context.Context, left, right string) (bool, error) {
	files, err := driver.connect(ctx)
	if err != nil {
		return false, err
	}
	_, release := driver.bind(ctx, operationTimeout)
	defer release()
	leftDigest, err := remoteDigest(files, left)
	if err != nil {
		return false, err
	}
	rightDigest, err := remoteDigest(files, right)
	if err != nil {
		return false, err
	}
	return leftDigest == rightDigest, nil
}

func (driver *sshDriver) Remove(ctx context.Context, target string) error {
	files, err := driver.connect(ctx)
	if err != nil {
		return err
	}
	_, release := driver.bind(ctx, operationTimeout)
	defer release()
	if err := files.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New("cannot remove the target file")
	}
	return nil
}

// ReplaceFromLocal uploads the prepared file and replaces the target with it.
func (driver *sshDriver) ReplaceFromLocal(
	ctx context.Context, localPath, targetPath string,
) (ReplacementResult, error) {
	files, err := driver.connect(ctx)
	if err != nil {
		return ReplacementResult{}, err
	}
	source, err := os.Open(localPath)
	if err != nil {
		return ReplacementResult{}, errors.New("cannot read the prepared save")
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return ReplacementResult{}, err
	}
	if !info.Mode().IsRegular() {
		return ReplacementResult{}, errors.New("the prepared save is not a regular file")
	}
	return driver.replace(ctx, files, source, info.Mode().Perm(), info.Size(), targetPath)
}

// ReplaceOnTarget replaces the target with a file that already lives on it.
func (driver *sshDriver) ReplaceOnTarget(
	ctx context.Context, sourcePath, targetPath string,
) (ReplacementResult, error) {
	files, err := driver.connect(ctx)
	if err != nil {
		return ReplacementResult{}, err
	}
	opening, release := driver.bind(ctx, operationTimeout)
	source, err := files.Open(sourcePath)
	if err != nil {
		release()
		if ctxErr := opening.Err(); ctxErr != nil {
			return ReplacementResult{}, ctxErr
		}
		return ReplacementResult{}, errors.New("cannot read the selected backup on the target")
	}
	defer source.Close()
	info, err := source.Stat()
	release()
	if err != nil {
		return ReplacementResult{}, err
	}
	return driver.replace(ctx, files, source, info.Mode().Perm(), info.Size(), targetPath)
}

// replace is the single implementation of the safe replacement of section 6.
//
// Everything before the rename leaves the existing target untouched, the rename
// is the one irreversible point, and the verification after it always runs so
// the caller learns the real state of the target.
func (driver *sshDriver) replace(
	ctx context.Context,
	files *sftp.Client,
	source io.Reader,
	permissions os.FileMode,
	expectedSize int64,
	targetPath string,
) (ReplacementResult, error) {
	if _, supported := files.HasExtension(posixRenameExtension); !supported {
		return ReplacementResult{}, ErrAtomicReplaceUnsupported
	}
	// Everything up to and including the rename request runs under the caller's
	// clock. The verification below deliberately does not.
	staging, releaseStaging := driver.bind(ctx, operationTimeout)
	if err := staging.Err(); err != nil {
		releaseStaging()
		return ReplacementResult{}, err
	}
	staged, stagedPath, err := createUniqueRemoteFile(
		files, path.Dir(targetPath), ".saveforge-incoming-")
	if err != nil {
		releaseStaging()
		return ReplacementResult{}, err
	}
	renamed := false
	defer func() {
		if !renamed {
			// Only this driver's own staging file is ever removed, and only under a
			// clock of its own so the cleanup cannot hang either.
			_, release := driver.bind(context.WithoutCancel(ctx), verifyTimeout)
			_ = files.Remove(stagedPath)
			release()
		}
	}()
	digest := sha256.New()
	written, err := io.Copy(io.MultiWriter(staged, digest), source)
	if err != nil {
		_ = staged.Close()
		releaseStaging()
		return ReplacementResult{}, errors.New("the save could not be transferred to the target")
	}
	// A server without fsync support is not a reason to refuse: the staged file
	// is verified by reading it back, which is the stronger check anyway.
	_ = staged.Chmod(permissions)
	_ = staged.Sync()
	if err := staged.Close(); err != nil {
		releaseStaging()
		return ReplacementResult{}, errors.New("the transferred file could not be closed")
	}
	expected := hex.EncodeToString(digest.Sum(nil))
	if expectedSize >= 0 && written != expectedSize {
		releaseStaging()
		return ReplacementResult{}, errors.New("the transferred file has the wrong size")
	}
	stagedInfo, err := files.Stat(stagedPath)
	if err != nil {
		releaseStaging()
		return ReplacementResult{}, errors.New("the transferred file could not be inspected")
	}
	if stagedInfo.Size() != written {
		releaseStaging()
		return ReplacementResult{}, errors.New("the transferred file has the wrong size on the target")
	}
	stagedDigest, err := remoteDigest(files, stagedPath)
	if err != nil {
		releaseStaging()
		return ReplacementResult{}, err
	}
	if stagedDigest != expected {
		releaseStaging()
		return ReplacementResult{}, errors.New("the transferred file does not match the prepared save")
	}
	if err := staging.Err(); err != nil {
		releaseStaging()
		return ReplacementResult{}, err
	}

	renameErr := driver.renameOnTarget(files, stagedPath, targetPath)
	// The clock the caller owns ends here. Whatever it does now cannot change
	// what the server already did with the rename request.
	releaseStaging()
	if renameErr != nil {
		var status *sftp.StatusError
		if errors.As(renameErr, &status) {
			// The server answered the rename request, and its answer was a refusal.
			// That is a certain negative: the target still holds what it held.
			return ReplacementResult{}, errors.New("the target save could not be replaced")
		}
		// The request was sent and no answer came back. A lost answer is not
		// evidence the server did nothing, so this is neither "unchanged" nor
		// "replaced" and it is never retried from here.
		renamed = true
		return ReplacementResult{Outcome: ReplacementUndetermined}, errors.New(
			"the replacement of the target save was requested and its result is unknown")
	}
	renamed = true

	// The replacement happened, so its verification is owed to the user whatever
	// the caller's context now says. It runs detached and on its own clock: a
	// context cancelled earlier must not turn a confirmed replacement into one
	// that was never even looked at.
	_, releaseVerify := driver.bind(context.WithoutCancel(ctx), verifyTimeout)
	finalDigest, digestErr := remoteDigest(files, targetPath)
	releaseVerify()
	if digestErr != nil {
		// The replacement is done and the connection or the server failed after
		// it. The caller is told the target was replaced but not verified.
		return ReplacementResult{Outcome: ReplacementPerformed}, errors.New(
			"the replaced target save could not be verified")
	}
	if finalDigest != expected {
		return ReplacementResult{Outcome: ReplacementPerformed}, errors.New(
			"the replaced target save does not match the prepared save")
	}
	return ReplacementResult{Outcome: ReplacementPerformed, Verified: true}, nil
}

func (driver *sshDriver) CopyToLocal(ctx context.Context, targetPath, localPath string) error {
	files, err := driver.connect(ctx)
	if err != nil {
		return err
	}
	_, release := driver.bind(ctx, operationTimeout)
	defer release()
	reader, err := files.Open(targetPath)
	if err != nil {
		return errors.New("cannot read the target save")
	}
	defer reader.Close()
	writer, err := os.OpenFile(localPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("cannot create the local file: %w", err)
	}
	if _, err := io.Copy(writer, reader); err != nil {
		_ = writer.Close()
		_ = os.Remove(localPath)
		return errors.New("cannot download the target save")
	}
	if err := writer.Sync(); err != nil {
		_ = writer.Close()
		_ = os.Remove(localPath)
		return err
	}
	return writer.Close()
}

func (driver *sshDriver) RunStart(ctx context.Context) (CommandOutcome, error) {
	return driver.run(ctx, driver.target.StartCommand, "start command")
}

func (driver *sshDriver) RunStop(ctx context.Context) (CommandOutcome, error) {
	return driver.run(ctx, driver.target.StopCommand, "stop command")
}

// GameStatus runs the target's configured status command and maps its exit code
// onto the three states. Nothing else is consulted: this driver does not guess
// from a process list, from the start command or from the save's timestamp.
func (driver *sshDriver) GameStatus(ctx context.Context) (GameStatus, error) {
	// The status command gets a clock of its own. A command that never returns
	// is an unknown state, not an operation that hangs.
	bounded, cancel := context.WithTimeout(ctx, statusCommandTimeout)
	defer cancel()
	outcome, err := driver.run(bounded, driver.target.StatusCommand, "status command")
	return interpretGameStatus(outcome, err)
}

// run executes exactly the configured command line. A non-zero exit is an
// outcome, not an error: "the stop command found no process" has to be visible.
func (driver *sshDriver) run(
	ctx context.Context, command string, field string,
) (CommandOutcome, error) {
	if command == "" {
		return CommandOutcome{Configured: false, Detail: "no " + field + " is configured"}, nil
	}
	if err := ctx.Err(); err != nil {
		return CommandOutcome{}, err
	}
	execute := driver.exec
	if execute == nil {
		if _, err := driver.connect(ctx); err != nil {
			return CommandOutcome{}, err
		}
		execute = driver.runInSession
	}
	code, err := execute(ctx, command)
	if err != nil {
		return CommandOutcome{Configured: true}, fmt.Errorf("the %s could not be run", field)
	}
	outcome := CommandOutcome{Configured: true, Executed: true, ExitCode: code}
	if code != 0 {
		// The command's own output is never carried: it is the target machine's
		// and can contain paths, user names or secrets.
		outcome.Detail = fmt.Sprintf("the %s ended with exit code %d", field, code)
	}
	return outcome, nil
}

// runInSession is the production command runner.
//
// Opening the session and running the command both block on the server, so both
// run under bind: cancelling the context closes the transport, which is what
// makes them return. Nothing is left running behind this function, and no
// goroutine outlives it.
func (driver *sshDriver) runInSession(ctx context.Context, command string) (int, error) {
	driver.mutex.Lock()
	client := driver.client
	driver.mutex.Unlock()
	if client == nil {
		return 0, errors.New("the SSH connection is not open")
	}
	bounded, release := driver.bind(ctx, operationTimeout)
	defer release()
	session, err := client.NewSession()
	if err != nil {
		if ctxErr := bounded.Err(); ctxErr != nil {
			return 0, ctxErr
		}
		return 0, err
	}
	defer session.Close()
	err = session.Run(command)
	if err == nil {
		return 0, nil
	}
	var exitError *ssh.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitStatus(), nil
	}
	if ctxErr := bounded.Err(); ctxErr != nil {
		return 0, ctxErr
	}
	return 0, err
}

// WaitForStableSave waits until the size and modification time of the target
// save are unchanged across several consecutive observations.
func (driver *sshDriver) WaitForStableSave(ctx context.Context, target string) error {
	files, err := driver.connect(ctx)
	if err != nil {
		return err
	}
	_, release := driver.bind(ctx, operationTimeout)
	defer release()
	previous := ""
	stable := 0
	for stable < stabilisationRounds {
		if err := ctx.Err(); err != nil {
			return err
		}
		current := "absent"
		info, statErr := files.Stat(target)
		switch {
		case statErr == nil:
			current = fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
		case errors.Is(statErr, os.ErrNotExist):
		default:
			return errors.New("cannot observe the target save")
		}
		if current == previous {
			stable++
		} else {
			stable = 1
			previous = current
		}
		if stable < stabilisationRounds {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(stabilisationPoll):
			}
		}
	}
	return nil
}

// Close ends the connection without being able to hang on it.
//
// The raw transport is closed first, so neither of the two protocol closes
// below can block writing a farewell to a server that stopped answering. A
// socket that could not be closed is not information anyone can act on, so
// nothing is reported from here.
func (driver *sshDriver) Close() error {
	driver.mutex.Lock()
	files, client, connection := driver.files, driver.client, driver.connection
	driver.files, driver.client, driver.connection = nil, nil, nil
	driver.mutex.Unlock()
	if connection != nil {
		_ = connection.Close()
	}
	if files != nil {
		_ = files.Close()
	}
	if client != nil {
		_ = client.Close()
	}
	return nil
}

// createUniqueRemoteFile creates a file the driver owns, never overwriting an
// existing one. The exclusive create is what makes a leftover staging file from
// another run impossible to clobber.
func createUniqueRemoteFile(
	files *sftp.Client, directory string, prefix string,
) (*sftp.File, string, error) {
	for attempt := 0; attempt < 16; attempt++ {
		suffix := make([]byte, 8)
		if _, err := rand.Read(suffix); err != nil {
			return nil, "", err
		}
		name := path.Join(directory, prefix+hex.EncodeToString(suffix))
		file, err := files.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
		if err == nil {
			return file, name, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, "", errors.New("cannot create the target staging file")
		}
	}
	return nil, "", errors.New("cannot allocate a unique staging file on the target")
}

func remoteDigest(files *sftp.Client, target string) (string, error) {
	file, err := files.Open(target)
	if err != nil {
		return "", errors.New("cannot verify the target file")
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", errors.New("cannot verify the target file")
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
