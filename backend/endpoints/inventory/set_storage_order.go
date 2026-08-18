/*
Endpoint: SetStorageOrder
EndpointID: set_storage_order
Purpose: Sets the complete order of supported Storage instances without changing their semantic contents.
How it works: The runtime handler classifies Storage records through GameCatalog and delegates one complete ordered identity permutation to SaveEngine.
Supported resource types: ItemDocument in a confirmed Storage order category, excluding the technical Unarmed record.
Input variables: saveSessionID, characterID, orderedOwnedItemIDs, expectedRevision.
GameCatalog variables read: item.gameID and item.category.
Save variables processed: acquisition indices of every supported Storage common record and Storage NextAcquisitionSortId (set to the next free bucket); physical rows, handles, quantities, key records, NextEquipIndex, Inventory, Equipment and GaItem data stay unchanged.
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

// SetStorageOrderEndpointID is the stable backend identifier of SetStorageOrder.
const SetStorageOrderEndpointID = "set_storage_order"

// SetStorageOrderDefinition describes the public mutation contract.
var SetStorageOrderDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetStorageOrder",
	ID:                         SetStorageOrderEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument in a confirmed Storage order category, excluding the technical Unarmed record",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "orderedOwnedItemIDs", "expectedRevision"},
	Description:                "Sets the complete order of supported Storage instances without changing their semantic contents.",
})

// SetStorageOrderResult reports the committed order in stable catalog terms.
type SetStorageOrderResult struct {
	SaveSessionID      string               `json:"saveSessionID"`
	SaveRevision       string               `json:"saveRevision"`
	CharacterID        int                  `json:"characterID"`
	OrderedResources   []schema.ResourceRef `json:"orderedResources"`
	AcquisitionIndices []uint32             `json:"acquisitionIndices"`
}

// SetStorageOrder replaces the complete supported order of Storage common.
func SetStorageOrder(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	orderedOwnedItemIDs []string,
	expectedRevision string,
) (SetStorageOrderResult, error) {
	if engine == nil {
		return SetStorageOrderResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetStorageOrderResult{}, errors.New("game catalog is not available")
	}

	mutation, err := engine.SetStorageOrder(
		saveSessionID, characterID, orderedOwnedItemIDs, expectedRevision,
		func(gameID uint32) (bool, error) {
			return supportsItemOrder(gameCatalog, gameID)
		})
	if err != nil {
		return SetStorageOrderResult{}, err
	}

	resources := make([]schema.ResourceRef, len(mutation.GameIDs))
	for index, gameID := range mutation.GameIDs {
		resource, found := gameCatalog.ItemByGameID(gameID)
		if !found {
			return SetStorageOrderResult{}, fmt.Errorf(
				"committed game ID 0x%08X could not be found in game catalog", gameID)
		}
		resources[index] = resource.Ref()
	}

	return SetStorageOrderResult{
		SaveSessionID:      mutation.SaveSessionID,
		SaveRevision:       mutation.SaveRevision,
		CharacterID:        mutation.CharacterID,
		OrderedResources:   resources,
		AcquisitionIndices: mutation.AcquisitionIndices,
	}, nil
}
