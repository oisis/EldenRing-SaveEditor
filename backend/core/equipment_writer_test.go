package core

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	testEquipHeaderOff          = 0x400
	testSpellsOff               = 0x1000
	equipmentWriterWeaponHandle = 0x80800010
	equipmentWriterArmorHandle  = 0x90800020
	equipmentWriterArrowHandle  = 0x80800030
	equipmentWriterBoltHandle   = 0x80800031
	equipmentWriterWeaponID     = 0x00100020
	equipmentWriterArmorID      = 0x10100040
	equipmentWriterArrowID      = 0x02FAF080
	equipmentWriterBoltID       = 0x03197500
	equipmentWriterTalisman     = 0xA00003E8
	equipmentWriterTalisman2    = 0xA00003F2
	equipmentWriterTalisman3    = 0xA00003FC
	equipmentWriterTalisman4    = 0xA0000406
	equipmentWriterTalismanRow  = 20
	equipmentWriterUnarmedRow   = 5
	equipmentWriterBareHeadRow  = 6
	equipmentWriterBareChestRow = 7
	equipmentWriterBareArmsRow  = 8
	equipmentWriterBareLegsRow  = 9
)

func makeEquipmentTestSlot() *SaveSlot {
	data := make([]byte, SlotSize)
	projHeader := testSpellsOff + DynEquipedSpells + DynEquipedItems + DynEquipedGestures
	binary.LittleEndian.PutUint32(data[projHeader:], 0)
	armaments := projHeader + 4

	slot := &SaveSlot{
		Data:                 data,
		MagicOffset:          testEquipHeaderOff - (DynSpEffect + DynEquipedItemIndex + DynActiveEquipedItems + DynEquipedItemsID),
		EquipItemsIDOffset:   testEquipHeaderOff,
		EquippedSpellsOffset: testSpellsOff,
		Player:               PlayerGameData{TalismanSlots: 3},
		GaMap: map[uint32]uint32{
			0x80800079:                  unarmedEquipmentItemID,
			equipmentWriterWeaponHandle: equipmentWriterWeaponID,
			0x80800011:                  0x00100021,
			equipmentWriterArmorHandle:  equipmentWriterArmorID,
			equipmentWriterArrowHandle:  equipmentWriterArrowID,
			equipmentWriterBoltHandle:   equipmentWriterBoltID,
			0x9080007F:                  0x10002710,
			0x90800080:                  0x10002774,
			0x90800081:                  0x100027D8,
			0x90800082:                  0x1000283C,
		},
	}
	slot.Inventory.CommonItems = make([]InventoryItem, equipmentWriterTalismanRow+1)
	slot.Inventory.CommonItems[0] = InventoryItem{GaItemHandle: equipmentWriterWeaponHandle, Quantity: 1}
	slot.Inventory.CommonItems[1] = InventoryItem{GaItemHandle: 0x80800011, Quantity: 1}
	slot.Inventory.CommonItems[2] = InventoryItem{GaItemHandle: equipmentWriterArmorHandle, Quantity: 1}
	slot.Inventory.CommonItems[3] = InventoryItem{GaItemHandle: equipmentWriterArrowHandle, Quantity: 1}
	slot.Inventory.CommonItems[4] = InventoryItem{GaItemHandle: equipmentWriterBoltHandle, Quantity: 1}
	slot.Inventory.CommonItems[equipmentWriterUnarmedRow] = InventoryItem{GaItemHandle: 0x80800079, Quantity: 1}
	slot.Inventory.CommonItems[equipmentWriterBareHeadRow] = InventoryItem{GaItemHandle: 0x9080007F, Quantity: 1}
	slot.Inventory.CommonItems[equipmentWriterBareChestRow] = InventoryItem{GaItemHandle: 0x90800080, Quantity: 1}
	slot.Inventory.CommonItems[equipmentWriterBareArmsRow] = InventoryItem{GaItemHandle: 0x90800081, Quantity: 1}
	slot.Inventory.CommonItems[equipmentWriterBareLegsRow] = InventoryItem{GaItemHandle: 0x90800082, Quantity: 1}
	slot.Inventory.CommonItems[equipmentWriterTalismanRow] = InventoryItem{GaItemHandle: equipmentWriterTalisman, Quantity: 1}

	// Native empty layout established by T547: all four representations point
	// at Unarmed / the per-slot bare armor rows, while ammunition is all-empty.
	for i := 0; i < ChrAsmFieldCount; i++ {
		binary.LittleEndian.PutUint32(data[armaments+i*4:], GaHandleInvalid)
	}
	for _, index := range []int{0, 1, 2, 3, 4, 5} {
		equipIndexOff, itemIDOff, handleOff, err := slot.equipmentRepresentationOffsets(index)
		if err != nil {
			panic(err)
		}
		binary.LittleEndian.PutUint32(data[equipIndexOff:], inventoryEquipIndexBase+equipmentWriterUnarmedRow)
		binary.LittleEndian.PutUint32(data[itemIDOff:], unarmedEquipmentItemID&0x0FFFFFFF)
		binary.LittleEndian.PutUint32(data[handleOff:], 0x80800079)
		binary.LittleEndian.PutUint32(data[armaments+index*4:], unarmedEquipmentItemID)
	}
	bareRows := map[int]int{
		12: equipmentWriterBareHeadRow,
		13: equipmentWriterBareChestRow,
		14: equipmentWriterBareArmsRow,
		15: equipmentWriterBareLegsRow,
	}
	bareHandles := map[int]uint32{
		12: 0x9080007F,
		13: 0x90800080,
		14: 0x90800081,
		15: 0x90800082,
	}
	for index, itemID := range bareArmorItemIDBySlot {
		equipIndexOff, itemIDOff, handleOff, err := slot.equipmentRepresentationOffsets(index)
		if err != nil {
			panic(err)
		}
		binary.LittleEndian.PutUint32(data[equipIndexOff:], inventoryEquipIndexBase+uint32(bareRows[index]))
		binary.LittleEndian.PutUint32(data[itemIDOff:], itemID&0x0FFFFFFF)
		binary.LittleEndian.PutUint32(data[handleOff:], bareHandles[index])
		binary.LittleEndian.PutUint32(data[armaments+index*4:], itemID)
	}
	for index := firstTalismanChrAsmIndex; index < firstTalismanChrAsmIndex+talismanSlotCount; index++ {
		equipIndexOff, itemIDOff, handleOff, err := slot.equipmentRepresentationOffsets(index)
		if err != nil {
			panic(err)
		}
		binary.LittleEndian.PutUint32(data[equipIndexOff:], GaHandleInvalid)
		binary.LittleEndian.PutUint32(data[itemIDOff:], GaHandleInvalid)
		binary.LittleEndian.PutUint32(data[handleOff:], 0)
	}
	return slot
}

func dynamicEquip(slot *SaveSlot, index int) uint32 {
	off, err := slot.equippedArmamentsOffset()
	if err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint32(slot.Data[off+index*4:])
}

func nativeEquipmentValues(slot *SaveSlot, index int) (equipIndex, itemID, handle, dynamic uint32) {
	equipIndexOff, itemIDOff, handleOff, err := slot.equipmentRepresentationOffsets(index)
	if err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint32(slot.Data[equipIndexOff:]),
		binary.LittleEndian.Uint32(slot.Data[itemIDOff:]),
		binary.LittleEndian.Uint32(slot.Data[handleOff:]),
		dynamicEquip(slot, index)
}

func talismanNativeValues(slot *SaveSlot, index int) (equipIndex, itemID, handle, dynamic uint32) {
	return nativeEquipmentValues(slot, index)
}

func TestWriteEquipment_WritesFourNativeRepresentations(t *testing.T) {
	slot := makeEquipmentTestSlot()
	if err := slot.WriteEquipment([]EquipmentWrite{
		{Slot: EquipSlotRightHandArmament1, Handle: equipmentWriterWeaponHandle},
		{Slot: EquipSlotArrows1, Handle: equipmentWriterArrowHandle},
		{Slot: EquipSlotHead, Handle: equipmentWriterArmorHandle},
	}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		index                               int
		equipIndex, itemID, handle, dynamic uint32
	}{
		{1, inventoryEquipIndexBase + 0, equipmentWriterWeaponID & 0x0FFFFFFF, equipmentWriterWeaponHandle, equipmentWriterWeaponID},
		{6, inventoryEquipIndexBase + 3, equipmentWriterArrowID & 0x0FFFFFFF, equipmentWriterArrowHandle, equipmentWriterArrowID},
		{12, inventoryEquipIndexBase + 2, equipmentWriterArmorID & 0x0FFFFFFF, equipmentWriterArmorHandle, equipmentWriterArmorID},
	}
	for _, tc := range cases {
		equipIndex, itemID, handle, dynamic := nativeEquipmentValues(slot, tc.index)
		if equipIndex != tc.equipIndex || itemID != tc.itemID || handle != tc.handle || dynamic != tc.dynamic {
			t.Errorf("slot %d = %08X/%08X/%08X/%08X, want %08X/%08X/%08X/%08X",
				tc.index,
				equipIndex, itemID, handle, dynamic,
				tc.equipIndex, tc.itemID, tc.handle, tc.dynamic)
		}
	}
}

func TestWriteEquipment_ClearUsesNativeSentinels(t *testing.T) {
	slot := makeEquipmentTestSlot()
	if err := slot.WriteEquipment([]EquipmentWrite{
		{Slot: EquipSlotLeftHandArmament1, Handle: 0},
		{Slot: EquipSlotArrows1, Handle: 0},
		{Slot: EquipSlotBolts1, Handle: 0},
		{Slot: EquipSlotHead, Handle: 0},
	}); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		index                               int
		equipIndex, itemID, handle, dynamic uint32
	}{
		{0, inventoryEquipIndexBase + equipmentWriterUnarmedRow, unarmedEquipmentItemID & 0x0FFFFFFF, 0x80800079, unarmedEquipmentItemID},
		{6, GaHandleInvalid, GaHandleInvalid, 0, GaHandleInvalid},
		{7, GaHandleInvalid, GaHandleInvalid, 0, GaHandleInvalid},
		{12, inventoryEquipIndexBase + equipmentWriterBareHeadRow, 0x00002710, 0x9080007F, 0x10002710},
	}
	for _, tc := range cases {
		equipIndex, itemID, handle, dynamic := nativeEquipmentValues(slot, tc.index)
		if equipIndex != tc.equipIndex || itemID != tc.itemID || handle != tc.handle || dynamic != tc.dynamic {
			t.Errorf("slot %d = %08X/%08X/%08X/%08X, want %08X/%08X/%08X/%08X",
				tc.index,
				equipIndex, itemID, handle, dynamic,
				tc.equipIndex, tc.itemID, tc.handle, tc.dynamic)
		}
	}
}

func TestWriteEquipment_ClearProvisionsMissingBareArmorPlaceholder(t *testing.T) {
	fixture := fragmentedGaItemRoundTripFixture(t)
	slot := fixture.Slot
	const occupiedArmorID = uint32(0x10000001)

	if err := AddItemsToSlotBatch(slot, []ItemToAdd{{ItemID: occupiedArmorID, InvQty: 1}}); err != nil {
		t.Fatalf("add occupied armor: %v", err)
	}
	occupiedHandle := inventoryHandleForItemID(slot, occupiedArmorID, nil)
	if occupiedHandle == 0 {
		t.Fatal("new occupied armor has no Inventory/GaMap handle")
	}
	if err := slot.WriteEquipment([]EquipmentWrite{{Slot: EquipSlotChest, Handle: occupiedHandle}}); err != nil {
		t.Fatalf("equip chest: %v", err)
	}
	if got := countInventoryItemID(slot, bareArmorItemIDBySlot[13]); got != 0 {
		t.Fatalf("fixture already has %d bare-chest placeholder(s), want 0", got)
	}

	if err := slot.WriteEquipment([]EquipmentWrite{{Slot: EquipSlotChest, Handle: 0}}); err != nil {
		t.Fatalf("clear chest: %v", err)
	}

	equipIndex, itemID, handle, dynamic := nativeEquipmentValues(slot, 13)
	if itemID != bareArmorItemIDBySlot[13]&0x0FFFFFFF || dynamic != bareArmorItemIDBySlot[13] {
		t.Fatalf("cleared chest = itemID/dynamic %08X/%08X, want %08X/%08X",
			itemID, dynamic, bareArmorItemIDBySlot[13]&0x0FFFFFFF, bareArmorItemIDBySlot[13])
	}
	if !nativeHandleMatches(slot, equipIndex, handle, bareArmorItemIDBySlot[13]) {
		t.Fatalf("cleared chest handle %08X does not reference provisioned bare-body row %08X", handle, equipIndex)
	}
	if got := countInventoryItemID(slot, bareArmorItemIDBySlot[13]); got != 1 {
		t.Fatalf("bare-chest placeholder count = %d, want 1", got)
	}
}

func TestWriteEquipment_ClearReusesExistingNativeUnarmedPlaceholder(t *testing.T) {
	fixture := fragmentedGaItemRoundTripFixture(t)
	slot := fixture.Slot
	const occupiedWeaponID = uint32(0x000F4240)

	adds := make([]ItemToAdd, 6)
	for i := range adds {
		adds[i] = ItemToAdd{ItemID: occupiedWeaponID, InvQty: 1}
	}
	if err := AddItemsToSlotBatch(slot, adds); err != nil {
		t.Fatalf("add occupied weapons: %v", err)
	}
	occupiedHandles := inventoryHandlesForItemID(slot, occupiedWeaponID)
	if len(occupiedHandles) < 6 {
		t.Fatalf("occupied weapon handles = %d, want at least 6", len(occupiedHandles))
	}

	equipWrites := make([]EquipmentWrite, 6)
	clearWrites := make([]EquipmentWrite, 6)
	for i := range equipWrites {
		equipWrites[i] = EquipmentWrite{Slot: EquipmentSlotKind(i), Handle: occupiedHandles[i]}
		clearWrites[i] = EquipmentWrite{Slot: EquipmentSlotKind(i), Handle: 0}
	}
	if err := slot.WriteEquipment(equipWrites); err != nil {
		t.Fatalf("equip six hand slots: %v", err)
	}
	if got := countInventoryItemID(slot, unarmedEquipmentItemID); got != 1 {
		t.Fatalf("initial Unarmed count = %d, want 1", got)
	}

	if err := slot.WriteEquipment(clearWrites); err != nil {
		t.Fatalf("clear six hand slots: %v", err)
	}

	for index := 0; index < 6; index++ {
		equipIndex, itemID, handle, dynamic := nativeEquipmentValues(slot, index)
		if itemID != unarmedEquipmentItemID || dynamic != unarmedEquipmentItemID {
			t.Errorf("hand slot %d = itemID/dynamic %08X/%08X, want Unarmed", index, itemID, dynamic)
		}
		if handle != fixture.Handles.Unarmed {
			t.Errorf("hand slot %d handle = %08X, want existing native Unarmed %08X", index, handle, fixture.Handles.Unarmed)
		}
		if !nativeHandleMatches(slot, equipIndex, handle, unarmedEquipmentItemID) {
			t.Errorf("hand slot %d handle %08X does not reference its Inventory row %08X", index, handle, equipIndex)
		}
	}
	if got := countInventoryItemID(slot, unarmedEquipmentItemID); got != 1 {
		t.Fatalf("final Unarmed count = %d, want the existing shared native record", got)
	}
}

func TestWriteEquipment_ClearProvisionsOneMissingUnarmedPlaceholder(t *testing.T) {
	fixture := fragmentedGaItemRoundTripFixture(t)
	slot := fixture.Slot
	const occupiedWeaponID = uint32(0x000F4240)

	if err := RemoveItemsFromSlot(slot, []uint32{fixture.Handles.Unarmed}, true, false); err != nil {
		t.Fatalf("remove fixture Unarmed: %v", err)
	}
	if got := countInventoryItemID(slot, unarmedEquipmentItemID); got != 0 {
		t.Fatalf("Unarmed count after fixture preparation = %d, want 0", got)
	}
	adds := []ItemToAdd{
		{ItemID: occupiedWeaponID, InvQty: 1},
		{ItemID: occupiedWeaponID, InvQty: 1},
	}
	if err := AddItemsToSlotBatch(slot, adds); err != nil {
		t.Fatalf("add occupied weapons: %v", err)
	}
	occupiedHandles := inventoryHandlesForItemID(slot, occupiedWeaponID)
	if len(occupiedHandles) < 2 {
		t.Fatalf("occupied weapon handles = %d, want at least 2", len(occupiedHandles))
	}
	if err := slot.WriteEquipment([]EquipmentWrite{
		{Slot: EquipSlotLeftHandArmament1, Handle: occupiedHandles[0]},
		{Slot: EquipSlotRightHandArmament1, Handle: occupiedHandles[1]},
	}); err != nil {
		t.Fatalf("equip two hand slots: %v", err)
	}

	if err := slot.WriteEquipment([]EquipmentWrite{
		{Slot: EquipSlotLeftHandArmament1, Handle: 0},
		{Slot: EquipSlotRightHandArmament1, Handle: 0},
	}); err != nil {
		t.Fatalf("clear two hand slots: %v", err)
	}

	_, _, leftHandle, _ := nativeEquipmentValues(slot, 0)
	_, _, rightHandle, _ := nativeEquipmentValues(slot, 1)
	if leftHandle == 0 || rightHandle != leftHandle {
		t.Fatalf("provisioned Unarmed handles left/right = %08X/%08X, want one shared non-zero handle", leftHandle, rightHandle)
	}
	if got := countInventoryItemID(slot, unarmedEquipmentItemID); got != 1 {
		t.Fatalf("final Unarmed count = %d, want one provisioned record", got)
	}
}

func TestWriteEquipment_NativeEmptyProvisionRollsBackOnLaterError(t *testing.T) {
	fixture := fragmentedGaItemRoundTripFixture(t)
	slot := fixture.Slot
	const occupiedArmorID = uint32(0x10000001)

	if err := AddItemsToSlotBatch(slot, []ItemToAdd{{ItemID: occupiedArmorID, InvQty: 1}}); err != nil {
		t.Fatalf("add occupied armor: %v", err)
	}
	occupiedHandle := inventoryHandleForItemID(slot, occupiedArmorID, nil)
	if err := slot.WriteEquipment([]EquipmentWrite{{Slot: EquipSlotChest, Handle: occupiedHandle}}); err != nil {
		t.Fatalf("equip chest: %v", err)
	}
	before := SnapshotSlot(slot)

	err := slot.WriteEquipment([]EquipmentWrite{
		{Slot: EquipSlotChest, Handle: 0},
		{Slot: EquipSlotRightHandArmament1, Handle: 0x8080FFFF},
	})
	if err == nil || !strings.Contains(err.Error(), "WriteEquipment[1]") {
		t.Fatalf("later invalid handle error = %v", err)
	}
	if after := SnapshotSlot(slot); !reflect.DeepEqual(after, before) {
		t.Fatal("failed batch did not roll back provisioned placeholder and slot state")
	}
}

func inventoryHandlesForItemID(slot *SaveSlot, itemID uint32) []uint32 {
	var handles []uint32
	for _, item := range slot.Inventory.CommonItems {
		if item.Quantity&0x7FFFFFFF != 0 && slot.GaMap[item.GaItemHandle] == itemID {
			handles = append(handles, item.GaItemHandle)
		}
	}
	return handles
}

func inventoryHandleForItemID(slot *SaveSlot, itemID uint32, excluded map[uint32]bool) uint32 {
	for _, handle := range inventoryHandlesForItemID(slot, itemID) {
		if !excluded[handle] {
			return handle
		}
	}
	return 0
}

func countInventoryItemID(slot *SaveSlot, itemID uint32) int {
	return len(inventoryHandlesForItemID(slot, itemID))
}

func TestWriteEquipment_TalismanWritesFourNativeRepresentations(t *testing.T) {
	slot := makeEquipmentTestSlot()
	if err := slot.WriteEquipment([]EquipmentWrite{{Slot: EquipSlotTalisman1, Handle: equipmentWriterTalisman}}); err != nil {
		t.Fatal(err)
	}

	equipIndex, itemID, handle, dynamic := talismanNativeValues(slot, firstTalismanChrAsmIndex)
	if equipIndex != inventoryEquipIndexBase+equipmentWriterTalismanRow {
		t.Errorf("EquipData equip_index = 0x%08X, want 0x%08X", equipIndex, uint32(inventoryEquipIndexBase+equipmentWriterTalismanRow))
	}
	if itemID != 0x000003E8 {
		t.Errorf("ChrAsm item ID = 0x%08X, want 0x000003E8", itemID)
	}
	if handle != equipmentWriterTalisman {
		t.Errorf("ChrAsm2 handle = 0x%08X, want 0x%08X", handle, uint32(equipmentWriterTalisman))
	}
	if dynamic != 0x200003E8 {
		t.Errorf("equipped-armaments item ID = 0x%08X, want 0x200003E8", dynamic)
	}
}

func TestWriteEquipment_TalismanClearUsesNativeSentinels(t *testing.T) {
	slot := makeEquipmentTestSlot()
	if err := slot.WriteEquipment([]EquipmentWrite{{Slot: EquipSlotTalisman1, Handle: equipmentWriterTalisman}}); err != nil {
		t.Fatal(err)
	}
	if err := slot.WriteEquipment([]EquipmentWrite{{Slot: EquipSlotTalisman1, Handle: 0}}); err != nil {
		t.Fatal(err)
	}

	equipIndex, itemID, handle, dynamic := talismanNativeValues(slot, firstTalismanChrAsmIndex)
	if equipIndex != GaHandleInvalid || itemID != GaHandleInvalid || handle != 0 || dynamic != GaHandleInvalid {
		t.Errorf("cleared talisman = %08X/%08X/%08X/%08X, want FFFFFFFF/FFFFFFFF/00000000/FFFFFFFF",
			equipIndex, itemID, handle, dynamic)
	}
}

func TestWriteEquipment_TalismanValidationIsAtomic(t *testing.T) {
	t.Run("locked-slot", func(t *testing.T) {
		slot := makeEquipmentTestSlot()
		slot.Player.TalismanSlots = 0
		before := append([]byte(nil), slot.Data...)
		err := slot.WriteEquipment([]EquipmentWrite{{Slot: EquipSlotTalisman2, Handle: equipmentWriterTalisman}})
		if err == nil || !strings.Contains(err.Error(), "locked") {
			t.Fatalf("error = %v, want locked-slot rejection", err)
		}
		if !bytes.Equal(slot.Data, before) {
			t.Error("locked-slot rejection mutated data")
		}
	})

	t.Run("wrong-handle-type", func(t *testing.T) {
		slot := makeEquipmentTestSlot()
		before := append([]byte(nil), slot.Data...)
		err := slot.WriteEquipment([]EquipmentWrite{{Slot: EquipSlotTalisman1, Handle: equipmentWriterWeaponHandle}})
		if err == nil || !strings.Contains(err.Error(), "not a talisman handle") {
			t.Fatalf("error = %v, want handle-type rejection", err)
		}
		if !bytes.Equal(slot.Data, before) {
			t.Error("handle-type rejection mutated data")
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		slot := makeEquipmentTestSlot()
		before := append([]byte(nil), slot.Data...)
		err := slot.WriteEquipment([]EquipmentWrite{
			{Slot: EquipSlotTalisman1, Handle: equipmentWriterTalisman},
			{Slot: EquipSlotTalisman2, Handle: equipmentWriterTalisman},
		})
		if err == nil || !strings.Contains(err.Error(), "cannot occupy slots 1 and 2") {
			t.Fatalf("error = %v, want duplicate-talisman rejection", err)
		}
		if !bytes.Equal(slot.Data, before) {
			t.Error("duplicate rejection mutated data")
		}
	})
}

func TestWriteEquipment_PreservesUnprovenHashBytes(t *testing.T) {
	slot := makeEquipmentTestSlot()
	copy(slot.Data[HashOffset:HashOffset+HashSize], bytes.Repeat([]byte{0xA5}, HashSize))
	before := append([]byte(nil), slot.Data[HashOffset:HashOffset+HashSize]...)
	if err := slot.WriteEquipment([]EquipmentWrite{{Slot: EquipSlotHead, Handle: equipmentWriterArmorHandle}}); err != nil {
		t.Fatal(err)
	}
	if got := slot.Data[HashOffset : HashOffset+HashSize]; !bytes.Equal(got, before) {
		t.Error("WriteEquipment changed the unproven hash block")
	}
}

func TestWriteEquipment_IsAtomicAcrossBothRegions(t *testing.T) {
	slot := makeEquipmentTestSlot()
	before := append([]byte(nil), slot.Data...)
	err := slot.WriteEquipment([]EquipmentWrite{
		{Slot: EquipSlotRightHandArmament1, Handle: equipmentWriterWeaponHandle},
		{Slot: EquipSlotHead, Handle: equipmentWriterWeaponHandle},
	})
	if err == nil || !strings.Contains(err.Error(), "cannot equip weapon") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(slot.Data, before) {
		t.Error("failed batch mutated slot data")
	}
}

func TestWriteEquipment_RejectsWrongOrMissingHandles(t *testing.T) {
	cases := []EquipmentWrite{
		{Slot: EquipSlotRightHandArmament1, Handle: 0xC0000050},
		{Slot: EquipSlotHead, Handle: equipmentWriterWeaponHandle},
		{Slot: EquipSlotRightHandArmament1, Handle: 0x80800099},
		{Slot: EquipSlotArrows1, Handle: GaHandleInvalid},
	}
	for _, write := range cases {
		slot := makeEquipmentTestSlot()
		if err := slot.WriteEquipment([]EquipmentWrite{write}); err == nil {
			t.Errorf("write %+v unexpectedly succeeded", write)
		}
	}
}

func TestWriteEquipment_RejectsDuplicateAndUnknownSlots(t *testing.T) {
	slot := makeEquipmentTestSlot()
	if err := slot.WriteEquipment([]EquipmentWrite{
		{Slot: EquipSlotRightHandArmament1, Handle: equipmentWriterWeaponHandle},
		{Slot: EquipSlotRightHandArmament1, Handle: 0},
	}); err == nil || !strings.Contains(err.Error(), "already written") {
		t.Fatalf("duplicate error = %v", err)
	}
	if err := slot.WriteEquipment([]EquipmentWrite{{Slot: EquipmentSlotKind(99), Handle: equipmentWriterWeaponHandle}}); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unknown-slot error = %v", err)
	}
}

func TestWriteEquipment_T547NativeClearContract(t *testing.T) {
	root := filepath.Join("..", "..", "tmp", "item-save-lab", "artifacts", "tasks")
	dirs, err := os.ReadDir(root)
	if err != nil {
		t.Skip("T547 artifacts unavailable")
	}
	paths := map[string]string{}
	for _, dir := range dirs {
		if !dir.IsDir() || (!strings.Contains(dir.Name(), "544") && !strings.Contains(dir.Name(), "547")) {
			continue
		}
		files, err := os.ReadDir(filepath.Join(root, dir.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".sl2") {
				continue
			}
			if strings.Contains(file.Name(), "baseline") {
				paths["before"] = filepath.Join(root, dir.Name(), file.Name())
			}
			if strings.Contains(file.Name(), "cleared") {
				paths["after"] = filepath.Join(root, dir.Name(), file.Name())
			}
		}
	}
	if paths["before"] == "" || paths["after"] == "" {
		t.Skip("T544/T547 artifacts unavailable")
	}
	before, err := LoadSave(paths["before"])
	if err != nil {
		t.Fatal(err)
	}
	after, err := LoadSave(paths["after"])
	if err != nil {
		t.Fatal(err)
	}
	writes := []EquipmentWrite{
		{Slot: EquipSlotLeftHandArmament1, Handle: 0},
		{Slot: EquipSlotRightHandArmament1, Handle: 0},
		{Slot: EquipSlotArrows1, Handle: 0},
		{Slot: EquipSlotBolts1, Handle: 0},
		{Slot: EquipSlotHead, Handle: 0},
		{Slot: EquipSlotChest, Handle: 0},
		{Slot: EquipSlotArms, Handle: 0},
		{Slot: EquipSlotLegs, Handle: 0},
	}
	if err := before.Slots[0].WriteEquipment(writes); err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{0, 1, 6, 7, 12, 13, 14, 15} {
		gotEquipIndex, gotItemID, gotHandle, gotDynamic := nativeEquipmentValues(&before.Slots[0], index)
		wantEquipIndex, wantItemID, wantHandle, wantDynamic := nativeEquipmentValues(&after.Slots[0], index)
		if gotEquipIndex != wantEquipIndex || gotItemID != wantItemID || gotDynamic != wantDynamic {
			t.Errorf("slot %d fixed/dynamic values do not match native clear: got %08X/%08X/%08X; want %08X/%08X/%08X",
				index,
				gotEquipIndex, gotItemID, gotDynamic,
				wantEquipIndex, wantItemID, wantDynamic)
		}
		if gotHandle == 0 || wantHandle == 0 {
			if gotHandle != wantHandle {
				t.Errorf("slot %d empty handle = %08X, native = %08X", index, gotHandle, wantHandle)
			}
			continue
		}
		if !nativeHandleMatches(&before.Slots[0], gotEquipIndex, gotHandle, gotDynamic) {
			t.Errorf("slot %d writer handle %08X is not the inventory row referenced by EquipData %08X for item %08X",
				index, gotHandle, gotEquipIndex, gotDynamic)
		}
		if !nativeHandleMatches(&after.Slots[0], wantEquipIndex, wantHandle, wantDynamic) {
			t.Errorf("slot %d native handle %08X is not the inventory row referenced by EquipData %08X for item %08X",
				index, wantHandle, wantEquipIndex, wantDynamic)
		}
	}
}

func TestWriteEquipment_T064NativeAmmoContract(t *testing.T) {
	save := loadTaskArtifact(t, "task-064-", "05-pickup-a-b.sl2")
	slot := &save.Slots[0]
	before := append([]byte(nil), slot.Data...)
	_, _, arrowHandle, _ := nativeEquipmentValues(slot, 6)
	_, _, boltHandle, _ := nativeEquipmentValues(slot, 7)

	if err := slot.WriteEquipment([]EquipmentWrite{
		{Slot: EquipSlotArrows1, Handle: 0},
		{Slot: EquipSlotBolts1, Handle: 0},
	}); err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{6, 7} {
		equipIndex, itemID, handle, dynamic := nativeEquipmentValues(slot, index)
		if equipIndex != GaHandleInvalid || itemID != GaHandleInvalid || handle != 0 || dynamic != GaHandleInvalid {
			t.Errorf("cleared ammo slot %d = %08X/%08X/%08X/%08X, want FFFFFFFF/FFFFFFFF/00000000/FFFFFFFF",
				index, equipIndex, itemID, handle, dynamic)
		}
	}

	if err := slot.WriteEquipment([]EquipmentWrite{
		{Slot: EquipSlotArrows1, Handle: arrowHandle},
		{Slot: EquipSlotBolts1, Handle: boltHandle},
	}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(slot.Data, before) {
		t.Error("clear then restore did not reproduce the byte-exact native T064 ammunition state")
	}
}

func TestWriteEquipment_T548NativeTalismanContract(t *testing.T) {
	t.Run("equip-one", func(t *testing.T) {
		removed := loadT548Artifact(t, "04-native-one-talisman-removed-cold.sl2")
		native := loadT548Artifact(t, "03-native-one-talisman-equipped-cold.sl2")
		if err := removed.Slots[0].WriteEquipment([]EquipmentWrite{
			{Slot: EquipSlotTalisman1, Handle: equipmentWriterTalisman},
		}); err != nil {
			t.Fatal(err)
		}
		assertTalismanNativeMatches(t, &removed.Slots[0], &native.Slots[0], firstTalismanChrAsmIndex)
	})

	t.Run("clear-one", func(t *testing.T) {
		equipped := loadT548Artifact(t, "03-native-one-talisman-equipped-cold.sl2")
		native := loadT548Artifact(t, "04-native-one-talisman-removed-cold.sl2")
		if err := equipped.Slots[0].WriteEquipment([]EquipmentWrite{
			{Slot: EquipSlotTalisman1, Handle: 0},
		}); err != nil {
			t.Fatal(err)
		}
		assertTalismanNativeMatches(t, &equipped.Slots[0], &native.Slots[0], firstTalismanChrAsmIndex)
	})

	t.Run("equip-four", func(t *testing.T) {
		removed := loadT548Artifact(t, "04-native-one-talisman-removed-cold.sl2")
		native := loadT548Artifact(t, "05-native-four-talismans-equipped-cold.sl2")
		writes := []EquipmentWrite{
			{Slot: EquipSlotTalisman1, Handle: equipmentWriterTalisman},
			{Slot: EquipSlotTalisman2, Handle: equipmentWriterTalisman2},
			{Slot: EquipSlotTalisman3, Handle: equipmentWriterTalisman3},
			{Slot: EquipSlotTalisman4, Handle: equipmentWriterTalisman4},
		}
		if err := removed.Slots[0].WriteEquipment(writes); err != nil {
			t.Fatal(err)
		}
		assertTalismanNativeMatches(t, &removed.Slots[0], &native.Slots[0], 17, 18, 19, 20)
	})
}

func assertTalismanNativeMatches(t *testing.T, got, want *SaveSlot, indices ...int) {
	t.Helper()
	for _, index := range indices {
		gotEquipIndex, gotItemID, gotHandle, gotDynamic := talismanNativeValues(got, index)
		wantEquipIndex, wantItemID, wantHandle, wantDynamic := talismanNativeValues(want, index)
		if gotEquipIndex != wantEquipIndex || gotItemID != wantItemID || gotHandle != wantHandle || gotDynamic != wantDynamic {
			t.Errorf("talisman slot %d does not match native T548 values: got %08X %08X %08X %08X; want %08X %08X %08X %08X",
				index-firstTalismanChrAsmIndex+1,
				gotEquipIndex, gotItemID, gotHandle, gotDynamic,
				wantEquipIndex, wantItemID, wantHandle, wantDynamic)
		}
	}
}

func loadT548Artifact(t *testing.T, name string) *SaveFile {
	return loadTaskArtifact(t, "task-548", name)
}

func loadTaskArtifact(t *testing.T, taskPrefix, name string) *SaveFile {
	t.Helper()
	root := filepath.Join("..", "..", "tmp", "item-save-lab", "artifacts", "tasks")
	dirs, err := filepath.Glob(filepath.Join(root, taskPrefix+"*"))
	if err != nil || len(dirs) != 1 {
		t.Skipf("%s artifacts unavailable", taskPrefix)
	}
	path := filepath.Join(dirs[0], name)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s artifact unavailable", taskPrefix)
	}
	save, err := LoadSave(path)
	if err != nil {
		t.Fatal(err)
	}
	return save
}

func nativeHandleMatches(slot *SaveSlot, equipIndex, handle, itemID uint32) bool {
	if equipIndex < inventoryEquipIndexBase {
		return false
	}
	row := int(equipIndex - inventoryEquipIndexBase)
	if row < 0 || row >= len(slot.Inventory.CommonItems) {
		return false
	}
	item := slot.Inventory.CommonItems[row]
	return item.GaItemHandle == handle &&
		item.Quantity&0x7FFFFFFF != 0 &&
		slot.GaMap[handle] == itemID
}
