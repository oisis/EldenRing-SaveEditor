/*
Endpoint: MoveOwnedItemsToInventory
EndpointID: move_owned_items_to_inventory
Purpose: Atomically moves several common Storage records into common Inventory as one save revision.
How it works: The runtime handler reads every addressed record through SaveEngine, resolves its one save-side game ID through the already loaded GameCatalog, derives the Inventory limit of each item from the shared Safety Profile policy and delegates one atomic batch mutation to SaveEngine. Each record is appended behind the records already held, in the order the caller listed them. The endpoint opens no file, parses no save data of its own and calls no other endpoint.
Supported resource types: ItemDocument with a known positive Inventory limit under the active profile.
Input variables: safetyProfile, saveSessionID, characterID, ownedItemIDs, expectedRevision.
GameCatalog variables read: item.storage.recordMode, item.storage.maxInventory and item.storage.safeModeMaxInventory of every addressed record.
Save variables processed: for every move the twelve bytes of the source Storage row, the Storage common count, the first free Inventory common row, the Inventory common count, Inventory NextAcquisitionSortId and the acquisition indices the insertion re-sorts; SaveEngine validates the complete batch against a private candidate image and finishes with full success or no change at all.
Implementation status: implemented
*/
package inventory

import (
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// MoveOwnedItemsToInventoryEndpointID is the stable backend identifier of MoveOwnedItemsToInventory.
const MoveOwnedItemsToInventoryEndpointID = "move_owned_items_to_inventory"

// MoveOwnedItemsToInventoryDefinition describes the public mutation contract.
var MoveOwnedItemsToInventoryDefinition = contract.MustDefine(contract.Definition{
	Name:                   "MoveOwnedItemsToInventory",
	ID:                     MoveOwnedItemsToInventoryEndpointID,
	Kind:                   contract.Mutation,
	SupportedResourceTypes: "ItemDocument with a known positive Inventory limit under the active profile",
	SupportedResourceVariables: []string{
		"safetyProfile", "saveSessionID", "characterID", "ownedItemIDs", "expectedRevision",
	},
	Description: "Atomically moves several common Storage records into common Inventory as one save revision.",
})

// MoveOwnedItemsToInventoryResult is the public name of the receipt SaveEngine
// owns. The endpoint adds no field, drops none and renames none.
type MoveOwnedItemsToInventoryResult = saveengine.MoveOwnedItemsResult

// MoveOwnedItemsToInventory moves every named Storage common record into common
// Inventory as one mutation of one revision.
//
// The batch either applies completely or changes nothing, and one receipt with
// one operationID describes the whole change. An empty list and a repeated
// ownedItemID are both rejected before the session is touched.
func MoveOwnedItemsToInventory(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	safetyProfile string,
	saveSessionID string,
	characterID int,
	ownedItemIDs []string,
	expectedRevision string,
) (MoveOwnedItemsToInventoryResult, error) {
	moves, err := resolveOwnedItemMoves(
		engine, gameCatalog, safetyProfile, saveSessionID, characterID, ownedItemIDs,
		saveengine.InventorySectionCommon)
	if err != nil {
		return MoveOwnedItemsToInventoryResult{}, err
	}
	return engine.MoveOwnedItemsToInventory(saveSessionID, characterID, moves, expectedRevision)
}
