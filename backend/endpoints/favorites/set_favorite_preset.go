/*
Endpoint: SetFavoritePreset
EndpointID: set_favorite_preset
Purpose: Saves or replaces the specified Favorites preset with validated character data.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: favoriteSlotID, sourceCharacterID, selection, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package favorites

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetFavoritePresetEndpointID is the stable backend identifier of SetFavoritePreset.
const SetFavoritePresetEndpointID = "set_favorite_preset"

// SetFavoritePresetDefinition describes the public mutation contract.
var SetFavoritePresetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetFavoritePreset",
	ID:                         SetFavoritePresetEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"favoriteSlotID", "sourceCharacterID", "selection", "expectedRevision"},
	Description:                "Saves or replaces the specified Favorites preset with validated character data.",
})
