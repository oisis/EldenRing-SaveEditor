package saveengine

import (
	"encoding/binary"
	"fmt"
)

// ApplyFavoritePresetResult reports one committed Mirror Favorites application.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat.
type ApplyFavoritePresetResult struct {
	MutationReceipt
	CharacterID    int `json:"characterID"`
	FavoriteSlotID int `json:"favoriteSlotID"`
}

// ApplyFavoritePreset applies the appearance fields represented by Mirror Favorites
// from the specified preset slot to an active character under expectedRevision control.
//
// favoriteSlotID must be in 0..14 and point to an active, populated preset slot.
// characterID must be in 0..9 and point to an active character.
// The preset supplies model IDs, face shape, body proportions, skin & cosmetics,
// and inverted body type to gender (0 -> male/1, 1 -> female/0).
// VoiceType, opaque FaceData block (0x70..0xB0), and all other character fields remain untouched.
func (engine *Engine) ApplyFavoritePreset(
	saveSessionID string,
	characterID int,
	favoriteSlotID int,
	expectedRevision string,
) (ApplyFavoritePresetResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return ApplyFavoritePresetResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if favoriteSlotID < 0 || favoriteSlotID >= favoriteSlotCount {
		return ApplyFavoritePresetResult{}, fmt.Errorf(
			"favoriteSlotID %d is outside the range 0..%d", favoriteSlotID, favoriteSlotCount-1)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return ApplyFavoritePresetResult{}, fmt.Errorf(
			"characterID %d is outside the range 0..%d", characterID, characterSlotCount-1)
	}

	committed, err := engine.commitCharacterRevision(saveSessionID, kindApplyFavoritePreset, characterID, func(loaded *loadedSave) error {
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

		base := userData10Base(loaded.session.platform)
		slotAt := favoriteSlotOffset(base, favoriteSlotID)
		if !loaded.snapshot.covers(slotAt, favoriteSlotSize) {
			return fmt.Errorf("favorite preset slot %d lies outside UserData10 bounds", favoriteSlotID)
		}

		slotBuf, err := loaded.snapshot.readAt(slotAt, favoriteSlotSize)
		if err != nil {
			return fmt.Errorf("cannot read favorite preset slot %d: %w", favoriteSlotID, err)
		}

		if string(slotBuf[favoriteMagicOffset:favoriteMagicOffset+len(favoriteMagic)]) != favoriteMagic {
			return fmt.Errorf("favorite preset slot %d is not active", favoriteSlotID)
		}

		bodyType := slotBuf[0x09]
		if bodyType > 1 {
			return fmt.Errorf("favorite preset slot %d has invalid body type %d, want 0..1", favoriteSlotID, bodyType)
		}

		var gender uint8
		if bodyType == 0 {
			gender = 1
		} else {
			gender = 0
		}

		var modelIDs [8]uint32
		for i := 0; i < 8; i++ {
			modelIDs[i] = binary.LittleEndian.Uint32(slotBuf[0x24+i*4 : 0x28+i*4])
		}
		var faceShape [64]uint8
		copy(faceShape[:], slotBuf[0x44:0x84])
		var body [7]uint8
		copy(body[:], slotBuf[0xC4:0xCB])
		var skin [91]uint8
		copy(skin[:], slotBuf[0xCB:0x126])

		anchor, err := findAppearancePlayerAnchor(loaded.snapshot, loaded.session.platform, characterID)
		if err != nil {
			return err
		}
		voiceBefore, err := loaded.snapshot.readAt(anchor+appearanceVoiceTypeOffset, 1)
		if err != nil {
			return fmt.Errorf("cannot read voice type of character %d: %w", characterID, err)
		}

		appearance := CharacterAppearanceValues{
			Gender:    gender,
			VoiceType: voiceBefore[0],
			ModelIDs:  modelIDs,
			FaceShape: faceShape,
			Body:      body,
			Skin:      skin,
		}

		return writeCharacterAppearance(loaded, characterID, appearance)
	})
	if err != nil {
		return ApplyFavoritePresetResult{}, err
	}

	return ApplyFavoritePresetResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		FavoriteSlotID:  favoriteSlotID,
	}, nil
}
