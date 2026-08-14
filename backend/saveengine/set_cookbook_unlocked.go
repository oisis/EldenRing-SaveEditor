package saveengine

import (
	"errors"
	"fmt"
)

// SetCookbookUnlockedResult reports one committed cookbook unlock state change.
//
// SaveRevision is the revision the change committed under, which is the previous
// one plus exactly 1. SaveSessionID and CharacterID match the request. Unlocked
// is the new unlock state stored in the save event flags. Catalog identity does
// not belong to SaveEngine and is therefore added by the endpoint receipt.
type SetCookbookUnlockedResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	Unlocked      bool   `json:"unlocked"`
}

// SetCookbookUnlocked sets or clears the event flag bit associated with a
// cookbook resource in one character slot of an existing session.
//
// saveSessionID is matched exactly. characterID is the physical slot 0..9.
//
// expectedRevision must be a canonical decimal saveRevision and match the
// session's current revision exactly.
//
// eventFlagID is the confirmed event flag identifier belonging to the cookbook.
// Only blocks 67 and 68 are supported; any other block is rejected.
//
// An inactive slot is rejected fail-closed, leaving the session and snapshot
// untouched.
//
// The mutation runs inside one critical section under Engine.mutex. The write is
// verified, and a failed verification restores the previous byte without
// advancing the revision or marking the session dirty.
func (engine *Engine) SetCookbookUnlocked(
	saveSessionID string,
	characterID int,
	eventFlagID uint32,
	unlocked bool,
	expectedRevision string,
) (SetCookbookUnlockedResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetCookbookUnlockedResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	block := eventFlagID / eventFlagsPerBlock
	if block != 67 && block != 68 {
		return SetCookbookUnlockedResult{}, fmt.Errorf(
			"event flag %d lies in block %d, which this reader does not support",
			eventFlagID, block)
	}

	position, err := resolveEventFlag(eventFlagID)
	if err != nil {
		return SetCookbookUnlockedResult{}, err
	}

	saveRevision, err := engine.commitRevision(saveSessionID, func(loaded *loadedSave) error {
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

		flag, err := loaded.snapshot.readAt(
			userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
		}
		if flag[0] != userData10ActiveFlagValue {
			return fmt.Errorf("character %d is not active", characterID)
		}

		sectionAt, err := eventFlagSectionStart(loaded, characterID)
		if err != nil {
			return err
		}

		byteOffset := sectionAt + position.offset
		oldRaw, err := loaded.snapshot.readAt(byteOffset, 1)
		if err != nil {
			return fmt.Errorf("cannot read event flag %d of character %d: %w", eventFlagID, characterID, err)
		}

		oldByte := oldRaw[0]
		var newByte byte
		if unlocked {
			newByte = oldByte | (1 << position.bit)
		} else {
			newByte = oldByte &^ (1 << position.bit)
		}

		if err := loaded.snapshot.writeAt(byteOffset, []byte{newByte}); err != nil {
			return fmt.Errorf("cannot write event flag %d of character %d: %w", eventFlagID, characterID, err)
		}

		written, err := loaded.snapshot.readAt(byteOffset, 1)
		if err == nil && written[0] == newByte {
			return nil
		}

		// Verification failed: roll back to original byte.
		if rollback := loaded.snapshot.writeAt(byteOffset, []byte{oldByte}); rollback != nil {
			return fmt.Errorf(
				"event flag %d of character %d could not be verified and could not be restored: %w",
				eventFlagID, characterID, rollback)
		}
		return errors.New("event flag mutation could not be verified; the save is unchanged")
	})
	if err != nil {
		return SetCookbookUnlockedResult{}, err
	}

	return SetCookbookUnlockedResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  saveRevision,
		CharacterID:   characterID,
		Unlocked:      unlocked,
	}, nil
}
