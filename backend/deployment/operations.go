package deployment

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/oisis/EldenRing-SaveForge/backend/hostsettings"
)

// The reasons an operation stopped before it finished. They are stable codes,
// not sentences: the interface decides what to ask and how to word it, and no
// layer classifies an outcome by matching text.
//
// None of them is an error. Every one of them describes a target that was left
// exactly as it was, and every one of them is resolved by an explicit user
// decision that is passed back in the next request.
const (
	// BlockedGameRunning is the hard block of section 4: while the game runs,
	// plain Upload and Download are refused and there is no Continue Anyway.
	BlockedGameRunning = "game_running"
	// BlockedGameStatusUnknown asks the user to confirm continuing after the
	// explicit warning section 4 requires.
	BlockedGameStatusUnknown = "game_status_unknown"
	// BlockedRemoteBackupConfirmation asks for the mandatory backup of an
	// existing target save. Refusing it cancels the operation; it can never
	// continue without the backup.
	BlockedRemoteBackupConfirmation = "remote_backup_confirmation_required"
	// BlockedStopGameConfirmation asks permission to stop a running game before
	// Deploy & Launch or Close & Download.
	BlockedStopGameConfirmation = "stop_game_confirmation_required"
	// BlockedCancelled reports a cooperative cancellation before the replacement
	// point.
	BlockedCancelled = "cancelled"
)

const (
	TargetStateUnchanged          = "unchanged"
	TargetStateReplacedVerified   = "replaced_verified"
	TargetStateReplacedUnverified = "replaced_unverified"

	FailureReplacement  = "replacement_failed"
	FailureVerification = "target_verification_failed"
	FailureLaunch       = "launch_failed"
	FailureMetadata     = "metadata_update_failed"
)

// The stage names an operation reports. They are the steps of section 3 and
// section 6 of deployment.md and nothing finer.
const (
	StageGameStatus = "game_status"
	StageStopGame   = "stop_game"
	StageStabilise  = "stabilise_save"
	StageBackup     = "backup_target"
	StageTransfer   = "transfer"
	StageReplace    = "replace_target"
	StageVerify     = "verify_target"
	StageLaunchGame = "launch_game"
	StageDownload   = "download"
	StageRetention  = "prune_backups"
)

// Stage is one completed or attempted step of an operation.
type Stage struct {
	Stage     string `json:"stage"`
	Completed bool   `json:"completed"`
	Detail    string `json:"detail,omitempty"`
}

// Progress is one live report of a running operation.
type Progress struct {
	OperationID string `json:"operationID"`
	TargetID    string `json:"targetID"`
	Stage       string `json:"stage"`
	Percent     int    `json:"percent"`
	ElapsedMS   int64  `json:"elapsedMS"`
	Finished    bool   `json:"finished"`
}

// OperationRequest carries one operation and the explicit decisions the user
// already made for it.
type OperationRequest struct {
	OperationID string `json:"operationID"`
	TargetID    string `json:"targetID"`
	// PreparedPath is the validated local temporary file an upload sends. The
	// caller produces it through the shared save preparation phase; this package
	// never serialises a session itself.
	PreparedPath string `json:"preparedPath,omitempty"`
	// StagingPath is the local temporary file a download writes.
	StagingPath string `json:"stagingPath,omitempty"`

	ContinueWithUnknownGameStatus bool `json:"continueWithUnknownGameStatus,omitempty"`
	ConfirmRemoteBackup           bool `json:"confirmRemoteBackup,omitempty"`
	ConfirmStopGame               bool `json:"confirmStopGame,omitempty"`
}

// OperationResult reports what an operation actually did.
type OperationResult struct {
	OperationID string     `json:"operationID"`
	TargetID    string     `json:"targetID"`
	Completed   bool       `json:"completed"`
	Blocked     string     `json:"blocked,omitempty"`
	Failure     string     `json:"failure,omitempty"`
	TargetState string     `json:"targetState"`
	GameStatus  GameStatus `json:"gameStatus"`
	Stages      []Stage    `json:"stages"`
	// BackupID names the mandatory backup this operation created, if any.
	BackupID string `json:"backupID,omitempty"`
	// LocalPath is the staging file a download produced. It is a temporary
	// location and never a durable save target.
	LocalPath string          `json:"localPath,omitempty"`
	Stop      *CommandOutcome `json:"stop,omitempty"`
	Launch    *CommandOutcome `json:"launch,omitempty"`
}

// Service performs deployment and Save Manager operations. It owns no settings
// of its own: the targets and the backup metadata come from Store and the
// remote backup policy and retention come from its callers, which read them
// from their own single sources of truth.
type Service struct {
	store    *Store
	settings *hostsettings.Store
	// progress is the optional live sink. The bridge wires it to a host event;
	// a test leaves it nil and reads the stages of the result instead.
	progress func(Progress)
	// newDriver is replaced by a test with a fake target. Production always uses
	// NewDriver, so no test double can ever reach a real machine.
	newDriver func(Target) (Driver, error)
	// now is the clock backup names are derived from.
	now func() time.Time
}

// NewService builds the service around one store.
func NewService(store *Store, settings *hostsettings.Store, progress func(Progress)) *Service {
	return &Service{
		store:     store,
		settings:  settings,
		progress:  progress,
		newDriver: NewDriver,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

func (service *Service) driverFor(targetID string) (Target, Driver, error) {
	target, err := service.store.GetTarget(targetID)
	if err != nil {
		return Target{}, nil, err
	}
	driver, err := service.newDriver(target)
	if err != nil {
		return Target{}, nil, err
	}
	return target, driver, nil
}

func (service *Service) report(started time.Time, request OperationRequest, stage string, percent int) {
	if service.progress == nil {
		return
	}
	service.progress(Progress{
		OperationID: request.OperationID,
		TargetID:    request.TargetID,
		Stage:       stage,
		Percent:     percent,
		ElapsedMS:   time.Since(started).Milliseconds(),
	})
}

func (service *Service) finish(started time.Time, request OperationRequest, result OperationResult) OperationResult {
	if service.progress != nil {
		service.progress(Progress{
			OperationID: request.OperationID,
			TargetID:    request.TargetID,
			Stage:       "finished",
			Percent:     100,
			ElapsedMS:   time.Since(started).Milliseconds(),
			Finished:    true,
		})
	}
	return result
}

// TestTargetResult reports what a configuration test established.
type TestTargetResult struct {
	TargetID string `json:"targetID"`
	// Reachable is true when the target's save location could be verified.
	Reachable bool `json:"reachable"`
	// HostKeyTrusted reports whether an SSH target already has an approved host
	// key. It is false for a local target, which has no host key at all.
	HostKeyTrusted bool       `json:"hostKeyTrusted"`
	GameStatus     GameStatus `json:"gameStatus"`
	SaveExists     bool       `json:"saveExists"`
}

// TestTarget verifies one target without changing anything on it.
func (service *Service) TestTarget(ctx context.Context, targetID string) (TestTargetResult, error) {
	target, driver, err := service.driverFor(targetID)
	if err != nil {
		return TestTargetResult{}, err
	}
	defer driver.Close()
	result := TestTargetResult{TargetID: targetID, GameStatus: GameUnknown}
	if target.Kind == KindSSH {
		_, trusted, keyErr := service.store.TrustedHostKey(target.Address())
		if keyErr != nil {
			return TestTargetResult{}, keyErr
		}
		result.HostKeyTrusted = trusted
	}
	if err := driver.Test(ctx); err != nil {
		return result, err
	}
	result.Reachable = true
	if exists, existsErr := driver.Exists(ctx, target.SavePath); existsErr == nil {
		result.SaveExists = exists
	}
	status, err := driver.GameStatus(ctx)
	if err != nil {
		return result, err
	}
	result.GameStatus = status
	return result, nil
}

// GameStatusOf reports the confirmed game state of one target.
func (service *Service) GameStatusOf(ctx context.Context, targetID string) (GameStatus, error) {
	_, driver, err := service.driverFor(targetID)
	if err != nil {
		return GameUnknown, err
	}
	defer driver.Close()
	return driver.GameStatus(ctx)
}

// Launch runs the target's configured start command as an explicit user action.
func (service *Service) Launch(ctx context.Context, targetID string) (CommandOutcome, error) {
	_, driver, err := service.driverFor(targetID)
	if err != nil {
		return CommandOutcome{}, err
	}
	defer driver.Close()
	return driver.RunStart(ctx)
}

// CloseGame runs the target's configured stop command as an explicit user
// action. A command that found no process is a truthful outcome, not a failure.
func (service *Service) CloseGame(ctx context.Context, targetID string) (CommandOutcome, error) {
	_, driver, err := service.driverFor(targetID)
	if err != nil {
		return CommandOutcome{}, err
	}
	defer driver.Close()
	return driver.RunStop(ctx)
}

// Upload replaces the target save with the prepared local file. launchAfter
// turns the same operation into Deploy & Launch: the two share every step up to
// and including the final verification, so there is one implementation of the
// safe replacement rather than two that could drift apart.
func (service *Service) Upload(
	ctx context.Context, request OperationRequest, launchAfter bool,
) (OperationResult, error) {
	started := time.Now()
	target, driver, err := service.driverFor(request.TargetID)
	if err != nil {
		return OperationResult{}, err
	}
	defer driver.Close()
	if request.PreparedPath == "" {
		return OperationResult{}, errors.New("an upload needs the prepared save file")
	}
	result := OperationResult{
		OperationID: request.OperationID,
		TargetID:    request.TargetID,
		Stages:      []Stage{},
		TargetState: TargetStateUnchanged,
	}

	service.report(started, request, StageGameStatus, 5)
	status, err := driver.GameStatus(ctx)
	if err != nil {
		return OperationResult{}, err
	}
	result.GameStatus = status
	result.Stages = append(result.Stages, Stage{Stage: StageGameStatus, Completed: true})

	switch status {
	case GameRunning:
		if !launchAfter {
			// Section 7: a plain Upload is blocked outright while the game runs.
			result.Blocked = BlockedGameRunning
			return service.finish(started, request, result), nil
		}
		if !request.ConfirmStopGame {
			result.Blocked = BlockedStopGameConfirmation
			return service.finish(started, request, result), nil
		}
		service.report(started, request, StageStopGame, 12)
		outcome, stopErr := driver.RunStop(ctx)
		if stopErr != nil {
			return OperationResult{}, stopErr
		}
		result.Stop = &outcome
		result.Stages = append(result.Stages, Stage{Stage: StageStopGame, Completed: true})
		service.report(started, request, StageStabilise, 18)
		if err := driver.WaitForStableSave(ctx, target.SavePath); err != nil {
			return service.cancelled(started, request, result, err)
		}
		result.Stages = append(result.Stages, Stage{Stage: StageStabilise, Completed: true})
	case GameUnknown:
		if !request.ContinueWithUnknownGameStatus {
			result.Blocked = BlockedGameStatusUnknown
			return service.finish(started, request, result), nil
		}
	}

	service.report(started, request, StageBackup, 30)
	backupResult, blocked, err := service.backupExistingTarget(ctx, target, driver, request)
	if err != nil {
		return service.cancelled(started, request, result, err)
	}
	if blocked != "" {
		result.Blocked = blocked
		return service.finish(started, request, result), nil
	}
	if backupResult != nil {
		result.BackupID = backupResult.ID
		result.Stages = append(result.Stages, Stage{Stage: StageBackup, Completed: true})
	} else {
		result.Stages = append(result.Stages, Stage{
			Stage: StageBackup, Completed: true, Detail: "the target has no existing save to back up",
		})
	}

	service.report(started, request, StageTransfer, 55)
	if err := ctx.Err(); err != nil {
		return service.cancelled(started, request, result, err)
	}
	// ReplaceFromLocal performs the transfer, the verification of the staged
	// copy, the atomic replacement and the verification of the final file. It is
	// the single irreversible point of this operation.
	service.report(started, request, StageReplace, 70)
	replacement, err := driver.ReplaceFromLocal(ctx, request.PreparedPath, target.SavePath)
	if err != nil {
		if replacement.Committed {
			result.TargetState = TargetStateReplacedUnverified
			result.Failure = FailureVerification
			result.Stages = append(result.Stages, Stage{Stage: StageReplace, Completed: true}, Stage{Stage: StageVerify, Detail: "the replaced target could not be verified"})
		} else {
			result.Failure = FailureReplacement
			result.Stages = append(result.Stages, Stage{Stage: StageReplace, Detail: "the target was not replaced"})
		}
		return service.finish(started, request, result), nil
	}
	result.TargetState = TargetStateReplacedVerified
	result.Stages = append(result.Stages,
		Stage{Stage: StageTransfer, Completed: true},
		Stage{Stage: StageReplace, Completed: true},
		Stage{Stage: StageVerify, Completed: true},
	)

	service.report(started, request, StageRetention, 85)
	if detail := service.pruneAutomaticBackups(ctx, target, driver); detail != "" {
		result.Stages = append(result.Stages, Stage{Stage: StageRetention, Detail: detail})
	} else {
		result.Stages = append(result.Stages, Stage{Stage: StageRetention, Completed: true})
	}

	if launchAfter {
		service.report(started, request, StageLaunchGame, 95)
		// Past the replacement point only the not-yet-performed steps can still
		// be cancelled, which is exactly what section 12 allows.
		if err := ctx.Err(); err != nil {
			result.Blocked = BlockedCancelled
			result.Stages = append(result.Stages, Stage{
				Stage: StageLaunchGame, Detail: "the game was not started; the target is already replaced",
			})
			return service.finish(started, request, result), nil
		}
		outcome, launchErr := driver.RunStart(ctx)
		if launchErr != nil {
			result.Failure = FailureLaunch
			result.Stages = append(result.Stages, Stage{Stage: StageLaunchGame, Detail: "the game could not be started"})
			return service.finish(started, request, result), nil
		}
		result.Launch = &outcome
		result.Stages = append(result.Stages, Stage{Stage: StageLaunchGame, Completed: true})
	}

	result.Completed = true
	return service.finish(started, request, result), nil
}

// Download copies the target save into a local staging file. stopFirst turns the
// same operation into Close & Download.
func (service *Service) Download(
	ctx context.Context, request OperationRequest, stopFirst bool,
) (OperationResult, error) {
	started := time.Now()
	target, driver, err := service.driverFor(request.TargetID)
	if err != nil {
		return OperationResult{}, err
	}
	defer driver.Close()
	if request.StagingPath == "" {
		return OperationResult{}, errors.New("a download needs a local staging path")
	}
	result := OperationResult{
		OperationID: request.OperationID,
		TargetID:    request.TargetID,
		Stages:      []Stage{},
		TargetState: TargetStateUnchanged,
	}

	service.report(started, request, StageGameStatus, 5)
	status, err := driver.GameStatus(ctx)
	if err != nil {
		return OperationResult{}, err
	}
	result.GameStatus = status
	result.Stages = append(result.Stages, Stage{Stage: StageGameStatus, Completed: true})

	switch status {
	case GameRunning:
		if !stopFirst {
			result.Blocked = BlockedGameRunning
			return service.finish(started, request, result), nil
		}
		if !request.ConfirmStopGame {
			result.Blocked = BlockedStopGameConfirmation
			return service.finish(started, request, result), nil
		}
		service.report(started, request, StageStopGame, 20)
		outcome, stopErr := driver.RunStop(ctx)
		if stopErr != nil {
			return OperationResult{}, stopErr
		}
		result.Stop = &outcome
		result.Stages = append(result.Stages, Stage{Stage: StageStopGame, Completed: true})
	case GameUnknown:
		if !request.ContinueWithUnknownGameStatus {
			result.Blocked = BlockedGameStatusUnknown
			return service.finish(started, request, result), nil
		}
	}

	service.report(started, request, StageStabilise, 45)
	if err := driver.WaitForStableSave(ctx, target.SavePath); err != nil {
		return service.cancelled(started, request, result, err)
	}
	result.Stages = append(result.Stages, Stage{Stage: StageStabilise, Completed: true})

	service.report(started, request, StageDownload, 75)
	exists, err := driver.Exists(ctx, target.SavePath)
	if err != nil {
		return OperationResult{}, err
	}
	if !exists {
		return OperationResult{}, errors.New("the target has no save to download")
	}
	if err := driver.CopyToLocal(ctx, target.SavePath, request.StagingPath); err != nil {
		return service.cancelled(started, request, result, err)
	}
	result.Stages = append(result.Stages, Stage{Stage: StageDownload, Completed: true})
	result.LocalPath = request.StagingPath
	result.Completed = true
	return service.finish(started, request, result), nil
}

// backupExistingTarget performs the mandatory backup of section 5. It returns a
// nil record when the target has no existing save, which is the only case in
// which a replacement may proceed without one.
func (service *Service) backupExistingTarget(
	ctx context.Context, target Target, driver Driver, request OperationRequest,
) (*BackupRecord, string, error) {
	exists, err := driver.Exists(ctx, target.SavePath)
	if err != nil {
		return nil, "", err
	}
	if !exists {
		return nil, "", nil
	}
	policy := hostsettings.DefaultRemoteBackupPolicy
	if service.settings != nil {
		settings, settingsErr := service.settings.Get()
		if settingsErr != nil {
			return nil, "", settingsErr
		}
		policy = settings.RemoteBackupPolicy
	}
	if policy == hostsettings.RemoteBackupAsk && !request.ConfirmRemoteBackup {
		return nil, BlockedRemoteBackupConfirmation, nil
	}
	record, err := service.createBackup(ctx, target, driver, false, nil, "")
	if err != nil {
		// Section 5: a failed backup blocks the replacement.
		return nil, "", fmt.Errorf("the mandatory target backup failed: %w", err)
	}
	return &record, "", nil
}

// createBackup copies the target save to a fresh backup file beside it and
// records the metadata. The name grammar is the one the local Save lifecycle
// already uses, so a user reads one naming convention everywhere.
func (service *Service) createBackup(
	ctx context.Context, target Target, driver Driver, manual bool, tags []string, description string,
) (BackupRecord, error) {
	now := service.now()
	base := path.Base(target.SavePath) + "." + now.Format("20060102150405")
	directory := target.BackupDirectory()
	fileName := ""
	for collision := 1; collision <= 1000; collision++ {
		candidate := base + "_bak"
		if collision > 1 {
			candidate = base + "_" + strconv.Itoa(collision) + "_bak"
		}
		taken, err := driver.Exists(ctx, joinTargetPath(target, directory, candidate))
		if err != nil {
			return BackupRecord{}, err
		}
		if !taken {
			fileName = candidate
			break
		}
	}
	if fileName == "" {
		return BackupRecord{}, errors.New("cannot allocate a unique backup name on the target")
	}
	destination := joinTargetPath(target, directory, fileName)
	if err := driver.CopyOnTarget(ctx, target.SavePath, destination); err != nil {
		return BackupRecord{}, err
	}
	verified, err := driver.FilesEqual(ctx, target.SavePath, destination)
	if err != nil {
		_ = driver.Remove(ctx, destination)
		return BackupRecord{}, err
	}
	if !verified {
		_ = driver.Remove(ctx, destination)
		return BackupRecord{}, errors.New("the backup does not match the target save")
	}
	record, err := service.store.AddBackup(BackupRecord{
		TargetID:    target.ID,
		FileName:    fileName,
		CreatedAt:   now.Format(time.RFC3339),
		Manual:      manual,
		Tags:        tags,
		Description: description,
	})
	if err != nil {
		_ = driver.Remove(ctx, destination)
		return BackupRecord{}, err
	}
	return record, nil
}

// pruneAutomaticBackups removes automatic backups past the retention window. It
// returns a short detail when something could not be pruned: a failure here is
// maintenance, not a reason to report a successful replacement as failed.
func (service *Service) pruneAutomaticBackups(ctx context.Context, target Target, driver Driver) string {
	retention := 0
	if service.settings != nil {
		// Retention itself stays owned by the Save Lifecycle settings; deployment
		// reads the value and never keeps a second copy of it.
		retention = deploymentBackupRetention
	}
	if retention < 1 {
		return ""
	}
	stale, err := service.store.AutomaticBackupsOverRetention(target.ID, retention)
	if err != nil {
		return "old target backups could not be reviewed"
	}
	for _, record := range stale {
		if err := driver.Remove(ctx, joinTargetPath(target, target.BackupDirectory(), record.FileName)); err != nil {
			return "old target backups could not be removed"
		}
		if _, err := service.store.RemoveBackup(target.ID, record.ID); err != nil {
			return "old target backups could not be forgotten"
		}
	}
	return ""
}

// deploymentBackupRetention is how many automatic target backups are kept.
// ponytail: one constant, not a second retention setting. Section 5 of
// deployment.md gives the user Ask and Always and no retention control for
// target backups, and manual Save Manager backups are exempt from it entirely.
const deploymentBackupRetention = 10

func (service *Service) cancelled(
	started time.Time, request OperationRequest, result OperationResult, err error,
) (OperationResult, error) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		result.Blocked = BlockedCancelled
		return service.finish(started, request, result), nil
	}
	return OperationResult{}, err
}

// joinTargetPath joins a directory and a file name in the target's own path
// syntax. A local target uses the host's separator; an SSH target is always a
// POSIX path.
func joinTargetPath(target Target, directory string, name string) string {
	if target.Kind == KindLocal {
		return localJoin(directory, name)
	}
	return path.Join(directory, name)
}
