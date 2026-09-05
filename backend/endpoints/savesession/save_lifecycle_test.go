package savesession

import (
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// One boundary scenario proves that the lifecycle endpoints delegate to the
// same Engine state instead of reconstructing history, validation or host-local
// stores in the endpoint layer. SaveEngine owns the detailed replay coverage.
func TestSaveLifecycleEndpointBoundary(t *testing.T) {
	engine := saveengine.NewWithOptions(saveengine.EngineOptions{StateDirectory: t.TempDir()})
	loaded, err := LoadSave(engine, writePCFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	history, err := GetOperationHistory(engine, loaded.SaveSessionID)
	if err != nil || history.SaveSessionID != loaded.SaveSessionID || len(history.Operations) != 0 {
		t.Fatalf("GetOperationHistory = %+v, %v", history, err)
	}
	validation, err := ValidateReviewChanges(engine, loaded.SaveSessionID, "0")
	if err != nil || !validation.Valid || validation.ValidationToken == "" {
		t.Fatalf("ValidateReviewChanges = %+v, %v", validation, err)
	}
	written, err := Save(
		engine, loaded.SaveSessionID, "0", validation.ValidationToken, false, false)
	if err != nil || written.SaveRevision != "1" || written.BackupPath == "" {
		t.Fatalf("Save = %+v, %v", written, err)
	}

	recent, err := RecordRecentFile(engine, loaded.SaveSessionID)
	if err != nil || len(recent) != 1 || recent[0].Path != written.Target {
		t.Fatalf("RecordRecentFile = %+v, %v", recent, err)
	}
	if recent, err = RemoveRecentFile(engine, recent[0].Path); err != nil || len(recent) != 0 {
		t.Fatalf("RemoveRecentFile = %+v, %v", recent, err)
	}
	if err := ClearRecentFiles(engine); err != nil {
		t.Fatalf("ClearRecentFiles: %v", err)
	}
	// An empty pattern means the default one, and the reported example is what
	// the pattern in effect actually produces.
	settings, err := SetSaveLifecycleSettings(engine, 5, "")
	if err != nil || settings.BackupRetention != 5 ||
		settings.BackupNamePattern != "{filename}.{timestamp}" ||
		settings.BackupNameExample == "" {
		t.Fatalf("SetSaveLifecycleSettings = %+v, %v", settings, err)
	}
	if settings, err = GetSaveLifecycleSettings(engine); err != nil || settings.BackupRetention != 5 {
		t.Fatalf("GetSaveLifecycleSettings = %+v, %v", settings, err)
	}
	custom, err := SetSaveLifecycleSettings(engine, 5, "{timestamp}-{filename}")
	if err != nil || custom.BackupNameExample != "20260824202530-ER0000.sl2_bak" {
		t.Fatalf("SetSaveLifecycleSettings with a custom pattern = %+v, %v", custom, err)
	}
	// The backend owns the validation rules: an unsafe pattern is refused rather
	// than sanitised, and the stored one is left as it was.
	if _, err := SetSaveLifecycleSettings(engine, 5, "../{filename}.{timestamp}"); err == nil {
		t.Fatal("SetSaveLifecycleSettings accepted a pattern that escapes the directory")
	}
	if stored, err := GetSaveLifecycleSettings(engine); err != nil ||
		stored.BackupNamePattern != "{timestamp}-{filename}" {
		t.Fatalf("settings after the refusal = %+v, %v", stored, err)
	}
	if journals, err := GetRecoveryJournals(engine); err != nil || len(journals) != 0 {
		t.Fatalf("GetRecoveryJournals = %+v, %v", journals, err)
	}

	validation, err = ValidateReviewChanges(engine, loaded.SaveSessionID, "1")
	if err != nil {
		t.Fatalf("second ValidateReviewChanges: %v", err)
	}
	target := filepath.Join(t.TempDir(), "ER0000-copy.sl2")
	copyResult, err := SaveAs(
		engine, loaded.SaveSessionID, "1", validation.ValidationToken, false, false, target)
	if err != nil || copyResult.Target != target || copyResult.SaveRevision != "2" {
		t.Fatalf("SaveAs = %+v, %v", copyResult, err)
	}
}

func TestSaveLifecycleEndpointsRejectMissingEngine(t *testing.T) {
	checks := []struct {
		name string
		call func() error
	}{
		{"GetOperationHistory", func() error { _, err := GetOperationHistory(nil, "session"); return err }},
		{"UndoLastOperation", func() error { _, err := UndoLastOperation(nil, "session", "0"); return err }},
		{"RedoLastOperation", func() error { _, err := RedoLastOperation(nil, "session", "0"); return err }},
		{"RevertOperation", func() error { _, err := RevertOperation(nil, "session", "operation", "0"); return err }},
		{"DiscardChanges", func() error { _, err := DiscardChanges(nil, "session", "0"); return err }},
		{"ValidateReviewChanges", func() error { _, err := ValidateReviewChanges(nil, "session", "0"); return err }},
		{"Save", func() error { _, err := Save(nil, "session", "0", "token", false, false); return err }},
		{"SaveAs", func() error { _, err := SaveAs(nil, "session", "0", "token", false, false, "target"); return err }},
		{"GetRecentFiles", func() error { _, err := GetRecentFiles(nil); return err }},
		{"RecordRecentFile", func() error { _, err := RecordRecentFile(nil, "session"); return err }},
		{"RemoveRecentFile", func() error { _, err := RemoveRecentFile(nil, "path"); return err }},
		{"ClearRecentFiles", func() error { return ClearRecentFiles(nil) }},
		{"GetRecoveryJournals", func() error { _, err := GetRecoveryJournals(nil); return err }},
		{"GetRecoveryJournal", func() error { _, err := GetRecoveryJournal(nil, "journal"); return err }},
		{"RestoreRecoveryJournal", func() error { _, err := RestoreRecoveryJournal(nil, "journal"); return err }},
		{"DiscardRecoveryJournal", func() error { return DiscardRecoveryJournal(nil, "journal") }},
		{"ExportRecoveryJournal", func() error { return ExportRecoveryJournal(nil, "journal", "target") }},
		{"GetSaveLifecycleSettings", func() error { _, err := GetSaveLifecycleSettings(nil); return err }},
		{"SetSaveLifecycleSettings", func() error { _, err := SetSaveLifecycleSettings(nil, 10, ""); return err }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); err == nil {
				t.Fatal("missing Engine was accepted")
			}
		})
	}
}
