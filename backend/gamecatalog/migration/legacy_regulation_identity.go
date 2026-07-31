package migration

import "fmt"

type primaryRegulationIdentity struct {
	Family RegulationFamily
	RowID  uint32
}

func primaryRegulationForLegacyItem(item seed) (primaryRegulationIdentity, error) {
	var family RegulationFamily
	switch item.Category {
	case "melee_armaments", "ranged_and_catalysts", "shields", "arrows_and_bolts":
		family = RegulationFamilyWeapon
	case "head", "chest", "arms", "legs":
		family = RegulationFamilyProtector
	case "talismans":
		family = RegulationFamilyAccessory
	case "ashes_of_war":
		family = RegulationFamilyAshOfWar
	case "sorceries", "incantations":
		family = RegulationFamilySpell
	case "gestures", "ashes", "crafting_materials", "bolstering_materials", "key_items", "tools", "info":
		family = RegulationFamilyGoods
	default:
		return primaryRegulationIdentity{}, fmt.Errorf(
			"item 0x%08X has unsupported legacy category %q",
			item.ID,
			item.Category,
		)
	}

	rowID := item.ID
	if family != RegulationFamilyWeapon {
		rowID &= 0x0FFFFFFF
	}
	return primaryRegulationIdentity{Family: family, RowID: rowID}, nil
}
