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
*/
package deployment

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
)

// GetTargetBackupsEndpointID is the stable backend identifier of GetTargetBackups.
const GetTargetBackupsEndpointID = "get_target_backups"

// GetTargetBackupsDefinition describes the public getter contract.
var GetTargetBackupsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetTargetBackups",
	ID:                         GetTargetBackupsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"targetID"},
	Description:                "Returns the backup library of one deployment target.",
})

// BackupEntry is one backup as the Save Manager table renders it. It carries no
// size: section 4.10.4 of the frontend specification states the table has no
// Size column.
type BackupEntry = deployment.BackupRecord

// GetTargetBackupsResult is the typed result of GetTargetBackups. Every Save
// Manager mutation below reports the library through it.
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
