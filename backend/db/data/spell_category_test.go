package data

import "testing"

// TestSpellCategories_RainOfFireIsIncantation pins the Category-level fix for
// "Rain of Fire" (0x401EA302): it is an incantation of Salza, not a sorcery, so
// it must live in Incantations only. The cross-map exclusivity check guards the
// general invariant that a spell ID can never be owned by both spell maps.
func TestSpellCategories_RainOfFireIsIncantation(t *testing.T) {
	const rainOfFire = 0x401EA302

	if _, ok := Sorceries[rainOfFire]; ok {
		t.Errorf("Rain of Fire (%#x) must not be in Sorceries", rainOfFire)
	}
	item, ok := Incantations[rainOfFire]
	if !ok {
		t.Fatalf("Rain of Fire (%#x) missing from Incantations", rainOfFire)
	}
	if item.Name != "Rain of Fire" {
		t.Errorf("Incantations[%#x].Name = %q, want %q", rainOfFire, item.Name, "Rain of Fire")
	}
	if item.Category != "incantations" {
		t.Errorf("Rain of Fire Category = %q, want %q", item.Category, "incantations")
	}
	if !hasFlag(item.Flags, "dlc") {
		t.Errorf("Rain of Fire Flags = %v, want the %q flag preserved", item.Flags, "dlc")
	}

	// Adjacent positive cases: the neighbouring entries in both maps stay put.
	const nightMaidensMist = 0x40001964
	if sc, ok := Sorceries[nightMaidensMist]; !ok {
		t.Errorf("Night Maiden's Mist (%#x) missing from Sorceries", nightMaidensMist)
	} else if sc.Category != "sorceries" {
		t.Errorf("Night Maiden's Mist Category = %q, want %q", sc.Category, "sorceries")
	}
	const furiousBlade = 0x401E9D1C
	if inc, ok := Incantations[furiousBlade]; !ok {
		t.Errorf("Furious Blade of Ansbach (%#x) missing from Incantations", furiousBlade)
	} else if inc.Category != "incantations" {
		t.Errorf("Furious Blade of Ansbach Category = %q, want %q", inc.Category, "incantations")
	}
}

// TestSpellCategories_NoOverlap enforces the general invariant behind the fix:
// no item ID may be owned by both Sorceries and Incantations.
func TestSpellCategories_NoOverlap(t *testing.T) {
	for id, sorcery := range Sorceries {
		if inc, dup := Incantations[id]; dup {
			t.Errorf("item %#x is in both maps: Sorceries %q / Incantations %q", id, sorcery.Name, inc.Name)
		}
	}
}
