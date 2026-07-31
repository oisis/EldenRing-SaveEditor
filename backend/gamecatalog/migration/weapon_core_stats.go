package migration

import "fmt"

type regulationWeaponCoreStats struct {
	weaponTypeID, requiredStrength, requiredDexterity   uint16
	sortGroupID, gemMountType                           uint8
	reinforceTypeID, attackPhysical, attackMagic        int32
	attackFire, attackLightning, attackHoly             int32
	attackStamina, guardPhysical, guardMagic            int32
	guardFire, guardLightning, guardHoly, guardBoost    int32
	requiredIntelligence, requiredFaith, requiredArcane int32
	critical, defaultAshOfWarID, maxUpgrade             int32
	weight                                              float64
	isInfusable, isSomber                               bool
}

func (context *generationContext) readRegulationWeaponCoreStats(
	item seed,
	row ParameterRow,
) (regulationWeaponCoreStats, error) {
	var result regulationWeaponCoreStats
	var err error
	if result.weaponTypeID, err = regulationUint16(row, "wepType"); err != nil {
		return result, err
	}
	if result.sortGroupID, err = regulationUint8(row, "sortGroupId"); err != nil {
		return result, err
	}
	if result.reinforceTypeID, err = regulationInt32(row, "reinforceTypeId"); err != nil {
		return result, err
	}
	if result.gemMountType, err = regulationUint8(row, "gemMountType"); err != nil {
		return result, err
	}
	disableGemAffinity, err := regulationUint8(row, "disableGemAttr")
	if err != nil {
		return result, err
	}
	if disableGemAffinity > 1 {
		return result, fmt.Errorf(
			"row %d disableGemAttr = %d is not boolean",
			row.RowID,
			disableGemAffinity,
		)
	}
	if result.weight, err = regulationFloat64(row, "weight"); err != nil {
		return result, err
	}
	if item.Category == "arrows_and_bolts" {
		result.weight = 0
	}

	intFields := []struct {
		name   string
		target *int32
	}{
		{"attackBasePhysics", &result.attackPhysical},
		{"attackBaseMagic", &result.attackMagic},
		{"attackBaseFire", &result.attackFire},
		{"attackBaseThunder", &result.attackLightning},
		{"attackBaseDark", &result.attackHoly},
		{"attackBaseStamina", &result.attackStamina},
		{"staminaGuardDef", &result.guardBoost},
		{"properMagic", &result.requiredIntelligence},
		{"properFaith", &result.requiredFaith},
		{"properLuck", &result.requiredArcane},
		{"swordArtsParamId", &result.defaultAshOfWarID},
	}
	for _, field := range intFields {
		if *field.target, err = regulationInt32(row, field.name); err != nil {
			return result, err
		}
	}
	if result.requiredStrength, err = regulationUint16(row, "properStrength"); err != nil {
		return result, err
	}
	if result.requiredDexterity, err = regulationUint16(row, "properAgility"); err != nil {
		return result, err
	}

	guardFields := []struct {
		name   string
		target *int32
	}{
		{"physGuardCutRate", &result.guardPhysical},
		{"magGuardCutRate", &result.guardMagic},
		{"fireGuardCutRate", &result.guardFire},
		{"thunGuardCutRate", &result.guardLightning},
		{"darkGuardCutRate", &result.guardHoly},
	}
	for _, field := range guardFields {
		value, readErr := regulationFloat64(row, field.name)
		if readErr != nil {
			return result, readErr
		}
		if value != float64(int32(value)) {
			return result, fmt.Errorf(
				"row %d %s = %v is not an integer display value",
				row.RowID,
				field.name,
				value,
			)
		}
		*field.target = int32(value)
	}

	criticalBonus, err := regulationInt32(row, "throwAtkRate")
	if err != nil {
		return result, err
	}
	result.critical = 100 + criticalBonus
	bandSize, err := context.weaponReinforcementBandSize(result.reinforceTypeID)
	if err != nil {
		return result, err
	}
	result.maxUpgrade = int32(bandSize - 1)
	result.isInfusable = result.gemMountType == 2 && disableGemAffinity == 0
	result.isSomber = bandSize == 11
	if item.WeaponEdit != nil &&
		item.WeaponEdit.CanChangeAffinity != result.isInfusable {
		return result, fmt.Errorf(
			"legacy weapon affinity gate differs from EquipParamWeapon gemMountType/disableGemAttr",
		)
	}
	if item.HasLegacyItem {
		if err := verifyLegacyWeaponCoreStats(item.WeaponStats, result); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (context *generationContext) weaponReinforcementBandSize(
	reinforceTypeID int32,
) (int, error) {
	if reinforceTypeID < 0 {
		return 0, fmt.Errorf("negative ReinforceParamWeapon ID %d", reinforceTypeID)
	}
	table, exists := context.regulation.Table(RegulationTableReinforceWeapon)
	if !exists {
		return 0, fmt.Errorf(
			"regulation table %q is not loaded",
			RegulationTableReinforceWeapon,
		)
	}
	base := uint32(reinforceTypeID)
	count := 0
	for _, row := range table.Rows() {
		if (row.RowID/100)*100 == base {
			count++
		}
	}
	if count == 0 {
		return 0, fmt.Errorf("ReinforceParamWeapon band %d is missing", reinforceTypeID)
	}
	return count, nil
}

func verifyLegacyWeaponCoreStats(
	legacy *weaponStatsSeed,
	actual regulationWeaponCoreStats,
) error {
	if legacy == nil {
		return fmt.Errorf("legacy WeaponStatsV1 row is missing")
	}
	if legacy.WepType != actual.weaponTypeID ||
		legacy.SortGroupID != actual.sortGroupID ||
		legacy.ReinforceTypeID != actual.reinforceTypeID ||
		legacy.GemMountType != actual.gemMountType ||
		legacy.Weight != actual.weight ||
		legacy.AttackPhysical != actual.attackPhysical ||
		legacy.AttackMagic != actual.attackMagic ||
		legacy.AttackFire != actual.attackFire ||
		legacy.AttackLightning != actual.attackLightning ||
		legacy.AttackHoly != actual.attackHoly ||
		legacy.AttackStamina != actual.attackStamina ||
		legacy.GuardPhysical != actual.guardPhysical ||
		legacy.GuardMagic != actual.guardMagic ||
		legacy.GuardFire != actual.guardFire ||
		legacy.GuardLightning != actual.guardLightning ||
		legacy.GuardHoly != actual.guardHoly ||
		legacy.GuardBoost != actual.guardBoost ||
		legacy.StatReqStr != int32(actual.requiredStrength) ||
		legacy.StatReqDex != int32(actual.requiredDexterity) ||
		legacy.StatReqInt != actual.requiredIntelligence ||
		legacy.StatReqFai != actual.requiredFaith ||
		legacy.StatReqArc != actual.requiredArcane ||
		legacy.Critical != actual.critical ||
		legacy.DefaultAoWID != actual.defaultAshOfWarID ||
		legacy.IsSomber != actual.isSomber ||
		legacy.MaxUpgrade != actual.maxUpgrade {
		return fmt.Errorf("legacy WeaponStatsV1 core values differ from Regulation")
	}
	return nil
}
