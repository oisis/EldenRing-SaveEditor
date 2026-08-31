package character

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestSetCharacterGenderAppliesTheConfirmedDefaultAppearance(t *testing.T) {
	gameCatalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}
	tests := []struct {
		name     string
		gender   uint8
		presetID string
		modelIDs [8]uint32
	}{
		{
			name:     "Type B",
			gender:   0,
			presetID: "ciri-the-princess-of-cintra-from-witcher",
			modelIDs: [8]uint32{40, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:     "Type A",
			gender:   1,
			presetID: "geralt-of-rivia-the-witcher",
			modelIDs: [8]uint32{0, 101, 0, 1, 3, 0, 6, 2},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine := saveengine.New()
			loaded, err := engine.LoadSave(
				writeGetCharacterAppearanceFixture(t, GetCharacterAppearanceResult{}), "", "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := SetCharacterGender(
				engine, gameCatalog, loaded.SaveSessionID,
				getCharacterAppearanceSlot, test.gender, "0")
			if err != nil {
				t.Fatalf("SetCharacterGender: %v", err)
			}
			if result.SaveSessionID != loaded.SaveSessionID ||
				result.SaveRevision != "1" ||
				result.CharacterID != getCharacterAppearanceSlot ||
				result.PresetID != test.presetID ||
				result.Appearance.Gender != test.gender ||
				result.Appearance.ModelIDs != test.modelIDs {
				t.Fatalf("result = %+v, want committed %s default", result, test.name)
			}

			stored, err := engine.GetCharacterAppearance(
				loaded.SaveSessionID, getCharacterAppearanceSlot)
			if err != nil {
				t.Fatalf("GetCharacterAppearance: %v", err)
			}
			if stored.Gender != result.Appearance.Gender ||
				stored.VoiceType != result.Appearance.VoiceType ||
				stored.ModelIDs != result.Appearance.ModelIDs ||
				stored.FaceShape != result.Appearance.FaceShape ||
				stored.Body != result.Appearance.Body ||
				stored.Skin != result.Appearance.Skin {
				t.Fatalf("stored appearance = %+v, want %+v", stored, result.Appearance)
			}
		})
	}
}

func TestSetCharacterGenderRejectsInvalidInputWithoutMutation(t *testing.T) {
	gameCatalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterAppearanceFixture(t, GetCharacterAppearanceResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterGender(
		engine, gameCatalog, loaded.SaveSessionID,
		getCharacterAppearanceSlot, 2, "0")
	if err == nil || !strings.Contains(err.Error(), "gender 2 is outside the range 0..1") {
		t.Fatalf("invalid-gender error = %v", err)
	}
	if !reflect.DeepEqual(result, SetCharacterGenderResult{}) {
		t.Fatalf("result = %+v, want zero value", result)
	}
	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Fatalf("session = %+v, want unchanged dirty state", info)
	}

	if _, err := SetCharacterGender(
		nil, gameCatalog, "session", 0, 0, "0"); err == nil {
		t.Fatal("SetCharacterGender accepted a nil engine")
	}
	if _, err := SetCharacterGender(
		engine, nil, loaded.SaveSessionID, getCharacterAppearanceSlot, 0, "0"); err == nil {
		t.Fatal("SetCharacterGender accepted a nil catalog")
	}
}
