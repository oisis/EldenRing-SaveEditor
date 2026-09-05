/*
Endpoint: GetHostSettings
EndpointID: get_host_settings
Purpose: Returns the persistent host application settings and the host directories the Settings screen can open.
How it works: The runtime handler reads the injected host settings store and reports its values together with the configuration directory it owns and the log directory the diagnostic service owns. It reads no save and mutates nothing.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: none.
Save variables read: none; host settings are not save state.
Implementation status: implemented
*/
package application

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/hostsettings"
)

// GetHostSettingsEndpointID is the stable backend identifier of GetHostSettings.
const GetHostSettingsEndpointID = "get_host_settings"

// GetHostSettingsDefinition describes the public getter contract.
var GetHostSettingsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetHostSettings",
	ID:                         GetHostSettingsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: nil,
	Description:                "Returns the persistent host application settings and the host directories the Settings screen can open.",
})

// HostSettingsResult reports the stored settings, the closed policy vocabulary
// and the two host directories the Settings screen offers to open.
//
// The directories are reported so the frontend can state whether the action is
// available at all; the frontend never sends a directory back and never builds
// one of its own. A host running without a state directory reports both as
// empty, which is a truthful "not available", not a hidden failure.
//
// GetHostSettings and SetHostSettings return the same value, so the type and
// the hostSettingsResult builder live with the getter that reads it first.
type HostSettingsResult struct {
	SkipReviewForNormalRisk       bool     `json:"skipReviewForNormalRisk"`
	RemoteBackupPolicy            string   `json:"remoteBackupPolicy"`
	AvailableRemoteBackupPolicies []string `json:"availableRemoteBackupPolicies"`
	DefaultRemoteBackupPolicy     string   `json:"defaultRemoteBackupPolicy"`
	ConfigurationDirectoryExists  bool     `json:"configurationDirectoryExists"`
	LogDirectoryExists            bool     `json:"logDirectoryExists"`
}

// GetHostSettings reports the stored host settings.
//
// diagnosticService is optional and is read only for the log directory: Debug
// Mode itself is never reported or stored here, so the diagnostic flag keeps
// exactly one owner.
func GetHostSettings(
	store *hostsettings.Store, diagnosticService *diagnostics.Service,
) (HostSettingsResult, error) {
	if store == nil {
		return HostSettingsResult{}, errors.New("host settings store is required")
	}
	settings, err := store.Get()
	if err != nil {
		return HostSettingsResult{}, err
	}
	return hostSettingsResult(store, diagnosticService, settings), nil
}

func hostSettingsResult(
	store *hostsettings.Store,
	diagnosticService *diagnostics.Service,
	settings hostsettings.Settings,
) HostSettingsResult {
	policies := hostsettings.RemoteBackupPolicies()
	available := make([]string, 0, len(policies))
	for _, policy := range policies {
		available = append(available, string(policy))
	}
	return HostSettingsResult{
		SkipReviewForNormalRisk:       settings.SkipReviewForNormalRisk,
		RemoteBackupPolicy:            string(settings.RemoteBackupPolicy),
		AvailableRemoteBackupPolicies: available,
		DefaultRemoteBackupPolicy:     string(hostsettings.DefaultRemoteBackupPolicy),
		ConfigurationDirectoryExists:  store.Directory() != "",
		LogDirectoryExists:            diagnosticService.Directory() != "",
	}
}
