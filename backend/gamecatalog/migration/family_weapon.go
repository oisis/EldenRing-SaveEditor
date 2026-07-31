package migration

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (context *generationContext) buildWeaponData(
	item seed,
	row ParameterRow,
) (*schema.WeaponData, error) {
	stats := item.WeaponStats
	if stats == nil {
		return nil, fmt.Errorf("weapon migration seed is missing")
	}
	iconID, err := regulationUint32(row, "iconId")
	if err != nil {
		return nil, err
	}
	sortID, err := regulationUint32(row, "sortId")
	if err != nil {
		return nil, err
	}
	equipment, err := readWeaponEquipmentFlags(row)
	if err != nil {
		return nil, err
	}
	scaling, err := readRegulationWeaponScaling(row, stats, item.HasLegacyItem)
	if err != nil {
		return nil, err
	}
	core, err := context.readRegulationWeaponCoreStats(item, row)
	if err != nil {
		return nil, err
	}
	data := &schema.WeaponData{
		SourceRowID:            knownRegulationFact(row.RowID, RegulationTableWeapon, "Row ID", row.RowID),
		IconID:                 knownRegulationFact(iconID, RegulationTableWeapon, "iconId", row.RowID),
		WeaponTypeID:           knownRegulationFact(core.weaponTypeID, RegulationTableWeapon, "wepType", row.RowID),
		SortID:                 knownRegulationFact(sortID, RegulationTableWeapon, "sortId", row.RowID),
		SortGroupID:            knownRegulationFact(core.sortGroupID, RegulationTableWeapon, "sortGroupId", row.RowID),
		ReinforceTypeID:        knownRegulationFact(core.reinforceTypeID, RegulationTableWeapon, "reinforceTypeId", row.RowID),
		GemMountType:           knownRegulationFact(core.gemMountType, RegulationTableWeapon, "gemMountType", row.RowID),
		Weight:                 knownRegulationFact(core.weight, RegulationTableWeapon, "weight with zero equip-load normalization for ammunition", row.RowID),
		AttackPhysical:         knownRegulationFact(core.attackPhysical, RegulationTableWeapon, "attackBasePhysics", row.RowID),
		AttackMagic:            knownRegulationFact(core.attackMagic, RegulationTableWeapon, "attackBaseMagic", row.RowID),
		AttackFire:             knownRegulationFact(core.attackFire, RegulationTableWeapon, "attackBaseFire", row.RowID),
		AttackLightning:        knownRegulationFact(core.attackLightning, RegulationTableWeapon, "attackBaseThunder", row.RowID),
		AttackHoly:             knownRegulationFact(core.attackHoly, RegulationTableWeapon, "attackBaseDark", row.RowID),
		AttackStamina:          knownRegulationFact(core.attackStamina, RegulationTableWeapon, "attackBaseStamina", row.RowID),
		GuardPhysical:          knownRegulationFact(core.guardPhysical, RegulationTableWeapon, "physGuardCutRate", row.RowID),
		GuardMagic:             knownRegulationFact(core.guardMagic, RegulationTableWeapon, "magGuardCutRate", row.RowID),
		GuardFire:              knownRegulationFact(core.guardFire, RegulationTableWeapon, "fireGuardCutRate", row.RowID),
		GuardLightning:         knownRegulationFact(core.guardLightning, RegulationTableWeapon, "thunGuardCutRate", row.RowID),
		GuardHoly:              knownRegulationFact(core.guardHoly, RegulationTableWeapon, "darkGuardCutRate", row.RowID),
		GuardBoost:             knownRegulationFact(core.guardBoost, RegulationTableWeapon, "staminaGuardDef", row.RowID),
		RequiredStrength:       knownRegulationFact(int32(core.requiredStrength), RegulationTableWeapon, "properStrength", row.RowID),
		RequiredDexterity:      knownRegulationFact(int32(core.requiredDexterity), RegulationTableWeapon, "properAgility", row.RowID),
		RequiredIntelligence:   knownRegulationFact(core.requiredIntelligence, RegulationTableWeapon, "properMagic", row.RowID),
		RequiredFaith:          knownRegulationFact(core.requiredFaith, RegulationTableWeapon, "properFaith", row.RowID),
		RequiredArcane:         knownRegulationFact(core.requiredArcane, RegulationTableWeapon, "properLuck", row.RowID),
		ScalingStrengthRaw:     knownRegulationFact(scaling.strength, RegulationTableWeapon, "correctStrength", row.RowID),
		ScalingDexterityRaw:    knownRegulationFact(scaling.dexterity, RegulationTableWeapon, "correctAgility", row.RowID),
		ScalingIntelligenceRaw: knownRegulationFact(scaling.intelligence, RegulationTableWeapon, "correctMagic", row.RowID),
		ScalingFaithRaw:        knownRegulationFact(scaling.faith, RegulationTableWeapon, "correctFaith", row.RowID),
		ScalingArcaneRaw:       knownRegulationFact(scaling.arcane, RegulationTableWeapon, "correctLuck", row.RowID),
		Critical:               knownRegulationFact(core.critical, RegulationTableWeapon, "base 100 plus throwAtkRate", row.RowID),
		StatusPoison:           weaponStatusFact(stats.StatusPoison, "StatusPoison", stats.Warnings),
		StatusBleed:            weaponStatusFact(stats.StatusBleed, "StatusBleed", stats.Warnings),
		StatusFrost:            weaponStatusFact(stats.StatusFrost, "StatusFrost", stats.Warnings),
		StatusSleep:            weaponStatusFact(stats.StatusSleep, "StatusSleep", stats.Warnings),
		StatusMadness:          weaponStatusFact(stats.StatusMadness, "StatusMadness", stats.Warnings),
		StatusScarletRot:       weaponStatusFact(stats.StatusScarletRot, "StatusScarletRot", stats.Warnings),
		DefaultAshOfWarID:      knownRegulationFact(core.defaultAshOfWarID, RegulationTableWeapon, "swordArtsParamId", row.RowID),
		SwordArtsName:          context.swordArtsNameFact(core.defaultAshOfWarID),
		SwordArtsNameSFV:       context.swordArtsNameSaveForgeValue(core.defaultAshOfWarID),
		IsInfusable:            knownRegulationDerivedFact(core.isInfusable, RegulationTableWeapon, "derived from gemMountType == 2 and disableGemAttr == 0", row.RowID, "gemMountType,disableGemAttr"),
		IsSomber:               knownRegulationDerivedFact(core.isSomber, RegulationTableReinforceWeapon, "derived from an 11-row reinforcement band", uint32(core.reinforceTypeID), "Row ID"),
		MaxUpgrade:             knownRegulationDerivedFact(core.maxUpgrade, RegulationTableReinforceWeapon, "derived from the reinforcement band row count", uint32(core.reinforceTypeID), "Row ID"),
		Warnings:               knownLegacyFact(cloneStrings(stats.Warnings), "copied from legacy WeaponStatsV1.Warnings"),
		RightHandEquipable:     knownRegulationFact(equipment.right, RegulationTableWeapon, "rightHandEquipable", row.RowID),
		LeftHandEquipable:      knownRegulationFact(equipment.left, RegulationTableWeapon, "leftHandEquipable", row.RowID),
		BothHandEquipable:      knownRegulationFact(equipment.both, RegulationTableWeapon, "bothHandEquipable", row.RowID),
		ArrowSlotEquipable:     knownRegulationFact(equipment.arrow, RegulationTableWeapon, "arrowSlotEquipable", row.RowID),
		BoltSlotEquipable:      knownRegulationFact(equipment.bolt, RegulationTableWeapon, "boltSlotEquipable", row.RowID),
	}
	passiveEffects := stats.PassiveEffects
	if !item.HasLegacyItem {
		passiveEffects, err = context.regulationWeaponPassiveEffects(row)
		if err != nil {
			return nil, err
		}
	}
	data.PassiveEffects, err = context.buildWeaponPassiveEffects(
		row,
		passiveEffects,
	)
	if err != nil {
		return nil, err
	}
	if err := context.attachWeaponSaveForgeValues(
		data,
		item,
		core,
		sortID,
		row.RowID,
	); err != nil {
		return nil, err
	}
	return data, nil
}
