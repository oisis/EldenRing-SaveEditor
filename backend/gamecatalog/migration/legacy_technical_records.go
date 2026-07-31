package migration

import (
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

const legacyGoodsItemPrefix uint32 = 0x40000000

func collectLegacyTechnicalRecords(items []seed) []technicalRecordSeed {
	itemIDs := make(map[uint32]struct{}, len(items))
	for _, item := range items {
		itemIDs[item.ID] = struct{}{}
	}
	result := make([]technicalRecordSeed, 0)
	for id, description := range data.Descriptions {
		if id&0xF0000000 != legacyGoodsItemPrefix {
			continue
		}
		if _, isItem := itemIDs[id]; isItem {
			continue
		}
		limits, hasLimits := data.GameLimitsByItemID[id]
		if !hasLimits || !limits.InventoryKnown || !limits.StorageKnown {
			continue
		}
		result = append(result, technicalRecordSeed{
			ID:          id,
			Description: *copyLegacyDescription(description),
			GameLimits: gameLimitsSeed{
				MaxInventory:   limits.MaxInventory,
				MaxStorage:     limits.MaxStorage,
				InventoryKnown: true,
				StorageKnown:   true,
			},
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}
