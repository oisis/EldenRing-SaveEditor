# ExportDiagnosticReport

## Overview

`ExportDiagnosticReport` writes a redacted diagnostic report to a host path the
user chose in the native Save As dialog.

The document carries the application version, the platform and architecture, the
supported schema range, the declared capabilities, the two host settings, the
diagnostic mode state, a bounded slice of the instance-wide diagnostic records
and, when a session is named, that session's structured diagnostic records. It
carries no save bytes, no item data, no source path, no SSH key path and no
arbitrary file content.

Both record sets pass the same sanitisation boundary as the console: they are
produced from the closed event catalogues, never composed from caller text. The
report deliberately carries neither the whole application configuration nor any
file from the local log directory — it is a snapshot of safe records, not a
bundle of the host's files.

The report is written through a sibling temporary file and renamed, so a failed
or interrupted export never leaves a half-written document behind. Cancelling
the dialog never reaches this endpoint: the bridge treats the empty path as an
ordinary outcome and calls nothing.

| | |
|---|---|
| EndpointID | `export_diagnostic_report` |
| Kind | Getter |
| Domain | `application` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/application](../../../backend/endpoints/application) |
| Save access | none — the session's own diagnostic records and the instance-wide records only, both produced from closed catalogues without private data |

## Input

```go
func ExportDiagnosticReport(
	applicationVersion string,
	settingsStore *hostsettings.Store,
	diagnosticService *diagnostics.Service,
	engine *saveengine.Engine,
	saveSessionID string,
	target string,
) (DiagnosticReportResult, error)
```

## Output

```go
type DiagnosticReportResult struct {
	Exported    bool `json:"exported"`
	RecordCount int  `json:"recordCount"`
	EventCount  int  `json:"eventCount"`
}
```

`recordCount` counts the named session's journal records and `eventCount` the
instance-wide records, reported separately so neither set is read as the other.

## Errors

| Condition | Result |
|---|---|
| `applicationVersion` is empty | `application version is required` — a backend wiring error |
| `target` is empty | `a diagnostic report target is required` |
| `saveSessionID` names no session | the session error of `GetSessionInfo`; nothing is written |
| the document cannot be written or renamed | `cannot write the diagnostic report` / `cannot store the diagnostic report` |

The chosen path is never repeated back to the caller: it is the user's own
private location and the caller already knows the path it supplied.
