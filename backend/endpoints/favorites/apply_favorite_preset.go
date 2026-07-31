/*
Endpoint: ApplyFavoritePreset
EndpointID: apply_favorite_preset
Purpose: Stosuje wskazany preset Favorites do postaci przez standardowe operacje domenowe.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: GameResource references.
Input variables: characterID, favoriteSlotID, selection, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package favorites

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// ApplyFavoritePresetEndpointID is the stable backend identifier of ApplyFavoritePreset.
const ApplyFavoritePresetEndpointID = "apply_favorite_preset"

// ApplyFavoritePresetDefinition describes the public mutation contract.
var ApplyFavoritePresetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "ApplyFavoritePreset",
	ID:                         ApplyFavoritePresetEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"characterID", "favoriteSlotID", "selection", "expectedRevision"},
	Description:                "Stosuje wskazany preset Favorites do postaci przez standardowe operacje domenowe.",
})
