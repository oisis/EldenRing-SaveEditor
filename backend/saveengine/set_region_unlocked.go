package saveengine

import (
	"fmt"
	"slices"
)

// SetRegionUnlockedResult reports one committed region unlock membership change.
//
// SaveRevision is the revision the change committed under, which is the previous
// one plus exactly 1. Catalog identity does not belong to SaveEngine and is
// added by the endpoint receipt.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type SetRegionUnlockedResult struct {
	MutationReceipt
	CharacterID int  `json:"characterID"`
	Unlocked    bool `json:"unlocked"`
}

// SetRegionUnlocked unlocks or locks one raw regionID in a character slot of an
// existing save session by updating its membership in the UnlockedRegions list.
//
// All other raw entries — zeros, duplicates, unknown IDs, and their exact order —
// are preserved without sorting or deduplicating the whole list.
//
// expectedRevision must be a canonical decimal saveRevision and match the
// session's current revision exactly. An inactive slot is rejected fail-closed.
func (engine *Engine) SetRegionUnlocked(
	saveSessionID string,
	characterID int,
	regionID uint32,
	unlocked bool,
	expectedRevision string,
) (SetRegionUnlockedResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetRegionUnlockedResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	committed, err := engine.commitCharacterRevision(
		saveSessionID,
		kindSetRegionUnlocked,
		characterID,
		func(loaded *loadedSave) error {
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

			currentRegions, err := readUnlockedRegions(loaded, characterID)
			if err != nil {
				return err
			}

			var nextRegions []uint32
			if unlocked {
				found := false
				for _, id := range currentRegions {
					if id == regionID {
						found = true
						break
					}
				}
				if found {
					nextRegions = currentRegions
				} else {
					nextRegions = make([]uint32, len(currentRegions)+1)
					copy(nextRegions, currentRegions)
					nextRegions[len(currentRegions)] = regionID
				}
			} else {
				nextRegions = make([]uint32, 0, len(currentRegions))
				for _, id := range currentRegions {
					if id != regionID {
						nextRegions = append(nextRegions, id)
					}
				}
			}

			if slices.Equal(currentRegions, nextRegions) {
				return nil
			}

			rebuiltSlot, err := rebuildSlotWithRegions(loaded, characterID, nextRegions)
			if err != nil {
				return err
			}

			slotBase, _ := eventFlagSlotBounds(loaded.session.platform, characterID)
			origSlot, err := loaded.snapshot.readAt(slotBase, characterSlotDataSize)
			if err != nil {
				return fmt.Errorf("cannot read character %d slot before write: %w", characterID, err)
			}

			restoreFailure := func(cause error) error {
				if restoreErr := loaded.snapshot.writeAt(slotBase, origSlot); restoreErr != nil {
					return fmt.Errorf("character %d rollback write failed (%w) after: %v", characterID, restoreErr, cause)
				}
				if !loaded.snapshot.sameAt(slotBase, origSlot) {
					return fmt.Errorf("character %d rollback verification failed after: %v", characterID, cause)
				}
				return cause
			}

			if err := loaded.snapshot.writeAt(slotBase, rebuiltSlot); err != nil {
				return restoreFailure(fmt.Errorf("character %d slot data could not be written: %w", characterID, err))
			}
			if !loaded.snapshot.sameAt(slotBase, rebuiltSlot) {
				return restoreFailure(fmt.Errorf("the rewritten slot data of character %d could not be verified", characterID))
			}
			verifiedRegions, err := readUnlockedRegions(loaded, characterID)
			if err != nil {
				return restoreFailure(fmt.Errorf("the rebuilt regions of character %d could not be read: %w", characterID, err))
			}
			if !slices.Equal(verifiedRegions, nextRegions) {
				return restoreFailure(fmt.Errorf("the rebuilt regions of character %d do not match expected list", characterID))
			}
			if _, err := eventFlagSectionStart(loaded, characterID); err != nil {
				return restoreFailure(fmt.Errorf("the event flag section of character %d could not be resolved after rebuild: %w", characterID, err))
			}
			return nil
		},
	)
	if err != nil {
		return SetRegionUnlockedResult{}, err
	}

	return SetRegionUnlockedResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		Unlocked:        unlocked,
	}, nil
}
