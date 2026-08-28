package db

import (
	"os"
	"path/filepath"
	"testing"

	"slices"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

// TestRegulation117_SpectralSteedAttire pins the three Regulation 1.17 Torrent
// skin goods as regular Key Items: catalog visibility, conservative caps, the
// standard AddItemsToCharacter path (no unlock flag) and generated text.
func TestRegulation117_SpectralSteedAttire(t *testing.T) {
	cases := []struct {
		id   uint32
		name string
		icon string
	}{
		{0x401EAA00, "Tree Sentinel Spectral Steed Attire", "items/key_items/tree_sentinel_spectral_steed_attire.png"},
		{0x401EAA0A, "Silver of Caria Spectral Steed Attire", "items/key_items/silver_of_caria_spectral_steed_attire.png"},
		{0x401EAA14, "Funereal Night Spectral Steed Attire", "items/key_items/funereal_night_spectral_steed_attire.png"},
	}

	entries := GetItemsByCategory("key_items", "PC")
	byID := map[uint32][]ItemEntry{}
	for _, e := range entries {
		byID[e.ID] = append(byID[e.ID], e)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := byID[tc.id]
			if len(got) != 1 {
				t.Fatalf("GetItemsByCategory(key_items)[0x%08X]: %d entries, want exactly 1", tc.id, len(got))
			}
			e := got[0]
			if e.Name != tc.name {
				t.Errorf("Name = %q, want %q", e.Name, tc.name)
			}
			if e.Category != "key_items" {
				t.Errorf("Category = %q, want key_items", e.Category)
			}
			if e.MaxInventory != 1 || e.MaxStorage != 0 || e.MaxUpgrade != 0 {
				t.Errorf("caps = inv %d/storage %d/upgrade %d, want 1/0/0", e.MaxInventory, e.MaxStorage, e.MaxUpgrade)
			}
			if !slices.Contains(e.Flags, "dlc") {
				t.Errorf("Flags = %v, want to contain \"dlc\"", e.Flags)
			}
			if e.IconPath != tc.icon {
				t.Errorf("IconPath = %q, want %q", e.IconPath, tc.icon)
			}
			// Standard add path: not routed through a World-tab unlock endpoint.
			if e.UnlockCategory != "" || e.FlagID != 0 {
				t.Errorf("UnlockCategory/FlagID = %q/%d, want \"\"/0", e.UnlockCategory, e.FlagID)
			}
			if e.RecordMode != ItemRecordModeQuantityStack {
				t.Errorf("RecordMode = %q, want %q", e.RecordMode, ItemRecordModeQuantityStack)
			}
			if e.GameMaxInventory != 1 || e.GameMaxStorage != 1 || !e.GameMaxInventoryKnown || !e.GameMaxStorageKnown {
				t.Errorf("game limits = inv %d(known %t)/storage %d(known %t), want 1(true)/1(true)",
					e.GameMaxInventory, e.GameMaxInventoryKnown, e.GameMaxStorage, e.GameMaxStorageKnown)
			}

			if item := GetItemData(tc.id); item.Name != tc.name || item.Category != "key_items" {
				t.Errorf("GetItemData = %q/%q, want %q/key_items", item.Name, item.Category, tc.name)
			}

			text, ok := data.ItemTexts[tc.id]
			if !ok {
				t.Fatalf("ItemTexts[0x%08X]: missing entry", tc.id)
			}
			if text.DisplayName != tc.name {
				t.Errorf("ItemTexts DisplayName = %q, want %q", text.DisplayName, tc.name)
			}
			if text.DisplayNameSource != data.TextSourceApp {
				t.Errorf("ItemTexts DisplayNameSource = %q, want %q", text.DisplayNameSource, data.TextSourceApp)
			}

			icon := filepath.Join("..", "..", "frontend", "public", e.IconPath)
			if _, err := os.Stat(icon); err != nil {
				t.Errorf("icon %s: %v", icon, err)
			}
		})
	}
}
