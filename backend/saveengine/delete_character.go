package saveengine

import (
	"bytes"
	"fmt"
)

// DeleteCharacterResult identifies the physical slot removed by one committed
// deletion. No deleted save bytes are exposed in the receipt.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat.
type DeleteCharacterResult struct {
	MutationReceipt
	CharacterID int `json:"characterID"`
}

// DeleteCharacter permanently clears one active or residual character in
// place. It changes exactly the slot data, its UserData10 activity flag and its
// complete ProfileSummary; no neighbouring slot is shifted or rewritten.
func (engine *Engine) DeleteCharacter(
	saveSessionID string,
	characterID int,
	expectedRevision string,
) (DeleteCharacterResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return DeleteCharacterResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	committed, err := engine.commitCharacterRevision(saveSessionID, kindDeleteCharacter, characterID, func(loaded *loadedSave) error {
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

		slotAt := slotDataBase(loaded.session.platform, characterID)
		flagAt := userData10Base(loaded.session.platform) +
			userData10ActiveFlagsOffset + int64(characterID)
		summaryAt := userData10Base(loaded.session.platform) + userData10SummaryOffset +
			int64(characterID)*userData10SummaryStride

		slotBefore, err := loaded.snapshot.readAt(slotAt, characterSlotDataSize)
		if err != nil {
			return fmt.Errorf("cannot read data of character %d: %w", characterID, err)
		}
		flagBefore, err := loaded.snapshot.readAt(flagAt, 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
		}
		summaryBefore, err := loaded.snapshot.readAt(summaryAt, userData10SummaryStride)
		if err != nil {
			return fmt.Errorf("cannot read profile summary of character %d: %w", characterID, err)
		}

		occupied, err := characterCanBeDeleted(
			loaded, characterID, flagBefore[0], slotBefore, summaryBefore)
		if err != nil {
			return err
		}
		if !occupied {
			return fmt.Errorf("character %d has no active or residual character data", characterID)
		}

		clearedSlot := make([]byte, characterSlotDataSize)
		clearedSummary := make([]byte, userData10SummaryStride)
		if err := loaded.snapshot.writeAt(slotAt, clearedSlot); err != nil {
			return fmt.Errorf("cannot clear data of character %d: %w", characterID, err)
		}
		if err := loaded.snapshot.writeAt(summaryAt, clearedSummary); err != nil {
			return restoreCharacterSlotRanges(
				loaded.snapshot, characterID, slotAt, slotBefore, summaryAt, summaryBefore,
				flagAt, flagBefore, fmt.Sprintf(
					"character %d was not deleted: the profile summary could not be cleared",
					characterID))
		}
		if err := loaded.snapshot.writeAt(flagAt, []byte{userData10InactiveFlagValue}); err != nil {
			return restoreCharacterSlotRanges(
				loaded.snapshot, characterID, slotAt, slotBefore, summaryAt, summaryBefore,
				flagAt, flagBefore, fmt.Sprintf(
					"character %d was not deleted: the activity flag could not be cleared",
					characterID))
		}

		slotWritten, slotErr := loaded.snapshot.readAt(slotAt, characterSlotDataSize)
		summaryWritten, summaryErr := loaded.snapshot.readAt(summaryAt, userData10SummaryStride)
		flagWritten, flagErr := loaded.snapshot.readAt(flagAt, 1)
		if slotErr == nil && summaryErr == nil && flagErr == nil &&
			bytes.Equal(slotWritten, clearedSlot) &&
			bytes.Equal(summaryWritten, clearedSummary) &&
			flagWritten[0] == userData10InactiveFlagValue {
			return nil
		}
		return restoreCharacterSlotRanges(
			loaded.snapshot, characterID, slotAt, slotBefore, summaryAt, summaryBefore,
			flagAt, flagBefore, fmt.Sprintf(
				"character %d was not deleted: the deletion could not be verified", characterID))
	})
	if err != nil {
		return DeleteCharacterResult{}, err
	}

	return DeleteCharacterResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
	}, nil
}

func characterCanBeDeleted(
	loaded *loadedSave,
	characterID int,
	activity byte,
	slotData []byte,
	summary []byte,
) (bool, error) {
	switch activity {
	case userData10ActiveFlagValue:
		return true, nil
	case userData10InactiveFlagValue:
	default:
		return false, fmt.Errorf("character %d has unsupported activity flag 0x%02X",
			characterID, activity)
	}

	if decodeCharacterName(summary[:summaryNameSize]) != "" {
		return true, nil
	}
	if bytesAreZero(slotData) && bytesAreZero(summary) {
		return false, nil
	}

	anchor, err := findStatsAnchor(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return false, fmt.Errorf(
			"cannot establish whether inactive character %d contains residual data: %w",
			characterID, err)
	}
	playerName, err := loaded.snapshot.readAt(
		anchor+playerCharacterNameOffset, summaryNameSize)
	if err != nil {
		return false, fmt.Errorf("cannot read residual PlayerGameData name of character %d: %w",
			characterID, err)
	}
	return decodeCharacterName(playerName) != "", nil
}
