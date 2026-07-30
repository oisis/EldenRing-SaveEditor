package data

import (
	"slices"
	"testing"
)

func TestUnobtainableItemsAreCutContent(t *testing.T) {
	tests := []struct {
		name   string
		itemID uint32
		items  map[uint32]ItemData
	}{
		{name: "Erdtree Prayerbook", itemID: 0x4000229C, items: KeyItems},
		{name: "The Carian Oath", itemID: 0x40002341, items: Gestures},
		{name: "Fetal Position", itemID: 0x4000234E, items: Gestures},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item, ok := test.items[test.itemID]
			if !ok {
				t.Fatalf("item 0x%08X is missing from the database", test.itemID)
			}
			if !slices.Contains(item.Flags, "cut_content") {
				t.Errorf("item 0x%08X (%q) is missing cut_content flag", test.itemID, item.Name)
			}
		})
	}
}
