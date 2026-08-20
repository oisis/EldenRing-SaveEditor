package core

import (
	"testing"
)

func TestComputeWeaponMatchmakingLevel(t *testing.T) {
	// Standard weapon: Dagger (base ID 0x000F4240, MaxUpgrade=25, melee_armaments)
	const daggerBase uint32 = 0x000F4240
	// Somber weapon: Moonveil (base ID 0x008A3EA0, MaxUpgrade=10, melee_armaments)
	const moonveilBase uint32 = 0x008A3EA0
	// Somber shield: Coil Shield (base ID 0x01CCD0C0, MaxUpgrade=10, shields)
	const coilShieldBase uint32 = 0x01CCD0C0
	// Somber bow: Harp Bow (base ID 0x0262CF30, MaxUpgrade=10, ranged_and_catalysts)
	const harpBowBase uint32 = 0x0262CF30
	// Spirit Ash: Mimic Tear +10 (base ID 0x40030530, category "ashes", MaxUpgrade=10)
	const mimicTearBase uint32 = 0x40030530

	// 1. Standard weapon +0..+25
	if got := ComputeWeaponMatchmakingLevel(daggerBase); got != 0 {
		t.Errorf("Dagger +0: got %d, want 0", got)
	}
	if got := ComputeWeaponMatchmakingLevel(daggerBase + 10); got != 10 {
		t.Errorf("Dagger +10: got %d, want 10", got)
	}
	if got := ComputeWeaponMatchmakingLevel(daggerBase + 25); got != 25 {
		t.Errorf("Dagger +25: got %d, want 25", got)
	}
	// Heavy Dagger +25 (infusion offset 100)
	if got := ComputeWeaponMatchmakingLevel(daggerBase + 100 + 25); got != 25 {
		t.Errorf("Heavy Dagger +25: got %d, want 25", got)
	}

	// 2. Somber scale mapping: 0, 0, 5, 7, 10, 12, 15, 17, 20, 24, 25
	expectedSomber := []uint8{0, 0, 5, 7, 10, 12, 15, 17, 20, 24, 25}
	for lvl, want := range expectedSomber {
		if got := ComputeWeaponMatchmakingLevel(moonveilBase + uint32(lvl)); got != want {
			t.Errorf("Moonveil +%d: got %d, want %d", lvl, got, want)
		}
	}
	if got := ComputeWeaponMatchmakingLevel(coilShieldBase + 10); got != 25 {
		t.Errorf("Coil Shield +10: got %d, want 25", got)
	}
	if got := ComputeWeaponMatchmakingLevel(harpBowBase + 10); got != 25 {
		t.Errorf("Harp Bow +10: got %d, want 25", got)
	}

	// 3. Spirit Ashes / non-weapons must be ignored (return 0) even with MaxUpgrade=10
	if got := ComputeWeaponMatchmakingLevel(mimicTearBase + 10); got != 0 {
		t.Errorf("Mimic Tear Ash +10: got %d, want 0 (must ignore ashes)", got)
	}
	// Talisman: Crimson Amber Medallion (0x20000000)
	if got := ComputeWeaponMatchmakingLevel(0x20000000); got != 0 {
		t.Errorf("Talisman: got %d, want 0", got)
	}
	// Armor: Iron Helmet (0x10009C40)
	if got := ComputeWeaponMatchmakingLevel(0x10009C40); got != 0 {
		t.Errorf("Armor: got %d, want 0", got)
	}
}

func TestSyncMatchmakingWeaponLevel_MonotonicityAndStorage(t *testing.T) {
	slotData := make([]byte, SlotSize)
	const magicOff = 0x155D0
	targetOff := magicOff + OffMatchmakingWeaponLevel

	slot := &SaveSlot{
		Data:        slotData,
		MagicOffset: magicOff,
		GaMap:       make(map[uint32]uint32),
	}

	// Initial matchmaking level is 15
	slot.Data[targetOff] = 15

	// Add weapon with matchmaking level 10 (Dagger +10) in Inventory
	const daggerBase uint32 = 0x000F4240
	slot.Inventory.CommonItems = []InventoryItem{
		{GaItemHandle: 0x80000001, Quantity: 1, Index: 100},
	}
	slot.GaMap[0x80000001] = daggerBase + 10

	// Sync should NOT lower the level from 15 to 10
	if err := SyncMatchmakingWeaponLevel(slot); err != nil {
		t.Fatalf("SyncMatchmakingWeaponLevel: %v", err)
	}
	if got := slot.Data[targetOff]; got != 15 {
		t.Errorf("Monotonicity check failed: got %d, want 15 (must not lower)", got)
	}

	// Now add Somber weapon +10 in Storage (Moonveil +10 -> matchmaking level 25)
	const moonveilBase uint32 = 0x008A3EA0
	slot.Storage.CommonItems = []InventoryItem{
		{GaItemHandle: 0x80000002, Quantity: 1, Index: 200},
	}
	slot.GaMap[0x80000002] = moonveilBase + 10

	// Sync should raise the level from 15 to 25
	if err := SyncMatchmakingWeaponLevel(slot); err != nil {
		t.Fatalf("SyncMatchmakingWeaponLevel: %v", err)
	}
	if got := slot.Data[targetOff]; got != 25 {
		t.Errorf("Storage weapon check failed: got %d, want 25", got)
	}
}

func TestSyncMatchmakingWeaponLevel_FailClosed(t *testing.T) {
	// 1. Nil slot
	if err := SyncMatchmakingWeaponLevel(nil); err == nil {
		t.Error("expected error for nil slot, got nil")
	}

	// 2. Empty data
	if err := SyncMatchmakingWeaponLevel(&SaveSlot{MagicOffset: 0x155D0}); err == nil {
		t.Error("expected error for empty slot data, got nil")
	}

	// 3. Invalid MagicOffset (< MinMagicOffset)
	slot := &SaveSlot{
		Data:        make([]byte, SlotSize),
		MagicOffset: 100, // less than MinMagicOffset (400)
	}
	if err := SyncMatchmakingWeaponLevel(slot); err == nil {
		t.Error("expected error for MagicOffset < MinMagicOffset, got nil")
	}
}
