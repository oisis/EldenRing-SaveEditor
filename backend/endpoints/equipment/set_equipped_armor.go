/*
Endpoint: SetEquippedArmor
EndpointID: set_equipped_armor
Purpose: Atomically sets armor in every armor slot.
How it works: The runtime handler validates exactly four owned-item assignments in head, chest, arms and legs order, checks every non-empty record through GameCatalog, and delegates one atomic mutation to SaveEngine.
Supported resource types: ItemDocument of family armor with capability equipment allowing the assigned armor slot.
Input variables: saveSessionID, characterID, slotAssignments, expectedRevision.
GameCatalog variables read: item.family, item.gameID and item.capabilities.equipment.
Save variables processed: the four armor fields in all four native representations; SaveEngine validates the complete plan and finishes with full success or rollback.
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

// SetEquippedArmorEndpointID is the stable backend identifier of SetEquippedArmor.
const SetEquippedArmorEndpointID = "set_equipped_armor"

// SetEquippedArmorDefinition describes the public mutation contract.
var SetEquippedArmorDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetEquippedArmor",
	ID:                         SetEquippedArmorEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument of family armor with capability equipment allowing the assigned armor slot",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "slotAssignments", "expectedRevision"},
	Description:                "Atomically sets armor in every armor slot.",
})

var equippedArmorSlots = [4]schema.EquipmentSlot{
	schema.EquipmentSlotHead,
	schema.EquipmentSlotChest,
	schema.EquipmentSlotArms,
	schema.EquipmentSlotLegs,
}

// SetEquippedArmorResult reports the committed assignment in catalog terms.
type SetEquippedArmorResult struct {
	SaveSessionID   string                 `json:"saveSessionID"`
	SaveRevision    string                 `json:"saveRevision"`
	CharacterID     int                    `json:"characterID"`
	SlotAssignments [4]*schema.ResourceRef `json:"slotAssignments"`
}

// SetEquippedArmor replaces head, chest, arms and legs in that order. The
// request must carry exactly four entries; nil clears the corresponding slot.
func SetEquippedArmor(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	slotAssignments []*string,
	expectedRevision string,
) (SetEquippedArmorResult, error) {
	if engine == nil {
		return SetEquippedArmorResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetEquippedArmorResult{}, errors.New("game catalog is not available")
	}
	if len(slotAssignments) != len(equippedArmorSlots) {
		return SetEquippedArmorResult{}, fmt.Errorf(
			"slotAssignments must contain exactly 4 positions; got %d", len(slotAssignments))
	}

	var fixedAssignments [4]*string
	copy(fixedAssignments[:], slotAssignments)
	validator := func(slot int, gameID uint32) error {
		resource, found := gameCatalog.ItemByGameID(gameID)
		if !found || resource.Item == nil {
			return fmt.Errorf("item with game ID 0x%08X is not found in game catalog", gameID)
		}
		item := resource.Item
		if !item.Family.Known || item.Family.Value != schema.ItemFamilyArmor {
			return fmt.Errorf("resource kind %q key %q has item family %q, want %q",
				resource.Kind, resource.Key, item.Family.Value, schema.ItemFamilyArmor)
		}
		equipment := item.Capabilities.Equipment
		if !equipment.Known || !equipment.Enabled || equipment.Rules == nil {
			return fmt.Errorf("resource kind %q key %q has no confirmed armor equipment capability",
				resource.Kind, resource.Key)
		}
		for _, allowed := range equipment.Rules.AllowedSlots {
			if allowed == equippedArmorSlots[slot] {
				return nil
			}
		}
		return fmt.Errorf("resource kind %q key %q cannot be equipped in slot %q",
			resource.Kind, resource.Key, equippedArmorSlots[slot])
	}

	mutation, err := engine.SetEquippedArmor(
		saveSessionID, characterID, fixedAssignments, expectedRevision, validator)
	if err != nil {
		return SetEquippedArmorResult{}, err
	}

	var resolved [4]*schema.ResourceRef
	for slot, gameID := range mutation.GameIDs {
		if gameID == 0 {
			continue
		}
		resource, found := gameCatalog.ItemByGameID(gameID)
		if !found {
			return SetEquippedArmorResult{}, fmt.Errorf(
				"committed game ID 0x%08X could not be found in game catalog", gameID)
		}
		ref := resource.Ref()
		resolved[slot] = &ref
	}

	return SetEquippedArmorResult{
		SaveSessionID:   mutation.SaveSessionID,
		SaveRevision:    mutation.SaveRevision,
		CharacterID:     mutation.CharacterID,
		SlotAssignments: resolved,
	}, nil
}
