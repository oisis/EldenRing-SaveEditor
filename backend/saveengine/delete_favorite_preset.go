package saveengine

import (
	"fmt"
)

// DeleteFavoritePresetResult reports one committed Mirror Favorites deletion.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat.
type DeleteFavoritePresetResult struct {
	MutationReceipt
	FavoriteSlotID int `json:"favoriteSlotID"`
}

// DeleteFavoritePreset clears the 0x130 bytes of one active Mirror Favorites
// preset slot in UserData10 under expectedRevision control. If the slot is
// inactive (missing the "FACE" magic), no bytes are changed, but the global
// revision advances and marks the session dirty.
func (engine *Engine) DeleteFavoritePreset(
	saveSessionID string,
	favoriteSlotID int,
	expectedRevision string,
) (DeleteFavoritePresetResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return DeleteFavoritePresetResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if favoriteSlotID < 0 || favoriteSlotID >= favoriteSlotCount {
		return DeleteFavoritePresetResult{}, fmt.Errorf(
			"favoriteSlotID %d is outside the range 0..%d", favoriteSlotID, favoriteSlotCount-1)
	}

	committed, err := engine.commitRevision(saveSessionID, kindDeleteFavoritePreset, func(loaded *loadedSave) error {
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return fmt.Errorf(
				"expectedRevision %q does not match the current saveRevision %q",
				expectedRevision, current)
		}

		base := userData10Base(loaded.session.platform)
		active, err := readFavoriteSlotActive(loaded.snapshot, base, favoriteSlotID)
		if err != nil {
			return err
		}
		if !active {
			return nil
		}

		slotAt := favoriteSlotOffset(base, favoriteSlotID)
		if err := applyByteWrites(loaded.snapshot, []byteWrite{{
			at:   slotAt,
			data: make([]byte, favoriteSlotSize),
		}}); err != nil {
			return fmt.Errorf("cannot delete favorite preset slot %d: %w", favoriteSlotID, err)
		}
		return nil
	})
	if err != nil {
		return DeleteFavoritePresetResult{}, err
	}

	return DeleteFavoritePresetResult{
		MutationReceipt: committed,
		FavoriteSlotID:  favoriteSlotID,
	}, nil
}
