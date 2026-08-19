package core

import (
	"encoding/binary"
	"testing"
)

// buildNextEquipFixture builds a SaveSlot with a pre-populated inventory
// (CommonItems slice + binary) and explicit NextEquipIndex / NextAcquisitionSortId.
// StorageBoxOffset is pointed outside Data so mapInventory skips storage parsing.
func buildNextEquipFixture(t *testing.T, items []InventoryItem, nextEquip, nextAcq uint32) *SaveSlot {
	t.Helper()
	magicOff := 0
	commonStart := magicOff + InvStartFromMagic
	keyStart := commonStart + CommonItemCount*InvRecordLen + InvKeyCountHeader
	nextEquipOff := keyStart + KeyItemCount*InvRecordLen
	nextAcqOff := nextEquipOff + 4
	bufSize := nextAcqOff + 4 + 64

	slot := &SaveSlot{
		Version:     1,
		MagicOffset: magicOff,
		Data:        make([]byte, bufSize),
		GaMap:       make(map[uint32]uint32),
	}

	// Full 2688-slot CommonItems slice — empty by default.
	slot.Inventory.CommonItems = make([]InventoryItem, CommonItemCount)
	for i := range slot.Inventory.CommonItems {
		slot.Inventory.CommonItems[i] = InventoryItem{GaItemHandle: GaHandleEmpty}
	}
	for i, it := range items {
		if i >= CommonItemCount {
			t.Fatalf("too many items: %d", len(items))
		}
		slot.Inventory.CommonItems[i] = it
		off := commonStart + i*InvRecordLen
		binary.LittleEndian.PutUint32(slot.Data[off:], it.GaItemHandle)
		binary.LittleEndian.PutUint32(slot.Data[off+4:], it.Quantity)
		binary.LittleEndian.PutUint32(slot.Data[off+8:], it.Index)
	}

	binary.LittleEndian.PutUint32(slot.Data[nextEquipOff:], nextEquip)
	binary.LittleEndian.PutUint32(slot.Data[nextAcqOff:], nextAcq)
	slot.Inventory.NextEquipIndex = nextEquip
	slot.Inventory.NextAcquisitionSortId = nextAcq
	slot.Inventory.nextEquipIndexOff = nextEquipOff
	slot.Inventory.nextAcqSortIdOff = nextAcqOff

	// Prevent storage parsing when mapInventory is called.
	slot.StorageBoxOffset = bufSize
	return slot
}

// buildStorageFixture builds a SaveSlot with empty storage binary and explicit counters.
// MagicOffset is pointed outside Data so mapInventory skips inventory parsing.
func buildStorageFixture(t *testing.T, nextEquip, nextAcq uint32) *SaveSlot {
	t.Helper()
	storageBoxOff := 0
	storageStart := storageBoxOff + StorageHeaderSkip
	nextEquipOff := storageStart + StorageNextEquipIdxRel
	nextAcqOff := storageStart + StorageNextAcqSortRel
	bufSize := nextAcqOff + 4 + 4

	slot := &SaveSlot{
		Version:          1,
		StorageBoxOffset: storageBoxOff,
		Data:             make([]byte, bufSize),
		GaMap:            make(map[uint32]uint32),
	}
	// All binary slots are zeroed (GaHandleEmpty=0) → all empty; in-memory list is nil.

	binary.LittleEndian.PutUint32(slot.Data[nextEquipOff:], nextEquip)
	binary.LittleEndian.PutUint32(slot.Data[nextAcqOff:], nextAcq)
	slot.Storage.NextEquipIndex = nextEquip
	slot.Storage.NextAcquisitionSortId = nextAcq
	slot.Storage.nextEquipIndexOff = nextEquipOff
	slot.Storage.nextAcqSortIdOff = nextAcqOff

	// Prevent inventory parsing when mapInventory is called.
	slot.MagicOffset = bufSize
	return slot
}

// buildStorageFixtureDense builds a SaveSlot whose Storage binary really holds
// `n` records at physical indices 0..n-1 with even acquisition indices 2,4,...,2n.
// buildStorageFixture leaves the binary table empty, which makes a NextEquipIndex
// of 500 physically impossible: under the native rule NextEquipIndex ==
// 128 + last_occupied_index, a counter of 500 means the last occupied physical
// index is 372, i.e. 373 records. Native evidence for the rule: ER0000-out.sl2
// (250 records, last index 249 -> 377) and t2-f4-refill.sl2 (1230 records,
// last index 1229 -> 1357).
func buildStorageFixtureDense(t *testing.T, n int, nextEquip, nextAcq uint32) *SaveSlot {
	t.Helper()
	slot := buildStorageFixture(t, nextEquip, nextAcq)
	storageStart := slot.StorageBoxOffset + StorageHeaderSkip
	for i := 0; i < n; i++ {
		it := InventoryItem{
			GaItemHandle: uint32(0xB0000001 + i),
			Quantity:     1,
			Index:        uint32(2 * (i + 1)),
		}
		off := storageStart + i*InvRecordLen
		binary.LittleEndian.PutUint32(slot.Data[off:], it.GaItemHandle)
		binary.LittleEndian.PutUint32(slot.Data[off+4:], it.Quantity)
		binary.LittleEndian.PutUint32(slot.Data[off+8:], it.Index)
		slot.Storage.CommonItems = append(slot.Storage.CommonItems, it)
	}
	binary.LittleEndian.PutUint32(slot.Data[slot.StorageBoxOffset:], uint32(n))
	return slot
}

// TestNextEquipIndex_InvInsert verifies that inserting a genuinely NEW
// Inventory.CommonItems record advances NextEquipIndex by exactly one — T050/T210
// native evidence (Throwing Dagger add: NextEquipIndex 433 -> 434) — while
// NextAcquisitionSortId advances past the newly assigned Index. This does not
// reintroduce the CE-108255-1 global reconcile (see
// TestNextEquipIndex_MapInventoryNoGlobalReconcile below): the +1 here is a
// local per-insert step, never a jump to match NextAcquisitionSortId or the
// item's own Index.
func TestNextEquipIndex_InvInsert(t *testing.T) {
	// Simulate genuine PC save: NextEquipIndex=500 well below NextAcquisitionSortId=1000.
	items := make([]InventoryItem, 10)
	for i := range items {
		items[i] = InventoryItem{GaItemHandle: uint32(0xB0000001 + i), Quantity: 1, Index: uint32(990 + i)}
	}
	slot := buildNextEquipFixture(t, items, 500, 1000)
	acqBefore := slot.Inventory.NextAcquisitionSortId // 1000

	const newHandle = uint32(0xB0ABCDEF)
	if err := addToInventory(slot, newHandle, 99, false, false, false); err != nil {
		t.Fatalf("addToInventory: %v", err)
	}

	const wantEquip = uint32(501)
	if slot.Inventory.NextEquipIndex != wantEquip {
		t.Errorf("struct NextEquipIndex: got %d, want %d", slot.Inventory.NextEquipIndex, wantEquip)
	}
	rawEquip := binary.LittleEndian.Uint32(slot.Data[slot.Inventory.nextEquipIndexOff:])
	if rawEquip != wantEquip {
		t.Errorf("binary NextEquipIndex: got %d, want %d", rawEquip, wantEquip)
	}
	// NextAcquisitionSortId is a high-water MARK: the new record gets mark+1,
	// then the counter advances to mark+2 (T050/T210). acqBefore=1000 is already
	// an even, above-reserved mark, so mark==acqBefore here.
	wantAcq := acqBefore + 2
	if slot.Inventory.NextAcquisitionSortId != wantAcq {
		t.Errorf("NextAcquisitionSortId: got %d, want %d", slot.Inventory.NextAcquisitionSortId, wantAcq)
	}
}

// TestNextEquipIndex_StorageInsert verifies that a direct-add to a Storage
// that was already non-empty before the enclosing batch started (i.e. NOT the
// T310/T330 empty-init case — storageBatchStartedEmpty=false) leaves
// NextEquipIndex untouched, per the general "NextEquipIndex is a separate
// game-owned counter" invariant. The empty-init counter jump is covered
// separately by TestAddToInventory_EmptyStorageFirstThrowingDaggerMatchesT310
// and the batch-scoped T330 rule by
// TestAddItemsToSlotBatch_EmptyStorageSixItemBatchMatchesT330.
func TestNextEquipIndex_StorageInsert(t *testing.T) {
	// 373 records occupy physical 0..372, which is exactly what NextEquipIndex=500
	// means under the native rule 128 + last_occupied_index. The new record lands at
	// physical 373, so the expected 501 below is 128 + 373 — unchanged.
	slot := buildStorageFixtureDense(t, 373, 500, 1000)
	acqBefore := slot.Storage.NextAcquisitionSortId // 1000

	const newHandle = uint32(0xB0ABCDEF)
	if err := addToInventory(slot, newHandle, 99, true, false, false); err != nil {
		t.Fatalf("addToInventory: %v", err)
	}

	// NextEquipIndex tracks the physical layout: 128 + last_occupied_index. The
	// fixture holds 373 records (physical 0..372), the new record takes physical
	// 373, so the counter becomes 128 + 373 = 501. Native evidence: three
	// untouched slots of ER0000-out.sl2 (250 records, last 249 -> 377; 88, last
	// 87 -> 215; 6, last 5 -> 133) and t2-f4-refill.sl2 (1230, last 1229 -> 1357).
	const wantEquip = uint32(501)
	if slot.Storage.NextEquipIndex != wantEquip {
		t.Errorf("struct NextEquipIndex: got %d, want %d", slot.Storage.NextEquipIndex, wantEquip)
	}
	rawEquip := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextEquipIndexOff:])
	if rawEquip != wantEquip {
		t.Errorf("binary NextEquipIndex: got %d, want %d", rawEquip, wantEquip)
	}

	// Storage keeps NextAcquisitionSortId in BUCKETS, not raw indices: the game
	// keys Order of Acquisition by Index>>1 and Storage stores Index = 2*bucket,
	// so the field is max(Index)/2 + 1. Native evidence: lab T330 (max Index 12
	// -> 7) and ER0000-out.sl2 (1086 -> 544, 1572 -> 787, 198 -> 100). This is
	// where Storage differs from Inventory, which keeps a raw mark of
	// max(Index) + 1 (15668 -> 15669, 969 -> 970).
	_ = acqBefore
	newIndex := uint32(0)
	for _, item := range slot.Storage.CommonItems {
		if item.GaItemHandle == newHandle {
			newIndex = item.Index
		}
	}
	if newIndex == 0 {
		t.Fatal("the inserted record was not found in Storage")
	}
	wantAcq := newIndex/2 + 1
	if slot.Storage.NextAcquisitionSortId != wantAcq {
		t.Errorf("NextAcquisitionSortId: got %d, want %d (bucket of Index %d)",
			slot.Storage.NextAcquisitionSortId, wantAcq, newIndex)
	}
	rawAcq := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextAcqSortIdOff:])
	if rawAcq != wantAcq {
		t.Errorf("binary NextAcquisitionSortId: got %d, want %d", rawAcq, wantAcq)
	}
}

// TestNextEquipIndex_MapInventoryNoGlobalReconcile guards against re-introducing
// CE-108255-1: mapInventory must NOT globally bump NextEquipIndex to match
// NextAcquisitionSortId on every load. Items well above NextEquipIndex must not
// trigger a write-back.
func TestNextEquipIndex_MapInventoryNoGlobalReconcile(t *testing.T) {
	items := make([]InventoryItem, 10)
	for i := range items {
		items[i] = InventoryItem{GaItemHandle: uint32(0xB0000001 + i), Quantity: 1, Index: uint32(1000 + i)}
	}
	// NextEquipIndex=50 far below item indices; NextAcquisitionSortId=1200.
	const lowEquip = uint32(50)
	slot := buildNextEquipFixture(t, items, lowEquip, 1200)

	if err := slot.mapInventory(); err != nil {
		t.Fatalf("mapInventory: %v", err)
	}

	if slot.Inventory.NextEquipIndex != lowEquip {
		t.Errorf("mapInventory changed NextEquipIndex: got %d, want %d",
			slot.Inventory.NextEquipIndex, lowEquip)
	}
	rawEquip := binary.LittleEndian.Uint32(slot.Data[slot.Inventory.nextEquipIndexOff:])
	if rawEquip != lowEquip {
		t.Errorf("mapInventory wrote NextEquipIndex to binary: got %d, want %d", rawEquip, lowEquip)
	}
}
