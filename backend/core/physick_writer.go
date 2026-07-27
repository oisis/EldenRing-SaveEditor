package core

import (
	"encoding/binary"
	"fmt"
)

// PhysickSlotKind identifies one of the two active Wondrous Physick tear fields
// in the EquipPhysicsData block. Slot1 is physicsOff+0, Slot2 is physicsOff+4.
// The game does not left-pack, so a single tear may legitimately live in Slot2
// while Slot1 stays empty.
type PhysickSlotKind int

const (
	PhysickSlot1 PhysickSlotKind = iota
	PhysickSlot2
)

const physickSlotCount = 2

// PhysickWrite is one resolved Wondrous Physick tear-field write. Value is the
// exact u32 to store: a bare GoodsParam tear ID (0x40…, already derived from the
// owned inventory handle via db.HandleToItemID) or GaHandleInvalid to clear the
// slot. This writer performs no ID normalization and no display-alias
// canonicalization — a technical variant such as Crimson 0x40002AFA is written
// through byte-for-byte.
type PhysickWrite struct {
	Slot  PhysickSlotKind
	Value uint32
}

// WritePhysick writes the two active Wondrous Physick tear fields atomically.
//
// It touches ONLY the first two u32 of the EquipPhysicsData block (the active
// mixture). The trailing third u32, every equipped-armaments field, quick
// items, pouch, spells, TutorialData, EventFlags and inventory pickups are all
// left untouched. Slot MD5 is recalculated by SaveFile.SaveFile as usual.
//
// Empty slots use GaHandleInvalid (0xFFFFFFFF); clearing never left-packs the
// surviving tear. A single tear may not occupy both slots at once.
//
// Concurrency: callers that share a SaveSlot across goroutines must hold the
// slot-level lock for the entire WritePhysick call.
func (s *SaveSlot) WritePhysick(writes []PhysickWrite) error {
	if s == nil {
		return fmt.Errorf("WritePhysick: nil slot")
	}
	if len(writes) == 0 {
		return nil
	}
	armamentsOff, err := s.equippedArmamentsOffset()
	if err != nil {
		return fmt.Errorf("WritePhysick: %w", err)
	}
	// EquipPhysicsData (DynEquipePhysics = 0x0C) starts right after the 0x9C
	// armaments block; its first two u32 are the active tears, the third is left
	// intact.
	physicsOff := armamentsOff + DynEquipedArmaments
	if physicsOff < armamentsOff || physicsOff+DynEquipePhysics > len(s.Data) {
		return fmt.Errorf("WritePhysick: EquipPhysicsData block out of bounds")
	}

	// Resolve the final pair by starting from the current on-disk mixture and
	// applying each write, so a batch that touches only one slot preserves the
	// other, and duplicate detection sees the whole final state.
	final := [physickSlotCount]uint32{
		binary.LittleEndian.Uint32(s.Data[physicsOff:]),
		binary.LittleEndian.Uint32(s.Data[physicsOff+4:]),
	}
	seen := make(map[int]int, len(writes))
	for i, w := range writes {
		idx := int(w.Slot)
		if idx < 0 || idx >= physickSlotCount {
			return fmt.Errorf("WritePhysick[%d]: unsupported slot kind %d", i, idx)
		}
		if prev, dup := seen[idx]; dup {
			return fmt.Errorf("WritePhysick[%d]: slot %d already written at writes[%d]", i, idx, prev)
		}
		seen[idx] = i
		final[idx] = w.Value
	}
	if final[0] != GaHandleInvalid && final[0] == final[1] {
		return fmt.Errorf("WritePhysick: tear 0x%08X cannot occupy both Physick slots", final[0])
	}

	// All validation passed — write only the two mixture fields.
	binary.LittleEndian.PutUint32(s.Data[physicsOff:], final[0])
	binary.LittleEndian.PutUint32(s.Data[physicsOff+4:], final[1])
	return nil
}
