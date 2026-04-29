package core

import (
	"encoding/binary"
	"os"
	"testing"
)

// pickActiveSlot loads a save and returns a pointer to the first active slot.
func pickActiveSlot(t *testing.T, savePath string) *SaveSlot {
	t.Helper()
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		t.Skipf("Test save not found: %s", savePath)
	}
	save, err := LoadSave(savePath)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	for i := 0; i < 10; i++ {
		s := &save.Slots[i]
		if s.Version != 0 && s.TutorialDataOffset > 0 {
			return s
		}
	}
	t.Fatalf("no active slot with computed TutorialDataOffset in %s", savePath)
	return nil
}

// TutorialData header (count, size) must be self-consistent for any active
// slot: count <= TutorialDataMaxIDs (255) AND size >= 4 + count*4. If this
// invariant fails, TutorialDataOffset likely points outside the real
// TutorialData section (regression check for offset-chain bugs in
// calculateDynamicOffsets / DynGaItemsOther / DynTutorialData).
func TestTutorialDataHeaderConsistency(t *testing.T) {
	slot := pickActiveSlot(t, "../../tmp/save/ER0000.sl2")

	off := slot.TutorialDataOffset
	if off+TutorialDataIDsOff > len(slot.Data) {
		t.Fatalf("TutorialDataOffset 0x%X out of bounds (slot len %d)", off, len(slot.Data))
	}
	size := binary.LittleEndian.Uint32(slot.Data[off+4:])
	count := binary.LittleEndian.Uint32(slot.Data[off+TutorialDataCountOff:])

	if count > TutorialDataMaxIDs {
		t.Errorf("TutorialData count %d exceeds max %d (likely reading from wrong offset)",
			count, TutorialDataMaxIDs)
	}
	minSize := uint32(4) + count*4
	if size < minSize {
		t.Errorf("TutorialData size 0x%X < required 4+count*4 = 0x%X (count=%d)", size, minSize, count)
	}
	if size > tutorialDataMaxData {
		t.Errorf("TutorialData size 0x%X exceeds max 0x%X", size, tutorialDataMaxData)
	}
}

// AppendTutorialID + ReadTutorialIDs round-trip on real save data: id must
// appear in the post-append read AND the rest of the slot.Data must remain
// byte-identical (only TutorialData payload may change).
func TestAppendTutorialIDOnlyTouchesTutorialPayload(t *testing.T) {
	slot := pickActiveSlot(t, "../../tmp/save/ER0000.sl2")

	off := slot.TutorialDataOffset
	size := binary.LittleEndian.Uint32(slot.Data[off+4:])
	payloadEnd := off + TutorialDataIDsOff + int(size) - 4
	if payloadEnd > len(slot.Data) {
		t.Fatalf("TutorialData payload [0x%X..0x%X) out of slot bounds", off, payloadEnd)
	}

	// Snapshot the bytes BEFORE TutorialData and AFTER TutorialData payload.
	preSnapshot := append([]byte(nil), slot.Data[:off+TutorialDataIDsOff]...)
	postSnapshot := append([]byte(nil), slot.Data[payloadEnd:]...)

	const probeID uint32 = 0x7FFE_0001
	if err := AppendTutorialID(slot, probeID); err != nil {
		t.Fatalf("AppendTutorialID failed: %v", err)
	}

	// Pre-region (everything up to count header inclusive) must change ONLY at
	// the count u32 (count incremented by 1).
	for i := 0; i < off+TutorialDataCountOff; i++ {
		if preSnapshot[i] != slot.Data[i] {
			t.Errorf("byte 0x%X outside TutorialData header changed (was 0x%02X, now 0x%02X)",
				i, preSnapshot[i], slot.Data[i])
			return
		}
	}
	// Post-region (everything after TutorialData payload) must be unchanged.
	for i := range postSnapshot {
		if postSnapshot[i] != slot.Data[payloadEnd+i] {
			t.Errorf("byte 0x%X after TutorialData changed (was 0x%02X, now 0x%02X)",
				payloadEnd+i, postSnapshot[i], slot.Data[payloadEnd+i])
			return
		}
	}

	// Probe id must be visible in the list.
	ids, err := ReadTutorialIDs(slot)
	if err != nil {
		t.Fatalf("ReadTutorialIDs failed after append: %v", err)
	}
	found := false
	for _, id := range ids {
		if id == probeID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("probe id 0x%X not found in TutorialData after append", probeID)
	}
}
