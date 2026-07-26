package db

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

// TestSpellMemorySlots_CoversEverySpell asserts the generated SpellMemorySlots
// map is complete: every sorcery and incantation has a known cost in [1,3]. This
// is what lets SaveEquippedSpells reject an unknown cost as a genuine fault
// rather than a routine gap — if this fails, the fail-closed guard would start
// blocking real loadouts.
func TestSpellMemorySlots_CoversEverySpell(t *testing.T) {
	known := make(map[uint32]struct{}, len(data.Sorceries)+len(data.Incantations))
	for _, group := range []struct {
		name  string
		items map[uint32]data.ItemData
	}{
		{"sorceries", data.Sorceries},
		{"incantations", data.Incantations},
	} {
		for id, item := range group.items {
			known[id] = struct{}{}
			cost, ok := SpellMemorySlotCost(id)
			if !ok {
				t.Errorf("%s: no Memory Slot cost for %q (0x%08X)", group.name, item.Name, id)
				continue
			}
			if cost < 1 || cost > 3 {
				t.Errorf("%s: %q (0x%08X) cost = %d, want 1-3", group.name, item.Name, id, cost)
			}
		}
	}
	if len(data.SpellMemorySlots) != len(known) {
		t.Errorf("SpellMemorySlots has %d entries, want exactly %d known spells", len(data.SpellMemorySlots), len(known))
	}
	for id := range data.SpellMemorySlots {
		if _, ok := known[id]; !ok {
			t.Errorf("SpellMemorySlots contains non-database spell ID 0x%08X", id)
		}
	}
}

// TestSpellMemorySlots_AgreeWithDescriptions cross-checks the generated cost
// against the legacy Descriptions[id].Spell.Slots for every base spell that has
// one. The DLC spells absent from Descriptions are precisely why the generated
// map exists, so a missing description is not an error here.
func TestSpellMemorySlots_AgreeWithDescriptions(t *testing.T) {
	checked := 0
	for _, group := range []map[uint32]data.ItemData{data.Sorceries, data.Incantations} {
		for id := range group {
			desc, ok := data.Descriptions[id]
			if !ok || desc.Spell == nil || desc.Spell.Slots == 0 {
				continue
			}
			cost, ok := SpellMemorySlotCost(id)
			if !ok {
				t.Errorf("0x%08X: has Descriptions.Spell.Slots but no generated cost", id)
				continue
			}
			if cost != desc.Spell.Slots {
				t.Errorf("0x%08X: generated cost %d != Descriptions.Spell.Slots %d", id, cost, desc.Spell.Slots)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("cross-checked no base spells; Descriptions data missing?")
	}
}

// TestSpellMemorySlots_DLCCosts pins the two DLC spells from the report to their
// Magic.csv slotLength values (which Descriptions does not carry).
func TestSpellMemorySlots_DLCCosts(t *testing.T) {
	for _, tc := range []struct {
		id   uint32
		want uint32
		name string
	}{
		{0x401E96DC, 2, "Blades of Stone"},
		{0x401E9E7A, 1, "Aspects of the Crucible: Thorns"},
	} {
		cost, ok := SpellMemorySlotCost(tc.id)
		if !ok {
			t.Errorf("%s (0x%08X): missing cost", tc.name, tc.id)
			continue
		}
		if cost != tc.want {
			t.Errorf("%s (0x%08X): cost = %d, want %d", tc.name, tc.id, cost, tc.want)
		}
	}
}

// TestSpellMemorySlotCost_UnknownFailsClosed confirms the helper reports ok=false
// for a non-spell ID instead of inventing a cost.
func TestSpellMemorySlotCost_UnknownFailsClosed(t *testing.T) {
	if _, ok := SpellMemorySlotCost(0x00000000); ok {
		t.Error("expected ok=false for a non-spell ID")
	}
	if _, ok := SpellMemorySlotCost(0x40000FA0); !ok {
		t.Error("expected ok=true for Glintstone Pebble")
	}
}
