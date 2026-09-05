/*
Endpoint: SetHostSettings
EndpointID: set_host_settings
Purpose: Stores the persistent host application settings and returns the settings now in effect.
How it works: The runtime handler validates the stated remote backup policy, writes the complete settings value atomically through the host settings store and returns the stored state. It touches no save session.
Supported resource types: —.
Input variables: skipReviewForNormalRisk, remoteBackupPolicy.
GameCatalog variables read: none.
Save variables processed: none; host settings never enter a save, a snapshot or a recovery journal.
Implementation status: implemented
*/
package application

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/hostsettings"
)

// SetHostSettingsEndpointID is the stable backend identifier of SetHostSettings.
const SetHostSettingsEndpointID = "set_host_settings"

// SetHostSettingsDefinition describes the public mutation contract.
var SetHostSettingsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetHostSettings",
	ID:                         SetHostSettingsEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"skipReviewForNormalRisk", "remoteBackupPolicy"},
	Description:                "Stores the persistent host application settings and returns the settings now in effect.",
})

// SetHostSettings stores a complete settings value and reports what is now in
// effect. It stores the whole value rather than a patch: a partial write would
// need a second, implicit source of truth for the fields it left out.
func SetHostSettings(
	store *hostsettings.Store,
	diagnosticService *diagnostics.Service,
	skipReviewForNormalRisk bool,
	remoteBackupPolicy string,
) (HostSettingsResult, error) {
	if store == nil {
		return HostSettingsResult{}, errors.New("host settings store is required")
	}
	settings, err := store.Set(skipReviewForNormalRisk, remoteBackupPolicy)
	if err != nil {
		return HostSettingsResult{}, err
	}
	return hostSettingsResult(store, diagnosticService, settings), nil
}
