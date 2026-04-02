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

// GetItemName returns the name of an item by its ID and category.
func GetItemName(id uint32, category string) string {
	switch category {
	case "Weapon":
		for baseID, name := range data.Weapons {
			if (id & 0xFFFFFF00) == (baseID & 0xFFFFFF00) {
				level := id - baseID
				if level > 0 {
					return fmt.Sprintf("%s +%d", name, level)
				}
				return name
			}
		}
		return fmt.Sprintf("Unknown Weapon (0x%X)", id)
	case "Armor":
		if name, ok := data.Armors[id]; ok && name != "" {
			return name
		}
		return fmt.Sprintf("Unknown Armor (0x%X)", id)
	case "Talisman":
		if name, ok := data.Talismans[id]; ok && name != "" {
			return name
		}
		return fmt.Sprintf("Unknown Talisman (0x%X)", id)
	case "Item":
		if name, ok := data.Items[id]; ok && name != "" {
			return name
		}
		return fmt.Sprintf("Unknown Item (0x%X)", id)
	case "Ash of War":
		if name, ok := data.Aows[id]; ok && name != "" {
			return name
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
		// For weapons, we only want base items (usually ending in 0)
		if category == "weapons" && id%100 != 0 {
			continue
		}
		// Filter by prefix
		if (id & 0xF0000000) != prefix && !(category == "weapons" && (id&0xF0000000) == 0) {
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
