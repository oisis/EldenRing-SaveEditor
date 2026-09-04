# ExportDiagnosticReport

## Overview

`ExportDiagnosticReport` writes a redacted diagnostic report to a host path the
user chose in the native Save As dialog.

The document carries the application version, the platform and architecture, the
supported schema range, the declared capabilities, the two host settings and,
when a session is named, that session's structured diagnostic records. It
carries no save bytes, no item data, no source path, no SSH key path and no
arbitrary file content.

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
| Save access | none — the session's own diagnostic records only, which the engine produces without private data |

## Input

```go
func ExportDiagnosticReport(
	applicationVersion string,
	settingsStore *hostsettings.Store,
	engine *saveengine.Engine,
	saveSessionID string,
	target string,
) (DiagnosticReportResult, error)
```

## Output

```go
type DiagnosticReportResult struct {
	Exported    bool `json:"exported"`
	RecordCount int `json:"recordCount"`
}
```

## Errors

| Condition | Result |
|---|---|
| `applicationVersion` is empty | `application version is required` — a backend wiring error |
| `target` is empty | `a diagnostic report target is required` |
| `saveSessionID` names no session | the session error of `GetSessionInfo`; nothing is written |
| the document cannot be written or renamed | `cannot write the diagnostic report` / `cannot store the diagnostic report` |

The chosen path is never repeated back to the caller: it is the user's own
private location and the caller already knows the path it supplied.
