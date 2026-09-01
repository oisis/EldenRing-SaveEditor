/*
Endpoint: ApplyNetworkPreset
EndpointID: apply_network_preset
Purpose: Applies a backend preset through the same domain operation as SetNetworkSettings.
How it works: The runtime handler resolves presetID exactly in the loaded GameCatalog and delegates its complete parameter set to the same SaveEngine operation as SetNetworkSettings.
Supported resource types: —.
Input variables: saveSessionID, presetID, expectedRevision.
GameCatalog variables read: the ID and complete 22-field parameter set of one network preset from regulation/network_params.json.
Save variables processed: the 22 NetworkParam values in UserData11 through SaveEngine.SetNetworkSettings; no separate writer exists here.
Implementation status: implemented
*/
package network

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// ApplyNetworkPresetEndpointID is the stable backend identifier of ApplyNetworkPreset.
const ApplyNetworkPresetEndpointID = "apply_network_preset"

// ApplyNetworkPresetDefinition describes the public mutation contract.
var ApplyNetworkPresetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "ApplyNetworkPreset",
	ID:                         ApplyNetworkPresetEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "presetID", "expectedRevision"},
	Description:                "Applies a backend preset through the same domain operation as SetNetworkSettings.",
})

// ApplyNetworkPresetResult reports the selected preset and the committed save
// revision. NetworkSettings is the complete set passed to SaveEngine.
type ApplyNetworkPresetResult struct {
	SaveSessionID   string                         `json:"saveSessionID"`
	SaveRevision    string                         `json:"saveRevision"`
	PresetID        string                         `json:"presetID"`
	NetworkSettings gamecatalog.NetworkParamValues `json:"networkSettings"`
}

// ApplyNetworkPreset resolves one backend preset and applies it through the
// same SaveEngine writer as direct settings, under this endpoint's own
// operation kind. It owns no binary rule and does not recognise the legacy
// aliases absent from GameCatalog.
func ApplyNetworkPreset(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	presetID string,
	expectedRevision string,
) (ApplyNetworkPresetResult, error) {
	if engine == nil {
		return ApplyNetworkPresetResult{}, errors.New("save engine is not available")
	}
	presets, err := networkPresets(gameCatalog)
	if err != nil {
		return ApplyNetworkPresetResult{}, err
	}
	preset, err := findNetworkPreset(presets, presetID)
	if err != nil {
		return ApplyNetworkPresetResult{}, err
	}
	committed, err := engine.ApplyNetworkPreset(
		saveSessionID, preset.Parameters, expectedRevision)
	if err != nil {
		return ApplyNetworkPresetResult{}, err
	}
	return ApplyNetworkPresetResult{
		SaveSessionID:   committed.SaveSessionID,
		SaveRevision:    committed.SaveRevision,
		PresetID:        preset.ID,
		NetworkSettings: committed.NetworkSettings,
	}, nil
}
