package db

import (
	"github.com/oisis/EldenRing-SaveEditor/backend/db/data"
	"sort"
)

// ItemEntry represents a single item from the game database.
type ItemEntry struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
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
		// Skip empty names
		if name == "" {
			continue
		}
		items = append(items, ItemEntry{ID: id, Name: name})
	}

	// Sort alphabetically by name
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items
}
