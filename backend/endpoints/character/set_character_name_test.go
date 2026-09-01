package character

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestSetCharacterNameReturnsTheSaveEngineReceipt(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterName(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, "Melina", "0")
	if err != nil {
		t.Fatalf("SetCharacterName: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, loaded.SaveSessionID,
		SetCharacterNameEndpointID, "1")
	// The receipt is pinned from the result because operationID names one
	// execution and cannot be predicted; every other member is asserted above.
	want := SetCharacterNameResult{
		MutationReceipt: result.MutationReceipt,
		CharacterID:     getCharacterStatsSlot,
		Name:            "Melina",
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	profile, err := engine.GetCharacterProfile(
		loaded.SaveSessionID, getCharacterStatsSlot)
	if err != nil {
		t.Fatalf("GetCharacterProfile: %v", err)
	}
	if profile.Name != "Melina" {
		t.Errorf("profile name = %q, want Melina", profile.Name)
	}
}

func TestSetCharacterNameRejectsMissingEngine(t *testing.T) {
	result, err := SetCharacterName(nil, "any-session", 0, "Melina", "0")
	if err == nil || err.Error() != "save engine is not available" {
		t.Fatalf("error = %v, want missing-engine error", err)
	}
	if !reflect.DeepEqual(result, SetCharacterNameResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}

func TestSetCharacterNameDelegatesValidationWithoutMutation(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterName(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, "", "0")
	if err == nil || err.Error() != "name is required" {
		t.Fatalf("error = %v, want name validation error", err)
	}
	if !reflect.DeepEqual(result, SetCharacterNameResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("session after rejection = %+v, want clean", info)
	}
	accepted, err := SetCharacterName(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, "Melina", "0")
	if err != nil {
		t.Fatalf("valid request after rejection: %v", err)
	}
	if accepted.SaveRevision != "1" {
		t.Errorf("saveRevision after rejected request = %q, want first commit revision 1",
			accepted.SaveRevision)
	}
}
