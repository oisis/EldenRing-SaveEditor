package main

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
)

// Equipment section offsets used by the synthetic fixture. Values mirror the
// hermetic reader test in backend/core so ReadEquippedState exercises the real
// equipped-armaments and EquipItemData layouts without touching any on-disk save.
const (
	testEquipItemsIDOffset   = 0x10000
	testEquippedSpellsOffset = 0x20000
	testProjCount            = 3 // non-zero, so the fixture exercises the projectile skip
)

// buildEquipSlot builds a fully in-memory SaveSlot with parsed equipment
// offsets and the given equipped / quick-item / pouch values written into
// slot.Data at the exact positions ReadEquippedState reads from. The 22 equipped
// values go into the equipped-armaments block (past a non-zero projectiles
// section); the legacy EquipItemsID header gets decoy values so a regression to
// that header is caught. No real save bytes, no DB, no file I/O.
func buildEquipSlot(equipped [core.ChrAsmFieldCount]uint32, quick [10]core.RawEquipItem, pouch [6]core.RawEquipItem) core.SaveSlot {
	data := make([]byte, core.SlotSize)

	// Decoy header — the reader must not read equipment from here.
	for i := 0; i < core.ChrAsmFieldCount; i++ {
		binary.LittleEndian.PutUint32(data[testEquipItemsIDOffset+i*4:], 0x8001FF00|uint32(i))
	}

	base := testEquippedSpellsOffset + core.DynEquipedSpells // EquipItemData base
	for i := 0; i < 10; i++ {
		off := base + i*8
		binary.LittleEndian.PutUint32(data[off:], quick[i].ItemID)
		binary.LittleEndian.PutUint32(data[off+4:], quick[i].EquipIndex)
	}
	// pouch starts after 10 quick pairs (0x50) + active_slot i32 (0x54).
	pouchBase := base + 10*8 + 4
	for i := 0; i < 6; i++ {
		off := pouchBase + i*8
		binary.LittleEndian.PutUint32(data[off:], pouch[i].ItemID)
		binary.LittleEndian.PutUint32(data[off+4:], pouch[i].EquipIndex)
	}

	// Equipped-armaments block, past the variable projectiles section.
	projHeaderOff := testEquippedSpellsOffset + core.DynEquipedSpells + core.DynEquipedItems + core.DynEquipedGestures
	binary.LittleEndian.PutUint32(data[projHeaderOff:], testProjCount)
	armamentsOff := projHeaderOff + 4 + testProjCount*8
	for i := 0; i < core.ChrAsmFieldCount; i++ {
		binary.LittleEndian.PutUint32(data[armamentsOff+i*4:], equipped[i])
	}

	return core.SaveSlot{
		Data:                 data,
		Version:              1,
		MagicOffset:          testEquipItemsIDOffset - (core.DynSpEffect + core.DynEquipedItemIndex + core.DynActiveEquipedItems + core.DynEquipedItemsID),
		EquipItemsIDOffset:   testEquipItemsIDOffset,
		EquippedSpellsOffset: testEquippedSpellsOffset,
		Player:               core.PlayerGameData{TalismanSlots: 3},
	}
}

func TestGetEquipmentSnapshot_UsesUnlockedTalismanSlotCount(t *testing.T) {
	var equipped [core.ChrAsmFieldCount]uint32
	// Real talismans in every visible-position source slot. This proves that
	// locked slots do not leak into the snapshot or the current load.
	equipped[17] = 0x200003E8 // Crimson Amber Medallion, 0.3
	equipped[18] = 0x200003E8
	equipped[19] = 0x200003E8
	equipped[20] = 0x200003E8

	for _, tc := range []struct {
		additional uint8
		want       int
	}{
		{additional: 0, want: 1},
		{additional: 1, want: 2},
		{additional: 2, want: 3},
		{additional: 3, want: 4},
		{additional: 99, want: 4},
	} {
		t.Run(fmt.Sprintf("additional=%d", tc.additional), func(t *testing.T) {
			slot := buildEquipSlot(equipped, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
			slot.Player.TalismanSlots = tc.additional

			snap, err := newEquipmentApp(slot).GetEquipmentSnapshot(0)
			if err != nil {
				t.Fatalf("GetEquipmentSnapshot: %v", err)
			}
			if snap.ActiveTalismanSlots != tc.want {
				t.Errorf("ActiveTalismanSlots = %d, want %d", snap.ActiveTalismanSlots, tc.want)
			}
			for i := 0; i < len(snap.Talismans); i++ {
				if got := snap.Talismans[i].Occupied; got != (i < tc.want) {
					t.Errorf("Talismans[%d].Occupied = %v, want %v", i, got, i < tc.want)
				}
			}
			if wantLoad := float64(tc.want) * 0.3; snap.CurrentEquipLoad < wantLoad-0.001 || snap.CurrentEquipLoad > wantLoad+0.001 {
				t.Errorf("CurrentEquipLoad = %.3f, want %.1f", snap.CurrentEquipLoad, wantLoad)
			}
		})
	}
}

// newEquipmentApp wires the synthetic slot into slot 0 of a fresh App with that
// slot active, so GetEquipmentSnapshot(0) reads it end-to-end.
func newEquipmentApp(slot core.SaveSlot) *App {
	app := NewApp()
	app.save = &core.SaveFile{}
	app.save.Slots[0] = slot
	app.save.ActiveSlots[0] = true
	return app
}

func putSpellRecord(slot *core.SaveSlot, index int, id uint32) {
	off := slot.EquippedSpellsOffset + index*core.EquippedSpellSlotSize
	binary.LittleEndian.PutUint32(slot.Data[off:], id)
	follower := uint32(0)
	if id != core.EquippedSpellEmptySentinel {
		follower = core.EquippedSpellOccupiedFollower
	}
	binary.LittleEndian.PutUint32(slot.Data[off+4:], follower)
}

func TestGetEquipmentSnapshot_ProjectsSpellsAndActiveIndex(t *testing.T) {
	slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	for i := 0; i < core.EquippedSpellSlotCount; i++ {
		putSpellRecord(&slot, i, core.EquippedSpellEmptySentinel)
	}
	putSpellRecord(&slot, 0, 0x00000FA0)
	putSpellRecord(&slot, 1, 0x00001770)
	binary.LittleEndian.PutUint32(slot.Data[slot.EquippedSpellsOffset+core.EquippedSpellActiveIndexOffset:], 1)

	snap, err := newEquipmentApp(slot).GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	if got := snap.Spells[0]; !got.Occupied || !got.Resolved || got.RawID != 0x00000FA0 || got.Name != "Glintstone Pebble" {
		t.Errorf("Spells[0] = %+v, want resolved Glintstone Pebble", got)
	}
	if got := snap.Spells[1]; !got.Occupied || !got.Resolved || got.RawID != 0x00001770 || got.Name != "Catch Flame" {
		t.Errorf("Spells[1] = %+v, want resolved Catch Flame", got)
	}
	if snap.Spells[2].Occupied || snap.Spells[2].RawID != core.EquippedSpellEmptySentinel {
		t.Errorf("Spells[2] = %+v, want empty sentinel view", snap.Spells[2])
	}
	if snap.ActiveSpellIndex != 1 {
		t.Errorf("ActiveSpellIndex = %d, want 1", snap.ActiveSpellIndex)
	}
}

func TestSaveEquippedSpells_CompactsAndPreservesActiveIndex(t *testing.T) {
	slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	for i := 0; i < core.EquippedSpellSlotCount; i++ {
		putSpellRecord(&slot, i, core.EquippedSpellEmptySentinel)
	}
	putSpellRecord(&slot, 0, 0x00000FA0)
	putSpellRecord(&slot, 1, 0x00001770)
	binary.LittleEndian.PutUint32(slot.Data[slot.EquippedSpellsOffset+core.EquippedSpellActiveIndexOffset:], 1)
	app := newEquipmentApp(slot)

	if err := app.SaveEquippedSpells(0, []uint32{0x40000FA0, 0x40001158}); err != nil {
		t.Fatalf("SaveEquippedSpells: %v", err)
	}
	raw, err := app.save.Slots[0].ReadEquippedState()
	if err != nil {
		t.Fatalf("ReadEquippedState: %v", err)
	}
	if got, want := raw.Spells[0], uint32(0x00000FA0); got != want {
		t.Errorf("Spells[0] = 0x%08X, want 0x%08X", got, want)
	}
	if got, want := raw.Spells[1], uint32(0x00001158); got != want {
		t.Errorf("Spells[1] = 0x%08X, want 0x%08X", got, want)
	}
	if raw.Spells[2] != core.EquippedSpellEmptySentinel {
		t.Errorf("Spells[2] = 0x%08X, want empty sentinel", raw.Spells[2])
	}
	if raw.ActiveSpellIndex != 1 {
		t.Errorf("ActiveSpellIndex = %d, want unchanged 1", raw.ActiveSpellIndex)
	}
}

func TestSaveEquippedSpells_NormalizesOutOfRangeActiveIndexToZero(t *testing.T) {
	slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	for i := 0; i < core.EquippedSpellSlotCount; i++ {
		putSpellRecord(&slot, i, core.EquippedSpellEmptySentinel)
	}
	putSpellRecord(&slot, 0, 0x00000FA0)
	putSpellRecord(&slot, 1, 0x00001770)
	putSpellRecord(&slot, 2, 0x00001158)
	binary.LittleEndian.PutUint32(slot.Data[slot.EquippedSpellsOffset+core.EquippedSpellActiveIndexOffset:], 2)
	app := newEquipmentApp(slot)

	if err := app.SaveEquippedSpells(0, []uint32{0x40000FA0, 0x40001770}); err != nil {
		t.Fatalf("SaveEquippedSpells: %v", err)
	}
	raw, err := app.save.Slots[0].ReadEquippedState()
	if err != nil {
		t.Fatalf("ReadEquippedState: %v", err)
	}
	if raw.ActiveSpellIndex != 0 {
		t.Errorf("ActiveSpellIndex = %d, want native fallback 0", raw.ActiveSpellIndex)
	}
}

func TestSaveEquipment_WritesOwnedHandleToFourNativeRepresentations(t *testing.T) {
	var equipped [core.ChrAsmFieldCount]uint32
	equipped[1] = 0x0001ADB0 // native empty right-hand item (Unarmed)
	slot := buildEquipSlot(equipped, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	const (
		unarmedHandle = 0x80800079
		clubHandle    = 0x80800010
		clubItemID    = 0x00100020
	)
	slot.GaMap = map[uint32]uint32{
		unarmedHandle: 0x0001ADB0,
		clubHandle:    clubItemID,
	}
	slot.Inventory.CommonItems = []core.InventoryItem{{GaItemHandle: clubHandle, Quantity: 1}}

	app := newEquipmentApp(slot)
	if err := app.SaveEquipment(0, []EquipmentChange{{Slot: core.EquipSlotRightHandArmament1, Handle: clubHandle}}); err != nil {
		t.Fatalf("SaveEquipment: %v", err)
	}
	raw, err := app.save.Slots[0].ReadEquippedState()
	if err != nil {
		t.Fatalf("ReadEquippedState: %v", err)
	}
	if got := raw.Equipped[1]; got != clubItemID {
		t.Errorf("dynamic right hand = 0x%08X, want 0x%08X", got, uint32(clubItemID))
	}
	fieldOff := 1 + 1*4
	equipIndexBase := slot.MagicOffset + core.DynSpEffect
	itemIDBase := equipIndexBase + core.DynEquipedItemIndex + core.DynActiveEquipedItems
	handleBase := itemIDBase + core.DynEquipedItemsID
	if got := binary.LittleEndian.Uint32(app.save.Slots[0].Data[equipIndexBase+fieldOff:]); got != 0x180 {
		t.Errorf("EquipData right hand index = 0x%08X, want 0x00000180", got)
	}
	if got := binary.LittleEndian.Uint32(app.save.Slots[0].Data[itemIDBase+fieldOff:]); got != clubItemID&0x0FFFFFFF {
		t.Errorf("ChrAsm right hand ID = 0x%08X, want 0x%08X", got, uint32(clubItemID&0x0FFFFFFF))
	}
	if got := binary.LittleEndian.Uint32(app.save.Slots[0].Data[handleBase+fieldOff:]); got != clubHandle {
		t.Errorf("ChrAsm2 right hand handle = 0x%08X, want 0x%08X", got, uint32(clubHandle))
	}
	snap, err := app.GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	if got := snap.RightHandArmaments[0].Handle; got != clubHandle {
		t.Errorf("snapshot Handle = 0x%08X, want owned handle 0x%08X", got, uint32(clubHandle))
	}
}

func TestSaveEquipment_WritesOwnedTalismanAndProjectsItsHandle(t *testing.T) {
	slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	const (
		talismanHandle = core.ItemTypeAccessory | 0x000003E8
		inventoryRow   = 4
	)
	slot.Inventory.CommonItems = make([]core.InventoryItem, inventoryRow+1)
	slot.Inventory.CommonItems[inventoryRow] = core.InventoryItem{GaItemHandle: talismanHandle, Quantity: 1}
	app := newEquipmentApp(slot)

	if err := app.SaveEquipment(0, []EquipmentChange{{Slot: core.EquipSlotTalisman1, Handle: talismanHandle}}); err != nil {
		t.Fatalf("SaveEquipment: %v", err)
	}
	raw, err := app.save.Slots[0].ReadEquippedState()
	if err != nil {
		t.Fatalf("ReadEquippedState: %v", err)
	}
	if got := raw.Equipped[17]; got != 0x200003E8 {
		t.Errorf("dynamic talisman = 0x%08X, want 0x200003E8", got)
	}

	fieldOff := 1 + 17*4
	equipIndexBase := slot.MagicOffset + core.DynSpEffect
	itemIDBase := equipIndexBase + core.DynEquipedItemIndex + core.DynActiveEquipedItems
	handleBase := itemIDBase + core.DynEquipedItemsID
	if got := binary.LittleEndian.Uint32(app.save.Slots[0].Data[equipIndexBase+fieldOff:]); got != 0x180+inventoryRow {
		t.Errorf("EquipData talisman index = 0x%08X, want 0x%08X", got, uint32(0x180+inventoryRow))
	}
	if got := binary.LittleEndian.Uint32(app.save.Slots[0].Data[itemIDBase+fieldOff:]); got != 0x000003E8 {
		t.Errorf("ChrAsm talisman ID = 0x%08X, want 0x000003E8", got)
	}
	if got := binary.LittleEndian.Uint32(app.save.Slots[0].Data[handleBase+fieldOff:]); got != talismanHandle {
		t.Errorf("ChrAsm2 talisman handle = 0x%08X, want 0x%08X", got, uint32(talismanHandle))
	}

	snap, err := app.GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	if got := snap.Talismans[0]; !got.Occupied || !got.Resolved || got.Handle != talismanHandle || got.RawID != 0x200003E8 {
		t.Errorf("Talismans[0] = %+v, want equipped Crimson Amber Medallion with owned handle", got)
	}
}

func TestSaveQuickPouchItems_WritesOwnedGoodsAndProjectsQuantity(t *testing.T) {
	slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	const (
		daggerHandle    = uint32(0xB00006A4)
		daggerID        = uint32(0x400006A4)
		telescopeHandle = uint32(0xB00007F8)
		telescopeID     = uint32(0x400007F8)
	)
	slot.GaMap = map[uint32]uint32{
		daggerHandle:    daggerID,
		telescopeHandle: telescopeID,
	}
	slot.Inventory.CommonItems = []core.InventoryItem{
		{GaItemHandle: daggerHandle, Quantity: 17},
		{GaItemHandle: telescopeHandle, Quantity: 1},
	}
	app := newEquipmentApp(slot)

	if err := app.SaveQuickPouchItems(0, []QuickPouchChange{
		{Slot: core.QuickPouchSlotQuick1, Handle: daggerHandle},
		{Slot: core.QuickPouchSlotPouch1, Handle: telescopeHandle},
	}); err != nil {
		t.Fatalf("SaveQuickPouchItems: %v", err)
	}
	raw, err := app.save.Slots[0].ReadEquippedState()
	if err != nil {
		t.Fatalf("ReadEquippedState: %v", err)
	}
	if got := raw.QuickItems[0]; got.ItemID != daggerHandle || got.EquipIndex != 0x180 {
		t.Errorf("QuickItems[0] = %+v, want dagger handle and Inventory row 0", got)
	}
	if got := raw.Pouch[0]; got.ItemID != telescopeHandle || got.EquipIndex != 0x181 {
		t.Errorf("Pouch[0] = %+v, want telescope handle and Inventory row 1", got)
	}

	snap, err := app.GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	if got := snap.QuickItems[0]; !got.Occupied || !got.Resolved || got.Handle != daggerHandle || got.Quantity != 17 {
		t.Errorf("QuickItems[0] = %+v, want resolved Throwing Dagger quantity 17", got)
	}
	if got := snap.Pouch[0]; !got.Occupied || !got.Resolved || got.Handle != telescopeHandle || got.Quantity != 1 {
		t.Errorf("Pouch[0] = %+v, want resolved Telescope quantity 1", got)
	}
}

// TestSaveQuickPouchItems_AcceptsWondrousPhysickVariants proves the flask is
// eligible for Quick Items / Pouch. Its filled variant (0x400000FA) has no
// category row of its own; eligibility must resolve via the canonical empty
// variant (0x400000FB). The raw handle is preserved everywhere, the dynamic
// GoodsParam representation carries the matching raw item ID, and a second batch
// entry is untouched — proving the fix keeps the batch atomic and complete.
func TestSaveQuickPouchItems_AcceptsWondrousPhysickVariants(t *testing.T) {
	const (
		daggerHandle = uint32(0xB00006A4)
		daggerID     = uint32(0x400006A4)
		physickName  = "Flask of Wondrous Physick"
	)
	// Dynamic Quick Items representation lives in the equipped-armaments block.
	projHeaderOff := testEquippedSpellsOffset + core.DynEquipedSpells + core.DynEquipedItems + core.DynEquipedGestures
	armamentsOff := projHeaderOff + 4 + testProjCount*8
	quick1DynamicOff := armamentsOff + core.ChrAsmEquipmentSize

	for _, tc := range []struct {
		name        string
		flaskHandle uint32
		wantDynamic uint32
	}{
		{"filled-0xB00000FA", 0xB00000FA, 0x400000FA},
		{"empty-0xB00000FB", 0xB00000FB, 0x400000FB},
	} {
		t.Run(tc.name, func(t *testing.T) {
			slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
			slot.Inventory.CommonItems = []core.InventoryItem{
				{GaItemHandle: tc.flaskHandle, Quantity: 1},
				{GaItemHandle: daggerHandle, Quantity: 17},
			}
			app := newEquipmentApp(slot)

			if err := app.SaveQuickPouchItems(0, []QuickPouchChange{
				{Slot: core.QuickPouchSlotQuick1, Handle: tc.flaskHandle},
				{Slot: core.QuickPouchSlotPouch1, Handle: daggerHandle},
			}); err != nil {
				t.Fatalf("SaveQuickPouchItems: %v", err)
			}

			raw, err := app.save.Slots[0].ReadEquippedState()
			if err != nil {
				t.Fatalf("ReadEquippedState: %v", err)
			}
			// Handle representation preserves the raw flask handle unchanged.
			if got := raw.QuickItems[0]; got.ItemID != tc.flaskHandle || got.EquipIndex != 0x180 {
				t.Errorf("QuickItems[0] = %+v, want flask handle 0x%08X at Inventory row 0", got, tc.flaskHandle)
			}
			// Dynamic GoodsParam representation carries the matching raw item ID
			// (0x400000FA / 0x400000FB), never the display alias.
			if got := binary.LittleEndian.Uint32(app.save.Slots[0].Data[quick1DynamicOff:]); got != tc.wantDynamic {
				t.Errorf("dynamic Quick1 = 0x%08X, want raw 0x%08X", got, tc.wantDynamic)
			}
			// The second batch entry is preserved intact.
			if got := raw.Pouch[0]; got.ItemID != daggerHandle || got.EquipIndex != 0x181 {
				t.Errorf("Pouch[0] = %+v, want dagger handle 0x%08X at Inventory row 1", got, daggerHandle)
			}

			snap, err := app.GetEquipmentSnapshot(0)
			if err != nil {
				t.Fatalf("GetEquipmentSnapshot: %v", err)
			}
			if got := snap.QuickItems[0]; !got.Occupied || !got.Resolved || got.Handle != tc.flaskHandle || got.Name != physickName || got.Quantity != 1 {
				t.Errorf("QuickItems[0] = %+v, want resolved %q handle 0x%08X quantity 1", got, physickName, tc.flaskHandle)
			}
			if got := snap.Pouch[0]; !got.Occupied || got.Handle != daggerHandle || got.Quantity != 17 {
				t.Errorf("Pouch[0] = %+v, want dagger handle 0x%08X quantity 17", got, daggerHandle)
			}
		})
	}

	// A batch mixing the now-eligible flask with an ineligible item still fails
	// atomically: accepting the flask must not weaken validation of siblings.
	t.Run("atomic-rejection-with-flask", func(t *testing.T) {
		const tearHandle = uint32(0xB0002AFB) // Crimson Crystal Tear: not tools/ashes
		slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
		slot.Inventory.CommonItems = []core.InventoryItem{
			{GaItemHandle: 0xB00000FA, Quantity: 1},
			{GaItemHandle: tearHandle, Quantity: 1},
		}
		app := newEquipmentApp(slot)

		err := app.SaveQuickPouchItems(0, []QuickPouchChange{
			{Slot: core.QuickPouchSlotQuick1, Handle: 0xB00000FA},
			{Slot: core.QuickPouchSlotPouch1, Handle: tearHandle},
		})
		if err == nil || !strings.Contains(err.Error(), "not eligible") {
			t.Fatalf("SaveQuickPouchItems error = %v, want eligibility rejection", err)
		}
		raw, readErr := app.save.Slots[0].ReadEquippedState()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if raw.QuickItems[0].ItemID != 0 || raw.Pouch[0].ItemID != 0 {
			t.Errorf("slots mutated after rejected batch: quick=0x%08X pouch=0x%08X", raw.QuickItems[0].ItemID, raw.Pouch[0].ItemID)
		}
	})
}

func TestSaveQuickPouchItems_ClearIsProjectedAsEmpty(t *testing.T) {
	quick := [10]core.RawEquipItem{{ItemID: 0xB00006A4, EquipIndex: 0x180}}
	pouch := [6]core.RawEquipItem{{ItemID: 0xB00007F8, EquipIndex: 0x181}}
	slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, quick, pouch)
	slot.GaMap = map[uint32]uint32{
		0xB00006A4: 0x400006A4,
		0xB00007F8: 0x400007F8,
	}
	slot.Inventory.CommonItems = []core.InventoryItem{
		{GaItemHandle: 0xB00006A4, Quantity: 10},
		{GaItemHandle: 0xB00007F8, Quantity: 1},
	}
	app := newEquipmentApp(slot)

	if err := app.SaveQuickPouchItems(0, []QuickPouchChange{
		{Slot: core.QuickPouchSlotQuick1, Handle: 0},
		{Slot: core.QuickPouchSlotPouch1, Handle: 0},
	}); err != nil {
		t.Fatalf("SaveQuickPouchItems: %v", err)
	}
	snap, err := app.GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	if snap.QuickItems[0].Occupied || snap.Pouch[0].Occupied {
		t.Errorf("cleared views remain occupied: quick=%+v pouch=%+v", snap.QuickItems[0], snap.Pouch[0])
	}
}

func TestSaveQuickPouchItems_RejectsNonEligibleGoodsBeforeMutation(t *testing.T) {
	slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	const tearHandle = uint32(0xB0002AFB)
	slot.Inventory.CommonItems = []core.InventoryItem{{GaItemHandle: tearHandle, Quantity: 1}}
	app := newEquipmentApp(slot)

	err := app.SaveQuickPouchItems(0, []QuickPouchChange{{Slot: core.QuickPouchSlotQuick1, Handle: tearHandle}})
	if err == nil || !strings.Contains(err.Error(), "not eligible") {
		t.Fatalf("SaveQuickPouchItems error = %v, want eligibility rejection", err)
	}
	raw, readErr := app.save.Slots[0].ReadEquippedState()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if raw.QuickItems[0].ItemID != 0 {
		t.Errorf("QuickItems[0] mutated to 0x%08X after rejected write", raw.QuickItems[0].ItemID)
	}
}

func TestGetEquipmentSnapshot_ValidationErrors(t *testing.T) {
	app := NewApp()
	// No save loaded.
	if _, err := app.GetEquipmentSnapshot(0); err == nil {
		t.Error("expected error when no save is loaded")
	}

	// Save loaded but index out of range / inactive.
	app.save = &core.SaveFile{}
	if _, err := app.GetEquipmentSnapshot(-1); err == nil {
		t.Error("expected error for negative index")
	}
	if _, err := app.GetEquipmentSnapshot(10); err == nil {
		t.Error("expected error for index >= 10")
	}
	if _, err := app.GetEquipmentSnapshot(0); err == nil {
		t.Error("expected error for inactive slot")
	}
}

// TestGetEquipmentSnapshot_Mapping is the primary regression proof: a fully
// synthetic, in-memory slot with distinct values per ChrAsm index and distinct
// item_id vs equip_index for every quick/pouch pair. It asserts the exact
// App-level raw-save-to-UI mapping order by RawID, so any accidental slot swap
// (e.g. left/right, arrows/bolts, item_id/equip_index) fails loudly.
func TestGetEquipmentSnapshot_Mapping(t *testing.T) {
	// Distinct, index-tagged, non-sentinel ChrAsm values: the low byte is the
	// ChrAsm index, so a swap is obvious in the failure message.
	var chrAsm [core.ChrAsmFieldCount]uint32
	chrAsmRaw := func(idx int) uint32 { return 0xC0DE0000 | uint32(idx) }
	for i := range chrAsm {
		chrAsm[i] = chrAsmRaw(i)
	}

	// Quick/pouch: item_id and equip_index deliberately in different ranges so a
	// pair-vs-flat misread or an item_id/equip_index swap is unmistakable.
	var quick [10]core.RawEquipItem
	for i := range quick {
		quick[i] = core.RawEquipItem{ItemID: 0xB0000100 + uint32(i), EquipIndex: 0x200 + uint32(i)}
	}
	var pouch [6]core.RawEquipItem
	for i := range pouch {
		pouch[i] = core.RawEquipItem{ItemID: 0xB0000300 + uint32(i), EquipIndex: 0x400 + uint32(i)}
	}

	app := newEquipmentApp(buildEquipSlot(chrAsm, quick, pouch))

	before := make([]byte, len(app.save.Slots[0].Data))
	copy(before, app.save.Slots[0].Data)

	snap, err := app.GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}

	// Assert a group of views maps to the expected ChrAsm indices, in order.
	assertChrAsm := func(group string, views []EquipmentSlotView, indices []int) {
		t.Helper()
		for i, idx := range indices {
			want := chrAsmRaw(idx)
			if views[i].RawID != want {
				t.Errorf("%s[%d].RawID = 0x%08X, want ChrAsm[%d] = 0x%08X", group, i, views[i].RawID, idx, want)
			}
			if !views[i].Occupied {
				t.Errorf("%s[%d] not Occupied", group, i)
			}
		}
	}

	// Exact App-level UI order (see app_equipment.go GetEquipmentSnapshot).
	assertChrAsm("RightHandArmaments", snap.RightHandArmaments[:], []int{1, 3, 5})
	assertChrAsm("LeftHandArmaments", snap.LeftHandArmaments[:], []int{0, 2, 4})
	assertChrAsm("Arrows", snap.Arrows[:], []int{6, 8})
	assertChrAsm("Bolts", snap.Bolts[:], []int{7, 9})
	assertChrAsm("Armor", snap.Armor[:], []int{12, 13, 14, 15})
	assertChrAsm("Talismans", snap.Talismans[:], []int{17, 18, 19, 20})

	// Quick items expose item_id in pair order, never equip_index.
	for i := 0; i < 10; i++ {
		want := 0xB0000100 + uint32(i)
		if snap.QuickItems[i].RawID != want {
			t.Errorf("QuickItems[%d].RawID = 0x%08X, want item_id 0x%08X (equip_index leaked?)", i, snap.QuickItems[i].RawID, want)
		}
		if !snap.QuickItems[i].Occupied {
			t.Errorf("QuickItems[%d] not Occupied", i)
		}
	}
	// Pouch exposes item_id in pair order, never equip_index.
	for i := 0; i < 6; i++ {
		want := 0xB0000300 + uint32(i)
		if snap.Pouch[i].RawID != want {
			t.Errorf("Pouch[%d].RawID = 0x%08X, want item_id 0x%08X (equip_index leaked?)", i, snap.Pouch[i].RawID, want)
		}
		if !snap.Pouch[i].Occupied {
			t.Errorf("Pouch[%d] not Occupied", i)
		}
	}

	// Read-only: the endpoint must not mutate slot data.
	if string(before) != string(app.save.Slots[0].Data) {
		t.Error("GetEquipmentSnapshot mutated slot data")
	}
}

// TestGetEquipmentSnapshot_ResolvesKnownIDs is focused supplemental coverage:
// it seeds one armament, one ammo item, one goods item and one talisman with
// real DB-backed IDs (in their correct handle encoding) and confirms each
// resolves through the intended normalization path. It does not depend on names
// for the mapping — only that resolution succeeds and RawID is preserved.
func TestGetEquipmentSnapshot_ResolvesKnownIDs(t *testing.T) {
	var chrAsm [core.ChrAsmFieldCount]uint32
	// Armament (weapon): stored as itemID | 0x80000000; normalized via & 0x7FFFFFFF.
	const armamentRaw = 0x80000000 | 0x0131A230 // Magma Whip Candlestick
	chrAsm[1] = armamentRaw                     // RightHandArmaments[0]
	// Ammo: bare item ID, no normalization.
	const ammoRaw = 0x031BBEF0 // Bloodbone Bolt
	chrAsm[7] = ammoRaw        // Bolts[0]
	// Talisman: talisman handle (0xA0…); normalized to canonical 0x20 prefix.
	const talismanRaw = 0xA0000820 // Winged Sword Insignia (0x20000820)
	chrAsm[17] = talismanRaw       // Talismans[0]

	var quick [10]core.RawEquipItem
	// Goods: 0xB0 goods handle; normalized via HandleToItemID → 0x40… item ID.
	const goodsRaw = 0xB00050CA // Fire Blossom (0x400050CA)
	quick[0] = core.RawEquipItem{ItemID: goodsRaw, EquipIndex: 0x200}
	var pouch [6]core.RawEquipItem

	app := newEquipmentApp(buildEquipSlot(chrAsm, quick, pouch))
	snap, err := app.GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}

	cases := []struct {
		name    string
		view    EquipmentSlotView
		wantRaw uint32
	}{
		{"armament", snap.RightHandArmaments[0], armamentRaw},
		{"ammo", snap.Bolts[0], ammoRaw},
		{"talisman", snap.Talismans[0], talismanRaw},
		{"goods", snap.QuickItems[0], goodsRaw},
	}
	for _, c := range cases {
		if c.view.RawID != c.wantRaw {
			t.Errorf("%s RawID = 0x%08X, want 0x%08X", c.name, c.view.RawID, c.wantRaw)
		}
		if !c.view.Resolved || c.view.Name == "" {
			t.Errorf("%s did not resolve: Resolved=%v Name=%q (normalization path broken?)", c.name, c.view.Resolved, c.view.Name)
		}
	}
}

func TestResolveEquipView_WeaponUpgradeExcludesInfusionOffset(t *testing.T) {
	// Dismounter +4 with an affinity at offset +600. The stored value must
	// display its actual upgrade level, not the combined database offset +604.
	const dismounterInfusedPlusFour = 0x80000000 | (0x007A6020 + 604)

	view := resolveEquipView(dismounterInfusedPlusFour, classHandArmament)
	if !view.Resolved {
		t.Fatalf("Dismounter did not resolve: %+v", view)
	}
	if view.Name != "Dismounter +4" {
		t.Errorf("Name = %q, want %q", view.Name, "Dismounter +4")
	}
}

func TestGetEquipmentSnapshot_ResolvesBothWondrousPhysickPouchVariants(t *testing.T) {
	const (
		filledPhysickHandle = 0xB00000FA
		emptyPhysickHandle  = 0xB00000FB
		physickName         = "Flask of Wondrous Physick"
		physickIcon         = "items/tools/flask_of_wondrous_physick.png"
	)

	for _, raw := range []uint32{filledPhysickHandle, emptyPhysickHandle} {
		t.Run(fmt.Sprintf("0x%08X", raw), func(t *testing.T) {
			var pouch [6]core.RawEquipItem
			// UI top-right Quick Pouch slot is Pouch[1].
			pouch[1] = core.RawEquipItem{ItemID: raw, EquipIndex: 0x1E2}

			snap, err := newEquipmentApp(buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, pouch)).GetEquipmentSnapshot(0)
			if err != nil {
				t.Fatalf("GetEquipmentSnapshot: %v", err)
			}

			got := snap.Pouch[1]
			if got.RawID != raw {
				t.Errorf("RawID = 0x%08X, want original 0x%08X", got.RawID, raw)
			}
			if !got.Occupied || !got.Resolved {
				t.Errorf("Pouch[1] resolved state = occupied:%v resolved:%v, want both true", got.Occupied, got.Resolved)
			}
			if got.Name != physickName || got.IconPath != physickIcon {
				t.Errorf("Pouch[1] = name:%q icon:%q, want name:%q icon:%q", got.Name, got.IconPath, physickName, physickIcon)
			}
		})
	}
}

func TestGetEquipmentSnapshot_TreatsUnarmedAsEmptyHandSlot(t *testing.T) {
	var equipped [core.ChrAsmFieldCount]uint32
	equipped[1] = unarmedItemID // RightHandArmaments[0]

	snap, err := newEquipmentApp(buildEquipSlot(equipped, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})).GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	if got := snap.RightHandArmaments[0]; got.Occupied || got.Resolved || got.RawID != 0 || got.Name != "" || got.IconPath != "" {
		t.Errorf("Unarmed hand slot = %+v, want empty view", got)
	}
}

func TestGetEquipmentSnapshot_TreatsBareArmorAsEmpty(t *testing.T) {
	var equipped [core.ChrAsmFieldCount]uint32
	bareArmor := [4]uint32{0x10002710, 0x10002774, 0x100027D8, 0x1000283C}
	for i, itemID := range bareArmor {
		equipped[12+i] = itemID
	}

	snap, err := newEquipmentApp(buildEquipSlot(equipped, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})).GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	for i, view := range snap.Armor {
		if view.Occupied || view.Resolved || view.RawID != 0 || view.Name != "" || view.IconPath != "" {
			t.Errorf("bare armor slot %d = %+v, want empty view", i, view)
		}
	}
	if !snap.EquipLoadKnown || snap.CurrentEquipLoad != 0 {
		t.Errorf("bare armor Equip Load = %.1f / known=%v, want 0.0 / true", snap.CurrentEquipLoad, snap.EquipLoadKnown)
	}
}

func TestGetEquipmentSnapshot_UsesEnduranceForBaseEquipLoad(t *testing.T) {
	slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	slot.Player.Endurance = 20

	snap, err := newEquipmentApp(slot).GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	if snap.MaxEquipLoad != 64.1 {
		t.Errorf("MaxEquipLoad = %.1f, want 64.1", snap.MaxEquipLoad)
	}
}

func TestGetEquipmentSnapshot_AppliesPermanentEquipLoadModifiers(t *testing.T) {
	var equipped [core.ChrAsmFieldCount]uint32
	equipped[12] = 0x104F0A60 // Fire Knight Helm: +4.5% max Equip Load.
	equipped[17] = 0x20000408 // Great-Jar's Arsenal: +19%.
	equipped[18] = 0x20000412 // Erdtree's Favor +2: +8%.
	equipped[19] = 0x2000041A // Radagon's Scarseal: +3 Endurance.

	slot := buildEquipSlot(equipped, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	slot.Player.Endurance = 20

	snap, err := newEquipmentApp(slot).GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	// END 20 + 3 = 68.8. Direct bonuses stack additively: 19% + 8% + 4.5%.
	const want = 68.8 * 1.315
	if snap.MaxEquipLoad != want {
		t.Errorf("MaxEquipLoad = %.4f, want %.4f", snap.MaxEquipLoad, want)
	}
	if got, wantClass := snap.EquipLoadClass, string(core.EquipLoadLight); got != wantClass {
		t.Errorf("EquipLoadClass = %q, want %q", got, wantClass)
	}
}

func TestGetEquipmentSnapshot_IgnoresModifiersInLockedTalismanSlots(t *testing.T) {
	var equipped [core.ChrAsmFieldCount]uint32
	equipped[18] = 0x20000408 // Great-Jar's Arsenal, but this slot is locked.

	slot := buildEquipSlot(equipped, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
	slot.Player.Endurance = 20
	slot.Player.TalismanSlots = 0 // One usable slot: index 17 only.

	snap, err := newEquipmentApp(slot).GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	if snap.MaxEquipLoad != 64.1 {
		t.Errorf("MaxEquipLoad = %.1f, want base 64.1 without a locked-slot modifier", snap.MaxEquipLoad)
	}
}

func TestGetEquipmentSnapshot_UsesMemoryStonesAndMoonOfNokstellaForSpellSlots(t *testing.T) {
	for _, tc := range []struct {
		name                string
		memoryStones        uint32
		moonChrAsmIndex     int
		additionalTalismans uint8
		want                int
	}{
		{"base slots", 0, -1, 3, 2},
		{"all memory stones", 8, -1, 3, 10},
		{"memory stones clamp", 99, -1, 3, 10},
		{"moon of nokstella", 8, 17, 3, 12},
		{"moon in locked talisman field", 8, 20, 2, 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var equipped [core.ChrAsmFieldCount]uint32
			if tc.moonChrAsmIndex >= 0 {
				equipped[tc.moonChrAsmIndex] = moonOfNokstellaItemID
			}
			slot := buildEquipSlot(equipped, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
			slot.Player.TalismanSlots = tc.additionalTalismans
			slot.Inventory.CommonItems = []core.InventoryItem{{GaItemHandle: memoryStonesHandle, Quantity: tc.memoryStones}}

			snap, err := newEquipmentApp(slot).GetEquipmentSnapshot(0)
			if err != nil {
				t.Fatalf("GetEquipmentSnapshot: %v", err)
			}
			if snap.ActiveSpellSlots != tc.want {
				t.Errorf("ActiveSpellSlots = %d, want %d", snap.ActiveSpellSlots, tc.want)
			}
		})
	}
}

func TestGetEquipmentSnapshot_SumsCurrentEquipLoad(t *testing.T) {
	var equipped [core.ChrAsmFieldCount]uint32
	equipped[1] = 0x000F4240  // Dagger, 1.5 weight
	equipped[2] = 0x02082C11  // Frenzied Flame Seal, weightless
	equipped[17] = 0x200003E8 // Crimson Amber Medallion, 0.3 weight

	snap, err := newEquipmentApp(buildEquipSlot(equipped, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})).GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	if !snap.EquipLoadKnown {
		t.Fatal("EquipLoadKnown = false, want true")
	}
	if snap.CurrentEquipLoad < 1.79 || snap.CurrentEquipLoad > 1.81 {
		t.Errorf("CurrentEquipLoad = %.3f, want 1.8", snap.CurrentEquipLoad)
	}
	if snap.EquipLoadClass != string(core.EquipLoadLight) {
		t.Errorf("EquipLoadClass = %q, want %q", snap.EquipLoadClass, core.EquipLoadLight)
	}
}

func TestGetEquipmentSnapshot_HidesPartialEquipLoadForUnknownItem(t *testing.T) {
	var equipped [core.ChrAsmFieldCount]uint32
	equipped[1] = 0x000F4240 // known Dagger
	equipped[3] = 0x007FFFFF // unknown armament

	snap, err := newEquipmentApp(buildEquipSlot(equipped, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})).GetEquipmentSnapshot(0)
	if err != nil {
		t.Fatalf("GetEquipmentSnapshot: %v", err)
	}
	if snap.EquipLoadKnown {
		t.Error("EquipLoadKnown = true for an unknown equipped item")
	}
}

// writePhysickTears writes the two active tear IDs into the EquipPhysicsData
// block of a slot built by buildEquipSlot, at the exact offset ReadEquippedState
// reads from (armamentsOff + DynEquipedArmaments).
func writePhysickTears(slot core.SaveSlot, tear0, tear1 uint32) {
	projHeaderOff := testEquippedSpellsOffset + core.DynEquipedSpells + core.DynEquipedItems + core.DynEquipedGestures
	armamentsOff := projHeaderOff + 4 + testProjCount*8
	physicsOff := armamentsOff + core.DynEquipedArmaments
	binary.LittleEndian.PutUint32(slot.Data[physicsOff:], tear0)
	binary.LittleEndian.PutUint32(slot.Data[physicsOff+4:], tear1)
}

func TestGetEquipmentSnapshot_ResolvesPhysickTears(t *testing.T) {
	const (
		crimsonVariantRaw = 0x40002AFA // technical variant, display -> canonical Crimson Crystal Tear
		greenspillRaw     = 0x40002AF9 // standalone tear, must stay Greenspill (not a neighbour ID)
	)
	cases := []struct {
		name         string
		raw          uint32
		wantResolved bool
		wantName     string
	}{
		{"crimson-variant-canonical", crimsonVariantRaw, true, "Crimson Crystal Tear"},
		{"greenspill-standalone", greenspillRaw, true, "Greenspill Crystal Tear"},
		{"zero-unresolved", 0x00000000, false, "Unknown item (0x00000000)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
			writePhysickTears(slot, tc.raw, 0x40002AFE) // second tear: Speckled Hardtear, always resolvable

			snap, err := newEquipmentApp(slot).GetEquipmentSnapshot(0)
			if err != nil {
				t.Fatalf("GetEquipmentSnapshot: %v", err)
			}
			got := snap.Physick[0]
			if got.RawID != tc.raw {
				t.Errorf("RawID = 0x%08X, want native 0x%08X", got.RawID, tc.raw)
			}
			if got.Resolved != tc.wantResolved {
				t.Errorf("Resolved = %v, want %v", got.Resolved, tc.wantResolved)
			}
			if got.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tc.wantName)
			}
			// Unresolved tears must stay visible (occupied), never a silent empty slot.
			if !got.Occupied {
				t.Errorf("Physick[0].Occupied = false; unresolved tears must remain visible")
			}
			if tc.name == "crimson-variant-canonical" && strings.Contains(got.Name, "(Variant)") {
				t.Errorf("display name %q must not carry the technical (Variant) suffix", got.Name)
			}
		})
	}
}

// TestGetEquipmentSnapshot_PhysickSentinelIsEmpty proves the T545 contract: the
// 0xFFFFFFFF native sentinel yields an empty slot on either field with the raw
// value preserved, while a lone Greenspill tear in slot 2 stays resolved —
// confirming slot 1 / slot 2 independence and the physicsOff+0 / +4 mapping.
func TestGetEquipmentSnapshot_PhysickSentinelIsEmpty(t *testing.T) {
	const (
		sentinel      = 0xFFFFFFFF
		greenspillRaw = 0x40002AF9 // resolves to Greenspill Crystal Tear
	)
	assertEmptySentinel := func(t *testing.T, got EquipmentSlotView) {
		t.Helper()
		if got.Occupied {
			t.Errorf("Occupied = true, want false for sentinel")
		}
		if got.Resolved {
			t.Errorf("Resolved = true, want false for sentinel")
		}
		if got.RawID != sentinel {
			t.Errorf("RawID = 0x%08X, want 0x%08X", got.RawID, uint32(sentinel))
		}
	}

	t.Run("sentinel-slot1-empty", func(t *testing.T) {
		slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
		writePhysickTears(slot, sentinel, sentinel)
		snap, err := newEquipmentApp(slot).GetEquipmentSnapshot(0)
		if err != nil {
			t.Fatalf("GetEquipmentSnapshot: %v", err)
		}
		assertEmptySentinel(t, snap.Physick[0])
	})

	t.Run("sentinel-slot2-empty", func(t *testing.T) {
		slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
		writePhysickTears(slot, sentinel, sentinel)
		snap, err := newEquipmentApp(slot).GetEquipmentSnapshot(0)
		if err != nil {
			t.Fatalf("GetEquipmentSnapshot: %v", err)
		}
		assertEmptySentinel(t, snap.Physick[1])
	})

	// Game does not left-pack: an empty slot 1 with a lone tear in slot 2.
	t.Run("empty-slot1-greenspill-slot2", func(t *testing.T) {
		slot := buildEquipSlot([core.ChrAsmFieldCount]uint32{}, [10]core.RawEquipItem{}, [6]core.RawEquipItem{})
		writePhysickTears(slot, sentinel, greenspillRaw)
		snap, err := newEquipmentApp(slot).GetEquipmentSnapshot(0)
		if err != nil {
			t.Fatalf("GetEquipmentSnapshot: %v", err)
		}
		assertEmptySentinel(t, snap.Physick[0])
		got := snap.Physick[1]
		if !got.Occupied || !got.Resolved {
			t.Errorf("slot 2 = occupied:%v resolved:%v, want both true", got.Occupied, got.Resolved)
		}
		if got.Name != "Greenspill Crystal Tear" {
			t.Errorf("slot 2 Name = %q, want %q", got.Name, "Greenspill Crystal Tear")
		}
		if got.RawID != greenspillRaw {
			t.Errorf("slot 2 RawID = 0x%08X, want 0x%08X", got.RawID, uint32(greenspillRaw))
		}
	})
}
