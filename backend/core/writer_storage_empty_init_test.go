package core

import (
	"encoding/binary"
	"testing"
)

// TestAddToInventory_EmptyStorageFirstThrowingDaggerMatchesT310 locks in the
// T310 native-confirmed contract for a direct-add into a genuinely empty
// Storage: base hub-elleh.sl2 has Storage.NextAcquisitionSortId=1,
// Storage.NextEquipIndex=0, and after the game natively moved a Throwing
// Dagger into Storage the record landed at Index=2, NextAcquisitionSortId
// became 2, and NextEquipIndex jumped to 128. This test drives the same
// direct-add write path (addToInventory, isStorage=true) that
// AddItemsToSlotBatch uses, and must reproduce those exact values instead of
// the pre-fix 434/435/0 (which came from wrongly applying Inventory's
// InvEquipReservedMax floor to Storage).
func TestAddToInventory_EmptyStorageFirstThrowingDaggerMatchesT310(t *testing.T) {
	slot := buildStorageFixture(t, 0, 1)

	const handle = uint32(ItemTypeItem | 0x6A4) // Throwing Dagger goods handle (0xB00006A4)
	if err := addToInventory(slot, handle, 1, true, false, true); err != nil {
		t.Fatalf("addToInventory: %v", err)
	}

	if len(slot.Storage.CommonItems) != 1 {
		t.Fatalf("CommonItems count: got %d, want 1", len(slot.Storage.CommonItems))
	}
	got := slot.Storage.CommonItems[0]
	if got.GaItemHandle != handle || got.Quantity != 1 {
		t.Fatalf("record: got handle=0x%08X qty=%d, want handle=0x%08X qty=1", got.GaItemHandle, got.Quantity, handle)
	}
	if got.Index != 2 {
		t.Fatalf("record Index: got %d, want 2 (T310)", got.Index)
	}

	if slot.Storage.NextAcquisitionSortId != 2 {
		t.Fatalf("Storage.NextAcquisitionSortId: got %d, want 2 (T310)", slot.Storage.NextAcquisitionSortId)
	}
	if slot.Storage.NextEquipIndex != 128 {
		t.Fatalf("Storage.NextEquipIndex: got %d, want 128 (T310)", slot.Storage.NextEquipIndex)
	}

	// Binary integrity: struct fields must match what was actually written to Data.
	rawAcq := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextAcqSortIdOff:])
	if rawAcq != 2 {
		t.Errorf("binary NextAcquisitionSortId: got %d, want 2", rawAcq)
	}
	rawEquip := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextEquipIndexOff:])
	if rawEquip != 128 {
		t.Errorf("binary NextEquipIndex: got %d, want 128", rawEquip)
	}

	recordOff := StorageHeaderSkip
	rawHandle := binary.LittleEndian.Uint32(slot.Data[recordOff:])
	rawQty := binary.LittleEndian.Uint32(slot.Data[recordOff+4:])
	rawIdx := binary.LittleEndian.Uint32(slot.Data[recordOff+8:])
	if rawHandle != handle || rawQty != 1 || rawIdx != 2 {
		t.Errorf("binary record: got handle=0x%08X qty=%d idx=%d, want handle=0x%08X qty=1 idx=2",
			rawHandle, rawQty, rawIdx, handle)
	}
}

// storageEmptyInitFixture returns a fully serializable slot (GaItems, GaItemData,
// section map — the same production fixture the GaItem-allocation tests use) with
// Storage reset to the exact T310 empty signature: no CommonItems records,
// NextAcquisitionSortId=1, NextEquipIndex=0. fragmentedGaItemRoundTripFixtureForVersion
// pre-populates Storage with one reference Armor record, so that record (and the
// header/counters it implies) is cleared here before returning.
func storageEmptyInitFixture(t *testing.T) *SaveSlot {
	t.Helper()
	slot := fragmentedGaItemRoundTripFixtureForVersion(t, GaItemVersionBreak+1).Slot

	storageRecordOff := slot.StorageBoxOffset + StorageHeaderSkip
	for i := 0; i < InvRecordLen; i++ {
		slot.Data[storageRecordOff+i] = 0
	}
	binary.LittleEndian.PutUint32(slot.Data[slot.StorageBoxOffset:], 0)
	slot.Storage.CommonItems = nil

	binary.LittleEndian.PutUint32(slot.Data[slot.Storage.nextAcqSortIdOff:], 1)
	binary.LittleEndian.PutUint32(slot.Data[slot.Storage.nextEquipIndexOff:], 0)
	slot.Storage.NextAcquisitionSortId = 1
	slot.Storage.NextEquipIndex = 0

	return slot
}

// TestAddItemsToSlotBatch_EmptyStorageSixItemBatchMatchesT330 locks in the T330
// native-confirmed contract: the game natively moved six items into a
// genuinely empty Storage (NextAcquisitionSortId=1, NextEquipIndex=0) and,
// after both post-transfer cold starts, all six were visible in Storage with
// record indexes {2,4,6,8,10,12}, Storage.NextEquipIndex=133, and
// Storage.NextAcquisitionSortId=7. T320 showed that records/header/GaItemData
// existing in the file is not sufficient for in-game visibility, so this test
// asserts the exact counter values T330 confirmed the game itself produces,
// not just "some" values — and it drives the real production entry point,
// AddItemsToSlotBatch, rather than a bare sequence of addToInventory calls, so
// the storageBatchStartedEmpty scoping (decided once from pre-batch state) is
// exercised the same way the app exercises it.
func TestAddItemsToSlotBatch_EmptyStorageSixItemBatchMatchesT330(t *testing.T) {
	slot := storageEmptyInitFixture(t)

	// Six copies of one talisman ID: no serialized GaItem, so this stays on the
	// direct addToInventory write path (no RebuildSlotFull involved) while still
	// producing six distinct physical Storage records, same as T330's six
	// distinct native items — record-index/counter bookkeeping doesn't depend on
	// item type.
	if err := AddItemsToSlotBatch(slot, []ItemToAdd{{ItemID: testTrickMirrorID, StorageQty: 6}}); err != nil {
		t.Fatalf("AddItemsToSlotBatch: %v", err)
	}

	if len(slot.Storage.CommonItems) != 6 {
		t.Fatalf("CommonItems count: got %d, want 6 (T330)", len(slot.Storage.CommonItems))
	}

	wantIndexes := map[uint32]bool{2: true, 4: true, 6: true, 8: true, 10: true, 12: true}
	gotIndexes := make(map[uint32]bool)
	for _, item := range slot.Storage.CommonItems {
		gotIndexes[item.Index] = true
	}
	for idx := range wantIndexes {
		if !gotIndexes[idx] {
			t.Errorf("missing expected record Index %d", idx)
		}
	}
	for idx := range gotIndexes {
		if !wantIndexes[idx] {
			t.Errorf("unexpected record Index %d", idx)
		}
	}

	if slot.Storage.NextEquipIndex != 133 {
		t.Errorf("Storage.NextEquipIndex: got %d, want 133 (T330)", slot.Storage.NextEquipIndex)
	}
	if slot.Storage.NextAcquisitionSortId != 7 {
		t.Errorf("Storage.NextAcquisitionSortId: got %d, want 7 (T330)", slot.Storage.NextAcquisitionSortId)
	}

	rawEquip := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextEquipIndexOff:])
	if rawEquip != 133 {
		t.Errorf("binary NextEquipIndex: got %d, want 133", rawEquip)
	}
	rawAcq := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextAcqSortIdOff:])
	if rawAcq != 7 {
		t.Errorf("binary NextAcquisitionSortId: got %d, want 7", rawAcq)
	}
}

// TestAddItemsToSlotBatch_SecondBatchOnPopulatedStorageAdvancesBothCounters
// covers a second, independent AddItemsToSlotBatch call against a Storage that
// is already populated — the case a user hits on every add after the first,
// because storageBatchStartedEmpty is decided fresh per call from Storage's
// state at that call's start.
//
// Both counters advance exactly as they do in a first batch. NextEquipIndex
// takes +1 per record; NextAcquisitionSortId takes the BUCKET of the new
// record's Index, because Storage stores Index = 2*bucket and the game keys
// Order of Acquisition by Index>>1.
//
// This test previously asserted the opposite — NextEquipIndex untouched and
// NextAcquisitionSortId as a raw high-water mark — on the grounds that lab T330
// only proved the +1 rule for a batch that started from an empty Storage. That
// caution turned out to be wrong: three untouched slots of ER0000-out.sl2
// (250 records -> STOequip 377, 88 -> 215, 6 -> 133, i.e. 127 + N) and their
// bucket values (max Index 1086 -> 544, 1572 -> 787, 198 -> 100) show the rules
// hold regardless of whether the container was empty when the save was loaded.
// The raw high-water rule the old assertion encoded is the INVENTORY rule
// (max(Index) + 1), which had been applied to Storage by mistake.
func TestAddItemsToSlotBatch_SecondBatchOnPopulatedStorageAdvancesBothCounters(t *testing.T) {
	slot := storageEmptyInitFixture(t)

	if err := AddItemsToSlotBatch(slot, []ItemToAdd{{ItemID: testTrickMirrorID, StorageQty: 1}}); err != nil {
		t.Fatalf("first AddItemsToSlotBatch: %v", err)
	}
	if len(slot.Storage.CommonItems) != 1 {
		t.Fatalf("after first batch: CommonItems count = %d, want 1", len(slot.Storage.CommonItems))
	}
	if got := slot.Storage.CommonItems[0].Index; got != 2 {
		t.Fatalf("after first batch: record Index = %d, want 2 (T310 init)", got)
	}
	if slot.Storage.NextAcquisitionSortId != 2 {
		t.Fatalf("after first batch: Storage.NextAcquisitionSortId = %d, want 2 (T310 init)", slot.Storage.NextAcquisitionSortId)
	}
	if slot.Storage.NextEquipIndex != 128 {
		t.Fatalf("after first batch: Storage.NextEquipIndex = %d, want 128 (T310 init)", slot.Storage.NextEquipIndex)
	}
	equipBeforeSecondBatch := slot.Storage.NextEquipIndex

	if err := AddItemsToSlotBatch(slot, []ItemToAdd{{ItemID: testTrickMirrorID, StorageQty: 1}}); err != nil {
		t.Fatalf("second AddItemsToSlotBatch: %v", err)
	}

	if len(slot.Storage.CommonItems) != 2 {
		t.Fatalf("after second batch: CommonItems count = %d, want 2", len(slot.Storage.CommonItems))
	}
	// Both talisman records share the same id-derived handle (dup=true), so the
	// new record is identified by its Index, not its handle.
	foundNewRecord := false
	for _, item := range slot.Storage.CommonItems {
		if item.Index == 4 {
			foundNewRecord = true
		}
	}
	if !foundNewRecord {
		t.Errorf("after second batch: no record with Index=4 (pre-existing policy: next even index past the existing record at Index=2)")
	}

	if want := equipBeforeSecondBatch + 1; slot.Storage.NextEquipIndex != want {
		t.Errorf("Storage.NextEquipIndex: got %d, want %d (+1 for the second record)",
			slot.Storage.NextEquipIndex, want)
	}
	// The new record sits at Index=4, whose bucket is 2, so the next free bucket
	// is 3. The old assertion of 5 was the raw Inventory rule (Index+1).
	if slot.Storage.NextAcquisitionSortId != 3 {
		t.Errorf("Storage.NextAcquisitionSortId: got %d, want 3 (bucket of Index=4, i.e. 4/2+1)",
			slot.Storage.NextAcquisitionSortId)
	}

	rawEquip := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextEquipIndexOff:])
	if want := equipBeforeSecondBatch + 1; rawEquip != want {
		t.Errorf("binary NextEquipIndex: got %d, want %d", rawEquip, want)
	}
	rawAcq := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextAcqSortIdOff:])
	if rawAcq != 3 {
		t.Errorf("binary NextAcquisitionSortId: got %d, want 3", rawAcq)
	}
}

// TestAddItemsToSlotBatchForStorageSession_SixIndependentCallsOnEmptyStorageMatchT350
// exercises the explicit T350 escape hatch at the core layer in isolation from
// the app: six independent AddItemsToSlotBatchForStorageSession calls, each
// adding one item, all passing sessionStorageEmptyAtStart=true (as the caller
// — App.AddItemsToCharacter — would once it has established that Storage was
// genuinely empty when this series began). This must reach the exact same end
// state as T330's single six-item native batch (indexes 2,4,6,8,10,12,
// NextEquipIndex=133, NextAcquisitionSortId=7). The full app-level path is
// covered separately by TestAddItemsToCharacter_SixIndependentDatabaseAddCallsOnEmptyStorageMatchT350
// in app_storage_add_session_test.go.
func TestAddItemsToSlotBatchForStorageSession_SixIndependentCallsOnEmptyStorageMatchT350(t *testing.T) {
	slot := storageEmptyInitFixture(t)

	for i := 0; i < 6; i++ {
		if err := AddItemsToSlotBatchForStorageSession(slot, []ItemToAdd{{ItemID: testTrickMirrorID, StorageQty: 1}}, true); err != nil {
			t.Fatalf("independent AddItemsToSlotBatchForStorageSession call %d: %v", i+1, err)
		}
	}

	if len(slot.Storage.CommonItems) != 6 {
		t.Fatalf("CommonItems count: got %d, want 6 (T350)", len(slot.Storage.CommonItems))
	}

	wantIndexes := map[uint32]bool{2: true, 4: true, 6: true, 8: true, 10: true, 12: true}
	gotIndexes := make(map[uint32]bool)
	for _, item := range slot.Storage.CommonItems {
		gotIndexes[item.Index] = true
	}
	for idx := range wantIndexes {
		if !gotIndexes[idx] {
			t.Errorf("missing expected record Index %d", idx)
		}
	}
	for idx := range gotIndexes {
		if !wantIndexes[idx] {
			t.Errorf("unexpected record Index %d", idx)
		}
	}

	if slot.Storage.NextEquipIndex != 133 {
		t.Errorf("Storage.NextEquipIndex: got %d, want 133 (T350)", slot.Storage.NextEquipIndex)
	}
	if slot.Storage.NextAcquisitionSortId != 7 {
		t.Errorf("Storage.NextAcquisitionSortId: got %d, want 7 (T350)", slot.Storage.NextAcquisitionSortId)
	}

	rawEquip := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextEquipIndexOff:])
	if rawEquip != 133 {
		t.Errorf("binary NextEquipIndex: got %d, want 133", rawEquip)
	}
	rawAcq := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextAcqSortIdOff:])
	if rawAcq != 7 {
		t.Errorf("binary NextAcquisitionSortId: got %d, want 7", rawAcq)
	}
}

// nativeT330StorageFixture reproduces the persisted state of the native lab
// artifact task-330-native-storage-mega through the writer's own verified
// empty-Storage path: six Storage records at the even indices 2..12, with
// NextEquipIndex 133 and NextAcquisitionSortId 7.
//
// Building it this way rather than hand-writing the bytes makes the fixture
// self-validating: if the empty-Storage contract ever drifts from T310/T330,
// these tests fail in setup instead of silently testing a state the app can no
// longer produce.
func nativeT330StorageFixture(t *testing.T) *SaveSlot {
	t.Helper()
	slot := storageEmptyInitFixture(t)

	items := make([]ItemToAdd, 0, 6)
	for i := 0; i < 6; i++ {
		items = append(items, ItemToAdd{ItemID: testTrickMirrorID, StorageQty: 1})
	}
	if err := AddItemsToSlotBatch(slot, items); err != nil {
		t.Fatalf("seeding batch: %v", err)
	}

	if got := len(slot.Storage.CommonItems); got != 6 {
		t.Fatalf("seeded Storage records: got %d, want 6", got)
	}
	for i, item := range slot.Storage.CommonItems {
		if want := uint32(2 + i*2); item.Index != want {
			t.Fatalf("seeded record %d: Index = %d, want %d (native even stride 2)", i, item.Index, want)
		}
	}
	if slot.Storage.NextEquipIndex != 133 {
		t.Fatalf("seeded NextEquipIndex = %d, want 133 (native T330)", slot.Storage.NextEquipIndex)
	}
	if slot.Storage.NextAcquisitionSortId != 7 {
		t.Fatalf("seeded NextAcquisitionSortId = %d, want 7 (native T330)", slot.Storage.NextAcquisitionSortId)
	}
	return slot
}

// TestAddItemsToSlotBatch_PopulatedStorageMatchesNativeCounters is the
// regression for the frozen-counter defect: a Storage that already holds records
// must keep advancing both counters exactly as the game does.
//
// Starting from the native T330 state, one further record must reach Index 14,
// NextEquipIndex 134 (127 + 7 records) and NextAcquisitionSortId 8 (the bucket
// of Index 14).
//
// Before the fix the writer produced 133 and 15: the counter stayed frozen and
// the acquisition mark was computed with the raw Inventory rule.
func TestAddItemsToSlotBatch_PopulatedStorageMatchesNativeCounters(t *testing.T) {
	slot := nativeT330StorageFixture(t)

	if err := AddItemsToSlotBatch(slot, []ItemToAdd{{ItemID: testTrickMirrorID, StorageQty: 1}}); err != nil {
		t.Fatalf("second batch: %v", err)
	}

	if got := len(slot.Storage.CommonItems); got != 7 {
		t.Fatalf("Storage records: got %d, want 7", got)
	}
	found := false
	for _, item := range slot.Storage.CommonItems {
		if item.Index == 14 {
			found = true
		}
	}
	if !found {
		t.Errorf("no record at Index 14 (next even index past 12); got %v", storageIndices(slot))
	}
	if slot.Storage.NextEquipIndex != 134 {
		t.Errorf("Storage.NextEquipIndex: got %d, want 134 (127 + 7 records)",
			slot.Storage.NextEquipIndex)
	}
	if slot.Storage.NextAcquisitionSortId != 8 {
		t.Errorf("Storage.NextAcquisitionSortId: got %d, want 8 (bucket of Index 14)",
			slot.Storage.NextAcquisitionSortId)
	}

	rawEquip := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextEquipIndexOff:])
	if rawEquip != 134 {
		t.Errorf("binary NextEquipIndex: got %d, want 134", rawEquip)
	}
	rawAcq := binary.LittleEndian.Uint32(slot.Data[slot.Storage.nextAcqSortIdOff:])
	if rawAcq != 8 {
		t.Errorf("binary NextAcquisitionSortId: got %d, want 8", rawAcq)
	}
}

// TestAddItemsToSlotBatch_PopulatedStorageCountersTrackRecordCount proves the
// defect cannot come back in the form that mattered: the drift grew with every
// add, so a single-record test would still pass while a real save was hundreds
// of records out of step. Each batch here is independent, which is what a user
// does when adding items repeatedly to a loaded save.
func TestAddItemsToSlotBatch_PopulatedStorageCountersTrackRecordCount(t *testing.T) {
	slot := nativeT330StorageFixture(t)

	const batches = 50
	for i := 0; i < batches; i++ {
		if err := AddItemsToSlotBatch(slot, []ItemToAdd{{ItemID: testTrickMirrorID, StorageQty: 1}}); err != nil {
			t.Fatalf("batch %d: %v", i, err)
		}
	}

	records := len(slot.Storage.CommonItems)
	if records != 6+batches {
		t.Fatalf("Storage records: got %d, want %d", records, 6+batches)
	}
	if want := uint32(127 + records); slot.Storage.NextEquipIndex != want {
		t.Errorf("Storage.NextEquipIndex: got %d, want %d (127 + %d records)",
			slot.Storage.NextEquipIndex, want, records)
	}
	if want := uint32(records + 1); slot.Storage.NextAcquisitionSortId != want {
		t.Errorf("Storage.NextAcquisitionSortId: got %d, want %d (one bucket per record)",
			slot.Storage.NextAcquisitionSortId, want)
	}
}

// storageIndices lists the Index of every occupied Storage record, for failure
// messages.
func storageIndices(slot *SaveSlot) []uint32 {
	out := []uint32{}
	for _, item := range slot.Storage.CommonItems {
		out = append(out, item.Index)
	}
	return out
}
