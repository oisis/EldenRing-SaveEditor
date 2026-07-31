package migration

import "fmt"

type regulationWeaponScaling struct {
	strength     float64
	dexterity    float64
	intelligence float64
	faith        float64
	arcane       float64
}

type regulationWeaponScalingField struct {
	name   string
	target *float64
	legacy int32
}

func readRegulationWeaponScaling(
	row ParameterRow,
	legacy *weaponStatsSeed,
	verifyLegacy bool,
) (regulationWeaponScaling, error) {
	result := regulationWeaponScaling{}
	fields := []regulationWeaponScalingField{
		{"correctStrength", &result.strength, legacy.ScalingStrRaw},
		{"correctAgility", &result.dexterity, legacy.ScalingDexRaw},
		{"correctMagic", &result.intelligence, legacy.ScalingIntRaw},
		{"correctFaith", &result.faith, legacy.ScalingFaiRaw},
		{"correctLuck", &result.arcane, legacy.ScalingArcRaw},
	}
	for _, field := range fields {
		value, err := regulationFloat64(row, field.name)
		if err != nil {
			return regulationWeaponScaling{}, err
		}
		if verifyLegacy && int32(value) != field.legacy {
			return regulationWeaponScaling{}, fmt.Errorf(
				"row %d %s = %v, legacy truncated value = %d",
				row.RowID,
				field.name,
				value,
				field.legacy,
			)
		}
		*field.target = value
	}
	return result, nil
}
