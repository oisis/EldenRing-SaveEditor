/*
Endpoint: ExportDiagnosticReport
EndpointID: export_diagnostic_report
Purpose: Writes a redacted diagnostic report to a host path the user chose in the native Save As dialog.
How it works: The runtime handler assembles the application version, the supported schema range, the declared capabilities, the host settings flags, the diagnostic mode state, a bounded slice of the instance-wide diagnostic records and, when a session is open, that session's structured diagnostic records. It writes the document atomically to the stated target and returns the number of records it carried.
Supported resource types: —.
Input variables: saveSessionID, target.
GameCatalog variables read: only the MinimumSchemaVersion and CurrentSchemaVersion constants.
Save variables read: none. The report carries no save bytes, no item data and no source path; only structured diagnostic records, which the engine and the diagnostic service produce from closed catalogues without private data. No configuration document and no file from the log directory is ever copied into it.
Implementation status: implemented
*/
package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/oisis/EldenRing-SaveForge/backend/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/hostsettings"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// ExportDiagnosticReportEndpointID is the stable backend identifier of
// ExportDiagnosticReport.
const ExportDiagnosticReportEndpointID = "export_diagnostic_report"

// diagnosticReportRecordLimit caps how many structured records one report
// carries. The journal is a ring buffer, so this is a report size bound rather
// than a filter: it exists so a long session cannot produce an unbounded file.
const diagnosticReportRecordLimit = 2000

// diagnosticReportEventLimit bounds the instance-wide slice the report carries.
// It selects the newest 200 records from the 500-record buffer.
const diagnosticReportEventLimit = 200

// ExportDiagnosticReportDefinition describes the public getter contract. The
// endpoint writes a file the user explicitly asked for and changes no
// application state, so it stays a getter of the diagnostic state rather than a
// mutation of it.
var ExportDiagnosticReportDefinition = contract.MustDefine(contract.Definition{
	Name:                       "ExportDiagnosticReport",
	ID:                         ExportDiagnosticReportEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "target"},
	Description:                "Writes a redacted diagnostic report to a host path the user chose in the native Save As dialog.",
})

// DiagnosticReportResult reports what was written, without repeating the target
// path back to the frontend: the path is the user's own private host location
// and the frontend already knows the one it passed in.
type DiagnosticReportResult struct {
	Exported    bool `json:"exported"`
	RecordCount int  `json:"recordCount"`
	// EventCount is the number of instance-wide diagnostic records the report
	// carried, reported separately from the session's own journal records.
	EventCount int `json:"eventCount"`
}

// diagnosticReport is the exact document shape written to disk. Every field is
// either a build constant, a boolean setting or an engine-produced structured
// record. Nothing here can carry a save byte, an SSH key path, a token or an
// arbitrary file's contents.
type diagnosticReport struct {
	Report             string                        `json:"report"`
	GeneratedAt        string                        `json:"generatedAt"`
	ApplicationVersion string                        `json:"applicationVersion"`
	Platform           string                        `json:"platform"`
	Architecture       string                        `json:"architecture"`
	SupportedSchemas   []SupportedSchema             `json:"supportedSchemas"`
	Capabilities       []string                      `json:"capabilities"`
	Settings           diagnosticReportSettings      `json:"settings"`
	Diagnostics        diagnostics.State             `json:"diagnostics"`
	Events             []diagnostics.Record          `json:"events"`
	Session            *diagnosticReportSessionState `json:"session,omitempty"`
	Records            []saveengine.DiagnosticRecord `json:"records"`
}

type diagnosticReportSettings struct {
	SkipReviewForNormalRisk bool   `json:"skipReviewForNormalRisk"`
	RemoteBackupPolicy      string `json:"remoteBackupPolicy"`
}

// diagnosticReportSessionState names the open session without describing where
// it came from: the platform and format are build facts and the revision is an
// opaque counter, while the source path is deliberately absent.
type diagnosticReportSessionState struct {
	SaveSessionID  string `json:"saveSessionID"`
	Platform       string `json:"platform"`
	Format         string `json:"format"`
	SaveRevision   string `json:"saveRevision"`
	UnsavedChanges bool   `json:"unsavedChanges"`
}

// ExportDiagnosticReport writes the redacted report to target.
//
// saveSessionID is optional: with an empty value the report describes the host
// only. target is the path the host's own Save As dialog returned; a cancelled
// dialog never reaches this endpoint, because the bridge treats the empty path
// as an ordinary outcome and calls nothing.
func ExportDiagnosticReport(
	applicationVersion string,
	settingsStore *hostsettings.Store,
	diagnosticService *diagnostics.Service,
	engine *saveengine.Engine,
	saveSessionID string,
	target string,
) (DiagnosticReportResult, error) {
	if applicationVersion == "" {
		return DiagnosticReportResult{}, errors.New("application version is required")
	}
	if target == "" {
		return DiagnosticReportResult{}, errors.New("a diagnostic report target is required")
	}
	settings := hostsettings.Defaults()
	if settingsStore != nil {
		stored, err := settingsStore.Get()
		if err != nil {
			return DiagnosticReportResult{}, err
		}
		settings = stored
	}

	report := diagnosticReport{
		Report:             "saveforge.diagnostic-report",
		GeneratedAt:        time.Now().UTC().Format(time.RFC3339),
		ApplicationVersion: applicationVersion,
		Platform:           runtime.GOOS,
		Architecture:       runtime.GOARCH,
		SupportedSchemas: []SupportedSchema{{
			Name:           gameCatalogSchemaName,
			MinimumVersion: schema.MinimumSchemaVersion,
			CurrentVersion: schema.CurrentSchemaVersion,
		}},
		Capabilities: []string{catalogReadCapability},
		Settings: diagnosticReportSettings{
			SkipReviewForNormalRisk: settings.SkipReviewForNormalRisk,
			RemoteBackupPolicy:      string(settings.RemoteBackupPolicy),
		},
		Records:     []saveengine.DiagnosticRecord{},
		Events:      []diagnostics.Record{},
		Diagnostics: diagnosticService.State(),
	}

	// The instance-wide records go through exactly the same reader the console
	// uses, so the report cannot show a record the console could not, and the
	// slice is bounded by the same record limit as the session journal.
	if diagnosticService != nil {
		report.Events = diagnosticService.RecentRecords(diagnosticReportEventLimit)
	}

	if saveSessionID != "" && engine != nil {
		info, err := engine.GetSessionInfo(saveSessionID)
		if err != nil {
			return DiagnosticReportResult{}, err
		}
		report.Session = &diagnosticReportSessionState{
			SaveSessionID:  info.SaveSessionID,
			Platform:       info.Platform,
			Format:         info.Format,
			SaveRevision:   info.SaveRevision,
			UnsavedChanges: info.UnsavedChanges,
		}
		log, err := engine.GetDiagnosticLog(saveSessionID, "", diagnosticReportRecordLimit, "", "")
		if err != nil {
			return DiagnosticReportResult{}, err
		}
		if log.Records != nil {
			report.Records = log.Records
		}
	}

	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return DiagnosticReportResult{}, err
	}
	encoded = append(encoded, '\n')

	// The report is written through a sibling temporary file and renamed, so a
	// failed or interrupted export never leaves a half-written document at the
	// path the user chose.
	temporary := target + ".tmp"
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return DiagnosticReportResult{}, fmt.Errorf("cannot create the report directory: %w", err)
	}
	if err := os.WriteFile(temporary, encoded, 0o600); err != nil {
		return DiagnosticReportResult{}, fmt.Errorf("cannot write the diagnostic report: %w", err)
	}
	if err := os.Rename(temporary, target); err != nil {
		_ = os.Remove(temporary)
		return DiagnosticReportResult{}, fmt.Errorf("cannot store the diagnostic report: %w", err)
	}
	return DiagnosticReportResult{
		Exported:    true,
		RecordCount: len(report.Records),
		EventCount:  len(report.Events),
	}, nil
}
