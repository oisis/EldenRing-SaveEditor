package diagnostics_test

import (
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestGetDiagnosticLog_NilEngine(t *testing.T) {
	_, err := diagnostics.GetDiagnosticLog(nil, "some-id", "", 50, "", "")
	if err == nil {
		t.Fatal("expected error for nil engine")
	}
}

func TestGetDiagnosticLog_GetterIsNoOp(t *testing.T) {
	engine := saveengine.New()
	path := writeTestPCSave(t, "pc_noop.sl2")
	info, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	sessionID := info.SaveSessionID

	// 1. Initial diagnostic log query
	res1, err := diagnostics.GetDiagnosticLog(engine, sessionID, "", 50, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog: %v", err)
	}
	if len(res1.Records) != 1 {
		t.Fatalf("len(Records) = %d, want 1", len(res1.Records))
	}

	// 2. Second diagnostic log query — verify no new records, no mutation
	res2, err := diagnostics.GetDiagnosticLog(engine, sessionID, "", 50, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog second: %v", err)
	}
	if len(res2.Records) != 1 {
		t.Fatalf("len(Records) second = %d, want 1 (getter mutated log!)", len(res2.Records))
	}

	// 3. Query session info — verify dirty is false, revision is unchanged
	sessInfo, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if sessInfo.UnsavedChanges {
		t.Fatalf("UnsavedChanges = true, want false (getter dirtied session!)")
	}

	// 4. Query undo state — verify no undo point created
	undoState, err := engine.GetUndoState(sessionID, 0)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if undoState.Available {
		t.Fatalf("undoState.Available = true, want false (getter created undo!)")
	}
}

func TestGetSaveValidationReport_DoesNotEmitDiagnosticRecordAndIsNoOpOnJournal(t *testing.T) {
	engine, sessionID := loadReportFixture(t, reportTestFixture{platform: saveengine.PlatformPC})
	catalog := reportTestCatalog(t)

	// 1. Initial state check: session_loaded is the only record
	beforeLog, err := diagnostics.GetDiagnosticLog(engine, sessionID, "", 50, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog before: %v", err)
	}
	if len(beforeLog.Records) != 1 {
		t.Fatalf("len(beforeLog.Records) = %d, want 1", len(beforeLog.Records))
	}

	// 2. Call GetSaveValidationReport
	report, err := diagnostics.GetSaveValidationReport(engine, catalog, sessionID, reportTestSlot, "")
	if err != nil {
		t.Fatalf("GetSaveValidationReport: %v", err)
	}
	if !report.Active {
		t.Fatalf("expected active character report")
	}

	// 3. Confirm diagnostic journal is untouched
	afterLog, err := diagnostics.GetDiagnosticLog(engine, sessionID, "", 50, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog after: %v", err)
	}
	if len(afterLog.Records) != 1 {
		t.Fatalf("GetSaveValidationReport emitted a diagnostic record! len = %d, want 1", len(afterLog.Records))
	}
	if afterLog.Records[0].Seq != beforeLog.Records[0].Seq ||
		afterLog.Records[0].Event != beforeLog.Records[0].Event ||
		afterLog.Records[0].Revision != beforeLog.Records[0].Revision {
		t.Fatalf("diagnostic record changed after validation report: got %+v, want %+v", afterLog.Records[0], beforeLog.Records[0])
	}

	// 4. Confirm session dirty state and undo are unchanged
	sessInfo, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if sessInfo.UnsavedChanges {
		t.Errorf("UnsavedChanges = true, want false")
	}

	undoState, err := engine.GetUndoState(sessionID, reportTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if undoState.Available {
		t.Errorf("undoState.Available = true, want false")
	}
}

func TestGetDiagnosticLog_Integration(t *testing.T) {
	engine := saveengine.New()
	path := writeTestPCSave(t, "pc_endpoint_integration.sl2")
	info, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	sessionID := info.SaveSessionID

	// Write save to advance revision and create save_written event
	target := filepath.Join(t.TempDir(), "target.sl2")
	if _, err := engine.WriteSave(sessionID, "0", target); err != nil {
		t.Fatalf("WriteSave: %v", err)
	}

	res, err := diagnostics.GetDiagnosticLog(engine, sessionID, "", 50, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog: %v", err)
	}
	if res.SaveSessionID != sessionID {
		t.Errorf("SaveSessionID = %q, want %q", res.SaveSessionID, sessionID)
	}
	if len(res.Records) != 2 {
		t.Fatalf("len(Records) = %d, want 2", len(res.Records))
	}
	if res.Records[0].Event != "session_loaded" {
		t.Errorf("record 0 event = %q, want session_loaded", res.Records[0].Event)
	}
	if res.Records[1].Event != "save_written" {
		t.Errorf("record 1 event = %q, want save_written", res.Records[1].Event)
	}
}

// Helpers for endpoint test
func writeTestPCSave(t *testing.T, name string) string {
	t.Helper()
	return writeReportFixture(t, reportTestFixture{platform: saveengine.PlatformPC})
}
