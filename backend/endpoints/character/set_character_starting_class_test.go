package character

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestSetCharacterStartingClassReturnsTheSaveEngineReceipt(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterStartingClass(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, 4, true, "0")
	if err != nil {
		t.Fatalf("SetCharacterStartingClass: %v", err)
	}

	if result.SaveSessionID != loaded.SaveSessionID || result.SaveRevision != "1" ||
		result.CharacterID != getCharacterStatsSlot || result.StartingClassID != 4 {
		t.Fatalf("result = %+v, unexpected receipt values", result)
	}
	// The receipt reports the Astrologer base build straight from the class
	// document, at the class base level 6.
	want := saveengine.CharacterAttributes{
		Vigor: 9, Mind: 15, Endurance: 9, Strength: 8,
		Dexterity: 12, Intelligence: 16, Faith: 7, Arcane: 9,
	}
	if result.Attributes != want || result.Level != 6 {
		t.Fatalf("result = %+v, want the Astrologer base %+v at level 6", result, want)
	}
}

// The destructive reset is refused unless the caller confirms it, and the
// refusal reaches this layer unchanged: no receipt, no mutation, no revision.
func TestSetCharacterStartingClassRequiresConfirmReset(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterStartingClass(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, 4, false, "0")
	if err == nil || !strings.Contains(err.Error(), "confirmReset must be true") {
		t.Fatalf("error = %v, want the confirmReset rejection", err)
	}
	if !reflect.DeepEqual(result, SetCharacterStartingClassResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}

	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("session after the unconfirmed request = %+v, want untouched", info)
	}

	confirmed, err := SetCharacterStartingClass(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, 4, true, "0")
	if err != nil {
		t.Fatalf("confirmed request: %v", err)
	}
	if confirmed.SaveRevision != "1" {
		t.Errorf("saveRevision = %q, want the first commit revision 1", confirmed.SaveRevision)
	}
}

func TestSetCharacterStartingClassRejectsMissingEngine(t *testing.T) {
	result, err := SetCharacterStartingClass(
		nil, "any-session", 0, 1, true, "0")
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
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterStartingClass(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, 99, true, "0")
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
		engine, loaded.SaveSessionID, getCharacterStatsSlot, 0, true, "0")
	if err != nil {
		t.Fatalf("valid request after rejection: %v", err)
	}
	if accepted.SaveRevision != "1" {
		t.Errorf("saveRevision after rejected request = %q, want first commit revision 1",
			accepted.SaveRevision)
	}
}
