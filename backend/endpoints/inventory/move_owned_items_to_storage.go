/*
Endpoint: MoveOwnedItemsToStorage
EndpointID: move_owned_items_to_storage
Purpose: Atomically moves several common Inventory records into common Storage as one save revision.
How it works: The runtime handler reads every addressed record through SaveEngine, resolves its one save-side game ID through the already loaded GameCatalog, derives the Storage limit of each item from the shared Safety Profile policy and delegates one atomic batch mutation to SaveEngine. Each record is appended behind the records already in Storage, in the order the caller listed them. The endpoint opens no file, parses no save data of its own and calls no other endpoint.
Supported resource types: ItemDocument with a known positive Storage limit under the active profile.
Input variables: safetyProfile, saveSessionID, characterID, ownedItemIDs, expectedRevision.
GameCatalog variables read: item.storage.recordMode, item.storage.maxStorage and item.storage.safeModeMaxStorage of every addressed record.
Save variables processed: for every move the twelve bytes of the source Inventory row, the Inventory common count, the common Storage section image, the Storage common count and both trailing Storage allocators; SaveEngine validates the complete batch against a private candidate image and finishes with full success or no change at all.
Implementation status: implemented
*/
package inventory

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/safetyprofile"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// MoveOwnedItemsToStorageEndpointID is the stable backend identifier of MoveOwnedItemsToStorage.
const MoveOwnedItemsToStorageEndpointID = "move_owned_items_to_storage"

// MoveOwnedItemsToStorageDefinition describes the public mutation contract.
var MoveOwnedItemsToStorageDefinition = contract.MustDefine(contract.Definition{
	Name:                   "MoveOwnedItemsToStorage",
	ID:                     MoveOwnedItemsToStorageEndpointID,
	Kind:                   contract.Mutation,
	SupportedResourceTypes: "ItemDocument with a known positive Storage limit under the active profile",
	SupportedResourceVariables: []string{
		"safetyProfile", "saveSessionID", "characterID", "ownedItemIDs", "expectedRevision",
	},
	Description: "Atomically moves several common Inventory records into common Storage as one save revision.",
})

// MoveOwnedItemsToStorageResult is the public name of the receipt SaveEngine
// owns. The endpoint adds no field, drops none and renames none.
type MoveOwnedItemsToStorageResult = saveengine.MoveOwnedItemsResult

// MoveOwnedItemsToStorage moves every named Inventory common record into common
// Storage as one mutation of one revision.
//
// The batch either applies completely or changes nothing, and one receipt with
// one operationID describes the whole change. An empty list and a repeated
// ownedItemID are both rejected before the session is touched.
func MoveOwnedItemsToStorage(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	safetyProfile string,
	saveSessionID string,
	characterID int,
	ownedItemIDs []string,
	expectedRevision string,
) (MoveOwnedItemsToStorageResult, error) {
	moves, err := resolveOwnedItemMoves(
		engine, gameCatalog, safetyProfile, saveSessionID, characterID, ownedItemIDs,
		saveengine.StorageSectionCommon)
	if err != nil {
		return MoveOwnedItemsToStorageResult{}, err
	}
	return engine.MoveOwnedItemsToStorage(saveSessionID, characterID, moves, expectedRevision)
}

// resolveOwnedItemMoves resolves the catalog facts both batch moves need. It is
// the one place either direction reads: the destination decides which container
// limit applies and which source container is valid, and nothing else differs.
//
// destinationSection is the section the records are moving into, which is what
// selects the destination limit. It is one of the two common sections.
func resolveOwnedItemMoves(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	safetyProfile string,
	saveSessionID string,
	characterID int,
	ownedItemIDs []string,
	destinationSection string,
) ([]saveengine.OwnedItemMove, error) {
	if engine == nil {
		return nil, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return nil, errors.New("game catalog is not available")
	}
	profile, err := safetyprofile.Parse(safetyProfile)
	if err != nil {
		return nil, err
	}
	if len(ownedItemIDs) == 0 {
		return nil, errors.New("ownedItemIDs must not be empty")
	}

	toStorage := destinationSection == saveengine.StorageSectionCommon
	sourceContainer := "storage"
	sourceSection := saveengine.StorageSectionCommon
	if toStorage {
		sourceContainer = "inventory"
		sourceSection = saveengine.InventorySectionCommon
	}

	moves := make([]saveengine.OwnedItemMove, 0, len(ownedItemIDs))
	for index, ownedItemID := range ownedItemIDs {
		owned, err := engine.GetOwnedItem(saveSessionID, characterID, ownedItemID)
		if err != nil {
			return nil, fmt.Errorf("ownedItemIDs[%d]: %w", index, err)
		}
		if owned.Container != sourceContainer {
			return nil, fmt.Errorf(
				"ownedItemIDs[%d]: owned item %q is in %s, not %s",
				index, ownedItemID, owned.Container, sourceContainer)
		}
		if owned.ContainerSection != sourceSection {
			return nil, fmt.Errorf(
				"ownedItemIDs[%d]: owned item %q is in section %q; moving key records is not supported",
				index, ownedItemID, owned.ContainerSection)
		}

		gameIDs, err := engine.ResolveGaItemIDs(
			saveSessionID, characterID, []uint32{owned.GaItemHandle})
		if err != nil {
			return nil, fmt.Errorf("ownedItemIDs[%d]: %w", index, err)
		}
		gameID := gameIDs[0]
		resource, exists := gameCatalog.ItemByGameID(gameID)
		if !exists || resource.Kind != schema.ResourceKindItem ||
			resource.Item == nil || resource.Key == "" {
			return nil, fmt.Errorf(
				"ownedItemIDs[%d]: owned item %q: game ID 0x%08X is not a known item",
				index, ownedItemID, gameID)
		}
		item := resource.Item
		var limit uint32
		var known bool
		if toStorage {
			limit, known = safetyprofile.StorageLimit(profile, item)
		} else {
			limit, known = safetyprofile.InventoryLimit(profile, item)
		}
		if !known || limit == 0 {
			return nil, fmt.Errorf(
				"ownedItemIDs[%d]: owned item %q: item 0x%08X carries no limit for the destination container",
				index, ownedItemID, gameID)
		}
		if !item.Storage.RecordMode.Known {
			return nil, fmt.Errorf(
				"ownedItemIDs[%d]: owned item %q: item 0x%08X has an unknown record mode",
				index, ownedItemID, gameID)
		}
		var separateInstances bool
		switch item.Storage.RecordMode.Value {
		case schema.RecordModeQuantityStack:
			separateInstances = false
		case schema.RecordModeSeparateInstances:
			separateInstances = true
		default:
			return nil, fmt.Errorf(
				"ownedItemIDs[%d]: owned item %q: item 0x%08X has unsupported record mode %q",
				index, ownedItemID, gameID, item.Storage.RecordMode.Value)
		}

		moves = append(moves, saveengine.OwnedItemMove{
			OwnedItemID:       ownedItemID,
			ExpectedGameID:    gameID,
			MaxQuantity:       limit,
			SeparateInstances: separateInstances,
		})
	}
	return moves, nil
}
