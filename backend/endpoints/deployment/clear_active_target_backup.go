/*
Endpoint: ClearActiveTargetBackup
EndpointID: clear_active_target_backup
Purpose: Removes the active mark from the backups of one target.
How it works: The runtime handler clears the metadata flag. No file on the target is touched.
Supported resource types: —.
Input variables: targetID.
GameCatalog variables read: none.
Save variables processed: none.
Implementation status: implemented
*/
package deployment

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// ClearActiveTargetBackupEndpointID is the stable backend identifier of ClearActiveTargetBackup.
const ClearActiveTargetBackupEndpointID = "clear_active_target_backup"

// ClearActiveTargetBackupDefinition describes the public mutation contract.
var ClearActiveTargetBackupDefinition = contract.MustDefine(contract.Definition{
	Name:                       "ClearActiveTargetBackup",
	ID:                         ClearActiveTargetBackupEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID"},
	Description:                "Removes the active mark from the backups of one target.",
})

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
