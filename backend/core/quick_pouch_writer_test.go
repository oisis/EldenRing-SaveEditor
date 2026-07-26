package core

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

const (
	quickPouchDaggerHandle    = uint32(0xB00006A4)
	quickPouchDaggerID        = uint32(0x400006A4)
	quickPouchTelescopeHandle = uint32(0xB00007F8)
	quickPouchTelescopeID     = uint32(0x400007F8)
	quickPouchStoneHandle     = uint32(0xB0000758)
	quickPouchStoneID         = uint32(0x40000758)
)

func makeQuickPouchWriterTestSlot() *SaveSlot {
	slot := makeEquipmentTestSlot()
	slot.GaMap[quickPouchDaggerHandle] = quickPouchDaggerID
	slot.GaMap[quickPouchTelescopeHandle] = quickPouchTelescopeID
	slot.GaMap[quickPouchStoneHandle] = quickPouchStoneID
	slot.Inventory.CommonItems = append(slot.Inventory.CommonItems,
		InventoryItem{GaItemHandle: quickPouchDaggerHandle, Quantity: 20},
		InventoryItem{GaItemHandle: quickPouchTelescopeHandle, Quantity: 1},
		InventoryItem{GaItemHandle: quickPouchStoneHandle, Quantity: 5},
	)

	pairBase := slot.EquippedSpellsOffset + DynEquipedSpells
	for i := 0; i < equipItemDataQuickCount; i++ {
		off := pairBase + i*8
		binary.LittleEndian.PutUint32(slot.Data[off:], 0)
		binary.LittleEndian.PutUint32(slot.Data[off+4:], GaHandleInvalid)
	}
	binary.LittleEndian.PutUint32(slot.Data[pairBase+equipItemDataActiveOff:], 4)
	for i := 0; i < equipItemDataPouchCount; i++ {
		off := pairBase + equipItemDataPouchOff + i*8
		binary.LittleEndian.PutUint32(slot.Data[off:], 0)
		binary.LittleEndian.PutUint32(slot.Data[off+4:], GaHandleInvalid)
	}

	armamentsOff, err := slot.equippedArmamentsOffset()
	if err != nil {
		panic(err)
	}
	for i := 0; i < equipItemDataQuickCount; i++ {
		binary.LittleEndian.PutUint32(slot.Data[armamentsOff+quickPouchDynamicQuick+i*4:], GaHandleInvalid)
	}
	for i := 0; i < equipItemDataPouchCount; i++ {
		binary.LittleEndian.PutUint32(slot.Data[armamentsOff+quickPouchDynamicPouch+i*4:], GaHandleInvalid)
	}
	binary.LittleEndian.PutUint32(slot.Data[HashOffset+9*4:], 0xCAFEBABE)
	return slot
}

func TestWriteQuickPouch_WritesBothNativeRepresentations(t *testing.T) {
	slot := makeQuickPouchWriterTestSlot()
	daggerRow := len(slot.Inventory.CommonItems) - 3
	telescopeRow := len(slot.Inventory.CommonItems) - 2

	if err := slot.WriteQuickPouch([]QuickPouchWrite{
		{Slot: QuickPouchSlotQuick1, Handle: quickPouchDaggerHandle},
		{Slot: QuickPouchSlotPouch1, Handle: quickPouchTelescopeHandle},
	}); err != nil {
		t.Fatal(err)
	}

	raw, err := slot.ReadEquippedState()
	if err != nil {
		t.Fatal(err)
	}
	if got := raw.QuickItems[0]; got.ItemID != quickPouchDaggerHandle || got.EquipIndex != inventoryEquipIndexBase+uint32(daggerRow) {
		t.Errorf("QuickItems[0] = %+v, want handle 0x%08X and row 0x%X", got, quickPouchDaggerHandle, inventoryEquipIndexBase+uint32(daggerRow))
	}
	if got := raw.Pouch[0]; got.ItemID != quickPouchTelescopeHandle || got.EquipIndex != inventoryEquipIndexBase+uint32(telescopeRow) {
		t.Errorf("Pouch[0] = %+v, want handle 0x%08X and row 0x%X", got, quickPouchTelescopeHandle, inventoryEquipIndexBase+uint32(telescopeRow))
	}
	armamentsOff, _ := slot.equippedArmamentsOffset()
	if got := binary.LittleEndian.Uint32(slot.Data[armamentsOff+quickPouchDynamicQuick:]); got != quickPouchDaggerID {
		t.Errorf("QuickItems direct ID = 0x%08X, want 0x%08X", got, quickPouchDaggerID)
	}
	if got := binary.LittleEndian.Uint32(slot.Data[armamentsOff+quickPouchDynamicPouch:]); got != quickPouchTelescopeID {
		t.Errorf("Pouch direct ID = 0x%08X, want 0x%08X", got, quickPouchTelescopeID)
	}
}

func TestWriteQuickPouch_ClearUsesNativeSentinels(t *testing.T) {
	slot := makeQuickPouchWriterTestSlot()
	if err := slot.WriteQuickPouch([]QuickPouchWrite{
		{Slot: QuickPouchSlotQuick1, Handle: quickPouchDaggerHandle},
		{Slot: QuickPouchSlotPouch1, Handle: quickPouchTelescopeHandle},
	}); err != nil {
		t.Fatal(err)
	}
	if err := slot.WriteQuickPouch([]QuickPouchWrite{
		{Slot: QuickPouchSlotQuick1, Handle: 0},
		{Slot: QuickPouchSlotPouch1, Handle: 0},
	}); err != nil {
		t.Fatal(err)
	}

	raw, err := slot.ReadEquippedState()
	if err != nil {
		t.Fatal(err)
	}
	for name, pair := range map[string]RawEquipItem{"quick": raw.QuickItems[0], "pouch": raw.Pouch[0]} {
		if pair.ItemID != 0 || pair.EquipIndex != GaHandleInvalid {
			t.Errorf("%s clear = %+v, want {0, 0xFFFFFFFF}", name, pair)
		}
	}
	armamentsOff, _ := slot.equippedArmamentsOffset()
	for name, off := range map[string]int{
		"quick": armamentsOff + quickPouchDynamicQuick,
		"pouch": armamentsOff + quickPouchDynamicPouch,
	} {
		if got := binary.LittleEndian.Uint32(slot.Data[off:]); got != GaHandleInvalid {
			t.Errorf("%s direct clear = 0x%08X, want 0xFFFFFFFF", name, got)
		}
	}
}

func TestWriteQuickPouch_PreservesActiveQuantityAndHash(t *testing.T) {
	slot := makeQuickPouchWriterTestSlot()
	pairBase := slot.EquippedSpellsOffset + DynEquipedSpells
	activeBefore := append([]byte(nil), slot.Data[pairBase+equipItemDataActiveOff:pairBase+equipItemDataActiveOff+4]...)
	hashBefore := append([]byte(nil), slot.Data[HashOffset+9*4:HashOffset+10*4]...)
	inventoryBefore := append([]InventoryItem(nil), slot.Inventory.CommonItems...)

	if err := slot.WriteQuickPouch([]QuickPouchWrite{{Slot: QuickPouchSlotQuick1, Handle: quickPouchDaggerHandle}}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(activeBefore, slot.Data[pairBase+equipItemDataActiveOff:pairBase+equipItemDataActiveOff+4]) {
		t.Error("active Quick Item index changed")
	}
	if !bytes.Equal(hashBefore, slot.Data[HashOffset+9*4:HashOffset+10*4]) {
		t.Error("unproven hash[9] changed")
	}
	for i := range inventoryBefore {
		if slot.Inventory.CommonItems[i] != inventoryBefore[i] {
			t.Fatalf("Inventory.CommonItems[%d] changed: got %+v want %+v", i, slot.Inventory.CommonItems[i], inventoryBefore[i])
		}
	}
}

func TestWriteQuickPouch_ValidatesAtomically(t *testing.T) {
	slot := makeQuickPouchWriterTestSlot()
	before := append([]byte(nil), slot.Data...)
	err := slot.WriteQuickPouch([]QuickPouchWrite{
		{Slot: QuickPouchSlotQuick1, Handle: quickPouchDaggerHandle},
		{Slot: QuickPouchSlotQuick2, Handle: 0xA00003E8},
	})
	if err == nil || !strings.Contains(err.Error(), "not a goods handle") {
		t.Fatalf("WriteQuickPouch error = %v, want wrong-handle rejection", err)
	}
	if !bytes.Equal(before, slot.Data) {
		t.Error("slot mutated after validation failure")
	}
}

func TestWriteQuickPouch_RejectsDuplicatesWithinFamilyButAllowsCrossFamily(t *testing.T) {
	slot := makeQuickPouchWriterTestSlot()
	if err := slot.WriteQuickPouch([]QuickPouchWrite{
		{Slot: QuickPouchSlotQuick1, Handle: quickPouchDaggerHandle},
		{Slot: QuickPouchSlotQuick2, Handle: quickPouchDaggerHandle},
	}); err == nil || !strings.Contains(err.Error(), "Quick Items item") {
		t.Fatalf("duplicate Quick Items error = %v", err)
	}

	if err := slot.WriteQuickPouch([]QuickPouchWrite{
		{Slot: QuickPouchSlotQuick1, Handle: quickPouchDaggerHandle},
		{Slot: QuickPouchSlotPouch1, Handle: quickPouchDaggerHandle},
	}); err != nil {
		t.Fatalf("same item in separate Quick/Pouch families should be allowed: %v", err)
	}
}

func TestWriteQuickPouch_RejectsDuplicateSlotAndUnsupportedSlot(t *testing.T) {
	slot := makeQuickPouchWriterTestSlot()
	if err := slot.WriteQuickPouch([]QuickPouchWrite{
		{Slot: QuickPouchSlotQuick1, Handle: quickPouchDaggerHandle},
		{Slot: QuickPouchSlotQuick1, Handle: quickPouchStoneHandle},
	}); err == nil || !strings.Contains(err.Error(), "already written") {
		t.Fatalf("duplicate slot error = %v", err)
	}
	if err := slot.WriteQuickPouch([]QuickPouchWrite{{Slot: QuickPouchSlotKind(99), Handle: quickPouchDaggerHandle}}); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unsupported slot error = %v", err)
	}
}

func TestWriteQuickPouch_T544NativeClearRestoreContract(t *testing.T) {
	save := loadTaskArtifact(t, "task-544", "01-native-equipment-baseline.sl2")
	slot := &save.Slots[0]
	before := append([]byte(nil), slot.Data...)
	raw, err := slot.ReadEquippedState()
	if err != nil {
		t.Fatal(err)
	}
	quickHandle := raw.QuickItems[0].ItemID
	pouchHandle := raw.Pouch[0].ItemID
	if quickHandle == 0 || quickHandle == GaHandleInvalid || pouchHandle == 0 || pouchHandle == GaHandleInvalid {
		t.Fatalf("T544 precondition failed: quick=0x%08X pouch=0x%08X", quickHandle, pouchHandle)
	}

	if err := slot.WriteQuickPouch([]QuickPouchWrite{
		{Slot: QuickPouchSlotQuick1, Handle: 0},
		{Slot: QuickPouchSlotPouch1, Handle: 0},
	}); err != nil {
		t.Fatal(err)
	}
	if err := slot.WriteQuickPouch([]QuickPouchWrite{
		{Slot: QuickPouchSlotQuick1, Handle: quickHandle},
		{Slot: QuickPouchSlotPouch1, Handle: pouchHandle},
	}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(slot.Data, before) {
		t.Error("clear then restore did not reproduce the byte-exact native T544 Quick Item / Pouch state")
	}
}
