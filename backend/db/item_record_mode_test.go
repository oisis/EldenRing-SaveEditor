package db

import "testing"

func TestItemRecordMode(t *testing.T) {
	tests := []struct {
		name     string
		id       uint32
		category string
		want     string
	}{
		{"furlcalling finger remedy stack", 0x40000096, "tools", ItemRecordModeQuantityStack},
		{"sorcery goods stack", 0x40000FA0, "sorceries", ItemRecordModeQuantityStack},
		{"spirit ashes goods stack", 0x400318F8, "ashes", ItemRecordModeQuantityStack},
		{"ammunition stack override", 0x02FAF080, "arrows_and_bolts", ItemRecordModeQuantityStack},
		{"weapon instance", 0x003085E0, "melee_armaments", ItemRecordModeSeparateInstances},
		{"armor instance", 0x10000000, "head", ItemRecordModeSeparateInstances},
		{"talisman instance", 0x20000000, "talismans", ItemRecordModeSeparateInstances},
		{"ash of war instance", 0x80002710, "ashes_of_war", ItemRecordModeSeparateInstances},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ItemRecordMode(tt.id, tt.category); got != tt.want {
				t.Fatalf("ItemRecordMode(0x%08X, %q) = %q, want %q", tt.id, tt.category, got, tt.want)
			}
		})
	}
}
