package saveengine

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func writeTestPCSave(t *testing.T, name string) string {
	t.Helper()
	return writeApplyTemplateFixture(t, PlatformPC, false)
}

func writeTestPS4Save(t *testing.T, name string) string {
	t.Helper()
	return writeApplyTemplateFixture(t, PlatformPS4, false)
}

func TestDiagnosticLog_EmittersLifecycle(t *testing.T) {
	fixedTime := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	engine := New()
	engine.now = func() time.Time { return fixedTime }

	path := writeTestPCSave(t, "pc_emitter.sl2")
	info, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	sessionID := info.SaveSessionID

	// 1. Check session_loaded event
	logResult, err := engine.GetDiagnosticLog(sessionID, "", 50, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog: %v", err)
	}
	if len(logResult.Records) != 1 {
		t.Fatalf("len(Records) = %d, want 1", len(logResult.Records))
	}
	rec0 := logResult.Records[0]
	if rec0.Seq != 1 {
		t.Errorf("rec0.Seq = %d, want 1", rec0.Seq)
	}
	if rec0.Timestamp != fixedTime.Format(time.RFC3339) {
		t.Errorf("rec0.Timestamp = %q, want %q", rec0.Timestamp, fixedTime.Format(time.RFC3339))
	}
	if rec0.Severity != DiagnosticSeverityInfo {
		t.Errorf("rec0.Severity = %q, want %q", rec0.Severity, DiagnosticSeverityInfo)
	}
	if rec0.Scope != DiagnosticScopeSession {
		t.Errorf("rec0.Scope = %q, want %q", rec0.Scope, DiagnosticScopeSession)
	}
	if rec0.Event != DiagnosticEventSessionLoaded {
		t.Errorf("rec0.Event = %q, want %q", rec0.Event, DiagnosticEventSessionLoaded)
	}
	if rec0.Message != DiagnosticMessageSessionLoaded {
		t.Errorf("rec0.Message = %q, want %q", rec0.Message, DiagnosticMessageSessionLoaded)
	}
	if rec0.Revision != "0" {
		t.Errorf("rec0.Revision = %q, want %q", rec0.Revision, "0")
	}
	if rec0.CharacterID != nil {
		t.Errorf("rec0.CharacterID = %v, want nil", rec0.CharacterID)
	}

	// 2. WriteSave -> save_written event
	targetPath := filepath.Join(t.TempDir(), "target.sl2")
	writeResult, err := engine.WriteSave(sessionID, "0", targetPath)
	if err != nil {
		t.Fatalf("WriteSave: %v", err)
	}
	if writeResult.SaveRevision != "1" {
		t.Fatalf("SaveRevision = %q, want 1", writeResult.SaveRevision)
	}

	logResult, err = engine.GetDiagnosticLog(sessionID, "", 50, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog after write: %v", err)
	}
	if len(logResult.Records) != 2 {
		t.Fatalf("len(Records) = %d, want 2", len(logResult.Records))
	}
	rec1 := logResult.Records[1]
	if rec1.Seq != 2 {
		t.Errorf("rec1.Seq = %d, want 2", rec1.Seq)
	}
	if rec1.Event != DiagnosticEventSaveWritten {
		t.Errorf("rec1.Event = %q, want %q", rec1.Event, DiagnosticEventSaveWritten)
	}
	if rec1.Revision != "1" {
		t.Errorf("rec1.Revision = %q, want 1", rec1.Revision)
	}

	// 3. ApplyRepairPlan with actions == 0 -> no event emitted
	repairResult0, err := engine.ApplyRepairPlan(sessionID, 0, nil, "1")
	if err != nil {
		t.Fatalf("ApplyRepairPlan (0 actions): %v", err)
	}
	if repairResult0.Applied {
		t.Errorf("Applied = true, want false")
	}
	logResult, err = engine.GetDiagnosticLog(sessionID, "", 50, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog: %v", err)
	}
	if len(logResult.Records) != 2 {
		t.Fatalf("len(Records) after 0 actions = %d, want 2", len(logResult.Records))
	}

	// 4. ApplyRepairPlan with valid action -> repairs_applied event
	// Setup a valid character attribute write action
	repairAction := RepairAction{
		Operation: RepairOperationSetCharacterStats,
		Attributes: &CharacterAttributes{
			Vigor: 20, Mind: 20, Endurance: 20, Strength: 20,
			Dexterity: 20, Intelligence: 20, Faith: 20, Arcane: 20,
		},
	}
	repairResult1, err := engine.ApplyRepairPlan(sessionID, 0, []RepairAction{repairAction}, "1")
	if err != nil {
		t.Fatalf("ApplyRepairPlan (with action): %v", err)
	}
	if !repairResult1.Applied {
		t.Fatalf("Applied = false, want true")
	}
	if repairResult1.SaveRevision != "2" {
		t.Fatalf("SaveRevision = %q, want 2", repairResult1.SaveRevision)
	}

	logResult, err = engine.GetDiagnosticLog(sessionID, "", 50, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog after repair: %v", err)
	}
	if len(logResult.Records) != 3 {
		t.Fatalf("len(Records) = %d, want 3", len(logResult.Records))
	}
	rec2 := logResult.Records[2]
	if rec2.Seq != 3 {
		t.Errorf("rec2.Seq = %d, want 3", rec2.Seq)
	}
	if rec2.Scope != DiagnosticScopeRepairs {
		t.Errorf("rec2.Scope = %q, want %q", rec2.Scope, DiagnosticScopeRepairs)
	}
	if rec2.Event != DiagnosticEventRepairsApplied {
		t.Errorf("rec2.Event = %q, want %q", rec2.Event, DiagnosticEventRepairsApplied)
	}
	if rec2.Message != DiagnosticMessageRepairsApplied {
		t.Errorf("rec2.Message = %q, want %q", rec2.Message, DiagnosticMessageRepairsApplied)
	}
	if rec2.CharacterID == nil || *rec2.CharacterID != 0 {
		t.Errorf("rec2.CharacterID = %v, want 0", rec2.CharacterID)
	}
	if rec2.Revision != "2" {
		t.Errorf("rec2.Revision = %q, want 2", rec2.Revision)
	}
}

func TestDiagnosticLog_RingBufferRolloverAndCursorExpiry(t *testing.T) {
	engine := New()
	path := writeTestPCSave(t, "pc_ring.sl2")
	info, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	sessionID := info.SaveSessionID

	// Manually inject 550 records into the session journal to test ring buffer
	engine.mutex.Lock()
	session := engine.sessions[sessionID].session
	for i := 2; i <= 550; i++ {
		session.appendDiagnosticRecord(
			time.Now().UTC(),
			DiagnosticScopeSession,
			DiagnosticSeverityInfo,
			DiagnosticEventSaveWritten,
			DiagnosticMessageSaveWritten,
			nil,
			fmt.Sprint(i),
		)
	}
	engine.mutex.Unlock()

	// The session should hold exactly 500 records: seq 51 to 550
	res, err := engine.GetDiagnosticLog(sessionID, "", 10, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog: %v", err)
	}
	if res.TotalBuffered != 500 {
		t.Fatalf("TotalBuffered = %d, want 500", res.TotalBuffered)
	}
	if res.OldestAvailableCursor != "51" {
		t.Fatalf("OldestAvailableCursor = %q, want 51", res.OldestAvailableCursor)
	}
	if len(res.Records) != 10 {
		t.Fatalf("len(Records) = %d, want 10", len(res.Records))
	}
	if res.Records[0].Seq != 51 {
		t.Fatalf("first seq = %d, want 51", res.Records[0].Seq)
	}
	if res.Records[9].Seq != 60 {
		t.Fatalf("tenth seq = %d, want 60", res.Records[9].Seq)
	}
	if res.NextCursor != "60" {
		t.Fatalf("NextCursor = %q, want 60", res.NextCursor)
	}
	if !res.HasMore {
		t.Fatalf("HasMore = false, want true")
	}
	if res.CursorExpired {
		t.Fatalf("CursorExpired = true for empty cursor, want false")
	}

	// Test boundary cursor: cursor="50" is (oldest - 1), so next record to read is 51 -> NOT expired
	res50, err := engine.GetDiagnosticLog(sessionID, "50", 10, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog(50): %v", err)
	}
	if res50.CursorExpired {
		t.Fatalf("cursor=50 should NOT be expired")
	}
	if len(res50.Records) != 10 || res50.Records[0].Seq != 51 {
		t.Fatalf("cursor=50 should return starting from 51")
	}

	// Test expired cursor: cursor="49" means expected 50, but 50 was overwritten -> expired!
	res49, err := engine.GetDiagnosticLog(sessionID, "49", 10, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog(49): %v", err)
	}
	if !res49.CursorExpired {
		t.Fatalf("cursor=49 should be expired")
	}
	if len(res49.Records) != 10 || res49.Records[0].Seq != 51 {
		t.Fatalf("expired cursor should return starting from oldest 51")
	}

	// Test cursor at the end: cursor="550"
	res550, err := engine.GetDiagnosticLog(sessionID, "550", 10, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog(550): %v", err)
	}
	if len(res550.Records) != 0 {
		t.Fatalf("len(Records) at end = %d, want 0", len(res550.Records))
	}
	if res550.HasMore {
		t.Fatalf("HasMore = true, want false")
	}
	if res550.NextCursor != "550" {
		t.Fatalf("NextCursor = %q, want 550", res550.NextCursor)
	}
	if res550.CursorExpired {
		t.Fatalf("cursor=550 should not be expired")
	}
}

func TestDiagnosticLog_FilteringAndPagination(t *testing.T) {
	engine := New()
	path := writeTestPCSave(t, "pc_filters.sl2")
	info, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	sessionID := info.SaveSessionID

	// Append 1 repair event
	engine.mutex.Lock()
	session := engine.sessions[sessionID].session
	slot := 0
	session.appendDiagnosticRecord(
		time.Now().UTC(),
		DiagnosticScopeRepairs,
		DiagnosticSeverityInfo,
		DiagnosticEventRepairsApplied,
		DiagnosticMessageRepairsApplied,
		&slot,
		"1",
	)
	engine.mutex.Unlock()

	// Total 2 records: [1: session/info, 2: repairs/info]

	// Scope filter = "session"
	resSession, err := engine.GetDiagnosticLog(sessionID, "", 50, "", "session")
	if err != nil {
		t.Fatalf("filter session: %v", err)
	}
	if len(resSession.Records) != 1 || resSession.Records[0].Event != "session_loaded" {
		t.Fatalf("scope=session unexpected records: %+v", resSession.Records)
	}
	if resSession.NextCursor != "1" {
		t.Fatalf("NextCursor = %q, want 1", resSession.NextCursor)
	}

	// Scope filter = "repairs"
	resRepairs, err := engine.GetDiagnosticLog(sessionID, "", 50, "", "repairs")
	if err != nil {
		t.Fatalf("filter repairs: %v", err)
	}
	if len(resRepairs.Records) != 1 || resRepairs.Records[0].Event != "repairs_applied" {
		t.Fatalf("scope=repairs unexpected records: %+v", resRepairs.Records)
	}
	if resRepairs.NextCursor != "2" {
		t.Fatalf("NextCursor = %q, want 2", resRepairs.NextCursor)
	}

	// Severity filter = "warning" (valid severity, 0 records in v1)
	resWarning, err := engine.GetDiagnosticLog(sessionID, "", 50, "warning", "")
	if err != nil {
		t.Fatalf("filter warning: %v", err)
	}
	if len(resWarning.Records) != 0 {
		t.Fatalf("severity=warning should return 0 records, got %d", len(resWarning.Records))
	}
	if resWarning.HasMore {
		t.Fatalf("HasMore = true, want false")
	}
	if resWarning.NextCursor != "2" {
		t.Fatalf("NextCursor for empty filter match = %q, want newest seq 2", resWarning.NextCursor)
	}

	// Severity filter = "error" (valid severity, 0 records in v1)
	resError, err := engine.GetDiagnosticLog(sessionID, "", 50, "error", "")
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}
	if len(resError.Records) != 0 {
		t.Fatalf("severity=error should return 0 records, got %d", len(resError.Records))
	}

	// Pagination: limit=1
	page1, err := engine.GetDiagnosticLog(sessionID, "", 1, "", "")
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(page1.Records) != 1 || page1.Records[0].Seq != 1 {
		t.Fatalf("page 1 unexpected: %+v", page1.Records)
	}
	if !page1.HasMore {
		t.Fatalf("page 1 HasMore should be true")
	}
	if page1.NextCursor != "1" {
		t.Fatalf("page 1 NextCursor = %q, want 1", page1.NextCursor)
	}

	page2, err := engine.GetDiagnosticLog(sessionID, page1.NextCursor, 1, "", "")
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(page2.Records) != 1 || page2.Records[0].Seq != 2 {
		t.Fatalf("page 2 unexpected: %+v", page2.Records)
	}
	if page2.HasMore {
		t.Fatalf("page 2 HasMore should be false")
	}
	if page2.NextCursor != "2" {
		t.Fatalf("page 2 NextCursor = %q, want 2", page2.NextCursor)
	}
}

func TestDiagnosticLog_ValidationAndErrors(t *testing.T) {
	engine := New()
	path := writeTestPCSave(t, "pc_val.sl2")
	info, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	sessionID := info.SaveSessionID

	// Required sessionID
	if _, err := engine.GetDiagnosticLog("", "", 50, "", ""); err == nil {
		t.Errorf("expected error for empty sessionID")
	}

	// Unknown sessionID
	if _, err := engine.GetDiagnosticLog("unknown-id", "", 50, "", ""); err == nil {
		t.Errorf("expected error for unknown sessionID")
	}

	// Invalid limit
	if _, err := engine.GetDiagnosticLog(sessionID, "", -1, "", ""); err == nil {
		t.Errorf("expected error for limit -1")
	}
	if _, err := engine.GetDiagnosticLog(sessionID, "", 201, "", ""); err == nil {
		t.Errorf("expected error for limit 201")
	}

	// Invalid severity
	if _, err := engine.GetDiagnosticLog(sessionID, "", 50, "critical", ""); err == nil {
		t.Errorf("expected error for severity 'critical'")
	}

	// Invalid scope
	if _, err := engine.GetDiagnosticLog(sessionID, "", 50, "", "inventory"); err == nil {
		t.Errorf("expected error for scope 'inventory'")
	}

	// Canonical cursor validations
	nonCanonicalCursors := []string{"01", "00", "007", "+1", " 1", "1 ", "-1", "abc", "18446744073709551616"}
	for _, badCursor := range nonCanonicalCursors {
		if _, err := engine.GetDiagnosticLog(sessionID, badCursor, 50, "", ""); err == nil {
			t.Errorf("expected error for non-canonical cursor %q", badCursor)
		}
	}

	// Valid cursor "0"
	if _, err := engine.GetDiagnosticLog(sessionID, "0", 50, "", ""); err != nil {
		t.Errorf("cursor '0' should be accepted, got err: %v", err)
	}
}

func TestDiagnosticLog_ApplyRepairsAtomicEmissionAndOrder(t *testing.T) {
	engine := New()
	path := writeTestPCSave(t, "pc_atomic_repair.sl2")
	info, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	sessionID := info.SaveSessionID

	// 1. Zero actions -> Applied=false, no event emitted
	resNoAction, err := engine.ApplyRepairPlan(sessionID, 0, nil, "0")
	if err != nil {
		t.Fatalf("ApplyRepairPlan zero actions: %v", err)
	}
	if resNoAction.Applied {
		t.Errorf("Applied = true for zero actions, want false")
	}
	logRes, err := engine.GetDiagnosticLog(sessionID, "", 50, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog: %v", err)
	}
	if len(logRes.Records) != 1 {
		t.Fatalf("zero actions emitted a record! len = %d, want 1", len(logRes.Records))
	}

	// 2. Concurrent repair & write operations to verify strict monotonic ordering
	var wg sync.WaitGroup
	// Run an ApplyRepairPlan that succeeds
	action := RepairAction{
		Operation: RepairOperationSetCharacterStats,
		Attributes: &CharacterAttributes{
			Vigor: 25, Mind: 20, Endurance: 20, Strength: 20,
			Dexterity: 20, Intelligence: 20, Faith: 20, Arcane: 20,
		},
	}

	repairRes, err := engine.ApplyRepairPlan(sessionID, 0, []RepairAction{action}, "0")
	if err != nil {
		t.Fatalf("ApplyRepairPlan action: %v", err)
	}
	if !repairRes.Applied || repairRes.SaveRevision != "1" {
		t.Fatalf("ApplyRepairPlan result: %+v, want Applied=true, SaveRevision=1", repairRes)
	}

	// Immediate log check: record must be present with revision "1" and seq 2
	logRes, err = engine.GetDiagnosticLog(sessionID, "", 50, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog: %v", err)
	}
	if len(logRes.Records) != 2 {
		t.Fatalf("len(Records) = %d, want 2", len(logRes.Records))
	}
	if logRes.Records[1].Event != DiagnosticEventRepairsApplied {
		t.Errorf("record 1 event = %q, want %q", logRes.Records[1].Event, DiagnosticEventRepairsApplied)
	}
	if logRes.Records[1].Revision != "1" {
		t.Errorf("record 1 revision = %q, want %q", logRes.Records[1].Revision, "1")
	}
	if logRes.Records[1].Seq != 2 {
		t.Errorf("record 1 seq = %d, want 2", logRes.Records[1].Seq)
	}

	// 3. Concurrent readers while performing WriteSave
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				_, _ = engine.GetDiagnosticLog(sessionID, "", 50, "", "")
			}
		}()
	}

	target := filepath.Join(t.TempDir(), "target_atomic.sl2")
	writeRes, err := engine.WriteSave(sessionID, "1", target)
	if err != nil {
		t.Fatalf("WriteSave: %v", err)
	}
	if writeRes.SaveRevision != "2" {
		t.Fatalf("WriteSave revision = %q, want 2", writeRes.SaveRevision)
	}

	wg.Wait()

	// Final verification of journal ordering: session_loaded (1, rev 0) -> repairs_applied (2, rev 1) -> save_written (3, rev 2)
	finalLog, err := engine.GetDiagnosticLog(sessionID, "", 50, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog: %v", err)
	}
	if len(finalLog.Records) != 3 {
		t.Fatalf("len(finalLog.Records) = %d, want 3", len(finalLog.Records))
	}
	for idx, expectedEvent := range []string{DiagnosticEventSessionLoaded, DiagnosticEventRepairsApplied, DiagnosticEventSaveWritten} {
		rec := finalLog.Records[idx]
		if rec.Seq != uint64(idx+1) {
			t.Errorf("record %d seq = %d, want %d", idx, rec.Seq, idx+1)
		}
		if rec.Event != expectedEvent {
			t.Errorf("record %d event = %q, want %q", idx, rec.Event, expectedEvent)
		}
	}
}

func TestDiagnosticLog_DeepCopyAndPrivacy(t *testing.T) {
	engine := New()
	path := writeTestPCSave(t, "pc_privacy.sl2")
	info, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	sessionID := info.SaveSessionID

	engine.mutex.Lock()
	session := engine.sessions[sessionID].session
	slot := 3
	session.appendDiagnosticRecord(
		time.Now().UTC(),
		DiagnosticScopeRepairs,
		DiagnosticSeverityInfo,
		DiagnosticEventRepairsApplied,
		DiagnosticMessageRepairsApplied,
		&slot,
		"1",
	)
	engine.mutex.Unlock()

	res, err := engine.GetDiagnosticLog(sessionID, "", 50, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog: %v", err)
	}

	// Check privacy: ensure no path or secret leaks
	for _, rec := range res.Records {
		if strings.Contains(rec.Message, "/") || strings.Contains(rec.Message, "\\") {
			t.Errorf("leak detected in message: %q", rec.Message)
		}
		if strings.Contains(rec.Event, "/") || strings.Contains(rec.Event, "\\") {
			t.Errorf("leak detected in event: %q", rec.Event)
		}
	}

	// Mutate returned copy
	if len(res.Records) > 1 && res.Records[1].CharacterID != nil {
		*res.Records[1].CharacterID = 99
	}

	// Re-query and verify internal state wasn't mutated
	res2, err := engine.GetDiagnosticLog(sessionID, "", 50, "", "")
	if err != nil {
		t.Fatalf("GetDiagnosticLog: %v", err)
	}
	if *res2.Records[1].CharacterID != 3 {
		t.Fatalf("internal CharacterID was modified! got %d, want 3", *res2.Records[1].CharacterID)
	}
}

func TestDiagnosticLog_PCAndPS4_LifecycleAndReload(t *testing.T) {
	for _, platform := range []struct {
		name        string
		savePlat    Platform
		platStr     string
		fixtureName string
	}{
		{name: "PC", savePlat: PlatformPC, platStr: "", fixtureName: "pc_reload.sl2"},
		{name: "PS4", savePlat: PlatformPS4, platStr: "ps4", fixtureName: "ps4_reload.bin"},
	} {
		t.Run(platform.name, func(t *testing.T) {
			engine1 := New()
			path := writeApplyTemplateFixture(t, platform.savePlat, false)
			info, err := engine1.LoadSave(path, platform.platStr)
			if err != nil {
				t.Fatalf("LoadSave (%s): %v", platform.name, err)
			}
			sessionID1 := info.SaveSessionID

			// 1. Initial log must contain exactly 1 session_loaded record
			log1, err := engine1.GetDiagnosticLog(sessionID1, "", 50, "", "")
			if err != nil {
				t.Fatalf("GetDiagnosticLog 1 (%s): %v", platform.name, err)
			}
			if len(log1.Records) != 1 || log1.Records[0].Event != DiagnosticEventSessionLoaded {
				t.Fatalf("initial records (%s): %+v", platform.name, log1.Records)
			}

			// 2. Write save to a temporary file
			outPath := filepath.Join(t.TempDir(), platform.fixtureName)
			writeRes, err := engine1.WriteSave(sessionID1, "0", outPath)
			if err != nil {
				t.Fatalf("WriteSave (%s): %v", platform.name, err)
			}
			if writeRes.SaveRevision != "1" {
				t.Fatalf("WriteSave revision (%s) = %q, want 1", platform.name, writeRes.SaveRevision)
			}

			// Engine1 session now has 2 records: session_loaded and save_written
			log1AfterWrite, err := engine1.GetDiagnosticLog(sessionID1, "", 50, "", "")
			if err != nil {
				t.Fatalf("GetDiagnosticLog after write (%s): %v", platform.name, err)
			}
			if len(log1AfterWrite.Records) != 2 {
				t.Fatalf("len(Records) after write (%s) = %d, want 2", platform.name, len(log1AfterWrite.Records))
			}

			// 3. Reload written file in a fresh Engine instance
			engine2 := New()
			info2, err := engine2.LoadSave(outPath, platform.platStr)
			if err != nil {
				t.Fatalf("LoadSave fresh engine (%s): %v", platform.name, err)
			}
			sessionID2 := info2.SaveSessionID

			// 4. Fresh session has exactly 1 new session_loaded record (in-memory journal is not persisted in save file)
			log2, err := engine2.GetDiagnosticLog(sessionID2, "", 50, "", "")
			if err != nil {
				t.Fatalf("GetDiagnosticLog fresh session (%s): %v", platform.name, err)
			}
			if len(log2.Records) != 1 {
				t.Fatalf("fresh session records count (%s) = %d, want 1 (journal leaked into save binary!)", platform.name, len(log2.Records))
			}
			if log2.Records[0].Seq != 1 || log2.Records[0].Event != DiagnosticEventSessionLoaded || log2.Records[0].Revision != "0" {
				t.Fatalf("fresh session record (%s) = %+v, want new session_loaded record", platform.name, log2.Records[0])
			}
		})
	}
}

func TestDiagnosticLog_ConcurrentReadsAndWrites(t *testing.T) {
	engine := New()
	path := writeTestPCSave(t, "pc_race.sl2")
	info, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	sessionID := info.SaveSessionID

	var wg sync.WaitGroup
	// 5 readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = engine.GetDiagnosticLog(sessionID, "", 50, "", "")
			}
		}()
	}

	// 2 writers
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			target := filepath.Join(t.TempDir(), fmt.Sprintf("race_%d.sl2", id))
			for j := 0; j < 10; j++ {
				engine.mutex.Lock()
				rev := engine.sessions[sessionID].session.revisionString()
				engine.mutex.Unlock()
				_, _ = engine.WriteSave(sessionID, rev, target)
			}
		}(i)
	}

	wg.Wait()
}
