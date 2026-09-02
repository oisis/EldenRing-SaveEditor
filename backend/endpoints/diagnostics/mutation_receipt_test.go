package diagnostics_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/diagnostics"
)

// ApplyRepairs is the last save-session mutation to embed the shared receipt,
// and the only one besides SetCharacterActive with a success that commits
// nothing. These tests own both variants of its public result.
//
// The scope list is written out literally rather than read back from
// saveengine.ChangedScopesForMutationKind: a test that asks the same table the
// implementation asks proves only that the table was read.
var applyRepairsScopes = []string{
	"save.session", "character.list", "character.profile", "character.stats",
	"inventory", "storage", "equipment.loadout", "diagnostics.report",
}

func TestApplyRepairsAppliedResultCarriesItsCommitReceipt(t *testing.T) {
	illegal := [8]uint32{0, 200, 40, 12, 13, 9, 9, 7}
	engine, session, report := planFor(t, reportTestFixture{
		attributes: &illegal,
		inventory: []reportTestRow{
			{index: 0, handle: reportTestAccessory, quantity: 2},
			{index: 1, handle: reportTestGoodsSmallStack, quantity: 0},
		},
	})
	catalog := reportTestCatalog(t)
	ids := allIssueIDs(report)
	plan, err := diagnostics.GetRepairPlan(
		engine, catalog, session, reportTestSlot, report.SaveRevision, ids)
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}

	result, err := diagnostics.ApplyRepairs(
		engine, catalog, session, reportTestSlot, ids, plan.PlanToken, report.SaveRevision)
	if err != nil {
		t.Fatalf("ApplyRepairs: %v", err)
	}
	if !result.Applied {
		t.Fatalf("result = %+v, want a committed execution", result)
	}
	if result.OperationID == "" {
		t.Error("a committed repair carries no operationID")
	}
	if result.OperationKind != diagnostics.ApplyRepairsEndpointID {
		t.Errorf("operationKind = %q, want the EndpointID %q",
			result.OperationKind, diagnostics.ApplyRepairsEndpointID)
	}
	if result.SaveSessionID != session {
		t.Errorf("saveSessionID = %q, want %q", result.SaveSessionID, session)
	}
	if result.SaveRevision == report.SaveRevision {
		t.Errorf("saveRevision = %q, want the newly committed revision", result.SaveRevision)
	}
	if !reflect.DeepEqual(result.ChangedScopes, applyRepairsScopes) {
		t.Errorf("changedScopes = %v, want exactly %v in canonical order",
			result.ChangedScopes, applyRepairsScopes)
	}
	assertFlatApplyRepairsJSON(t, result, []string{
		"operationID", "operationKind", "saveSessionID", "saveRevision", "changedScopes",
		"characterID", "applied", "actions", "rejected",
	})
}

func TestApplyRepairsNoActionResultCarriesNoExecution(t *testing.T) {
	wrongLevel := uint32(40)
	engine, session, report := planFor(t, reportTestFixture{level: &wrongLevel})
	catalog := reportTestCatalog(t)
	ids := allIssueIDs(report)
	plan, err := diagnostics.GetRepairPlan(
		engine, catalog, session, reportTestSlot, report.SaveRevision, ids)
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}

	result, err := diagnostics.ApplyRepairs(
		engine, catalog, session, reportTestSlot, ids, plan.PlanToken, report.SaveRevision)
	if err != nil {
		t.Fatalf("ApplyRepairs: %v", err)
	}
	if result.Applied {
		t.Fatalf("result = %+v, want a selection with no executable action", result)
	}
	if result.SaveSessionID != session || result.SaveRevision != report.SaveRevision {
		t.Errorf("result = %+v, want the session at its unchanged revision %q",
			result, report.SaveRevision)
	}
	if result.OperationID != "" || result.OperationKind != "" || result.ChangedScopes != nil {
		t.Errorf("result = %+v, want no minted execution", result)
	}
	assertFlatApplyRepairsJSON(t, result, []string{
		"saveSessionID", "saveRevision", "characterID", "applied", "actions", "rejected",
	})

	// The event stream and the session state must be exactly where they were.
	state, err := engine.GetSessionInfo(session)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if state.EventSequence != "0" || state.SaveRevision != report.SaveRevision ||
		state.UnsavedChanges {
		t.Errorf("session = %+v, want an untouched session", state)
	}
}

// A rejected execution returns the complete zero result and never exposes the
// operationID it had prepared.
func TestRejectedApplyRepairsExposesNoPartialReceipt(t *testing.T) {
	engine, session, report := planFor(t, reportTestFixture{
		inventory: []reportTestRow{{index: 0, handle: reportTestAccessory, quantity: 2}},
	})
	catalog := reportTestCatalog(t)
	ids := allIssueIDs(report)
	plan, err := diagnostics.GetRepairPlan(
		engine, catalog, session, reportTestSlot, report.SaveRevision, ids)
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}

	result, err := diagnostics.ApplyRepairs(
		engine, catalog, session, reportTestSlot, ids, plan.PlanToken, "99")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", result)
	}
	if !reflect.DeepEqual(result, diagnostics.ApplyRepairsResult{}) {
		t.Errorf("rejected ApplyRepairs = %+v, want the complete zero result", result)
	}
	state, err := engine.GetSessionInfo(session)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if state.EventSequence != "0" {
		t.Errorf("eventSequence = %q, want no published event for a rejection",
			state.EventSequence)
	}
}

func assertFlatApplyRepairsJSON(t *testing.T, result any, wantKeys []string) {
	t.Helper()

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal %T: %v", result, err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("decode %s: %v", encoded, err)
	}
	if _, nested := payload["receipt"]; nested {
		t.Errorf("%T nests the receipt instead of flattening it: %s", result, encoded)
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
