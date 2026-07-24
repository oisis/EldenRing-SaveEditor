package db

import (
	"sort"
	"testing"
)

// TestGetPouchEligibleItems locks the Quick pouch eligibility policy: only
// "tools" and "ashes" DB items qualify, and the result is name-sorted.
func TestGetPouchEligibleItems(t *testing.T) {
	items := GetPouchEligibleItems("PS4")

	byID := make(map[uint32]bool, len(items))
	for _, it := range items {
		byID[it.ID] = true
	}

	included := []struct {
		id   uint32
		name string
	}{
		{0x400003E9, "Flask of Crimson Tears (tools)"},
		{0x40000067, "Finger Severer (tools)"},
		{0x400318F8, "Fanged Imp Ashes (ashes)"},
	}
	for _, c := range included {
		if !byID[c.id] {
			t.Errorf("expected %s (0x%08X) to be pouch-eligible, missing from result", c.name, c.id)
		}
	}

	excluded := []struct {
		id   uint32
		name string
	}{
		{0x00F58390, "Bolt of Gransax (weapon)"},
		{0x200003E8, "Crimson Amber Medallion (talisman)"},
		{0x02FAF080, "Arrow (arrow)"},
		{0x40000FA0, "Glintstone Pebble (sorcery)"},
		{0x40000852, "Celestial Dew (key item)"},
		{0x80002710, "Lion's Claw (Ash of War)"},
	}
	for _, c := range excluded {
		if byID[c.id] {
			t.Errorf("expected %s (0x%08X) to be excluded, but it is pouch-eligible", c.name, c.id)
		}
	}

	// Every returned entry must belong to an eligible category, and only those.
	for _, it := range items {
		if it.Category != "tools" && it.Category != "ashes" {
			t.Errorf("unexpected category %q for item %q (0x%08X)", it.Category, it.Name, it.ID)
		}
	}

	if !sort.SliceIsSorted(items, func(i, j int) bool { return items[i].Name < items[j].Name }) {
		t.Error("result is not sorted by name")
	}
}
