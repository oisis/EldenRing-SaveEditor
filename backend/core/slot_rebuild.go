package core

import (
	"encoding/binary"
	"fmt"
)

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
// Hybrid blob strategy:
//   - Most sections are copied verbatim from slot.Data.
//   - "unlocked_regions" is reserialized from slot.UnlockedRegions so the
//     count header and id array stay consistent if the slice was mutated.
//   - "dlc" and "player_data_hash" are anchored to their fixed offsets
//     (DlcSectionOffset, HashOffset) — they always end the slot.
//   - The "post_unlocked_regions" blob (everything between regions and DLC)
//     absorbs any size delta produced by mutating UnlockedRegions:
//       * regions shrank → blob is right-zero-padded to fill the gap;
//       * regions grew   → trailing bytes of the blob are trimmed.
//     Whether trimming is safe depends on slack at the blob's tail
//     (Step 3 / spec/30 will quantify this).
//
// For an unmodified slot (len(UnlockedRegions) unchanged) this produces
// byte-for-byte identical output to slot.Data.
//
// Reference: tmp/repos/er-save-manager/src/er_save_manager/parser/slot_rebuild.py
// (rebuild_slot_with_map).
func RebuildSlot(slot *SaveSlot) ([]byte, error) {
	if slot == nil {
		return nil, fmt.Errorf("RebuildSlot: nil slot")
	}
	if len(slot.Data) != SlotSize {
		return nil, fmt.Errorf("RebuildSlot: slot.Data size %d, want %d", len(slot.Data), SlotSize)
	}
	if len(slot.SectionMap) == 0 {
		return nil, fmt.Errorf("RebuildSlot: SectionMap not populated")
	}

	out := make([]byte, SlotSize)

	// Empty / unparseable slot — single covering section, verbatim copy.
	if len(slot.SectionMap) == 1 && slot.SectionMap[0].Name == SectionEmptySlot {
		copy(out, slot.Data)
		return out, nil
	}

	cursor := 0
	for _, sec := range slot.SectionMap {
		switch sec.Name {

		case SectionUnlockedRegs:
			if cursor+4+4*len(slot.UnlockedRegions) > DlcSectionOffset {
				return nil, fmt.Errorf("RebuildSlot: unlocked_regions (count=%d) overflows DlcSectionOffset",
					len(slot.UnlockedRegions))
			}
			binary.LittleEndian.PutUint32(out[cursor:], uint32(len(slot.UnlockedRegions)))
			cursor += 4
			for _, id := range slot.UnlockedRegions {
				binary.LittleEndian.PutUint32(out[cursor:], id)
				cursor += 4
			}

		case SectionPostUnlockedRegs:
			// Stretch / shrink the blob to keep DLC anchored at DlcSectionOffset.
			avail := DlcSectionOffset - cursor
			if avail < 0 {
				return nil, fmt.Errorf("RebuildSlot: cursor 0x%X already past DlcSectionOffset 0x%X",
					cursor, DlcSectionOffset)
			}
			blob := slot.Data[sec.Start:sec.End]
			n := len(blob)
			if n > avail {
				n = avail
			}
			copy(out[cursor:], blob[:n])
			// Bytes [cursor+n, cursor+avail) stay zero (out is zero-initialised).
			cursor += avail

		case SectionDLC:
			if cursor != DlcSectionOffset {
				return nil, fmt.Errorf("RebuildSlot: DLC misaligned — cursor 0x%X, want 0x%X",
					cursor, DlcSectionOffset)
			}
			copy(out[cursor:], slot.Data[sec.Start:sec.End])
			cursor += sec.Size()

		case SectionHash:
			if cursor != HashOffset {
				return nil, fmt.Errorf("RebuildSlot: hash misaligned — cursor 0x%X, want 0x%X",
					cursor, HashOffset)
			}
			copy(out[cursor:], slot.Data[sec.Start:sec.End])
			cursor += sec.Size()

		default:
			// Pre-regions blob and any future fixed-size sections.
			copy(out[cursor:], slot.Data[sec.Start:sec.End])
			cursor += sec.Size()
		}
	}

	if cursor != SlotSize {
		return nil, fmt.Errorf("RebuildSlot: ended at cursor 0x%X, want SlotSize 0x%X", cursor, SlotSize)
	}
	return out, nil
}
