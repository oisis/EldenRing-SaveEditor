package application

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
)

func TestTemplateSpellSlotLimit_NormalAndMoonOfNokstella(t *testing.T) {
	tests := []struct {
		name          string
		talismanSlots uint8
		moonSlotIndex int
		want          int
	}{
		{
			name:          "normal character",
			talismanSlots: 3,
			moonSlotIndex: -1,
			want:          standardSpellSlotLimit,
		},
		{
			name:          "moon in unlocked pouch",
			talismanSlots: 1,
			moonSlotIndex: 1,
			want:          maxSpellSlotCount,
		},
		{
			name:          "moon in locked pouch does not count",
			talismanSlots: 0,
			moonSlotIndex: 1,
			want:          standardSpellSlotLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slot := &core.SaveSlot{}
			slot.Player.TalismanSlots = tt.talismanSlots
			raw := core.RawEquippedState{}
			if tt.moonSlotIndex >= 0 {
				raw.Equipped[17+tt.moonSlotIndex] = moonOfNokstellaItemID
			}
			if got := templateSpellSlotLimit(slot, raw); got != tt.want {
				t.Fatalf("templateSpellSlotLimit = %d, want %d", got, tt.want)
			}
		})
	}
}
