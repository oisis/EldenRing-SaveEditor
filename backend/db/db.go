package db

import (
	"fmt"
	"github.com/oisis/EldenRing-SaveEditor/backend/db/data"
	"sort"
	"strings"
)

// ItemEntry represents a single item from the game database.
type ItemEntry struct {
	ID           uint32   `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	MaxInventory uint32   `json:"maxInventory"`
	MaxStorage   uint32   `json:"maxStorage"`
	MaxUpgrade   uint32   `json:"maxUpgrade"`
	IconPath     string   `json:"iconPath"`
	Flags        []string `json:"flags"`
}

// InfuseType represents a weapon infusion type and its ID offset.
type InfuseType struct {
	Name   string `json:"name"`
	Offset int    `json:"offset"`
}

// InfuseTypes lists all weapon infusion types in Elden Ring order.
var InfuseTypes = []InfuseType{
	{"Standard", 0},
	{"Heavy", 100},
	{"Keen", 200},
	{"Quality", 300},
	{"Fire", 400},
	{"Flame Art", 500},
	{"Lightning", 600},
	{"Sacred", 700},
	{"Magic", 800},
	{"Cold", 900},
	{"Poison", 1000},
	{"Blood", 1100},
	{"Occult", 1200},
}

// GraceEntry represents a Site of Grace.
type GraceEntry struct {
	ID      uint32 `json:"id"`
	Name    string `json:"name"`
	Region  string `json:"region"`
	Visited bool   `json:"visited"`
}

// globalItemIndex provides O(1) item lookup by ID, built once at startup.
var globalItemIndex map[uint32]data.ItemData

func init() {
	allMaps := []map[uint32]data.ItemData{
		data.Weapons, data.RangedAndCatalysts, data.Shields, data.ArrowsAndBolts,
		data.Helms, data.Chest, data.Arms, data.Legs,
		data.Talismans, data.Aows, data.Gestures,
		data.StandardAshes,
		data.Sorceries, data.Incantations, data.CraftingMaterials,
		data.BolsteringMaterials, data.KeyItems,
		data.Tools,
	}
	size := 0
	for _, m := range allMaps {
		size += len(m)
	}
	globalItemIndex = make(map[uint32]data.ItemData, size)
	for _, m := range allMaps {
		for id, entry := range m {
			globalItemIndex[id] = entry
		}
	}
}

// GetItemData returns the full metadata of an item by its ID via the global index.
func GetItemData(id uint32) data.ItemData {
	if item, ok := globalItemIndex[id]; ok {
		return item
	}
	return data.ItemData{}
}

// findAshBase searches StandardAshes for the base (+0) entry matching the given name prefix.
// baseName must already have any " +N" suffix stripped. Returns (entry, baseID) or zero values.
func findAshBase(baseName string, idPrefix uint32) (data.ItemData, uint32) {
	for ashID, ashEntry := range data.StandardAshes {
		if ashEntry.Name == baseName {
			return ashEntry, (ashID&0x0FFFFFFF)|idPrefix
		}
	}
	return data.ItemData{}, 0
}

// GetItemDataFuzzy returns item metadata for an exact ID, or falls back to a base lookup for:
//   - Spirit ashes (0x40... PC or 0xB0... PS4): each upgrade level is a separate DB entry;
//     this finds the base (+0) entry so currentUpgrade can be computed from the ID difference.
//   - Upgraded/infused weapons: PS4 (0x80...) and PC (0x00...) via byte-masked base search.
//
// The returned ItemData.Name is the base name without "+N" (caller appends "+N" if needed).
func GetItemDataFuzzy(id uint32) (data.ItemData, uint32) {
	exact := GetItemData(id)
	if exact.Name != "" {
		// Spirit ashes store each upgrade level as a separate DB entry with "+N" in the name.
		// Find the base (+0) entry so currentUpgrade = id - baseID is computed correctly.
		if exact.Category == "ashes" && strings.Contains(exact.Name, " +") {
			baseName := exact.Name[:strings.Index(exact.Name, " +")]
			if entry, baseID := findAshBase(baseName, id&0xF0000000); baseID != 0 {
				return entry, baseID
			}
		}
		return exact, id
	}

	prefix := id & 0xF0000000

	// PS4 spirit ashes use 0xB0... goods IDs; the DB stores them as 0x40... PC IDs.
	if prefix == 0xB0000000 {
		pcID := (id & 0x0FFFFFFF) | 0x40000000
		pcEntry := GetItemData(pcID)
		if pcEntry.Name != "" && pcEntry.Category == "ashes" {
			baseName := pcEntry.Name
			if idx := strings.Index(baseName, " +"); idx >= 0 {
				baseName = baseName[:idx]
			}
			if entry, baseID := findAshBase(baseName, 0xB0000000); baseID != 0 {
				return entry, baseID
			}
			return pcEntry, id
		}
	}

	// Weapon fuzzy search: handles PS4 (0x80...) and PC (0x00...) upgraded/infused weapons.
	// Uses a byte-masked comparison (id & 0xFFFFFF00) which is accurate for standard upgrades
	// (offset 0–25). Heavily infused weapons (offset > 255) are not matched here.
	if prefix == 0x80000000 || prefix == 0 {
		weaponMaps := []map[uint32]data.ItemData{data.Weapons, data.RangedAndCatalysts, data.Shields}
		masked := id & 0xFFFFFF00
		for _, m := range weaponMaps {
			for baseID, item := range m {
				if baseID&0xFFFFFF00 == masked && item.Name != "" {
					return item, baseID
				}
			}
		}
	}

	return data.ItemData{}, id
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
	weaponMaps := []map[uint32]data.ItemData{data.RangedAndCatalysts, data.Shields}
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

	return fmt.Sprintf("Unknown Item (0x%X)", id)
}

// IsArrowID returns true if the given item ID corresponds to an arrow or bolt.
// Arrows/bolts have 0x82... prefix (PS4) or 0x02... prefix (PC) and are stackable despite
// appearing weapon-like in the GaItems type system.
func IsArrowID(id uint32) bool {
	_, ok := data.ArrowsAndBolts[id]
	return ok
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

// ps4Prefix maps PS4 item-type prefixes to their PC equivalents.
// All data maps store both variants; this lets us pick the right one per platform.
//
//	PS4 → PC: 0x80→0x00 (Weapon), 0x90→0x10 (Armor), 0xA0→0x20 (Accessory),
//	          0xB0→0x40 (Goods), 0xC0→0x80 (AoW)
func toPCPrefix(ps4 uint32) uint32 {
	switch ps4 {
	case 0x80000000:
		return 0x00000000
	case 0x90000000:
		return 0x10000000
	case 0xA0000000:
		return 0x20000000
	case 0xB0000000:
		return 0x40000000
	case 0xC0000000:
		return 0x80000000
	default:
		return ps4
	}
}

// GetItemsByCategory returns a sorted list of items for a given category.
// platform must be "PC" or "PS4" — it controls which ID variant is returned.
func GetItemsByCategory(category, platform string) []ItemEntry {
	if category == "all" {
		return GetAllItems(platform)
	}

	isPC := platform == "PC"

	var items []ItemEntry

	// processMap adds items from source to the result list.
	// ps4Prefix is the PS4-side upper nibble for this data map; it is converted to the
	// PC equivalent when platform == "PC".
	processMap := func(source map[uint32]data.ItemData, catName string, ps4Prefix uint32) {
		wantPrefix := ps4Prefix
		if isPC {
			wantPrefix = toPCPrefix(ps4Prefix)
		}
		for id, item := range source {
			if item.Name == "" || item.Name == "Unarmed" {
				continue
			}
			if id&0xF0000000 != wantPrefix {
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
				Flags:        item.Flags,
			})
		}
	}

	switch category {
	case "melee_armaments":
		processMap(data.Weapons, "melee_armaments", 0x80000000)
		items = filterInfuseVariants(items)
	case "ranged_and_catalysts":
		processMap(data.RangedAndCatalysts, "ranged_and_catalysts", 0x80000000)
		items = filterInfuseVariants(items)
	case "shields":
		processMap(data.Shields, "shields", 0x80000000)
		items = filterInfuseVariants(items)
	case "head":
		processMap(data.Helms, "head", 0x90000000)
	case "arms":
		processMap(data.Arms, "arms", 0x90000000)
	case "legs":
		processMap(data.Legs, "legs", 0x90000000)
	case "chest":
		processMap(data.Chest, "chest", 0x90000000)
	case "talismans":
		processMap(data.Talismans, "talismans", 0xA0000000)
	case "ashes_of_war":
		processMap(data.Aows, "ashes_of_war", 0xC0000000)
	case "ashes":
		// StandardAshes has both 0x40... (PC) and 0xB0... (PS4) entries for each upgrade level.
		// Iterate only 0x40... base (+0) entries and remap to PS4 prefix when needed.
		ashTargetPrefix := uint32(0xB0000000)
		if isPC {
			ashTargetPrefix = 0x40000000
		}
		for id, item := range data.StandardAshes {
			if id&0xF0000000 != 0x40000000 {
				continue // always iterate from PC entries to avoid duplicates
			}
			if item.Name == "" || strings.Contains(item.Name, " +") {
				continue
			}
			items = append(items, ItemEntry{
				ID:           (id & 0x0FFFFFFF) | ashTargetPrefix,
				Name:         item.Name,
				Category:     "ashes",
				MaxInventory: item.MaxInventory,
				MaxStorage:   item.MaxStorage,
				MaxUpgrade:   item.MaxUpgrade,
				IconPath:     item.IconPath,
				Flags:        item.Flags,
			})
		}
	case "gestures":
		processMap(data.Gestures, "gestures", 0xB0000000)
	case "sorceries":
		processMap(data.Sorceries, "sorceries", 0xB0000000)
	case "incantations":
		processMap(data.Incantations, "incantations", 0xB0000000)
	case "crafting_materials":
		processMap(data.CraftingMaterials, "crafting_materials", 0xB0000000)
	case "bolstering_materials":
		processMap(data.BolsteringMaterials, "bolstering_materials", 0xB0000000)
	case "arrows_and_bolts":
		// ArrowsAndBolts has both PC (0x02...) and PS4 (0x82...) entries.
		// Filter by platform bit 31.
		for id, item := range data.ArrowsAndBolts {
			if item.Name == "" {
				continue
			}
			hasPS4Bit := id&0x80000000 != 0
			if isPC && hasPS4Bit {
				continue
			}
			if !isPC && !hasPS4Bit {
				continue
			}
			items = append(items, ItemEntry{
				ID:           id,
				Name:         item.Name,
				Category:     "arrows_and_bolts",
				MaxInventory: item.MaxInventory,
				MaxStorage:   item.MaxStorage,
				MaxUpgrade:   item.MaxUpgrade,
				IconPath:     item.IconPath,
				Flags:        item.Flags,
			})
		}
	case "tools":
		wantPrefix := uint32(0xB0000000)
		if isPC {
			wantPrefix = 0x40000000
		}
		for id, item := range data.Tools {
			if item.Name == "" {
				continue
			}
			if id&0xF0000000 != wantPrefix {
				continue
			}
			// Filter upgraded Flask variants — only keep base versions (no " +N" suffix)
			if strings.Contains(item.Name, "Flask of") && strings.Contains(item.Name, " +") {
				continue
			}
			items = append(items, ItemEntry{
				ID:           id,
				Name:         item.Name,
				Category:     "tools",
				MaxInventory: item.MaxInventory,
				MaxStorage:   item.MaxStorage,
				MaxUpgrade:   item.MaxUpgrade,
				IconPath:     item.IconPath,
				Flags:        item.Flags,
			})
		}
	case "key_items":
		processMap(data.KeyItems, "key_items", 0xB0000000)
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
		return "ashes_of_war"
	default:
		return "tools"
	}
}

// GetAllItems returns all items from all categories for the given platform.
func GetAllItems(platform string) []ItemEntry {
	var all []ItemEntry
	cats := []string{
		"melee_armaments", "ranged_and_catalysts", "shields", "arrows_and_bolts",
		"head", "chest", "arms", "legs",
		"talismans", "ashes_of_war", "gestures",
		"ashes",
		"sorceries", "incantations", "crafting_materials",
		"bolstering_materials", "key_items",
		"tools",
	}
	for _, cat := range cats {
		all = append(all, GetItemsByCategory(cat, platform)...)
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
// For IDs in the lookup table, uses the precomputed byte/bit offsets.
// For all other IDs (e.g. Sites of Grace), uses the standard formula:
// byte = id / 8, bit = 7 - (id % 8).
// Returns error if the computed byte offset is out of bounds.
func GetEventFlag(flags []byte, id uint32) (bool, error) {
	var byteIdx uint32
	var bitIdx uint8
	if info, ok := data.EventFlags[id]; ok {
		byteIdx = info.Byte
		bitIdx = info.Bit
	} else {
		byteIdx = id / 8
		bitIdx = uint8(7 - (id % 8))
	}
	if int(byteIdx) >= len(flags) {
		return false, fmt.Errorf("event flag %d (byte %d) out of bounds (flags len %d)", id, byteIdx, len(flags))
	}
	return (flags[byteIdx] & (1 << bitIdx)) != 0, nil
}

// filterInfuseVariants removes infuse-variant entries from a weapon item list.
// A variant is detected when id - N×100 (N=1..12) exists in the same list,
// meaning it is a non-standard infuse copy of a base weapon already present.
// Items with maxUpgrade != 25 are always kept (boss weapons, non-upgradeable).
func filterInfuseVariants(items []ItemEntry) []ItemEntry {
	idSet := make(map[uint32]bool, len(items))
	for _, item := range items {
		idSet[item.ID] = true
	}

	result := items[:0]
	for _, item := range items {
		if item.MaxUpgrade != 25 {
			result = append(result, item)
			continue
		}
		isVariant := false
		for n := uint32(1); n <= 12; n++ {
			offset := n * 100
			if item.ID >= offset && idSet[item.ID-offset] {
				isVariant = true
				break
			}
		}
		if !isVariant {
			result = append(result, item)
		}
	}
	return result
}

// GetInfuseTypes returns all weapon infusion types.
func GetInfuseTypes() []InfuseType {
	return InfuseTypes
}

// SetEventFlag sets or clears a specific event flag in the bit array.
// For IDs in the lookup table, uses the precomputed byte/bit offsets.
// For all other IDs (e.g. Sites of Grace), uses the standard formula:
// byte = id / 8, bit = 7 - (id % 8).
// Returns error if the computed byte offset is out of bounds.
func SetEventFlag(flags []byte, id uint32, value bool) error {
	var byteIdx uint32
	var bitIdx uint8
	if info, ok := data.EventFlags[id]; ok {
		byteIdx = info.Byte
		bitIdx = info.Bit
	} else {
		byteIdx = id / 8
		bitIdx = uint8(7 - (id % 8))
	}
	if int(byteIdx) >= len(flags) {
		return fmt.Errorf("event flag %d (byte %d) out of bounds (flags len %d)", id, byteIdx, len(flags))
	}
	if value {
		flags[byteIdx] |= (1 << bitIdx)
	} else {
		flags[byteIdx] &= ^(1 << bitIdx)
	}
	return nil
}
