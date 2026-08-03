package migration

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (context *generationContext) attachWeaponSaveForgeValues(
	data *schema.WeaponData,
	item seed,
	core regulationWeaponCoreStats,
	sortID uint32,
	sourceRowID uint32,
) error {
	if !item.HasLegacyItem {
		return nil
	}
	stats := item.WeaponStats
	if stats == nil {
		return nil
	}
	if stats.ItemID != item.ID {
		return fmt.Errorf(
			"legacy WeaponStatsV1 item ID 0x%08X does not match item 0x%08X",
			stats.ItemID,
			item.ID,
		)
	}
	if stats.SourceRowID != sourceRowID {
		return fmt.Errorf(
			"legacy WeaponStatsV1 source row %d does not match EquipParamWeapon row %d for item 0x%08X",
			stats.SourceRowID,
			sourceRowID,
			item.ID,
		)
	}
	var description *legacyWeaponStats
	if item.Description != nil {
		description = item.Description.Weapon
	}
	descriptionCandidate := func(value func(*legacyWeaponStats) int32) legacyCandidate[int32] {
		if description == nil {
			return legacyCandidate[int32]{}
		}
		return legacyCandidate[int32]{
			available: true,
			value:     value(description),
			source:    "Descriptions.Weapon",
		}
	}
	descriptionWeight := legacyCandidate[float64]{}
	if description != nil {
		descriptionWeight = legacyCandidate[float64]{
			available: true,
			value:     description.Weight,
			source:    "Descriptions.Weapon.Weight",
		}
	}
	itemWeight := legacyCandidate[float64]{}
	if item.Weight != nil {
		itemWeight = legacyCandidate[float64]{
			available: true,
			value:     *item.Weight,
			source:    "ItemWeights",
		}
	}
	weaponEditType := legacyCandidate[uint16]{}
	weaponEditGem := legacyCandidate[uint8]{}
	if item.WeaponEdit != nil {
		weaponEditType = legacyCandidate[uint16]{
			available: true,
			value:     item.WeaponEdit.WepType,
			source:    "WeaponGemMounts.WepType",
		}
		weaponEditGem = legacyCandidate[uint8]{
			available: true,
			value:     item.WeaponEdit.GemMountType,
			source:    "WeaponGemMounts.GemMountType",
		}
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

	var err error
	if data.WeaponTypeIDSFV, err = saveForgeConsensusValue(
		"weaponTypeID",
		core.weaponTypeID,
		legacyCandidate[uint16]{true, stats.WepType, "WeaponStatsV1.WepType"},
		weaponEditType,
	); err != nil {
		return err
	}
	if data.SortIDSFV, err = saveForgeConsensusValue(
		"sortID",
		sortID,
		sortIDCandidate,
	); err != nil {
		return err
	}
	if data.SortGroupIDSFV, err = saveForgeConsensusValue(
		"sortGroupID",
		core.sortGroupID,
		legacyCandidate[uint8]{true, stats.SortGroupID, "WeaponStatsV1.SortGroupID"},
		sortGroupCandidate,
	); err != nil {
		return err
	}
	if data.ReinforceTypeIDSFV, err = saveForgeConsensusValue(
		"reinforceTypeID",
		core.reinforceTypeID,
		legacyCandidate[int32]{true, stats.ReinforceTypeID, "WeaponStatsV1.ReinforceTypeID"},
	); err != nil {
		return err
	}
	if data.GemMountTypeSFV, err = saveForgeConsensusValue(
		"gemMountType",
		core.gemMountType,
		legacyCandidate[uint8]{true, stats.GemMountType, "WeaponStatsV1.GemMountType"},
		weaponEditGem,
	); err != nil {
		return err
	}
	if data.WeightSFV, err = saveForgeWeightValue(
		core.weight,
		legacyCandidate[float64]{true, stats.Weight, "WeaponStatsV1.Weight"},
		itemWeight,
		descriptionWeight,
	); err != nil {
		return err
	}

	type intField struct {
		name          string
		authoritative int32
		legacy        int32
		description   legacyCandidate[int32]
		target        **schema.Fact[int32]
	}
	fields := []intField{
		{"attackPhysical", core.attackPhysical, stats.AttackPhysical, descriptionCandidate(func(value *legacyWeaponStats) int32 { return int32(value.PhysDamage) }), &data.AttackPhysicalSFV},
		{"attackMagic", core.attackMagic, stats.AttackMagic, descriptionCandidate(func(value *legacyWeaponStats) int32 { return int32(value.MagDamage) }), &data.AttackMagicSFV},
		{"attackFire", core.attackFire, stats.AttackFire, descriptionCandidate(func(value *legacyWeaponStats) int32 { return int32(value.FireDamage) }), &data.AttackFireSFV},
		{"attackLightning", core.attackLightning, stats.AttackLightning, descriptionCandidate(func(value *legacyWeaponStats) int32 { return int32(value.LitDamage) }), &data.AttackLightningSFV},
		{"attackHoly", core.attackHoly, stats.AttackHoly, descriptionCandidate(func(value *legacyWeaponStats) int32 { return int32(value.HolyDamage) }), &data.AttackHolySFV},
		{"attackStamina", core.attackStamina, stats.AttackStamina, legacyCandidate[int32]{}, &data.AttackStaminaSFV},
		{"guardPhysical", core.guardPhysical, stats.GuardPhysical, legacyCandidate[int32]{}, &data.GuardPhysicalSFV},
		{"guardMagic", core.guardMagic, stats.GuardMagic, legacyCandidate[int32]{}, &data.GuardMagicSFV},
		{"guardFire", core.guardFire, stats.GuardFire, legacyCandidate[int32]{}, &data.GuardFireSFV},
		{"guardLightning", core.guardLightning, stats.GuardLightning, legacyCandidate[int32]{}, &data.GuardLightningSFV},
		{"guardHoly", core.guardHoly, stats.GuardHoly, legacyCandidate[int32]{}, &data.GuardHolySFV},
		{"guardBoost", core.guardBoost, stats.GuardBoost, legacyCandidate[int32]{}, &data.GuardBoostSFV},
		{"requiredStrength", int32(core.requiredStrength), stats.StatReqStr, descriptionCandidate(func(value *legacyWeaponStats) int32 { return int32(value.ReqStr) }), &data.RequiredStrengthSFV},
		{"requiredDexterity", int32(core.requiredDexterity), stats.StatReqDex, descriptionCandidate(func(value *legacyWeaponStats) int32 { return int32(value.ReqDex) }), &data.RequiredDexteritySFV},
		{"requiredIntelligence", core.requiredIntelligence, stats.StatReqInt, descriptionCandidate(func(value *legacyWeaponStats) int32 { return int32(value.ReqInt) }), &data.RequiredIntelligenceSFV},
		{"requiredFaith", core.requiredFaith, stats.StatReqFai, descriptionCandidate(func(value *legacyWeaponStats) int32 { return int32(value.ReqFai) }), &data.RequiredFaithSFV},
		{"requiredArcane", core.requiredArcane, stats.StatReqArc, descriptionCandidate(func(value *legacyWeaponStats) int32 { return int32(value.ReqArc) }), &data.RequiredArcaneSFV},
		{"critical", core.critical, stats.Critical, legacyCandidate[int32]{}, &data.CriticalSFV},
		{"defaultAshOfWarID", core.defaultAshOfWarID, stats.DefaultAoWID, legacyCandidate[int32]{}, &data.DefaultAshOfWarIDSFV},
	}
	for _, field := range fields {
		*field.target, err = saveForgeConsensusValue(
			field.name,
			field.authoritative,
			legacyCandidate[int32]{true, field.legacy, "WeaponStatsV1"},
			field.description,
		)
		if err != nil {
			return err
		}
	}
	if data.IsSomberSFV, err = saveForgeConsensusValue(
		"isSomber",
		core.isSomber,
		legacyCandidate[bool]{true, stats.IsSomber, "WeaponStatsV1.IsSomber"},
	); err != nil {
		return err
	}
	maxUpgradeCandidate := legacyCandidate[int32]{
		available: true,
		value:     int32(item.MaxUpgrade),
		source:    "ItemData.MaxUpgrade",
	}
	if data.MaxUpgradeSFV, err = saveForgeConsensusValue(
		"maxUpgrade",
		core.maxUpgrade,
		legacyCandidate[int32]{true, stats.MaxUpgrade, "WeaponStatsV1.MaxUpgrade"},
		maxUpgradeCandidate,
	); err != nil {
		return err
	}
	discardReviewedWeaponSaveForgeValues(data, item)
	return nil
}
