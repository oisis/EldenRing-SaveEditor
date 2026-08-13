package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	playerRunesOffset   = int64(-331)
	characterRunesLimit = uint32(999_999_999)
)

// SetCharacterRunesResult reports one committed held-runes assignment.
type SetCharacterRunesResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	Runes         uint32 `json:"runes"`
}

// SetCharacterRunes assigns the held runes of one active character. Held runes
// occupy one confirmed little-endian uint32 in PlayerGameData. The adjacent
// TotalGetSoul field (called SoulMemory by legacy code) and every other field
// remain untouched.
func (engine *Engine) SetCharacterRunes(
	saveSessionID string,
	characterID int,
	runes uint32,
	expectedRevision string,
) (SetCharacterRunesResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetCharacterRunesResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if runes > characterRunesLimit {
		return SetCharacterRunesResult{}, fmt.Errorf(
			"runes %d exceeds the maximum %d", runes, characterRunesLimit)
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

		anchor, err := findStatsAnchor(loaded.snapshot, loaded.session.platform, characterID)
		if err != nil {
			return err
		}
		fieldAt := anchor + playerRunesOffset
		before, err := loaded.snapshot.readAt(fieldAt, 4)
		if err != nil {
			return fmt.Errorf("cannot read runes of character %d: %w", characterID, err)
		}

		encoded := make([]byte, 4)
		binary.LittleEndian.PutUint32(encoded, runes)
		if binary.LittleEndian.Uint32(before) == runes {
			return nil
		}
		if err := loaded.snapshot.writeAt(fieldAt, encoded); err != nil {
			return fmt.Errorf("cannot write runes of character %d: %w", characterID, err)
		}

		written, err := loaded.snapshot.readAt(fieldAt, 4)
		if err == nil && binary.LittleEndian.Uint32(written) == runes {
			return nil
		}
		if rollback := loaded.snapshot.writeAt(fieldAt, before); rollback != nil {
			return fmt.Errorf(
				"runes of character %d could not be verified and could not be restored: %w",
				characterID, rollback)
		}
		return errors.New("runes mutation could not be verified; the save is unchanged")
	})
	if err != nil {
		return SetCharacterRunesResult{}, err
	}

	return SetCharacterRunesResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  saveRevision,
		CharacterID:   characterID,
		Runes:         runes,
	}, nil
}
