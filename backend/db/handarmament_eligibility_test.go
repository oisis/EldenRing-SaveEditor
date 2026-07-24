package db

import (
	"slices"
	"sort"
	"testing"
)

// TestGetHandArmamentEligibleItems locks the right/left-hand armament slot
// policy: only melee_armaments, ranged_and_catalysts and shields qualify, and
// the result is name-sorted. Both hands share this single policy.
func TestGetHandArmamentEligibleItems(t *testing.T) {
	items := GetHandArmamentEligibleItems("PS4")

	byID := make(map[uint32]bool, len(items))
	for _, it := range items {
		byID[it.ID] = true
	}

	included := []struct {
		id   uint32
		name string
	}{
		{0x003085E0, "Claymore (melee)"},
		{0x02817AC0, "Greatbow (ranged)"},
		{0x01F98610, "Astrologer's Staff (staff)"},
		{0x0206CC80, "Finger Seal (Sacred Seal)"},
		{0x01DB0190, "Brass Shield (shield)"},
		{0x016E3600, "Torch (torch)"},
	}
	for _, c := range included {
		if !byID[c.id] {
			t.Errorf("expected %s (0x%08X) to be hand-armament-eligible, missing from result", c.name, c.id)
		}
	}

	excluded := []struct {
		id   uint32
		name string
	}{
		{0x02FAF080, "Arrow (arrow)"},
		{0x400003E9, "Flask of Crimson Tears (tools)"},
		{0x200003E8, "Crimson Amber Medallion (talisman)"},
		{0x80002710, "Lion's Claw (Ash of War)"},
	}
	for _, c := range excluded {
		if byID[c.id] {
			t.Errorf("expected %s (0x%08X) to be excluded, but it is hand-armament-eligible", c.name, c.id)
		}
	}

	// Every returned entry must belong to an eligible category, and only those.
	eligible := []string{"melee_armaments", "ranged_and_catalysts", "shields"}
	for _, it := range items {
		if !slices.Contains(eligible, it.Category) {
			t.Errorf("unexpected category %q for item %q (0x%08X)", it.Category, it.Name, it.ID)
		}
	}

	if !sort.SliceIsSorted(items, func(i, j int) bool { return items[i].Name < items[j].Name }) {
		t.Error("result is not sorted by name")
	}
}
