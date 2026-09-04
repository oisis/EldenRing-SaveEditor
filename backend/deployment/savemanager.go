package deployment

import (
	"context"
	"errors"
	"time"
)

// This file is the Save Manager half of the package. It shares the targets, the
// drivers and the backup metadata of the deployment operations above; there is
// deliberately no second target model and no second backup implementation.

// CreateManualBackup copies the current target save into a new manual backup.
// Manual backups are exempt from automatic retention, so this path never prunes.
func (service *Service) CreateManualBackup(
	ctx context.Context, targetID string, tags []string, description string,
) (BackupRecord, error) {
	target, driver, err := service.driverFor(targetID)
	if err != nil {
		return BackupRecord{}, err
	}
	defer driver.Close()
	exists, err := driver.Exists(ctx, target.SavePath)
	if err != nil {
		return BackupRecord{}, err
	}
	if !exists {
		return BackupRecord{}, errors.New("the target has no save to back up")
	}
	return service.createBackup(ctx, target, driver, true, tags, description)
}

// ActivateBackup makes one backup the target's current save.
//
// It is a replacement of the target file, so it obeys the same safety rules as
// a deployment: the current target save is backed up first, the replacement is
// atomic, and a game that is confirmed to be running blocks the operation.
func (service *Service) ActivateBackup(
	ctx context.Context, request OperationRequest, backupID string,
) (OperationResult, error) {
	started := time.Now()
	target, driver, err := service.driverFor(request.TargetID)
	if err != nil {
		return OperationResult{}, err
	}
	defer driver.Close()

	records, err := service.store.ListBackups(request.TargetID)
	if err != nil {
		return OperationResult{}, err
	}
	index := indexOfBackup(records, backupID)
	if index < 0 {
		// A backup the store does not know cannot be activated. Nothing here
		// falls back to a file name the frontend supplied.
		return OperationResult{}, errors.New("the selected backup does not exist")
	}
	backupPath := joinTargetPath(target, target.BackupDirectory(), records[index].FileName)
	backupExists, err := driver.Exists(ctx, backupPath)
	if err != nil {
		return OperationResult{}, err
	}
	if !backupExists {
		return OperationResult{}, errors.New("the selected backup is missing on the target")
	}

	result := OperationResult{
		OperationID: request.OperationID,
		TargetID:    request.TargetID,
		Stages:      []Stage{},
		TargetState: TargetStateUnchanged,
	}
	service.report(started, request, StageGameStatus, 10)
	status, err := driver.GameStatus(ctx)
	if err != nil {
		return OperationResult{}, err
	}
	result.GameStatus = status
	result.Stages = append(result.Stages, Stage{Stage: StageGameStatus, Completed: true})
	switch status {
	case GameRunning:
		result.Blocked = BlockedGameRunning
		return service.finish(started, request, result), nil
	case GameUnknown:
		if !request.ContinueWithUnknownGameStatus {
			result.Blocked = BlockedGameStatusUnknown
			return service.finish(started, request, result), nil
		}
	}

	service.report(started, request, StageBackup, 40)
	created, blocked, err := service.backupExistingTarget(ctx, target, driver, request)
	if err != nil {
		return service.cancelled(started, request, result, err)
	}
	if blocked != "" {
		result.Blocked = blocked
		return service.finish(started, request, result), nil
	}
	if created != nil {
		result.BackupID = created.ID
	}
	result.Stages = append(result.Stages, Stage{Stage: StageBackup, Completed: true})

	service.report(started, request, StageReplace, 70)
	// The backup is staged and renamed exactly like an upload, so activation has
	// the same single irreversible point and the same final verification.
	replacement, err := driver.ReplaceOnTarget(ctx, backupPath, target.SavePath)
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
		Stage{Stage: StageReplace, Completed: true},
		Stage{Stage: StageVerify, Completed: true},
	)

	if _, err := service.store.SetActiveBackup(request.TargetID, backupID); err != nil {
		result.Failure = FailureMetadata
		return service.finish(started, request, result), nil
	}
	result.Completed = true
	return service.finish(started, request, result), nil
}

// ClearActiveBackup removes the active mark. It is metadata only and touches no
// file on the target.
func (service *Service) ClearActiveBackup(targetID string) ([]BackupRecord, error) {
	return service.store.SetActiveBackup(targetID, "")
}

// DeleteBackup removes one backup file from the target and forgets its record.
// The file is removed first: a record without a file would present a backup the
// user can no longer restore.
func (service *Service) DeleteBackup(
	ctx context.Context, targetID string, backupID string,
) error {
	target, driver, err := service.driverFor(targetID)
	if err != nil {
		return err
	}
	defer driver.Close()
	records, err := service.store.ListBackups(targetID)
	if err != nil {
		return err
	}
	index := indexOfBackup(records, backupID)
	if index < 0 {
		return errors.New("the selected backup does not exist")
	}
	if err := driver.Remove(
		ctx, joinTargetPath(target, target.BackupDirectory(), records[index].FileName)); err != nil {
		return err
	}
	_, err = service.store.RemoveBackup(targetID, backupID)
	return err
}

// DownloadBackup copies one backup off the target into a local path the user
// chose in the native Save As dialog. The destination is never derived here: a
// download can only ever write where the host dialog already agreed to.
func (service *Service) DownloadBackup(
	ctx context.Context, targetID string, backupID string, localPath string,
) error {
	if localPath == "" {
		return errors.New("a backup download needs a target path")
	}
	target, driver, err := service.driverFor(targetID)
	if err != nil {
		return err
	}
	defer driver.Close()
	records, err := service.store.ListBackups(targetID)
	if err != nil {
		return err
	}
	index := indexOfBackup(records, backupID)
	if index < 0 {
		return errors.New("the selected backup does not exist")
	}
	return driver.CopyToLocal(
		ctx, joinTargetPath(target, target.BackupDirectory(), records[index].FileName), localPath)
}
