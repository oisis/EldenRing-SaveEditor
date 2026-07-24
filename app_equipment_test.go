package main

import (
	"encoding/binary"
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
		EquipItemsIDOffset:   testEquipItemsIDOffset,
		EquippedSpellsOffset: testEquippedSpellsOffset,
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
