/*
Endpoint: DownloadTargetBackup
EndpointID: download_target_backup
Purpose: Copies one backup off its target into a local path the user chose.
How it works: The runtime handler copies the backup file to the path the host's native Save As dialog returned. It derives no destination of its own and overwrites nothing the dialog did not agree to.
Supported resource types: —.
Input variables: targetID, backupID, target.
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

// DownloadTargetBackupEndpointID is the stable backend identifier of DownloadTargetBackup.
const DownloadTargetBackupEndpointID = "download_target_backup"

// DownloadTargetBackupDefinition describes the public mutation contract.
var DownloadTargetBackupDefinition = contract.MustDefine(contract.Definition{
	Name:                       "DownloadTargetBackup",
	ID:                         DownloadTargetBackupEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID", "backupID", "target"},
	Description:                "Copies one backup off its target into a local path the user chose.",
})

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
