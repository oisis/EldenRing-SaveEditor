package application

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/editor"
)

func TestAddItemsToCharacter_RaisesMatchmakingLevel(t *testing.T) {
	app := diagnosticGaItemApp(t)
	slot := &app.save.Slots[0]

	targetOff := slot.MagicOffset + core.OffMatchmakingWeaponLevel
	slot.Data[targetOff] = 4 // start at level 4

	// Add Dagger +25 (base 0x000F4240, regular weapon)
	const daggerBase uint32 = 0x000F4240
	res, err := app.AddItemsToCharacter(0, []uint32{daggerBase}, 25, 0, 0, 0, 1, 0)
	if err != nil {
		t.Fatalf("AddItemsToCharacter: %v", err)
	}
	if res.Added == 0 {
		t.Fatalf("AddItemsToCharacter added 0 items (CapHit=%q)", res.CapHit)
	}

	targetOff = slot.MagicOffset + core.OffMatchmakingWeaponLevel
	if got := slot.Data[targetOff]; got != 25 {
		t.Errorf("matchmaking level after adding Dagger +25: got %d, want 25", got)
	}
}

func TestSaveInventoryWorkspaceChanges_RaisesMatchmakingLevel_Somber(t *testing.T) {
	app := diagnosticGaItemApp(t)
	slot := &app.save.Slots[0]

	targetOff := slot.MagicOffset + core.OffMatchmakingWeaponLevel
	slot.Data[targetOff] = 4 // start at level 4

	snap, err := app.StartInventoryEditSession(0)
	if err != nil {
		t.Fatalf("StartInventoryEditSession: %v", err)
	}

	// Add a Somber weapon to workspace (Moonveil, base 0x008A3EA0)
	const moonveilBase uint32 = 0x008A3EA0
	snap, err = app.AddInventoryWorkspaceItem(snap.SessionID, editor.AddItemSpec{ItemID: moonveilBase}, "inventory", 0)
	if err != nil {
		t.Fatalf("AddInventoryWorkspaceItem: %v", err)
	}

	// Find the added item UID
	addedUID := ""
	for _, it := range snap.InventoryItems {
		if it.BaseItemID == moonveilBase {
			addedUID = it.UID
			break
		}
	}
	if addedUID == "" {
		t.Fatalf("added Moonveil not found in workspace items")
	}

	// Upgrade Moonveil to +10 (Somber +10 -> matchmaking level 25)
	snap, err = app.UpdateInventoryWorkspaceWeapon(snap.SessionID, addedUID, editor.WeaponPatch{
		SetUpgrade: true,
		Upgrade:    10,
	})
	if err != nil {
		t.Fatalf("UpdateInventoryWorkspaceWeapon: %v", err)
	}

	// Commit workspace save
	_, err = app.SaveInventoryWorkspaceChanges(snap.SessionID)
	if err != nil {
		t.Fatalf("SaveInventoryWorkspaceChanges: %v", err)
	}

	targetOff = slot.MagicOffset + core.OffMatchmakingWeaponLevel
	if got := slot.Data[targetOff]; got != 25 {
		t.Errorf("matchmaking level after SaveInventoryWorkspaceChanges with Moonveil +10: got %d, want 25", got)
	}
}
