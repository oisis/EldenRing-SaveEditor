package migration

import (
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

func collectLegacyUnlocks(itemID uint32) []unlockSeed {
	var result []unlockSeed
	add := func(kind string, source map[uint32]uint32, metadata func(uint32) (string, string)) {
		if flagID, ok := source[itemID]; ok {
			name, category := metadata(flagID)
			result = append(result, unlockSeed{
				Kind:     kind,
				FlagID:   flagID,
				Name:     name,
				Category: category,
			})
		}
	}
	noMetadata := func(uint32) (string, string) { return "", "" }
	add("ash_of_war", data.AoWItemToFlagID, noMetadata)
	add("bell_bearing", data.BellBearingItemToFlagID, legacyBellBearingMetadata)
	add("cookbook", data.CookbookItemToFlagID, legacyCookbookMetadata)
	add("map", data.MapFragmentItemToFlagID, noMetadata)
	add("whetblade", data.WhetbladeItemToFlagID, legacyWhetbladeMetadata)
	sort.Slice(result, func(i, j int) bool {
		if result[i].Kind != result[j].Kind {
			return result[i].Kind < result[j].Kind
		}
		return result[i].FlagID < result[j].FlagID
	})
	return result
}

func legacyBellBearingMetadata(flagID uint32) (string, string) {
	value, ok := data.BellBearings[flagID]
	if !ok {
		return "", ""
	}
	return value.Name, value.Category
}

func legacyCookbookMetadata(flagID uint32) (string, string) {
	value, ok := data.Cookbooks[flagID]
	if !ok {
		return "", ""
	}
	return value.Name, value.Category
}

func legacyWhetbladeMetadata(flagID uint32) (string, string) {
	value, ok := data.Whetblades[flagID]
	if !ok {
		return "", ""
	}
	return value.Name, ""
}
