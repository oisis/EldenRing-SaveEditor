/*
Endpoint: GetNetworkSettings
EndpointID: get_network_settings
Purpose: Zwraca aktualne parametry sieciowe save.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: —.
Input variables: saveSessionID.
GameCatalog variables read: none required by the current contract.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package network

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetNetworkSettingsEndpointID is the stable backend identifier of GetNetworkSettings.
const GetNetworkSettingsEndpointID = "get_network_settings"

// GetNetworkSettingsDefinition describes the public getter contract.
var GetNetworkSettingsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetNetworkSettings",
	ID:                         GetNetworkSettingsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID"},
	Description:                "Zwraca aktualne parametry sieciowe save.",
})
