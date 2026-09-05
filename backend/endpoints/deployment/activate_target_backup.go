/*
Endpoint: ActivateTargetBackup
EndpointID: activate_target_backup
Purpose: Makes one backup the current save of its target.
How it works: The runtime handler blocks while the game is confirmed to be running, takes the mandatory backup of the current target save, atomically replaces the target with the selected backup, verifies the result and marks the backup active.
Supported resource types: —.
Input variables: targetID, backupID, confirmations.
GameCatalog variables read: none.
Save variables processed: none of the open session; only the target's own file is replaced.
Implementation status: implemented
*/
package deployment

import (
	"context"
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// ActivateTargetBackupEndpointID is the stable backend identifier of ActivateTargetBackup.
const ActivateTargetBackupEndpointID = "activate_target_backup"

// ActivateTargetBackupDefinition describes the public mutation contract.
var ActivateTargetBackupDefinition = contract.MustDefine(contract.Definition{
	Name:                       "ActivateTargetBackup",
	ID:                         ActivateTargetBackupEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID", "backupID"},
	Description:                "Makes one backup the current save of its target.",
})

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
