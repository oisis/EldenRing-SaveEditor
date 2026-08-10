/*
Endpoint: GetQuickItems
EndpointID: get_quick_items
Purpose: Zwraca 10 surowych rekordów Quick Items jednego slotu postaci wraz z surowym aktywnym slotem, bez rozwiązywania ich w GameCatalog.
How it works: The runtime handler passes saveSessionID and characterID to SaveEngine, which reads one slot of the private snapshot of an already loaded session. The endpoint opens no file, reads no snapshot and parses no save data of its own.
Supported resource types: —.
Input variables: saveSessionID, characterID.
GameCatalog variables read: none; this stage returns raw state and resolves no ItemDocument.
Save variables read: the UserData10 activity flag of the requested slot and, for an active slot, the ten raw EquipItemData quick-item records and the raw active-slot value behind them; the getter is non-mutating and computes no value.
Implementation status: implemented.
*/
package equipment

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetQuickItemsEndpointID is the stable backend identifier of GetQuickItems.
const GetQuickItemsEndpointID = "get_quick_items"

// GetQuickItemsDefinition describes the public getter contract.
var GetQuickItemsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetQuickItems",
	ID:                         GetQuickItemsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Zwraca stan slotów Quick Items.",
})

// GetQuickItemsResult is the typed result of GetQuickItems. The shape is owned
// by SaveEngine, so the endpoint neither reshapes nor duplicates it. Every value
// stays exactly as stored in the save.
type GetQuickItemsResult = saveengine.CharacterQuickItems

// GetQuickItems returns the raw quick-item state stored in one character slot of
// an existing save session.
//
// The endpoint is thin: it rejects a missing engine and delegates everything
// else. Validating saveSessionID and characterID, reading the snapshot and
// deciding what an active, inactive or residual slot exposes belong to
// SaveEngine. The session must already exist; this endpoint never creates one,
// so it calls neither LoadSave nor any other endpoint, opens no file, reads no
// GameCatalog and returns no raw save byte.
func GetQuickItems(engine *saveengine.Engine, saveSessionID string, characterID int) (GetQuickItemsResult, error) {
	if engine == nil {
		return GetQuickItemsResult{}, errors.New("save engine is not available")
	}
	return engine.GetQuickItems(saveSessionID, characterID)
}
