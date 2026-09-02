package saveengine

import (
	"errors"
	"fmt"
)

// SetSummoningPoolActivatedResult reports one committed activation state change.
//
// SaveRevision is the revision the change committed under, which is the previous
// one plus exactly 1. SaveSessionID and CharacterID match the request. Activated
// is the new state stored in the save event flags. Catalog identity does not
// belong to SaveEngine and is therefore added by the endpoint receipt.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type SetSummoningPoolActivatedResult struct {
	MutationReceipt
	CharacterID int  `json:"characterID"`
	Activated   bool `json:"activated"`
}

// summoningPoolFlagBlock is the one confirmed event-flag block the activation
// flags of every curated summoning pool lie in. The catalog enforces the same
// bound on its documents; this method restates it so a flag of another block can
// never be flipped through the summoning pool entry point.
const summoningPoolFlagBlock = 670

// SetSummoningPoolActivated sets or clears the event flag bit associated with a
// summoning pool resource in one character slot of an existing session.
//
// saveSessionID is matched exactly. characterID is the physical slot 0..9.
//
// expectedRevision must be a canonical decimal saveRevision and match the
// session's current revision exactly.
//
// eventFlagID is the confirmed activation event flag of the pool. Only block 670
// is accepted; any other block is rejected.
//
// An inactive slot is rejected fail-closed, leaving the session and snapshot
// untouched.
//
// The mutation runs inside one critical section under Engine.mutex. The write is
// verified, and a failed verification restores the previous byte without
// advancing the revision or marking the session dirty. Persisting the snapshot
// stays the job of WriteSave; no file on disk is opened here.
func (engine *Engine) SetSummoningPoolActivated(
	saveSessionID string,
	characterID int,
	eventFlagID uint32,
	activated bool,
	expectedRevision string,
) (SetSummoningPoolActivatedResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetSummoningPoolActivatedResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if block := eventFlagID / eventFlagsPerBlock; block != summoningPoolFlagBlock {
		return SetSummoningPoolActivatedResult{}, fmt.Errorf(
			"event flag %d lies in block %d, which is not the confirmed summoning pool block %d",
			eventFlagID, block, summoningPoolFlagBlock)
	}

	position, err := resolveEventFlag(eventFlagID)
	if err != nil {
		return SetSummoningPoolActivatedResult{}, err
	}

	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetSummoningPoolActivated, characterID, func(loaded *loadedSave) error {
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
		if activated {
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
		return SetSummoningPoolActivatedResult{}, err
	}

	return SetSummoningPoolActivatedResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		Activated:       activated,
	}, nil
}
