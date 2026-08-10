/*
Endpoint: ApplyNetworkPreset
EndpointID: apply_network_preset
Purpose: Applies a backend preset through the same domain operation as SetNetworkSettings.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: —.
Input variables: saveSessionID, presetID, expectedRevision.
GameCatalog variables read: none required by the current contract.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package network

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

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
