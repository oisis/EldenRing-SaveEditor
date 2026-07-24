package db

import (
	"sort"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

// TestGetPhysickEligibleItems locks the Wondrous Physick slot policy: only
// Crystal Tears (data.SubcatKeyCrystalTears, base + DLC) qualify, name-sorted.
func TestGetPhysickEligibleItems(t *testing.T) {
	items := GetPhysickEligibleItems("PS4")

	byID := make(map[uint32]bool, len(items))
	for _, it := range items {
		byID[it.ID] = true
	}

	included := []struct {
		id   uint32
		name string
	}{
		{0x40002AF8, "Crimsonspill Crystal Tear (base)"},
		{0x401EAFBE, "Deflecting Hardtear (DLC)"},
		{0x401EAF82, "Crimsonburst Dried Tear (DLC)"},
	}
	for _, c := range included {
		if !byID[c.id] {
			t.Errorf("expected %s (0x%08X) to be physick-eligible, missing from result", c.name, c.id)
		}
	}

	excluded := []struct {
		id   uint32
		name string
	}{
		{0x40001FF9, "Larval Tear (key item)"},
		{0x40002724, "Sacred Tear (bolstering material)"},
		{0x40000852, "Celestial Dew (key item)"},
		{0x400003E9, "Flask of Crimson Tears (tools)"},
		{0x00F58390, "Bolt of Gransax (weapon)"},
		{0x80002710, "Lion's Claw (Ash of War)"},
	}
	for _, c := range excluded {
		if byID[c.id] {
			t.Errorf("expected %s (0x%08X) to be excluded, but it is physick-eligible", c.name, c.id)
		}
	}

	// Every returned entry must be a key_items Crystal Tear, and only those.
	for _, it := range items {
		if it.Category != "key_items" {
			t.Errorf("unexpected category %q for item %q (0x%08X)", it.Category, it.Name, it.ID)
		}
		if it.SubCategory != data.SubcatKeyCrystalTears {
			t.Errorf("unexpected sub-category %q for item %q (0x%08X)", it.SubCategory, it.Name, it.ID)
		}
	}

	if !sort.SliceIsSorted(items, func(i, j int) bool { return items[i].Name < items[j].Name }) {
		t.Error("result is not sorted by name")
	}
}
