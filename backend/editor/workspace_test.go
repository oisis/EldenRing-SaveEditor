package editor

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

// fakeRecord places a 12-byte inventory record at the given offset.
func fakeRecord(buf []byte, off int, handle, qty, idx uint32) {
	binary.LittleEndian.PutUint32(buf[off:], handle)
	binary.LittleEndian.PutUint32(buf[off+4:], qty)
	binary.LittleEndian.PutUint32(buf[off+8:], idx)
}

// fixtureSlot builds a minimal *core.SaveSlot with both Inventory and
// Storage record regions writable. Storage starts immediately after the
// inventory area so a single byte slice covers both. GaMap is initialized
// empty — callers add entries as needed.
//
// Layout:
//
//	[0 .. invStart)              padding
//	[invStart .. invEnd)         CommonItemCount × 12 bytes
//	[storageStart .. storageEnd) StorageCommonCount × 12 bytes
func fixtureSlot(t *testing.T) (*core.SaveSlot, int, int) {
	t.Helper()
	const magicOff = 0
	invStart := magicOff + core.InvStartFromMagic
	invEnd := invStart + core.CommonItemCount*core.InvRecordLen
	storageBoxOff := invEnd + 64 // arbitrary gap to mimic FaceData padding
	storageStart := storageBoxOff + core.StorageHeaderSkip
	storageEnd := storageStart + core.StorageCommonCount*core.InvRecordLen
	bufSize := storageEnd + 64

	slot := &core.SaveSlot{
		Version:          1,
		MagicOffset:      magicOff,
		StorageBoxOffset: storageBoxOff,
		Data:             make([]byte, bufSize),
		GaMap:            make(map[uint32]uint32),
	}
	return slot, invStart, storageStart
}

func TestBuildSnapshot_EmptySlot(t *testing.T) {
	slot := &core.SaveSlot{Version: 0, Data: make([]byte, 16)}
	snap, err := BuildSnapshot(slot, "ses-x", 3)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if snap.SessionID != "ses-x" || snap.CharacterIndex != 3 {
		t.Fatalf("metadata mismatch: %+v", snap)
	}
	if len(snap.InventoryItems) != 0 || len(snap.StorageItems) != 0 {
		t.Fatalf("expected empty editable lists, got inv=%d sto=%d",
			len(snap.InventoryItems), len(snap.StorageItems))
	}
	if len(snap.UnsupportedInventoryRecords) != 0 || len(snap.UnsupportedStorageRecords) != 0 {
		t.Fatalf("expected empty unsupported lists")
	}
}

func TestBuildSnapshot_NilSlotReturnsError(t *testing.T) {
	_, err := BuildSnapshot(nil, "ses-x", 0)
	if err == nil {
		t.Fatal("expected error for nil slot")
	}
}

func TestDefaultAoWForBaseID_ResolvesCrossbowKickAlias(t *testing.T) {
	id, name := defaultAoWForBaseID(0x029020C0)
	if id != 4990 || name != "Kick" {
		t.Fatalf("crossbow default skill = %d %q, want 4990 %q", id, name, "Kick")
	}
}

// TestBuildSnapshot_SupportedAndUnsupportedClassification places a
// weapon (supported) and an unknown handle (unsupported) into the same
// inventory and verifies they land in the correct buckets.
func TestBuildSnapshot_SupportedAndUnsupportedClassification(t *testing.T) {
	slot, invStart, _ := fixtureSlot(t)

	// Real DB weapon: Dagger 0x000F4240 (melee_armaments).
	wepHandle := uint32(0x80800001)
	wepID := uint32(0x000F4240)
	slot.GaMap[wepHandle] = wepID
	fakeRecord(slot.Data, invStart+0*core.InvRecordLen, wepHandle, 1, 1000)

	// Unknown / synthetic handle. db.HandleToItemID will decode something
	// the DB has no entry for → ReasonUnknownItem.
	unkHandle := uint32(0xB0123456)
	fakeRecord(slot.Data, invStart+1*core.InvRecordLen, unkHandle, 5, 1002)

	snap, err := BuildSnapshot(slot, "ses-c", 0)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}

	if len(snap.InventoryItems) != 1 {
		t.Fatalf("expected 1 editable item, got %d (%+v)", len(snap.InventoryItems), snap.InventoryItems)
	}
	it := snap.InventoryItems[0]
	if it.OriginalHandle != wepHandle {
		t.Errorf("editable handle = 0x%08X, want 0x%08X", it.OriginalHandle, wepHandle)
	}
	if it.Container != ContainerInventory {
		t.Errorf("container = %q, want inventory", it.Container)
	}
	if it.Position != 0 {
		t.Errorf("position = %d, want 0", it.Position)
	}
	if !it.IsWeapon {
		t.Errorf("Dagger should be IsWeapon=true")
	}
	if it.UID != "hnd:0x80800001" {
		t.Errorf("UID = %q, want hnd:0x80800001", it.UID)
	}
	if !it.HasGaItem {
		t.Errorf("HasGaItem should be true for GaMap-resolved handle")
	}

	if len(snap.UnsupportedInventoryRecords) != 1 {
		t.Fatalf("expected 1 unsupported, got %d", len(snap.UnsupportedInventoryRecords))
	}
	if snap.UnsupportedInventoryRecords[0].Reason == "" {
		t.Error("unsupported record missing Reason")
	}
	// Verify empty slots are skipped (we placed only 2 records).
	if len(snap.UnsupportedInventoryRecords)+len(snap.InventoryItems) != 2 {
		t.Errorf("total non-empty records mismatch")
	}
}

func TestBuildSnapshot_SortedByAcquisitionAscending(t *testing.T) {
	slot, invStart, _ := fixtureSlot(t)

	// Two daggers; second placed earlier physically but with higher acq.
	hA, hB := uint32(0x80800010), uint32(0x80800011)
	slot.GaMap[hA] = 0x000F4240 // Dagger
	slot.GaMap[hB] = 0x000F4240 // Dagger again (different handle)
	fakeRecord(slot.Data, invStart+0*core.InvRecordLen, hA, 1, 5000)
	fakeRecord(slot.Data, invStart+1*core.InvRecordLen, hB, 1, 1000)

	snap, err := BuildSnapshot(slot, "ses-s", 0)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if len(snap.InventoryItems) != 2 {
		t.Fatalf("expected 2 items, got %d", len(snap.InventoryItems))
	}
	if snap.InventoryItems[0].OriginalHandle != hB || snap.InventoryItems[1].OriginalHandle != hA {
		t.Errorf("sort order wrong: [%08X, %08X]",
			snap.InventoryItems[0].OriginalHandle, snap.InventoryItems[1].OriginalHandle)
	}
	if snap.InventoryItems[0].Position != 0 || snap.InventoryItems[1].Position != 1 {
		t.Errorf("positions not assigned sequentially: %d %d",
			snap.InventoryItems[0].Position, snap.InventoryItems[1].Position)
	}
}

func TestBuildSnapshot_UnarmedIsTechnicalPassthrough(t *testing.T) {
	slot, invStart, _ := fixtureSlot(t)
	unarmedHandle := uint32(0x80800020)
	slot.GaMap[unarmedHandle] = 0x0001ADB0 // Unarmed base ID
	fakeRecord(slot.Data, invStart+0*core.InvRecordLen, unarmedHandle, 1, 509)

	snap, err := BuildSnapshot(slot, "ses-u", 0)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if len(snap.InventoryItems) != 0 {
		t.Errorf("Unarmed should not be editable, got %+v", snap.InventoryItems)
	}
	if len(snap.UnsupportedInventoryRecords) != 1 {
		t.Fatalf("expected Unarmed in unsupported, got %d", len(snap.UnsupportedInventoryRecords))
	}
	got := snap.UnsupportedInventoryRecords[0].Reason
	if got != ReasonTechnicalPlaceholder {
		t.Errorf("Reason = %q, want %q", got, ReasonTechnicalPlaceholder)
	}
}

func TestBuildSnapshot_StorageScanned(t *testing.T) {
	slot, _, stoStart := fixtureSlot(t)
	hStorage := uint32(0x80800030)
	slot.GaMap[hStorage] = 0x000F4240 // Dagger
	fakeRecord(slot.Data, stoStart+0*core.InvRecordLen, hStorage, 1, 7000)

	snap, err := BuildSnapshot(slot, "ses-st", 0)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if len(snap.StorageItems) != 1 {
		t.Fatalf("expected 1 storage item, got %d", len(snap.StorageItems))
	}
	if snap.StorageItems[0].Container != ContainerStorage {
		t.Errorf("storage item container = %q", snap.StorageItems[0].Container)
	}
	if !strings.HasPrefix(snap.StorageItems[0].UID, "hnd:") {
		t.Errorf("UID prefix wrong: %q", snap.StorageItems[0].UID)
	}
}

func TestBuildSnapshot_DuplicateHandlePushedToPassthrough(t *testing.T) {
	slot, invStart, _ := fixtureSlot(t)
	h := uint32(0x80800040)
	slot.GaMap[h] = 0x000F4240
	// Same handle in two physical slots.
	fakeRecord(slot.Data, invStart+0*core.InvRecordLen, h, 1, 1000)
	fakeRecord(slot.Data, invStart+5*core.InvRecordLen, h, 1, 1010)

	snap, err := BuildSnapshot(slot, "ses-d", 0)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if len(snap.InventoryItems) != 1 {
		t.Fatalf("expected 1 editable (first occurrence), got %d", len(snap.InventoryItems))
	}
	if len(snap.UnsupportedInventoryRecords) != 1 {
		t.Fatalf("expected 1 passthrough (dup), got %d", len(snap.UnsupportedInventoryRecords))
	}
	if snap.UnsupportedInventoryRecords[0].Reason != ReasonDuplicateHandle {
		t.Errorf("Reason = %q, want %q",
			snap.UnsupportedInventoryRecords[0].Reason, ReasonDuplicateHandle)
	}
}

// TestBuildSnapshot_AffinityWeaponCanonicalMetadata covers the workspace half of
// the canonical-identity contract, in both containers, on the existing public
// EditableItem fields. Cold Dueling Shield +5 is an affinity anchor row that is
// itself a DB entry — the case the old range scan resolved non-deterministically
// and, when it picked the anchor, decoded as "Standard +5".
func TestBuildSnapshot_AffinityWeaponCanonicalMetadata(t *testing.T) {
	const (
		duelingShieldBase = uint32(0x03B9ACA0)
		coldPlus5         = duelingShieldBase + 905
		invHandle         = uint32(0x80800011)
		stoHandle         = uint32(0x80800012)
	)
	slot, invStart, storageStart := fixtureSlot(t)
	slot.GaMap[invHandle] = coldPlus5
	slot.GaMap[stoHandle] = coldPlus5
	fakeRecord(slot.Data, invStart, invHandle, 1, 1000)
	fakeRecord(slot.Data, storageStart, stoHandle, 1, 2000)

	snap, err := BuildSnapshot(slot, "ses-aff", 0)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if len(snap.InventoryItems) != 1 || len(snap.StorageItems) != 1 {
		t.Fatalf("want one editable item per container, got inv=%d sto=%d",
			len(snap.InventoryItems), len(snap.StorageItems))
	}

	wantSort := data.ItemSortKeys[duelingShieldBase]
	wantAoWID, wantAoWName := defaultAoWForBaseID(duelingShieldBase)
	for _, c := range []struct {
		container string
		it        EditableItem
	}{
		{"inventory", snap.InventoryItems[0]},
		{"storage", snap.StorageItems[0]},
	} {
		it := c.it
		if it.ItemID != coldPlus5 {
			t.Errorf("%s: ItemID = 0x%08X, want the raw ID preserved as 0x%08X", c.container, it.ItemID, coldPlus5)
		}
		if it.BaseItemID != duelingShieldBase {
			t.Errorf("%s: BaseItemID = 0x%08X, want 0x%08X", c.container, it.BaseItemID, duelingShieldBase)
		}
		if it.Name != "Dueling Shield" {
			t.Errorf("%s: Name = %q, want %q", c.container, it.Name, "Dueling Shield")
		}
		if it.InfusionName != "Cold" {
			t.Errorf("%s: InfusionName = %q, want %q", c.container, it.InfusionName, "Cold")
		}
		if it.CurrentUpgrade != 5 {
			t.Errorf("%s: CurrentUpgrade = %d, want 5", c.container, it.CurrentUpgrade)
		}
		if it.MaxUpgrade != 25 {
			t.Errorf("%s: MaxUpgrade = %d, want 25", c.container, it.MaxUpgrade)
		}
		// Sorting stays keyed on the canonical base, matching every affinity row
		// that has no DB entry of its own.
		if it.SortID != wantSort.SortId || it.SortGroupID != wantSort.SortGroupId {
			t.Errorf("%s: sort key = {%d,%d}, want the canonical base key {%d,%d}",
				c.container, it.SortID, it.SortGroupID, wantSort.SortId, wantSort.SortGroupId)
		}
		if it.Weight != data.ItemWeights[duelingShieldBase] {
			t.Errorf("%s: Weight = %v, want %v", c.container, it.Weight, data.ItemWeights[duelingShieldBase])
		}
		if it.WepType != 90 {
			t.Errorf("%s: WepType = %d, want 90 (Dueling / Thrusting Shields)", c.container, it.WepType)
		}
		if !it.CanChangeAffinity {
			t.Errorf("%s: CanChangeAffinity = false, want true", c.container)
		}
		if it.DefaultAoWID != wantAoWID || it.DefaultAoWName != wantAoWName {
			t.Errorf("%s: default AoW = %d %q, want %d %q",
				c.container, it.DefaultAoWID, it.DefaultAoWName, wantAoWID, wantAoWName)
		}
	}
}
