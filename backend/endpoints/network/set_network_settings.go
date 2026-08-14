/*
Endpoint: SetNetworkSettings
EndpointID: set_network_settings
Purpose: Atomically sets a complete validated network parameter set.
How it works: The runtime handler delegates the complete request to SaveEngine, which validates every value and expected revision before atomically replacing the NetworkParam row.
Supported resource types: —.
Input variables: saveSessionID, networkSettings, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: the 22 NetworkParam values in UserData11; no other regulation row or save field is changed.
Implementation status: implemented
*/
package network

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetNetworkSettingsEndpointID is the stable backend identifier of SetNetworkSettings.
const SetNetworkSettingsEndpointID = "set_network_settings"

// SetNetworkSettingsDefinition describes the public mutation contract.
var SetNetworkSettingsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetNetworkSettings",
	ID:                         SetNetworkSettingsEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "networkSettings", "expectedRevision"},
	Description:                "Atomically sets a complete validated network parameter set.",
})

// SetNetworkSettingsResult is the SaveEngine mutation receipt.
type SetNetworkSettingsResult = saveengine.SetNetworkSettingsResult

// SetNetworkSettings replaces the complete supported network parameter set of
// one loaded session. SaveEngine owns validation, revision and binary-format
// rules; the endpoint has no GameCatalog dependency.
func SetNetworkSettings(
	engine *saveengine.Engine,
	saveSessionID string,
	networkSettings gamecatalog.NetworkParamValues,
	expectedRevision string,
) (SetNetworkSettingsResult, error) {
	if engine == nil {
		return SetNetworkSettingsResult{}, errors.New("save engine is not available")
	}
	return engine.SetNetworkSettings(saveSessionID, networkSettings, expectedRevision)
}
