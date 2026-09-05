package desktop

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
	diagnosticsservice "github.com/oisis/EldenRing-SaveForge/backend/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/hostsettings"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// The Debug Mode toggle must reach the one diagnostic service and come back
// as the state actually in effect, and it must leave the save side alone: it
// is a runtime application setting, not a save mutation.
func TestSetDiagnosticModeRoundTripsAndTouchesNoSaveState(t *testing.T) {
	logs := filepath.Join(t.TempDir(), "logs")
	engine := saveengine.New()
	bridge := NewBridgeWithDependencies(Dependencies{
		ApplicationVersion: "2.0.0",
		SaveEngine:         engine,
		HostSettings:       hostsettings.NewStore(t.TempDir()),
		Diagnostics:        diagnosticsservice.NewService(diagnosticsservice.Options{Directory: logs}),
	})
	bridge.Startup(context.Background())
	t.Cleanup(bridge.diagnostics.Close)

	initial, err := bridge.GetDiagnosticMode()
	if err != nil {
		t.Fatalf("GetDiagnosticMode: %v", err)
	}
	if initial.Enabled {
		t.Fatalf("a fresh instance reported Debug Mode as enabled")
	}

	enabled, err := bridge.SetDiagnosticMode(true)
	if err != nil {
		t.Fatalf("SetDiagnosticMode(true): %v", err)
	}
	if !enabled.Enabled {
		t.Fatalf("SetDiagnosticMode(true) reported enabled = false")
	}
	if read, err := bridge.GetDiagnosticMode(); err != nil || !read.Enabled {
		t.Fatalf("GetDiagnosticMode after enabling = %+v, %v", read, err)
	}

	// The engine is the only owner of save state, and the toggle never reaches
	// it: no session appears and nothing becomes dirty.
	if engine.HasUnsavedChanges() {
		t.Errorf("changing Debug Mode marked save state as changed")
	}
	if _, err := engine.GetDiagnosticLog("", "", 0, "", ""); err == nil {
		t.Errorf("changing Debug Mode created a save session")
	}

	if disabled, err := bridge.SetDiagnosticMode(false); err != nil || disabled.Enabled {
		t.Fatalf("SetDiagnosticMode(false) = %+v, %v", disabled, err)
	}
}

func TestDiagnosticStagesUseConfirmedResultsAndPreserveUnknownReplacement(t *testing.T) {
	service := diagnosticsservice.NewService(diagnosticsservice.Options{})
	bridge := NewBridgeWithDependencies(Dependencies{Diagnostics: service})
	service.SetDebugMode(true)
	logger := bridge.beginOperation(diagnosticsservice.OperationDeployToTarget, "")
	logger.finishOperationResult(deployment.OperationResult{
		TargetState: deployment.TargetStateReplacementUndetermined,
		Failure:     deployment.FailureReplacementUndetermined,
		Stages: []deployment.Stage{
			{Stage: deployment.StageBackup, Completed: true},
			{Stage: deployment.StageReplace, Completed: false},
		},
	}, nil)
	page, err := service.Records("", 50, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Records) != 4 {
		t.Fatalf("records: %+v", page.Records)
	}
	stage, finish := page.Records[2], page.Records[3]
	if stage.Stage != deployment.StageBackup || finish.Severity != diagnosticsservice.SeverityError ||
		finish.TargetState != deployment.TargetStateReplacementUndetermined ||
		finish.Code != deployment.FailureReplacementUndetermined {
		t.Fatalf("stage: %+v, finish: %+v", stage, finish)
	}
}

// The console reads the instance-wide stream, which must work before any save
// is opened and must not repeat a record the caller already holds.
func TestGetDiagnosticEventsWorksWithoutASaveAndReadsIncrementally(t *testing.T) {
	bridge := NewBridgeWithDependencies(Dependencies{
		ApplicationVersion: "2.0.0",
		Diagnostics:        diagnosticsservice.NewService(diagnosticsservice.Options{}),
	})
	bridge.Startup(context.Background())

	if _, err := bridge.SetDiagnosticMode(true); err != nil {
		t.Fatalf("SetDiagnosticMode: %v", err)
	}
	first, err := bridge.GetDiagnosticEvents("", 50, "")
	if err != nil {
		t.Fatalf("GetDiagnosticEvents: %v", err)
	}
	if len(first.Records) != 1 {
		t.Fatalf("first page = %d records, want the mode change alone", len(first.Records))
	}

	logger := bridge.beginOperation(diagnosticsservice.OperationOpenSave, "")
	logger.finishWithError(nil)

	next, err := bridge.GetDiagnosticEvents(first.NextCursor, 50, "")
	if err != nil {
		t.Fatalf("GetDiagnosticEvents: %v", err)
	}
	if len(next.Records) != 2 {
		t.Fatalf("second page = %d records, want the started and finished pair", len(next.Records))
	}
	for _, record := range next.Records {
		if record.Operation != diagnosticsservice.OperationOpenSave {
			t.Errorf("record operation = %q, want %q",
				record.Operation, diagnosticsservice.OperationOpenSave)
		}
	}
}

// The log directory is opened by identifier from the diagnostic service's own
// location. A host without one still refuses the identifier.
func TestOpenHostLocationOpensTheDiagnosticServiceLogDirectory(t *testing.T) {
	logs := filepath.Join(t.TempDir(), "logs")
	opened := []string{}
	bridge := NewBridgeWithDependencies(Dependencies{
		ApplicationVersion: "2.0.0",
		HostSettings:       hostsettings.NewStore(t.TempDir()),
		Diagnostics:        diagnosticsservice.NewService(diagnosticsservice.Options{Directory: logs}),
	})
	bridge.openHostURL = func(_ context.Context, url string) { opened = append(opened, url) }
	bridge.Startup(context.Background())
	t.Cleanup(bridge.diagnostics.Close)

	if err := bridge.OpenHostLocation("logs"); err != nil {
		t.Fatalf("OpenHostLocation(logs): %v", err)
	}
	if len(opened) != 1 || opened[0] != "file://"+logs {
		t.Fatalf("opened = %v, want the diagnostic log directory", opened)
	}
	if err := bridge.OpenHostLocation("/etc"); err == nil {
		t.Fatalf("an arbitrary path was accepted as a host location")
	}
}
