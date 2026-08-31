package character

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestSetCharacterActiveReturnsTheSaveEngineReceipt(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterActive(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, false, "0")
	if err != nil {
		t.Fatalf("SetCharacterActive: %v", err)
	}
	want := SetCharacterActiveResult{
		SaveSessionID: loaded.SaveSessionID,
		SaveRevision:  "1",
		CharacterID:   getCharacterStatsSlot,
		Active:        false,
	}
	if result != want {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	profile, err := engine.GetCharacterProfile(
		loaded.SaveSessionID, getCharacterStatsSlot)
	if err != nil {
		t.Fatalf("GetCharacterProfile: %v", err)
	}
	if profile.Active {
		t.Errorf("profile = %+v, want inactive", profile)
	}
}

func TestSetCharacterActiveRejectsMissingEngine(t *testing.T) {
	result, err := SetCharacterActive(nil, "any-session", 0, false, "0")
	if err == nil || err.Error() != "save engine is not available" {
		t.Fatalf("error = %v, want missing-engine error", err)
	}
	if !reflect.DeepEqual(result, SetCharacterActiveResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
