package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// SetCharacterStartingClassResult reports one committed starting-class mutation.
// It returns the session ID, the new revision, the character ID, the applied
// startingClassID, the eight base attributes of that class, its base level and
// the SoulMemory the character keeps, which this mutation never changes.
type SetCharacterStartingClassResult struct {
	SaveSessionID   string              `json:"saveSessionID"`
	SaveRevision    string              `json:"saveRevision"`
	CharacterID     int                 `json:"characterID"`
	StartingClassID uint8               `json:"startingClassID"`
	Attributes      CharacterAttributes `json:"attributes"`
	Level           uint32              `json:"level"`
	SoulMemory      uint32              `json:"soulMemory"`
}

// SetCharacterStartingClass atomically assigns the starting class of one active
// character, synchronising both the PlayerGameData and ProfileSummary copies.
//
// This is a destructive build reset, not a class relabel. The eight attributes
// become exactly the base attributes of the target class and the level becomes
// exactly its ClassDocument.Level, both taken from the embedded GameCatalog.
// Points the player distributed are discarded, attributes are lowered as freely
// as they are raised, and the level is never derived from the attribute sum
// here: the class carries its own confirmed soulLv.
//
// Because the reset destroys the current build, it runs only when the caller
// sets confirmReset. A missing or false confirmReset is rejected before anything
// is read or written, so the save and the revision stay exactly as they were.
//
// A committed reset leaves the ordinary single undo point of the session under
// kindSetCharacterStartingClass, so UndoCharacterChanges restores the previous
// class, level, attributes, SoulMemory and held runes. That point is one level
// deep and not durable: the next mutation replaces it and WriteSave ends the
// possibility of undoing, exactly as for every other character mutation.
//
// TotalGetSoul (SoulMemory) and the held runes are preserved byte for byte. A
// class change earns and spends nothing, so neither the lifetime-rune floor nor
// the rune purse belongs to this mutation; SetCharacterStats remains the only
// path that raises TotalGetSoul.
//
// Nothing else changes: the name, gender, appearance, inventory, storage,
// equipment, spells and every unrelated byte are preserved.
func (engine *Engine) SetCharacterStartingClass(
	saveSessionID string,
	characterID int,
	startingClassID uint8,
	confirmReset bool,
	expectedRevision string,
) (SetCharacterStartingClassResult, error) {
	if !confirmReset {
		return SetCharacterStartingClassResult{}, errors.New(
			"confirmReset must be true: changing the starting class resets all eight attributes " +
				"and the level to the base values of the target class")
	}
	if !isCanonicalRevision(expectedRevision) {
		return SetCharacterStartingClassResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	definition, err := startingClass(startingClassID)
	if err != nil {
		return SetCharacterStartingClassResult{}, err
	}

	var (
		resultingAttributes CharacterAttributes
		resultingLevel      uint32
		resultingSoulMemory uint32
	)

	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetCharacterStartingClass, characterID, func(loaded *loadedSave) error {
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

		// The reset copies the class definition verbatim. Whatever the eight
		// stored attributes were, legal or not, they are replaced rather than
		// carried forward, so a build cannot survive the change in any form.
		values := definition.attributes
		level := definition.level

		// TotalGetSoul is read only to report it. It stays inside the cloned
		// block untouched, together with the held runes.
		sm := binary.LittleEndian.Uint32(blockBefore[statsBlockTotalGetSoulPosition:])

		blockAfter := bytes.Clone(blockBefore)
		for index, value := range values {
			binary.LittleEndian.PutUint32(blockAfter[index*4:], value)
		}
		binary.LittleEndian.PutUint32(blockAfter[statsBlockLevelPosition:], level)

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
		SaveSessionID:   saveSessionID,
		SaveRevision:    committed.SaveRevision,
		CharacterID:     characterID,
		StartingClassID: startingClassID,
		Attributes:      resultingAttributes,
		Level:           resultingLevel,
		SoulMemory:      resultingSoulMemory,
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
