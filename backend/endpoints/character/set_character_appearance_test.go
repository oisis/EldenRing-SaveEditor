package character

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func setCharacterAppearanceEndpointValues() CharacterAppearanceValues {
	values := CharacterAppearanceValues{
		Gender:    1,
		VoiceType: 5,
		ModelIDs:  [8]uint32{0, 0x121, 4, 3, 2, 1, 0, 5},
	}
	for index := range values.FaceShape {
		values.FaceShape[index] = uint8(index) + 1
	}
	for index := range values.Body {
		values.Body[index] = uint8(index) + 0x41
	}
	for index := range values.Skin {
		values.Skin[index] = uint8(index) + 0x81
	}
	return values
}

func TestSetCharacterAppearanceReturnsTheSaveEngineReceipt(t *testing.T) {
	values := setCharacterAppearanceEndpointValues()
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterAppearanceFixture(t, GetCharacterAppearanceResult{}), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterAppearance(
		engine, loaded.SaveSessionID, getCharacterAppearanceSlot, values, "0")
	if err != nil {
		t.Fatalf("SetCharacterAppearance: %v", err)
	}
	want := SetCharacterAppearanceResult{
		SaveSessionID: loaded.SaveSessionID,
		SaveRevision:  "1",
		CharacterID:   getCharacterAppearanceSlot,
		Appearance:    values,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestSetCharacterAppearanceRejectsMissingEngineAndInvalidValues(t *testing.T) {
	values := setCharacterAppearanceEndpointValues()
	result, err := SetCharacterAppearance(nil, "any-session", 0, values, "0")
	if err == nil || err.Error() != "save engine is not available" {
		t.Fatalf("missing-engine error = %v", err)
	}
	if !reflect.DeepEqual(result, SetCharacterAppearanceResult{}) {
		t.Errorf("missing-engine result = %+v, want zero", result)
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterAppearanceFixture(t, GetCharacterAppearanceResult{}), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	values.VoiceType = 6
	result, err = SetCharacterAppearance(
		engine, loaded.SaveSessionID, getCharacterAppearanceSlot, values, "0")
	if err == nil || err.Error() != "appearance.voiceType 6 is outside the range 0..5" {
		t.Fatalf("validation error = %v", err)
	}
	if !reflect.DeepEqual(result, SetCharacterAppearanceResult{}) {
		t.Errorf("validation result = %+v, want zero", result)
	}
	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Error("rejected endpoint request marked the session dirty")
	}
}
