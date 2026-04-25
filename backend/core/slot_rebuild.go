package core

import "fmt"

// Section names emitted by buildSectionMap. Stable identifiers so callers
// (rebuild logic, tests, diagnostics) can reference sections by name.
const (
	SectionEmptySlot       = "empty_slot"
	SectionPreUnlockedRegs = "pre_unlocked_regions"
	SectionUnlockedRegs    = "unlocked_regions"
	SectionPostUnlockedRegs = "post_unlocked_regions"
	SectionDLC             = "dlc"
	SectionHash            = "player_data_hash"
)

// SectionRange describes a contiguous byte range inside slot.Data.
// End is exclusive. Size returns End - Start.
type SectionRange struct {
	Name  string
	Start int
	End   int
}

// Size returns the byte length of the section.
func (r SectionRange) Size() int { return r.End - r.Start }

// buildSectionMap computes the section boundaries used by RebuildSlot.
//
// For empty / unparseable slots (Version == 0 or UnlockedRegionsOffset == 0)
// we emit a single "empty_slot" section spanning the whole 0x280000 buffer —
// rebuilding such a slot is a verbatim copy.
//
// For active slots we emit five sections that together cover [0, SlotSize):
//   pre_unlocked_regions   [0, UnlockedRegionsOffset)
//   unlocked_regions       [UnlockedRegionsOffset, UnlockedRegionsOffset + 4 + 4*N)
//   post_unlocked_regions  [unlocked_regions_end, DlcSectionOffset)
//   dlc                    [DlcSectionOffset, DlcSectionOffset + DlcSectionSize)
//   player_data_hash       [HashOffset, HashOffset + HashSize)
//
// The post_unlocked_regions blob folds together every section after the
// regions block (horse, blood_stain, ..., event_flags, world_area, net_man,
// weather, time, base_version, steam_id, ps5_activity). Future steps may
// split it further if a struct rebuild is needed for any of them.
func (s *SaveSlot) buildSectionMap() error {
	if len(s.Data) != SlotSize {
		return fmt.Errorf("buildSectionMap: slot.Data size %d, want %d", len(s.Data), SlotSize)
	}

	// Empty / unparseable slot — single covering section.
	if s.Version == 0 || s.UnlockedRegionsOffset == 0 {
		s.SectionMap = []SectionRange{{Name: SectionEmptySlot, Start: 0, End: SlotSize}}
		return nil
	}

	regionsStart := s.UnlockedRegionsOffset
	regionsEnd := regionsStart + 4 + 4*len(s.UnlockedRegions)

	if regionsStart < 0 || regionsStart > DlcSectionOffset {
		return fmt.Errorf("buildSectionMap: UnlockedRegionsOffset 0x%X outside valid range", regionsStart)
	}
	if regionsEnd > DlcSectionOffset {
		return fmt.Errorf("buildSectionMap: regions end 0x%X past DlcSectionOffset 0x%X",
			regionsEnd, DlcSectionOffset)
	}

	s.SectionMap = []SectionRange{
		{Name: SectionPreUnlockedRegs, Start: 0, End: regionsStart},
		{Name: SectionUnlockedRegs, Start: regionsStart, End: regionsEnd},
		{Name: SectionPostUnlockedRegs, Start: regionsEnd, End: DlcSectionOffset},
		{Name: SectionDLC, Start: DlcSectionOffset, End: DlcSectionOffset + DlcSectionSize},
		{Name: SectionHash, Start: HashOffset, End: HashOffset + HashSize},
	}
	return validateSectionMap(s.SectionMap)
}

// validateSectionMap checks that sections cover [0, SlotSize) contiguously,
// in ascending order, with no gaps and no overlaps.
func validateSectionMap(sections []SectionRange) error {
	if len(sections) == 0 {
		return fmt.Errorf("section map is empty")
	}
	if sections[0].Start != 0 {
		return fmt.Errorf("first section %q starts at 0x%X, want 0", sections[0].Name, sections[0].Start)
	}
	for i, sec := range sections {
		if sec.End <= sec.Start {
			return fmt.Errorf("section %q has non-positive size [0x%X, 0x%X)", sec.Name, sec.Start, sec.End)
		}
		if i > 0 && sec.Start != sections[i-1].End {
			return fmt.Errorf("gap or overlap between %q and %q (0x%X != 0x%X)",
				sections[i-1].Name, sec.Name, sections[i-1].End, sec.Start)
		}
	}
	last := sections[len(sections)-1]
	if last.End != SlotSize {
		return fmt.Errorf("last section %q ends at 0x%X, want SlotSize 0x%X", last.Name, last.End, SlotSize)
	}
	return nil
}

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
