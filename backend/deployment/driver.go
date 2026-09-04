package deployment

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// CommandOutcome reports what running one configured command actually did.
// Section 10 of deployment.md requires the real result, including the case
// where the stop command found no process: that is not a save corruption and
// must be stated as such rather than reported as a failure.
type CommandOutcome struct {
	// Configured is false when the target states no command for this action.
	Configured bool `json:"configured"`
	Executed   bool `json:"executed"`
	ExitCode   int  `json:"exitCode"`
	// Detail is a short, safe explanation. It never carries the command's own
	// output: that output is the target machine's and may contain paths, user
	// names or secrets this application must not surface.
	Detail string `json:"detail,omitempty"`
}

// ReplacementResult records the irreversible replacement point separately
// from the verification which follows it.
type ReplacementResult struct {
	Committed bool
	Verified  bool
}

// Driver is the target-side half of every deployment operation. One
// implementation talks to the local filesystem and one to an SSH host; nothing
// above this interface knows which of the two it is holding.
type Driver interface {
	Kind() Kind
	// Test verifies that the target is reachable and its save path usable. It
	// changes nothing.
	Test(ctx context.Context) error
	Exists(ctx context.Context, path string) (bool, error)
	// CopyOnTarget duplicates a file inside the target system. It is how a
	// mandatory backup is taken without moving the data through this machine.
	CopyOnTarget(ctx context.Context, source string, destination string) error
	FilesEqual(ctx context.Context, left string, right string) (bool, error)
	Remove(ctx context.Context, path string) error
	// ReplaceFromLocal uploads localPath and atomically replaces targetPath with
	// it. An implementation that cannot do the replacement atomically must fail
	// instead of overwriting in place: section 6 of deployment.md states there is
	// no fallback to a direct overwrite.
	ReplaceFromLocal(ctx context.Context, localPath string, targetPath string) (ReplacementResult, error)
	// ReplaceOnTarget is ReplaceFromLocal for a source that already lives on the
	// target, which is how Save Manager restores a backup without moving the
	// data through this machine. It obeys the same staging, verification and
	// atomic replacement rules.
	ReplaceOnTarget(ctx context.Context, sourcePath string, targetPath string) (ReplacementResult, error)
	CopyToLocal(ctx context.Context, targetPath string, localPath string) error
	RunStart(ctx context.Context) (CommandOutcome, error)
	RunStop(ctx context.Context) (CommandOutcome, error)
	// GameStatus reports the confirmed state of the game on the target.
	GameStatus(ctx context.Context) (GameStatus, error)
	// WaitForStableSave blocks until the target save has stopped changing, so a
	// download never captures a file the game is still writing.
	WaitForStableSave(ctx context.Context, path string) error
	Close() error
}

// ErrSSHTransportUnavailable is the fail-closed answer of every SSH operation in
// this build.
//
// SSH itself is specified and configured here: the target model, the key-only
// rule and the Trust On First Use host key store are complete. What is missing
// is a file transfer client able to write a temporary file on the target and
// rename it over the save atomically. The module has no such dependency —
// github.com/pkg/sftp is not required by this repository — and the alternatives
// available with the current dependencies are a direct overwrite or a shell
// pipeline assembled from the target's paths. Both are refused: the first has no
// safe replacement point and the second builds a command by concatenation.
//
// So the SSH driver refuses every operation rather than performing an unsafe
// one. It never falls back to an unverified host key and never overwrites a save
// in place.
var ErrSSHTransportUnavailable = errors.New(
	"SSH deployment is not available in this build: it needs an SFTP client to replace a target save atomically")

// NewDriver returns the driver of one target.
func NewDriver(target Target) (Driver, error) {
	switch target.Kind {
	case KindLocal:
		return &localDriver{target: target}, nil
	case KindSSH:
		return &sshDriver{target: target}, nil
	}
	return nil, fmt.Errorf("unknown deployment target kind %q", target.Kind)
}

// sshDriver is the fail-closed SSH adapter described by
// ErrSSHTransportUnavailable.
type sshDriver struct{ target Target }

func (driver *sshDriver) Kind() Kind { return KindSSH }

func (driver *sshDriver) Test(context.Context) error { return ErrSSHTransportUnavailable }

func (driver *sshDriver) Exists(context.Context, string) (bool, error) {
	return false, ErrSSHTransportUnavailable
}

func (driver *sshDriver) CopyOnTarget(context.Context, string, string) error {
	return ErrSSHTransportUnavailable
}

func (driver *sshDriver) FilesEqual(context.Context, string, string) (bool, error) {
	return false, ErrSSHTransportUnavailable
}

func (driver *sshDriver) Remove(context.Context, string) error { return ErrSSHTransportUnavailable }

func (driver *sshDriver) ReplaceFromLocal(context.Context, string, string) (ReplacementResult, error) {
	return ReplacementResult{}, ErrSSHTransportUnavailable
}

func (driver *sshDriver) ReplaceOnTarget(context.Context, string, string) (ReplacementResult, error) {
	return ReplacementResult{}, ErrSSHTransportUnavailable
}

func (driver *sshDriver) CopyToLocal(context.Context, string, string) error {
	return ErrSSHTransportUnavailable
}

func (driver *sshDriver) RunStart(context.Context) (CommandOutcome, error) {
	return CommandOutcome{}, ErrSSHTransportUnavailable
}

func (driver *sshDriver) RunStop(context.Context) (CommandOutcome, error) {
	return CommandOutcome{}, ErrSSHTransportUnavailable
}

func (driver *sshDriver) GameStatus(context.Context) (GameStatus, error) {
	return GameUnknown, ErrSSHTransportUnavailable
}

func (driver *sshDriver) WaitForStableSave(context.Context, string) error {
	return ErrSSHTransportUnavailable
}

func (driver *sshDriver) Close() error { return nil }

// stabilisationPoll and stabilisationRounds define what "the save stopped
// changing" means: the size and the modification time must be identical across
// this many consecutive observations.
const (
	stabilisationPoll   = 500 * time.Millisecond
	stabilisationRounds = 4
)
