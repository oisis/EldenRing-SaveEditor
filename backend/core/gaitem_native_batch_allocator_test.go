package core

import (
	"reflect"
	"testing"
)

func TestNativeGaItemBatchAllocator_AllocatesMixedBatchInAscendingHoles(t *testing.T) {
	slot := makeTestSlot(8)
	oldAoW := uint32(ItemTypeAow | gaItemHandleValidBit | 3)
	oldWeapon := uint32(ItemTypeWeapon | gaItemHandleValidBit | 5)
	slot.GaItems[0] = nativeTestRecord(oldAoW, 0x80000003)
	slot.GaItems[5] = nativeTestRecord(oldWeapon, 0x00100005)
	slot.GaMap[oldAoW] = 0x80000003
	slot.GaMap[oldWeapon] = 0x00100005

	allocator, err := newNativeGaItemBatchAllocator(slot)
	if err != nil {
		t.Fatalf("newNativeGaItemBatchAllocator: %v", err)
	}

	armor, err := allocator.allocate(ItemTypeArmor, 0x10100000)
	if err != nil {
		t.Fatalf("allocate armor: %v", err)
	}
	aow, err := allocator.allocate(ItemTypeAow, 0x80000010)
	if err != nil {
		t.Fatalf("allocate AoW: %v", err)
	}
	weapon, err := allocator.allocate(ItemTypeWeapon, 0x00100020)
	if err != nil {
		t.Fatalf("allocate weapon: %v", err)
	}

	if got := []uint32{armor & 0xFFFF, aow & 0xFFFF, weapon & 0xFFFF}; !reflect.DeepEqual(got, []uint32{0, 1, 2}) {
		t.Fatalf("physical indices = %v, want [0 1 2]", got)
	}
	if err := allocator.commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	wantProjection := map[int]uint32{
		0: aow,
		1: oldAoW,
		2: armor,
		3: weapon,
		5: oldWeapon,
	}
	for position, want := range wantProjection {
		if got := slot.GaItems[position].Handle; got != want {
			t.Errorf("GaItems[%d].Handle = 0x%08X, want 0x%08X", position, got, want)
		}
	}
	if _, err := analyzeNativeGaItemLayout(slot); err != nil {
		t.Fatalf("post-commit layout: %v", err)
	}
}

func TestNativeGaItemBatchAllocator_MatchesSingleAllocator(t *testing.T) {
	newSlot := func() *SaveSlot {
		slot := makeTestSlot(10)
		aow := uint32(ItemTypeAow | gaItemHandleValidBit | 3)
		weapon := uint32(ItemTypeWeapon | gaItemHandleValidBit | 7)
		slot.GaItems[0] = nativeTestRecord(aow, 0x80000003)
		slot.GaItems[7] = nativeTestRecord(weapon, 0x00100007)
		slot.GaMap[aow] = 0x80000003
		slot.GaMap[weapon] = 0x00100007
		return slot
	}
	adds := []struct {
		prefix uint32
		itemID uint32
	}{
		{ItemTypeArmor, 0x10100000},
		{ItemTypeAow, 0x80000010},
		{ItemTypeWeapon, 0x00100020},
		{ItemTypeArmor, 0x10100030},
	}

	single := newSlot()
	var singleHandles []uint32
	for _, add := range adds {
		handle, err := generateUniqueHandle(single, add.prefix)
		if err != nil {
			t.Fatalf("generateUniqueHandle: %v", err)
		}
		if err := allocateGaItem(single, handle, add.itemID); err != nil {
			t.Fatalf("allocateGaItem: %v", err)
		}
		single.GaMap[handle] = add.itemID
		singleHandles = append(singleHandles, handle)
	}

	batch := newSlot()
	allocator, err := newNativeGaItemBatchAllocator(batch)
	if err != nil {
		t.Fatalf("newNativeGaItemBatchAllocator: %v", err)
	}
	var batchHandles []uint32
	for _, add := range adds {
		handle, err := allocator.allocate(add.prefix, add.itemID)
		if err != nil {
			t.Fatalf("batch allocate: %v", err)
		}
		batchHandles = append(batchHandles, handle)
	}
	if err := allocator.commit(); err != nil {
		t.Fatalf("batch commit: %v", err)
	}

	if !reflect.DeepEqual(batchHandles, singleHandles) {
		t.Fatalf("batch handles = %v, single handles = %v", batchHandles, singleHandles)
	}
	if !reflect.DeepEqual(nonEmptyGaItems(batch.GaItems), nonEmptyGaItems(single.GaItems)) {
		t.Fatalf("batch projection differs from single allocator\nbatch=%+v\nsingle=%+v",
			nonEmptyGaItems(batch.GaItems), nonEmptyGaItems(single.GaItems))
	}
	if !reflect.DeepEqual(batch.GaMap, single.GaMap) {
		t.Fatalf("batch GaMap = %v, single GaMap = %v", batch.GaMap, single.GaMap)
	}
	if batch.NextAoWIndex != single.NextAoWIndex ||
		batch.NextArmamentIndex != single.NextArmamentIndex ||
		batch.NextGaItemHandle != single.NextGaItemHandle {
		t.Fatalf("batch tracking = {%d,%d,%d}, single = {%d,%d,%d}",
			batch.NextAoWIndex, batch.NextArmamentIndex, batch.NextGaItemHandle,
			single.NextAoWIndex, single.NextArmamentIndex, single.NextGaItemHandle)
	}
}

func nonEmptyGaItems(items []GaItemFull) []GaItemFull {
	result := make([]GaItemFull, 0, len(items))
	for _, item := range items {
		if !item.IsEmpty() {
			result = append(result, item)
		}
	}
	return result
}

func TestNativeGaItemBatchAllocator_FullTableDoesNotMutateSlot(t *testing.T) {
	slot := makeTestSlot(3)
	for index := range slot.GaItems {
		handle := uint32(ItemTypeWeapon | gaItemHandleValidBit | uint32(index))
		slot.GaItems[index] = nativeTestRecord(handle, uint32(0x00100000+index))
		slot.GaMap[handle] = uint32(0x00100000 + index)
	}
	beforeItems := append([]GaItemFull(nil), slot.GaItems...)
	beforeMap := make(map[uint32]uint32, len(slot.GaMap))
	for handle, itemID := range slot.GaMap {
		beforeMap[handle] = itemID
	}

	allocator, err := newNativeGaItemBatchAllocator(slot)
	if err != nil {
		t.Fatalf("newNativeGaItemBatchAllocator: %v", err)
	}
	if _, err := allocator.allocate(ItemTypeArmor, 0x10100000); err == nil || !contains(err.Error(), "no free index") {
		t.Fatalf("allocate error = %v, want no free index", err)
	}
	if !reflect.DeepEqual(slot.GaItems, beforeItems) || !reflect.DeepEqual(slot.GaMap, beforeMap) {
		t.Fatal("full-table refusal mutated the slot")
	}
}

func TestNativeGaItemBatchAllocator_RejectsInvalidItemWithoutConsumingHole(t *testing.T) {
	slot := makeTestSlot(4)
	allocator, err := newNativeGaItemBatchAllocator(slot)
	if err != nil {
		t.Fatalf("newNativeGaItemBatchAllocator: %v", err)
	}

	if _, err := allocator.allocate(ItemTypeWeapon, 0x10100000); err == nil || !contains(err.Error(), "conflicts with item ID") {
		t.Fatalf("invalid allocation error = %v, want type mismatch", err)
	}
	handle, err := allocator.allocate(ItemTypeArmor, 0x10100000)
	if err != nil {
		t.Fatalf("valid allocation: %v", err)
	}
	if got := handle & 0xFFFF; got != 0 {
		t.Fatalf("physical index = %d, want unconsumed first hole 0", got)
	}
}

func TestNativeGaItemBatchAllocator_RejectsMalformedInitialProjection(t *testing.T) {
	slot := makeTestSlot(4)
	slot.GaItems[0] = nativeTestRecord(ItemTypeWeapon|gaItemHandleValidBit|3, 0x00100003)
	before := append([]GaItemFull(nil), slot.GaItems...)

	if _, err := newNativeGaItemBatchAllocator(slot); err == nil || !contains(err.Error(), "native GaItem projection") {
		t.Fatalf("newNativeGaItemBatchAllocator error = %v, want projection refusal", err)
	}
	if !reflect.DeepEqual(slot.GaItems, before) {
		t.Fatal("projection refusal mutated GaItems")
	}
}

func TestAddItemsToSlotBatch_MixedPhysicalItemsUseValidProjection(t *testing.T) {
	slot := fragmentedGaItemRoundTripFixture(t).Slot
	items := []ItemToAdd{
		{ItemID: 0x000F4241, InvQty: 1},
		{ItemID: 0x10000002, InvQty: 1},
		{ItemID: 0x80000002, InvQty: 1},
	}

	if err := AddItemsToSlotBatch(slot, items); err != nil {
		t.Fatalf("AddItemsToSlotBatch: %v", err)
	}
	for _, item := range items {
		found := false
		for _, record := range slot.GaItems {
			if !record.IsEmpty() && record.ItemID == item.ItemID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GaItem 0x%08X not found after batch add", item.ItemID)
		}
	}
	if _, err := analyzeNativeGaItemLayout(slot); err != nil {
		t.Fatalf("post-batch layout: %v", err)
	}
}
