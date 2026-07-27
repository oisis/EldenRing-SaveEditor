package vm

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/db"
)

func TestMapParsedSlotToVM_ExposesTechnicalStackLimits(t *testing.T) {
	slot := &core.SaveSlot{Version: 1, GaMap: make(map[uint32]uint32)}
	slot.Inventory.CommonItems = []core.InventoryItem{
		{GaItemHandle: 0xB0000096, Quantity: 3, Index: 1}, // Furlcalling Finger Remedy
		{GaItemHandle: 0xB0000FA0, Quantity: 3, Index: 2}, // Glintstone Pebble
		{GaItemHandle: 0xB00318F8, Quantity: 1, Index: 3}, // Fanged Imp Ashes
	}

	got, err := MapParsedSlotToVM(slot)
	if err != nil {
		t.Fatalf("MapParsedSlotToVM: %v", err)
	}
	if len(got.Inventory) != 3 {
		t.Fatalf("Inventory rows = %d, want 3", len(got.Inventory))
	}

	byID := make(map[uint32]ItemViewModel)
	for _, item := range got.Inventory {
		byID[item.ID] = item
	}
	remedy := byID[0x40000096]
	if remedy.RecordMode != db.ItemRecordModeQuantityStack {
		t.Errorf("Furlcalling Finger Remedy RecordMode = %q", remedy.RecordMode)
	}
	if !remedy.GameMaxInventoryKnown || !remedy.GameMaxStorageKnown || remedy.GameMaxInventory != 999 || remedy.GameMaxStorage != 999 {
		t.Errorf("Furlcalling Finger Remedy technical caps = %d/%d (known %v/%v), want 999/999 known",
			remedy.GameMaxInventory, remedy.GameMaxStorage, remedy.GameMaxInventoryKnown, remedy.GameMaxStorageKnown)
	}

	spell := byID[0x40000FA0]
	if spell.RecordMode != db.ItemRecordModeQuantityStack {
		t.Errorf("spell RecordMode = %q", spell.RecordMode)
	}
	if !spell.GameMaxInventoryKnown || !spell.GameMaxStorageKnown || spell.GameMaxInventory != 99 || spell.GameMaxStorage != 600 {
		t.Errorf("spell technical caps = %d/%d (known %v/%v), want 99/600 known",
			spell.GameMaxInventory, spell.GameMaxStorage, spell.GameMaxInventoryKnown, spell.GameMaxStorageKnown)
	}

	ash := byID[0x400318F8]
	if ash.RecordMode != db.ItemRecordModeQuantityStack {
		t.Errorf("spirit ash RecordMode = %q", ash.RecordMode)
	}
	if !ash.GameMaxInventoryKnown || !ash.GameMaxStorageKnown || ash.GameMaxInventory != 1 || ash.GameMaxStorage != 600 {
		t.Errorf("spirit ash technical caps = %d/%d (known %v/%v), want 1/600 known",
			ash.GameMaxInventory, ash.GameMaxStorage, ash.GameMaxInventoryKnown, ash.GameMaxStorageKnown)
	}
}

func TestMapParsedSlotToVM_MarksEquippedAoWCopy(t *testing.T) {
	const (
		aowItemID    = uint32(0x80002710)
		aowHandle    = uint32(0xC0800001)
		weaponItemID = uint32(0x003085E0)
		weaponHandle = uint32(0x80800002)
	)
	slot := &core.SaveSlot{
		Version: 1,
		GaMap: map[uint32]uint32{
			aowHandle:    aowItemID,
			weaponHandle: weaponItemID,
		},
		GaItems: []core.GaItemFull{
			{Handle: aowHandle, ItemID: aowItemID},
			{Handle: weaponHandle, ItemID: weaponItemID, AoWGaItemHandle: aowHandle},
		},
	}
	slot.Inventory.CommonItems = []core.InventoryItem{
		{GaItemHandle: weaponHandle, Quantity: 1, Index: 1},
	}

	got, err := MapParsedSlotToVM(slot)
	if err != nil {
		t.Fatalf("MapParsedSlotToVM: %v", err)
	}
	if len(got.Inventory) != 1 {
		t.Fatalf("Inventory rows = %d, want only the weapon container record", len(got.Inventory))
	}
	if len(got.AttachedItems) != 1 {
		t.Fatalf("AttachedItems rows = %d, want one attached AoW", len(got.AttachedItems))
	}
	item := got.AttachedItems[0]
	if item.Handle == 0 {
		t.Fatal("equipped AoW copy missing from attached-items VM")
	}
	if item.RecordMode != db.ItemRecordModeSeparateInstances {
		t.Errorf("RecordMode = %q", item.RecordMode)
	}
	if !item.IsEquippedAoW || item.EquippedByWeapon != weaponHandle {
		t.Errorf("equipped state = %v by 0x%08X, want true by 0x%08X",
			item.IsEquippedAoW, item.EquippedByWeapon, weaponHandle)
	}
	if item.EquippedByWeaponName != "Claymore" {
		t.Errorf("equipped weapon name = %q, want Claymore", item.EquippedByWeaponName)
	}
	if !item.ReadOnly {
		t.Error("equipped AoW copy must be read-only")
	}
}

func TestMapParsedSlotToVM_AttachedAoWDoesNotInheritStoredWeaponContainer(t *testing.T) {
	const (
		aowItemID    = uint32(0x80002710)
		aowHandle    = uint32(0xC0800001)
		weaponItemID = uint32(0x003085E0)
		weaponHandle = uint32(0x80800002)
	)
	slot := &core.SaveSlot{
		Version: 1,
		GaMap: map[uint32]uint32{
			aowHandle:    aowItemID,
			weaponHandle: weaponItemID,
		},
		GaItems: []core.GaItemFull{
			{Handle: aowHandle, ItemID: aowItemID},
			{Handle: weaponHandle, ItemID: weaponItemID, AoWGaItemHandle: aowHandle},
		},
	}
	slot.Storage.CommonItems = []core.InventoryItem{
		{GaItemHandle: weaponHandle, Quantity: 1, Index: 1},
	}

	got, err := MapParsedSlotToVM(slot)
	if err != nil {
		t.Fatalf("MapParsedSlotToVM: %v", err)
	}
	if len(got.Storage) != 1 || got.Storage[0].Handle != weaponHandle {
		t.Fatalf("Storage = %+v, want only the weapon record", got.Storage)
	}
	if len(got.AttachedItems) != 1 || got.AttachedItems[0].Handle != aowHandle {
		t.Fatalf("AttachedItems = %+v, want attached AoW 0x%08X", got.AttachedItems, aowHandle)
	}
	if got.AttachedItems[0].EquippedByWeaponName != "Claymore" {
		t.Errorf("attached AoW owner = %q, want Claymore", got.AttachedItems[0].EquippedByWeaponName)
	}
}
