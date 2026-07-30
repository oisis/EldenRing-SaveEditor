package data

import "testing"

func TestAlteredCutContentArmorFlags(t *testing.T) {
	tests := []struct {
		name   string
		itemID uint32
		items  map[uint32]ItemData
	}{
		{name: "Ragged Hat (Altered)", itemID: 0x100952B8, items: Helms},
		{name: "Ragged Armor (Altered)", itemID: 0x1009531C, items: Chest},
		{name: "Brave's Battlewear (Altered)", itemID: 0x100AB630, items: Chest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item, ok := test.items[test.itemID]
			if !ok {
				t.Fatalf("item 0x%X is missing from the database", test.itemID)
			}

			for _, flag := range []string{"cut_content", "ban_risk"} {
				if !hasFlag(item.Flags, flag) {
					t.Errorf("item 0x%X (%q) is missing %q flag", test.itemID, item.Name, flag)
				}
			}
		})
	}
}
