package db

import (
	"sort"
	"testing"
)

// Knight Armor set — one piece per equipment slot (row 3). Each ID belongs to
// exactly one armor category, so it must appear in its own slot filter and be
// excluded from the other three.
const (
	knightHelm      = uint32(0x1016E360) // head
	knightArmor     = uint32(0x1016E3C4) // chest
	knightGauntlets = uint32(0x1016E428) // arms
	knightGreaves   = uint32(0x1016E48C) // legs
)

// assertSlotFilter checks that items contains wantID, excludes each of otherIDs,
// that every entry carries wantCategory, and that the result is name-sorted.
func assertSlotFilter(t *testing.T, items []ItemEntry, wantID uint32, wantCategory string, otherIDs []uint32) {
	t.Helper()

	byID := make(map[uint32]bool, len(items))
	for _, it := range items {
		byID[it.ID] = true
	}

	if !byID[wantID] {
		t.Errorf("expected item 0x%08X to be %s-slot-eligible, missing from result", wantID, wantCategory)
	}
	for _, id := range otherIDs {
		if byID[id] {
			t.Errorf("expected item 0x%08X to be excluded from %s slot, but it is present", id, wantCategory)
		}
	}

	for _, it := range items {
		if it.Category != wantCategory {
			t.Errorf("unexpected category %q for item %q (0x%08X), want %q", it.Category, it.Name, it.ID, wantCategory)
		}
	}

	if !sort.SliceIsSorted(items, func(i, j int) bool { return items[i].Name < items[j].Name }) {
		t.Error("result is not sorted by name")
	}
}

func TestGetHeadSlotEligibleItems(t *testing.T) {
	assertSlotFilter(t, GetHeadSlotEligibleItems("PS4"), knightHelm, "head",
		[]uint32{knightArmor, knightGauntlets, knightGreaves})
}

func TestGetChestSlotEligibleItems(t *testing.T) {
	assertSlotFilter(t, GetChestSlotEligibleItems("PS4"), knightArmor, "chest",
		[]uint32{knightHelm, knightGauntlets, knightGreaves})
}

func TestGetArmsSlotEligibleItems(t *testing.T) {
	assertSlotFilter(t, GetArmsSlotEligibleItems("PS4"), knightGauntlets, "arms",
		[]uint32{knightHelm, knightArmor, knightGreaves})
}

func TestGetLegsSlotEligibleItems(t *testing.T) {
	assertSlotFilter(t, GetLegsSlotEligibleItems("PS4"), knightGreaves, "legs",
		[]uint32{knightHelm, knightArmor, knightGauntlets})
}
