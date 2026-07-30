package dbviewer

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func countUnknownFacts(resource schema.Resource) int {
	item := resource.Item
	count := countUnknown(
		resource.Label.Known,
		item.GameID.Known,
		item.Family.Known,
		item.Subcategory.Known,
		item.Presentation.CanonicalName.Known,
		item.Presentation.Description.Known,
		item.Presentation.IconPath.Known,
		item.Storage.RecordMode.Known,
		item.Storage.MaxInventory.Known,
		item.Storage.MaxStorage.Known,
		item.Safety.CutContent.Known,
		item.Safety.BanRisk.Known,
		item.Capabilities.Upgrade.Known,
		item.Capabilities.Infusion.Known,
		item.Capabilities.AshOfWarMount.Known,
		item.Capabilities.Stack.Known,
		item.Capabilities.Equipment.Known,
	)
	if item.Weapon != nil {
		count += countUnknown(
			item.Weapon.SourceRowID.Known,
			item.Weapon.WeaponTypeID.Known,
			item.Weapon.Weight.Known,
			item.Weapon.AttackPhysical.Known,
			item.Weapon.RequiredStrength.Known,
			item.Weapon.RequiredDexterity.Known,
			item.Weapon.Critical.Known,
		)
	}
	if item.AshOfWar != nil {
		count += countUnknown(
			item.AshOfWar.SourceRowID.Known,
			item.AshOfWar.CompatibilityMask.Known,
		)
	}
	return count
}

func countUnknown(values ...bool) int {
	count := 0
	for _, known := range values {
		if !known {
			count++
		}
	}
	return count
}
