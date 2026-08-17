/*
Endpoint: ApplyFavoritePreset
EndpointID: apply_favorite_preset
Purpose: Applies the specified Mirror Favorites preset to an active character.
How it works: The runtime handler validates the save engine and delegates the atomic mutation to SaveEngine under expectedRevision control. The endpoint reads no GameCatalog, requires no selection object, and modifies nothing directly.
Supported resource types: —.
Input variables: saveSessionID, characterID, favoriteSlotID, expectedRevision.
GameCatalog variables read: none.
Save variables processed: the appearance fields of the active character represented by Mirror Favorites; VoiceType and opaque FaceData remain untouched, gender is updated from the inverted preset body type, and the mutation records a standard single undo point for the character.
Implementation status: implemented
*/
package favorites

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// ApplyFavoritePresetEndpointID is the stable backend identifier of ApplyFavoritePreset.
const ApplyFavoritePresetEndpointID = "apply_favorite_preset"

// ApplyFavoritePresetDefinition describes the public mutation contract.
var ApplyFavoritePresetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "ApplyFavoritePreset",
	ID:                         ApplyFavoritePresetEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "favoriteSlotID", "expectedRevision"},
	Description:                "Applies the specified Mirror Favorites preset to an active character.",
})

// ApplyFavoritePresetResult is the typed receipt of ApplyFavoritePreset.
type ApplyFavoritePresetResult = saveengine.ApplyFavoritePresetResult

// ApplyFavoritePreset applies the specified Mirror Favorites preset to an active character.
func ApplyFavoritePreset(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	favoriteSlotID int,
	expectedRevision string,
) (ApplyFavoritePresetResult, error) {
	if engine == nil {
		return ApplyFavoritePresetResult{}, errors.New("save engine is not available")
	}
	return engine.ApplyFavoritePreset(saveSessionID, characterID, favoriteSlotID, expectedRevision)
}
