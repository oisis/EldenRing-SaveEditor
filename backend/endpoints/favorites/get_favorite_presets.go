/*
Endpoint: GetFavoritePresets
EndpointID: get_favorite_presets
Purpose: Zwraca zapisane presety Favorites i ich przypisania.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: —.
Input variables: favoriteSlotID.
GameCatalog variables read: none required by the current contract.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package favorites

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetFavoritePresetsEndpointID is the stable backend identifier of GetFavoritePresets.
const GetFavoritePresetsEndpointID = "get_favorite_presets"

// GetFavoritePresetsDefinition describes the public getter contract.
var GetFavoritePresetsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetFavoritePresets",
	ID:                         GetFavoritePresetsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"favoriteSlotID"},
	Description:                "Zwraca zapisane presety Favorites i ich przypisania.",
})
