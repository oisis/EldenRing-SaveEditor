package db

import (
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

// GetItemsByCategory returns a sorted list of items for a given category.
func GetItemsByCategory(category string) []ItemEntry {
	var source map[uint32]string
	switch category {
	case "weapons":
		source = data.Weapons
	case "armors":
		source = data.Armors
	case "items":
		source = data.Items
	case "talismans":
		source = data.Talismans
	default:
		return nil
	}

	items := make([]ItemEntry, 0, len(source))
	for id, name := range source {
		if name == "" {
			continue
		}
		items = append(items, ItemEntry{ID: id, Name: name})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items
}

// GetGracesByRegion returns Sites of Grace grouped and sorted by region.
func GetGracesByRegion() map[string][]GraceEntry {
	grouped := make(map[string][]GraceEntry)
	
	for id, fullName := range data.Graces {
		parts := strings.Split(fullName, " (")
		name := parts[0]
		region := "Unknown"
		if len(parts) > 1 {
			region = strings.TrimSuffix(parts[1], ")")
		}
		
		grouped[region] = append(grouped[region], GraceEntry{
			ID:     id,
			Name:   name,
			Region: region,
		})
	}

	// Sort graces within each region
	for region := range grouped {
		sort.Slice(grouped[region], func(i, j int) bool {
			return grouped[region][i].Name < grouped[region][j].Name
		})
	}

	return grouped
}
