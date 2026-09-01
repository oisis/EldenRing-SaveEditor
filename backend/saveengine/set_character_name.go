package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

const playerCharacterNameOffset = int64(-0x11B)

// SetCharacterNameResult reports one committed character-name assignment.
type SetCharacterNameResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	Name          string `json:"name"`
}

// SetCharacterName assigns the name of one active character slot. The save
// carries the same 16-unit UTF-16LE field in PlayerGameData and UserData10's
// ProfileSummary; both copies are validated, written and verified as one
// mutation. The adjacent two-byte values and every unrelated byte remain
// untouched.
func (engine *Engine) SetCharacterName(
	saveSessionID string,
	characterID int,
	name string,
	expectedRevision string,
) (SetCharacterNameResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetCharacterNameResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	encoded, err := encodeCharacterName(name)
	if err != nil {
		return SetCharacterNameResult{}, err
	}

	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetCharacterName, characterID, func(loaded *loadedSave) error {
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

		anchor, err := findStatsAnchor(loaded.snapshot, loaded.session.platform, characterID)
		if err != nil {
			return err
		}
		playerAt := anchor + playerCharacterNameOffset
		summaryAt := userData10Base(loaded.session.platform) + userData10SummaryOffset +
			int64(characterID)*userData10SummaryStride + summaryNameOffset

		playerBefore, err := loaded.snapshot.readAt(playerAt, summaryNameSize)
		if err != nil {
			return fmt.Errorf("cannot read PlayerGameData name of character %d: %w", characterID, err)
		}
		summaryBefore, err := loaded.snapshot.readAt(summaryAt, summaryNameSize)
		if err != nil {
			return fmt.Errorf("cannot read profile summary name of character %d: %w", characterID, err)
		}
		if bytes.Equal(playerBefore, encoded) && bytes.Equal(summaryBefore, encoded) {
			return nil
		}

		if err := loaded.snapshot.writeAt(playerAt, encoded); err != nil {
			return fmt.Errorf("cannot write PlayerGameData name of character %d: %w", characterID, err)
		}
		if err := loaded.snapshot.writeAt(summaryAt, encoded); err != nil {
			if rollback := restoreCharacterNameFields(
				loaded.snapshot, playerAt, playerBefore, summaryAt, summaryBefore); rollback != nil {
				return fmt.Errorf(
					"profile summary name of character %d could not be written and the prior name could not be restored: %w",
					characterID, rollback)
			}
			return fmt.Errorf("cannot write profile summary name of character %d: %w", characterID, err)
		}

		playerWritten, playerErr := loaded.snapshot.readAt(playerAt, summaryNameSize)
		summaryWritten, summaryErr := loaded.snapshot.readAt(summaryAt, summaryNameSize)
		if playerErr == nil && summaryErr == nil &&
			bytes.Equal(playerWritten, encoded) && bytes.Equal(summaryWritten, encoded) {
			return nil
		}

		if rollback := restoreCharacterNameFields(
			loaded.snapshot, playerAt, playerBefore, summaryAt, summaryBefore); rollback != nil {
			return fmt.Errorf(
				"character name of character %d could not be verified and could not be restored: %w",
				characterID, rollback)
		}
		return errors.New("character name mutation could not be verified; the save is unchanged")
	})
	if err != nil {
		return SetCharacterNameResult{}, err
	}

	return SetCharacterNameResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  committed.SaveRevision,
		CharacterID:   characterID,
		Name:          name,
	}, nil
}

func encodeCharacterName(name string) ([]byte, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}
	if !utf8.ValidString(name) {
		return nil, errors.New("name must be valid UTF-8")
	}
	if strings.ContainsRune(name, '\x00') {
		return nil, errors.New("name must not contain NUL")
	}

	units := utf16.Encode([]rune(name))
	if len(units) > summaryNameUnits {
		return nil, fmt.Errorf(
			"name uses %d UTF-16 code units, maximum is %d", len(units), summaryNameUnits)
	}

	encoded := make([]byte, summaryNameSize)
	for index, unit := range units {
		binary.LittleEndian.PutUint16(encoded[index*2:], unit)
	}
	return encoded, nil
}

func restoreCharacterNameFields(
	snapshot *codec,
	playerAt int64,
	playerBefore []byte,
	summaryAt int64,
	summaryBefore []byte,
) error {
	return errors.Join(
		snapshot.writeAt(playerAt, playerBefore),
		snapshot.writeAt(summaryAt, summaryBefore),
	)
}
