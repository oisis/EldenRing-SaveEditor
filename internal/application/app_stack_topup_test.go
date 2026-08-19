package application

import (
	"encoding/binary"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
)

// fanDaggersID is a plain stackable goods item (catalog caps: MaxInventory 40,
// MaxStorage 600) — the item Test 3 phase 3 surfaced the stack-replacement bug on.
const fanDaggersID = uint32(0x400006D6)

// stackTopUpApp builds a real, byte-accurate empty single-slot save through
// the public core.SaveSlot.Read path (same approach as
// emptyStorageDatabaseAddApp) and seeds a stack of `seed` Fan Daggers into the
// requested container using the ordinary Database-Add endpoint, so the
// starting state is one the application itself can produce.
func stackTopUpApp(t *testing.T, seedInv, seedStorage int) *App {
	t.Helper()

	data := make([]byte, core.SlotSize)
	binary.LittleEndian.PutUint32(data, core.GaItemVersionBreak+1)
	magicOffset := core.GaItemsStart + core.DynPlayerData - 1
	copy(data[magicOffset:], core.MagicPattern)

	var slot core.SaveSlot
	if err := slot.Read(core.NewReader(data), string(core.PlatformPC)); err != nil {
		t.Fatalf("SaveSlot.Read: %v", err)
	}

	app := NewApp()
	app.save = &core.SaveFile{Platform: core.PlatformPC}
	app.save.UserData10.Data = make([]byte, 0x60000)
	app.save.Slots[0] = slot

	if seedInv > 0 || seedStorage > 0 {
		if _, err := app.AddItemsToCharacter(0, []uint32{fanDaggersID}, 0, 0, 0, 0, seedInv, seedStorage); err != nil {
			t.Fatalf("seeding the stack: %v", err)
		}
	}
	return app
}

func stackQty(t *testing.T, app *App, storage bool) uint32 {
	t.Helper()
	items := app.save.Slots[0].Inventory.CommonItems
	where := "Inventory"
	if storage {
		items = app.save.Slots[0].Storage.CommonItems
		where = "Storage"
	}
	total, records := uint32(0), 0
	for _, it := range items {
		if it.GaItemHandle == 0xB00006D6 {
			total += it.Quantity
			records++
		}
	}
	if records == 0 {
		t.Fatalf("no %s record for Fan Daggers", where)
	}
	if records > 1 {
		t.Fatalf("%s holds %d Fan Daggers records, want 1 — a stackable top-up must never split", where, records)
	}
	return total
}

// TestAddItemsToCharacter_StorageStackIsToppedUpNotReplaced is the regression
// for the silent data loss found in Test 3 phase 3: adding 5 Fan Daggers to a
// Storage stack of 586 rewrote the record to 5 instead of raising it to 591.
// core.addToInventory takes qty as the TARGET TOTAL for a stackable record, so
// the caller must fold in the quantity already held.
func TestAddItemsToCharacter_StorageStackIsToppedUpNotReplaced(t *testing.T) {
	app := stackTopUpApp(t, 0, 586)
	if got := stackQty(t, app, true); got != 586 {
		t.Fatalf("seed failed: Storage stack = %d, want 586", got)
	}

	if _, err := app.AddItemsToCharacter(0, []uint32{fanDaggersID}, 0, 0, 0, 0, 0, 5); err != nil {
		t.Fatalf("AddItemsToCharacter: %v", err)
	}

	if got := stackQty(t, app, true); got != 591 {
		t.Fatalf("Storage stack = %d, want 591 (586 held + 5 added)", got)
	}
}

// The same invariant on the Inventory side — the caller path is shared, so a
// fix that only handled Storage would still lose held quantity.
func TestAddItemsToCharacter_InventoryStackIsToppedUpNotReplaced(t *testing.T) {
	app := stackTopUpApp(t, 10, 0)
	if got := stackQty(t, app, false); got != 10 {
		t.Fatalf("seed failed: Inventory stack = %d, want 10", got)
	}

	if _, err := app.AddItemsToCharacter(0, []uint32{fanDaggersID}, 0, 0, 0, 0, 5, 0); err != nil {
		t.Fatalf("AddItemsToCharacter: %v", err)
	}

	if got := stackQty(t, app, false); got != 15 {
		t.Fatalf("Inventory stack = %d, want 15 (10 held + 5 added)", got)
	}
}

// Boundary: topping up past the cap saturates instead of overflowing. This is
// exactly what the original "SET quantity, don't ADD" rule in writer.go was
// protecting against, and it must keep holding after the fix.
func TestAddItemsToCharacter_StackTopUpSaturatesAtCap(t *testing.T) {
	app := stackTopUpApp(t, 0, 598)

	if _, err := app.AddItemsToCharacter(0, []uint32{fanDaggersID}, 0, 0, 0, 0, 0, 50); err != nil {
		t.Fatalf("AddItemsToCharacter: %v", err)
	}

	if got := stackQty(t, app, true); got != 600 {
		t.Fatalf("Storage stack = %d, want 600 (MaxStorage cap, not 648)", got)
	}
}

// Boundary: a stack already at the cap is skipped and left untouched — the
// pre-existing guard this fix must not disturb.
func TestAddItemsToCharacter_StackAtCapIsSkippedUnchanged(t *testing.T) {
	app := stackTopUpApp(t, 0, 600)

	result, err := app.AddItemsToCharacter(0, []uint32{fanDaggersID}, 0, 0, 0, 0, 0, 5)
	if err != nil {
		t.Fatalf("AddItemsToCharacter: %v", err)
	}
	if got := stackQty(t, app, true); got != 600 {
		t.Fatalf("Storage stack = %d, want 600 unchanged", got)
	}
	if result.Added != 0 {
		t.Fatalf("Added = %d, want 0 for an at-cap stack", result.Added)
	}
}

// A stackable item NOT yet present must be written with exactly the requested
// quantity — the top-up must never inflate a fresh record.
func TestAddItemsToCharacter_NewStackUsesRequestedQtyOnly(t *testing.T) {
	app := stackTopUpApp(t, 0, 0)

	if _, err := app.AddItemsToCharacter(0, []uint32{fanDaggersID}, 0, 0, 0, 0, 0, 5); err != nil {
		t.Fatalf("AddItemsToCharacter: %v", err)
	}

	if got := stackQty(t, app, true); got != 5 {
		t.Fatalf("new Storage stack = %d, want exactly 5", got)
	}
}
