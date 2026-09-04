package deployment

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/hostsettings"
)

// The tests in this file use a local target inside t.TempDir() only. No real
// save, no real deployment target and no network or SSH connection takes part
// in any of them.

func newTestService(t *testing.T) (*Service, *Store, Target, string) {
	t.Helper()
	state := t.TempDir()
	targetDirectory := t.TempDir()
	savePath := filepath.Join(targetDirectory, "ER0000.sl2")

	store := NewStore(state)
	target, err := store.CreateTarget(Target{
		Name:     "Local test target",
		Kind:     KindLocal,
		SavePath: savePath,
	})
	if err != nil {
		t.Fatalf("CreateTarget: %v", err)
	}
	settings := hostsettings.NewStore(state)
	return NewService(store, settings, nil), store, target, savePath
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// TestUploadBlocksOnUnknownStatusBacksUpAndReplacesAtomically walks the whole
// upload contract in one flow: the unknown game state blocks until the user
// says otherwise, the existing target save is backed up before it is replaced,
// and the replacement leaves the target holding exactly the prepared bytes.
func TestUploadBlocksOnUnknownStatusBacksUpAndReplacesAtomically(t *testing.T) {
	service, store, target, savePath := newTestService(t)
	writeFile(t, savePath, "the existing target save")
	prepared := filepath.Join(t.TempDir(), "prepared.sl2")
	writeFile(t, prepared, "the prepared save")

	request := OperationRequest{
		OperationID:  "operation-1",
		TargetID:     target.ID,
		PreparedPath: prepared,
	}

	blocked, err := service.Upload(context.Background(), request, false)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if blocked.Blocked != BlockedGameStatusUnknown || blocked.Completed || blocked.TargetState != TargetStateUnchanged {
		t.Fatalf("first upload = %+v, want a block on the unknown game state", blocked)
	}
	if readFile(t, savePath) != "the existing target save" {
		t.Fatal("a blocked upload changed the target save")
	}

	request.ContinueWithUnknownGameStatus = true
	confirmable, err := service.Upload(context.Background(), request, false)
	if err != nil {
		t.Fatalf("Upload after continuing: %v", err)
	}
	if confirmable.Blocked != BlockedRemoteBackupConfirmation || confirmable.Completed {
		t.Fatalf("second upload = %+v, want a block on the mandatory backup", confirmable)
	}
	if readFile(t, savePath) != "the existing target save" {
		t.Fatal("an upload blocked on the backup confirmation changed the target save")
	}

	request.ConfirmRemoteBackup = true
	done, err := service.Upload(context.Background(), request, false)
	if err != nil {
		t.Fatalf("Upload after confirming the backup: %v", err)
	}
	if !done.Completed || done.Blocked != "" || done.TargetState != TargetStateReplacedVerified {
		t.Fatalf("third upload = %+v, want a completed operation", done)
	}
	if readFile(t, savePath) != "the prepared save" {
		t.Fatalf("the target save = %q, want the prepared bytes", readFile(t, savePath))
	}
	if done.BackupID == "" {
		t.Fatal("the mandatory backup was not recorded")
	}

	backups, err := store.ListBackups(target.ID)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(backups) != 1 || backups[0].Manual {
		t.Fatalf("backups = %+v, want one automatic backup", backups)
	}
	backupPath := filepath.Join(filepath.Dir(savePath), backups[0].FileName)
	if readFile(t, backupPath) != "the existing target save" {
		t.Fatalf("the backup = %q, want the save that was replaced", readFile(t, backupPath))
	}
	// Nothing staged is left beside the target.
	staged, err := filepath.Glob(filepath.Join(filepath.Dir(savePath), ".saveforge-incoming-*"))
	if err != nil || len(staged) != 0 {
		t.Fatalf("staging files after replacement = %v (glob error %v), want none", staged, err)
	}
}

// TestUploadCreatesNoBackupForAMissingTargetSave is the one case section 5
// allows without a backup, and it must not turn into a general exemption.
func TestUploadCreatesNoBackupForAMissingTargetSave(t *testing.T) {
	service, store, target, savePath := newTestService(t)
	prepared := filepath.Join(t.TempDir(), "prepared.sl2")
	writeFile(t, prepared, "the prepared save")

	result, err := service.Upload(context.Background(), OperationRequest{
		OperationID:                   "operation-1",
		TargetID:                      target.ID,
		PreparedPath:                  prepared,
		ContinueWithUnknownGameStatus: true,
	}, false)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if !result.Completed || result.BackupID != "" {
		t.Fatalf("result = %+v, want a completed upload with no backup", result)
	}
	if readFile(t, savePath) != "the prepared save" {
		t.Fatal("the new target save was not written")
	}
	backups, err := store.ListBackups(target.ID)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(backups) != 0 {
		t.Fatalf("backups = %+v, want none for a target that had no save", backups)
	}
}

// TestAlwaysPolicySkipsTheQuestionButNeverTheBackup is the whole difference the
// remote backup policy makes: it changes whether the user is asked, never
// whether the backup happens.
func TestAlwaysPolicySkipsTheQuestionButNeverTheBackup(t *testing.T) {
	state := t.TempDir()
	targetDirectory := t.TempDir()
	savePath := filepath.Join(targetDirectory, "ER0000.sl2")
	store := NewStore(state)
	target, err := store.CreateTarget(Target{Name: "t", Kind: KindLocal, SavePath: savePath})
	if err != nil {
		t.Fatalf("CreateTarget: %v", err)
	}
	settings := hostsettings.NewStore(state)
	if _, err := settings.Set(false, string(hostsettings.RemoteBackupAlways)); err != nil {
		t.Fatalf("Set host settings: %v", err)
	}
	service := NewService(store, settings, nil)

	writeFile(t, savePath, "the existing target save")
	prepared := filepath.Join(t.TempDir(), "prepared.sl2")
	writeFile(t, prepared, "the prepared save")

	result, err := service.Upload(context.Background(), OperationRequest{
		OperationID:                   "operation-1",
		TargetID:                      target.ID,
		PreparedPath:                  prepared,
		ContinueWithUnknownGameStatus: true,
	}, false)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if !result.Completed || result.BackupID == "" {
		t.Fatalf("result = %+v, want a completed upload that still created the backup", result)
	}
}

// TestActivateBackupBacksUpTheCurrentSaveFirst is the Save Manager rule of
// section 11: activating replaces the target, so the target's current save is
// preserved before it disappears.
func TestActivateBackupBacksUpTheCurrentSaveFirst(t *testing.T) {
	service, store, target, savePath := newTestService(t)
	writeFile(t, savePath, "the original save")

	created, err := service.CreateManualBackup(
		context.Background(), target.ID, []string{"before-dlc"}, "Before the DLC")
	if err != nil {
		t.Fatalf("CreateManualBackup: %v", err)
	}
	if !created.Manual {
		t.Fatalf("the Save Manager backup = %+v, want a manual one", created)
	}
	writeFile(t, savePath, "a later save")

	result, err := service.ActivateBackup(context.Background(), OperationRequest{
		OperationID:                   "operation-1",
		TargetID:                      target.ID,
		ContinueWithUnknownGameStatus: true,
		ConfirmRemoteBackup:           true,
	}, created.ID)
	if err != nil {
		t.Fatalf("ActivateBackup: %v", err)
	}
	if !result.Completed || result.TargetState != TargetStateReplacedVerified {
		t.Fatalf("result = %+v, want a completed activation", result)
	}
	if readFile(t, savePath) != "the original save" {
		t.Fatalf("the target save = %q, want the activated backup", readFile(t, savePath))
	}

	backups, err := store.ListBackups(target.ID)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("backups = %+v, want the manual one plus the mandatory backup of the replaced save", backups)
	}
	active := 0
	for _, backup := range backups {
		if backup.Active {
			active++
			if backup.ID != created.ID {
				t.Fatalf("the active backup is %q, want %q", backup.ID, created.ID)
			}
		}
		if !backup.Manual && readFile(t, filepath.Join(filepath.Dir(savePath), backup.FileName)) != "a later save" {
			t.Fatal("the save that was replaced was not backed up")
		}
	}
	if active != 1 {
		t.Fatalf("%d backups are marked active, want exactly one", active)
	}
}

// TestActivateAnUnknownBackupIsRefused: a backup the store does not know can
// never become the target save, whatever the caller states.
func TestActivateAnUnknownBackupIsRefused(t *testing.T) {
	service, _, target, savePath := newTestService(t)
	writeFile(t, savePath, "the original save")

	if _, err := service.ActivateBackup(context.Background(), OperationRequest{
		OperationID:                   "operation-1",
		TargetID:                      target.ID,
		ContinueWithUnknownGameStatus: true,
		ConfirmRemoteBackup:           true,
	}, "backup-that-does-not-exist"); err == nil {
		t.Fatal("ActivateBackup accepted a backup the store does not know")
	}
	if readFile(t, savePath) != "the original save" {
		t.Fatal("a refused activation changed the target save")
	}
}

// TestSSHTargetsFailClosed states the honest limit of this build: an SSH target
// can be configured, but no operation on it silently
// falls back to an unsafe path.
func TestSSHTargetsFailClosed(t *testing.T) {
	state := t.TempDir()
	store := NewStore(state)
	target, err := store.CreateTarget(Target{
		Name:     "Remote",
		Kind:     KindSSH,
		SavePath: "/home/deck/ER0000.sl2",
		Host:     "192.0.2.1",
		Port:     22,
		User:     "deck",
		KeyPath:  "/home/user/.ssh/id_ed25519",
	})
	if err != nil {
		t.Fatalf("CreateTarget: %v", err)
	}
	service := NewService(store, hostsettings.NewStore(state), nil)

	// No connection is attempted: the driver refuses before reaching a network.
	if _, err := service.TestTarget(context.Background(), target.ID); err == nil {
		t.Fatal("TestTarget on an SSH target reported success")
	}
	if _, err := service.Upload(context.Background(), OperationRequest{
		OperationID:                   "operation-1",
		TargetID:                      target.ID,
		PreparedPath:                  filepath.Join(t.TempDir(), "prepared.sl2"),
		ContinueWithUnknownGameStatus: true,
		ConfirmRemoteBackup:           true,
	}, false); err == nil {
		t.Fatal("Upload to an SSH target reported success")
	}
}

// TestSSHTargetValidationRefusesPasswordOnlyConfiguration keeps the key-only
// rule at the configuration boundary, where it is cheapest to enforce.
func TestSSHTargetValidationRefusesPasswordOnlyConfiguration(t *testing.T) {
	base := Target{
		Name:     "Remote",
		Kind:     KindSSH,
		SavePath: "/home/deck/ER0000.sl2",
		Host:     "192.0.2.1",
		Port:     22,
		User:     "deck",
	}
	if err := base.Validate(); err == nil {
		t.Fatal("an SSH target with no key path was accepted")
	}
	withKey := base
	withKey.KeyPath = "/home/user/.ssh/id_ed25519"
	if err := withKey.Validate(); err != nil {
		t.Fatalf("a complete SSH target was refused: %v", err)
	}
	relativeSSH := withKey
	relativeSSH.SavePath = "relative/ER0000.sl2"
	if err := relativeSSH.Validate(); err == nil {
		t.Fatal("an SSH target with a relative save path was accepted")
	}
	relativeLocal := Target{Name: "Local", Kind: KindLocal, SavePath: "relative/ER0000.sl2"}
	if err := relativeLocal.Validate(); err == nil {
		t.Fatal("a local target with a relative save path was accepted")
	}
	// A command that could become two commands is refused before it is stored.
	multiline := withKey
	multiline.StartCommand = "steam -applaunch 1245620\nrm -rf /"
	if err := multiline.Validate(); err == nil {
		t.Fatal("a multi-line start command was accepted")
	}
}

// TestHostKeyTrustIsExplicitAndDroppedWhenTheAddressChanges is the Trust On
// First Use contract: nothing trusts a key on its own, and moving a target to
// another machine never inherits the decision made about the previous one.
func TestHostKeyTrustIsExplicitAndDroppedWhenTheAddressChanges(t *testing.T) {
	store := NewStore(t.TempDir())
	target, err := store.CreateTarget(Target{
		Name: "Remote", Kind: KindSSH, SavePath: "/s.sl2",
		Host: "192.0.2.1", Port: 22, User: "deck", KeyPath: "/k",
	})
	if err != nil {
		t.Fatalf("CreateTarget: %v", err)
	}
	if _, trusted, err := store.TrustedHostKey(target.Address()); err != nil || trusted {
		t.Fatalf("a fresh target already trusts a host key (trusted=%v, err=%v)", trusted, err)
	}
	if err := store.TrustHostKey(target.Address(), "SHA256:abc"); err != nil {
		t.Fatalf("TrustHostKey: %v", err)
	}
	fingerprint, trusted, err := store.TrustedHostKey(target.Address())
	if err != nil || !trusted || fingerprint != "SHA256:abc" {
		t.Fatalf("TrustedHostKey = %q, %v, %v", fingerprint, trusted, err)
	}

	moved := target
	moved.Host = "192.0.2.2"
	if _, err := store.UpdateTarget(moved); err != nil {
		t.Fatalf("UpdateTarget: %v", err)
	}
	if _, trusted, err := store.TrustedHostKey(moved.Address()); err != nil || trusted {
		t.Fatal("the new address inherited the fingerprint approved for the previous one")
	}
	if _, trusted, err := store.TrustedHostKey(target.Address()); err != nil || trusted {
		t.Fatal("the fingerprint of the previous address survived the move")
	}
}
