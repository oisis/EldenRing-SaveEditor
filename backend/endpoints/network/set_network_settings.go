/*
Endpoint: SetNetworkSettings
EndpointID: set_network_settings
Purpose: Atomically sets a complete validated network parameter set.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: —.
Input variables: saveSessionID, networkSettings, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package network

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

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
