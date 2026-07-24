package core

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// readerTestProjCount is the non-zero acquired-projectiles count baked into the
// fixture, so the test proves the reader skips 4 + projCount*8 bytes to reach
// the equipped-armaments block.
const readerTestProjCount = 3

// armamentsSlotValue is the distinct value the fixture writes for equipped slot
// idx inside the armaments block. Kept disjoint from the decoy header values.
func armamentsSlotValue(idx int) uint32 { return 0xEA000000 | uint32(idx) }

// decoyHeaderValue is written into the legacy EquipItemsID header. The reader
// must NOT return these — they mimic the encoded ga-handles found on real saves.
func decoyHeaderValue(idx int) uint32 { return 0x80010000 | uint32(idx) }

// makeReaderTestSlot builds a synthetic slot with parsed EquipItemsIDOffset and
// EquippedSpellsOffset. The 22 equipped values are written into the
// equipped-armaments block (past a non-zero projectiles section); decoy values
// are written into EquipItemsIDOffset so a regression back to that header fails.
// Quick/pouch pairs get distinct item_id / equip_index so a flat-u32 misread is
// detectable.
func makeReaderTestSlot() *SaveSlot {
	data := make([]byte, SlotSize)
	equipOff := 0x10000
	spellsOff := 0x20000

	// Decoy header — must be ignored by the reader.
	for i := 0; i < ChrAsmFieldCount; i++ {
		binary.LittleEndian.PutUint32(data[equipOff+i*4:], decoyHeaderValue(i))
	}

	// EquipItemData: quick pairs, active slot, pouch pairs.
	base := spellsOff + DynEquipedSpells
	for i := 0; i < equipItemDataQuickCount; i++ {
		off := base + i*8
		binary.LittleEndian.PutUint32(data[off:], uint32(0xB0000100+i)) // item_id
		binary.LittleEndian.PutUint32(data[off+4:], uint32(0x200+i))    // equip_index
	}
	binary.LittleEndian.PutUint32(data[base+equipItemDataActiveOff:], 5) // active_slot
	for i := 0; i < equipItemDataPouchCount; i++ {
		off := base + equipItemDataPouchOff + i*8
		binary.LittleEndian.PutUint32(data[off:], uint32(0xB0000300+i)) // item_id
		binary.LittleEndian.PutUint32(data[off+4:], uint32(0x400+i))    // equip_index
	}

	// Variable projectiles section + equipped-armaments block.
	projHeaderOff := spellsOff + DynEquipedSpells + DynEquipedItems + DynEquipedGestures
	binary.LittleEndian.PutUint32(data[projHeaderOff:], readerTestProjCount)
	armamentsOff := projHeaderOff + 4 + readerTestProjCount*8
	for i := 0; i < ChrAsmFieldCount; i++ {
		binary.LittleEndian.PutUint32(data[armamentsOff+i*4:], armamentsSlotValue(i))
	}

	return &SaveSlot{
		Data:                 data,
		EquipItemsIDOffset:   equipOff,
		EquippedSpellsOffset: spellsOff,
	}
}

func TestReadEquippedState_ArmamentsBlockMapping(t *testing.T) {
	slot := makeReaderTestSlot()
	st, err := slot.ReadEquippedState()
	if err != nil {
		t.Fatalf("ReadEquippedState: %v", err)
	}
	// Every equipped index must come from the armaments block, never the header.
	for i := 0; i < ChrAsmFieldCount; i++ {
		if got := st.Equipped[i]; got != armamentsSlotValue(i) {
			t.Errorf("Equipped[%d] = 0x%08X, want 0x%08X (read from EquipItemsID header instead?)",
				i, got, armamentsSlotValue(i))
		}
		if st.Equipped[i] == decoyHeaderValue(i) {
			t.Errorf("Equipped[%d] leaked decoy header value 0x%08X", i, st.Equipped[i])
		}
	}
	// Confirmed UI mapping indices (right 1/3/5, left 0/2/4, arrows 6/8,
	// bolts 7/9, armor 12-15, talismans 17-20) preserve their positions.
	for _, idx := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 12, 13, 14, 15, 17, 18, 19, 20} {
		if st.Equipped[idx] != armamentsSlotValue(idx) {
			t.Errorf("equipped slot idx %d misread", idx)
		}
	}
}

// TestReadEquippedState_OffsetArithmetic proves the reader lands at
// projHeaderOff + 4 + projCount*8: shifting projectile data by one entry must
// shift where the armaments block is read from.
func TestReadEquippedState_OffsetArithmetic(t *testing.T) {
	slot := makeReaderTestSlot()
	// Bump the projectile count by 1 without moving the armaments payload:
	// the reader should now read 8 bytes further and see zeros (payload moved
	// out from under it), proving the arithmetic is count-driven.
	projHeaderOff := slot.EquippedSpellsOffset + DynEquipedSpells + DynEquipedItems + DynEquipedGestures
	binary.LittleEndian.PutUint32(slot.Data[projHeaderOff:], readerTestProjCount+1)

	st, err := slot.ReadEquippedState()
	if err != nil {
		t.Fatalf("ReadEquippedState: %v", err)
	}
	if st.Equipped[0] == armamentsSlotValue(0) {
		t.Error("reader did not shift with projectile count (offset not 4 + count*8)")
	}
}

func TestReadEquippedState_QuickPouchPairLayout(t *testing.T) {
	slot := makeReaderTestSlot()
	st, err := slot.ReadEquippedState()
	if err != nil {
		t.Fatalf("ReadEquippedState: %v", err)
	}
	if st.ActiveQuick != 5 {
		t.Errorf("ActiveQuick = %d, want 5", st.ActiveQuick)
	}
	for i := 0; i < 10; i++ {
		wantID := uint32(0xB0000100 + i)
		wantIdx := uint32(0x200 + i)
		if st.QuickItems[i].ItemID != wantID {
			t.Errorf("QuickItems[%d].ItemID = 0x%X, want 0x%X", i, st.QuickItems[i].ItemID, wantID)
		}
		if st.QuickItems[i].EquipIndex != wantIdx {
			t.Errorf("QuickItems[%d].EquipIndex = 0x%X, want 0x%X (pair layout read as flat u32?)", i, st.QuickItems[i].EquipIndex, wantIdx)
		}
	}
	for i := 0; i < 6; i++ {
		wantID := uint32(0xB0000300 + i)
		wantIdx := uint32(0x400 + i)
		if st.Pouch[i].ItemID != wantID {
			t.Errorf("Pouch[%d].ItemID = 0x%X, want 0x%X", i, st.Pouch[i].ItemID, wantID)
		}
		if st.Pouch[i].EquipIndex != wantIdx {
			t.Errorf("Pouch[%d].EquipIndex = 0x%X, want 0x%X", i, st.Pouch[i].EquipIndex, wantIdx)
		}
	}
}

func TestReadEquippedState_EmptySentinels(t *testing.T) {
	slot := makeReaderTestSlot()
	// Empty equipped slots: 0x00000000 and 0xFFFFFFFF in the armaments block.
	projHeaderOff := slot.EquippedSpellsOffset + DynEquipedSpells + DynEquipedItems + DynEquipedGestures
	armamentsOff := projHeaderOff + 4 + readerTestProjCount*8
	binary.LittleEndian.PutUint32(slot.Data[armamentsOff+1*4:], 0x00000000)
	binary.LittleEndian.PutUint32(slot.Data[armamentsOff+3*4:], 0xFFFFFFFF)
	// Empty quick pair sentinel {0, 0xFFFFFFFF}.
	base := slot.EquippedSpellsOffset + DynEquipedSpells
	binary.LittleEndian.PutUint32(slot.Data[base:], 0x00000000)
	binary.LittleEndian.PutUint32(slot.Data[base+4:], 0xFFFFFFFF)

	st, err := slot.ReadEquippedState()
	if err != nil {
		t.Fatalf("ReadEquippedState: %v", err)
	}
	if st.Equipped[1] != 0 || st.Equipped[3] != 0xFFFFFFFF {
		t.Errorf("empty sentinels not preserved: [1]=0x%X [3]=0x%X", st.Equipped[1], st.Equipped[3])
	}
	if st.QuickItems[0].ItemID != 0 || st.QuickItems[0].EquipIndex != 0xFFFFFFFF {
		t.Errorf("empty quick sentinel not preserved: %+v", st.QuickItems[0])
	}
}

func TestReadEquippedState_DoesNotMutate(t *testing.T) {
	slot := makeReaderTestSlot()
	before := make([]byte, len(slot.Data))
	copy(before, slot.Data)

	if _, err := slot.ReadEquippedState(); err != nil {
		t.Fatalf("ReadEquippedState: %v", err)
	}
	if !bytes.Equal(before, slot.Data) {
		t.Error("ReadEquippedState mutated slot.Data")
	}
}

func TestReadEquippedState_InvalidSlot(t *testing.T) {
	// Unparsed offsets must error, not panic.
	slot := &SaveSlot{Data: make([]byte, SlotSize)}
	if _, err := slot.ReadEquippedState(); err == nil {
		t.Error("expected error for unparsed offsets")
	}
	// Out-of-bounds armaments block: EquippedSpellsOffset near end of buffer.
	slot2 := &SaveSlot{
		Data:                 make([]byte, SlotSize),
		EquipItemsIDOffset:   0x10000,
		EquippedSpellsOffset: SlotSize - 0x10,
	}
	if _, err := slot2.ReadEquippedState(); err == nil {
		t.Error("expected out-of-bounds error for armaments block")
	}
	// Invalid projectile count must be rejected, not used as an offset.
	slot3 := makeReaderTestSlot()
	projHeaderOff := slot3.EquippedSpellsOffset + DynEquipedSpells + DynEquipedItems + DynEquipedGestures
	binary.LittleEndian.PutUint32(slot3.Data[projHeaderOff:], MaxProjCount+1)
	if _, err := slot3.ReadEquippedState(); err == nil {
		t.Error("expected error for invalid projectile count")
	}
}
