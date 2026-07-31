package migration

import (
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

func collectLegacySwordArtsNames() []swordArtsNameSeed {
	result := make([]swordArtsNameSeed, 0, len(data.SwordArtsNames))
	for id, name := range data.SwordArtsNames {
		result = append(result, swordArtsNameSeed{ID: id, Name: name})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}
