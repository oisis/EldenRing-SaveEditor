/*
Endpoint: SetFavoritePreset
EndpointID: set_favorite_preset
Purpose: Saves or replaces the specified Favorites preset with all appearance fields represented by Mirror Favorites from an active character.
How it works: The runtime handler validates the save engine and delegates the atomic mutation to SaveEngine under expectedRevision control. The endpoint reads no GameCatalog, requires no selection object, and modifies nothing directly.
Supported resource types: —.
Input variables: saveSessionID, favoriteSlotID, sourceCharacterID, expectedRevision.
GameCatalog variables read: none.
Save variables processed: the specified global Mirror Favorites slot in UserData10; the slot is populated with the complete 0x130-byte preset buffer derived from the active character's appearance fields supported by Mirror Favorites.
Implementation status: implemented
*/
package favorites

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetFavoritePresetEndpointID is the stable backend identifier of SetFavoritePreset.
const SetFavoritePresetEndpointID = "set_favorite_preset"

// SetFavoritePresetDefinition describes the public mutation contract.
var SetFavoritePresetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetFavoritePreset",
	ID:                         SetFavoritePresetEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "favoriteSlotID", "sourceCharacterID", "expectedRevision"},
	Description:                "Saves or replaces the specified Favorites preset with all appearance fields represented by Mirror Favorites from an active character.",
})

// SetFavoritePresetResult is the typed receipt of SetFavoritePreset.
type SetFavoritePresetResult = saveengine.SetFavoritePresetResult

// SetFavoritePreset saves all appearance fields represented by Mirror Favorites
// from an active character into the specified preset slot in an existing save session.
func SetFavoritePreset(
	engine *saveengine.Engine,
	saveSessionID string,
	favoriteSlotID int,
	sourceCharacterID int,
	expectedRevision string,
) (SetFavoritePresetResult, error) {
	if engine == nil {
		return SetFavoritePresetResult{}, errors.New("save engine is not available")
	}
	return engine.SetFavoritePreset(saveSessionID, favoriteSlotID, sourceCharacterID, expectedRevision)
}
