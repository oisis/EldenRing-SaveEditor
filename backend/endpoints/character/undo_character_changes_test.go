package character

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestUndoCharacterChangesReturnsTheSaveEngineReceipt(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	before, err := GetCharacterStats(engine, loaded.SaveSessionID, getCharacterStatsSlot)
	if err != nil {
		t.Fatalf("GetCharacterStats: %v", err)
	}
	if _, err := SetCharacterStats(
		engine, loaded.SaveSessionID, getCharacterStatsSlot,
		setCharacterStatsAttributes, "recalculate", "0"); err != nil {
		t.Fatalf("SetCharacterStats: %v", err)
	}
	state, err := GetUndoState(engine, loaded.SaveSessionID, getCharacterStatsSlot)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}

	result, err := UndoCharacterChanges(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, state.UndoToken, "1")
	if err != nil {
		t.Fatalf("UndoCharacterChanges: %v", err)
	}
	want := UndoCharacterChangesResult{
		SaveSessionID:       loaded.SaveSessionID,
		SaveRevision:        "2",
		CharacterID:         getCharacterStatsSlot,
		UndoneOperationKind: SetCharacterStatsEndpointID,
	}
	if result != want {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	after, err := GetCharacterStats(engine, loaded.SaveSessionID, getCharacterStatsSlot)
	if err != nil {
		t.Fatalf("GetCharacterStats after undo: %v", err)
	}
	if after.SaveRevision != result.SaveRevision {
		t.Fatalf("statistics revision = %q, want %q", after.SaveRevision, result.SaveRevision)
	}
	before.SaveRevision = after.SaveRevision
	if !reflect.DeepEqual(after, before) {
		t.Errorf("statistics after undo = %+v, want the pre-mutation %+v", after, before)
	}
}

func TestUndoCharacterChangesRejectsMissingEngine(t *testing.T) {
	result, err := UndoCharacterChanges(nil, "any-session", 0, "any-token", "0")
	if err == nil || err.Error() != "save engine is not available" {
		t.Fatalf("error = %v, want missing-engine error", err)
	}
	if !reflect.DeepEqual(result, UndoCharacterChangesResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
