package db

import (
	"fmt"
	"github.com/oisis/EldenRing-SaveEditor/backend/db/data"
	"sort"
	"strings"
)

// ItemEntry represents a single item from the game database.
type ItemEntry struct {
	ID           uint32 `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	MaxInventory uint32 `json:"maxInventory"`
	MaxStorage   uint32 `json:"maxStorage"`
	MaxUpgrade   uint32 `json:"maxUpgrade"`
	IconPath     string `json:"iconPath"`
}

// GraceEntry represents a Site of Grace.
type GraceEntry struct {
	ID     uint32 `json:"id"`
	Name   string `json:"name"`
	Region string `region"`
}

// GetItemData returns the full metadata of an item by its ID and category.
func GetItemData(id uint32, category string) data.ItemData {
	// Search in all relevant maps
	allMaps := []map[uint32]data.ItemData{
		data.Weapons, data.Bows, data.Shields, data.Staffs, data.Seals, data.ArrowsAndBolts,
		data.Helms, data.Chest, data.Gauntlets, data.Leggings,
		data.Talismans, data.Aows, data.Gestures,
		data.StandardAshes,
		data.Sorceries, data.Incantations, data.CraftingMaterials,
		data.BolsteringMaterials,
		data.SacredFlasks, data.ThrowingPots, data.PerfumeArts, data.Throwables,
		data.Grease, data.MiscTools, data.QuestTools, data.GoldenRunes,
		data.Remembrances, data.Multiplayer, data.Consumables, data.Keyitems,
	}

	for _, m := range allMaps {
		if item, ok := m[id]; ok {
			return item
		}
	}

	return data.ItemData{Name: GetItemName(id, category)}
}

// GetItemName returns the name of an item by its ID and category.
func GetItemName(id uint32, category string) string {
	// Special handling for weapons with levels
	for baseID, item := range data.Weapons {
		if (id & 0xFFFFFF00) == (baseID & 0xFFFFFF00) {
			level := id - baseID
			if level > 0 {
				return fmt.Sprintf("%s +%d", item.Name, level)
			}
			return item.Name
		}
	}
	// Check other weapon-like categories for levels
	weaponMaps := []map[uint32]data.ItemData{data.Bows, data.Shields, data.Staffs, data.Seals}
	for _, m := range weaponMaps {
		for baseID, item := range m {
			if (id & 0xFFFFFF00) == (baseID & 0xFFFFFF00) {
				level := id - baseID
				if level > 0 {
					return fmt.Sprintf("%s +%d", item.Name, level)
				}
				return item.Name
			}
		}
	}

	itemData := GetItemData(id, category)
	if itemData.Name != "" {
		return itemData.Name
	}

	return fmt.Sprintf("Unknown Item (0x%X)", id)
}

// GetItemCategoryFromHandle returns the category string based on the GaItemHandle prefix.
func GetItemCategoryFromHandle(handle uint32) string {
	switch handle & 0xF0000000 {
	case 0x80000000:
		return "Weapon"
	case 0x90000000:
		return "Armor"
	case 0xA0000000:
		return "Talisman"
	case 0xB0000000:
		return "Item"
	case 0xC0000000:
		return "Ash of War"
	default:
		return "Unknown"
	}
}

// GetItemsByCategory returns a sorted list of items for a given category.
func GetItemsByCategory(category string) []ItemEntry {
	if category == "all" {
		return GetAllItems()
	}

	var items []ItemEntry

	// isHandleID returns true when the ID uses the inventory-handle prefix
	// (high nibble >= 8 and not a raw game-table ID).
	// Each item exists twice in the data: once with the raw prefix (0x0–0x4)
	// and once with the handle prefix (raw | 0x80000000 mapped per category):
	//   weapons/armor: 0x0→0x8, 0x1→0x9
	//   talismans:     0x2→0xA
	//   items/goods:   0x4→0xB
	//   aows:          0x8→0xC
	// We keep only the handle-prefix variant to avoid duplicates in the UI.
	isHandleID := func(id uint32) bool {
		switch id & 0xF0000000 {
		case 0x80000000, 0x90000000, 0xA0000000, 0xB0000000, 0xC0000000:
			return true
		}
		return false
	}

	processMap := func(source map[uint32]data.ItemData, catName string) {
		for id, item := range source {
			if item.Name == "" || item.Name == "Unarmed" {
				continue
			}
			if !isHandleID(id) {
				continue
			}
			items = append(items, ItemEntry{
				ID:           id,
				Name:         item.Name,
				Category:     catName,
				MaxInventory: item.MaxInventory,
				MaxStorage:   item.MaxStorage,
				MaxUpgrade:   item.MaxUpgrade,
				IconPath:     item.IconPath,
			})
		}
	}

	switch category {
	case "weapons":
		processMap(data.Weapons, "weapons")
	case "bows":
		processMap(data.Bows, "bows")
	case "seals":
		processMap(data.Seals, "seals")
	case "staffs":
		processMap(data.Staffs, "staffs")
	case "shields":
		processMap(data.Shields, "shields")
	case "helms":
		processMap(data.Helms, "helms")
	case "gauntlets":
		processMap(data.Gauntlets, "gauntlets")
	case "leggings":
		processMap(data.Leggings, "leggings")
	case "chest":
		processMap(data.Chest, "chest")
	case "talismans":
		processMap(data.Talismans, "talismans")
	case "aows":
		processMap(data.Aows, "ashes")
	case "ashes":
		processMap(data.StandardAshes, "ashes")
	case "gestures":
		processMap(data.Gestures, "gestures")
	case "sorceries":
		processMap(data.Sorceries, "sorceries")
	case "incantations":
		processMap(data.Incantations, "incantations")
	case "crafting_materials":
		processMap(data.CraftingMaterials, "crafting_materials")
	case "bolstering_materials":
		processMap(data.BolsteringMaterials, "bolstering_materials")
	case "arrows_and_bolts":
		processMap(data.ArrowsAndBolts, "arrows_and_bolts")
	case "sacred_flasks":
		processMap(data.SacredFlasks, "sacred_flasks")
	case "throwing_pots":
		processMap(data.ThrowingPots, "throwing_pots")
	case "perfume_arts":
		processMap(data.PerfumeArts, "perfume_arts")
	case "throwables":
		processMap(data.Throwables, "throwables")
	case "grease":
		processMap(data.Grease, "grease")
	case "misc_tools":
		processMap(data.MiscTools, "misc_tools")
	case "quest_tools":
		processMap(data.QuestTools, "quest_tools")
	case "golden_runes":
		processMap(data.GoldenRunes, "golden_runes")
	case "remembrances":
		processMap(data.Remembrances, "remembrances")
	case "multiplayer":
		processMap(data.Multiplayer, "multiplayer")
	case "consumables":
		processMap(data.Consumables, "consumables")
	case "keyitems":
		processMap(data.Keyitems, "keyitems")
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items
}

// GetItemSubCategory returns the granular category string for an item.
func GetItemSubCategory(id uint32, item data.ItemData, broadCategory string) string {
	if item.Category != "" {
		return item.Category
	}

	// Fallback for items without category
	switch broadCategory {
	case "Weapon":
		return "weapons"
	case "Armor":
		return "chest"
	case "Talisman":
		return "talismans"
	case "Ash of War":
		return "aows"
	default:
		return "consumables"
	}
}

// GetAllItems returns all items from all categories.
func GetAllItems() []ItemEntry {
	var all []ItemEntry
	cats := []string{
		"weapons", "bows", "shields", "staffs", "seals", "arrows_and_bolts",
		"helms", "chest", "gauntlets", "leggings",
		"talismans", "aows", "gestures",
		"ashes",
		"sorceries", "incantations", "crafting_materials",
		"bolstering_materials",
		"sacred_flasks", "throwing_pots", "perfume_arts", "throwables",
		"grease", "misc_tools", "quest_tools", "golden_runes",
		"remembrances", "multiplayer", "consumables", "keyitems",
	}
	for _, cat := range cats {
		all = append(all, GetItemsByCategory(cat)...)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})

	return all
}

// GetAllGraces returns all Sites of Grace as a flat list.
func GetAllGraces() []GraceEntry {
	graces := make([]GraceEntry, 0, len(data.Graces))

	// Map game regions to our specific map filenames
	regionMap := map[string]string{
		"Ainsel River":           "Ainsel River",
		"Altus Plateau":          "Altus Plateau",
		"Caelid":                 "Caelid",
		"Consecrated Snowfield":  "Consecrated Snowfield",
		"Crumbling Farum Azula":  "Crumbling Farum Azula",
		"Deeproot Depths":        "Deeproot Depths",
		"Dragonbarrow":           "Dragonbarrow",
		"Forbidden Lands":        "Forbidden Lands",
		"Lake of Rot":            "Lake of Rot",
		"Leyndell Ashen Capital": "Leyndell, Royal Capital",
		"Leyndell Royal Capital": "Leyndell, Royal Capital",
		"Miquella's Haligtree":   "Miquella's Haligtree",
		"Mohgwyn Palace":         "Mohgwyn Palace",
		"Mt. Gelmir":             "Mt. Gelmir",
		"Shadow of the Erdtree":  "Shadow of the Erdtree",
		"Siofra River":           "Siofra River",
		"Weeping Peninsula":      "Weeping Peninsula",
	}

	for id, fullName := range data.Graces {
		parts := strings.Split(fullName, " (")
		name := parts[0]
		region := "Unknown"

		if len(parts) > 1 {
			rawRegion := strings.TrimSuffix(parts[1], ")")

			// Detailed sub-region mapping
			if rawRegion == "Limgrave" || rawRegion == "Roundtable Hold" {
				region = "Limgrave West" // Default
				eastKeywords := []string{"Mistwood", "Haight", "Siofra River Well", "Third Church of Marika", "Agheel Lake South"}
				for _, kw := range eastKeywords {
					if strings.Contains(name, kw) {
						region = "Limgrave East"
						break
					}
				}
			} else if rawRegion == "Liurnia of the Lakes" {
				region = "Liurnia North" // Default
				eastKeywords := []string{"Eastern Liurnia", "Church of Vows", "Ainsel River Well", "Eastern Tableland", "Jarburg", "Liurnia Highway"}
				westKeywords := []string{"Western Liurnia", "Carian Manor", "Four Belfries", "Revenger's Shack", "Temple Quarter", "Moongazing", "Caria Manor"}

				for _, kw := range eastKeywords {
					if strings.Contains(name, kw) {
						region = "Liurnia East"
						break
					}
				}
				if region == "Liurnia North" {
					for _, kw := range westKeywords {
						if strings.Contains(name, kw) {
							region = "Liurnia West"
							break
						}
					}
				}
			} else if rawRegion == "Mountaintops of the Giants" {
				region = "Mountaintops of the Giants East" // Default
				westKeywords := []string{"Castle Sol", "Snow Valley", "Freezing Lake", "Ancient Snow Valley", "First Church of Marika", "Whiteridge"}
				for _, kw := range westKeywords {
					if strings.Contains(name, kw) {
						region = "Mountaintops of the Giants West"
						break
					}
				}
			} else if mapped, ok := regionMap[rawRegion]; ok {
				region = mapped
			} else {
				region = rawRegion
			}
		}

		graces = append(graces, GraceEntry{
			ID:     id,
			Name:   name,
			Region: region,
		})
	}

	sort.Slice(graces, func(i, j int) bool {
		if graces[i].Region != graces[j].Region {
			return graces[i].Region < graces[j].Region
		}
		return graces[i].Name < graces[j].Name
	})

	return graces
}

// GetEventFlag checks if a specific event flag is set in the bit array.
func GetEventFlag(flags []byte, id uint32) bool {
	info, ok := data.EventFlags[id]
	if !ok {
		return false
	}
	if int(info.Byte) >= len(flags) {
		return false
	}

	return (flags[info.Byte] & (1 << info.Bit)) != 0
}

// SetEventFlag sets or clears a specific event flag in the bit array.
func SetEventFlag(flags []byte, id uint32, value bool) {
	info, ok := data.EventFlags[id]
	if !ok {
		return
	}
	if int(info.Byte) >= len(flags) {
		return
	}

	if value {
		flags[info.Byte] |= (1 << info.Bit)
	} else {
		flags[info.Byte] &= ^(1 << info.Bit)
	}
}
