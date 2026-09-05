/*
Endpoint: GetTargetBackups
EndpointID: get_target_backups
Purpose: Returns the backup library of one deployment target.
How it works: The runtime handler reads the host-owned backup metadata of the target and returns it newest first, together with the actions the backend supports for it. It contacts no target.
Supported resource types: —.
Input variables: targetID.
GameCatalog variables read: none.
Save variables read: none; backup metadata is host state.
Implementation status: implemented

Endpoint: CreateTargetBackup
EndpointID: create_target_backup
Purpose: Creates a manual backup of the current save of one target.
How it works: The runtime handler copies the target save to a fresh backup file beside it, verifies the copy exists and records it as a manual backup, which automatic retention never removes.
Supported resource types: —.
Input variables: targetID, tags, description.
GameCatalog variables processed: none.
Save variables processed: none of the open session; only the target's own file is copied.
Implementation status: implemented

Endpoint: ActivateTargetBackup
EndpointID: activate_target_backup
Purpose: Makes one backup the current save of its target.
How it works: The runtime handler blocks while the game is confirmed to be running, takes the mandatory backup of the current target save, atomically replaces the target with the selected backup, verifies the result and marks the backup active.
Supported resource types: —.
Input variables: targetID, backupID, confirmations.
GameCatalog variables processed: none.
Save variables processed: none of the open session; only the target's own file is replaced.
Implementation status: implemented

Endpoint: ClearActiveTargetBackup
EndpointID: clear_active_target_backup
Purpose: Removes the active mark from the backups of one target.
How it works: The runtime handler clears the metadata flag. No file on the target is touched.
Supported resource types: —.
Input variables: targetID.
GameCatalog variables processed: none.
Save variables processed: none.
Implementation status: implemented

Endpoint: UpdateTargetBackup
EndpointID: update_target_backup
Purpose: Replaces the tags and the description of one backup.
How it works: The runtime handler stores the new metadata. No file on the target is touched.
Supported resource types: —.
Input variables: targetID, backupID, tags, description.
GameCatalog variables processed: none.
Save variables processed: none.
Implementation status: implemented

Endpoint: DeleteTargetBackup
EndpointID: delete_target_backup
Purpose: Removes one backup from its target after an explicit confirmation.
How it works: The runtime handler deletes the backup file on the target and then forgets its record, so a record never survives the file it describes.
Supported resource types: —.
Input variables: targetID, backupID.
GameCatalog variables processed: none.
Save variables processed: none.
Implementation status: implemented

Endpoint: DownloadTargetBackup
EndpointID: download_target_backup
Purpose: Copies one backup off its target into a local path the user chose.
How it works: The runtime handler copies the backup file to the path the host's native Save As dialog returned. It derives no destination of its own and overwrites nothing the dialog did not agree to.
Supported resource types: —.
Input variables: targetID, backupID, target.
GameCatalog variables processed: none.
Save variables processed: none.
Implementation status: implemented
*/
package deployment

import (
	"context"
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// The stable backend identifiers of the Save Manager endpoints.
const (
	GetTargetBackupsEndpointID        = "get_target_backups"
	CreateTargetBackupEndpointID      = "create_target_backup"
	ActivateTargetBackupEndpointID    = "activate_target_backup"
	ClearActiveTargetBackupEndpointID = "clear_active_target_backup"
	UpdateTargetBackupEndpointID      = "update_target_backup"
	DeleteTargetBackupEndpointID      = "delete_target_backup"
	DownloadTargetBackupEndpointID    = "download_target_backup"
)

var (
	GetTargetBackupsDefinition = contract.MustDefine(contract.Definition{
		Name:                       "GetTargetBackups",
		ID:                         GetTargetBackupsEndpointID,
		Kind:                       contract.Getter,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID"},
		Description:                "Returns the backup library of one deployment target.",
	})
	CreateTargetBackupDefinition = contract.MustDefine(contract.Definition{
		Name:                       "CreateTargetBackup",
		ID:                         CreateTargetBackupEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID", "tags", "description"},
		Description:                "Creates a manual backup of the current save of one target.",
	})
	ActivateTargetBackupDefinition = contract.MustDefine(contract.Definition{
		Name:                       "ActivateTargetBackup",
		ID:                         ActivateTargetBackupEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID", "backupID"},
		Description:                "Makes one backup the current save of its target.",
	})
	ClearActiveTargetBackupDefinition = contract.MustDefine(contract.Definition{
		Name:                       "ClearActiveTargetBackup",
		ID:                         ClearActiveTargetBackupEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID"},
		Description:                "Removes the active mark from the backups of one target.",
	})
	UpdateTargetBackupDefinition = contract.MustDefine(contract.Definition{
		Name:                       "UpdateTargetBackup",
		ID:                         UpdateTargetBackupEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID", "backupID", "tags", "description"},
		Description:                "Replaces the tags and the description of one backup.",
	})
	DeleteTargetBackupDefinition = contract.MustDefine(contract.Definition{
		Name:                       "DeleteTargetBackup",
		ID:                         DeleteTargetBackupEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID", "backupID"},
		Description:                "Removes one backup from its target after an explicit confirmation.",
	})
	DownloadTargetBackupDefinition = contract.MustDefine(contract.Definition{
		Name:                       "DownloadTargetBackup",
		ID:                         DownloadTargetBackupEndpointID,
		Kind:                       contract.Mutation,
		SupportedResourceTypes:     "—",
		SupportedResourceVariables: []string{"targetID", "backupID", "target"},
		Description:                "Copies one backup off its target into a local path the user chose.",
	})
)

// BackupEntry is one backup as the Save Manager table renders it. It carries no
// size: section 4.10.4 of the frontend specification states the table has no
// Size column.
type BackupEntry = deployment.BackupRecord

// GetTargetBackupsResult is the typed result of GetTargetBackups.
//
// The capability flags are the backend's own statement of what it can do for
// this target, so the interface enables actions from the contract instead of
// from its own assumptions about a target kind.
type GetTargetBackupsResult struct {
	TargetID string        `json:"targetID"`
	Backups  []BackupEntry `json:"backups"`
	// TransferSupported is false while the target kind has no safe transfer
	// implementation. Every action below it is then unavailable.
	TransferSupported bool   `json:"transferSupported"`
	UnsupportedReason string `json:"unsupportedReason,omitempty"`
}

// GetTargetBackups reports the backup library of one target.
func GetTargetBackups(store *deployment.Store, targetID string) (GetTargetBackupsResult, error) {
	if store == nil {
		return GetTargetBackupsResult{}, errors.New("deployment store is required")
	}
	target, err := store.GetTarget(targetID)
	if err != nil {
		return GetTargetBackupsResult{}, err
	}
	backups, err := store.ListBackups(targetID)
	if err != nil {
		return GetTargetBackupsResult{}, err
	}
	if backups == nil {
		backups = []BackupEntry{}
	}
	return GetTargetBackupsResult{
		TargetID:          targetID,
		Backups:           backups,
		TransferSupported: deployment.TransferSupported(target.Kind),
	}, nil
}

// CreateTargetBackup creates one manual backup and reports the library.
func CreateTargetBackup(
	ctx context.Context,
	service *deployment.Service,
	store *deployment.Store,
	targetID string,
	tags []string,
	description string,
) (GetTargetBackupsResult, error) {
	if service == nil {
		return GetTargetBackupsResult{}, errors.New("deployment service is required")
	}
	if _, err := service.CreateManualBackup(ctx, targetID, tags, description); err != nil {
		return GetTargetBackupsResult{}, err
	}
	return GetTargetBackups(store, targetID)
}

// ActivateTargetBackupResult carries the operation outcome together with the
// library as it stands afterwards, so the caller never has to guess whether a
// blocked activation changed anything.
type ActivateTargetBackupResult struct {
	Operation OperationResult        `json:"operation"`
	Backups   GetTargetBackupsResult `json:"backups"`
}

// ActivateTargetBackup makes one backup the current save of its target.
func ActivateTargetBackup(
	ctx context.Context,
	service *deployment.Service,
	store *deployment.Store,
	operationID string,
	targetID string,
	backupID string,
	continueWithUnknownGameStatus bool,
	confirmRemoteBackup bool,
) (ActivateTargetBackupResult, error) {
	if service == nil {
		return ActivateTargetBackupResult{}, errors.New("deployment service is required")
	}
	operation, err := service.ActivateBackup(ctx, deployment.OperationRequest{
		OperationID:                   operationID,
		TargetID:                      targetID,
		ContinueWithUnknownGameStatus: continueWithUnknownGameStatus,
		ConfirmRemoteBackup:           confirmRemoteBackup,
	}, backupID)
	if err != nil {
		return ActivateTargetBackupResult{}, err
	}
	backups, err := GetTargetBackups(store, targetID)
	if err != nil {
		return ActivateTargetBackupResult{}, err
	}
	return ActivateTargetBackupResult{Operation: operation, Backups: backups}, nil
}

// ClearActiveTargetBackup removes the active mark.
func ClearActiveTargetBackup(
	service *deployment.Service, store *deployment.Store, targetID string,
) (GetTargetBackupsResult, error) {
	if service == nil {
		return GetTargetBackupsResult{}, errors.New("deployment service is required")
	}
	if _, err := service.ClearActiveBackup(targetID); err != nil {
		return GetTargetBackupsResult{}, err
	}
	return GetTargetBackups(store, targetID)
}

// UpdateTargetBackup replaces the tags and the description of one backup.
func UpdateTargetBackup(
	store *deployment.Store,
	targetID string,
	backupID string,
	tags []string,
	description string,
) (GetTargetBackupsResult, error) {
	if store == nil {
		return GetTargetBackupsResult{}, errors.New("deployment store is required")
	}
	if _, err := store.UpdateBackupMetadata(targetID, backupID, tags, description); err != nil {
		return GetTargetBackupsResult{}, err
	}
	return GetTargetBackups(store, targetID)
}

// DeleteTargetBackup removes one backup from its target.
func DeleteTargetBackup(
	ctx context.Context,
	service *deployment.Service,
	store *deployment.Store,
	targetID string,
	backupID string,
) (GetTargetBackupsResult, error) {
	if service == nil {
		return GetTargetBackupsResult{}, errors.New("deployment service is required")
	}
	if err := service.DeleteBackup(ctx, targetID, backupID); err != nil {
		return GetTargetBackupsResult{}, err
	}
	return GetTargetBackups(store, targetID)
}

// DownloadTargetBackupResult reports where a downloaded backup was written, so
// the caller can offer to open it in the editor as a separate explicit action.
type DownloadTargetBackupResult struct {
	TargetID string `json:"targetID"`
	BackupID string `json:"backupID"`
	Target   string `json:"target"`
}

// DownloadTargetBackup copies one backup to the path the host dialog returned.
func DownloadTargetBackup(
	ctx context.Context,
	service *deployment.Service,
	targetID string,
	backupID string,
	target string,
) (DownloadTargetBackupResult, error) {
	if service == nil {
		return DownloadTargetBackupResult{}, errors.New("deployment service is required")
	}
	if err := service.DownloadBackup(ctx, targetID, backupID, target); err != nil {
		return DownloadTargetBackupResult{}, err
	}
	return DownloadTargetBackupResult{TargetID: targetID, BackupID: backupID, Target: target}, nil
}
