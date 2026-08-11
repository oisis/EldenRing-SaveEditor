/*
Endpoint: GetPouchItems
EndpointID: get_pouch_items
Purpose: Returns the 6 raw Pouch records of one character slot without resolving them through GameCatalog.
How it works: The runtime handler passes saveSessionID and characterID to SaveEngine, which reads one slot of the private snapshot of an already loaded session. The endpoint opens no file, reads no snapshot and parses no save data of its own.
Supported resource types: —.
Input variables: saveSessionID, characterID.
GameCatalog variables read: none; this stage returns raw state and resolves no ItemDocument.
Save variables read: the UserData10 activity flag of the requested slot and, for an active slot, the six raw EquipItemData pouch records that follow the quick items and the active-slot value; the getter is non-mutating and computes no value.
Implementation status: implemented
*/
package equipment

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetPouchItemsEndpointID is the stable backend identifier of GetPouchItems.
const GetPouchItemsEndpointID = "get_pouch_items"

// GetPouchItemsDefinition describes the public getter contract.
var GetPouchItemsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetPouchItems",
	ID:                         GetPouchItemsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Returns the state of Pouch slots.",
})

// GetPouchItemsResult is the typed result of GetPouchItems. The shape is owned
// by SaveEngine, so the endpoint neither reshapes nor duplicates it. Every value
// stays exactly as stored in the save.
type GetPouchItemsResult = saveengine.CharacterPouchItems

// GetPouchItems returns the raw pouch state stored in one character slot of an
// existing save session.
//
// The endpoint is thin: it rejects a missing engine and delegates everything
// else. Validating saveSessionID and characterID, reading the snapshot and
// deciding what an active, inactive or residual slot exposes belong to
// SaveEngine. The session must already exist; this endpoint never creates one,
// so it calls neither LoadSave nor any other endpoint, opens no file, reads no
// GameCatalog and returns no raw save byte.
func GetPouchItems(engine *saveengine.Engine, saveSessionID string, characterID int) (GetPouchItemsResult, error) {
	if engine == nil {
		return GetPouchItemsResult{}, errors.New("save engine is not available")
	}
	return engine.GetPouchItems(saveSessionID, characterID)
}
