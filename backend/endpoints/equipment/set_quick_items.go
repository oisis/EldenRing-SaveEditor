/*
Endpoint: SetQuickItems
EndpointID: set_quick_items
Purpose: Atomically sets the contents of Quick Items slots.
How it works: The runtime handler validates the complete request and expected revision, validates every non-empty ownedItemID assignment, checks for equipment capability allowing slot quick_item via GameCatalog, and delegates one atomic mutation to SaveEngine under expectedRevision control.
Supported resource types: ItemDocument of family goods with capability equipment allowing slot quick_item.
Input variables: saveSessionID, characterID, slotAssignments, expectedRevision.
GameCatalog variables read: item.family, item.gameID, item.capabilities.equipment.
Save variables processed: the ten Quick Items records in EquipItemData and equipped-armaments tail fields; SaveEngine validates the complete plan and finishes with full success or rollback.
Implementation status: implemented
*/
package equipment

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// SetQuickItemsEndpointID is the stable backend identifier of SetQuickItems.
const SetQuickItemsEndpointID = "set_quick_items"

// SetQuickItemsDefinition describes the public mutation contract.
var SetQuickItemsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetQuickItems",
	ID:                         SetQuickItemsEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument of family goods with capability equipment allowing slot quick_item",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "slotAssignments", "expectedRevision"},
	Description:                "Atomically sets the contents of Quick Items slots.",
})

// SetQuickItemsResult is the typed result of SetQuickItems.
type SetQuickItemsResult struct {
	SaveSessionID   string                  `json:"saveSessionID"`
	SaveRevision    string                  `json:"saveRevision"`
	CharacterID     int                     `json:"characterID"`
	SlotAssignments [10]*schema.ResourceRef `json:"slotAssignments"`
}

// SetQuickItems replaces all ten Quick Items positions of one active character.
// slotAssignments must contain exactly ten entries; nil clears that position.
func SetQuickItems(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	slotAssignments []*string,
	expectedRevision string,
) (SetQuickItemsResult, error) {
	if engine == nil {
		return SetQuickItemsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetQuickItemsResult{}, errors.New("game catalog is not available")
	}
	if len(slotAssignments) != 10 {
		return SetQuickItemsResult{}, fmt.Errorf(
			"slotAssignments must contain exactly 10 positions; got %d", len(slotAssignments))
	}

	var fixedAssignments [10]*string
	copy(fixedAssignments[:], slotAssignments)

	validator := func(gameID uint32) error {
		resource, found := gameCatalog.ItemByGameID(gameID)
		if !found || resource.Item == nil {
			return fmt.Errorf("item with game ID 0x%08X is not found in game catalog", gameID)
		}
		item := resource.Item
		if !item.Family.Known || item.Family.Value != schema.ItemFamilyGoods {
			return fmt.Errorf(
				"resource kind %q key %q has item family %q, want %q",
				resource.Kind, resource.Key, item.Family.Value, schema.ItemFamilyGoods)
		}
		equipment := item.Capabilities.Equipment
		if !equipment.Known || !equipment.Enabled || equipment.Rules == nil {
			return fmt.Errorf(
				"resource kind %q key %q has no confirmed quick-item equipment capability",
				resource.Kind, resource.Key)
		}
		for _, slot := range equipment.Rules.AllowedSlots {
			if slot == schema.EquipmentSlotQuickItem {
				return nil
			}
		}
		return fmt.Errorf(
			"resource kind %q key %q cannot be equipped in a quick-item slot",
			resource.Kind, resource.Key)
	}

	mutation, err := engine.SetQuickItems(
		saveSessionID, characterID, fixedAssignments, expectedRevision, validator)
	if err != nil {
		return SetQuickItemsResult{}, err
	}

	var resolved [10]*schema.ResourceRef
	for i, gameID := range mutation.GameIDs {
		if gameID == saveengine.QuickItemEmptyGameID {
			continue
		}
		resource, found := gameCatalog.ItemByGameID(gameID)
		if !found {
			return SetQuickItemsResult{}, fmt.Errorf(
				"committed game ID 0x%08X could not be found in game catalog", gameID)
		}
		ref := resource.Ref()
		resolved[i] = &ref
	}

	return SetQuickItemsResult{
		SaveSessionID:   mutation.SaveSessionID,
		SaveRevision:    mutation.SaveRevision,
		CharacterID:     mutation.CharacterID,
		SlotAssignments: resolved,
	}, nil
}
