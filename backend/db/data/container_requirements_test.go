package data

import (
	"testing"
)

// TestRequiredContainer_Consistency verifies that every gated item:
// 1. Exists in the Tools DB (no orphan IDs in the map).
// 2. Maps to one of the four known container key item IDs.
func TestRequiredContainer_Consistency(t *testing.T) {
	validContainers := map[uint32]bool{
		CrackedPotKeyItemID:      true,
		RitualPotKeyItemID:       true,
		PerfumeBottleKeyItemID:   true,
		HeftyCrackedPotKeyItemID: true,
	}

	for itemID, cID := range RequiredContainer {
		if !validContainers[cID] {
			t.Errorf("item 0x%X maps to unknown container 0x%X", itemID, cID)
		}
		if _, ok := Tools[itemID]; !ok {
			t.Errorf("item 0x%X in RequiredContainer not found in Tools DB", itemID)
		}
	}
}

// TestRequiredContainer_KeyItemsExist verifies all four containers have entries
// in the KeyItems DB with sensible MaxInventory values.
func TestRequiredContainer_KeyItemsExist(t *testing.T) {
	containers := []struct {
		id       uint32
		name     string
		expected uint32
	}{
		{CrackedPotKeyItemID, "Cracked Pot", 20},
		{RitualPotKeyItemID, "Ritual Pot", 10},
		{PerfumeBottleKeyItemID, "Perfume Bottle", 10},
		{HeftyCrackedPotKeyItemID, "Hefty Cracked Pot", 10},
	}

	for _, c := range containers {
		entry, ok := KeyItems[c.id]
		if !ok {
			t.Errorf("container %s (0x%X) missing from KeyItems DB", c.name, c.id)
			continue
		}
		if entry.MaxInventory != c.expected {
			t.Errorf("container %s: MaxInventory = %d, expected %d", c.name, entry.MaxInventory, c.expected)
		}
	}
}

// capLookup returns each container's cap from the KeyItems DB.
func capLookup(c uint32) int { return int(KeyItems[c].MaxInventory) }

// TestApplyContainerCap_FirstAddFits verifies an empty inventory accepts a
// pot stack up to the container cap.
func TestApplyContainerCap_FirstAddFits(t *testing.T) {
	const firePot uint32 = 0x4000012C // Cracked Pot
	itemQty := map[uint32]int{}
	containerQty := map[uint32]int{}

	d := ApplyContainerCap(firePot, 20, itemQty, containerQty, capLookup)
	if d.EffectiveQty != 20 || d.CutQty != 0 {
		t.Errorf("first add: got effective=%d cut=%d, want effective=20 cut=0", d.EffectiveQty, d.CutQty)
	}
	if itemQty[firePot] != 20 || containerQty[CrackedPotKeyItemID] != 20 {
		t.Errorf("state: itemQty=%d containerQty=%d, want 20/20", itemQty[firePot], containerQty[CrackedPotKeyItemID])
	}
}

// TestApplyContainerCap_PartialCutAcrossBatch reproduces user point #3:
// 2 types × qty 12 = 24 → cut 4 → final total 20.
func TestApplyContainerCap_PartialCutAcrossBatch(t *testing.T) {
	const firePot uint32 = 0x4000012C
	const lightningPot uint32 = 0x40000140
	itemQty := map[uint32]int{}
	containerQty := map[uint32]int{}

	// First item: Fire Pot qty 12 → fits (0 + 12 ≤ 20).
	d1 := ApplyContainerCap(firePot, 12, itemQty, containerQty, capLookup)
	if d1.EffectiveQty != 12 || d1.CutQty != 0 {
		t.Errorf("Fire Pot: got effective=%d cut=%d, want 12/0", d1.EffectiveQty, d1.CutQty)
	}

	// Second item: Lightning Pot qty 12 → would total 24, cap 20 → cut 4 → effective 8.
	d2 := ApplyContainerCap(lightningPot, 12, itemQty, containerQty, capLookup)
	if d2.EffectiveQty != 8 || d2.CutQty != 4 {
		t.Errorf("Lightning Pot: got effective=%d cut=%d, want 8/4", d2.EffectiveQty, d2.CutQty)
	}

	if got := containerQty[CrackedPotKeyItemID]; got != 20 {
		t.Errorf("container total: %d, want 20 (capped)", got)
	}
}

// TestApplyContainerCap_RitualVsCrackedSeparate ensures different containers
// have independent caps.
func TestApplyContainerCap_RitualVsCrackedSeparate(t *testing.T) {
	const firePot uint32 = 0x4000012C        // Cracked
	const redmaneFirePot uint32 = 0x4000012D // Ritual
	itemQty := map[uint32]int{}
	containerQty := map[uint32]int{}

	// Fill Cracked Pot to cap.
	ApplyContainerCap(firePot, 20, itemQty, containerQty, capLookup)
	if containerQty[CrackedPotKeyItemID] != 20 {
		t.Fatalf("preload Cracked: %d, want 20", containerQty[CrackedPotKeyItemID])
	}

	// Add Ritual Pot — should fit independently (cap 10, 0 + 10 ≤ 10).
	d := ApplyContainerCap(redmaneFirePot, 10, itemQty, containerQty, capLookup)
	if d.EffectiveQty != 10 || d.CutQty != 0 {
		t.Errorf("Ritual independent of Cracked: got %d/%d, want 10/0", d.EffectiveQty, d.CutQty)
	}
	if containerQty[RitualPotKeyItemID] != 10 {
		t.Errorf("Ritual total: %d, want 10", containerQty[RitualPotKeyItemID])
	}
}

// TestApplyContainerCap_ExistingStackMerge verifies SET semantics:
// existing stack 5, add target 15 → delta 10 consumes container slots.
func TestApplyContainerCap_ExistingStackMerge(t *testing.T) {
	const firePot uint32 = 0x4000012C
	itemQty := map[uint32]int{firePot: 5}
	containerQty := map[uint32]int{CrackedPotKeyItemID: 5}

	// SET Fire Pot to 15 (existing 5 → target 15, delta +10 ≤ 15 remain).
	d := ApplyContainerCap(firePot, 15, itemQty, containerQty, capLookup)
	if d.EffectiveQty != 15 || d.CutQty != 0 {
		t.Errorf("merge: got %d/%d, want 15/0", d.EffectiveQty, d.CutQty)
	}
	if containerQty[CrackedPotKeyItemID] != 15 {
		t.Errorf("container after merge: %d, want 15", containerQty[CrackedPotKeyItemID])
	}
}

// TestApplyContainerCap_NonGatedPassThrough verifies items without a
// container (e.g. flasks) are not affected.
func TestApplyContainerCap_NonGatedPassThrough(t *testing.T) {
	itemQty := map[uint32]int{}
	containerQty := map[uint32]int{}

	d := ApplyContainerCap(0x40000419, 1, itemQty, containerQty, capLookup) // Sacred Flask
	if d.EffectiveQty != 1 || d.CutQty != 0 {
		t.Errorf("non-gated: got %d/%d, want 1/0", d.EffectiveQty, d.CutQty)
	}
}

// TestApplyContainerCap_TargetEqualToExisting is a no-op (delta = 0).
func TestApplyContainerCap_TargetEqualToExisting(t *testing.T) {
	const firePot uint32 = 0x4000012C
	itemQty := map[uint32]int{firePot: 10}
	containerQty := map[uint32]int{CrackedPotKeyItemID: 10}

	d := ApplyContainerCap(firePot, 10, itemQty, containerQty, capLookup)
	if d.EffectiveQty != 10 || d.CutQty != 0 {
		t.Errorf("no-op: got %d/%d, want 10/0", d.EffectiveQty, d.CutQty)
	}
	if containerQty[CrackedPotKeyItemID] != 10 {
		t.Errorf("container unchanged: %d, want 10", containerQty[CrackedPotKeyItemID])
	}
}
