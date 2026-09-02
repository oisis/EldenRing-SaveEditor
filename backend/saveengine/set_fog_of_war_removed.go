package saveengine

import (
	"bytes"
	"errors"
	"fmt"
)

// Confirmed bounds of the global Fog of War bitfield, measured from the first
// byte behind the dynamic UnlockedRegions list. SaveForge 1.5.8 and 1.6.8 carry
// the identical pair — 0x087E and an inclusive 0x10B0 — and their fill loop runs
// over exactly these bytes.
//
// The prefix in front of 0x087E holds structured horse and bloodstain data, and
// the first byte behind 0x10B0 belongs to MenuProfile, so both ends are hard
// limits rather than a comfort margin.
const (
	fogOfWarFieldStart = 0x087E
	fogOfWarFieldEnd   = 0x10B0
	fogOfWarFieldSize  = fogOfWarFieldEnd - fogOfWarFieldStart + 1
)

// SetFogOfWarRemovedResult reports one committed Fog of War removal.
//
// SaveRevision is the revision the change committed under, which is the previous
// one plus exactly 1.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type SetFogOfWarRemovedResult struct {
	MutationReceipt
	CharacterID int  `json:"characterID"`
	Removed     bool `json:"removed"`
}

// SetFogOfWarRemoved fills the global Fog of War bitfield of one character slot
// with 0xFF, removing the cosmetic grey exploration overlay.
//
// This is the semantics SaveForge 1.5.8 and 1.6.8 shared byte for byte in
// applyRemoveFogOfWar: an in-place fill of 2099 bytes, no data shifting, no slot
// rebuild and no offset recalculation. Repeating it leaves the same state.
//
// removed=false is rejected fail-closed before anything is read or written. The
// bitfield is a per-tile exploration mask whose bit-to-tile mapping is unknown,
// so zeroing it would not restore an earlier exploration state — it would
// destroy the one the save still carries. Neither legacy version implemented the
// inverse either.
//
// The field is not a map region, not an event flag and not an item: no
// MapRegion, UnlockedRegions entry, map fragment, event flag, DLC cover-layer
// coordinate, Inventory or Storage byte is read or written here.
//
// expectedRevision must be a canonical decimal saveRevision and match the
// session's current revision exactly. An inactive slot is rejected fail-closed.
func (engine *Engine) SetFogOfWarRemoved(
	saveSessionID string,
	characterID int,
	removed bool,
	expectedRevision string,
) (SetFogOfWarRemovedResult, error) {
	if !removed {
		return SetFogOfWarRemovedResult{}, errors.New(
			"removed must be true; restoring Fog of War has no confirmed contract")
	}
	if !isCanonicalRevision(expectedRevision) {
		return SetFogOfWarRemovedResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetFogOfWarRemoved, characterID, func(loaded *loadedSave) error {
		if characterID < 0 || characterID >= characterSlotCount {
			return fmt.Errorf("characterID %d is outside the range 0..%d",
				characterID, characterSlotCount-1)
		}
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return fmt.Errorf(
				"expectedRevision %q does not match the current saveRevision %q",
				expectedRevision, current)
		}
		active, err := loaded.snapshot.readAt(
			userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
		}
		if active[0] != userData10ActiveFlagValue {
			return fmt.Errorf("character %d is not active", characterID)
		}

		// The region list is the only dynamic part in front of the field. Its
		// shared locator validates the declared count and keeps every offset in
		// int64, so a corrupt length can never wrap into a small, seemingly
		// valid position. Only the logical slot bound is checked here: the
		// mandatory undo point already read the complete physical slot, and
		// applyByteWrites bounds-checks the plan again before it writes.
		countAt, count, slotEnd, err := unlockedRegionsBounds(loaded, characterID)
		if err != nil {
			return err
		}
		fieldAt := countAt + 4 + count*regionRecordSize + fogOfWarFieldStart
		if fieldAt+fogOfWarFieldSize > slotEnd {
			return fmt.Errorf(
				"Fog of War bitfield of character %d does not fit into its slot", characterID)
		}
		writes := []byteWrite{{at: fieldAt, data: bytes.Repeat([]byte{0xFF}, fogOfWarFieldSize)}}
		if err := applyByteWrites(loaded.snapshot, writes); err != nil {
			return fmt.Errorf(
				"cannot remove the Fog of War of character %d: %w", characterID, err)
		}
		return nil
	})
	if err != nil {
		return SetFogOfWarRemovedResult{}, err
	}

	return SetFogOfWarRemovedResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		Removed:         true,
	}, nil
}
