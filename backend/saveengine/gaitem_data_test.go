package saveengine

import (
	"testing"
)

// These tests exercise the GaItemData insert through the public mutation that
// owns it, so the ordering rule is proven where a user meets it, and they read
// the active prefix back out of the snapshot byte for byte. The fixture, the
// offsets and the assertions on untouched ranges are the ones
// add_item_to_inventory_test.go states.

// addItemTestGaItemDataEntries reads the first count active entries out of a
// slot-data copy and proves that every one of them carries the flag the game
// sets.
func addItemTestGaItemDataEntries(t *testing.T, slot []byte, count int) []uint32 {
	t.Helper()

	entries := make([]uint32, count)
	for index := 0; index < count; index++ {
		at := int64(addItemTestGaItemDataArrayAt + index*addItemTestGaItemEntrySize)
		entries[index] = addItemTestUint32(slot, at)
		if flag := addItemTestUint32(slot, at+4); flag != 1 {
			t.Errorf("active entry %d carries the flag %d, want 1", index, flag)
		}
	}
	return entries
}

func TestGaItemDataInsertKeepsTheOrdinaryEntriesSorted(t *testing.T) {
	cases := []struct {
		name    string
		stored  []uint32
		added   uint32
		wantAll []uint32
	}{
		{
			name:    "empty section",
			stored:  nil,
			added:   0x40000200,
			wantAll: []uint32{0x40000200},
		},
		{
			name:    "lower bound inside the ordinary run",
			stored:  []uint32{0x40000100, 0x40000300, 0x40000400},
			added:   0x40000200,
			wantAll: []uint32{0x40000100, 0x40000200, 0x40000300, 0x40000400},
		},
		{
			name:    "behind the last ordinary entry",
			stored:  []uint32{0x40000100, 0x40000300},
			added:   0x40000500,
			wantAll: []uint32{0x40000100, 0x40000300, 0x40000500},
		},
		{
			// The Ash of War segment sits at the end of the active prefix. An
			// ordinary entry is never placed inside it and never sorted against it:
			// the whole segment is shifted right with its order and contents intact.
			name:    "in front of the ash of war segment",
			stored:  []uint32{0x40000100, 0x40000400, 0x80000009, 0x80000001},
			added:   0x40000200,
			wantAll: []uint32{0x40000100, 0x40000200, 0x40000400, 0x80000009, 0x80000001},
		},
		{
			// Only the last ascending run counts as sorted, so an unsorted legacy
			// prefix in front of it is left exactly as it is rather than repaired.
			name:    "behind an unsorted legacy prefix",
			stored:  []uint32{0x40000900, 0x40000100, 0x40000300},
			added:   0x40000200,
			wantAll: []uint32{0x40000900, 0x40000100, 0x40000200, 0x40000300},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			content := addItemTestFixture{
				platform: PlatformPC, slot: 2, tailMarker: true,
				gaItemData: testCase.stored,
			}
			engine := New()
			loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)

			if _, err := engine.AddItemToInventory(
				loaded.SaveSessionID, content.slot, testCase.added, 1, "0", false, 40, 600); err != nil {
				t.Fatalf("AddItemToInventory: %v", err)
			}

			after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)
			if count := addItemTestUint32(after, addItemTestGaItemDataAt); count != uint32(len(testCase.wantAll)) {
				t.Fatalf("the GaItemData count is %d, want %d", count, len(testCase.wantAll))
			}
			entries := addItemTestGaItemDataEntries(t, after, len(testCase.wantAll))
			for index, want := range testCase.wantAll {
				if entries[index] != want {
					t.Errorf("active entry %d is 0x%08X, want 0x%08X", index, entries[index], want)
				}
			}

			// Nothing outside the inventory record, its three counters and the two
			// GaItemData ranges may move — in particular not the second half of the
			// preallocated area behind the active capacity, and not the unknown
			// tail behind it.
			arrayEnd := int64(addItemTestGaItemDataArrayAt +
				len(testCase.wantAll)*addItemTestGaItemEntrySize)
			addItemTestAssertChanged(t, before, after, [][2]int64{
				{addItemTestCommonAt, addItemTestCommonAt + addItemTestRecordSize},
				{addItemTestCommonCountAt, addItemTestCommonCountAt + 4},
				{addItemTestNextEquipAt, addItemTestNextEquipAt + 4},
				{addItemTestNextAcqAt, addItemTestNextAcqAt + 4},
				{addItemTestGaItemDataAt, addItemTestGaItemDataAt + 4},
				{addItemTestGaItemDataArrayAt, arrayEnd},
			})
		})
	}
}

func TestGaItemDataInsertIsIdempotent(t *testing.T) {
	// The item is already listed in GaItemData but the character holds no
	// physical record of it, so the mutation plans an insert and the section
	// itself refuses to add a second entry for the same ID.
	content := addItemTestFixture{
		platform: PlatformPC, slot: 2, tailMarker: true,
		gaItemData: []uint32{0x40000100, addItemTestGoodsID, 0x80000001},
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)

	if _, err := engine.AddItemToInventory(
		loaded.SaveSessionID, content.slot, addItemTestGoodsID, 1, "0", false, 40, 600); err != nil {
		t.Fatalf("AddItemToInventory: %v", err)
	}

	after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)
	addItemTestAssertChanged(t, before, after, [][2]int64{
		{addItemTestCommonAt, addItemTestCommonAt + addItemTestRecordSize},
		{addItemTestCommonCountAt, addItemTestCommonCountAt + 4},
		{addItemTestNextEquipAt, addItemTestNextEquipAt + 4},
		{addItemTestNextAcqAt, addItemTestNextAcqAt + 4},
	})
	if count := addItemTestUint32(after, addItemTestGaItemDataAt); count != 3 {
		t.Errorf("the GaItemData count moved to %d, want 3", count)
	}
}
