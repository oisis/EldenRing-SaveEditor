package core

// baseEquipLoadTenths contains the base maximum Equip Load for Endurance
// 0–99, expressed in tenths to preserve the game's one-decimal display value.
// Values below the playable minimum use the same 45.0 baseline as END 1–8.
var baseEquipLoadTenths = [...]uint16{
	450, 450, 450, 450, 450, 450, 450, 450, 450, 466,
	482, 498, 514, 529, 545, 561, 577, 593, 609, 625,
	641, 656, 672, 688, 704, 720, 730, 741, 752, 764,
	776, 789, 802, 815, 828, 841, 854, 868, 881, 895,
	909, 923, 937, 951, 965, 979, 994, 1008, 1022, 1037,
	1052, 1066, 1081, 1096, 1110, 1125, 1140, 1155, 1170, 1185,
	1200, 1210, 1221, 1231, 1241, 1251, 1262, 1272, 1282, 1292,
	1303, 1313, 1323, 1333, 1344, 1354, 1364, 1374, 1385, 1395,
	1405, 1415, 1426, 1436, 1446, 1456, 1467, 1477, 1487, 1497,
	1508, 1518, 1528, 1538, 1549, 1559, 1569, 1579, 1590, 1600,
}

// EquipLoadClass is the game's movement category for current equipment weight.
type EquipLoadClass string

const (
	EquipLoadLight      EquipLoadClass = "Light"
	EquipLoadMedium     EquipLoadClass = "Medium"
	EquipLoadHeavy      EquipLoadClass = "Heavy"
	EquipLoadOverloaded EquipLoadClass = "Overloaded"
)

// BaseEquipLoad returns the maximum Equip Load derived only from Endurance.
func BaseEquipLoad(endurance uint32) float64 {
	if endurance >= uint32(len(baseEquipLoadTenths)) {
		endurance = uint32(len(baseEquipLoadTenths) - 1)
	}
	return float64(baseEquipLoadTenths[endurance]) / 10
}

// MaxEquipLoad returns the maximum Equip Load after permanent equipped-item
// effects. Endurance bonuses alter the stat before its load curve is read;
// direct bonuses are additive percentage effects on that curve value.
// Temporary effects, Great Runes, and Physick are intentionally excluded by
// the caller.
func MaxEquipLoad(endurance uint32, enduranceBonus int, equipLoadRate float64) float64 {
	effectiveEndurance := int64(endurance) + int64(enduranceBonus)
	if effectiveEndurance < 0 {
		effectiveEndurance = 0
	}
	return BaseEquipLoad(uint32(effectiveEndurance)) * (1 + equipLoadRate)
}

// ClassifyEquipLoad returns the movement category for the current load ratio.
// The thresholds are Light <30%, Medium <70%, Heavy <100%, and Overloaded >=100%.
func ClassifyEquipLoad(current, maximum float64) EquipLoadClass {
	if maximum <= 0 || current >= maximum {
		return EquipLoadOverloaded
	}
	ratio := current / maximum
	switch {
	case ratio < 0.30:
		return EquipLoadLight
	case ratio < 0.70:
		return EquipLoadMedium
	default:
		return EquipLoadHeavy
	}
}
