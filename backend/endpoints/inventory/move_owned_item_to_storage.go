/*
Endpoint: MoveOwnedItemToStorage
EndpointID: move_owned_item_to_storage
Purpose: Atomically moves one common Inventory record to common Storage.
How it works: The runtime handler resolves the addressed record and its item document, derives the Storage limit and delegates one atomic move to SaveEngine.
Supported resource types: ItemDocument with a known positive maxStorage.
Input variables: saveSessionID, characterID, ownedItemID, targetPosition, expectedRevision.
GameCatalog variables read: item.gameID and item.storage.maxStorage.
Save variables processed: one Inventory common record and count, the affected common Storage rows and count, and Storage acquisition allocators (NextEquipIndex and NextAcquisitionSortId); GaItem data and references stay unchanged.
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

// MoveOwnedItemToStorageEndpointID is the stable backend identifier of MoveOwnedItemToStorage.
const MoveOwnedItemToStorageEndpointID = "move_owned_item_to_storage"

// MoveOwnedItemToStorageDefinition describes the public mutation contract.
var MoveOwnedItemToStorageDefinition = contract.MustDefine(contract.Definition{
	Name:                       "MoveOwnedItemToStorage",
	ID:                         MoveOwnedItemToStorageEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument with a known positive maxStorage",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "ownedItemID", "targetPosition", "expectedRevision"},
	Description:                "Atomically moves one common Inventory record to common Storage.",
})

type MoveOwnedItemToStorageResult = saveengine.MoveOwnedItemToStorageResult

func MoveOwnedItemToStorage(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	ownedItemID string,
	targetPosition int,
	expectedRevision string,
) (MoveOwnedItemToStorageResult, error) {
	if engine == nil {
		return MoveOwnedItemToStorageResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return MoveOwnedItemToStorageResult{}, errors.New("game catalog is not available")
	}

	owned, err := engine.GetOwnedItem(saveSessionID, characterID, ownedItemID)
	if err != nil {
		return MoveOwnedItemToStorageResult{}, err
	}
	if owned.Container != "inventory" {
		return MoveOwnedItemToStorageResult{}, fmt.Errorf(
			"owned item %q is in %s, not Inventory", ownedItemID, owned.Container)
	}
	if owned.ContainerSection != saveengine.InventorySectionCommon {
		return MoveOwnedItemToStorageResult{}, fmt.Errorf(
			"owned item %q is in Inventory section %q; moving key records to Storage is not supported",
			ownedItemID, owned.ContainerSection)
	}

	gameIDs, err := engine.ResolveGaItemIDs(
		saveSessionID, characterID, []uint32{owned.GaItemHandle})
	if err != nil {
		return MoveOwnedItemToStorageResult{}, err
	}
	gameID := gameIDs[0]
	resource, exists := gameCatalog.ItemByGameID(gameID)
	if !exists || resource.Kind != schema.ResourceKindItem || resource.Item == nil || resource.Key == "" {
		return MoveOwnedItemToStorageResult{}, fmt.Errorf(
			"owned item %q: game ID 0x%08X is not a known item", ownedItemID, gameID)
	}
	if !resource.Item.Storage.MaxStorage.Known || resource.Item.Storage.MaxStorage.Value == 0 {
		return MoveOwnedItemToStorageResult{}, fmt.Errorf(
			"owned item %q: item 0x%08X cannot be stored", ownedItemID, gameID)
	}
	if !resource.Item.Storage.RecordMode.Known {
		return MoveOwnedItemToStorageResult{}, fmt.Errorf(
			"owned item %q: item 0x%08X has an unknown record mode", ownedItemID, gameID)
	}
	var separateInstances bool
	switch resource.Item.Storage.RecordMode.Value {
	case schema.RecordModeQuantityStack:
		separateInstances = false
	case schema.RecordModeSeparateInstances:
		separateInstances = true
	default:
		return MoveOwnedItemToStorageResult{}, fmt.Errorf(
			"owned item %q: item 0x%08X has unsupported record mode %q",
			ownedItemID, gameID, resource.Item.Storage.RecordMode.Value)
	}

	return engine.MoveOwnedItemToStorage(
		saveSessionID,
		characterID,
		ownedItemID,
		targetPosition,
		expectedRevision,
		gameID,
		resource.Item.Storage.MaxStorage.Value,
		separateInstances,
	)
}
