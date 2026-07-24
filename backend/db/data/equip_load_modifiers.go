package data

// EquipLoadModifier describes a permanent, equipped-item effect on maximum
// Equip Load. Values are sourced from the item's linked SpEffectParam row in
// regulation.bin. Temporary effects, Great Runes, and Physick are deliberately
// outside this table.
type EquipLoadModifier struct {
	EnduranceBonus int
	EquipLoadRate  float64
}

// EquipLoadModifiers is keyed by canonical item ID. EquipLoadRate is the
// additive percentage bonus (0.15 means +15%), matching the game's stacking of
// the Arsenal and Erdtree's Favor effect families.
var EquipLoadModifiers = map[uint32]EquipLoadModifier{
	// Talismans.
	0x20000406: {EquipLoadRate: 0.15},  // Arsenal Charm
	0x20000407: {EquipLoadRate: 0.17},  // Arsenal Charm +1
	0x20000408: {EquipLoadRate: 0.19},  // Great-Jar's Arsenal
	0x20000410: {EquipLoadRate: 0.05},  // Erdtree's Favor
	0x20000411: {EquipLoadRate: 0.065}, // Erdtree's Favor +1
	0x20000412: {EquipLoadRate: 0.08},  // Erdtree's Favor +2
	0x2000041A: {EnduranceBonus: 3},    // Radagon's Scarseal
	0x2000041B: {EnduranceBonus: 5},    // Radagon's Soreseal

	// Head armor.
	0x100F1B30: {EnduranceBonus: 2},    // Hierodas Glintstone Crown
	0x10108A60: {EnduranceBonus: 2},    // Imp Head (Wolf)
	0x104F0A60: {EquipLoadRate: 0.045}, // Fire Knight Helm
}
