/*
Endpoint: SetInventoryOrder
EndpointID: set_inventory_order
Purpose: Sets the complete order of supported Inventory instances without changing their semantic contents.
How it works: The runtime handler classifies Inventory records through GameCatalog and delegates one complete ordered identity permutation to SaveEngine.
Supported resource types: ItemDocument in a confirmed Inventory order category, excluding the technical Unarmed record.
Input variables: saveSessionID, characterID, orderedOwnedItemIDs, expectedRevision.
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

// SetInventoryOrderEndpointID is the stable backend identifier of SetInventoryOrder.
const SetInventoryOrderEndpointID = "set_inventory_order"

const itemOrderUnarmedKey = "0001ADB0"

// SetInventoryOrderDefinition describes the public mutation contract.
var SetInventoryOrderDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetInventoryOrder",
	ID:                         SetInventoryOrderEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument in a confirmed Inventory order category, excluding the technical Unarmed record",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "orderedOwnedItemIDs", "expectedRevision"},
	Description:                "Sets the complete order of supported Inventory instances without changing their semantic contents.",
})

// SetInventoryOrderResult reports the committed order in stable catalog terms.
//
// The receipt is the one the SaveEngine commit path produced, embedded
// anonymously so the JSON stays flat and carries no nested receipt object.
type SetInventoryOrderResult struct {
	saveengine.MutationReceipt
	CharacterID        int                  `json:"characterID"`
	OrderedResources   []schema.ResourceRef `json:"orderedResources"`
	AcquisitionIndices []uint32             `json:"acquisitionIndices"`
}

// SetInventoryOrder replaces the complete supported order of Inventory common.
func SetInventoryOrder(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	orderedOwnedItemIDs []string,
	expectedRevision string,
) (SetInventoryOrderResult, error) {
	if engine == nil {
		return SetInventoryOrderResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetInventoryOrderResult{}, errors.New("game catalog is not available")
	}

	mutation, err := engine.SetInventoryOrder(
		saveSessionID, characterID, orderedOwnedItemIDs, expectedRevision,
		func(gameID uint32) (bool, error) {
			return supportsItemOrder(gameCatalog, gameID)
		})
	if err != nil {
		return SetInventoryOrderResult{}, err
	}

	resources := make([]schema.ResourceRef, len(mutation.GameIDs))
	for index, gameID := range mutation.GameIDs {
		resource, found := gameCatalog.ItemByGameID(gameID)
		if !found {
			return SetInventoryOrderResult{}, fmt.Errorf(
				"committed game ID 0x%08X could not be found in game catalog", gameID)
		}
		resources[index] = resource.Ref()
	}

	return SetInventoryOrderResult{
		MutationReceipt:    mutation.MutationReceipt,
		CharacterID:        mutation.CharacterID,
		OrderedResources:   resources,
		AcquisitionIndices: mutation.AcquisitionIndices,
	}, nil
}

func supportsItemOrder(gameCatalog *gamecatalog.Catalog, gameID uint32) (bool, error) {
	resource, found := gameCatalog.ItemByGameID(gameID)
	if !found || resource.Kind != schema.ResourceKindItem ||
		resource.Item == nil || resource.Key == "" {
		return false, fmt.Errorf(
			"item with game ID 0x%08X is not found in game catalog", gameID)
	}
	if !resource.Item.Category.Known {
		return false, fmt.Errorf(
			"resource kind %q key %q has an unknown category", resource.Kind, resource.Key)
	}
	if resource.Key == itemOrderUnarmedKey {
		return false, nil
	}
	switch resource.Item.Category.Value {
	case "melee_armaments", "ranged_and_catalysts", "shields", "talismans",
		"head", "chest", "arms", "legs":
		return true, nil
	default:
		return false, nil
	}
}
