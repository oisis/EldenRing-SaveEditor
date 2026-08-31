package savesession

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestWriteSaveDelegatesToSaveEngine(t *testing.T) {
	engine := saveengine.New()
	session, err := engine.LoadSave(writePCFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	target := filepath.Join(t.TempDir(), "written.sl2")

	result, err := WriteSave(engine, session.SaveSessionID, "0", target)
	if err != nil {
		t.Fatalf("WriteSave: %v", err)
	}
	want := WriteSaveResult{SaveSessionID: session.SaveSessionID, SaveRevision: "1"}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("WriteSave result = %+v, want %+v", result, want)
	}
	if _, err := saveengine.New().LoadSave(target, "pc", "local"); err != nil {
		t.Fatalf("reload written target: %v", err)
	}
}

func TestWriteSaveRejectsMissingEngineAndForwardsErrors(t *testing.T) {
	if result, err := WriteSave(nil, "session", "0", "target"); err == nil ||
		err.Error() != "save engine is not available" || result != (WriteSaveResult{}) {
		t.Fatalf("nil-engine result = %+v, error = %v", result, err)
	}

	engine := saveengine.New()
	result, err := WriteSave(engine, "missing", "00", "target")
	if err == nil || !strings.Contains(err.Error(), "canonical decimal saveRevision") {
		t.Fatalf("forwarded error = %v", err)
	}
	if result != (WriteSaveResult{}) {
		t.Errorf("rejected result = %+v, want zero value", result)
	}
}

func TestWriteSaveContractMatchesRuntime(t *testing.T) {
	if WriteSaveDefinition.SupportedResourceTypes != "—" {
		t.Errorf("SupportedResourceTypes = %q, want no catalog resource dependency",
			WriteSaveDefinition.SupportedResourceTypes)
	}
	wantVariables := []string{"saveSessionID", "expectedRevision", "target"}
	if !reflect.DeepEqual(WriteSaveDefinition.SupportedResourceVariables, wantVariables) {
		t.Errorf("SupportedResourceVariables = %v, want %v",
			WriteSaveDefinition.SupportedResourceVariables, wantVariables)
	}
}
