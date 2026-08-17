/*
Endpoint: GetFavoritePresets
EndpointID: get_favorite_presets
Purpose: Returns the occupancy state of the 15 global Mirror Favorites appearance preset slots stored in UserData10.
How it works: The runtime handler passes saveSessionID and optional favoriteSlotID to SaveEngine, which reads the preset slots from the private snapshot of an already loaded session. The endpoint opens no file, reads no snapshot, parses no save data of its own, reads no GameCatalog and modifies nothing.
Supported resource types: —.
Input variables: saveSessionID, favoriteSlotID.
GameCatalog variables read: none.
Save variables read: the 15 global Mirror Favorites slots in UserData10; the getter is non-mutating.
Implementation status: implemented
*/
package favorites

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetFavoritePresetsEndpointID is the stable backend identifier of GetFavoritePresets.
const GetFavoritePresetsEndpointID = "get_favorite_presets"

// GetFavoritePresetsDefinition describes the public getter contract.
var GetFavoritePresetsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetFavoritePresets",
	ID:                         GetFavoritePresetsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "favoriteSlotID"},
	Description:                "Returns the occupancy state of the 15 global Mirror Favorites appearance preset slots stored in UserData10.",
})

// FavoritePreset is the occupancy state of one Mirror Favorites preset slot.
type FavoritePreset = saveengine.FavoritePreset

// GetFavoritePresetsResult is the typed result of GetFavoritePresets: the session
// that was read and the list of preset slot occupancy records.
type GetFavoritePresetsResult = saveengine.FavoritePresetsState

// GetFavoritePresets returns the occupancy state of Mirror Favorites preset slots
// from an existing save session.
//
// The endpoint is thin: it rejects a missing engine and delegates everything
// else to SaveEngine.
func GetFavoritePresets(
	engine *saveengine.Engine,
	saveSessionID string,
	favoriteSlotID *int,
) (GetFavoritePresetsResult, error) {
	if engine == nil {
		return GetFavoritePresetsResult{}, errors.New("save engine is not available")
	}
	return engine.GetFavoritePresets(saveSessionID, favoriteSlotID)
}
