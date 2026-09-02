package character

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestSetCharacterActiveReturnsTheSaveEngineReceipt(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetCharacterActive(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, false, "0")
	if err != nil {
		t.Fatalf("SetCharacterActive: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, loaded.SaveSessionID,
		SetCharacterActiveEndpointID, "1")
	// The receipt is pinned from the result because operationID names one
	// execution and cannot be predicted; every other member is asserted above.
	want := SetCharacterActiveResult{
		MutationReceipt: result.MutationReceipt,
		Changed:         true,
		CharacterID:     getCharacterStatsSlot,
		Active:          false,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	profile, err := engine.GetCharacterProfile(
		loaded.SaveSessionID, getCharacterStatsSlot)
	if err != nil {
		t.Fatalf("GetCharacterProfile: %v", err)
	}
	if profile.Active {
		t.Errorf("profile = %+v, want inactive", profile)
	}
}

// The idempotent request is the endpoint's second success variant. It commits
// nothing, so the three execution members of the receipt are absent from the
// payload rather than present and empty.
func TestSetCharacterActiveIdempotentRequestCarriesNoExecution(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// The fixture slot is already active, so requesting active again changes
	// nothing at all.
	result, err := SetCharacterActive(
		engine, loaded.SaveSessionID, getCharacterStatsSlot, true, "0")
	if err != nil {
		t.Fatalf("SetCharacterActive: %v", err)
	}
	if result.Changed {
		t.Fatalf("result = %+v, want changed=false", result)
	}
	want := SetCharacterActiveResult{
		MutationReceipt: saveengine.MutationReceipt{
			SaveSessionID: loaded.SaveSessionID,
			SaveRevision:  "0",
		},
		Changed:     false,
		CharacterID: getCharacterStatsSlot,
		Active:      true,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("decode %s: %v", encoded, err)
	}
	for _, absent := range []string{"operationID", "operationKind", "changedScopes", "receipt"} {
		if _, present := payload[absent]; present {
			t.Errorf("payload carries %q, want it absent: %s", absent, encoded)
		}
	}
	if len(payload) != 5 {
		t.Errorf("payload = %s, want exactly changed, saveSessionID, saveRevision, characterID and active",
			encoded)
	}

	state, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if state.SaveRevision != "0" || state.UnsavedChanges || state.EventSequence != "0" {
		t.Errorf("session = %+v, want an untouched session", state)
	}
}

func TestSetCharacterActiveRejectsMissingEngine(t *testing.T) {
	result, err := SetCharacterActive(nil, "any-session", 0, false, "0")
	if err == nil || err.Error() != "save engine is not available" {
		t.Fatalf("error = %v, want missing-engine error", err)
	}
	if !reflect.DeepEqual(result, SetCharacterActiveResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
