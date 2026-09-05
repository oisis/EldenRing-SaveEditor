/*
Endpoint: DeleteTargetBackup
EndpointID: delete_target_backup
Purpose: Removes one backup from its target after an explicit confirmation.
How it works: The runtime handler deletes the backup file on the target and then forgets its record, so a record never survives the file it describes.
Supported resource types: —.
Input variables: targetID, backupID.
GameCatalog variables read: none.
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

// DeleteTargetBackupEndpointID is the stable backend identifier of DeleteTargetBackup.
const DeleteTargetBackupEndpointID = "delete_target_backup"

// DeleteTargetBackupDefinition describes the public mutation contract.
var DeleteTargetBackupDefinition = contract.MustDefine(contract.Definition{
	Name:                       "DeleteTargetBackup",
	ID:                         DeleteTargetBackupEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID", "backupID"},
	Description:                "Removes one backup from its target after an explicit confirmation.",
})

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
