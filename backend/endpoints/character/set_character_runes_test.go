package character

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestSetCharacterRunesReturnsTheSaveEngineReceipt(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterRunes(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, 999_999_999, "0")
	if err != nil {
		t.Fatalf("SetCharacterRunes: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, loaded.SaveSessionID,
		SetCharacterRunesEndpointID, "1")
	// The receipt is pinned from the result because operationID names one
	// execution and cannot be predicted; every other member is asserted above.
	want := SetCharacterRunesResult{
		MutationReceipt: result.MutationReceipt,
		CharacterID:     getCharacterStatsSlot,
		Runes:           999_999_999,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestSetCharacterRunesRejectsMissingEngine(t *testing.T) {
	result, err := SetCharacterRunes(nil, "any-session", 0, 0, "0")
	if err == nil || err.Error() != "save engine is not available" {
		t.Fatalf("error = %v, want missing-engine error", err)
	}
	if !reflect.DeepEqual(result, SetCharacterRunesResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}

func TestSetCharacterRunesDelegatesRangeValidationWithoutMutation(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterRunes(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, 1_000_000_000, "0")
	if err == nil || err.Error() != "runes 1000000000 exceeds the maximum 999999999" {
		t.Fatalf("error = %v, want legal-maximum error", err)
	}
	if !reflect.DeepEqual(result, SetCharacterRunesResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("session after rejection = %+v, want clean", info)
	}
	accepted, err := SetCharacterRunes(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, 0, "0")
	if err != nil {
		t.Fatalf("valid request after rejection: %v", err)
	}
	if accepted.SaveRevision != "1" {
		t.Errorf("saveRevision after rejected request = %q, want first commit revision 1",
			accepted.SaveRevision)
	}
}
