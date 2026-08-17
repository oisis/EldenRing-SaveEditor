/*
Endpoint: DeleteFavoritePreset
EndpointID: delete_favorite_preset
Purpose: Deletes the specified Favorites preset without changing other slots.
How it works: The runtime handler validates the save engine and delegates the atomic mutation to SaveEngine under expectedRevision control. The endpoint reads no GameCatalog, requires no characterID, and modifies nothing directly.
Supported resource types: —.
Input variables: saveSessionID, favoriteSlotID, expectedRevision.
GameCatalog variables read: none.
Save variables processed: the specified global Mirror Favorites slot in UserData10; active slots are zeroed (0x130 bytes) with verification and rollback.
Implementation status: implemented
*/
package favorites

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// DeleteFavoritePresetEndpointID is the stable backend identifier of DeleteFavoritePreset.
const DeleteFavoritePresetEndpointID = "delete_favorite_preset"

// DeleteFavoritePresetDefinition describes the public mutation contract.
var DeleteFavoritePresetDefinition = contract.MustDefine(contract.Definition{
	Name:                       "DeleteFavoritePreset",
	ID:                         DeleteFavoritePresetEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "favoriteSlotID", "expectedRevision"},
	Description:                "Deletes the specified Favorites preset without changing other slots.",
})

// DeleteFavoritePresetResult is the typed receipt of DeleteFavoritePreset.
type DeleteFavoritePresetResult = saveengine.DeleteFavoritePresetResult

// DeleteFavoritePreset clears the specified Mirror Favorites preset slot in an
// existing save session.
func DeleteFavoritePreset(
	engine *saveengine.Engine,
	saveSessionID string,
	favoriteSlotID int,
	expectedRevision string,
) (DeleteFavoritePresetResult, error) {
	if engine == nil {
		return DeleteFavoritePresetResult{}, errors.New("save engine is not available")
	}
	return engine.DeleteFavoritePreset(saveSessionID, favoriteSlotID, expectedRevision)
}
