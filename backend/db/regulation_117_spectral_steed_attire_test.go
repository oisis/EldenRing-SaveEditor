package db

import (
	"os"
	"path/filepath"
	"testing"

	"slices"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

// TestRegulation117_SpectralSteedAttire pins the three Regulation 1.17 Torrent
// skin goods: they stay out of the Item Database catalog on every platform, but
// keep the full record the World tab needs (name, caps, icon, generated text).
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

	catalog := map[string]map[uint32]int{}
	for _, platform := range []string{"PC", "PS4"} {
		byID := map[uint32]int{}
		for _, e := range GetItemsByCategory("key_items", platform) {
			byID[e.ID]++
		}
		catalog[platform] = byID
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Not offered through the Item Database — the World tab owns these.
			for _, platform := range []string{"PC", "PS4"} {
				if n := catalog[platform][tc.id]; n != 0 {
					t.Errorf("GetItemsByCategory(key_items, %s)[0x%08X]: %d entries, want 0", platform, tc.id, n)
				}
			}

			item := GetItemData(tc.id)
			if item.Name != tc.name {
				t.Errorf("GetItemData Name = %q, want %q", item.Name, tc.name)
			}
			if item.Category != "key_items" {
				t.Errorf("GetItemData Category = %q, want key_items", item.Category)
			}
			if item.MaxInventory != 1 || item.MaxStorage != 0 || item.MaxUpgrade != 0 {
				t.Errorf("caps = inv %d/storage %d/upgrade %d, want 1/0/0", item.MaxInventory, item.MaxStorage, item.MaxUpgrade)
			}
			if !slices.Contains(item.Flags, "dlc") || !slices.Contains(item.Flags, "no_database") {
				t.Errorf("Flags = %v, want to contain \"dlc\" and \"no_database\"", item.Flags)
			}
			if item.IconPath != tc.icon {
				t.Errorf("IconPath = %q, want %q", item.IconPath, tc.icon)
			}

			// Hiding the item from the catalog must not weaken the enriched
			// entry the rest of the app resolves by ID.
			entry := GetItemEntryByID(tc.id)
			if entry == nil {
				t.Fatalf("GetItemEntryByID(0x%08X) = nil, want the enriched entry", tc.id)
			}
			// Standard add path: not routed through a World-tab unlock endpoint.
			if entry.UnlockCategory != "" || entry.FlagID != 0 {
				t.Errorf("UnlockCategory/FlagID = %q/%d, want \"\"/0", entry.UnlockCategory, entry.FlagID)
			}
			if entry.RecordMode != ItemRecordModeQuantityStack {
				t.Errorf("RecordMode = %q, want %q", entry.RecordMode, ItemRecordModeQuantityStack)
			}
			if entry.GameMaxInventory != 1 || entry.GameMaxStorage != 1 || !entry.GameMaxInventoryKnown || !entry.GameMaxStorageKnown {
				t.Errorf("game limits = inv %d(known %t)/storage %d(known %t), want 1(true)/1(true)",
					entry.GameMaxInventory, entry.GameMaxInventoryKnown, entry.GameMaxStorage, entry.GameMaxStorageKnown)
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

			icon := filepath.Join("..", "..", "frontend", "public", item.IconPath)
			if _, err := os.Stat(icon); err != nil {
				t.Errorf("icon %s: %v", icon, err)
			}
		})
	}

	// The World tab still sees all four appearances, icons included.
	entries := GetAllSpectralSteedAttires()
	if len(entries) != 4 {
		t.Fatalf("GetAllSpectralSteedAttires: %d entries, want 4", len(entries))
	}
	for _, e := range entries {
		if e.ItemID == 0 {
			continue
		}
		if !slices.ContainsFunc(cases, func(c struct {
			id   uint32
			name string
			icon string
		}) bool {
			return c.id == e.ItemID && c.name == e.Name && c.icon == e.IconPath
		}) {
			t.Errorf("World entry %+v does not match a catalog-hidden attire record", e)
		}
	}
}
