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
	Region string `json:"region"`
}

// GetItemData returns the full metadata of an item by its ID and category.
func GetItemData(id uint32, category string) data.ItemData {
	switch category {
	case "Weapon":
		if item, ok := data.Weapons[id]; ok {
			return item
		}
	case "Armor":
		if item, ok := data.Armors[id]; ok {
			return item
		}
	case "Talisman":
		if item, ok := data.Talismans[id]; ok {
			return item
		}
	case "Item":
		if item, ok := data.Items[id]; ok {
			return item
		}
		if item, ok := data.SpiritAshes[id]; ok {
			return item
		}
		if item, ok := data.Gestures[id]; ok {
			return item
		}
	case "Spirit Ash":
		if item, ok := data.SpiritAshes[id]; ok {
			return item
		}
	case "Gesture":
		if item, ok := data.Gestures[id]; ok {
			return item
		}
	case "Ash of War":
		if item, ok := data.Aows[id]; ok {
			return item
		}
	}
	return data.ItemData{Name: GetItemName(id, category)}
}

// GetItemName returns the name of an item by its ID and category.
func GetItemName(id uint32, category string) string {
	switch category {
	case "Weapon":
		for baseID, item := range data.Weapons {
			if (id & 0xFFFFFF00) == (baseID & 0xFFFFFF00) {
				level := id - baseID
				if level > 0 {
					return fmt.Sprintf("%s +%d", item.Name, level)
				}
				return item.Name
			}
		}
		return fmt.Sprintf("Unknown Weapon (0x%X)", id)
	case "Armor":
		if item, ok := data.Armors[id]; ok && item.Name != "" {
			return item.Name
		}
		return fmt.Sprintf("Unknown Armor (0x%X)", id)
	case "Talisman":
		if item, ok := data.Talismans[id]; ok && item.Name != "" {
			return item.Name
		}
		return fmt.Sprintf("Unknown Talisman (0x%X)", id)
	case "Item":
		if item, ok := data.Items[id]; ok && item.Name != "" {
			return item.Name
		}
		if item, ok := data.SpiritAshes[id]; ok && item.Name != "" {
			return item.Name
		}
		if item, ok := data.Gestures[id]; ok && item.Name != "" {
			return item.Name
		}
		return fmt.Sprintf("Unknown Item (0x%X)", id)
	case "Spirit Ash":
		if item, ok := data.SpiritAshes[id]; ok && item.Name != "" {
			return item.Name
		}
		return fmt.Sprintf("Unknown Spirit Ash (0x%X)", id)
	case "Gesture":
		if item, ok := data.Gestures[id]; ok && item.Name != "" {
			return item.Name
		}
		return fmt.Sprintf("Unknown Gesture (0x%X)", id)
	case "Ash of War":
		if item, ok := data.Aows[id]; ok && item.Name != "" {
			return item.Name
		}
		return fmt.Sprintf("Unknown Ash of War (0x%X)", id)
	default:
		return fmt.Sprintf("Unknown Item (0x%X)", id)
	}
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

	// Define which maps to search based on category
	searchWeapons := false
	searchArmors := false
	searchItems := false
	searchTalismans := false
	searchAows := false
	searchSpiritAshes := false
	searchGestures := false

	switch category {
	case "weapons":
		searchWeapons = true
	case "bows", "seals", "staffs", "shields":
		searchWeapons = true
	case "armors", "helms", "gauntlets", "leggings", "chest":
		searchArmors = true
	case "items", "sorceries", "incantations", "materials", "upgrade", "keyitems", "consumables":
		searchItems = true
	case "spiritashes":
		searchSpiritAshes = true
	case "gestures":
		searchGestures = true
	case "ammo":
		searchItems = true
		searchWeapons = true // Arrows/Bolts can be in both
	case "talismans":
		searchTalismans = true
	case "aows":
		searchAows = true
	}

	processMap := func(source map[uint32]data.ItemData, prefix uint32, catName string) {
		for id, item := range source {
			if item.Name == "" || item.Name == "Unarmed" {
				continue
			}

			// Filter by prefix
			if (id&0xF0000000) != prefix && !(prefix == 0 && (id&0xF0000000) == 0) {
				continue
			}

			// For weapons/bows/etc, we only want base items (usually ending in 0)
			if prefix == 0 && id%100 != 0 {
				continue
			}

			// Sub-category filtering
			itemSubCat := GetItemSubCategory(id, item, getBroadCategory(prefix))

			if category != "all" {
				if category == "weapons" {
					// "weapons" category should only show actual weapons, not bows, shields, etc.
					if itemSubCat != "weapons" {
						continue
					}
				} else if category == "armors" {
					// "armors" category should only show full sets/chest pieces if we want,
					// but usually it's a catch-all. Let's make it only show "chest" or "armors".
					if itemSubCat != "armors" && itemSubCat != "chest" {
						continue
					}
				} else if category == "items" {
					// "items" is a catch-all for goods, but let's exclude specific sub-cats if needed.
					// For now, keep it as is or filter to "consumables".
					if itemSubCat != "consumables" && itemSubCat != "items" {
						continue
					}
				} else {
					// Specific sub-category requested (e.g., "shields", "bows")
					if itemSubCat != category {
						continue
					}
				}
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

	if searchWeapons {
		processMap(data.Weapons, 0x00000000, "weapons")
	}
	if searchArmors {
		processMap(data.Armors, 0x10000000, "armor")
	}
	if searchItems {
		processMap(data.Items, 0x40000000, "goods")
	}
	if searchTalismans {
		processMap(data.Talismans, 0x20000000, "talismans")
	}
	if searchAows {
		processMap(data.Aows, 0xC0000000, "ashes")
	}
	if searchSpiritAshes {
		processMap(data.SpiritAshes, 0x40000000, "spiritashes")
	}
	if searchGestures {
		processMap(data.Gestures, 0x40000000, "gestures")
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items
}

func getBroadCategory(prefix uint32) string {
	switch prefix {
	case 0x00000000:
		return "Weapon"
	case 0x10000000:
		return "Armor"
	case 0x20000000:
		return "Talisman"
	case 0x40000000:
		return "Item"
	case 0xC0000000:
		return "Ash of War"
	default:
		return "Unknown"
	}
}

// GetItemSubCategory returns the granular category string for an item.
func GetItemSubCategory(id uint32, item data.ItemData, broadCategory string) string {
	if broadCategory == "Weapon" {
		if itemMatchesCategory(id, item, "ammo") {
			return "ammo"
		}
		if itemMatchesCategory(id, item, "bows") {
			return "bows"
		}
		if itemMatchesCategory(id, item, "seals") {
			return "seals"
		}
		if itemMatchesCategory(id, item, "staffs") {
			return "staffs"
		}
		if itemMatchesCategory(id, item, "shields") {
			return "shields"
		}
		return "weapons"
	}
	if broadCategory == "Armor" {
		if itemMatchesCategory(id, item, "helms") {
			return "helms"
		}
		if itemMatchesCategory(id, item, "gauntlets") {
			return "gauntlets"
		}
		if itemMatchesCategory(id, item, "leggings") {
			return "leggings"
		}
		if itemMatchesCategory(id, item, "chest") {
			return "chest"
		}
		return "armors"
	}
	if broadCategory == "Talisman" {
		return "talismans"
	}
	if broadCategory == "Ash of War" {
		return "aows"
	}

	// For Items (Goods), check granular categories
	if itemMatchesCategory(id, item, "sorceries") {
		return "sorceries"
	}
	if itemMatchesCategory(id, item, "incantations") {
		return "incantations"
	}
	if itemMatchesCategory(id, item, "spiritashes") {
		return "spiritashes"
	}
	if itemMatchesCategory(id, item, "gestures") {
		return "gestures"
	}
	if itemMatchesCategory(id, item, "materials") {
		return "materials"
	}
	if itemMatchesCategory(id, item, "upgrade") {
		return "upgrade"
	}
	if itemMatchesCategory(id, item, "ammo") {
		return "ammo"
	}
	if itemMatchesCategory(id, item, "keyitems") {
		return "keyitems"
	}

	return "consumables"
}

func itemMatchesCategory(id uint32, item data.ItemData, category string) bool {
	nameLower := strings.ToLower(item.Name)
	switch category {
	case "bows":
		return strings.Contains(nameLower, "bow") || strings.Contains(nameLower, "ballista") || strings.Contains(nameLower, "crossbow")
	case "seals":
		return strings.Contains(nameLower, "seal")
	case "staffs":
		return strings.Contains(nameLower, "staff") || strings.Contains(nameLower, "scepter")
	case "shields":
		return strings.Contains(nameLower, "shield") || strings.Contains(nameLower, "buckler") ||
			strings.Contains(nameLower, "roundshield") || strings.Contains(nameLower, "greatshield") ||
			strings.Contains(nameLower, "towershield") || strings.Contains(nameLower, "mirrorshield")
	case "helms":
		return strings.Contains(nameLower, "helm") || strings.Contains(nameLower, "hood") ||
			strings.Contains(nameLower, "mask") || strings.Contains(nameLower, "crown") ||
			strings.Contains(nameLower, "headband") || strings.Contains(nameLower, "hat") ||
			strings.Contains(nameLower, "coif")
	case "gauntlets":
		return strings.Contains(nameLower, "gauntlets") || strings.Contains(nameLower, "gloves") ||
			strings.Contains(nameLower, "bracers") || strings.Contains(nameLower, "manchettes") ||
			strings.Contains(nameLower, "bracer")
	case "leggings":
		return strings.Contains(nameLower, "greaves") || strings.Contains(nameLower, "trousers") ||
			strings.Contains(nameLower, "boots") || strings.Contains(nameLower, "leggings") ||
			strings.Contains(nameLower, "gaiters") || strings.Contains(nameLower, "shoes") ||
			strings.Contains(nameLower, "skirt")
	case "chest":
		return !itemMatchesCategory(id, item, "helms") &&
			!itemMatchesCategory(id, item, "gauntlets") &&
			!itemMatchesCategory(id, item, "leggings")
	case "sorceries":
		return id >= 0x40000FA0 && id <= 0x4000157C
	case "incantations":
		return id >= 0x40001770 && id <= 0x40002134
	case "spiritashes":
		return item.MaxUpgrade == 10 || strings.Contains(item.IconPath, "spirit_ashes")
	case "gestures":
		return strings.Contains(item.IconPath, "gestures")
	case "materials":
		craftingKeywords := []string{
			"mushroom", "leaf", "flower", "fruit", "butterfly", "firefly",
			"root", "moss", "resin", "bone", "feather", "liver", "meat",
			"blood", "eye", "skin", "horn", "fang", "claw", "scale",
			"shell", "egg", "string", "crystal", "fragment", "shard",
			"arteria", "starlight", "dew", "nectar", "mold", "calculus",
		}
		for _, kw := range craftingKeywords {
			if strings.Contains(nameLower, kw) {
				return true
			}
		}
		return false
	case "upgrade":
		return strings.Contains(nameLower, "smithing stone") || strings.Contains(nameLower, "glovewort") || strings.Contains(nameLower, "somber")
	case "ammo":
		if strings.Contains(nameLower, "bolt of gransax") {
			return false
		}
		return strings.Contains(nameLower, "arrow") || strings.Contains(nameLower, "bolt")
	case "keyitems":
		keyItemKeywords := []string{
			"key", "map", "letter", "note", "painting", "bell bearing",
			"crystal tear", "great rune", "mending rune", "remembrance",
			"shackle", "whetblade", "cookbook", "scroll",
			"cracked pot", "ritual pot", "perfume bottle", "memory stone",
			"talisman pouch", "withered finger", "furled finger", "severer",
			"effigy", "cipher ring", "bloody finger", "recusant finger",
			"whistle", "physick", "telescope", "lantern", "tonic",
		}
		for _, kw := range keyItemKeywords {
			if strings.Contains(nameLower, kw) {
				return true
			}
		}
		return false
	case "consumables":
		return !itemMatchesCategory(id, item, "sorceries") &&
			!itemMatchesCategory(id, item, "incantations") &&
			!itemMatchesCategory(id, item, "materials") &&
			!itemMatchesCategory(id, item, "upgrade") &&
			!itemMatchesCategory(id, item, "ammo") &&
			!itemMatchesCategory(id, item, "keyitems") &&
			!itemMatchesCategory(id, item, "spiritashes") &&
			!itemMatchesCategory(id, item, "gestures")
	}
	return false
}

// GetAllItems returns all items from all categories.
func GetAllItems() []ItemEntry {
	var all []ItemEntry
	cats := []string{"weapons", "armors", "items", "talismans", "aows", "spiritashes", "gestures"}
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
