package saveengine

import "fmt"

// mapRegionVisibilityBlock is the single event flag block the confirmed map
// region visibility table occupies.
//
// Block 63 — the transient "Map Fragment acquired" pickup triggers the game
// itself sets and clears — is deliberately not accepted, because SaveForge 1.5.8
// and 1.6.8 never wrote it from this operation either, despite the misleading
// comment their SetMapRegionFlags carried. The system map display flags and the
// sub-region flags that produce black tiles are outside this mutation for the
// same reason: the caller cannot name a flag at all, and the endpoint above only
// resolves the curated safe table.
const mapRegionVisibilityBlock = 62

// SetMapRegionRevealedResult reports one committed map region visibility change.
//
// SaveRevision is the revision the change committed under, which is the previous
// one plus exactly 1. Catalog identity does not belong to SaveEngine and is
// added by the endpoint receipt.
type SetMapRegionRevealedResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	Revealed      bool   `json:"revealed"`
}

// SetMapRegionRevealed sets or clears the visibility of one map region in a
// character slot of an existing session and, when the region has one, keeps its
// Map Fragment item in step.
//
// visibleEventFlagID is the confirmed visibility flag of the region and has to
// lie in block 62. mapFragmentGameID is the goods game ID of the Map Fragment
// the catalog pairs with that flag, or 0 for a region that has none. Both are
// fully resolved by the endpoint; SaveEngine derives neither from the other.
//
// This is the semantics SaveForge 1.5.8 and 1.6.8 shared byte for byte in
// applyMapRegionUnlock: revealing writes the visibility bit and adds the
// fragment, hiding clears the bit and removes the fragment, and the acquired
// flag of block 63 is never touched in either direction.
//
// expectedRevision must be a canonical decimal saveRevision and match the
// session's current revision exactly. An inactive slot is rejected fail-closed.
//
// The flag byte and the Inventory record form one plan, so the two either commit
// together or leave the snapshot, the dirty state and the revision untouched.
// That closes the gap SaveForge 1.x left open, where a failed item add kept the
// flag it had already written.
func (engine *Engine) SetMapRegionRevealed(
	saveSessionID string,
	characterID int,
	visibleEventFlagID uint32,
	mapFragmentGameID uint32,
	revealed bool,
	expectedRevision string,
) (SetMapRegionRevealedResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetMapRegionRevealedResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if block := visibleEventFlagID / eventFlagsPerBlock; block != mapRegionVisibilityBlock {
		return SetMapRegionRevealedResult{}, fmt.Errorf(
			"map region event flag %d lies in block %d, want block %d",
			visibleEventFlagID, block, mapRegionVisibilityBlock)
	}
	position, err := resolveEventFlag(visibleEventFlagID)
	if err != nil {
		return SetMapRegionRevealedResult{}, err
	}
	// Every one of the 24 confirmed Map Fragments is a goods item, so its handle
	// is derived from the ID and no GaItem record has to be allocated. Anything
	// else is refused here rather than reaching the record planner.
	if mapFragmentGameID != 0 && mapFragmentGameID&gaItemHandleTypeMask != 0x40000000 {
		return SetMapRegionRevealedResult{}, fmt.Errorf(
			"Map Fragment game ID 0x%08X is not a goods ID", mapFragmentGameID)
	}

	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetMapRegionRevealed, characterID, func(loaded *loadedSave) error {
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

		// The Inventory side is planned first, because it is the part that can
		// run out of records, and nothing may be written before it is known to
		// fit.
		var writes []byteWrite
		if mapFragmentGameID != 0 {
			records, err := inventoryHeldRecords(loaded, characterID)
			if err != nil {
				return err
			}
			writes, err = planItemPresence(
				loaded, characterID, records, mapFragmentGameID, revealed, "Map Fragment")
			if err != nil {
				return err
			}
		}

		sectionAt, err := eventFlagSectionStart(loaded, characterID)
		if err != nil {
			return err
		}
		flagAt := sectionAt + position.offset
		raw, err := loaded.snapshot.readAt(flagAt, 1)
		if err != nil {
			return fmt.Errorf("cannot read event flag %d of character %d: %w",
				visibleEventFlagID, characterID, err)
		}
		updated := raw[0] &^ (1 << position.bit)
		if revealed {
			updated = raw[0] | (1 << position.bit)
		}
		writes = append(writes, byteWrite{at: flagAt, data: []byte{updated}})

		if err := applyByteWrites(loaded.snapshot, writes); err != nil {
			return fmt.Errorf("cannot set map region flag %d: %w", visibleEventFlagID, err)
		}
		return nil
	})
	if err != nil {
		return SetMapRegionRevealedResult{}, err
	}

	return SetMapRegionRevealedResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  committed.SaveRevision,
		CharacterID:   characterID,
		Revealed:      revealed,
	}, nil
}
