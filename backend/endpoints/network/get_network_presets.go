/*
Endpoint: GetNetworkPresets
EndpointID: get_network_presets
Purpose: Returns backend network parameter presets.
How it works: The runtime handler reads the network presets of the already loaded GameCatalog, which owns and validates backend/gamecatalog/data/regulation/network_params.json, and returns them as a typed result. It reads no save and no file of its own, and it modifies nothing.
Supported resource types: —.
Input variables: presetID.
GameCatalog variables read: network presets of regulation/network_params.json.
Save variables read: none.
Implementation status: implemented
*/
package network

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
)

// GetNetworkPresetsEndpointID is the stable backend identifier of GetNetworkPresets.
const GetNetworkPresetsEndpointID = "get_network_presets"

// GetNetworkPresetsDefinition describes the public getter contract.
var GetNetworkPresetsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetNetworkPresets",
	ID:                         GetNetworkPresetsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"presetID"},
	Description:                "Returns backend network parameter presets.",
})

// NetworkPreset is one backend preset: its stable identifier and the full
// parameter set the preset stands for. The catalog owns the type, so the public
// JSON shape stays identical to the stored data.
type NetworkPreset = gamecatalog.NetworkPreset

// GetNetworkPresetsResult is the typed result of GetNetworkPresets.
type GetNetworkPresetsResult struct {
	Presets []NetworkPreset `json:"presets"`
}

// GetNetworkPresets reports the network parameter presets the backend offers
// today, read from the loaded GameCatalog.
//
// An empty presetID returns every preset in catalog order, which starts with the
// default preset vanilla. A non-empty presetID returns exactly the named preset,
// matched exactly and case-sensitively: the value is never trimmed, normalised
// or resolved through an alias, so an unknown identifier is an error instead of
// a silent fallback.
//
// The parameter values come from regulation/network_params.json, which the
// catalog loads and validates once; no value is copied here and no file is read
// per call. Presets that existed only in SaveForge 1.x are deliberately not
// part of that data and are therefore unknown identifiers.
func GetNetworkPresets(
	gameCatalog *gamecatalog.Catalog,
	presetID string,
) (GetNetworkPresetsResult, error) {
	presets, err := networkPresets(gameCatalog)
	if err != nil {
		return GetNetworkPresetsResult{}, err
	}

	if presetID == "" {
		return GetNetworkPresetsResult{Presets: presets}, nil
	}

	preset, err := findNetworkPreset(presets, presetID)
	if err != nil {
		return GetNetworkPresetsResult{}, err
	}
	return GetNetworkPresetsResult{Presets: []NetworkPreset{preset}}, nil
}

func networkPresets(gameCatalog *gamecatalog.Catalog) ([]NetworkPreset, error) {
	if gameCatalog == nil {
		return nil, errors.New("game catalog is not loaded")
	}
	// The catalog returns an independent copy, so a caller mutating one result
	// cannot affect another.
	return gameCatalog.NetworkPresets()
}

// findNetworkPreset is the single exact-ID rule shared by listing one preset
// and applying it. It deliberately has no aliases, trimming or fallback.
func findNetworkPreset(presets []NetworkPreset, presetID string) (NetworkPreset, error) {
	if presetID == "" {
		return NetworkPreset{}, errors.New("presetID is required")
	}
	for _, preset := range presets {
		if preset.ID == presetID {
			return preset, nil
		}
	}
	return NetworkPreset{}, fmt.Errorf("unknown network preset %q", presetID)
}
