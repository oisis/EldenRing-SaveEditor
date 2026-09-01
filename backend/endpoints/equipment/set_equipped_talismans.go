/*
Endpoint: SetEquippedTalismans
EndpointID: set_equipped_talismans
Purpose: Atomically sets talismans while respecting the number of unlocked slots.
How it works: The runtime handler validates each selected owned inventory record through GameCatalog and delegates one complete compact assignment to SaveEngine under expectedRevision control.
Supported resource types: ItemDocument of family talisman with capability equipment allowing slot talisman.
Input variables: saveSessionID, characterID, orderedOwnedItemIDs, expectedRevision.
GameCatalog variables read: item.family, item.gameID and item.capabilities.equipment.
Save variables processed: the four player-visible talisman fields in all four native representations and the unlocked talisman-slot count; SaveEngine validates the complete plan and finishes with full success or rollback.
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

// SetEquippedTalismansEndpointID is the stable backend identifier of SetEquippedTalismans.
const SetEquippedTalismansEndpointID = "set_equipped_talismans"

// SetEquippedTalismansDefinition describes the public mutation contract.
var SetEquippedTalismansDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetEquippedTalismans",
	ID:                         SetEquippedTalismansEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument of family talisman with capability equipment allowing slot talisman",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "orderedOwnedItemIDs", "expectedRevision"},
	Description:                "Atomically sets talismans while respecting the number of unlocked slots.",
})

// SetEquippedTalismansResult reports the committed loadout in public catalog terms.
//
// The receipt is the one the SaveEngine commit path produced, embedded
// anonymously so the JSON stays flat and carries no nested receipt object.
type SetEquippedTalismansResult struct {
	saveengine.MutationReceipt
	CharacterID      int                  `json:"characterID"`
	OrderedResources []schema.ResourceRef `json:"orderedResources"`
	UnlockedSlots    int                  `json:"unlockedSlots"`
}

// SetEquippedTalismans replaces the compact talisman loadout of one active character.
func SetEquippedTalismans(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	orderedOwnedItemIDs []string,
	expectedRevision string,
) (SetEquippedTalismansResult, error) {
	if engine == nil {
		return SetEquippedTalismansResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetEquippedTalismansResult{}, errors.New("game catalog is not available")
	}
	if len(orderedOwnedItemIDs) > 4 {
		return SetEquippedTalismansResult{}, fmt.Errorf(
			"orderedOwnedItemIDs contains %d talismans, want at most 4", len(orderedOwnedItemIDs))
	}

	validator := func(gameID uint32) error {
		resource, found := gameCatalog.ItemByGameID(gameID)
		if !found || resource.Item == nil {
			return fmt.Errorf("item with game ID 0x%08X is not found in game catalog", gameID)
		}
		item := resource.Item
		if !item.Family.Known || item.Family.Value != schema.ItemFamilyTalisman {
			return fmt.Errorf("resource kind %q key %q has item family %q, want %q",
				resource.Kind, resource.Key, item.Family.Value, schema.ItemFamilyTalisman)
		}
		equipment := item.Capabilities.Equipment
		if !equipment.Known || !equipment.Enabled || equipment.Rules == nil {
			return fmt.Errorf("resource kind %q key %q has no confirmed talisman equipment capability",
				resource.Kind, resource.Key)
		}
		for _, slot := range equipment.Rules.AllowedSlots {
			if slot == schema.EquipmentSlotTalisman {
				return nil
			}
		}
		return fmt.Errorf("resource kind %q key %q cannot be equipped in a talisman slot",
			resource.Kind, resource.Key)
	}

	mutation, err := engine.SetEquippedTalismans(
		saveSessionID, characterID, orderedOwnedItemIDs, expectedRevision, validator)
	if err != nil {
		return SetEquippedTalismansResult{}, err
	}

	resources := make([]schema.ResourceRef, len(mutation.GameIDs))
	for index, gameID := range mutation.GameIDs {
		resource, found := gameCatalog.ItemByGameID(gameID)
		if !found {
			return SetEquippedTalismansResult{}, fmt.Errorf(
				"committed game ID 0x%08X could not be found in game catalog", gameID)
		}
		resources[index] = resource.Ref()
	}

	return SetEquippedTalismansResult{
		MutationReceipt:  mutation.MutationReceipt,
		CharacterID:      mutation.CharacterID,
		OrderedResources: resources,
		UnlockedSlots:    mutation.UnlockedSlots,
	}, nil
}
