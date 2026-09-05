/*
Endpoint: CreateTargetBackup
EndpointID: create_target_backup
Purpose: Creates a manual backup of the current save of one target.
How it works: The runtime handler copies the target save to a fresh backup file beside it, verifies the copy exists and records it as a manual backup, which automatic retention never removes.
Supported resource types: —.
Input variables: targetID, tags, description.
GameCatalog variables read: none.
Save variables processed: none of the open session; only the target's own file is copied.
Implementation status: implemented
*/
package deployment

import (
	"context"
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// CreateTargetBackupEndpointID is the stable backend identifier of CreateTargetBackup.
const CreateTargetBackupEndpointID = "create_target_backup"

// CreateTargetBackupDefinition describes the public mutation contract.
var CreateTargetBackupDefinition = contract.MustDefine(contract.Definition{
	Name:                       "CreateTargetBackup",
	ID:                         CreateTargetBackupEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID", "tags", "description"},
	Description:                "Creates a manual backup of the current save of one target.",
})

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
