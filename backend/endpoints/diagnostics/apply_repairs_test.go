package diagnostics_test

import (
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestApplyRepairs_ExecutesTheSealedActionsAtomically(t *testing.T) {
	for _, platform := range []saveengine.Platform{saveengine.PlatformPC, saveengine.PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			illegal := [8]uint32{0, 200, 40, 12, 13, 9, 9, 7}
			engine, session, report := planFor(t, reportTestFixture{
				platform:   platform,
				attributes: &illegal,
				inventory: []reportTestRow{
					{index: 0, handle: reportTestAccessory, quantity: 2},
					{index: 1, handle: reportTestGoodsSmallStack, quantity: 0},
				},
			})
			catalog := reportTestCatalog(t)
			ids := allIssueIDs(report)
			plan, err := diagnostics.GetRepairPlan(engine, catalog, session, reportTestSlot, report.SaveRevision, ids)
			if err != nil {
				t.Fatalf("GetRepairPlan: %v", err)
			}
			if len(plan.Actions) != 3 {
				t.Fatalf("planned actions = %d, want quantity, removal and stats", len(plan.Actions))
			}

			result, err := diagnostics.ApplyRepairs(
				engine, catalog, session, reportTestSlot, ids, plan.PlanToken, report.SaveRevision)
			if err != nil {
				t.Fatalf("ApplyRepairs: %v", err)
			}
			if !result.Applied || result.SaveRevision == report.SaveRevision {
				t.Fatalf("result = %+v, want a committed newer revision", result)
			}
			if len(result.Actions) != len(plan.Actions) || len(result.Rejected) != len(plan.Rejected) {
				t.Errorf("receipt differs from re-derived plan: actions=%d/%d rejected=%d/%d",
					len(result.Actions), len(plan.Actions), len(result.Rejected), len(plan.Rejected))
			}

			state, err := engine.GetSessionInfo(session)
			if err != nil {
				t.Fatalf("GetSessionInfo: %v", err)
			}
			if !state.UnsavedChanges {
				t.Error("UnsavedChanges = false after a committed repair")
			}
			undo, err := engine.GetUndoState(session, reportTestSlot)
			if err != nil {
				t.Fatalf("GetUndoState: %v", err)
			}
			if !undo.Available || undo.OperationKind != "apply_repairs" {
				t.Errorf("undo = %+v, want one apply_repairs point", undo)
			}

			assertNoExecutableRepairIssues(t, engine, catalog, session)

			output := filepath.Join(t.TempDir(), "repaired.sl2")
			if _, err := engine.WriteSave(session, result.SaveRevision, output); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloaded := saveengine.New()
			loaded, err := reloaded.LoadSave(output, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave repaired output: %v", err)
			}
			assertNoExecutableRepairIssues(t, reloaded, catalog, loaded.SaveSessionID)
		})
	}
}

func assertNoExecutableRepairIssues(t *testing.T, engine *saveengine.Engine, catalog *gamecatalog.Catalog, session string) {
	t.Helper()
	after, err := diagnostics.GetSaveValidationReport(engine, catalog, session, reportTestSlot, "")
	if err != nil {
		t.Fatalf("GetSaveValidationReport after apply: %v", err)
	}
	for _, issue := range after.Issues {
		switch issue.Code {
		case "quantity_zero", "quantity_above_stack_limit", "attribute_out_of_range", "attribute_below_class_minimum":
			t.Errorf("executable issue remains after apply: %+v", issue)
		}
	}
}

func TestApplyRepairs_RejectsBadOrStalePlanWithoutMutation(t *testing.T) {
	engine, session, report := planFor(t, reportTestFixture{
		inventory: []reportTestRow{{index: 0, handle: reportTestAccessory, quantity: 2}},
	})
	catalog := reportTestCatalog(t)
	ids := allIssueIDs(report)
	plan, err := diagnostics.GetRepairPlan(engine, catalog, session, reportTestSlot, report.SaveRevision, ids)
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}

	if _, err := diagnostics.ApplyRepairs(engine, catalog, session, reportTestSlot, ids, "wrong-token", report.SaveRevision); err == nil {
		t.Error("ApplyRepairs accepted a different token")
	}
	state, err := engine.GetSessionInfo(session)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if state.UnsavedChanges {
		t.Error("bad token marked the session dirty")
	}

	changed, err := engine.SetCharacterRunes(session, reportTestSlot, 1234, report.SaveRevision)
	if err != nil {
		t.Fatalf("SetCharacterRunes: %v", err)
	}
	if _, err := diagnostics.ApplyRepairs(engine, catalog, session, reportTestSlot, ids, plan.PlanToken, report.SaveRevision); err == nil {
		t.Error("ApplyRepairs accepted a plan from an older revision")
	}
	state, err = engine.GetSessionInfo(session)
	if err != nil {
		t.Fatalf("GetSessionInfo after stale apply: %v", err)
	}
	if state.SaveSessionID == "" || state.UnsavedChanges != true {
		t.Errorf("state after the unrelated mutation = %+v", state)
	}
	if stateRevision, err := engine.GetUndoState(session, reportTestSlot); err != nil || stateRevision.SaveRevision != changed.SaveRevision {
		t.Errorf("stale apply changed revision: undo=%+v err=%v, want %q", stateRevision, err, changed.SaveRevision)
	}
}

func TestApplyRepairs_ReturnsRejectedOnlySelectionWithoutMutation(t *testing.T) {
	wrongLevel := uint32(40)
	engine, session, report := planFor(t, reportTestFixture{level: &wrongLevel})
	catalog := reportTestCatalog(t)
	ids := allIssueIDs(report)
	plan, err := diagnostics.GetRepairPlan(engine, catalog, session, reportTestSlot, report.SaveRevision, ids)
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}
	if len(plan.Actions) != 0 || len(plan.Rejected) == 0 {
		t.Fatalf("plan = %+v, want rejected-only selection", plan)
	}

	result, err := diagnostics.ApplyRepairs(
		engine, catalog, session, reportTestSlot, ids, plan.PlanToken, report.SaveRevision)
	if err != nil {
		t.Fatalf("ApplyRepairs: %v", err)
	}
	if result.Applied || result.SaveRevision != report.SaveRevision || len(result.Rejected) != len(plan.Rejected) {
		t.Errorf("rejected-only result = %+v, want unchanged revision and returned rejections", result)
	}
	state, err := engine.GetSessionInfo(session)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if state.UnsavedChanges {
		t.Error("rejected-only selection marked the session dirty")
	}
	undo, err := engine.GetUndoState(session, reportTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if undo.Available {
		t.Error("rejected-only selection created an undo point")
	}
}
