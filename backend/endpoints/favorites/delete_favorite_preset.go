/*
Endpoint: DeleteFavoritePreset
EndpointID: delete_favorite_preset
Purpose: Deletes the specified Favorites preset without changing other slots.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: favoriteSlotID, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package favorites

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// DeleteFavoritePresetEndpointID is the stable backend identifier of DeleteFavoritePreset.
const DeleteFavoritePresetEndpointID = "delete_favorite_preset"

// DeleteFavoritePresetDefinition describes the public mutation contract.
var DeleteFavoritePresetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "DeleteFavoritePreset",
	ID:                         DeleteFavoritePresetEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"favoriteSlotID", "expectedRevision"},
	Description:                "Deletes the specified Favorites preset without changing other slots.",
})
