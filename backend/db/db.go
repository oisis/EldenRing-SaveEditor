package db

import (
	"fmt"
	"github.com/oisis/EldenRing-SaveEditor/backend/db/data"
	"sort"
	"strings"
)

// ItemEntry represents a single item from the game database.
type ItemEntry struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
}

// GraceEntry represents a Site of Grace.
type GraceEntry struct {
	ID     uint32 `json:"id"`
	Name   string `json:"name"`
	Region string `json:"region"`
}

// GetItemName returns the name of an item by its ID, searching across all categories.
func GetItemName(id uint32) string {
	// Normalize ID: Handle prefixes (0x8, 0x9, 0xA, 0xB) to canonical prefixes (0x0, 0x1, 0x2, 0x4)
	normalizedID := id
	prefix := id & 0xF0000000
	switch prefix {
	case 0x80000000:
		normalizedID = id & 0x0FFFFFFF // Weapon handle -> Weapon ID
	case 0x90000000:
		normalizedID = (id & 0x0FFFFFFF) | 0x10000000 // Armor handle -> Armor ID
	case 0xA0000000:
		normalizedID = (id & 0x0FFFFFFF) | 0x20000000 // Talisman handle -> Talisman ID
	case 0xB0000000:
		normalizedID = (id & 0x0FFFFFFF) | 0x40000000 // Item handle -> Item ID
	}

	// 1. Try Weapons with upgrade masking
	if (normalizedID & 0xF0000000) == 0 {
		baseID := normalizedID
		if normalizedID > 100000 {
			baseID = (normalizedID / 100) * 100
		}
		if name, ok := data.Weapons[baseID]; ok && name != "" {
			upgrade := normalizedID % 100
			if upgrade > 0 {
				return fmt.Sprintf("%s +%d", name, upgrade)
			}
			return name
		}
	}

	// 2. Try Armors
	if name, ok := data.Armors[normalizedID]; ok && name != "" {
		return name
	}

	// 3. Try Talismans
	if name, ok := data.Talismans[normalizedID]; ok && name != "" {
		return name
	}

	// 4. Try Items (Goods)
	if name, ok := data.Items[normalizedID]; ok && name != "" {
		return name
	}

	// 5. Try Ash of War (both 0x8 and 0xC prefixes)
	if name, ok := data.Aows[normalizedID]; ok && name != "" {
		return name
	}
	// Also try with 0xC prefix if it was 0x8
	if prefix == 0x80000000 {
		aowID := (id & 0x0FFFFFFF) | 0xC0000000
		if name, ok := data.Aows[aowID]; ok && name != "" {
			return name
		}
	}

	return fmt.Sprintf("Unknown Item (0x%X)", id)
}

// GetItemCategory returns the category name based on the item ID prefix.
func GetItemCategory(id uint32) string {
	switch id & 0xF0000000 {
	case 0x00000000, 0x80000000:
		// Note: 0x8 is also used for AoW handles, but usually we check weapons first
		return "Weapon"
	case 0x10000000, 0x90000000:
		return "Armor"
	case 0x20000000, 0xA0000000:
		return "Talisman"
	case 0x40000000, 0xB0000000:
		return "Item"
	case 0xC0000000:
		return "Ash of War"
	default:
		return "Unknown"
	}
}

// GetItemsByCategory returns a sorted list of items for a given category.
func GetItemsByCategory(category string) []ItemEntry {
	var source map[uint32]string
	var prefix uint32

	switch category {
	case "weapons":
		source = data.Weapons
		prefix = 0x00000000
	case "armors":
		source = data.Armors
		prefix = 0x10000000
	case "items":
		source = data.Items
		prefix = 0x40000000
	case "talismans":
		source = data.Talismans
		prefix = 0x20000000
	case "aows":
		source = data.Aows
		prefix = 0xC0000000
	default:
		return nil
	}

	items := make([]ItemEntry, 0, len(source))
	for id, name := range source {
		if name == "" {
			continue
		}
		// Filter for canonical item ID prefix to avoid duplicates with handle prefixes
		if (id & 0xF0000000) != prefix {
			continue
		}
		items = append(items, ItemEntry{ID: id, Name: name})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items
}

// GetAllGraces returns all Sites of Grace as a flat list.
func GetAllGraces() []GraceEntry {
	graces := make([]GraceEntry, 0, len(data.Graces))
	
	for id, fullName := range data.Graces {
		parts := strings.Split(fullName, " (")
		name := parts[0]
		region := "Unknown"
		if len(parts) > 1 {
			region = strings.TrimSuffix(parts[1], ")")
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
