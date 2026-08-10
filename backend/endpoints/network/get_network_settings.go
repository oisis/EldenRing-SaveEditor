/*
Endpoint: GetNetworkSettings
EndpointID: get_network_settings
Purpose: Returns the current network parameters stored in the save.
How it works: The runtime handler passes saveSessionID to SaveEngine, which reads the UserData11 regulation of an already loaded session and returns the 22 stored network parameters. The endpoint opens no file, reads no snapshot, parses no save data of its own, reads no GameCatalog and compares nothing against a preset.
Supported resource types: —.
Input variables: saveSessionID.
GameCatalog variables read: none; gamecatalog.NetworkParamValues is used as the shared typed model only.
Save variables read: the 22 NetworkParam values stored in UserData11 of the session; the getter is non-mutating and normalises no value.
Implementation status: implemented.
*/
package network

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetNetworkSettingsEndpointID is the stable backend identifier of GetNetworkSettings.
const GetNetworkSettingsEndpointID = "get_network_settings"

// GetNetworkSettingsDefinition describes the public getter contract.
var GetNetworkSettingsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetNetworkSettings",
	ID:                         GetNetworkSettingsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID"},
	Description:                "Returns the current network parameters stored in the save.",
})

// GetNetworkSettingsResult is the typed result of GetNetworkSettings: the
// session that was read and the parameter set stored in it. The parameter model
// is the one GameCatalog already owns, so no second 22-field model exists.
type GetNetworkSettingsResult struct {
	SaveSessionID string                         `json:"saveSessionID"`
	Parameters    gamecatalog.NetworkParamValues `json:"parameters"`
}

// GetNetworkSettings returns the network parameters stored in the save of an
// existing session.
//
// The endpoint is thin: it rejects a missing engine and delegates everything
// else. Validating saveSessionID, locating UserData11 on PC and PS4 and decoding
// the regulation belong to SaveEngine. The session must already exist; this
// endpoint never creates one, so it calls neither LoadSave nor any other
// endpoint, opens no file and returns no raw save byte.
//
// The result is the state of the loaded save: no value is validated against a
// preset range, normalised or replaced by a default, and a save whose UserData11
// cannot be decoded produces an error instead of a partial or fallback parameter
// set.
func GetNetworkSettings(engine *saveengine.Engine, saveSessionID string) (GetNetworkSettingsResult, error) {
	if engine == nil {
		return GetNetworkSettingsResult{}, errors.New("save engine is not available")
	}
	parameters, err := engine.GetNetworkSettings(saveSessionID)
	if err != nil {
		return GetNetworkSettingsResult{}, err
	}
	return GetNetworkSettingsResult{SaveSessionID: saveSessionID, Parameters: parameters}, nil
}
