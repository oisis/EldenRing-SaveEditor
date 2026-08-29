package application

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/db"
	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

// Spectral Steed Attire (Regulation 1.17) coverage. The confirmed PC contract is
// that flags 6700-6703 are mutually exclusive; the getter must never repair an
// ambiguous save and the setter must never mutate after a failed validation.

const (
	treeSentinelAttireItem = uint32(0x401EAA00)
	silverOfCariaItem      = uint32(0x401EAA0A)
	funerealNightItem      = uint32(0x401EAA14)
)

// spectralSteedApp builds a PC save with a zeroed event-flags region large
// enough for 6700-6703.
func spectralSteedApp(t *testing.T) *App {
	t.Helper()
	app := gameItemAddApp(false)
	app.save.Platform = "PC"
	withUnlockEventFlags(app, 4096)
	return app
}

func spectralSteedFlags(t *testing.T, app *App) []byte {
	t.Helper()
	slot := &app.save.Slots[0]
	return slot.Data[slot.EventFlagsOffset:]
}

func setSpectralSteedFlag(t *testing.T, app *App, id uint32, value bool) {
	t.Helper()
	if err := db.SetEventFlag(spectralSteedFlags(t, app), id, value); err != nil {
		t.Fatalf("seed flag %d: %v", id, err)
	}
}

// giveAttireItem puts the attire key item into the character's KeyItems section
// using the editor-computed handle.
func giveAttireItem(app *App, itemID uint32) {
	slot := &app.save.Slots[0]
	handle := (itemID & 0x0FFFFFFF) | db.ItemIDToHandlePrefix(itemID)
	slot.Inventory.KeyItems = append(slot.Inventory.KeyItems, core.InventoryItem{
		GaItemHandle: handle,
		Quantity:     1,
	})
}

func addAttireItems(t *testing.T, app *App, itemIDs ...uint32) {
	t.Helper()
	if err := core.AddItemsToSlot(&app.save.Slots[0], itemIDs, 1, 0, false); err != nil {
		t.Fatalf("add attire items: %v", err)
	}
}

func spectralSteedFlagStates(t *testing.T, app *App) map[uint32]bool {
	t.Helper()
	flags := spectralSteedFlags(t, app)
	states := map[uint32]bool{}
	for _, a := range data.SpectralSteedAttires {
		on, err := db.GetEventFlag(flags, a.FlagID)
		if err != nil {
			t.Fatalf("read flag %d: %v", a.FlagID, err)
		}
		states[a.FlagID] = on
	}
	return states
}

func TestGetSpectralSteedAttireResolvesEachFlag(t *testing.T) {
	for _, want := range []uint32{6700, 6701, 6702, 6703} {
		app := spectralSteedApp(t)
		setSpectralSteedFlag(t, app, want, true)

		state, err := app.GetSpectralSteedAttire(0)
		if err != nil {
			t.Fatalf("flag %d: %v", want, err)
		}
		if state.Status != db.SpectralSteedAttireResolved {
			t.Fatalf("flag %d: status = %q, want resolved", want, state.Status)
		}
		if state.ActiveID != want {
			t.Fatalf("flag %d: activeID = %d", want, state.ActiveID)
		}
		if len(state.Entries) != 4 {
			t.Fatalf("flag %d: got %d entries, want 4", want, len(state.Entries))
		}
	}
}

func TestGetSpectralSteedAttireLegacyWhenAllFlagsClear(t *testing.T) {
	app := spectralSteedApp(t)

	state, err := app.GetSpectralSteedAttire(0)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != db.SpectralSteedAttireLegacy {
		t.Fatalf("status = %q, want legacy", state.Status)
	}
	if state.ActiveID != 0 {
		t.Fatalf("activeID = %d, want 0 — the getter must not guess", state.ActiveID)
	}
	// Default Appearance stays offered even in the legacy state.
	if state.Entries[0].ID != data.SpectralSteedAttireDefaultFlag || !state.Entries[0].Owned {
		t.Fatalf("default entry = %+v, want flag 6700 available", state.Entries[0])
	}
	for _, s := range spectralSteedFlagStates(t, app) {
		if s {
			t.Fatal("getter mutated an event flag")
		}
	}
}

func TestGetSpectralSteedAttireConflictWhenTwoFlagsSet(t *testing.T) {
	app := spectralSteedApp(t)
	setSpectralSteedFlag(t, app, 6701, true)
	setSpectralSteedFlag(t, app, 6703, true)

	state, err := app.GetSpectralSteedAttire(0)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != db.SpectralSteedAttireConflict {
		t.Fatalf("status = %q, want conflict", state.Status)
	}
	if state.ActiveID != 0 {
		t.Fatalf("activeID = %d, want 0", state.ActiveID)
	}
	states := spectralSteedFlagStates(t, app)
	if !states[6701] || !states[6703] || states[6700] || states[6702] {
		t.Fatalf("getter mutated flags: %v", states)
	}
}

func TestGetSpectralSteedAttireOwnershipFollowsInventoryItem(t *testing.T) {
	app := spectralSteedApp(t)
	giveAttireItem(app, silverOfCariaItem)

	state, err := app.GetSpectralSteedAttire(0)
	if err != nil {
		t.Fatal(err)
	}
	want := map[uint32]bool{
		6700: true, // default needs no item
		6701: false,
		6702: true,
		6703: false,
	}
	for _, e := range state.Entries {
		if e.Owned != want[e.ID] {
			t.Fatalf("entry %d owned = %v, want %v", e.ID, e.Owned, want[e.ID])
		}
	}
	// Icons come from the existing item database; default has none.
	if state.Entries[0].IconPath != "" {
		t.Fatalf("default icon = %q, want empty", state.Entries[0].IconPath)
	}
	for _, e := range state.Entries[1:] {
		if e.IconPath == "" {
			t.Fatalf("attire %d has no icon path", e.ID)
		}
	}
}

func TestSetSpectralSteedAttireRejectsUnknownID(t *testing.T) {
	app := spectralSteedApp(t)
	setSpectralSteedFlag(t, app, 6700, true)

	for _, bad := range []uint32{0, 6699, 6704, 60100} {
		if err := app.SetSpectralSteedAttire(0, bad); err == nil {
			t.Fatalf("id %d: expected rejection", bad)
		}
	}
	states := spectralSteedFlagStates(t, app)
	if !states[6700] || states[6701] || states[6702] || states[6703] {
		t.Fatalf("rejected call mutated flags: %v", states)
	}
}

func TestSetSpectralSteedAttireRequiresItem(t *testing.T) {
	app := spectralSteedApp(t)
	setSpectralSteedFlag(t, app, 6700, true)

	err := app.SetSpectralSteedAttire(0, 6702)
	if err == nil {
		t.Fatal("expected rejection without the key item")
	}
	if !strings.Contains(err.Error(), "requires its key item") {
		t.Fatalf("error = %v", err)
	}
	states := spectralSteedFlagStates(t, app)
	if !states[6700] || states[6702] {
		t.Fatalf("failed set mutated flags: %v", states)
	}
}

func TestSetSpectralSteedAttireDefaultNeedsNoItem(t *testing.T) {
	app := spectralSteedApp(t)

	if err := app.SetSpectralSteedAttire(0, data.SpectralSteedAttireDefaultFlag); err != nil {
		t.Fatal(err)
	}
	states := spectralSteedFlagStates(t, app)
	if !states[6700] || states[6701] || states[6702] || states[6703] {
		t.Fatalf("flags = %v, want only 6700", states)
	}
}

func TestSetSpectralSteedAttireClearsPreviousSelection(t *testing.T) {
	app := spectralSteedApp(t)
	setSpectralSteedFlag(t, app, 6700, true)
	giveAttireItem(app, treeSentinelAttireItem)
	giveAttireItem(app, funerealNightItem)

	if err := app.SetSpectralSteedAttire(0, 6701); err != nil {
		t.Fatal(err)
	}
	if states := spectralSteedFlagStates(t, app); !states[6701] || states[6700] || states[6702] || states[6703] {
		t.Fatalf("flags = %v, want only 6701", states)
	}

	if err := app.SetSpectralSteedAttire(0, 6703); err != nil {
		t.Fatal(err)
	}
	states := spectralSteedFlagStates(t, app)
	if !states[6703] || states[6700] || states[6701] || states[6702] {
		t.Fatalf("flags = %v, want only 6703", states)
	}

	state, err := app.GetSpectralSteedAttire(0)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != db.SpectralSteedAttireResolved || state.ActiveID != 6703 {
		t.Fatalf("state = %+v, want resolved 6703", state)
	}
}

// A write that cannot reach every owned flag must leave no partial selection:
// the previously active appearance stays untouched.
func TestSetSpectralSteedAttireErrorLeavesNoPartialMutation(t *testing.T) {
	app := spectralSteedApp(t)
	setSpectralSteedFlag(t, app, 6700, true)
	giveAttireItem(app, treeSentinelAttireItem)

	// Shrink the event-flags region so 6700-6703 fall out of bounds mid-write.
	slot := &app.save.Slots[0]
	before := append([]byte(nil), slot.Data...)
	slot.Data = slot.Data[:slot.EventFlagsOffset+8]

	if err := app.SetSpectralSteedAttire(0, 6701); err == nil {
		t.Fatal("expected an out-of-bounds write error")
	}
	if len(slot.Data) != slot.EventFlagsOffset+8 {
		t.Fatalf("slot data length changed to %d", len(slot.Data))
	}
	for i := range slot.Data {
		if slot.Data[i] != before[i] {
			t.Fatalf("partial mutation at byte %d", i)
		}
	}
}

// PS4/PS5 saves decode into the same slot model, so 6700-6703 behave exactly
// like on PC. PS5 is identified as the PS4 container format.
func TestSetSpectralSteedAttireOnPS4(t *testing.T) {
	app := spectralSteedApp(t)
	app.save.Platform = "PS4"
	setSpectralSteedFlag(t, app, 6700, true)
	giveAttireItem(app, treeSentinelAttireItem)

	if err := app.SetSpectralSteedAttire(0, 6701); err != nil {
		t.Fatalf("PS4 set must succeed: %v", err)
	}
	states := spectralSteedFlagStates(t, app)
	if !states[6701] || states[6700] || states[6702] || states[6703] {
		t.Fatalf("flags = %v, want only 6701 set", states)
	}
	if !spectralSteedItemOwned(&app.save.Slots[0], treeSentinelAttireItem) {
		t.Fatal("set removed the attire item")
	}
	state, err := app.GetSpectralSteedAttire(0)
	if err != nil {
		t.Fatalf("getter on PS4: %v", err)
	}
	if state.Status != db.SpectralSteedAttireResolved || state.ActiveID != 6701 {
		t.Fatalf("state = %+v, want resolved 6701", state)
	}
}

func TestLockAllSpectralSteedAttiresRemovesItemsAndRestoresDefault(t *testing.T) {
	app := spectralSteedApp(t)
	addAttireItems(t, app, treeSentinelAttireItem, silverOfCariaItem, funerealNightItem)
	setSpectralSteedFlag(t, app, 6703, true)

	if err := app.LockAllSpectralSteedAttires(0); err != nil {
		t.Fatal(err)
	}

	state, err := app.GetSpectralSteedAttire(0)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != db.SpectralSteedAttireResolved || state.ActiveID != data.SpectralSteedAttireDefaultFlag {
		t.Fatalf("state = %+v, want resolved default appearance", state)
	}
	for _, entry := range state.Entries {
		wantOwned := entry.ID == data.SpectralSteedAttireDefaultFlag
		if entry.Owned != wantOwned {
			t.Fatalf("entry %d owned = %v, want %v", entry.ID, entry.Owned, wantOwned)
		}
	}
}

func TestLockAllSpectralSteedAttiresRejectsTruncatedFlagsWithoutMutation(t *testing.T) {
	app := spectralSteedApp(t)
	giveAttireItem(app, treeSentinelAttireItem)
	setSpectralSteedFlag(t, app, 6701, true)
	slot := &app.save.Slots[0]
	slot.Data = slot.Data[:slot.EventFlagsOffset+8]
	before := core.CloneSlot(slot)

	if err := app.LockAllSpectralSteedAttires(0); err == nil {
		t.Fatal("expected truncated event flags to be rejected")
	}
	if len(slot.Inventory.KeyItems) != len(before.Inventory.KeyItems) || slot.Inventory.KeyItems[0] != before.Inventory.KeyItems[0] {
		t.Fatalf("failed lock mutated inventory: got %+v, want %+v", slot.Inventory.KeyItems, before.Inventory.KeyItems)
	}
	if string(slot.Data) != string(before.Data) {
		t.Fatal("failed lock mutated slot data")
	}
}

func TestLockAllSpectralSteedAttiresOnPS4(t *testing.T) {
	app := spectralSteedApp(t)
	app.save.Platform = "PS4"
	addAttireItems(t, app, treeSentinelAttireItem, silverOfCariaItem, funerealNightItem)
	setSpectralSteedFlag(t, app, 6703, true)

	if err := app.LockAllSpectralSteedAttires(0); err != nil {
		t.Fatalf("PS4 lock all must succeed: %v", err)
	}
	for _, itemID := range []uint32{treeSentinelAttireItem, silverOfCariaItem, funerealNightItem} {
		if spectralSteedItemOwned(&app.save.Slots[0], itemID) {
			t.Fatalf("item 0x%08X still owned after lock all", itemID)
		}
	}
	states := spectralSteedFlagStates(t, app)
	if !states[data.SpectralSteedAttireDefaultFlag] || states[6701] || states[6702] || states[6703] {
		t.Fatalf("flags = %v, want only default appearance set", states)
	}
	state, err := app.GetSpectralSteedAttire(0)
	if err != nil {
		t.Fatalf("getter on PS4: %v", err)
	}
	if state.Status != db.SpectralSteedAttireResolved || state.ActiveID != data.SpectralSteedAttireDefaultFlag {
		t.Fatalf("state = %+v, want resolved default appearance", state)
	}
}

// A slot whose event-flag region was never located must not be reported as a
// pre-Regulation-1.17 save: "legacy" is a positive finding about readable flags.
func TestGetSpectralSteedAttireRejectsMissingEventFlagsOffset(t *testing.T) {
	app := spectralSteedApp(t)
	app.save.Slots[0].EventFlagsOffset = 0

	if _, err := app.GetSpectralSteedAttire(0); err == nil {
		t.Fatal("expected error for missing event flags offset")
	} else if !strings.Contains(err.Error(), "event flags offset") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// A truncated flag region makes every 6700-6703 read fail; the getter must
// propagate that instead of collapsing it into "legacy".
func TestGetSpectralSteedAttireRejectsTruncatedFlagRegion(t *testing.T) {
	app := spectralSteedApp(t)
	slot := &app.save.Slots[0]
	slot.Data = slot.Data[:slot.EventFlagsOffset+8]

	if _, err := app.GetSpectralSteedAttire(0); err == nil {
		t.Fatal("expected error for truncated event flag region")
	} else if !strings.Contains(err.Error(), "out of bounds") {
		t.Fatalf("unexpected error: %v", err)
	}
}
