package migration

import (
	"reflect"
	"testing"
)

func TestCollectLegacySnapshotExactCounts(t *testing.T) {
	wantCategories := map[string]int{
		"melee_armaments":      439,
		"ranged_and_catalysts": 64,
		"shields":              165,
		"arrows_and_bolts":     68,
		"head":                 212,
		"chest":                252,
		"arms":                 122,
		"legs":                 137,
		"talismans":            155,
		"ashes_of_war":         116,
		"gestures":             54,
		"ashes":                924,
		"sorceries":            85,
		"incantations":         128,
		"crafting_materials":   108,
		"bolstering_materials": 43,
		"key_items":            349,
		"tools":                313,
		"info":                 102,
	}

	snapshot := collectLegacySnapshot()
	if got := len(snapshot.Items); got != 3836 {
		t.Fatalf("item count = %d, want 3836", got)
	}
	gotCategories := make(map[string]int)
	seenIDs := make(map[uint32]struct{}, len(snapshot.Items))
	for _, item := range snapshot.Items {
		gotCategories[item.Category]++
		if _, exists := seenIDs[item.ID]; exists {
			t.Fatalf("duplicate item ID 0x%08X", item.ID)
		}
		seenIDs[item.ID] = struct{}{}
	}
	if !reflect.DeepEqual(gotCategories, wantCategories) {
		t.Fatalf("category counts = %#v, want %#v", gotCategories, wantCategories)
	}
	if got := len(snapshot.Aliases); got != 37 {
		t.Fatalf("alias count = %d, want 37", got)
	}
	if got := len(snapshot.GestureSlots); got != 57 {
		t.Fatalf("gesture slot count = %d, want 57", got)
	}

	seenSlots := make(map[uint32]struct{}, len(snapshot.GestureSlots))
	for _, slot := range snapshot.GestureSlots {
		if _, exists := seenSlots[slot.SlotID]; exists {
			t.Fatalf("duplicate gesture slot ID %d", slot.SlotID)
		}
		seenSlots[slot.SlotID] = struct{}{}
	}
}

func TestCollectLegacySnapshotIsDeterministic(t *testing.T) {
	first := collectLegacySnapshot()
	second := collectLegacySnapshot()
	if !reflect.DeepEqual(first, second) {
		t.Fatal("two legacy snapshots differ")
	}
}
