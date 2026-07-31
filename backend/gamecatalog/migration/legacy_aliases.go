package migration

import (
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

func collectLegacyAliases() []aliasSeed {
	result := make([]aliasSeed, 0, len(data.TechnicalItemAliases))
	for aliasID, canonicalID := range data.TechnicalItemAliases {
		result = append(result, aliasSeed{AliasID: aliasID, CanonicalID: canonicalID})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].AliasID < result[j].AliasID
	})
	return result
}
