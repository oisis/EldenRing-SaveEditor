package core

import (
	"encoding/binary"
	"fmt"
)

// RawEquipItem is one {item_id, equip_index} pair from the EquipItemData
// section (quick items and pouch). A pair is 8 bytes: item_id (u32) followed
// by equip_index (u32), both little-endian. Empty slots use the observed
// sentinel {item_id: 0, equip_index: 0xFFFFFFFF}.
type RawEquipItem struct {
	ItemID     uint32
	EquipIndex uint32
}

// RawEquippedState holds the raw equipped values read from a slot, exactly as
// stored on disk — no ID normalization, no DB resolution. It is produced by a
// pure reader that never mutates SaveSlot.Data.
type RawEquippedState struct {
	// Equipped holds the 22 u32 slot values read from the equipped-armaments
	// block — the full equipment state that follows the variable-length
	// acquired-projectiles section. Unlike the EquipItemsID header, this block
	// stores direct item IDs (weapons/armor as itemID|0x80000000, ammo/talismans
	// bare), so db.GetItemDataFuzzy can resolve them. Index meaning matches the UI
	// slot layout (e.g. [1]/[3]/[5] right-hand armaments, [12]-[15] armor,
	// [17]-[20] talismans).
	Equipped    [ChrAsmFieldCount]uint32
	QuickItems  [10]RawEquipItem
	Pouch       [6]RawEquipItem
	ActiveQuick int32
}

// EquipItemData section layout (see tmp/equipment/equipped-state-research.md §3.2):
// 10 quick pairs (0x50), then active_slot i32 (0x54), then 6 pouch pairs.
// This is the correct per-slot pair structure — NOT the flat 16×u32 hash
// window that readQuickItemIDs uses.
const (
	equipItemDataQuickCount = 10
	equipItemDataPouchCount = 6
	equipItemDataActiveOff  = equipItemDataQuickCount * 8 // 0x50
	equipItemDataPouchOff   = equipItemDataActiveOff + 4  // 0x54
)

// ReadEquippedState reads the raw equipped item state for the slot. It is a
// pure reader: it never mutates s.Data and never calls any writer / hash /
// repack / save path. All reads are bounds-checked.
func (s *SaveSlot) ReadEquippedState() (RawEquippedState, error) {
	var st RawEquippedState
	if s == nil {
		return st, fmt.Errorf("ReadEquippedState: nil slot")
	}
	if s.EquippedSpellsOffset <= 0 {
		return st, fmt.Errorf("ReadEquippedState: offsets not parsed")
	}

	// The 22 equipped slot values live in the equipped-armaments block, which
	// starts after the variable-length acquired-projectiles section:
	//   projHeader → projectileCount u32 → projectileCount×8 bytes → armaments (0x9C).
	// The EquipItemsID header (s.EquipItemsIDOffset) is intentionally NOT used:
	// on real PC/PS4 saves it holds encoded ga-handles the item DB cannot resolve.
	projHeaderOff := s.EquippedSpellsOffset + DynEquipedSpells + DynEquipedItems + DynEquipedGestures
	if projHeaderOff < 0 || projHeaderOff+4 > len(s.Data) {
		return st, fmt.Errorf("ReadEquippedState: projectile header out of bounds")
	}
	projCount := int(binary.LittleEndian.Uint32(s.Data[projHeaderOff:]))
	if projCount < 0 || projCount > MaxProjCount {
		return st, fmt.Errorf("ReadEquippedState: invalid projectile count %d", projCount)
	}
	armamentsOff := projHeaderOff + 4 + projCount*8
	if armamentsOff < projHeaderOff || armamentsOff+DynEquipedArmaments > len(s.Data) {
		return st, fmt.Errorf("ReadEquippedState: equipped-armaments block out of bounds")
	}
	for i := 0; i < ChrAsmFieldCount; i++ {
		st.Equipped[i] = binary.LittleEndian.Uint32(s.Data[armamentsOff+i*4:])
	}

	base := s.EquippedSpellsOffset + DynEquipedSpells // start of EquipItemData
	if base < 0 || base+DynEquipedItems > len(s.Data) {
		return st, fmt.Errorf("ReadEquippedState: EquipItemData section out of bounds")
	}
	for i := 0; i < equipItemDataQuickCount; i++ {
		off := base + i*8
		st.QuickItems[i] = RawEquipItem{
			ItemID:     binary.LittleEndian.Uint32(s.Data[off:]),
			EquipIndex: binary.LittleEndian.Uint32(s.Data[off+4:]),
		}
	}
	st.ActiveQuick = int32(binary.LittleEndian.Uint32(s.Data[base+equipItemDataActiveOff:]))
	for i := 0; i < equipItemDataPouchCount; i++ {
		off := base + equipItemDataPouchOff + i*8
		st.Pouch[i] = RawEquipItem{
			ItemID:     binary.LittleEndian.Uint32(s.Data[off:]),
			EquipIndex: binary.LittleEndian.Uint32(s.Data[off+4:]),
		}
	}
	return st, nil
}
