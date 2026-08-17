package saveengine

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// SetFavoritePresetResult reports one committed Mirror Favorites write.
type SetFavoritePresetResult struct {
	SaveSessionID     string `json:"saveSessionID"`
	SaveRevision      string `json:"saveRevision"`
	FavoriteSlotID    int    `json:"favoriteSlotID"`
	SourceCharacterID int    `json:"sourceCharacterID"`
}

// SetFavoritePreset saves all appearance fields represented by Mirror Favorites
// from an active character into the specified preset slot in UserData10 under
// expectedRevision control.
//
// favoriteSlotID must be in 0..14.
// sourceCharacterID must be in 0..9 and point to an active character.
// The whole 0x130-byte slot buffer is built from the character's gender, model IDs,
// face shape sliders, unknown/opaque block, body proportions, and skin & cosmetics
// (VoiceType is not represented in Mirror Favorites).
// If the slot already matches the target bytes identically, it is a byte-level no-op,
// but advances revision and marks the session dirty as standard for commitRevision.
func (engine *Engine) SetFavoritePreset(
	saveSessionID string,
	favoriteSlotID int,
	sourceCharacterID int,
	expectedRevision string,
) (SetFavoritePresetResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetFavoritePresetResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if favoriteSlotID < 0 || favoriteSlotID >= favoriteSlotCount {
		return SetFavoritePresetResult{}, fmt.Errorf(
			"favoriteSlotID %d is outside the range 0..%d", favoriteSlotID, favoriteSlotCount-1)
	}
	if sourceCharacterID < 0 || sourceCharacterID >= characterSlotCount {
		return SetFavoritePresetResult{}, fmt.Errorf(
			"sourceCharacterID %d is outside the range 0..%d", sourceCharacterID, characterSlotCount-1)
	}

	saveRevision, err := engine.commitRevision(saveSessionID, func(loaded *loadedSave) error {
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return fmt.Errorf(
				"expectedRevision %q does not match the current saveRevision %q",
				expectedRevision, current)
		}

		flag, err := loaded.snapshot.readAt(
			userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(sourceCharacterID), 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of character %d: %w", sourceCharacterID, err)
		}
		if flag[0] != userData10ActiveFlagValue {
			return fmt.Errorf("character %d is not active", sourceCharacterID)
		}

		anchor, err := findAppearancePlayerAnchor(
			loaded.snapshot, loaded.session.platform, sourceCharacterID)
		if err != nil {
			return err
		}
		gender, err := loaded.snapshot.readAt(anchor+appearanceGenderOffset, 1)
		if err != nil {
			return fmt.Errorf("cannot read gender of character %d: %w", sourceCharacterID, err)
		}
		if gender[0] > appearanceMaximumGender {
			return fmt.Errorf("gender of character %d is %d, want 0..%d",
				sourceCharacterID, gender[0], appearanceMaximumGender)
		}

		_, faceBlock, err := readFaceData(
			loaded.snapshot, loaded.session.platform, sourceCharacterID)
		if err != nil {
			return err
		}

		buf := make([]byte, favoriteSlotSize)

		// Slot header
		binary.LittleEndian.PutUint16(buf[0x00:], 0xFACE)
		binary.LittleEndian.PutUint32(buf[0x04:], 0x11D0)
		buf[0x08] = 1
		if gender[0] == 1 {
			buf[0x09] = 0 // male in Favorites
		} else {
			buf[0x09] = 1 // female in Favorites
		}

		// FACE block header
		copy(buf[favoriteMagicOffset:], favoriteMagic)
		binary.LittleEndian.PutUint32(buf[0x1C:], 4)
		binary.LittleEndian.PutUint32(buf[0x20:], 0x120)

		// Model IDs (32 bytes)
		copy(buf[0x24:0x44], faceBlock[faceDataModelIDsOffset:faceDataModelIDsOffset+faceDataModelIDCount*4])

		// Face shape sliders (64 bytes)
		copy(buf[0x44:0x84], faceBlock[faceDataFaceShapeOffset:faceDataFaceShapeOffset+faceDataFaceShapeSize])

		// Opaque / unknown block (64 bytes)
		copy(buf[0x84:0xC4], faceBlock[0x70:0xB0])

		// Body proportions (7 bytes)
		copy(buf[0xC4:0xCB], faceBlock[faceDataBodyOffset:faceDataBodyOffset+faceDataBodySize])

		// Skin & cosmetics (91 bytes)
		copy(buf[0xCB:0x126], faceBlock[faceDataSkinOffset:faceDataSkinOffset+faceDataSkinSize])

		base := userData10Base(loaded.session.platform)
		slotAt := favoriteSlotOffset(base, favoriteSlotID)
		existing, err := loaded.snapshot.readAt(slotAt, favoriteSlotSize)
		if err != nil {
			return fmt.Errorf("cannot read favorite preset slot %d: %w", favoriteSlotID, err)
		}
		if bytes.Equal(existing, buf) {
			return nil
		}

		if err := applyByteWrites(loaded.snapshot, []byteWrite{{
			at:   slotAt,
			data: buf,
		}}); err != nil {
			return fmt.Errorf("cannot write favorite preset slot %d: %w", favoriteSlotID, err)
		}
		return nil
	})
	if err != nil {
		return SetFavoritePresetResult{}, err
	}

	return SetFavoritePresetResult{
		SaveSessionID:     saveSessionID,
		SaveRevision:      saveRevision,
		FavoriteSlotID:    favoriteSlotID,
		SourceCharacterID: sourceCharacterID,
	}, nil
}
