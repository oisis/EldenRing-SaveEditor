package data

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// Regulation 1.17 added four armor sets. These tests pin the full shape of
// that addition: each piece lives in exactly one category map, carries the
// DLC flag and the category placeholder icon, and is covered by the generated
// weight, sort-key and item-text tables.
//
// Altered variants are deliberately out of scope.
var regulation117Armor = []struct {
	id       uint32
	name     string
	category string
	items    map[uint32]ItemData
}{
	{0x1051A270, "Silver Grooved Helm", "head", Helms},
	{0x1051A2D4, "Silver Grooved Armor", "chest", Chest},
	{0x1051A338, "Silver Grooved Gauntlets", "arms", Arms},
	{0x1051A39C, "Silver Grooved Greaves", "legs", Legs},

	{0x1051F090, "Steel Helm", "head", Helms},
	{0x1051F0F4, "Steel Armor", "chest", Chest},
	{0x1051F158, "Steel Gauntlets", "arms", Arms},
	{0x1051F1BC, "Steel Greaves", "legs", Legs},

	{0x1051C980, "Leontiel's Hat", "head", Helms},
	{0x1051C9E4, "Leontiel's Armor", "chest", Chest},
	{0x1051CA48, "Leontiel's Leather Gloves", "arms", Arms},
	{0x1051CAAC, "Leontiel's Boots", "legs", Legs},

	{0x10517B60, "Broken Gold Mask", "head", Helms},
	{0x10517BC4, "Gold Tattoo (Chest)", "chest", Chest},
	{0x10517C28, "Gold Tattoo (Arm)", "arms", Arms},
	{0x10517C8C, "Gold Tattoo (Leg)", "legs", Legs},
}

// allArmorMaps is used to prove each ID appears in exactly one category map.
var allArmorMaps = map[string]map[uint32]ItemData{
	"head":  Helms,
	"chest": Chest,
	"arms":  Arms,
	"legs":  Legs,
}

func TestRegulation117ArmorRecords(t *testing.T) {
	for _, want := range regulation117Armor {
		item, ok := want.items[want.id]
		if !ok {
			t.Errorf("%#x (%q): missing from the %s map", want.id, want.name, want.category)
			continue
		}
		if item.Name != want.name {
			t.Errorf("%#x: Name = %q, want %q", want.id, item.Name, want.name)
		}
		if item.Category != want.category {
			t.Errorf("%#x (%q): Category = %q, want %q", want.id, want.name, item.Category, want.category)
		}
		if item.MaxInventory != 1 {
			t.Errorf("%#x (%q): MaxInventory = %d, want 1", want.id, want.name, item.MaxInventory)
		}
		if item.MaxStorage != 1 {
			t.Errorf("%#x (%q): MaxStorage = %d, want 1", want.id, want.name, item.MaxStorage)
		}
		if item.MaxUpgrade != 0 {
			t.Errorf("%#x (%q): MaxUpgrade = %d, want 0", want.id, want.name, item.MaxUpgrade)
		}
		if !hasFlag(item.Flags, "dlc") {
			t.Errorf("%#x (%q): Flags = %v, want the dlc flag", want.id, want.name, item.Flags)
		}

		wantIcon := "items/" + want.category + "/missing_icon.png"
		if item.IconPath != wantIcon {
			t.Errorf("%#x (%q): IconPath = %q, want %q", want.id, want.name, item.IconPath, wantIcon)
		}
		iconFile := filepath.Join("../../../frontend/public", filepath.FromSlash(wantIcon))
		if _, err := os.Stat(iconFile); err != nil {
			t.Errorf("%#x (%q): placeholder missing on disk: %s", want.id, want.name, iconFile)
		}
	}
}

// TestRegulation117ArmorSingleCategory proves no ID leaked into a second map.
func TestRegulation117ArmorSingleCategory(t *testing.T) {
	for _, want := range regulation117Armor {
		for category, m := range allArmorMaps {
			_, present := m[want.id]
			if present && category != want.category {
				t.Errorf("%#x (%q): also present in the %s map", want.id, want.name, category)
			}
			if !present && category == want.category {
				t.Errorf("%#x (%q): absent from its own %s map", want.id, want.name, category)
			}
		}
	}
}

// TestRegulation117ArmorGeneratedTables covers the tables the armor records
// depend on but do not own: weights, sort keys and item texts.
func TestRegulation117ArmorGeneratedTables(t *testing.T) {
	for _, want := range regulation117Armor {
		if _, ok := ItemWeights[want.id]; !ok {
			t.Errorf("%#x (%q): missing from ItemWeights", want.id, want.name)
		}
		if _, ok := ItemSortKeys[want.id]; !ok {
			t.Errorf("%#x (%q): missing from ItemSortKeys", want.id, want.name)
		}
		text, ok := ItemTexts[want.id]
		if !ok {
			t.Errorf("%#x (%q): missing from ItemTexts", want.id, want.name)
			continue
		}
		if text.DisplayName != want.name {
			t.Errorf("%#x: ItemTexts DisplayName = %q, want %q", want.id, text.DisplayName, want.name)
		}
	}
}

// TestRegulation117ArmorWeightAnchors pins a few Regulation 1.17 weights so a
// bad weights regeneration cannot silently shift the whole table.
func TestRegulation117ArmorWeightAnchors(t *testing.T) {
	anchors := []struct {
		id   uint32
		name string
		want float64
	}{
		{0x10517B60, "Broken Gold Mask", 2.0},
		{0x1051A2D4, "Silver Grooved Armor", 7.1},
		{0x1051C9E4, "Leontiel's Armor", 8.3},
		{0x1051F0F4, "Steel Armor", 18.5},
	}
	for _, a := range anchors {
		got, ok := ItemWeights[a.id]
		if !ok {
			t.Errorf("%#x (%q): missing from ItemWeights", a.id, a.name)
			continue
		}
		if math.Abs(got-a.want) > 1e-4 {
			t.Errorf("%#x (%q): weight = %v, want %v", a.id, a.name, got, a.want)
		}
	}
}
