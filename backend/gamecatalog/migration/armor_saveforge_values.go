package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func attachArmorSaveForgeValues(
	data *schema.ArmorData,
	item seed,
	stats regulationArmorStats,
	sortID uint32,
	sortGroupID uint8,
) error {
	if !item.HasLegacyItem {
		return nil
	}
	sortIDCandidate := legacyCandidate[uint32]{}
	sortGroupCandidate := legacyCandidate[uint8]{}
	if item.SortKey != nil {
		sortIDCandidate = legacyCandidate[uint32]{
			available: true,
			value:     item.SortKey.SortID,
			source:    "ItemSortKeys.SortId",
		}
		sortGroupCandidate = legacyCandidate[uint8]{
			available: true,
			value:     item.SortKey.SortGroupID,
			source:    "ItemSortKeys.SortGroupId",
		}
	}
	itemWeight := legacyCandidate[float64]{}
	armorWeight := legacyCandidate[float64]{}
	if item.Weight != nil {
		itemWeight = legacyCandidate[float64]{
			available: true,
			value:     *item.Weight,
			source:    "ItemWeights",
		}
	}
	if item.Description != nil {
		if item.Description.Armor != nil {
			armorWeight = legacyCandidate[float64]{
				available: true,
				value:     item.Description.Armor.Weight,
				source:    "Descriptions.Armor.Weight",
			}
		}
	}

	var err error
	if data.SortIDSFV, err = saveForgeConsensusValue(
		"sortID",
		sortID,
		sortIDCandidate,
	); err != nil {
		return err
	}
	if data.SortGroupIDSFV, err = saveForgeConsensusValue(
		"sortGroupID",
		sortGroupID,
		sortGroupCandidate,
	); err != nil {
		return err
	}
	if data.WeightSFV, err = saveForgeWeightValue(
		stats.weight,
		itemWeight,
		armorWeight,
	); err != nil {
		return err
	}
	if item.Description == nil || item.Description.Armor == nil {
		return nil
	}
	legacy := item.Description.Armor
	floatFields := []struct {
		name          string
		authoritative float64
		legacy        float64
		target        **schema.Fact[float64]
	}{
		{"physical", stats.physical, legacy.Physical, &data.PhysicalSFV},
		{"strike", stats.strike, legacy.Strike, &data.StrikeSFV},
		{"slash", stats.slash, legacy.Slash, &data.SlashSFV},
		{"pierce", stats.pierce, legacy.Pierce, &data.PierceSFV},
		{"magic", stats.magic, legacy.Magic, &data.MagicSFV},
		{"fire", stats.fire, legacy.Fire, &data.FireSFV},
		{"lightning", stats.lightning, legacy.Lightning, &data.LightningSFV},
		{"holy", stats.holy, legacy.Holy, &data.HolySFV},
	}
	for _, field := range floatFields {
		*field.target = saveForgeValue(
			true,
			field.legacy,
			field.authoritative,
			"preserved conflicting SaveForge value from Descriptions.Armor",
		)
	}
	uintFields := []struct {
		name          string
		authoritative uint32
		legacy        uint32
		target        **schema.Fact[uint32]
	}{
		{"immunity", stats.immunity, legacy.Immunity, &data.ImmunitySFV},
		{"robustness", stats.robustness, legacy.Robustness, &data.RobustnessSFV},
		{"focus", stats.focus, legacy.Focus, &data.FocusSFV},
		{"vitality", stats.vitality, legacy.Vitality, &data.VitalitySFV},
	}
	for _, field := range uintFields {
		*field.target = saveForgeValue(
			true,
			field.legacy,
			field.authoritative,
			"preserved conflicting SaveForge value from Descriptions.Armor",
		)
	}
	discardReviewedArmorSaveForgeValues(data, item)
	return nil
}
