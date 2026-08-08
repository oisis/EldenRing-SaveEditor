/*
Endpoint: GetNetworkPresets
EndpointID: get_network_presets
Purpose: Zwraca backendowe presety parametrów sieciowych.
How it works: The runtime handler builds the current backend presets directly from the preset functions of backend/core and returns them as a typed result. It reads no save, no GameCatalog and no application state, and it modifies nothing.
Supported resource types: —.
Input variables: presetID.
GameCatalog variables read: none.
Save variables read: none.
Implementation status: implemented.
*/
package network

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
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
	Description:                "Zwraca backendowe presety parametrów sieciowych.",
})

// NetworkPreset is one backend preset: its stable identifier and the full
// parameter set the preset stands for.
type NetworkPreset struct {
	ID         string                  `json:"id"`
	Parameters core.NetworkParamValues `json:"parameters"`
}

// GetNetworkPresetsResult is the typed result of GetNetworkPresets.
type GetNetworkPresetsResult struct {
	Presets []NetworkPreset `json:"presets"`
}

// GetNetworkPresets reports the network parameter presets the backend offers
// today.
//
// An empty presetID returns every preset in the order below. A non-empty
// presetID returns exactly the named preset, matched exactly and
// case-sensitively: the value is never trimmed, normalised or resolved through
// an alias, so an unknown identifier is an error instead of a silent fallback.
//
// The parameter values always come from the preset functions of backend/core,
// which own them; no value is copied here. The legacy presets of backend/core
// (fast-invasions, light-invasions, fast-summons, fast-blue, aggressive-host
// and NetworkParamFast) are deliberately not exposed.
func GetNetworkPresets(presetID string) (GetNetworkPresetsResult, error) {
	// Built per call, so a caller mutating one result cannot affect another.
	// ponytail: a local literal, not a package-level table; the order is the
	// contract and a shared slice would be mutable from outside.
	presets := []NetworkPreset{
		{ID: "vanilla", Parameters: core.NetworkParamDefaults()},
		{ID: "faster-reds", Parameters: core.NetworkParamFasterReds()},
		{ID: "aggressive-reds", Parameters: core.NetworkParamAggressiveReds()},
		{ID: "faster-summons", Parameters: core.NetworkParamFasterSummons()},
		{ID: "aggressive-summons", Parameters: core.NetworkParamAggressiveSummons()},
		{ID: "faster-blue", Parameters: core.NetworkParamFasterBlue()},
		{ID: "aggressive-blue", Parameters: core.NetworkParamAggressiveBlue()},
		{ID: "faster-summon-host", Parameters: core.NetworkParamFasterSummonHost()},
		{ID: "aggressive-summon-host", Parameters: core.NetworkParamAggressiveSummonHost()},
		{ID: "faster-summon-guest", Parameters: core.NetworkParamFasterSummonGuest()},
		{ID: "aggressive-summon-guest", Parameters: core.NetworkParamAggressiveSummonGuest()},
		{ID: "faster-hunter", Parameters: core.NetworkParamFasterHunter()},
		{ID: "aggressive-hunter", Parameters: core.NetworkParamAggressiveHunter()},
	}

	if presetID == "" {
		return GetNetworkPresetsResult{Presets: presets}, nil
	}

	for _, preset := range presets {
		if preset.ID == presetID {
			return GetNetworkPresetsResult{Presets: []NetworkPreset{preset}}, nil
		}
	}

	return GetNetworkPresetsResult{}, fmt.Errorf("unknown network preset %q", presetID)
}
