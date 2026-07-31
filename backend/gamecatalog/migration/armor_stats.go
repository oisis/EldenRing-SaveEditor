package migration

import "math"

type regulationArmorStats struct {
	weight     float64
	physical   float64
	strike     float64
	slash      float64
	pierce     float64
	magic      float64
	fire       float64
	lightning  float64
	holy       float64
	immunity   uint32
	robustness uint32
	focus      uint32
	vitality   uint32
	poise      float64
}

func readRegulationArmorStats(
	row ParameterRow,
) (regulationArmorStats, error) {
	var result regulationArmorStats
	floatFields := []struct {
		name      string
		target    *float64
		transform func(float64) float64
	}{
		{"weight", &result.weight, roundArmorDisplayValue},
		{"neutralDamageCutRate", &result.physical, armorNegation},
		{"blowDamageCutRate", &result.strike, armorNegation},
		{"slashDamageCutRate", &result.slash, armorNegation},
		{"thrustDamageCutRate", &result.pierce, armorNegation},
		{"magicDamageCutRate", &result.magic, armorNegation},
		{"fireDamageCutRate", &result.fire, armorNegation},
		{"thunderDamageCutRate", &result.lightning, armorNegation},
		{"darkDamageCutRate", &result.holy, armorNegation},
		{"toughnessCorrectRate", &result.poise, armorPoise},
	}
	for _, field := range floatFields {
		value, err := regulationFloat64(row, field.name)
		if err != nil {
			return regulationArmorStats{}, err
		}
		*field.target = field.transform(value)
	}

	uintFields := []struct {
		name   string
		target *uint32
	}{
		{"resistPoison", &result.immunity},
		{"resistBlood", &result.robustness},
		{"resistSleep", &result.focus},
		{"resistCurse", &result.vitality},
	}
	for _, field := range uintFields {
		value, err := regulationUint32(row, field.name)
		if err != nil {
			return regulationArmorStats{}, err
		}
		*field.target = value
	}
	return result, nil
}

func armorNegation(value float64) float64 {
	return roundArmorDisplayValue((1 - value) * 100)
}

func armorPoise(value float64) float64 {
	return math.Round(value * 1000)
}

func roundArmorDisplayValue(value float64) float64 {
	return math.Round(value*10) / 10
}
