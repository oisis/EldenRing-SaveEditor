/*
Endpoint: UpdateTargetBackup
EndpointID: update_target_backup
Purpose: Replaces the tags and the description of one backup.
How it works: The runtime handler stores the new metadata. No file on the target is touched.
Supported resource types: —.
Input variables: targetID, backupID, tags, description.
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

// UpdateTargetBackupEndpointID is the stable backend identifier of UpdateTargetBackup.
const UpdateTargetBackupEndpointID = "update_target_backup"

// UpdateTargetBackupDefinition describes the public mutation contract.
var UpdateTargetBackupDefinition = contract.MustDefine(contract.Definition{
	Name:                       "UpdateTargetBackup",
	ID:                         UpdateTargetBackupEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID", "backupID", "tags", "description"},
	Description:                "Replaces the tags and the description of one backup.",
})

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
