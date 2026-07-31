package migration

import (
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

func collectLegacyGestureSlots() []gestureSlotSeed {
	result := make([]gestureSlotSeed, 0, len(data.AllGestures))
	for _, gesture := range data.AllGestures {
		result = append(result, gestureSlotSeed{
			SlotID: gesture.ID, ItemID: gesture.ItemID,
			Name: gesture.Name, Category: gesture.Category,
			Flags: cloneStrings(gesture.Flags),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].SlotID < result[j].SlotID
	})
	return result
}
