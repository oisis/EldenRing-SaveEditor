/*
Endpoint: GetNetworkPresets
EndpointID: get_network_presets
Purpose: Zwraca backendowe presety parametrów sieciowych.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: —.
Input variables: presetID.
GameCatalog variables read: none required by the current contract.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package network

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

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
