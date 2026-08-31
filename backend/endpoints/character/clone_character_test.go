package character

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestCloneCharacterReturnsTheSaveEngineReceipt(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if _, err := engine.SetCharacterName(
		loaded.SaveSessionID, getCharacterStatsSlot, "Ranni", "0"); err != nil {
		t.Fatalf("SetCharacterName fixture setup: %v", err)
	}

	result, err := CloneCharacter(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, 5, "1")
	if err != nil {
		t.Fatalf("CloneCharacter: %v", err)
	}
	want := CloneCharacterResult{
		SaveSessionID:     loaded.SaveSessionID,
		SaveRevision:      "2",
		SourceCharacterID: getCharacterStatsSlot,
		TargetSlotID:      5,
		Name:              "Ranni 2",
	}
	if result != want {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestCloneCharacterRejectsMissingEngine(t *testing.T) {
	result, err := CloneCharacter(nil, "any-session", 0, 1, "0")
	if err == nil || err.Error() != "save engine is not available" {
		t.Fatalf("error = %v, want missing-engine error", err)
	}
	if !reflect.DeepEqual(result, CloneCharacterResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
