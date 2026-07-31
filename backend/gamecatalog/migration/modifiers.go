package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func buildModifiers(item seed) schema.ItemModifiers {
	if item.EquipLoad == nil {
		return schema.ItemModifiers{}
	}
	return schema.ItemModifiers{
		EquipLoad: &schema.EquipLoadModifier{
			EnduranceBonus: knownLegacyFact(
				item.EquipLoad.EnduranceBonus,
				"copied from legacy EquipLoadModifiers.EnduranceBonus",
			),
			EquipLoadRate: knownLegacyFact(
				item.EquipLoad.EquipLoadRate,
				"copied from legacy EquipLoadModifiers.EquipLoadRate",
			),
		},
	}
}
