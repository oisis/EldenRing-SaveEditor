package core

import "fmt"

// RebuildSlot serializes a SaveSlot into a fresh 0x280000-byte buffer.
//
// Step 0 (current): identity passthrough — returns a copy of slot.Data.
// Subsequent steps will replace this with a section-by-section rebuild
// driven by SaveSlot.SectionMap, with unlocked_regions reserialized from
// slot.UnlockedRegions (the only true variable-size section we need to
// mutate for Stage 2 of the Invasion Regions feature).
//
// Reference: tmp/repos/er-save-manager/src/er_save_manager/parser/slot_rebuild.py
// (rebuild_slot_with_map). Our hybrid approach treats most sections as opaque
// blobs rather than parsing each into a struct.
func RebuildSlot(slot *SaveSlot) ([]byte, error) {
	if slot == nil {
		return nil, fmt.Errorf("RebuildSlot: nil slot")
	}
	if len(slot.Data) != SlotSize {
		return nil, fmt.Errorf("RebuildSlot: slot.Data size %d, want %d", len(slot.Data), SlotSize)
	}

	out := make([]byte, SlotSize)
	copy(out, slot.Data)
	return out, nil
}
