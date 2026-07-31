package dbviewer

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestFamilyFactsSupportsEveryItemFamily(t *testing.T) {
	source := schema.SourceID("test")
	provenance := schema.Provenance{Source: source, Method: "test"}
	rowID := schema.Fact[uint32]{Known: true, Value: 100, Provenance: provenance}
	server := &Server{
		sources: map[schema.SourceID]schema.DataSource{
			source: {ID: source, Location: "regulation.bin/csv/test.csv"},
		},
	}

	tests := []struct {
		family schema.ItemFamily
		item   *schema.ItemDocument
		label  string
	}{
		{schema.ItemFamilyWeapon, &schema.ItemDocument{Weapon: &schema.WeaponData{SourceRowID: rowID}}, "Source row ID"},
		{schema.ItemFamilyArmor, &schema.ItemDocument{Armor: &schema.ArmorData{SourceRowID: rowID}}, "Source row ID"},
		{schema.ItemFamilyTalisman, &schema.ItemDocument{Talisman: &schema.TalismanData{SourceRowID: rowID}}, "Source row ID"},
		{schema.ItemFamilyAshOfWar, &schema.ItemDocument{AshOfWar: &schema.AshOfWarData{SourceRowID: rowID}}, "Source row ID"},
		{schema.ItemFamilySpell, &schema.ItemDocument{Spell: &schema.SpellData{SourceRowID: rowID}}, "Source row ID"},
		{schema.ItemFamilySpiritAsh, &schema.ItemDocument{SpiritAsh: &schema.SpiritAshData{SourceRowID: rowID}}, "Source row ID"},
		{schema.ItemFamilyGoods, &schema.ItemDocument{Goods: &schema.GoodsData{SourceRowID: rowID}}, "Source row ID"},
		{schema.ItemFamilyGesture, &schema.ItemDocument{Gesture: &schema.GestureData{GoodsSourceRowID: rowID}}, "Goods source row ID"},
	}

	for _, test := range tests {
		t.Run(string(test.family), func(t *testing.T) {
			test.item.Family.Value = test.family
			facts := server.familyFacts(test.item)
			if len(facts) == 0 {
				t.Fatal("familyFacts returned no facts")
			}
			if facts[0].Label != test.label || facts[0].Value != "100" {
				t.Fatalf("first family fact = %+v", facts[0])
			}
			if facts[0].SourceLocation != "regulation.bin/csv/test.csv" {
				t.Fatalf("source location = %q", facts[0].SourceLocation)
			}
		})
	}
}

func TestFamilyFactsFormatsCompatibilityMaskAsHexadecimal(t *testing.T) {
	item := &schema.ItemDocument{
		Family: schema.Fact[schema.ItemFamily]{Value: schema.ItemFamilyAshOfWar},
		AshOfWar: &schema.AshOfWarData{
			CompatibilityMask: schema.Fact[uint64]{Known: true, Value: 0x1234},
		},
	}
	facts := (&Server{}).familyFacts(item)
	for _, fact := range facts {
		if fact.Label == "Compatibility mask" {
			if fact.Value != "0x1234" {
				t.Fatalf("compatibility mask = %q", fact.Value)
			}
			return
		}
	}
	t.Fatal("compatibility mask fact is missing")
}
