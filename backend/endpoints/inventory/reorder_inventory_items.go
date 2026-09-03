/*
Endpoint: ReorderInventoryItems
EndpointID: reorder_inventory_items
Purpose: Moves a selected group of supported Inventory instances to a new position, anchored on one member of the group, as one atomic save revision.
How it works: The runtime handler classifies Inventory records through GameCatalog with the same rule SetInventoryOrder uses, and delegates the anchored placement plus one complete ordered identity permutation to SaveEngine. The group keeps its internal order, the selected records that were in front of the anchor stay in front of it and the ones behind it stay behind it. The endpoint opens no file and parses no save data of its own.
Supported resource types: ItemDocument in a confirmed Inventory order category, excluding the technical Unarmed record.
Input variables: saveSessionID, characterID, anchorOwnedItemID, groupOwnedItemIDs, targetPosition, expectedRevision.
GameCatalog variables read: item.gameID and item.category.
Save variables processed: acquisition indices of every supported Inventory common record and Inventory NextAcquisitionSortId; physical rows, handles, quantities, key records, NextEquipIndex, Equipment, Storage and GaItem data stay unchanged.
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

// ReorderInventoryItemsEndpointID is the stable backend identifier of ReorderInventoryItems.
const ReorderInventoryItemsEndpointID = "reorder_inventory_items"

// ReorderInventoryItemsDefinition describes the public mutation contract.
var ReorderInventoryItemsDefinition = contract.MustDefine(contract.Definition{
	Name:                   "ReorderInventoryItems",
	ID:                     ReorderInventoryItemsEndpointID,
	Kind:                   contract.Mutation,
	SupportedResourceTypes: "ItemDocument in a confirmed Inventory order category, excluding the technical Unarmed record",
	SupportedResourceVariables: []string{
		"saveSessionID", "characterID", "anchorOwnedItemID", "groupOwnedItemIDs",
		"targetPosition", "expectedRevision",
	},
	Description: "Moves a selected group of supported Inventory instances to a new position, anchored on one member of the group.",
})

// ReorderInventoryItemsResult reports the committed order in stable catalog
// terms, beside the identities the caller supplied.
//
// The receipt is the one the SaveEngine commit path produced, embedded
// anonymously so the JSON stays flat and carries no nested receipt object.
type ReorderInventoryItemsResult struct {
	saveengine.MutationReceipt
	CharacterID        int                  `json:"characterID"`
	OrderedResources   []schema.ResourceRef `json:"orderedResources"`
	AcquisitionIndices []uint32             `json:"acquisitionIndices"`
}

// ReorderInventoryItems places the selected group at targetPosition, anchored on
// anchorOwnedItemID.
//
// targetPosition is the zero-based position the anchor takes in the resulting
// supported Inventory order. Storage has no manual order and is deliberately
// not addressable here.
//
// The mutation is atomic: SaveEngine plans the complete permutation, validates
// it against a private candidate image of the snapshot and commits one revision,
// one history entry and one receipt. A group member that is not a supported
// Inventory common record, an anchor outside the group, a repeated identity and
// a position the group cannot occupy are all rejected before anything changes.
func ReorderInventoryItems(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	anchorOwnedItemID string,
	groupOwnedItemIDs []string,
	targetPosition int,
	expectedRevision string,
) (ReorderInventoryItemsResult, error) {
	if engine == nil {
		return ReorderInventoryItemsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return ReorderInventoryItemsResult{}, errors.New("game catalog is not available")
	}

	mutation, err := engine.ReorderInventoryItems(
		saveSessionID, characterID, anchorOwnedItemID, groupOwnedItemIDs, targetPosition,
		expectedRevision,
		func(gameID uint32) (bool, error) {
			return supportsItemOrder(gameCatalog, gameID)
		})
	if err != nil {
		return ReorderInventoryItemsResult{}, err
	}

	resources := make([]schema.ResourceRef, len(mutation.GameIDs))
	for index, gameID := range mutation.GameIDs {
		resource, found := gameCatalog.ItemByGameID(gameID)
		if !found {
			return ReorderInventoryItemsResult{}, fmt.Errorf(
				"committed game ID 0x%08X could not be found in game catalog", gameID)
		}
		resources[index] = resource.Ref()
	}

	return ReorderInventoryItemsResult{
		MutationReceipt:    mutation.MutationReceipt,
		CharacterID:        mutation.CharacterID,
		OrderedResources:   resources,
		AcquisitionIndices: mutation.AcquisitionIndices,
	}, nil
}
