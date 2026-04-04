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
		return fmt.Sprintf("Unknown Item (0x%X)", id)
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

	var source map[uint32]data.ItemData
	var prefix uint32
	var catName string

	switch category {
	case "weapons", "bows", "seals", "staffs", "shields":
		source = data.Weapons
		prefix = 0x00000000
		catName = "weapons"
	case "armors", "helms", "gauntlets", "leggings", "chest":
		source = data.Armors
		prefix = 0x10000000
		catName = "armor"
	case "items", "sorceries", "incantations", "materials", "upgrade", "ammo", "keyitems", "consumables", "spiritashes":
		source = data.Items
		prefix = 0x40000000
		catName = "goods"
	case "talismans":
		source = data.Talismans
		prefix = 0x20000000
		catName = "talismans"
	case "aows":
		source = data.Aows
		prefix = 0xC0000000
		catName = "ashes"
	default:
		return nil
	}

	items := make([]ItemEntry, 0, len(source))
	for id, item := range source {
		if item.Name == "" {
			continue
		}
		// For weapons, we only want base items (usually ending in 0)
		if (category == "weapons" || category == "bows" || category == "seals" || category == "staffs" || category == "shields") && id%100 != 0 {
			continue
		}
		// Filter by prefix
		if (id & 0xF0000000) != prefix && !(prefix == 0 && (id&0xF0000000) == 0) {
			continue
		}

		// Sub-category filtering
		if category != "weapons" && category != "armors" && category != "items" && category != "all" {
			if !itemMatchesCategory(id, item.Name, category) {
				continue
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

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items
}

// GetItemSubCategory returns the granular category string for an item.
func GetItemSubCategory(id uint32, name string, broadCategory string) string {
	if broadCategory == "Weapon" {
		if itemMatchesCategory(id, name, "bows") {
			return "bows"
		}
		if itemMatchesCategory(id, name, "seals") {
			return "seals"
		}
		if itemMatchesCategory(id, name, "staffs") {
			return "staffs"
		}
		if itemMatchesCategory(id, name, "shields") {
			return "shields"
		}
		return "weapons"
	}
	if broadCategory == "Armor" {
		if itemMatchesCategory(id, name, "helms") {
			return "helms"
		}
		if itemMatchesCategory(id, name, "gauntlets") {
			return "gauntlets"
		}
		if itemMatchesCategory(id, name, "leggings") {
			return "leggings"
		}
		if itemMatchesCategory(id, name, "chest") {
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
	if itemMatchesCategory(id, name, "sorceries") {
		return "sorceries"
	}
	if itemMatchesCategory(id, name, "incantations") {
		return "incantations"
	}
	if itemMatchesCategory(id, name, "spiritashes") {
		return "spiritashes"
	}
	if itemMatchesCategory(id, name, "materials") {
		return "materials"
	}
	if itemMatchesCategory(id, name, "upgrade") {
		return "upgrade"
	}
	if itemMatchesCategory(id, name, "ammo") {
		return "ammo"
	}
	if itemMatchesCategory(id, name, "keyitems") {
		return "keyitems"
	}

	return "consumables"
}

func itemMatchesCategory(id uint32, name string, category string) bool {
	nameLower := strings.ToLower(name)
	switch category {
	case "bows":
		return strings.Contains(nameLower, "bow") || strings.Contains(nameLower, "ballista")
	case "seals":
		return strings.Contains(nameLower, "seal")
	case "staffs":
		return strings.Contains(nameLower, "staff") || strings.Contains(nameLower, "scepter")
	case "shields":
		return strings.Contains(nameLower, "shield") || strings.Contains(nameLower, "buckler")
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
		return !itemMatchesCategory(id, name, "helms") && 
			!itemMatchesCategory(id, name, "gauntlets") && 
			!itemMatchesCategory(id, name, "leggings")
	case "sorceries":
		return id >= 0x40000FA0 && id <= 0x4000157C
	case "incantations":
		return id >= 0x40001770 && id <= 0x40002134
	case "spiritashes":
		return strings.Contains(nameLower, "ashes") && (id >= 0x40032898 || strings.Contains(nameLower, "spirit"))
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
		return strings.Contains(nameLower, "arrow") || strings.Contains(nameLower, "bolt")
	case "keyitems":
		keyItemKeywords := []string{
			"key", "map", "letter", "note", "painting", "bell bearing",
			"crystal tear", "great rune", "mending rune", "remembrance",
			"shackle", "whetblade", "cookbook", "scroll", "gesture",
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
		return !itemMatchesCategory(id, name, "sorceries") &&
			!itemMatchesCategory(id, name, "incantations") &&
			!itemMatchesCategory(id, name, "materials") &&
			!itemMatchesCategory(id, name, "upgrade") &&
			!itemMatchesCategory(id, name, "ammo") &&
			!itemMatchesCategory(id, name, "keyitems") &&
			!itemMatchesCategory(id, name, "spiritashes")
	}
	return false
}

// GetAllItems returns all items from all categories.
func GetAllItems() []ItemEntry {
	var all []ItemEntry
	cats := []string{"weapons", "armors", "items", "talismans", "aows"}
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
		"Ainsel River":               "Ainsel River",
		"Altus Plateau":              "Altus Plateau",
		"Caelid":                     "Caelid",
		"Consecrated Snowfield":      "Consecrated Snowfield",
		"Crumbling Farum Azula":      "Crumbling Farum Azula",
		"Deeproot Depths":            "Deeproot Depths",
		"Dragonbarrow":               "Dragonbarrow",
		"Forbidden Lands":            "Forbidden Lands",
		"Lake of Rot":                "Lake of Rot",
		"Leyndell Ashen Capital":     "Leyndell, Royal Capital",
		"Leyndell Royal Capital":     "Leyndell, Royal Capital",
		"Miquella's Haligtree":       "Miquella's Haligtree",
		"Mohgwyn Palace":             "Mohgwyn Palace",
		"Mt. Gelmir":                 "Mt. Gelmir",
		"Shadow of the Erdtree":      "Shadow of the Erdtree",
		"Siofra River":               "Siofra River",
		"Weeping Peninsula":          "Weeping Peninsula",
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
