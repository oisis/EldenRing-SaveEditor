/*
Endpoint: SetEquippedArmaments
EndpointID: set_equipped_armaments
Purpose: Atomically sets armaments in every hand slot and validates slot types and the existence of owned instances.
How it works: The runtime handler validates exactly six owned-item assignments in left 1, right 1, left 2, right 2, left 3 and right 3 order, checks every non-empty record through GameCatalog, and delegates one atomic mutation to SaveEngine.
Supported resource types: ItemDocument of family weapon with capability equipment allowing the assigned left-hand or right-hand slot.
Input variables: saveSessionID, characterID, slotAssignments, expectedRevision.
GameCatalog variables read: item.family, item.gameID and item.capabilities.equipment.
Save variables processed: the six hand-armament fields in all four native representations; SaveEngine validates the complete plan and finishes with full success or rollback.
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

// SetEquippedArmamentsEndpointID is the stable backend identifier of SetEquippedArmaments.
const SetEquippedArmamentsEndpointID = "set_equipped_armaments"

// SetEquippedArmamentsDefinition describes the public mutation contract.
var SetEquippedArmamentsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetEquippedArmaments",
	ID:                         SetEquippedArmamentsEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument of family weapon with capability equipment allowing the assigned hand slot",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "slotAssignments", "expectedRevision"},
	Description:                "Atomically sets armaments in every hand slot and validates slot types and the existence of owned instances.",
})

var equippedArmamentSlots = [6]schema.EquipmentSlot{
	schema.EquipmentSlotLeftHand,
	schema.EquipmentSlotRightHand,
	schema.EquipmentSlotLeftHand,
	schema.EquipmentSlotRightHand,
	schema.EquipmentSlotLeftHand,
	schema.EquipmentSlotRightHand,
}

// EquippedArmamentAssignment identifies the catalog item and exact materialized
// weapon variant committed to one hand slot.
type EquippedArmamentAssignment struct {
	Kind   schema.ResourceKind `json:"kind"`
	Key    string              `json:"key"`
	GameID uint32              `json:"gameID"`
}

// SetEquippedArmamentsResult reports the committed assignment in catalog terms.
//
// The receipt is the one the SaveEngine commit path produced, embedded
// anonymously so the JSON stays flat and carries no nested receipt object.
type SetEquippedArmamentsResult struct {
	saveengine.MutationReceipt
	CharacterID     int                            `json:"characterID"`
	SlotAssignments [6]*EquippedArmamentAssignment `json:"slotAssignments"`
}

// SetEquippedArmaments replaces left 1, right 1, left 2, right 2, left 3 and
// right 3 in that order. The request must carry exactly six entries; nil clears
// the corresponding hand slot.
func SetEquippedArmaments(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	slotAssignments []*string,
	expectedRevision string,
) (SetEquippedArmamentsResult, error) {
	if engine == nil {
		return SetEquippedArmamentsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetEquippedArmamentsResult{}, errors.New("game catalog is not available")
	}
	if len(slotAssignments) != len(equippedArmamentSlots) {
		return SetEquippedArmamentsResult{}, fmt.Errorf(
			"slotAssignments must contain exactly 6 positions; got %d", len(slotAssignments))
	}

	var fixedAssignments [6]*string
	copy(fixedAssignments[:], slotAssignments)
	validator := func(slot int, gameID uint32) error {
		resource, found := gameCatalog.ItemByGameID(gameID)
		if !found || resource.Item == nil {
			return fmt.Errorf("item with game ID 0x%08X is not found in game catalog", gameID)
		}
		return validateEquipmentSlotCompatibility(
			resource, schema.ItemFamilyWeapon, equippedArmamentSlots[slot],
			"hand", fmt.Sprintf("slot %q", equippedArmamentSlots[slot]))
	}

	mutation, err := engine.SetEquippedArmaments(
		saveSessionID, characterID, fixedAssignments, expectedRevision, validator)
	if err != nil {
		return SetEquippedArmamentsResult{}, err
	}

	var resolved [6]*EquippedArmamentAssignment
	for slot, gameID := range mutation.GameIDs {
		if gameID == 0 {
			continue
		}
		resource, found := gameCatalog.ItemByGameID(gameID)
		if !found {
			return SetEquippedArmamentsResult{}, fmt.Errorf(
				"committed game ID 0x%08X could not be found in game catalog", gameID)
		}
		resolved[slot] = &EquippedArmamentAssignment{
			Kind:   resource.Kind,
			Key:    resource.Key,
			GameID: gameID,
		}
	}

	return SetEquippedArmamentsResult{
		MutationReceipt: mutation.MutationReceipt,
		CharacterID:     mutation.CharacterID,
		SlotAssignments: resolved,
	}, nil
}
