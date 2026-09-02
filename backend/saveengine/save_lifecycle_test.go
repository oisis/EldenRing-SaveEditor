package saveengine

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestOperationHistoryUndoRedoAndSelectiveRevertOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			source, runesAt := writeUndoFixture(t, platform)
			engine := NewWithOptions(EngineOptions{StateDirectory: t.TempDir()})
			loaded, err := engine.LoadSave(source, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			id := loaded.SaveSessionID
			if _, err := engine.SetCharacterRunes(id, setRunesTestSlot, 500, "0"); err != nil {
				t.Fatalf("first SetCharacterRunes: %v", err)
			}
			if _, err := engine.SetCharacterRunes(id, setRunesTestSlot, 700, "1"); err != nil {
				t.Fatalf("second SetCharacterRunes: %v", err)
			}

			history, err := engine.GetOperationHistory(id)
			if err != nil {
				t.Fatalf("GetOperationHistory: %v", err)
			}
			if history.SaveRevision != "2" || history.UndoCount != 2 || history.RedoCount != 0 || len(history.Operations) != 2 {
				t.Fatalf("history after mutations = %+v", history)
			}
			if history.Operations[0].OperationID == history.Operations[1].OperationID ||
				history.Operations[0].CharacterID == nil ||
				*history.Operations[0].CharacterID != setRunesTestSlot {
				t.Fatalf("operation identities = %+v", history.Operations)
			}

			if _, err := engine.UndoLastOperation(id, "2"); err != nil {
				t.Fatalf("UndoLastOperation: %v", err)
			}
			if got := undoTestRunes(t, engine, id, runesAt); got != 500 {
				t.Fatalf("runes after undo = %d, want 500", got)
			}
			if _, err := engine.RedoLastOperation(id, "3"); err != nil {
				t.Fatalf("RedoLastOperation: %v", err)
			}
			if got := undoTestRunes(t, engine, id, runesAt); got != 700 {
				t.Fatalf("runes after redo = %d, want 700", got)
			}

			beforeRejectedRevert := append([]byte(nil), engine.sessions[id].snapshot.data...)
			if _, err := engine.RevertOperation(id, history.Operations[0].OperationID, "4"); err == nil {
				t.Fatal("RevertOperation accepted removal of an operation required by a later one")
			}
			if !bytes.Equal(beforeRejectedRevert, engine.sessions[id].snapshot.data) ||
				engine.sessions[id].session.revisionString() != "4" {
				t.Fatal("rejected selective revert changed the snapshot or revision")
			}

			latest := history.Operations[1].OperationID
			if _, err := engine.RevertOperation(id, latest, "4"); err != nil {
				t.Fatalf("RevertOperation(latest): %v", err)
			}
			if got := undoTestRunes(t, engine, id, runesAt); got != 500 {
				t.Fatalf("runes after selective revert = %d, want 500", got)
			}
			after, err := engine.GetOperationHistory(id)
			if err != nil || len(after.Operations) != 1 || after.RedoCount != 0 {
				t.Fatalf("history after selective revert = %+v, %v", after, err)
			}
		})
	}
}

func TestAutomaticBackupRetentionRemovesOnlyTheOldestMatchingFiles(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "ER0000.sl2")
	if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	created := make([]string, 0, 12)
	for index := 0; index < 12; index++ {
		path := filepath.Join(directory, "ER0000.sl2.20260902000000_"+
			strconv.Itoa(index+2)+"_bak")
		if err := os.WriteFile(path, []byte{byte(index)}, 0o600); err != nil {
			t.Fatalf("write backup %d: %v", index, err)
		}
		stamp := time.Unix(int64(index+1), 0)
		if err := os.Chtimes(path, stamp, stamp); err != nil {
			t.Fatalf("date backup %d: %v", index, err)
		}
		created = append(created, path)
	}
	manual := filepath.Join(directory, "ER0000.sl2.manual.bak")
	if err := os.WriteFile(manual, []byte("manual"), 0o600); err != nil {
		t.Fatalf("write manual backup: %v", err)
	}

	removed, reached, err := pruneAutomaticBackups(target, 10)
	if err != nil || removed != 2 || !reached {
		t.Fatalf("pruneAutomaticBackups = %d, %v, %v; want 2, true, nil", removed, reached, err)
	}
	for _, path := range created[:2] {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("oldest backup %q remains: %v", path, err)
		}
	}
	for _, path := range append(created[2:], manual) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("retained file %q is missing: %v", path, err)
		}
	}
}

func TestSaveLifecycleBacksUpValidatesAndClearsHistoryOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			source, _ := writeUndoFixture(t, platform)
			original, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}
			stateDirectory := t.TempDir()
			engine := NewWithOptions(EngineOptions{StateDirectory: stateDirectory})
			loaded, err := engine.LoadSave(source, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			if _, err := engine.SetCharacterRunes(
				loaded.SaveSessionID, setRunesTestSlot, 456789, "0"); err != nil {
				t.Fatalf("SetCharacterRunes: %v", err)
			}
			validation, err := engine.ValidateReviewChanges(loaded.SaveSessionID, "1")
			if err != nil || !validation.Valid || validation.ValidationToken == "" {
				t.Fatalf("ValidateReviewChanges = %+v, %v", validation, err)
			}
			result, err := engine.Save(
				loaded.SaveSessionID, "1", validation.ValidationToken, false, false)
			if err != nil {
				t.Fatalf("Save: %v", err)
			}
			if result.BackupPath == "" || result.Target != source || result.SaveRevision != "2" {
				t.Fatalf("Save result = %+v", result)
			}
			backup, err := os.ReadFile(result.BackupPath)
			if err != nil || !bytes.Equal(backup, original) {
				t.Fatalf("automatic backup differs from source: %v", err)
			}
			written, err := os.ReadFile(source)
			if err != nil || bytes.Equal(written, original) {
				t.Fatalf("save target was not replaced with the mutation: %v", err)
			}
			if err := validateSerialized(written, platform); err != nil {
				t.Fatalf("validate persisted target: %v", err)
			}
			info, err := engine.GetSessionInfo(loaded.SaveSessionID)
			if err != nil || info.UnsavedChanges {
				t.Fatalf("session after Save = %+v, %v", info, err)
			}
			history, err := engine.GetOperationHistory(loaded.SaveSessionID)
			if err != nil || len(history.Operations) != 0 || history.UndoCount != 0 || history.RedoCount != 0 {
				t.Fatalf("history after Save = %+v, %v", history, err)
			}
			journalPath, _ := engine.recoveryPath(loaded.SaveSessionID)
			if _, err := os.Stat(journalPath); !os.IsNotExist(err) {
				t.Fatalf("recovery journal remains after Save: %v", err)
			}
		})
	}
}

func TestRecoveryRecentFilesAndExternalSaveConflict(t *testing.T) {
	source, runesAt := writeUndoFixture(t, PlatformPC)
	stateDirectory := t.TempDir()
	engine := NewWithOptions(EngineOptions{StateDirectory: stateDirectory})
	loaded, err := engine.LoadSave(source, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if _, err := engine.RecordRecentFile(loaded.SaveSessionID); err != nil {
		t.Fatalf("RecordRecentFile: %v", err)
	}
	if _, err := engine.SetSaveLifecycleSettings(5); err != nil {
		t.Fatalf("SetSaveLifecycleSettings: %v", err)
	}
	if _, err := engine.SetCharacterRunes(loaded.SaveSessionID, setRunesTestSlot, 900, "0"); err != nil {
		t.Fatalf("SetCharacterRunes: %v", err)
	}

	restarted := NewWithOptions(EngineOptions{StateDirectory: stateDirectory})
	journals, err := restarted.GetRecoveryJournals()
	if err != nil || len(journals) != 1 || journals[0].Status != "compatible" {
		t.Fatalf("GetRecoveryJournals = %+v, %v", journals, err)
	}
	recovered, err := restarted.RestoreRecoveryJournal(journals[0].JournalID)
	if err != nil {
		t.Fatalf("RestoreRecoveryJournal: %v", err)
	}
	if got := undoTestRunes(t, restarted, recovered.SaveSessionID, runesAt); got != 900 {
		t.Fatalf("recovered runes = %d, want 900", got)
	}
	recent, err := restarted.GetRecentFiles()
	if err != nil || len(recent) != 1 || recent[0].Path != source {
		t.Fatalf("persisted Recent Files = %+v, %v", recent, err)
	}
	settings, err := restarted.GetSaveLifecycleSettings()
	if err != nil || settings.BackupRetention != 5 {
		t.Fatalf("persisted settings = %+v, %v", settings, err)
	}

	validation, err := restarted.ValidateReviewChanges(recovered.SaveSessionID, recovered.SaveRevision)
	if err != nil {
		t.Fatalf("ValidateReviewChanges: %v", err)
	}
	changed := append([]byte(nil), restarted.sessions[recovered.SaveSessionID].baseline.data...)
	changed[len(changed)-1] ^= 0x01
	if err := os.WriteFile(source, changed, 0o600); err != nil {
		t.Fatalf("replace source externally: %v", err)
	}
	if _, err := restarted.Save(
		recovered.SaveSessionID, recovered.SaveRevision, validation.ValidationToken,
		false, false); err == nil {
		t.Fatal("Save accepted a source changed outside SaveForge")
	}
	if matches, _ := filepath.Glob(source + ".*_bak"); len(matches) != 0 {
		t.Fatalf("rejected Save created backups: %v", matches)
	}
}

func TestSaveRequiresIndependentReviewRiskConfirmations(t *testing.T) {
	source, _ := writeUndoFixture(t, PlatformPC)
	engine := NewWithOptions(EngineOptions{StateDirectory: t.TempDir()})
	loaded, err := engine.LoadSave(source, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	validation, err := engine.ValidateReviewChanges(loaded.SaveSessionID, "0")
	if err != nil {
		t.Fatalf("ValidateReviewChanges: %v", err)
	}
	engine.mutex.Lock()
	authorization := engine.sessions[loaded.SaveSessionID].session.reviewAuthorization
	authorization.warningRequired = true
	authorization.banRiskRequired = true
	engine.mutex.Unlock()

	if _, err := engine.Save(
		loaded.SaveSessionID, "0", validation.ValidationToken, false, false); err == nil {
		t.Fatal("Save accepted missing warning and ban-risk confirmations")
	}
	if _, err := engine.Save(
		loaded.SaveSessionID, "0", validation.ValidationToken, true, false); err == nil {
		t.Fatal("Save accepted missing independent ban-risk confirmation")
	}
	if matches, _ := filepath.Glob(source + ".*_bak"); len(matches) != 0 {
		t.Fatalf("rejected confirmations created backups: %v", matches)
	}
	if _, err := engine.Save(
		loaded.SaveSessionID, "0", validation.ValidationToken, true, true); err != nil {
		t.Fatalf("Save with both required confirmations: %v", err)
	}
}
