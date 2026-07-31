package migration

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func attachSimpleFamilySaveForgeValues(
	item seed,
	sortID uint32,
	sortGroupID uint8,
	weight float64,
	sortIDSFV **schema.Fact[uint32],
	sortGroupIDSFV **schema.Fact[uint8],
	weightSFV **schema.Fact[float64],
) error {
	if !item.HasLegacyItem {
		return nil
	}
	if item.SortKey != nil {
		*sortIDSFV = saveForgeValue(
			true,
			item.SortKey.SortID,
			sortID,
			"preserved conflicting SaveForge value from ItemSortKeys.SortId",
		)
		*sortGroupIDSFV = saveForgeValue(
			true,
			item.SortKey.SortGroupID,
			sortGroupID,
			"preserved conflicting SaveForge value from ItemSortKeys.SortGroupId",
		)
	}
	itemWeight := legacyCandidate[float64]{}
	descriptionWeight := legacyCandidate[float64]{}
	if item.Weight != nil {
		itemWeight = legacyCandidate[float64]{
			available: true,
			value:     *item.Weight,
			source:    "ItemWeights",
		}
	}
	if item.Description != nil && item.Description.Weight != 0 {
		descriptionWeight = legacyCandidate[float64]{
			available: true,
			value:     item.Description.Weight,
			source:    "Descriptions.Weight",
		}
	}
	var err error
	*weightSFV, err = saveForgeWeightValue(
		weight,
		itemWeight,
		descriptionWeight,
	)
	return err
}

func attachSpellSaveForgeValues(
	data *schema.SpellData,
	item seed,
	sortID uint32,
	fpCost uint32,
	memorySlots uint8,
	requiredIntelligence uint32,
	requiredFaith uint32,
	requiredArcane uint32,
) error {
	if !item.HasLegacyItem {
		return nil
	}
	if item.SortKey != nil {
		data.SortIDSFV = saveForgeValue(
			true,
			item.SortKey.SortID,
			sortID,
			"preserved conflicting SaveForge value from ItemSortKeys.SortId",
		)
	}
	var description *legacySpellStats
	if item.Description != nil {
		description = item.Description.Spell
	}
	if description != nil && description.Slots > 0xFF {
		return fmt.Errorf(
			"legacy Descriptions.Spell.Slots %d exceeds uint8",
			description.Slots,
		)
	}
	descriptionUint := func(value func(*legacySpellStats) uint32) legacyCandidate[uint32] {
		if description == nil {
			return legacyCandidate[uint32]{}
		}
		return legacyCandidate[uint32]{
			available: true,
			value:     value(description),
			source:    "Descriptions.Spell",
		}
	}
	var err error
	if data.FPCostSFV, err = saveForgeConsensusValue(
		"fpCost",
		fpCost,
		descriptionUint(func(value *legacySpellStats) uint32 { return value.FPCost }),
	); err != nil {
		return err
	}
	memoryCandidates := []legacyCandidate[uint8]{}
	if item.SpellMemory != nil {
		memoryCandidates = append(memoryCandidates, legacyCandidate[uint8]{
			available: true,
			value:     *item.SpellMemory,
			source:    "SpellMemorySlots",
		})
	}
	if description != nil {
		memoryCandidates = append(memoryCandidates, legacyCandidate[uint8]{
			available: true,
			value:     uint8(description.Slots),
			source:    "Descriptions.Spell.Slots",
		})
	}
	if data.MemorySlotsSFV, err = saveForgeConsensusValue(
		"memorySlots",
		memorySlots,
		memoryCandidates...,
	); err != nil {
		return err
	}
	if data.RequiredIntelligenceSFV, err = saveForgeConsensusValue(
		"requiredIntelligence",
		requiredIntelligence,
		descriptionUint(func(value *legacySpellStats) uint32 { return value.ReqInt }),
	); err != nil {
		return err
	}
	if data.RequiredFaithSFV, err = saveForgeConsensusValue(
		"requiredFaith",
		requiredFaith,
		descriptionUint(func(value *legacySpellStats) uint32 { return value.ReqFai }),
	); err != nil {
		return err
	}
	if data.RequiredArcaneSFV, err = saveForgeConsensusValue(
		"requiredArcane",
		requiredArcane,
		descriptionUint(func(value *legacySpellStats) uint32 { return value.ReqArc }),
	); err != nil {
		return err
	}
	return nil
}
