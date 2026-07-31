package migration

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func itemFamily(item seed) (schema.ItemFamily, RegulationTableName, error) {
	switch item.Category {
	case "melee_armaments", "ranged_and_catalysts", "shields", "arrows_and_bolts":
		return schema.ItemFamilyWeapon, RegulationTableWeapon, nil
	case "head", "chest", "arms", "legs":
		return schema.ItemFamilyArmor, RegulationTableProtector, nil
	case "talismans":
		return schema.ItemFamilyTalisman, RegulationTableAccessory, nil
	case "ashes_of_war":
		return schema.ItemFamilyAshOfWar, RegulationTableGem, nil
	case "sorceries", "incantations":
		return schema.ItemFamilySpell, RegulationTableMagic, nil
	case "ashes":
		return schema.ItemFamilySpiritAsh, RegulationTableGoods, nil
	case "gestures":
		return schema.ItemFamilyGesture, RegulationTableGoods, nil
	case "crafting_materials", "bolstering_materials", "key_items", "tools", "info":
		return schema.ItemFamilyGoods, RegulationTableGoods, nil
	default:
		return "", "", fmt.Errorf("unsupported legacy category %q", item.Category)
	}
}

func (context *generationContext) attachFamilyData(
	document *schema.ItemDocument,
	item seed,
	family schema.ItemFamily,
) error {
	identity, err := primaryRegulationForLegacyItem(item)
	if err != nil {
		return err
	}
	lookup, exists, err := context.regulation.LookupFamilyRow(
		identity.Family,
		RegulationTableRolePrimary,
		identity.RowID,
	)
	if err != nil {
		return err
	}

	switch family {
	case schema.ItemFamilyWeapon:
		if !exists {
			return fmt.Errorf("weapon primary row %d is missing", identity.RowID)
		}
		document.Weapon, err = context.buildWeaponData(item, lookup.Row)
	case schema.ItemFamilyArmor:
		if !exists {
			return fmt.Errorf("armor primary row %d is missing", identity.RowID)
		}
		document.Armor, err = buildArmorData(item, lookup.Row)
	case schema.ItemFamilyTalisman:
		if !exists {
			return fmt.Errorf("talisman primary row %d is missing", identity.RowID)
		}
		document.Talisman, err = buildTalismanData(item, lookup.Row)
	case schema.ItemFamilyAshOfWar:
		if !exists {
			return fmt.Errorf("Ash of War primary row %d is missing", identity.RowID)
		}
		document.AshOfWar, err = context.buildAshOfWarData(item, lookup.Row)
	case schema.ItemFamilySpell:
		if !exists {
			return fmt.Errorf("spell primary row %d is missing", identity.RowID)
		}
		document.Spell, err = buildSpellData(item, lookup.Row)
	case schema.ItemFamilySpiritAsh:
		if !exists {
			return fmt.Errorf("spirit ash primary row %d is missing", identity.RowID)
		}
		document.SpiritAsh, err = buildSpiritAshData(item, lookup.Row)
	case schema.ItemFamilyGoods:
		if !exists {
			return fmt.Errorf("goods primary row %d is missing", identity.RowID)
		}
		document.Goods, err = buildGoodsData(item, lookup.Row)
	case schema.ItemFamilyGesture:
		document.Gesture, err = context.buildGestureData(item, lookup.Row, exists)
	default:
		err = fmt.Errorf("unsupported item family %q", family)
	}
	return err
}
