package character

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestDeleteCharacterReturnsTheSaveEngineReceipt(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := DeleteCharacter(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, "0")
	if err != nil {
		t.Fatalf("DeleteCharacter: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, loaded.SaveSessionID,
		DeleteCharacterEndpointID, "1")
	// The receipt is pinned from the result because operationID names one
	// execution and cannot be predicted; every other member is asserted above.
	want := DeleteCharacterResult{
		MutationReceipt: result.MutationReceipt,
		CharacterID:     getCharacterStatsSlot,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestDeleteCharacterRejectsMissingEngine(t *testing.T) {
	result, err := DeleteCharacter(nil, "any-session", 0, "0")
	if err == nil || err.Error() != "save engine is not available" {
		t.Fatalf("error = %v, want missing-engine error", err)
	}
	if !reflect.DeepEqual(result, DeleteCharacterResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
