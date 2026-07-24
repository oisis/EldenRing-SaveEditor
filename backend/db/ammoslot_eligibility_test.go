package db

import (
	"slices"
	"sort"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

// assertAmmoSlotResult checks the shared invariants for an ammo-slot filter:
// required IDs present, forbidden IDs absent, every entry in "arrows_and_bolts"
// with an allowed subcategory, and name-sorted output.
func assertAmmoSlotResult(t *testing.T, items []ItemEntry, include, exclude map[uint32]string, allowedSubcats []string) {
	t.Helper()

	byID := make(map[uint32]bool, len(items))
	for _, it := range items {
		byID[it.ID] = true
	}

	for id, name := range include {
		if !byID[id] {
			t.Errorf("expected %s (0x%08X) to be eligible, missing from result", name, id)
		}
	}
	for id, name := range exclude {
		if byID[id] {
			t.Errorf("expected %s (0x%08X) to be excluded, but it is eligible", name, id)
		}
	}

	for _, it := range items {
		if it.Category != "arrows_and_bolts" {
			t.Errorf("unexpected category %q for item %q (0x%08X)", it.Category, it.Name, it.ID)
		}
		if !slices.Contains(allowedSubcats, it.SubCategory) {
			t.Errorf("unexpected subcategory %q for item %q (0x%08X)", it.SubCategory, it.Name, it.ID)
		}
	}

	if !sort.SliceIsSorted(items, func(i, j int) bool { return items[i].Name < items[j].Name }) {
		t.Error("result is not sorted by name")
	}
}

// TestGetArrowSlotEligibleItems locks the arrow-slot policy: only the Arrows and
// Greatarrows subcategories qualify.
func TestGetArrowSlotEligibleItems(t *testing.T) {
	items := GetArrowSlotEligibleItems("PS4")

	include := map[uint32]string{
		0x02FAF080: "Arrow",
		0x030A32C0: "Great Arrow",
	}
	exclude := map[uint32]string{
		0x03197500: "Bolt",
		0x0328B740: "Ballista Bolt",
		0x003085E0: "Claymore",
		0x400003E9: "Flask of Crimson Tears",
	}
	allowed := []string{data.SubcatArrowsArrows, data.SubcatArrowsGreatarrows}
	assertAmmoSlotResult(t, items, include, exclude, allowed)
}

// TestGetBoltSlotEligibleItems locks the bolt-slot policy: only the Bolts and
// Greatbolts subcategories qualify.
func TestGetBoltSlotEligibleItems(t *testing.T) {
	items := GetBoltSlotEligibleItems("PS4")

	include := map[uint32]string{
		0x03197500: "Bolt",
		0x0328B740: "Ballista Bolt",
		0x03305860: "Rabbath's Greatbolt",
	}
	exclude := map[uint32]string{
		0x02FAF080: "Arrow",
		0x030A32C0: "Great Arrow",
		0x003085E0: "Claymore",
		0x400003E9: "Flask of Crimson Tears",
	}
	allowed := []string{data.SubcatArrowsBolts, data.SubcatArrowsGreatbolts}
	assertAmmoSlotResult(t, items, include, exclude, allowed)
}
