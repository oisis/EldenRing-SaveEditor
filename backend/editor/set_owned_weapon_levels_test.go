package editor

import "testing"

// batchSnap builds a mixed inventory + storage snapshot covering every
// branch of SetOwnedWeaponLevels: a standard weapon, an infused standard
// weapon, a somber weapon, a somber shield, a non-upgradeable weapon,
// a non-weapon, and a standard weapon parked in STORAGE (which must never
// be touched).
func batchSnap() *InventoryWorkspaceSnapshot {
	return &InventoryWorkspaceSnapshot{
		SessionID:      "ses-batch",
		CharacterIndex: 0,
		InventoryItems: []EditableItem{
			{
				UID: "hnd:0x80800001", Source: ItemSourceOriginal, Container: ContainerInventory,
				OriginalHandle: 0x80800001, ItemID: 0x00100000, BaseItemID: 0x00100000,
				Name: "Standard Sword", Category: "melee_armaments", Quantity: 1,
				CurrentUpgrade: 0, MaxUpgrade: 25, IsWeapon: true,
			},
			{
				// Infused (Heavy, offset 100) standard weapon at +20 — SET must
				// preserve the infusion and re-encode ItemID from the base.
				UID: "hnd:0x80800002", Source: ItemSourceOriginal, Container: ContainerInventory,
				OriginalHandle: 0x80800002, ItemID: 0x00100000 + 100 + 20, BaseItemID: 0x00100000,
				Name: "Heavy Sword", Category: "melee_armaments", Quantity: 1,
				CurrentUpgrade: 20, MaxUpgrade: 25, InfusionName: "Heavy", IsWeapon: true,
			},
			{
				UID: "hnd:0x80800003", Source: ItemSourceOriginal, Container: ContainerInventory,
				OriginalHandle: 0x80800003, ItemID: 0x00200003, BaseItemID: 0x00200000,
				Name: "Somber Katana", Category: "melee_armaments", Quantity: 1,
				CurrentUpgrade: 3, MaxUpgrade: 10, IsWeapon: true,
			},
			{
				UID: "hnd:0x80800004", Source: ItemSourceOriginal, Container: ContainerInventory,
				OriginalHandle: 0x80800004, ItemID: 0x00300000, BaseItemID: 0x00300000,
				Name: "Somber Shield", Category: "shields", Quantity: 1,
				CurrentUpgrade: 0, MaxUpgrade: 10, IsWeapon: true,
			},
			{
				// Non-upgradeable weapon (torch-like, MaxUpgrade 0) — skipped.
				UID: "hnd:0x80800005", Source: ItemSourceOriginal, Container: ContainerInventory,
				OriginalHandle: 0x80800005, ItemID: 0x00400000, BaseItemID: 0x00400000,
				Name: "Torch", Category: "melee_armaments", Quantity: 1,
				CurrentUpgrade: 0, MaxUpgrade: 0, IsWeapon: true,
			},
			{
				// Non-weapon (armor) — skipped.
				UID: "hnd:0x90800006", Source: ItemSourceOriginal, Container: ContainerInventory,
				OriginalHandle: 0x90800006, ItemID: 0x100249F0, BaseItemID: 0x100249F0,
				Name: "Iron Kasa", Category: "head", Quantity: 1, IsArmor: true,
			},
		},
		StorageItems: []EditableItem{
			{
				// Standard weapon in STORAGE — must stay untouched.
				UID: "hnd:0x80800007", Source: ItemSourceOriginal, Container: ContainerStorage,
				OriginalHandle: 0x80800007, ItemID: 0x00500002, BaseItemID: 0x00500000,
				Name: "Stored Sword", Category: "melee_armaments", Quantity: 1,
				CurrentUpgrade: 2, MaxUpgrade: 25, IsWeapon: true,
			},
		},
		UnsupportedInventoryRecords: []RawInventoryRecord{},
		UnsupportedStorageRecords:   []RawInventoryRecord{},
	}
}

func find(t *testing.T, items []EditableItem, uid string) EditableItem {
	t.Helper()
	for _, it := range items {
		if it.UID == uid {
			return it
		}
	}
	t.Fatalf("item %s not found", uid)
	return EditableItem{}
}

func TestSetOwnedWeaponLevels_SetsAndSkips(t *testing.T) {
	snap := batchSnap()
	changed, err := SetOwnedWeaponLevels(snap, 25, 10)
	if err != nil {
		t.Fatalf("SetOwnedWeaponLevels: %v", err)
	}
	// Standard sword, infused sword, somber katana, somber shield = 4.
	if changed != 4 {
		t.Fatalf("changed = %d, want 4", changed)
	}

	if it := find(t, snap.InventoryItems, "hnd:0x80800001"); it.CurrentUpgrade != 25 {
		t.Errorf("standard weapon = +%d, want +25", it.CurrentUpgrade)
	}
	if it := find(t, snap.InventoryItems, "hnd:0x80800003"); it.CurrentUpgrade != 10 {
		t.Errorf("somber weapon = +%d, want +10", it.CurrentUpgrade)
	}
	if it := find(t, snap.InventoryItems, "hnd:0x80800004"); it.CurrentUpgrade != 10 {
		t.Errorf("somber shield = +%d, want +10", it.CurrentUpgrade)
	}

	// Infusion preserved and ItemID re-encoded from base.
	inf := find(t, snap.InventoryItems, "hnd:0x80800002")
	if inf.InfusionName != "Heavy" {
		t.Errorf("infusion = %q, want Heavy", inf.InfusionName)
	}
	if inf.CurrentUpgrade != 25 {
		t.Errorf("infused weapon = +%d, want +25", inf.CurrentUpgrade)
	}
	if want := uint32(0x00100000 + 100 + 25); inf.ItemID != want {
		t.Errorf("infused ItemID = 0x%08X, want 0x%08X", inf.ItemID, want)
	}

	// Skipped: MaxUpgrade 0 weapon, armor, and the STORAGE weapon.
	if it := find(t, snap.InventoryItems, "hnd:0x80800005"); it.CurrentUpgrade != 0 {
		t.Errorf("MaxUpgrade=0 weapon changed to +%d", it.CurrentUpgrade)
	}
	if it := find(t, snap.InventoryItems, "hnd:0x90800006"); it.CurrentUpgrade != 0 {
		t.Errorf("armor changed to +%d", it.CurrentUpgrade)
	}
	if it := find(t, snap.StorageItems, "hnd:0x80800007"); it.CurrentUpgrade != 2 {
		t.Errorf("STORAGE weapon changed to +%d, want +2 (untouched)", it.CurrentUpgrade)
	}
}

func TestSetOwnedWeaponLevels_Downgrades(t *testing.T) {
	snap := batchSnap()
	// Standard sword starts at +0 via batchSnap; push it up first, then SET low.
	if _, err := SetOwnedWeaponLevels(snap, 25, 10); err != nil {
		t.Fatalf("prime: %v", err)
	}
	changed, err := SetOwnedWeaponLevels(snap, 3, 1)
	if err != nil {
		t.Fatalf("downgrade: %v", err)
	}
	if changed != 4 {
		t.Fatalf("changed = %d, want 4", changed)
	}
	if it := find(t, snap.InventoryItems, "hnd:0x80800001"); it.CurrentUpgrade != 3 {
		t.Errorf("standard weapon = +%d, want +3 (downgraded)", it.CurrentUpgrade)
	}
	if it := find(t, snap.InventoryItems, "hnd:0x80800003"); it.CurrentUpgrade != 1 {
		t.Errorf("somber weapon = +%d, want +1 (downgraded)", it.CurrentUpgrade)
	}
}

func TestSetOwnedWeaponLevels_Clamps(t *testing.T) {
	snap := batchSnap()
	changed, err := SetOwnedWeaponLevels(snap, 99, 99)
	if err != nil {
		t.Fatalf("SetOwnedWeaponLevels: %v", err)
	}
	if changed != 4 {
		t.Fatalf("changed = %d, want 4", changed)
	}
	if it := find(t, snap.InventoryItems, "hnd:0x80800001"); it.CurrentUpgrade != 25 {
		t.Errorf("standard clamp = +%d, want +25", it.CurrentUpgrade)
	}
	if it := find(t, snap.InventoryItems, "hnd:0x80800003"); it.CurrentUpgrade != 10 {
		t.Errorf("somber clamp = +%d, want +10", it.CurrentUpgrade)
	}
}

func TestSetOwnedWeaponLevels_AllAtTargetNoChange(t *testing.T) {
	snap := &InventoryWorkspaceSnapshot{
		SessionID: "ses-attarget",
		InventoryItems: []EditableItem{
			{
				UID: "s25", Container: ContainerInventory, ItemID: 0x00100019, BaseItemID: 0x00100000,
				Name: "Maxed Sword", Category: "melee_armaments", Quantity: 1,
				CurrentUpgrade: 25, MaxUpgrade: 25, IsWeapon: true,
			},
			{
				UID: "s10", Container: ContainerInventory, ItemID: 0x0020000A, BaseItemID: 0x00200000,
				Name: "Maxed Katana", Category: "melee_armaments", Quantity: 1,
				CurrentUpgrade: 10, MaxUpgrade: 10, IsWeapon: true,
			},
		},
	}
	changed, err := SetOwnedWeaponLevels(snap, 25, 10)
	if err != nil {
		t.Fatalf("SetOwnedWeaponLevels: %v", err)
	}
	if changed != 0 {
		t.Errorf("changed = %d, want 0 (all already at target)", changed)
	}
	if snap.Dirty {
		t.Error("snapshot marked Dirty by a no-op batch")
	}
	for _, it := range snap.InventoryItems {
		if it.HasPendingWeaponPatch {
			t.Errorf("%s got a pending weapon patch on a no-op", it.Name)
		}
	}
}

func TestSetOwnedWeaponLevels_MixedOnlyChangedCounted(t *testing.T) {
	snap := &InventoryWorkspaceSnapshot{
		SessionID: "ses-mixed",
		InventoryItems: []EditableItem{
			{
				// Already at target 25 → skipped.
				UID: "at25", Container: ContainerInventory, ItemID: 0x00100019, BaseItemID: 0x00100000,
				Name: "Already Maxed", Category: "melee_armaments", Quantity: 1,
				CurrentUpgrade: 25, MaxUpgrade: 25, IsWeapon: true,
			},
			{
				// Below target 10 → changed.
				UID: "below10", Container: ContainerInventory, ItemID: 0x00200003, BaseItemID: 0x00200000,
				Name: "Somber", Category: "melee_armaments", Quantity: 1,
				CurrentUpgrade: 3, MaxUpgrade: 10, IsWeapon: true,
			},
		},
	}
	changed, err := SetOwnedWeaponLevels(snap, 25, 10)
	if err != nil {
		t.Fatalf("SetOwnedWeaponLevels: %v", err)
	}
	if changed != 1 {
		t.Fatalf("changed = %d, want 1 (only the below-target weapon)", changed)
	}
	if it := find(t, snap.InventoryItems, "at25"); it.HasPendingWeaponPatch {
		t.Error("already-at-target weapon got a pending patch")
	}
	if it := find(t, snap.InventoryItems, "below10"); it.CurrentUpgrade != 10 || !it.HasPendingWeaponPatch {
		t.Errorf("changed weapon = +%d pending=%v, want +10 pending=true", it.CurrentUpgrade, it.HasPendingWeaponPatch)
	}
}

func TestSetOwnedWeaponLevels_NoWeaponsNoChange(t *testing.T) {
	snap := &InventoryWorkspaceSnapshot{
		SessionID: "ses-empty",
		InventoryItems: []EditableItem{
			{UID: "a", Category: "head", MaxUpgrade: 0, IsArmor: true},
		},
	}
	changed, err := SetOwnedWeaponLevels(snap, 25, 10)
	if err != nil {
		t.Fatalf("SetOwnedWeaponLevels: %v", err)
	}
	if changed != 0 {
		t.Errorf("changed = %d, want 0", changed)
	}
}
