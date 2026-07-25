package core

import (
	"encoding/binary"
	"fmt"
)

// EquippedSpellSlotCount is the number of spell slots in the EquippedSpells
// section (memory + skills shortcuts). Mirrors the 14-slot constant used by
// readSpellIDs in hash.go.
const EquippedSpellSlotCount = 14

// EquippedSpellSlotSize is the per-slot byte stride: spell_id u32 LE followed
// by follower/unk u32 LE.
const EquippedSpellSlotSize = 8

// EquippedSpellActiveIndexOffset is the u32 immediately after the fourteen
// spell records. It is a zero-based index into the currently equipped compact
// spell list.
const EquippedSpellActiveIndexOffset = EquippedSpellSlotCount * EquippedSpellSlotSize

// EquippedSpellEmptySentinel marks an empty spell slot. Both spell_id ==
// 0xFFFFFFFF AND follower == 0x00000000 are required by the game; mixing
// these is a corrupt state.
const EquippedSpellEmptySentinel uint32 = 0xFFFFFFFF

// EquippedSpellOccupiedFollower is written into the follower/unk u32 for any
// occupied slot. The lower-level meaning of this field (follower toggle / unk)
// is not relevant here; vanilla saves consistently use 0xFFFFFFFF for every
// occupied spell slot regardless of which spell is equipped.
const EquippedSpellOccupiedFollower uint32 = 0xFFFFFFFF

// PatchEquippedSpell writes a single spell slot in the EquippedSpells section
// of slot.Data.
//
// spellID is the raw MagicParam ID (e.g. Catch Flame = 0x1770), NOT a full
// item ID (the 0x40XXXXXX / 0x60XXXXXX prefixed form used elsewhere in the
// save). Passing EquippedSpellEmptySentinel (0xFFFFFFFF) clears the slot.
//
// Semantics:
//
//	spellID == 0xFFFFFFFF → write (spell_id=0xFFFFFFFF, follower=0x00000000)
//	spellID != 0xFFFFFFFF → write (spell_id=spellID,   follower=0xFFFFFFFF)
//
// Out of scope: this writer does NOT touch the slot hash block. Callers that
// want the in-save hash refreshed must invoke ComputeSlotHash separately, the
// same way other low-level core writers (PatchWeaponItemID, etc.) leave the
// hash update to the apply layer.
//
// Errors are returned WITHOUT mutating slot.Data. Idempotent writes (target
// bytes already match) are skipped.
func PatchEquippedSpell(slot *SaveSlot, slotIndex int, spellID uint32) error {
	if slot == nil {
		return fmt.Errorf("PatchEquippedSpell: slot is nil")
	}
	if slotIndex < 0 || slotIndex >= EquippedSpellSlotCount {
		return fmt.Errorf("PatchEquippedSpell: slotIndex %d out of range [0,%d)",
			slotIndex, EquippedSpellSlotCount)
	}
	if slot.EquippedSpellsOffset <= 0 {
		return fmt.Errorf("PatchEquippedSpell: EquippedSpellsOffset not initialised (got %d); call calculateDynamicOffsets first",
			slot.EquippedSpellsOffset)
	}

	off := slot.EquippedSpellsOffset + slotIndex*EquippedSpellSlotSize
	end := off + EquippedSpellSlotSize
	if end > len(slot.Data) {
		return fmt.Errorf("PatchEquippedSpell: slot %d at offset 0x%X exceeds Data length %d",
			slotIndex, off, len(slot.Data))
	}

	var newSpellID, newFollower uint32
	if spellID == EquippedSpellEmptySentinel {
		newSpellID = EquippedSpellEmptySentinel
		newFollower = 0x00000000
	} else {
		newSpellID = spellID
		newFollower = EquippedSpellOccupiedFollower
	}

	curSpellID := binary.LittleEndian.Uint32(slot.Data[off:])
	curFollower := binary.LittleEndian.Uint32(slot.Data[off+4:])
	if curSpellID == newSpellID && curFollower == newFollower {
		return nil
	}

	binary.LittleEndian.PutUint32(slot.Data[off:], newSpellID)
	binary.LittleEndian.PutUint32(slot.Data[off+4:], newFollower)
	return nil
}

// CalculateDynamicOffsets is the exported wrapper over the
// package-internal calculateDynamicOffsets. Phase 7d.3 introduces it so
// the apply-spells test fixtures (in package main) can materialise a
// calibrated SaveSlot from a hand-built buffer without running the full
// Read pipeline. The production code paths still call the unexported
// form via parseFromData; this wrapper exists solely as a test seam.
func (s *SaveSlot) CalculateDynamicOffsets() error {
	return s.calculateDynamicOffsets()
}

// SpellWrite is a single equipped-spell mutation request. SlotIndex is
// 0..13 (matches the save's spell slot ordering); SpellID is a raw
// MagicParam ID (e.g. 0x1770 for Catch Flame) or
// EquippedSpellEmptySentinel (0xFFFFFFFF) to clear the slot.
//
// Templates v2 stores spells with full DB-style item IDs (0x40XXXXXX);
// the apply layer is responsible for stripping the prefix via
// db.ItemIDToMagicParamID before constructing SpellWrite entries.
type SpellWrite struct {
	SlotIndex int
	SpellID   uint32
}

// WriteCompactSpells replaces the complete equipped spell list with spellIDs.
// The supplied raw MagicParam IDs are written contiguously from slot 0 and the
// remaining records are reset to the native empty pair (0xFFFFFFFF, 0).
//
// This is deliberately separate from WriteSpells, whose partial-patch
// semantics preserve unselected records. Use WriteCompactSpells only when the
// caller owns the entire requested loadout. A non-empty list is required: the
// native active-index semantics for an empty loadout have not been established.
//
// The method preserves the hash block and does not read or write active_index.
// The caller deliberately owns the gameplay semantics of a selected spell
// whose compact-list position changes; this low-level writer only persists the
// requested compact list. All validation occurs before mutation.
func (s *SaveSlot) WriteCompactSpells(spellIDs []uint32) error {
	return s.writeCompactSpells(spellIDs, nil)
}

// WriteCompactSpellsWithActiveIndex replaces a compact spell loadout and writes
// its active HUD index atomically. activeIndex must point at one of spellIDs.
// It is intended for the high-level application path after it has applied the
// native normalization contract; callers that merely patch spell records must
// keep using WriteCompactSpells.
func (s *SaveSlot) WriteCompactSpellsWithActiveIndex(spellIDs []uint32, activeIndex uint32) error {
	return s.writeCompactSpells(spellIDs, &activeIndex)
}

func (s *SaveSlot) writeCompactSpells(spellIDs []uint32, activeIndex *uint32) error {
	if s == nil {
		return fmt.Errorf("WriteCompactSpells: nil slot")
	}
	if len(spellIDs) == 0 || len(spellIDs) > EquippedSpellSlotCount {
		return fmt.Errorf("WriteCompactSpells: spell count %d out of range [1,%d]", len(spellIDs), EquippedSpellSlotCount)
	}
	if s.EquippedSpellsOffset <= 0 {
		return fmt.Errorf("WriteCompactSpells: EquippedSpellsOffset not initialised (got %d); call calculateDynamicOffsets first", s.EquippedSpellsOffset)
	}
	sectionEnd := s.EquippedSpellsOffset + EquippedSpellSlotCount*EquippedSpellSlotSize
	if activeIndex != nil {
		if *activeIndex >= uint32(len(spellIDs)) {
			return fmt.Errorf("WriteCompactSpells: active index %d out of range for %d spells", *activeIndex, len(spellIDs))
		}
		sectionEnd += 4
	}
	if sectionEnd > len(s.Data) {
		return fmt.Errorf("WriteCompactSpells: EquippedSpells section out of bounds (offset 0x%X, Data length %d)", s.EquippedSpellsOffset, len(s.Data))
	}
	for i, spellID := range spellIDs {
		if spellID == 0 || spellID == EquippedSpellEmptySentinel {
			return fmt.Errorf("WriteCompactSpells[%d]: spell ID 0x%08X is not valid in a compact loadout", i, spellID)
		}
	}
	for i := 0; i < EquippedSpellSlotCount; i++ {
		off := s.EquippedSpellsOffset + i*EquippedSpellSlotSize
		spellID := EquippedSpellEmptySentinel
		follower := uint32(0)
		if i < len(spellIDs) {
			spellID = spellIDs[i]
			follower = EquippedSpellOccupiedFollower
		}
		binary.LittleEndian.PutUint32(s.Data[off:], spellID)
		binary.LittleEndian.PutUint32(s.Data[off+4:], follower)
	}
	if activeIndex != nil {
		binary.LittleEndian.PutUint32(s.Data[s.EquippedSpellsOffset+EquippedSpellActiveIndexOffset:], *activeIndex)
	}
	return nil
}

// WriteSpells is the batch equivalent of PatchEquippedSpell.
//
// Atomicity: every write is structurally validated (slot index range +
// duplicate detection) BEFORE any byte is mutated. Any validation
// failure returns the error without touching slot.Data, matching
// WriteEquipment's no-partial-write invariant. Per-write semantic
// validation (offset bounds, nil slot, etc.) is delegated to
// PatchEquippedSpell, which itself never mutates on failure.
// It does not compact the list; full-loadout callers must use
// WriteCompactSpells.
//
// Hash discipline: this writer never touches the slot hash block. Native cold
// saves keep hash[10] zero. A game-written save immediately after changing the
// spell list may hold another transient value, which Elden Ring normalizes on
// the next cold save and which does not equal ComputeSlotHash. Preserving the
// loaded value avoids synthesizing an unproven intermediate representation.
//
// Concurrency: callers that share a SaveSlot across goroutines must
// hold the slot-level lock for the entire WriteSpells call.
func (s *SaveSlot) WriteSpells(writes []SpellWrite) error {
	if s == nil {
		return fmt.Errorf("WriteSpells: nil slot")
	}
	if s.EquippedSpellsOffset <= 0 {
		return fmt.Errorf("WriteSpells: EquippedSpellsOffset not initialised (got %d); call calculateDynamicOffsets first", s.EquippedSpellsOffset)
	}
	if s.EquippedSpellsOffset+EquippedSpellSlotCount*EquippedSpellSlotSize > len(s.Data) {
		return fmt.Errorf("WriteSpells: EquippedSpells section out of bounds (offset 0x%X, Data length %d)", s.EquippedSpellsOffset, len(s.Data))
	}
	if len(writes) == 0 {
		return nil
	}

	seen := make(map[int]int, len(writes))
	for i, w := range writes {
		if w.SlotIndex < 0 || w.SlotIndex >= EquippedSpellSlotCount {
			return fmt.Errorf("WriteSpells[%d]: slotIndex %d out of range [0,%d)", i, w.SlotIndex, EquippedSpellSlotCount)
		}
		if prev, dup := seen[w.SlotIndex]; dup {
			return fmt.Errorf("WriteSpells[%d]: slot index %d already written at writes[%d]", i, w.SlotIndex, prev)
		}
		seen[w.SlotIndex] = i
	}

	for _, w := range writes {
		if err := PatchEquippedSpell(s, w.SlotIndex, w.SpellID); err != nil {
			return fmt.Errorf("WriteSpells: %w", err)
		}
	}

	return nil
}
