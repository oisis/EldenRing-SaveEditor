package deployment

import (
	"context"
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

// ReplacementOutcome states what a driver established about the irreversible
// replacement point. Its three values are genuinely different answers and none
// of them may be reported as another.
//
// The zero value is the safe one: an operation that failed before it ever asked
// for the replacement left the target exactly as it was.
type ReplacementOutcome string

const (
	// ReplacementNotPerformed is a certain negative. The rename was never asked
	// for, or the target system answered the request with a refusal, so the
	// target still holds what it held.
	ReplacementNotPerformed ReplacementOutcome = ""
	// ReplacementPerformed is a certain positive: the rename was acknowledged.
	ReplacementPerformed ReplacementOutcome = "performed"
	// ReplacementUndetermined is neither. The replacement request was sent and
	// the answer to it was lost, so the target may or may not carry the new save.
	// Losing the answer is not evidence the server did nothing: it is never
	// reported as "unchanged", never reported as "replaced", never retried
	// automatically, and the game is never started after it.
	ReplacementUndetermined ReplacementOutcome = "undetermined"
)

// ReplacementResult records the irreversible replacement point separately
// from the verification which follows it.
type ReplacementResult struct {
	Outcome ReplacementOutcome
	// Verified is only ever true together with ReplacementPerformed: an
	// undetermined replacement has nothing that could have been verified.
	Verified bool
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
	// mandatory backup is taken without writing it to a durable local file.
	CopyOnTarget(ctx context.Context, source string, destination string) error
	FilesEqual(ctx context.Context, left string, right string) (bool, error)
	Remove(ctx context.Context, path string) error
	// ReplaceFromLocal uploads localPath and atomically replaces targetPath with
	// it. An implementation that cannot do the replacement atomically must fail
	// instead of overwriting in place: section 6 of deployment.md states there is
	// no fallback to a direct overwrite.
	ReplaceFromLocal(ctx context.Context, localPath string, targetPath string) (ReplacementResult, error)
	// ReplaceOnTarget is ReplaceFromLocal for a source that already lives on the
	// target, which is how Save Manager restores a backup without writing it to a
	// durable local file. It obeys the same staging, verification and atomic
	// replacement rules.
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

// NewDriver returns the driver of one target. keys is only used by the SSH
// driver, which verifies the host key against it under Trust On First Use.
func NewDriver(target Target, keys HostKeys) (Driver, error) {
	switch target.Kind {
	case KindLocal:
		return &localDriver{target: target}, nil
	case KindSSH:
		return &sshDriver{target: target, keys: keys}, nil
	}
	return nil, fmt.Errorf("unknown deployment target kind %q", target.Kind)
}

// TransferSupported reports whether this build can move a save to and from a
// kind of target. Both supported kinds now have a driver that stages, verifies
// and atomically replaces, so the interface no longer disables anything for a
// missing transport.
func TransferSupported(kind Kind) bool {
	return kind == KindLocal || kind == KindSSH
}

// interpretGameStatus maps the outcome of a configured status command onto the
// three states, and is the only place that mapping exists.
//
// The convention is stated to the user in the target form and in the endpoint
// documentation: exit 0 means the game runs, exit 1 means it does not, and
// anything else — no configured command, another exit code, a timeout, a
// transport fault or a command that could not be started — is unknown.
// Guessing from a process name, from the start command or from the operating
// system would invent a state this application cannot confirm.
func interpretGameStatus(outcome CommandOutcome, err error) (GameStatus, error) {
	if err != nil {
		return GameUnknown, nil
	}
	if !outcome.Configured || !outcome.Executed {
		return GameUnknown, nil
	}
	switch outcome.ExitCode {
	case 0:
		return GameRunning, nil
	case 1:
		return GameStopped, nil
	}
	return GameUnknown, nil
}

// stabilisationPoll and stabilisationRounds define what "the save stopped
// changing" means: the size and the modification time must be identical across
// this many consecutive observations.
const (
	stabilisationPoll   = 500 * time.Millisecond
	stabilisationRounds = 4
)
