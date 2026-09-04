package saveengine

import (
	"os"
	"path/filepath"
	"testing"
)

// TestExportForDeploymentWritesAValidatedFileAndLeavesTheSessionDirty covers the
// shared preparation phase deployment relies on: it must produce a file a
// reload accepts, and it must leave the session exactly as dirty as it was.
//
// The second half is the regression this test exists for. Preparing a
// deployment is not a Save: if it cleared the history, advanced the revision or
// marked the session clean, the user's own local file would silently look
// saved when nothing was written to it.
func TestExportForDeploymentWritesAValidatedFileAndLeavesTheSessionDirty(t *testing.T) {
	source, _ := writeUndoFixture(t, PlatformPC)
	engine := NewWithOptions(EngineOptions{StateDirectory: t.TempDir()})
	loaded, err := engine.LoadSave(source, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	id := loaded.SaveSessionID
	if _, err := engine.SetCharacterRunes(id, setRunesTestSlot, 500, "0"); err != nil {
		t.Fatalf("SetCharacterRunes: %v", err)
	}
	validation, err := engine.ValidateReviewChanges(id, "1")
	if err != nil {
		t.Fatalf("ValidateReviewChanges: %v", err)
	}
	if !validation.Valid {
		t.Fatalf("validation = %+v, want a valid review", validation)
	}

	target := filepath.Join(t.TempDir(), "prepared.sl2")
	result, err := engine.ExportForDeployment(id, "1", validation.ValidationToken, false, false, target)
	if err != nil {
		t.Fatalf("ExportForDeployment: %v", err)
	}
	if result.SaveRevision != "1" || result.Target != target || result.Platform != string(PlatformPC) {
		t.Fatalf("result = %+v", result)
	}
	written, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read the prepared file: %v", err)
	}
	if len(written) == 0 {
		t.Fatal("the prepared file is empty")
	}
	// The prepared bytes are the ones a reload accepts, which is what the
	// endpoint's final verification already proved; asserting it here keeps the
	// contract stated at the engine boundary too.
	if err := validateSerialized(written, PlatformPC); err != nil {
		t.Fatalf("the prepared file failed reload validation: %v", err)
	}

	info, err := engine.GetSessionInfo(id)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if !info.UnsavedChanges {
		t.Fatal("the session was marked clean by a deployment export")
	}
	if info.SaveRevision != "1" {
		t.Fatalf("saveRevision after the export = %q, want it unchanged", info.SaveRevision)
	}
	if info.SourcePath != source || info.SourceKind != string(SourceKindLocal) {
		t.Fatalf("the session source moved to the deployment file: %+v", info)
	}
	history, err := engine.GetOperationHistory(id)
	if err != nil {
		t.Fatalf("GetOperationHistory: %v", err)
	}
	if history.UndoCount != 1 || len(history.Operations) != 1 {
		t.Fatalf("the operation history was cleared by a deployment export: %+v", history)
	}
	// The user's own local file was not touched.
	current, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read the local source: %v", err)
	}
	if len(current) == len(written) && string(current) == string(written) {
		t.Fatal("the local source file was replaced by the deployment export")
	}
}

// TestExportForDeploymentRefusesAnUnauthorizedOrStaleRequest states the other
// half of the contract: the preparation is exactly as gated as a Save, so no
// deployment can bypass Review Changes or send a revision the caller did not
// validate.
func TestExportForDeploymentRefusesAnUnauthorizedOrStaleRequest(t *testing.T) {
	source, _ := writeUndoFixture(t, PlatformPC)
	engine := NewWithOptions(EngineOptions{StateDirectory: t.TempDir()})
	loaded, err := engine.LoadSave(source, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	id := loaded.SaveSessionID
	directory := t.TempDir()

	if _, err := engine.ExportForDeployment(
		id, "0", "", false, false, filepath.Join(directory, "a.sl2")); err == nil {
		t.Fatal("ExportForDeployment accepted a request with no validation token")
	}
	validation, err := engine.ValidateReviewChanges(id, "0")
	if err != nil {
		t.Fatalf("ValidateReviewChanges: %v", err)
	}
	if _, err := engine.ExportForDeployment(
		id, "1", validation.ValidationToken, false, false,
		filepath.Join(directory, "b.sl2")); err == nil {
		t.Fatal("ExportForDeployment accepted a stale expected revision")
	}
	for _, name := range []string{"a.sl2", "b.sl2"} {
		if _, err := os.Stat(filepath.Join(directory, name)); err == nil {
			t.Fatalf("a refused export still wrote %s", name)
		}
	}
}
