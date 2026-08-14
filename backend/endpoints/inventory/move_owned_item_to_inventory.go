/*
Endpoint: MoveOwnedItemToInventory
EndpointID: move_owned_item_to_inventory
Purpose: Atomically moves a specific instance from Storage to Inventory.
How it works: The runtime handler resolves one common Storage record and its item document, rejects ambiguous key routing, derives the Inventory limit and delegates one atomic move to SaveEngine.
Supported resource types: ItemDocument with a known non-key category and positive maxInventory.
Input variables: saveSessionID, characterID, ownedItemID, targetPosition, expectedRevision.
GameCatalog variables read: item.gameID, item.category and item.storage.maxInventory.
Save variables processed: one Storage common record and count, one free Inventory common row and count, the acquisition indices needed to realise the requested logical position, and Inventory NextAcquisitionSortId; physical rows already in Inventory, NextEquipIndex, GaItem data and references stay unchanged.
Implementation status: implemented
*/
package inventory

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// MoveOwnedItemToInventoryEndpointID is the stable backend identifier of MoveOwnedItemToInventory.
const MoveOwnedItemToInventoryEndpointID = "move_owned_item_to_inventory"

// moveInventoryKeyCategory is ambiguous for a common-only destination: the
// catalog category contains both common-routed and key-routed resources.
const moveInventoryKeyCategory = "key_items"

// MoveOwnedItemToInventoryDefinition describes the public mutation contract.
var MoveOwnedItemToInventoryDefinition = contract.MustDefine(contract.Definition{
	Name:                       "MoveOwnedItemToInventory",
	ID:                         MoveOwnedItemToInventoryEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument with a known non-key category and positive maxInventory",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "ownedItemID", "targetPosition", "expectedRevision"},
	Description:                "Atomically moves a specific instance from Storage to Inventory.",
})

type MoveOwnedItemToInventoryResult = saveengine.MoveOwnedItemToInventoryResult

func MoveOwnedItemToInventory(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	ownedItemID string,
	targetPosition int,
	expectedRevision string,
) (MoveOwnedItemToInventoryResult, error) {
	if engine == nil {
		return MoveOwnedItemToInventoryResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return MoveOwnedItemToInventoryResult{}, errors.New("game catalog is not available")
	}

	owned, err := engine.GetOwnedItem(saveSessionID, characterID, ownedItemID)
	if err != nil {
		return MoveOwnedItemToInventoryResult{}, err
	}
	if owned.Container != "storage" {
		return MoveOwnedItemToInventoryResult{}, fmt.Errorf(
			"owned item %q is in %s, not Storage", ownedItemID, owned.Container)
	}
	if owned.ContainerSection != saveengine.StorageSectionCommon {
		return MoveOwnedItemToInventoryResult{}, fmt.Errorf(
			"owned item %q is in Storage section %q; moving key records to Inventory is not supported",
			ownedItemID, owned.ContainerSection)
	}

	gameIDs, err := engine.ResolveGaItemIDs(
		saveSessionID, characterID, []uint32{owned.GaItemHandle})
	if err != nil {
		return MoveOwnedItemToInventoryResult{}, err
	}
	gameID := gameIDs[0]
	resource, exists := gameCatalog.ItemByGameID(gameID)
	if !exists || resource.Kind != schema.ResourceKindItem || resource.Item == nil || resource.Key == "" {
		return MoveOwnedItemToInventoryResult{}, fmt.Errorf(
			"owned item %q: game ID 0x%08X is not a known item", ownedItemID, gameID)
	}
	if !resource.Item.Category.Known {
		return MoveOwnedItemToInventoryResult{}, fmt.Errorf(
			"owned item %q: item 0x%08X has an unknown category", ownedItemID, gameID)
	}
	if resource.Item.Category.Value == moveInventoryKeyCategory {
		return MoveOwnedItemToInventoryResult{}, fmt.Errorf(
			"owned item %q: item 0x%08X is in category %q, which does not distinguish common from"+
				" key routing; this common-only endpoint rejects the category fail-closed",
			ownedItemID, gameID, moveInventoryKeyCategory)
	}
	if !resource.Item.Storage.MaxInventory.Known || resource.Item.Storage.MaxInventory.Value == 0 {
		return MoveOwnedItemToInventoryResult{}, fmt.Errorf(
			"owned item %q: item 0x%08X cannot be carried in Inventory", ownedItemID, gameID)
	}

	return engine.MoveOwnedItemToInventory(
		saveSessionID,
		characterID,
		ownedItemID,
		targetPosition,
		expectedRevision,
		gameID,
		resource.Item.Storage.MaxInventory.Value,
	)
}
