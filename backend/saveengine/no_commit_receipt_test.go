package saveengine

import (
	"encoding/json"
	"testing"
)

func TestMutationReceiptAlwaysSerializesTheCompleteCommittedContract(t *testing.T) {
	encoded, err := json.Marshal(MutationReceipt{})
	if err != nil {
		t.Fatalf("marshal MutationReceipt: %v", err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("decode %s: %v", encoded, err)
	}
	for _, required := range []string{
		"operationID", "operationKind", "saveSessionID", "saveRevision", "changedScopes",
	} {
		if _, present := payload[required]; !present {
			t.Errorf("MutationReceipt JSON is missing required member %q: %s", required, encoded)
		}
	}
}

// A success that commits nothing must be distinguishable from a committed
// mutation by its discriminator alone, and its payload must not carry the three
// members that describe one execution. They have to be absent, not empty: an
// empty operationID is still a member a consumer could branch on.

func TestSetCharacterActiveNoCommitResultOmitsTheExecutionMembers(t *testing.T) {
	engine, saveSessionID := loadSessionEventFixture(t)

	// The fixture slot is already active, so this request changes nothing.
	result, err := engine.SetCharacterActive(saveSessionID, setActiveTestSlot, true, "0")
	if err != nil {
		t.Fatalf("SetCharacterActive: %v", err)
	}
	if result.Changed {
		t.Fatalf("result = %+v, want changed=false", result)
	}
	if result.SaveSessionID != saveSessionID || result.SaveRevision != "0" {
		t.Errorf("result = %+v, want the session at the unchanged revision 0", result)
	}
	if result.OperationID != "" || result.OperationKind != "" || result.ChangedScopes != nil {
		t.Errorf("result = %+v, want no minted execution", result)
	}
	assertAbsentReceiptMembers(t, result,
		[]string{"changed", "saveSessionID", "saveRevision", "characterID", "active"})

	// The committed variant of the same endpoint keeps the complete receipt.
	committed, err := engine.SetCharacterActive(saveSessionID, setActiveTestSlot, false, "0")
	if err != nil {
		t.Fatalf("SetCharacterActive(false): %v", err)
	}
	if !committed.Changed {
		t.Fatalf("result = %+v, want changed=true", committed)
	}
	assertCommittedReceipt(t, committed.MutationReceipt, saveSessionID, kindSetCharacterActive, "1")
}

func TestApplyRepairPlanNoActionResultOmitsTheExecutionMembers(t *testing.T) {
	engine, saveSessionID := loadSessionEventFixture(t)

	result, err := engine.ApplyRepairPlan(saveSessionID, setActiveTestSlot, nil, "0")
	if err != nil {
		t.Fatalf("ApplyRepairPlan: %v", err)
	}
	if result.Applied {
		t.Fatalf("result = %+v, want applied=false", result)
	}
	if result.SaveSessionID != saveSessionID || result.SaveRevision != "0" {
		t.Errorf("result = %+v, want the session at the unchanged revision 0", result)
	}
	if result.OperationID != "" || result.OperationKind != "" || result.ChangedScopes != nil {
		t.Errorf("result = %+v, want no minted execution", result)
	}
	assertAbsentReceiptMembers(t, result,
		[]string{"saveSessionID", "saveRevision", "characterID", "applied"})

	info, err := engine.GetSessionInfo(saveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.SaveRevision != "0" || info.UnsavedChanges || info.EventSequence != "0" {
		t.Errorf("session = %+v, want an untouched session", info)
	}
}

// assertAbsentReceiptMembers proves the payload carries exactly wantKeys: the
// three execution members are gone rather than present and empty, and the
// receipt is still flat with no nested object.
func assertAbsentReceiptMembers(t *testing.T, result any, wantKeys []string) {
	t.Helper()

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal %T: %v", result, err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("decode %s: %v", encoded, err)
	}
	for _, absent := range []string{"operationID", "operationKind", "changedScopes", "receipt"} {
		if _, present := payload[absent]; present {
			t.Errorf("%T JSON carries %q, want it absent: %s", result, absent, encoded)
		}
	}
	for _, key := range wantKeys {
		if _, present := payload[key]; !present {
			t.Errorf("%T JSON is missing %q: %s", result, key, encoded)
		}
	}
	if len(payload) != len(wantKeys) {
		t.Errorf("%T JSON has %d members, want exactly %v: %s",
			result, len(payload), wantKeys, encoded)
	}
}
