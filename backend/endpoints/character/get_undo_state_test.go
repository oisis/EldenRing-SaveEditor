package character

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestGetUndoStateReturnsTheSaveEngineState(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	fresh, err := GetUndoState(engine, loaded.SaveSessionID, getCharacterStatsSlot)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	want := CharacterUndoState{
		SaveSessionID: loaded.SaveSessionID,
		SaveRevision:  "0",
		CharacterID:   getCharacterStatsSlot,
	}
	if fresh != want {
		t.Fatalf("undo state of a fresh session = %+v, want %+v", fresh, want)
	}

	if _, err := SetCharacterStats(
		engine, loaded.SaveSessionID, getCharacterStatsSlot,
		setCharacterStatsAttributes, "recalculate", "0"); err != nil {
		t.Fatalf("SetCharacterStats: %v", err)
	}

	state, err := GetUndoState(engine, loaded.SaveSessionID, getCharacterStatsSlot)
	if err != nil {
		t.Fatalf("GetUndoState after the mutation: %v", err)
	}
	if !state.Available || state.UndoToken == "" ||
		state.OperationID != SetCharacterStatsEndpointID || state.SaveRevision != "1" {
		t.Errorf("undo state = %+v, want an available set_character_stats point at revision 1", state)
	}
}

func TestGetUndoStateRejectsMissingEngine(t *testing.T) {
	result, err := GetUndoState(nil, "any-session", 0)
	if err == nil || err.Error() != "save engine is not available" {
		t.Fatalf("error = %v, want missing-engine error", err)
	}
	if !reflect.DeepEqual(result, CharacterUndoState{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
