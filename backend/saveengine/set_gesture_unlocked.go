package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	gestureEmptyRecord = uint32(0xFFFFFFFE)

	// GestureRingPreorderSlotID and GestureRingEarnedSlotID are the confirmed
	// canonical IDs of the one logical gesture resource that has two save forms.
	// The endpoint uses the same constants to validate its catalog declaration.
	GestureRingPreorderSlotID = uint32(227)
	GestureRingEarnedSlotID   = uint32(233)

	gestureRingLockedRecord = GestureRingEarnedSlotID - 1
)

// SetGestureUnlockedResult reports one committed gesture state change.
type SetGestureUnlockedResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	Unlocked      bool   `json:"unlocked"`
}

// SetGestureUnlocked assigns the unlock state of one canonical gesture slot in
// one active character. The public endpoint resolves catalog identity to the
// slot ID; SaveEngine alone interprets and mutates GestureGameData.
//
// Native records store an unlocked gesture as its odd canonical slot ID and a
// locked gesture as the preceding even value. Unlocking prefers that exact
// locked record and otherwise consumes the first native empty sentinel. Locking
// changes every duplicate of the exact canonical ID and leaves every unrelated,
// unknown and zero record untouched.
//
// Ring of Miquella is the one confirmed alias: 227 is the protected pre-order
// grant and 233 is the earnable form. A request for the logical resource uses
// 233. It is already unlocked when either form exists, never creates 227, and
// cannot be locked while 227 is present.
func (engine *Engine) SetGestureUnlocked(
	saveSessionID string,
	characterID int,
	slotID uint32,
	unlocked bool,
	expectedRevision string,
) (SetGestureUnlockedResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetGestureUnlockedResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if slotID == 0 || slotID&1 == 0 || slotID >= gestureEmptyRecord {
		return SetGestureUnlockedResult{}, fmt.Errorf(
			"gesture slot ID %d is not a supported canonical odd slot ID", slotID)
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

		sectionAt, err := gestureSectionStart(loaded, characterID)
		if err != nil {
			return err
		}
		before, err := loaded.snapshot.readAt(sectionAt, gestureSectionSize)
		if err != nil {
			return fmt.Errorf("cannot read gestures of character %d: %w", characterID, err)
		}

		after := append([]byte(nil), before...)
		records := make([]uint32, GestureSlotCount)
		for index := range records {
			records[index] = binary.LittleEndian.Uint32(after[index*gestureRecordSize:])
		}

		if err := assignGestureState(records, slotID, unlocked); err != nil {
			return err
		}
		for index, record := range records {
			binary.LittleEndian.PutUint32(after[index*gestureRecordSize:], record)
		}
		if bytes.Equal(before, after) {
			return nil
		}

		if err := loaded.snapshot.writeAt(sectionAt, after); err != nil {
			return fmt.Errorf("cannot write gestures of character %d: %w", characterID, err)
		}
		written, verifyErr := loaded.snapshot.readAt(sectionAt, gestureSectionSize)
		if verifyErr == nil && bytes.Equal(written, after) {
			return nil
		}

		if rollback := loaded.snapshot.writeAt(sectionAt, before); rollback != nil {
			return fmt.Errorf(
				"gestures of character %d could not be verified and could not be restored: %w",
				characterID, rollback)
		}
		return errors.New("gesture mutation could not be verified; the save is unchanged")
	})
	if err != nil {
		return SetGestureUnlockedResult{}, err
	}

	return SetGestureUnlockedResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  saveRevision,
		CharacterID:   characterID,
		Unlocked:      unlocked,
	}, nil
}

func assignGestureState(records []uint32, slotID uint32, unlocked bool) error {
	if slotID == GestureRingEarnedSlotID {
		return assignRingOfMiquellaState(records, unlocked)
	}

	if unlocked {
		if gestureRecordIndex(records, slotID) >= 0 {
			return nil
		}
		index := -1
		// Slot 1 is the protected Bow gesture. Its arithmetic predecessor is
		// zero, which is not a writable placeholder and must stay untouched.
		if slotID > 1 {
			index = gestureRecordIndex(records, slotID-1)
		}
		if index < 0 {
			index = gestureRecordIndex(records, gestureEmptyRecord)
		}
		if index < 0 {
			return fmt.Errorf("GestureGameData has no slot available for gesture %d", slotID)
		}
		records[index] = slotID
		return nil
	}

	if isProtectedGestureSlotID(slotID) && gestureRecordIndex(records, slotID) >= 0 {
		return fmt.Errorf("gesture %d is a protected starting gesture and cannot be locked", slotID)
	}
	for index, record := range records {
		if record == slotID {
			records[index] = slotID - 1
		}
	}
	return nil
}

func assignRingOfMiquellaState(records []uint32, unlocked bool) error {
	if unlocked {
		if gestureRecordIndex(records, GestureRingPreorderSlotID) >= 0 ||
			gestureRecordIndex(records, GestureRingEarnedSlotID) >= 0 {
			return nil
		}
		index := gestureRecordIndex(records, gestureRingLockedRecord)
		if index < 0 {
			index = gestureRecordIndex(records, gestureEmptyRecord)
		}
		if index < 0 {
			return errors.New("GestureGameData has no slot available for Ring of Miquella")
		}
		records[index] = GestureRingEarnedSlotID
		return nil
	}

	if gestureRecordIndex(records, GestureRingPreorderSlotID) >= 0 {
		return errors.New("Ring of Miquella is granted by pre-order slot 227 and cannot be locked")
	}
	for index, record := range records {
		if record == GestureRingEarnedSlotID {
			records[index] = gestureRingLockedRecord
		}
	}
	return nil
}

func gestureRecordIndex(records []uint32, wanted uint32) int {
	for index, record := range records {
		if record == wanted {
			return index
		}
	}
	return -1
}

func isProtectedGestureSlotID(slotID uint32) bool {
	switch slotID {
	case 1, 13, 15, 41, 43, 45, 47, 49, 101, 141, 161, 185, GestureRingPreorderSlotID:
		return true
	default:
		return false
	}
}
