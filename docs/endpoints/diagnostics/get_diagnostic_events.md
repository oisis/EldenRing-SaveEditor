# GetDiagnosticEvents

## Overview

`GetDiagnosticEvents` returns a safe portion of the instance-wide diagnostic
event stream: the records the bottom console renders.

It is deliberately not the same reader as
[get_diagnostic_log.md](get_diagnostic_log.md). That endpoint returns the
private journal of one save session; this one returns the global stream of the
running instance, so the console works before any save has been opened and
keeps working across sessions. Both are non-mutating and consume nothing.

The records come from a closed event catalogue. A caller states an event
identifier, never a sentence, and every other field is a closed identifier, a
generated correlation value or a number. An entry naming anything outside the
catalogue is rejected in full rather than sanitised, which is why no save byte,
path, host, account, command, token or raw error text can appear here.

| | |
|---|---|
| EndpointID | `get_diagnostic_events` |
| Kind | Getter |
| Domain | `diagnostics` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; the stream belongs to one running desktop instance, so it is deliberately not an HTTP route of the local explorer and is absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/diagnostics](../../../backend/endpoints/diagnostics) |
| Save access | none |

## Input

```go
func GetDiagnosticEvents(
	service *diagnostics.Service, cursor string, limit int, severity string,
) (GetDiagnosticEventsResult, error)
```

`cursor` is the sequence number of the last record the caller already holds, so
an incremental reader never receives the same record twice. An empty cursor
starts from the oldest retained record. `limit` defaults to 50 and accepts
1..200. `severity` filters on `debug`, `info`, `warning` or `error`, or is empty
for every record.

## Output

```go
type GetDiagnosticEventsResult = diagnostics.Page

type Page struct {
	Records               []Record `json:"records"`
	NextCursor            string   `json:"nextCursor"`
	HasMore               bool     `json:"hasMore"`
	TotalBuffered         int      `json:"totalBuffered"`
	CursorExpired         bool     `json:"cursorExpired"`
	OldestAvailableCursor string   `json:"oldestAvailableCursor"`
}

type Record struct {
	Seq           uint64 `json:"seq"`
	Timestamp     string `json:"timestamp"`
	Severity      string `json:"severity"`
	Event         string `json:"event"`
	Message       string `json:"message"`
	Operation     string `json:"operation,omitempty"`
	Stage         string `json:"stage,omitempty"`
	Status        string `json:"status,omitempty"`
	CorrelationID string `json:"correlationID,omitempty"`
	Code          string `json:"code,omitempty"`
	TargetState   string `json:"targetState,omitempty"`
	DurationMS    int64  `json:"durationMS,omitempty"`
	Count         int    `json:"count,omitempty"`
	Enabled       *bool  `json:"enabled,omitempty"`
	Version       string `json:"version,omitempty"`
	Build         string `json:"build,omitempty"`
	Platform      string `json:"platform,omitempty"`
}
```

The buffer holds the 500 newest records. `cursorExpired` states that the
records after the requested cursor were already evicted, so a caller that fell
behind restarts from `nextCursor` instead of silently rendering a gap.

The event catalogue is closed:

| Event | Severity | Carries |
|---|---|---|
| `application_started` | info | the application version, build identity and platform |
| `diagnostic_mode_changed` | info | `enabled` |
| `operation_started` | debug | the operation and its correlation value |
| `operation_finished` | info on success, warning on a block or a cancellation, error on a failure | the operation, the status, a closed reason code, the real deployment target state and the elapsed time |
| `operation_stage_finished` | debug | the stage identifier and the elapsed time |

Operation records are emitted at the shared boundaries only: opening a save,
`Save` and `Save As`, `ApplyRepairs`, deploy, download and backup activation.
Stage records are emitted only for stages confirmed completed in the returned
operation result. Durations measure intervals between progress notifications;
when no interval was measured, no duration is reported. An announced but failed
stage is not logged as completed. No endpoint or writer owns an extra emitter.

## Errors

| Condition | Result |
|---|---|
| the diagnostic service is not wired | `the diagnostic service is not available` |
| `limit` is outside 1..200 | `limit %d is outside the range 1..200` |
| `severity` names an unknown level | `unknown severity %q` |
| `cursor` is not a canonical decimal uint64 | `cursor must be a canonical decimal uint64; got %q` |
