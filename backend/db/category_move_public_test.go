package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

// findItemInCategory returns the entry for id inside the given category list.
func findItemInCategory(t *testing.T, category string, id uint32) (ItemEntry, bool) {
	t.Helper()
	for _, e := range GetItemsByCategory(category, "pc") {
		if e.ID == id {
			return e, true
		}
	}
	return ItemEntry{}, false
}

// TestGetItemsByCategory_CategoryMoves checks the corrected categories at the
// layer the UI actually consumes: each item must appear in exactly one tab.
func TestGetItemsByCategory_CategoryMoves(t *testing.T) {
	const rainOfFire = 0x401EA302
	if e, ok := findItemInCategory(t, "incantations", rainOfFire); !ok {
		t.Errorf("Rain of Fire (%#x) missing from GetItemsByCategory(%q)", rainOfFire, "incantations")
	} else if e.Category != "incantations" {
		t.Errorf("Rain of Fire Category = %q, want %q", e.Category, "incantations")
	}
	if _, ok := findItemInCategory(t, "sorceries", rainOfFire); ok {
		t.Errorf("Rain of Fire (%#x) must not appear in GetItemsByCategory(%q)", rainOfFire, "sorceries")
	}

	const rottenStaff = 0x016116A0
	e, ok := findItemInCategory(t, "melee_armaments", rottenStaff)
	if !ok {
		t.Fatalf("Rotten Staff (%#x) missing from GetItemsByCategory(%q)", rottenStaff, "melee_armaments")
	}
	if e.Category != "melee_armaments" {
		t.Errorf("Rotten Staff Category = %q, want %q", e.Category, "melee_armaments")
	}
	if e.SubCategory != data.SubcatMeleeColossalWeapons {
		t.Errorf("Rotten Staff SubCategory = %q, want %q", e.SubCategory, data.SubcatMeleeColossalWeapons)
	}
	if _, ok := findItemInCategory(t, "ranged_and_catalysts", rottenStaff); ok {
		t.Errorf("Rotten Staff (%#x) must not appear in GetItemsByCategory(%q)", rottenStaff, "ranged_and_catalysts")
	}
}

// TestCategoryMoveIconsRelocated pins the asset side of the same fix: the icons
// live under the corrected category directories and the stale copies are gone.
func TestCategoryMoveIconsRelocated(t *testing.T) {
	const publicRoot = "../../frontend/public"

	for _, want := range []string{
		"items/incantations/rain_of_fire.png",
		"items/melee_armaments/rotten_staff.png",
	} {
		if _, err := os.Stat(filepath.Join(publicRoot, filepath.FromSlash(want))); err != nil {
			t.Errorf("icon missing: %s (%v)", want, err)
		}
	}

	for _, stale := range []string{
		"items/sorceries/rain_of_fire.png",
		"items/ranged_and_catalysts/rotten_staff.png",
	} {
		if _, err := os.Stat(filepath.Join(publicRoot, filepath.FromSlash(stale))); err == nil {
			t.Errorf("stale icon still present: %s", stale)
		}
	}

	if got := data.Incantations[0x401EA302].IconPath; got != "items/incantations/rain_of_fire.png" {
		t.Errorf("Rain of Fire IconPath = %q, want %q", got, "items/incantations/rain_of_fire.png")
	}
	if got := data.Weapons[0x016116A0].IconPath; got != "items/melee_armaments/rotten_staff.png" {
		t.Errorf("Rotten Staff IconPath = %q, want %q", got, "items/melee_armaments/rotten_staff.png")
	}
}
