// Package diagnostics owns the runtime diagnostic state of one running
// application instance: the Debug Mode flag, the safe in-memory record buffer
// the console reads and the local JSONL sink.
//
// It is the single source of truth for all three. The composition root builds
// exactly one Service and injects it; nothing here is a package-level
// singleton, host settings never keep a second copy of the flag, and the
// frontend never owns the value it renders.
//
// Debug Mode is deliberately not persistent: every launch starts with it
// disabled. Turning it off stops new debug records; it never deletes what was
// already recorded, and info, warning and error records are written regardless
// of the flag.
package diagnostics

// Severity levels a record may carry. Only debug is gated by Debug Mode.
const (
	SeverityDebug   = "debug"
	SeverityInfo    = "info"
	SeverityWarning = "warning"
	SeverityError   = "error"
)

// The closed event catalog. A record is only ever produced from one of these
// identifiers, and its message text comes from the table below rather than
// from a caller-supplied string. That is what keeps a private path, a host
// name or an error text from reaching the buffer, the file or the report in
// the first place; redaction is not the protection here, absence is.
const (
	EventApplicationStarted     = "application_started"
	EventDiagnosticModeChanged  = "diagnostic_mode_changed"
	EventOperationStarted       = "operation_started"
	EventOperationFinished      = "operation_finished"
	EventOperationStageFinished = "operation_stage_finished"
)

// The closed operation catalog. These are the shared boundaries the desktop
// bridge reports on, never one identifier per endpoint.
const (
	OperationOpenSave           = "open_save"
	OperationSave               = "save"
	OperationSaveAs             = "save_as"
	OperationApplyRepairs       = "apply_repairs"
	OperationDeployToTarget     = "deploy_to_target"
	OperationDownloadFromTarget = "download_from_target"
	OperationActivateBackup     = "activate_target_backup"
)

// The closed outcome catalog of operation_finished.
const (
	StatusSucceeded = "succeeded"
	StatusBlocked   = "blocked"
	StatusCancelled = "cancelled"
	StatusFailed    = "failed"
)

// eventMessages is the complete set of messages this package can produce. A
// caller states an event, never a sentence.
var eventMessages = map[string]string{
	EventApplicationStarted:     "application started",
	EventDiagnosticModeChanged:  "diagnostic mode changed",
	EventOperationStarted:       "operation started",
	EventOperationFinished:      "operation finished",
	EventOperationStageFinished: "operation stage finished",
}

// eventSeverities fixes the severity of every event whose severity does not
// depend on an outcome. operation_finished is absent on purpose: its severity
// is derived from Status by severityFor.
var eventSeverities = map[string]string{
	EventApplicationStarted:     SeverityInfo,
	EventDiagnosticModeChanged:  SeverityInfo,
	EventOperationStarted:       SeverityDebug,
	EventOperationStageFinished: SeverityDebug,
}

var allowedOperations = map[string]bool{
	OperationOpenSave:           true,
	OperationSave:               true,
	OperationSaveAs:             true,
	OperationApplyRepairs:       true,
	OperationDeployToTarget:     true,
	OperationDownloadFromTarget: true,
	OperationActivateBackup:     true,
}

var allowedStatuses = map[string]bool{
	StatusSucceeded: true,
	StatusBlocked:   true,
	StatusCancelled: true,
	StatusFailed:    true,
}

// statusSeverities maps an outcome onto the severity operation_finished
// reports for it.
var statusSeverities = map[string]string{
	StatusSucceeded: SeverityInfo,
	StatusBlocked:   SeverityWarning,
	StatusCancelled: SeverityWarning,
	StatusFailed:    SeverityError,
}

var allowedSeverities = map[string]bool{
	SeverityDebug:   true,
	SeverityInfo:    true,
	SeverityWarning: true,
	SeverityError:   true,
}

// severityFor resolves the severity of one entry, or reports that the entry
// names no known event.
func severityFor(entry Entry) (string, bool) {
	if entry.Event == EventOperationFinished {
		severity, known := statusSeverities[entry.Status]
		return severity, known
	}
	severity, known := eventSeverities[entry.Event]
	return severity, known
}
