package core

import (
	"encoding/binary"
	"fmt"
)

// TutorialData layout in slot.Data at slot.TutorialDataOffset:
//
//	+0x00 (u16): unk0x0
//	+0x02 (u16): unk0x2
//	+0x04 (u32): size — total chunk size in bytes (typically 0x400)
//	+0x08 (u32): count — number of tutorial IDs in the list
//	+0x0C: count × u32 tutorial IDs (TutorialParam row IDs)
//
// When the game first triggers a tutorial popup (or first hands a tutorial-bound
// item like an "About *" or Crafting Kit pickup), it appends the corresponding
// TutorialParam row ID to this list. Subsequent triggers check the list and
// skip if the ID is already present — so pre-populating the list lets the
// editor prevent the "About item drops on ground" pickup duplicate.
//
// Source: er-save-manager src/er_save_manager/parser/world.py (TutorialDataChunk).
// Verified empirically by save diff: buying Crafting Kit at Kalé added ID 2010.
const (
	TutorialDataHeaderLen = 8  // unk0x0 + unk0x2 + size
	TutorialDataCountOff  = 8  // u32 count after header
	TutorialDataIDsOff    = 12 // first ID after count
	TutorialDataMaxIDs    = 0xFF // (size - 4) / 4 = (0x400 - 4) / 4 = 255
)

// ReadTutorialIDs returns the list of tutorial IDs currently registered in the
// slot's TutorialData block. Returns empty slice + error if offset is invalid.
func ReadTutorialIDs(slot *SaveSlot) ([]uint32, error) {
	if slot == nil || slot.TutorialDataOffset <= 0 {
		return nil, fmt.Errorf("tutorial data offset not computed")
	}
	off := slot.TutorialDataOffset
	if off+TutorialDataIDsOff > len(slot.Data) {
		return nil, fmt.Errorf("tutorial data offset 0x%X out of bounds (slot len %d)", off, len(slot.Data))
	}
	count := binary.LittleEndian.Uint32(slot.Data[off+TutorialDataCountOff:])
	size := binary.LittleEndian.Uint32(slot.Data[off+4:])
	maxFromSize := uint32(0)
	if size >= 4 {
		maxFromSize = (size - 4) / 4
	}
	if count > maxFromSize || count > TutorialDataMaxIDs {
		return nil, fmt.Errorf("tutorial count %d out of range (size=0x%X)", count, size)
	}
	end := off + TutorialDataIDsOff + int(count)*4
	if end > len(slot.Data) {
		return nil, fmt.Errorf("tutorial id array end 0x%X out of bounds", end)
	}
	ids := make([]uint32, count)
	for i := uint32(0); i < count; i++ {
		ids[i] = binary.LittleEndian.Uint32(slot.Data[off+TutorialDataIDsOff+int(i)*4:])
	}
	return ids, nil
}

// HasTutorialID returns true if the given tutorial ID is already in the list.
func HasTutorialID(slot *SaveSlot, id uint32) (bool, error) {
	ids, err := ReadTutorialIDs(slot)
	if err != nil {
		return false, err
	}
	for _, existing := range ids {
		if existing == id {
			return true, nil
		}
	}
	return false, nil
}

// AppendTutorialID adds a tutorial ID to the slot's TutorialData list.
// Idempotent: if the ID already exists, returns nil without modification.
// Returns an error if the list is full (255 IDs) or offset chain failed.
func AppendTutorialID(slot *SaveSlot, id uint32) error {
	if slot == nil || slot.TutorialDataOffset <= 0 {
		return fmt.Errorf("tutorial data offset not computed")
	}
	off := slot.TutorialDataOffset
	if off+TutorialDataIDsOff > len(slot.Data) {
		return fmt.Errorf("tutorial data offset 0x%X out of bounds", off)
	}
	count := binary.LittleEndian.Uint32(slot.Data[off+TutorialDataCountOff:])
	size := binary.LittleEndian.Uint32(slot.Data[off+4:])

	// Idempotency: scan existing IDs.
	for i := uint32(0); i < count; i++ {
		entry := binary.LittleEndian.Uint32(slot.Data[off+TutorialDataIDsOff+int(i)*4:])
		if entry == id {
			return nil
		}
	}

	// Bounds check: 4-byte count header + (count+1) × 4 must fit in size.
	used := 4 + (count+1)*4
	if size > 0 && used > size {
		return fmt.Errorf("tutorial list full (count=%d, size=0x%X)", count, size)
	}
	if count >= TutorialDataMaxIDs {
		return fmt.Errorf("tutorial list at hard cap %d", TutorialDataMaxIDs)
	}

	// Append.
	binary.LittleEndian.PutUint32(slot.Data[off+TutorialDataIDsOff+int(count)*4:], id)
	binary.LittleEndian.PutUint32(slot.Data[off+TutorialDataCountOff:], count+1)
	return nil
}
