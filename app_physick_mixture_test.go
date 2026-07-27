package main

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
)

// Owned goods handles used across the Physick mixture tests. HandleToItemID maps
// each to its bare GoodsParam tear ID (0xB0… -> 0x40…). Crimson 0x40002AFA is
// the first distinct Crimson Crystal Tear, picked up with the Physick Flask;
// the writer must preserve exactly what the character owns.
const (
	flaskHandle        = uint32(0xB00000FB) // Flask of Wondrous Physick (empty)
	greenspillHandle   = uint32(0xB0002AF9) // Greenspill Crystal Tear -> 0x40002AF9
	greenspillID       = uint32(0x40002AF9)
	crimsonFlaskHandle = uint32(0xB0002AFA) // first Crimson Crystal Tear -> 0x40002AFA
	crimsonFlaskID     = uint32(0x40002AFA)
	larvalHandle       = uint32(0xB0001FF9) // Larval Tear: goods, but NOT a crystal tear
)

// physickOffset returns the EquipPhysicsData base for a slot built by
// buildEquipSlot, mirroring ReadEquippedState's anchoring exactly.
func physickOffset() int {
	projHeaderOff := testEquippedSpellsOffset + core.DynEquipedSpells + core.DynEquipedItems + core.DynEquipedGestures
	armamentsOff := projHeaderOff + 4 + testProjCount*8
	return armamentsOff + core.DynEquipedArmaments
}

// newPhysickApp builds a synthetic slot carrying the flask plus the given owned
// tears, and seeds the two mixture fields empty and the trailing u32 with a
// sentinel so preservation can be asserted.
func newPhysickApp(t *testing.T, tears ...core.InventoryItem) (*App, int) {
	t.Helper()
	slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	slot.Inventory.CommonItems = append([]core.InventoryItem{{GaItemHandle: flaskHandle, Quantity: 1}}, tears...)

	physicsOff := physickOffset()
	binary.LittleEndian.PutUint32(slot.Data[physicsOff:], core.GaHandleInvalid)
	binary.LittleEndian.PutUint32(slot.Data[physicsOff+4:], core.GaHandleInvalid)
	binary.LittleEndian.PutUint32(slot.Data[physicsOff+8:], physickTrailingSentinel)

	return newEquipmentApp(slot), physicsOff
}

const physickTrailingSentinel = uint32(0xDEADBEEF)

func readPhysickFields(t *testing.T, app *App, physicsOff int) (slot0, slot1, trailing uint32) {
	t.Helper()
	data := app.save.Slots[0].Data
	return binary.LittleEndian.Uint32(data[physicsOff:]),
		binary.LittleEndian.Uint32(data[physicsOff+4:]),
		binary.LittleEndian.Uint32(data[physicsOff+8:])
}

func TestSavePhysickMixture_WritesTwoTearsAndPreservesTrailing(t *testing.T) {
	app, physicsOff := newPhysickApp(t,
		core.InventoryItem{GaItemHandle: greenspillHandle, Quantity: 1},
		core.InventoryItem{GaItemHandle: crimsonFlaskHandle, Quantity: 1},
	)

	if err := app.SavePhysickMixture(0, []PhysickChange{
		{Slot: core.PhysickSlot1, Handle: greenspillHandle},
		{Slot: core.PhysickSlot2, Handle: crimsonFlaskHandle},
	}); err != nil {
		t.Fatalf("SavePhysickMixture: %v", err)
	}

	slot0, slot1, trailing := readPhysickFields(t, app, physicsOff)
	if slot0 != greenspillID {
		t.Errorf("slot0 = 0x%08X, want 0x%08X", slot0, greenspillID)
	}
	// Crimson raw variant preserved exactly — not canonicalized to 0x40002AFB.
	if slot1 != crimsonFlaskID {
		t.Errorf("slot1 = 0x%08X, want first Crimson tear 0x%08X", slot1, crimsonFlaskID)
	}
	if trailing != physickTrailingSentinel {
		t.Errorf("trailing u32 = 0x%08X, want preserved 0x%08X", trailing, physickTrailingSentinel)
	}
}

func TestSavePhysickMixture_SwapAcrossSlots(t *testing.T) {
	app, physicsOff := newPhysickApp(t,
		core.InventoryItem{GaItemHandle: greenspillHandle, Quantity: 1},
		core.InventoryItem{GaItemHandle: crimsonFlaskHandle, Quantity: 1},
	)
	if err := app.SavePhysickMixture(0, []PhysickChange{
		{Slot: core.PhysickSlot1, Handle: crimsonFlaskHandle},
		{Slot: core.PhysickSlot2, Handle: greenspillHandle},
	}); err != nil {
		t.Fatalf("SavePhysickMixture: %v", err)
	}
	slot0, slot1, _ := readPhysickFields(t, app, physicsOff)
	if slot0 != crimsonFlaskID || slot1 != greenspillID {
		t.Errorf("swapped mixture = 0x%08X/0x%08X, want 0x%08X/0x%08X", slot0, slot1, crimsonFlaskID, greenspillID)
	}
}

func TestSavePhysickMixture_SingleTearInSlot2(t *testing.T) {
	app, physicsOff := newPhysickApp(t, core.InventoryItem{GaItemHandle: greenspillHandle, Quantity: 1})
	if err := app.SavePhysickMixture(0, []PhysickChange{
		{Slot: core.PhysickSlot2, Handle: greenspillHandle},
	}); err != nil {
		t.Fatalf("SavePhysickMixture: %v", err)
	}
	slot0, slot1, _ := readPhysickFields(t, app, physicsOff)
	// No left-packing: slot1 stays empty, the tear lives only in slot 2.
	if slot0 != core.GaHandleInvalid {
		t.Errorf("slot0 = 0x%08X, want empty sentinel (no left-pack)", slot0)
	}
	if slot1 != greenspillID {
		t.Errorf("slot1 = 0x%08X, want 0x%08X", slot1, greenspillID)
	}
}

func TestSavePhysickMixture_ClearWritesSentinel(t *testing.T) {
	app, physicsOff := newPhysickApp(t, core.InventoryItem{GaItemHandle: greenspillHandle, Quantity: 1})
	// Seed slot 1 with a tear, then clear it.
	binary.LittleEndian.PutUint32(app.save.Slots[0].Data[physicsOff:], greenspillID)
	if err := app.SavePhysickMixture(0, []PhysickChange{{Slot: core.PhysickSlot1, Handle: 0}}); err != nil {
		t.Fatalf("SavePhysickMixture: %v", err)
	}
	slot0, _, trailing := readPhysickFields(t, app, physicsOff)
	if slot0 != core.GaHandleInvalid {
		t.Errorf("cleared slot0 = 0x%08X, want 0xFFFFFFFF", slot0)
	}
	if trailing != physickTrailingSentinel {
		t.Errorf("trailing changed on clear: 0x%08X", trailing)
	}
}

func TestSavePhysickMixture_RejectsUnownedHandle(t *testing.T) {
	app, physicsOff := newPhysickApp(t) // flask only, no tears owned
	err := app.SavePhysickMixture(0, []PhysickChange{{Slot: core.PhysickSlot1, Handle: greenspillHandle}})
	if err == nil || !strings.Contains(err.Error(), "not an owned tear") {
		t.Fatalf("error = %v, want unowned-tear rejection", err)
	}
	slot0, slot1, _ := readPhysickFields(t, app, physicsOff)
	if slot0 != core.GaHandleInvalid || slot1 != core.GaHandleInvalid {
		t.Errorf("mixture mutated after rejection: 0x%08X/0x%08X", slot0, slot1)
	}
}

func TestSavePhysickMixture_RejectsNonTearGoods(t *testing.T) {
	app, _ := newPhysickApp(t, core.InventoryItem{GaItemHandle: larvalHandle, Quantity: 1})
	err := app.SavePhysickMixture(0, []PhysickChange{{Slot: core.PhysickSlot1, Handle: larvalHandle}})
	if err == nil || !strings.Contains(err.Error(), "not a Wondrous Physick crystal tear") {
		t.Fatalf("error = %v, want non-tear rejection", err)
	}
}

func TestSavePhysickMixture_RequiresFlask(t *testing.T) {
	slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	slot.Inventory.CommonItems = []core.InventoryItem{{GaItemHandle: greenspillHandle, Quantity: 1}}
	app := newEquipmentApp(slot)
	err := app.SavePhysickMixture(0, []PhysickChange{{Slot: core.PhysickSlot1, Handle: greenspillHandle}})
	if err == nil || !strings.Contains(err.Error(), "Flask of Wondrous Physick") {
		t.Fatalf("error = %v, want missing-flask rejection", err)
	}
}

func TestSavePhysickMixture_NoPartialMutationOnBatchError(t *testing.T) {
	// First entry is valid, second is unowned: the whole batch must be rejected
	// before any field is written (validate-then-mutate).
	app, physicsOff := newPhysickApp(t, core.InventoryItem{GaItemHandle: greenspillHandle, Quantity: 1})
	err := app.SavePhysickMixture(0, []PhysickChange{
		{Slot: core.PhysickSlot1, Handle: greenspillHandle},
		{Slot: core.PhysickSlot2, Handle: crimsonFlaskHandle}, // not owned
	})
	if err == nil {
		t.Fatal("expected batch rejection, got nil")
	}
	slot0, slot1, _ := readPhysickFields(t, app, physicsOff)
	if slot0 != core.GaHandleInvalid || slot1 != core.GaHandleInvalid {
		t.Errorf("partial mutation after rejected batch: 0x%08X/0x%08X", slot0, slot1)
	}
}

func TestSavePhysickMixture_KeyItemsContainerCounts(t *testing.T) {
	// A tear owned in KeyItems (not CommonItems) is still eligible; Storage is not
	// consulted, but the two carried containers are.
	slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	slot.Inventory.CommonItems = []core.InventoryItem{{GaItemHandle: flaskHandle, Quantity: 1}}
	slot.Inventory.KeyItems = []core.InventoryItem{{GaItemHandle: greenspillHandle, Quantity: 1}}
	app := newEquipmentApp(slot)
	physicsOff := physickOffset()
	if err := app.SavePhysickMixture(0, []PhysickChange{{Slot: core.PhysickSlot1, Handle: greenspillHandle}}); err != nil {
		t.Fatalf("SavePhysickMixture: %v", err)
	}
	if slot0, _, _ := readPhysickFields(t, app, physicsOff); slot0 != greenspillID {
		t.Errorf("slot0 = 0x%08X, want 0x%08X from KeyItems-owned tear", slot0, greenspillID)
	}
}

// seedPhysickTear builds a slot whose first Physick field carries rawID and
// places the matching owned handle into the given container, so the snapshot's
// handle resolution can be exercised end-to-end.
func seedPhysickTear(t *testing.T, rawID uint32, container string, owned core.InventoryItem) *App {
	t.Helper()
	slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	binary.LittleEndian.PutUint32(slot.Data[physickOffset():], rawID)
	binary.LittleEndian.PutUint32(slot.Data[physickOffset()+4:], core.GaHandleInvalid)
	switch container {
	case "common":
		slot.Inventory.CommonItems = []core.InventoryItem{owned}
	case "key":
		slot.Inventory.KeyItems = []core.InventoryItem{owned}
	default:
		t.Fatalf("unknown container %q", container)
	}
	return newEquipmentApp(slot)
}

func TestGetEquipmentSnapshot_ResolvesOwnedPhysickHandleFromCommonItems(t *testing.T) {
	app := seedPhysickTear(t, greenspillID, "common", core.InventoryItem{GaItemHandle: greenspillHandle, Quantity: 1})
	snap, err := app.GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	if got := snap.Physick[0]; !got.Occupied || !got.Resolved || got.RawID != greenspillID || got.Handle != greenspillHandle {
		t.Errorf("Physick[0] = %+v, want raw 0x%08X handle 0x%08X", got, greenspillID, greenspillHandle)
	}
}

func TestGetEquipmentSnapshot_ResolvesOwnedPhysickHandleFromKeyItems(t *testing.T) {
	app := seedPhysickTear(t, greenspillID, "key", core.InventoryItem{GaItemHandle: greenspillHandle, Quantity: 1})
	snap, err := app.GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	if got := snap.Physick[0].Handle; got != greenspillHandle {
		t.Errorf("Physick[0].Handle = 0x%08X, want 0x%08X from KeyItems", got, greenspillHandle)
	}
}

func TestGetEquipmentSnapshot_PhysickCrimsonVariantKeepsExactRawAndHandle(t *testing.T) {
	app := seedPhysickTear(t, crimsonFlaskID, "common", core.InventoryItem{GaItemHandle: crimsonFlaskHandle, Quantity: 1})
	snap, err := app.GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	got := snap.Physick[0]
	// Raw ID and owned handle are the exact technical variant — never canonicalized
	// to the picker's 0x40002AFB / 0xB0002AFB — while the display name resolves to
	// the canonical Crimson Crystal Tear.
	if got.RawID != crimsonFlaskID || got.Handle != crimsonFlaskHandle {
		t.Errorf("Physick[0] raw/handle = 0x%08X/0x%08X, want 0x%08X/0x%08X", got.RawID, got.Handle, crimsonFlaskID, crimsonFlaskHandle)
	}
	if !got.Resolved || !strings.Contains(got.Name, "Crimson") {
		t.Errorf("Physick[0] name = %q resolved=%v, want resolved Crimson", got.Name, got.Resolved)
	}
}

func TestGetEquipmentSnapshot_PhysickHandleZeroWhenNotOwned(t *testing.T) {
	// The tear ID is present in the mixture but no matching owned goods handle
	// exists; the view stays resolved-by-ID with Handle left unset.
	slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	binary.LittleEndian.PutUint32(slot.Data[physickOffset():], greenspillID)
	app := newEquipmentApp(slot)
	snap, err := app.GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	if got := snap.Physick[0]; got.Handle != 0 || got.RawID != greenspillID {
		t.Errorf("Physick[0] = %+v, want Handle 0 with raw 0x%08X preserved", got, greenspillID)
	}
}

// TestSavePhysickMixture_SnapshotReadBack proves the App binding round-trips: a
// written mixture is projected back through GetEquipmentSnapshot with the raw
// Crimson variant resolved to its canonical display name.
func TestSavePhysickMixture_SnapshotReadBack(t *testing.T) {
	app, _ := newPhysickApp(t,
		core.InventoryItem{GaItemHandle: greenspillHandle, Quantity: 1},
		core.InventoryItem{GaItemHandle: crimsonFlaskHandle, Quantity: 1},
	)
	if err := app.SavePhysickMixture(0, []PhysickChange{
		{Slot: core.PhysickSlot1, Handle: greenspillHandle},
		{Slot: core.PhysickSlot2, Handle: crimsonFlaskHandle},
	}); err != nil {
		t.Fatalf("SavePhysickMixture: %v", err)
	}
	snap, err := app.GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	if got := snap.Physick[0]; !got.Occupied || !got.Resolved || got.RawID != greenspillID {
		t.Errorf("Physick[0] = %+v, want resolved Greenspill 0x%08X", got, greenspillID)
	}
	if got := snap.Physick[1]; !got.Occupied || !got.Resolved || got.RawID != crimsonFlaskID || !strings.Contains(got.Name, "Crimson") {
		t.Errorf("Physick[1] = %+v, want first Crimson tear raw 0x%08X", got, crimsonFlaskID)
	}
}
