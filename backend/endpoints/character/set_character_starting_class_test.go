package character

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestSetCharacterStartingClassReturnsTheSaveEngineReceipt(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterStartingClass(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, 4, "0")
	if err != nil {
		t.Fatalf("SetCharacterStartingClass: %v", err)
	}

	if result.SaveSessionID != loaded.SaveSessionID || result.SaveRevision != "1" ||
		result.CharacterID != getCharacterStatsSlot || result.StartingClassID != 4 {
		t.Fatalf("result = %+v, unexpected receipt values", result)
	}
}

func TestSetCharacterStartingClassRejectsMissingEngine(t *testing.T) {
	result, err := SetCharacterStartingClass(
		nil, "any-session", 0, 1, "0")
	if err == nil || err.Error() != "save engine is not available" {
		t.Fatalf("error = %v, want missing-engine error", err)
	}
	if !reflect.DeepEqual(result, SetCharacterStartingClassResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}

func TestSetCharacterStartingClassDelegatesValidationWithoutMutation(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterStartingClass(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, 99, "0")
	if err == nil || err.Error() != "starting class 99 is unknown; its attribute minima are not confirmed" {
		t.Fatalf("error = %v, want unknown starting class error", err)
	}
	if !reflect.DeepEqual(result, SetCharacterStartingClassResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}

	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("session after rejection = %+v, want clean", info)
	}
	accepted, err := SetCharacterStartingClass(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, 0, "0")
	if err != nil {
		t.Fatalf("valid request after rejection: %v", err)
	}
	if accepted.SaveRevision != "1" {
		t.Errorf("saveRevision after rejected request = %q, want first commit revision 1",
			accepted.SaveRevision)
	}
}
