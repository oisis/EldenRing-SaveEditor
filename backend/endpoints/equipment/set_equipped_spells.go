/*
Endpoint: SetEquippedSpells
EndpointID: set_equipped_spells
Purpose: Atomically sets spell order and validates the total Memory Slots cost.
How it works: The runtime handler resolves each non-nil catalog resource reference, validates every spell item document, checks for equipment capability and memory slots, enforces uniqueness and maximum capacity, and delegates one atomic mutation to SaveEngine under expectedRevision control.
Supported resource types: ItemDocument: Spell z capability equipment.
Input variables: saveSessionID, characterID, orderedResources, expectedRevision.
GameCatalog variables read: item.family, item.gameID, item.spell.memorySlots and item.capabilities.equipment.
Save variables processed: the first 12 spell memory positions and active spell index in EquippedSpells; SaveEngine validates the complete plan and finishes with full success or rollback.
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

// SetEquippedSpellsEndpointID is the stable backend identifier of SetEquippedSpells.
const SetEquippedSpellsEndpointID = "set_equipped_spells"

// SetEquippedSpellsDefinition describes the public mutation contract.
var SetEquippedSpellsDefinition = contract.MustDefine(contract.Definition{
	Name:                   "SetEquippedSpells",
	ID:                     SetEquippedSpellsEndpointID,
	Kind:                   contract.Mutation,
	SupportedResourceTypes: "ItemDocument: Spell z capability equipment",
	SupportedResourceVariables: []string{
		"saveSessionID", "characterID", "orderedResources", "expectedRevision",
	},
	Description: "Atomically sets spell order and validates the total Memory Slots cost.",
})

// SetEquippedSpellsResult reports the committed spell loadout in public catalog terms.
type SetEquippedSpellsResult struct {
	SaveSessionID        string                `json:"saveSessionID"`
	SaveRevision         string                `json:"saveRevision"`
	CharacterID          int                   `json:"characterID"`
	OrderedResources     []*schema.ResourceRef `json:"orderedResources"`
	UsedMemorySlots      int                   `json:"usedMemorySlots"`
	AvailableMemorySlots int                   `json:"availableMemorySlots"`
}

// SetEquippedSpells replaces the compact spell memory positions of one active character.
func SetEquippedSpells(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	orderedResources []*schema.ResourceRef,
	expectedRevision string,
) (SetEquippedSpellsResult, error) {
	if engine == nil {
		return SetEquippedSpellsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return SetEquippedSpellsResult{}, errors.New("game catalog is not available")
	}
	if len(orderedResources) > 12 {
		return SetEquippedSpellsResult{}, fmt.Errorf("cannot equip more than 12 spells; got %d", len(orderedResources))
	}

	rawSpellIDs := make([]uint32, len(orderedResources))
	resolvedRefs := make([]*schema.ResourceRef, len(orderedResources))
	seenRawIDs := make(map[uint32]struct{}, len(orderedResources))
	usedMemorySlots := 0

	for index, declared := range orderedResources {
		if declared == nil {
			return SetEquippedSpellsResult{}, fmt.Errorf("orderedResources[%d] cannot be nil", index)
		}
		if declared.Kind == "" || declared.Key == "" {
			return SetEquippedSpellsResult{}, fmt.Errorf("orderedResources[%d] has empty kind or key", index)
		}

		resource, err := gameCatalog.ResourceByKindAndKey(declared.Kind, declared.Key)
		if err != nil {
			return SetEquippedSpellsResult{}, fmt.Errorf("orderedResources[%d]: %w", index, err)
		}

		rawID, memCost, err := gamecatalog.ValidateSpellResource(resource)
		if err != nil {
			return SetEquippedSpellsResult{}, fmt.Errorf("orderedResources[%d]: %w", index, err)
		}

		if _, duplicate := seenRawIDs[rawID]; duplicate {
			return SetEquippedSpellsResult{}, fmt.Errorf(
				"orderedResources[%d]: spell 0x%08X is duplicated", index, rawID|gamecatalog.EquippedSpellGameIDPrefix)
		}
		seenRawIDs[rawID] = struct{}{}
		rawSpellIDs[index] = rawID
		usedMemorySlots += memCost

		ref := resource.Ref()
		resolvedRefs[index] = &ref
	}

	if usedMemorySlots > 12 {
		return SetEquippedSpellsResult{}, fmt.Errorf(
			"used memory slots %d exceeds maximum capacity 12", usedMemorySlots)
	}

	mutation, err := engine.SetEquippedSpells(
		saveSessionID, characterID, rawSpellIDs, usedMemorySlots, expectedRevision)
	if err != nil {
		return SetEquippedSpellsResult{}, err
	}

	return SetEquippedSpellsResult{
		SaveSessionID:        mutation.SaveSessionID,
		SaveRevision:         mutation.SaveRevision,
		CharacterID:          mutation.CharacterID,
		OrderedResources:     resolvedRefs,
		UsedMemorySlots:      mutation.UsedMemorySlots,
		AvailableMemorySlots: mutation.AvailableMemorySlots,
	}, nil
}
