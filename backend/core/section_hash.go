package core

import "fmt"

// PlayerGameDataHash — 128 bytes (0x80), 11×u32 + 0x54 raw bytes.
//
// Reference: tmp/repos/er-save-manager/parser/world.py:PlayerGameDataHash
//
// Spec/22 describes the length as `slot_end - position`, but in practice
// every save we have observed places this 128-byte block at SlotSize - 0x80
// and any remaining tail bytes are zero padding (captured separately as
// SaveSlot.RestPadding by the rebuild parser).
type PlayerGameDataHash struct {
	Level                       uint32
	Stats                       uint32
	Archetype                   uint32
	PlayerGameData0xc0          uint32
	Padding                     uint32
	Runes                       uint32
	RunesMemory                 uint32
	EquippedWeapons             uint32
	EquippedArmorsAndTalismans  uint32
	EquippedItems               uint32
	EquippedSpells              uint32
	Rest                        [0x54]byte
}

const PlayerGameDataHashSize = 11*4 + 0x54 // 128

func (h *PlayerGameDataHash) Read(r *Reader) error {
	fields := []*uint32{
		&h.Level, &h.Stats, &h.Archetype, &h.PlayerGameData0xc0,
		&h.Padding, &h.Runes, &h.RunesMemory, &h.EquippedWeapons,
		&h.EquippedArmorsAndTalismans, &h.EquippedItems, &h.EquippedSpells,
	}
	for i, dst := range fields {
		v, err := r.ReadU32()
		if err != nil {
			return fmt.Errorf("hash.field[%d]: %w", i, err)
		}
		*dst = v
	}
	rest, err := r.ReadBytes(0x54)
	if err != nil {
		return fmt.Errorf("hash.rest: %w", err)
	}
	copy(h.Rest[:], rest)
	return nil
}

func (h *PlayerGameDataHash) Write(w *SectionWriter) {
	w.WriteU32(h.Level)
	w.WriteU32(h.Stats)
	w.WriteU32(h.Archetype)
	w.WriteU32(h.PlayerGameData0xc0)
	w.WriteU32(h.Padding)
	w.WriteU32(h.Runes)
	w.WriteU32(h.RunesMemory)
	w.WriteU32(h.EquippedWeapons)
	w.WriteU32(h.EquippedArmorsAndTalismans)
	w.WriteU32(h.EquippedItems)
	w.WriteU32(h.EquippedSpells)
	w.WriteBytes(h.Rest[:])
}
