package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	appearanceMaximumGender    = 1
	appearanceMaximumVoiceType = 5
	faceDataSexFlagsOffset     = 0x125
	faceDataSexFlagsSize       = 2
)

// CharacterAppearanceValues is the complete writable appearance model. The
// model IDs and parameter blocks are the raw values stored by the save.
type CharacterAppearanceValues struct {
	Gender    uint8     `json:"gender"`
	VoiceType uint8     `json:"voiceType"`
	ModelIDs  [8]uint32 `json:"modelIDs"`
	FaceShape [64]uint8 `json:"faceShape"`
	Body      [7]uint8  `json:"body"`
	Skin      [91]uint8 `json:"skin"`
}

// SetCharacterAppearanceResult reports one committed complete appearance
// assignment. The embedded receipt is the one the shared writer produced, so it
// names the public entry point that was called: SetCharacterAppearance,
// SetCharacterGender or ApplyCharacterAppearancePreset.
//
// The receipt is embedded anonymously, so saveSessionID and saveRevision keep
// their previous JSON names and the three new members join them flat.
type SetCharacterAppearanceResult struct {
	MutationReceipt
	CharacterID int                       `json:"characterID"`
	Appearance  CharacterAppearanceValues `json:"appearance"`
}

// SetCharacterAppearance atomically replaces the confirmed writable appearance
// fields of one active character. Unknown bytes in the FACE block remain
// unchanged; the two confirmed dependent sex-flag bytes are reset to zero.
func (engine *Engine) SetCharacterAppearance(
	saveSessionID string,
	characterID int,
	appearance CharacterAppearanceValues,
	expectedRevision string,
) (SetCharacterAppearanceResult, error) {
	return engine.setCharacterAppearance(
		saveSessionID, characterID, appearance, expectedRevision, kindSetCharacterAppearance)
}

// SetCharacterGenderAppearance is SetCharacterAppearance for the gender change,
// which replaces the whole appearance with the gender preset the caller
// resolved. It exists so the undo point reports the gender operation instead of
// the plain appearance one.
func (engine *Engine) SetCharacterGenderAppearance(
	saveSessionID string,
	characterID int,
	appearance CharacterAppearanceValues,
	expectedRevision string,
) (SetCharacterAppearanceResult, error) {
	return engine.setCharacterAppearance(
		saveSessionID, characterID, appearance, expectedRevision, kindSetCharacterGender)
}

// ApplyCharacterAppearancePreset is SetCharacterAppearance for an appearance
// preset the caller resolved. It exists so the undo point reports the preset
// operation instead of the plain appearance one.
func (engine *Engine) ApplyCharacterAppearancePreset(
	saveSessionID string,
	characterID int,
	appearance CharacterAppearanceValues,
	expectedRevision string,
) (SetCharacterAppearanceResult, error) {
	return engine.setCharacterAppearance(
		saveSessionID, characterID, appearance, expectedRevision, kindApplyAppearancePreset)
}

// setCharacterAppearance is the one writer behind the three public appearance
// entry points. operationKind is chosen by those entry points and never by a
// caller outside this package, so the undo point reports the operation the user
// actually performed.
func (engine *Engine) setCharacterAppearance(
	saveSessionID string,
	characterID int,
	appearance CharacterAppearanceValues,
	expectedRevision string,
	operationKind string,
) (SetCharacterAppearanceResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetCharacterAppearanceResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if appearance.Gender > appearanceMaximumGender {
		return SetCharacterAppearanceResult{}, fmt.Errorf(
			"appearance.gender %d is outside the range 0..%d",
			appearance.Gender, appearanceMaximumGender)
	}
	if appearance.VoiceType > appearanceMaximumVoiceType {
		return SetCharacterAppearanceResult{}, fmt.Errorf(
			"appearance.voiceType %d is outside the range 0..%d",
			appearance.VoiceType, appearanceMaximumVoiceType)
	}

	committed, err := engine.commitCharacterRevision(saveSessionID, operationKind, characterID, func(loaded *loadedSave) error {
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

		return writeCharacterAppearance(loaded, characterID, appearance)
	})
	if err != nil {
		return SetCharacterAppearanceResult{}, err
	}

	return SetCharacterAppearanceResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		Appearance:      appearance,
	}, nil
}

func writeCharacterAppearance(
	loaded *loadedSave,
	characterID int,
	appearance CharacterAppearanceValues,
) error {
	anchor, err := findAppearancePlayerAnchor(
		loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return err
	}
	faceAt, faceBefore, err := readFaceData(
		loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return err
	}
	genderAt := anchor + appearanceGenderOffset
	voiceAt := anchor + appearanceVoiceTypeOffset
	genderBefore, err := loaded.snapshot.readAt(genderAt, 1)
	if err != nil {
		return fmt.Errorf("cannot read gender of character %d: %w", characterID, err)
	}
	voiceBefore, err := loaded.snapshot.readAt(voiceAt, 1)
	if err != nil {
		return fmt.Errorf("cannot read voice type of character %d: %w", characterID, err)
	}

	faceAfter := bytes.Clone(faceBefore)
	for index, modelID := range appearance.ModelIDs {
		binary.LittleEndian.PutUint32(
			faceAfter[faceDataModelIDsOffset+index*4:], modelID)
	}
	copy(faceAfter[faceDataFaceShapeOffset:], appearance.FaceShape[:])
	copy(faceAfter[faceDataBodyOffset:], appearance.Body[:])
	copy(faceAfter[faceDataSkinOffset:], appearance.Skin[:])
	clear(faceAfter[faceDataSexFlagsOffset : faceDataSexFlagsOffset+faceDataSexFlagsSize])

	genderAfter := []byte{appearance.Gender}
	voiceAfter := []byte{appearance.VoiceType}
	if bytes.Equal(genderBefore, genderAfter) && bytes.Equal(voiceBefore, voiceAfter) &&
		bytes.Equal(faceBefore, faceAfter) {
		return nil
	}

	if err := loaded.snapshot.writeAt(genderAt, genderAfter); err != nil {
		return fmt.Errorf("cannot write gender of character %d: %w", characterID, err)
	}
	if err := loaded.snapshot.writeAt(voiceAt, voiceAfter); err != nil {
		if rollback := restoreCharacterAppearance(
			loaded.snapshot, genderAt, genderBefore, voiceAt, voiceBefore, faceAt, faceBefore); rollback != nil {
			return fmt.Errorf(
				"voice type of character %d could not be written and the prior appearance could not be restored: %w",
				characterID, rollback)
		}
		return fmt.Errorf("cannot write voice type of character %d: %w", characterID, err)
	}
	if err := loaded.snapshot.writeAt(faceAt, faceAfter); err != nil {
		if rollback := restoreCharacterAppearance(
			loaded.snapshot, genderAt, genderBefore, voiceAt, voiceBefore, faceAt, faceBefore); rollback != nil {
			return fmt.Errorf(
				"appearance block of character %d could not be written and the prior appearance could not be restored: %w",
				characterID, rollback)
		}
		return fmt.Errorf("cannot write appearance block of character %d: %w", characterID, err)
	}

	genderWritten, genderErr := loaded.snapshot.readAt(genderAt, 1)
	voiceWritten, voiceErr := loaded.snapshot.readAt(voiceAt, 1)
	faceWritten, faceErr := loaded.snapshot.readAt(faceAt, faceDataSize)
	if genderErr == nil && voiceErr == nil && faceErr == nil &&
		bytes.Equal(genderWritten, genderAfter) && bytes.Equal(voiceWritten, voiceAfter) &&
		bytes.Equal(faceWritten, faceAfter) {
		return nil
	}

	if rollback := restoreCharacterAppearance(
		loaded.snapshot, genderAt, genderBefore, voiceAt, voiceBefore, faceAt, faceBefore); rollback != nil {
		return fmt.Errorf(
			"appearance of character %d could not be verified and could not be restored: %w",
			characterID, rollback)
	}
	return errors.New("character appearance mutation could not be verified; the save is unchanged")
}

func restoreCharacterAppearance(
	snapshot *codec,
	genderAt int64,
	gender []byte,
	voiceAt int64,
	voice []byte,
	faceAt int64,
	face []byte,
) error {
	return errors.Join(
		snapshot.writeAt(genderAt, gender),
		snapshot.writeAt(voiceAt, voice),
		snapshot.writeAt(faceAt, face),
	)
}
