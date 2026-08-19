package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// SetCharacterStartingClassResult reports one committed starting-class mutation.
// It returns the session ID, the new revision, the character ID, the applied
// startingClassID, the eight resulting attributes, the resulting level, the
// resulting SoulMemory, and a boolean indicating whether any attribute was
// raised to meet the new class's minima.
type SetCharacterStartingClassResult struct {
	SaveSessionID    string              `json:"saveSessionID"`
	SaveRevision     string              `json:"saveRevision"`
	CharacterID      int                 `json:"characterID"`
	StartingClassID  uint8               `json:"startingClassID"`
	Attributes       CharacterAttributes `json:"attributes"`
	Level            uint32              `json:"level"`
	SoulMemory       uint32              `json:"soulMemory"`
	AttributesRaised bool                `json:"attributesRaised"`
}

// SetCharacterStartingClass atomically assigns the starting class of one active
// character, synchronising both the PlayerGameData and ProfileSummary copies.
//
// If any of the character's current attributes are below the base values of the
// new starting class, they are raised to meet the new class's minima; attributes
// at or above the minima are never lowered. The level is recalculated from the
// resulting attributes (level = sum - 79) and written to both PlayerGameData
// and ProfileSummary. If the recalculated level requires a higher TotalGetSoul
// (SoulMemory) than currently stored, TotalGetSoul is raised to that level's
// minimum and never lowered.
func (engine *Engine) SetCharacterStartingClass(
	saveSessionID string,
	characterID int,
	startingClassID uint8,
	expectedRevision string,
) (SetCharacterStartingClassResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetCharacterStartingClassResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	newMinima, err := startingClassMinima(startingClassID)
	if err != nil {
		return SetCharacterStartingClassResult{}, err
	}

	var (
		resultingAttributes CharacterAttributes
		resultingLevel      uint32
		resultingSoulMemory uint32
		attributesRaised    bool
	)

	saveRevision, err := engine.commitCharacterRevision(saveSessionID, opSetCharacterStartingClass, characterID, func(loaded *loadedSave) error {
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return fmt.Errorf(
				"expectedRevision %q does not match the current saveRevision %q",
				expectedRevision, current)
		}

		if characterID < 0 || characterID >= characterSlotCount {
			return fmt.Errorf("characterID %d is outside the range 0..%d",
				characterID, characterSlotCount-1)
		}

		base := userData10Base(loaded.session.platform)
		flag, err := loaded.snapshot.readAt(base+userData10ActiveFlagsOffset+int64(characterID), 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
		}
		if flag[0] != userData10ActiveFlagValue {
			return fmt.Errorf("character %d is not active", characterID)
		}

		anchor, err := findStatsAnchor(loaded.snapshot, loaded.session.platform, characterID)
		if err != nil {
			return err
		}

		summary := base + userData10SummaryOffset + int64(characterID)*userData10SummaryStride
		blockAt := anchor + statsWritableBlockOffset
		classAt := anchor + statsClassOffset
		summaryLevelAt := summary + summaryLevelOffset
		summaryClassAt := summary + summaryStartingClassOffset

		blockBefore, err := loaded.snapshot.readAt(blockAt, statsWritableBlockSize)
		if err != nil {
			return fmt.Errorf("cannot read statistics of character %d: %w", characterID, err)
		}
		classBefore, err := loaded.snapshot.readAt(classAt, 1)
		if err != nil {
			return fmt.Errorf("cannot read PlayerGameData class of character %d: %w", characterID, err)
		}
		summaryLevelBefore, err := loaded.snapshot.readAt(summaryLevelAt, summaryLevelSize)
		if err != nil {
			return fmt.Errorf("cannot read profile summary level of character %d: %w", characterID, err)
		}
		summaryClassBefore, err := loaded.snapshot.readAt(summaryClassAt, summaryIdentifierSize)
		if err != nil {
			return fmt.Errorf("cannot read profile summary class of character %d: %w", characterID, err)
		}

		var values [characterAttributeCount]uint32
		raised := false
		for index := 0; index < characterAttributeCount; index++ {
			val := binary.LittleEndian.Uint32(blockBefore[index*4:])
			if val < newMinima[index] {
				val = newMinima[index]
				raised = true
			}
			values[index] = val
		}

		level, err := recalculateCharacterLevel(values)
		if err != nil {
			return err
		}

		requiredSoulMemory := minimumSoulMemoryForLevel(level)
		storedSoulMemory := binary.LittleEndian.Uint32(blockBefore[statsBlockTotalGetSoulPosition:])
		sm := storedSoulMemory
		if sm < requiredSoulMemory {
			sm = requiredSoulMemory
		}

		blockAfter := bytes.Clone(blockBefore)
		for index, value := range values {
			binary.LittleEndian.PutUint32(blockAfter[index*4:], value)
		}
		binary.LittleEndian.PutUint32(blockAfter[statsBlockLevelPosition:], level)
		binary.LittleEndian.PutUint32(blockAfter[statsBlockTotalGetSoulPosition:], sm)

		classAfter := []byte{startingClassID}

		summaryLevelAfter := make([]byte, summaryLevelSize)
		binary.LittleEndian.PutUint32(summaryLevelAfter, level)

		summaryClassAfter := []byte{startingClassID}

		if bytes.Equal(blockBefore, blockAfter) &&
			bytes.Equal(classBefore, classAfter) &&
			bytes.Equal(summaryLevelBefore, summaryLevelAfter) &&
			bytes.Equal(summaryClassBefore, summaryClassAfter) {
			resultingAttributes = CharacterAttributes{
				Vigor: values[0], Mind: values[1], Endurance: values[2], Strength: values[3],
				Dexterity: values[4], Intelligence: values[5], Faith: values[6], Arcane: values[7],
			}
			resultingLevel = level
			resultingSoulMemory = sm
			attributesRaised = raised
			return nil
		}

		if err := loaded.snapshot.writeAt(blockAt, blockAfter); err != nil {
			return fmt.Errorf("cannot write statistics of character %d: %w", characterID, err)
		}
		if err := loaded.snapshot.writeAt(classAt, classAfter); err != nil {
			return restoreCharacterStartingClass(loaded.snapshot, characterID,
				blockAt, blockBefore, classAt, classBefore,
				summaryLevelAt, summaryLevelBefore, summaryClassAt, summaryClassBefore,
				fmt.Sprintf("cannot write PlayerGameData class of character %d: %v", characterID, err))
		}
		if err := loaded.snapshot.writeAt(summaryLevelAt, summaryLevelAfter); err != nil {
			return restoreCharacterStartingClass(loaded.snapshot, characterID,
				blockAt, blockBefore, classAt, classBefore,
				summaryLevelAt, summaryLevelBefore, summaryClassAt, summaryClassBefore,
				fmt.Sprintf("cannot write profile summary level of character %d: %v", characterID, err))
		}
		if err := loaded.snapshot.writeAt(summaryClassAt, summaryClassAfter); err != nil {
			return restoreCharacterStartingClass(loaded.snapshot, characterID,
				blockAt, blockBefore, classAt, classBefore,
				summaryLevelAt, summaryLevelBefore, summaryClassAt, summaryClassBefore,
				fmt.Sprintf("cannot write profile summary class of character %d: %v", characterID, err))
		}

		blockWritten, blockErr := loaded.snapshot.readAt(blockAt, statsWritableBlockSize)
		classWritten, classErr := loaded.snapshot.readAt(classAt, 1)
		summaryLevelWritten, summaryLevelErr := loaded.snapshot.readAt(summaryLevelAt, summaryLevelSize)
		summaryClassWritten, summaryClassErr := loaded.snapshot.readAt(summaryClassAt, summaryIdentifierSize)

		if blockErr == nil && classErr == nil && summaryLevelErr == nil && summaryClassErr == nil &&
			bytes.Equal(blockWritten, blockAfter) &&
			bytes.Equal(classWritten, classAfter) &&
			bytes.Equal(summaryLevelWritten, summaryLevelAfter) &&
			bytes.Equal(summaryClassWritten, summaryClassAfter) {
			resultingAttributes = CharacterAttributes{
				Vigor: values[0], Mind: values[1], Endurance: values[2], Strength: values[3],
				Dexterity: values[4], Intelligence: values[5], Faith: values[6], Arcane: values[7],
			}
			resultingLevel = level
			resultingSoulMemory = sm
			attributesRaised = raised
			return nil
		}

		return restoreCharacterStartingClass(loaded.snapshot, characterID,
			blockAt, blockBefore, classAt, classBefore,
			summaryLevelAt, summaryLevelBefore, summaryClassAt, summaryClassBefore,
			fmt.Sprintf("starting class mutation of character %d could not be verified", characterID))
	})
	if err != nil {
		return SetCharacterStartingClassResult{}, err
	}

	return SetCharacterStartingClassResult{
		SaveSessionID:    saveSessionID,
		SaveRevision:     saveRevision,
		CharacterID:      characterID,
		StartingClassID:  startingClassID,
		Attributes:       resultingAttributes,
		Level:            resultingLevel,
		SoulMemory:       resultingSoulMemory,
		AttributesRaised: attributesRaised,
	}, nil
}

// restoreCharacterStartingClass puts all four mutated ranges back and reports the
// failure that caused the rollback. A rollback that cannot be written or verified
// is reported instead, so a partially mutated snapshot is never presented as
// unchanged.
func restoreCharacterStartingClass(
	snapshot *codec,
	characterID int,
	blockAt int64,
	blockBefore []byte,
	classAt int64,
	classBefore []byte,
	summaryLevelAt int64,
	summaryLevelBefore []byte,
	summaryClassAt int64,
	summaryClassBefore []byte,
	failure string,
) error {
	if err := errors.Join(
		snapshot.writeAt(blockAt, blockBefore),
		snapshot.writeAt(classAt, classBefore),
		snapshot.writeAt(summaryLevelAt, summaryLevelBefore),
		snapshot.writeAt(summaryClassAt, summaryClassBefore),
	); err != nil {
		return fmt.Errorf("%s and the prior character data could not be restored: %w", failure, err)
	}

	blockRestored, blockErr := snapshot.readAt(blockAt, len(blockBefore))
	classRestored, classErr := snapshot.readAt(classAt, len(classBefore))
	summaryLevelRestored, summaryLevelErr := snapshot.readAt(summaryLevelAt, len(summaryLevelBefore))
	summaryClassRestored, summaryClassErr := snapshot.readAt(summaryClassAt, len(summaryClassBefore))
	if blockErr != nil || classErr != nil || summaryLevelErr != nil || summaryClassErr != nil ||
		!bytes.Equal(blockRestored, blockBefore) ||
		!bytes.Equal(classRestored, classBefore) ||
		!bytes.Equal(summaryLevelRestored, summaryLevelBefore) ||
		!bytes.Equal(summaryClassRestored, summaryClassBefore) {
		return fmt.Errorf("%s and the rollback of character %d could not be verified",
			failure, characterID)
	}
	return fmt.Errorf("%s; the save is unchanged", failure)
}
