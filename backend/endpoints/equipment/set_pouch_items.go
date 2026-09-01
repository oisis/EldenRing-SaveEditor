/*
Endpoint: SetPouchItems
EndpointID: set_pouch_items
Purpose: Atomically sets the contents of Pouch slots.
How it works: The runtime handler validates the complete request and expected revision, validates every non-empty ownedItemID assignment, checks for equipment capability allowing slot pouch via GameCatalog, and delegates one atomic mutation to SaveEngine under expectedRevision control.
Supported resource types: ItemDocument of family goods with capability equipment allowing slot pouch.
Input variables: saveSessionID, characterID, slotAssignments, expectedRevision.
GameCatalog variables read: item.family, item.gameID, item.capabilities.equipment.
Save variables processed: the six pouch records in EquipItemData and equipped-armaments tail fields; SaveEngine validates the complete plan and finishes with full success or rollback.
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

// SetPouchItemsEndpointID is the stable backend identifier of SetPouchItems.
const SetPouchItemsEndpointID = "set_pouch_items"

// SetPouchItemsDefinition describes the public mutation contract.
var SetPouchItemsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetPouchItems",
	ID:                         SetPouchItemsEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument of family goods with capability equipment allowing slot pouch",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "slotAssignments", "expectedRevision"},
	Description:                "Atomically sets the contents of Pouch slots.",
})

// SetPouchItemsResult is the typed result of SetPouchItems.
//
// The receipt is the one the SaveEngine commit path produced, embedded
// anonymously so the JSON stays flat and carries no nested receipt object.
type SetPouchItemsResult struct {
	saveengine.MutationReceipt
	CharacterID     int                    `json:"characterID"`
	SlotAssignments [6]*schema.ResourceRef `json:"slotAssignments"`
}

// SetPouchItems replaces the six Pouch slot positions of one active character.
// slotAssignments must contain exactly six entries; nil clears that exact position.
func SetPouchItems(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	slotAssignments []*string,
	expectedRevision string,
) (SetPouchItemsResult, error) {
	if engine == nil {
		return SetPouchItemsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetPouchItemsResult{}, errors.New("game catalog is not available")
	}
	if len(slotAssignments) != 6 {
		return SetPouchItemsResult{}, fmt.Errorf(
			"slotAssignments must contain exactly 6 positions; got %d", len(slotAssignments))
	}

	var fixedAssignments [6]*string
	copy(fixedAssignments[:], slotAssignments)

	validator := func(gameID uint32) error {
		resource, found := gameCatalog.ItemByGameID(gameID)
		if !found || resource.Item == nil {
			return fmt.Errorf(
				"item with game ID 0x%08X is not found in game catalog", gameID)
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
				"resource kind %q key %q has no confirmed pouch equipment capability",
				resource.Kind, resource.Key)
		}
		allowed := false
		for _, slot := range equipment.Rules.AllowedSlots {
			if slot == schema.EquipmentSlotPouch {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf(
				"resource kind %q key %q cannot be equipped in the pouch slot",
				resource.Kind, resource.Key)
		}
		return nil
	}

	mutation, err := engine.SetPouchItems(
		saveSessionID, characterID, fixedAssignments, expectedRevision, validator)
	if err != nil {
		return SetPouchItemsResult{}, err
	}

	var resolvedRefs [6]*schema.ResourceRef
	for i, gameID := range mutation.GameIDs {
		if gameID == saveengine.PouchEmptyGameID {
			resolvedRefs[i] = nil
			continue
		}
		resource, found := gameCatalog.ItemByGameID(gameID)
		if !found {
			return SetPouchItemsResult{}, fmt.Errorf(
				"committed game ID 0x%08X could not be found in game catalog", gameID)
		}
		ref := resource.Ref()
		resolvedRefs[i] = &ref
	}

	return SetPouchItemsResult{
		MutationReceipt: mutation.MutationReceipt,
		CharacterID:     mutation.CharacterID,
		SlotAssignments: resolvedRefs,
	}, nil
}
