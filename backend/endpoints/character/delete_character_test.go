package character

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestDeleteCharacterReturnsTheSaveEngineReceipt(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := DeleteCharacter(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, "0")
	if err != nil {
		t.Fatalf("DeleteCharacter: %v", err)
	}
	want := DeleteCharacterResult{
		SaveSessionID: loaded.SaveSessionID,
		SaveRevision:  "1",
		CharacterID:   getCharacterStatsSlot,
	}
	if result != want {
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
