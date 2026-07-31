package migration

import (
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

type legacyItemSource struct {
	category string
	items    map[uint32]data.ItemData
}

func collectLegacySnapshot() legacySnapshot {
	items := collectLegacyItems()
	return legacySnapshot{
		Items:            items,
		Aliases:          collectLegacyAliases(),
		GestureSlots:     collectLegacyGestureSlots(),
		SwordArtsNames:   collectLegacySwordArtsNames(),
		TechnicalRecords: collectLegacyTechnicalRecords(items),
	}
}

func collectLegacyItems() []seed {
	sources := []legacyItemSource{
		{"melee_armaments", data.Weapons},
		{"ranged_and_catalysts", data.RangedAndCatalysts},
		{"shields", data.Shields},
		{"arrows_and_bolts", data.ArrowsAndBolts},
		{"head", data.Helms},
		{"chest", data.Chest},
		{"arms", data.Arms},
		{"legs", data.Legs},
		{"talismans", data.Talismans},
		{"ashes_of_war", data.Aows},
		{"gestures", data.Gestures},
		{"ashes", data.StandardAshes},
		{"sorceries", data.Sorceries},
		{"incantations", data.Incantations},
		{"crafting_materials", data.CraftingMaterials},
		{"bolstering_materials", data.BolsteringMaterials},
		{"key_items", data.KeyItems},
		{"tools", data.Tools},
		{"info", data.Information},
	}

	items := make([]seed, 0, 3836)
	for _, source := range sources {
		for id, item := range source.items {
			items = append(items, newLegacySeed(id, source.category, item))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ID != items[j].ID {
			return items[i].ID < items[j].ID
		}
		return items[i].Category < items[j].Category
	})
	return items
}

func newLegacySeed(id uint32, category string, item data.ItemData) seed {
	result := seed{
		ID:                    id,
		HasLegacyItem:         true,
		Category:              category,
		Name:                  item.Name,
		Subcategory:           item.SubCategory,
		MaxInventory:          item.MaxInventory,
		MaxStorage:            item.MaxStorage,
		GameMaxInventory:      item.GameMaxInventory,
		GameMaxStorage:        item.GameMaxStorage,
		GameMaxInventoryKnown: item.GameMaxInventoryKnown,
		GameMaxStorageKnown:   item.GameMaxStorageKnown,
		MaxUpgrade:            item.MaxUpgrade,
		IconPath:              item.IconPath,
		Flags:                 cloneStrings(item.Flags),
		Unlocks:               collectLegacyUnlocks(id),
		Links:                 collectLegacyLinks(id),
	}
	enrichLegacySeed(&result)
	return result
}

func cloneStrings(values []string) []string {
	return append([]string(nil), values...)
}
