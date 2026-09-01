package character

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// setCharacterStatsAttributes is a legal Vagabond assignment: the synthetic
// fixture stores starting class 0, and the sum recalculates to level 44.
var setCharacterStatsAttributes = CharacterAttributes{
	Vigor: 20, Mind: 15, Endurance: 16, Strength: 20,
	Dexterity: 18, Intelligence: 12, Faith: 12, Arcane: 10,
}

func TestSetCharacterStatsReturnsTheSaveEngineReceipt(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterStats(
		engine, loaded.SaveSessionID, getCharacterStatsSlot,
		setCharacterStatsAttributes, "recalculate", "0")
	if err != nil {
		t.Fatalf("SetCharacterStats: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, loaded.SaveSessionID,
		SetCharacterStatsEndpointID, "1")
	// The receipt is pinned from the result because operationID names one
	// execution and cannot be predicted; every other member is asserted above.
	want := SetCharacterStatsResult{
		MutationReceipt: result.MutationReceipt,
		CharacterID:     getCharacterStatsSlot,
		Attributes:      setCharacterStatsAttributes,
		Level:           44,
		// The fixture is a Vagabond, whose base level is 9. The absolute total
		// for level 44 is 177486; the 473 runes of the class base level are not
		// owed, so the floor SaveEngine writes is 177013.
		SoulMemory: 177_486 - 473,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestSetCharacterStatsRejectsMissingEngine(t *testing.T) {
	result, err := SetCharacterStats(
		nil, "any-session", 0, setCharacterStatsAttributes, "recalculate", "0")
	if err == nil || err.Error() != "save engine is not available" {
		t.Fatalf("error = %v, want missing-engine error", err)
	}
	if !reflect.DeepEqual(result, SetCharacterStatsResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}

func TestSetCharacterStatsDelegatesValidationWithoutMutation(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterStats(
		engine, loaded.SaveSessionID, getCharacterStatsSlot,
		setCharacterStatsAttributes, "preserve", "0")
	if err == nil || err.Error() != `levelPolicy must be "recalculate"; got "preserve"` {
		t.Fatalf("error = %v, want level-policy error", err)
	}
	if !reflect.DeepEqual(result, SetCharacterStatsResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}

	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("session after rejection = %+v, want clean", info)
	}
	accepted, err := SetCharacterStats(
		engine, loaded.SaveSessionID, getCharacterStatsSlot,
		setCharacterStatsAttributes, "recalculate", "0")
	if err != nil {
		t.Fatalf("valid request after rejection: %v", err)
	}
	if accepted.SaveRevision != "1" {
		t.Errorf("saveRevision after rejected request = %q, want first commit revision 1",
			accepted.SaveRevision)
	}
}
