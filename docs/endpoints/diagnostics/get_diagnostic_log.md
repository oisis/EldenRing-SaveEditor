# GetDiagnosticLog

## Overview

`GetDiagnosticLog` returns a safe, structured portion of the current session's
in-memory diagnostic log.

It is read-only and non-mutating: calling this getter modifies neither the save,
nor the session, nor the log buffer, nor the application state. It reports safe,
pre-vetted diagnostic events emitted by SaveEngine operations and never leaks
private file paths, Steam or PSN IDs, user data, or raw save bytes.

The diagnostic log is strictly in-memory and bound to the active session. It is
never serialized or persisted into the save file binary.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetDiagnosticLog` never creates a
session, so calling it with an empty or unknown identifier returns an error.

| | |
|---|---|
| EndpointID | `get_diagnostic_log` |
| Kind | Getter |
| Domain | `diagnostics` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/diagnostic-log` of the local explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. There is no Wails binding, no CLI command and no frontend. |
| Implementation source | [../../../backend/endpoints/diagnostics/get_diagnostic_log.go](../../../backend/endpoints/diagnostics/get_diagnostic_log.go) |
| Test source | [../../../backend/endpoints/diagnostics/get_diagnostic_log_test.go](../../../backend/endpoints/diagnostics/get_diagnostic_log_test.go) |
| Save access | read-only — the session's private ring buffer, read under `engine.mutex`; no file is opened |
| Catalog access | none |
| Mutation | none — the snapshot, session, journal buffer and save file are left unchanged |

## Input

```go
func GetDiagnosticLog(
	engine *saveengine.Engine,
	saveSessionID string,
	cursor string,
	limit int,
	severity string,
	scope string,
) (GetDiagnosticLogResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the caller. A `nil` engine is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. |
| `cursor` | `string` | Exclusive sequence number token. Must be a canonical decimal uint64 (pattern `^(0|[1-9][0-9]*)$`) or empty string `""` to read from the oldest available record. |
| `limit` | `int` | Maximum records to return. `0` defaults to `50`; accepted range is `0..200`. |
| `severity` | `string` | One of `""`, `info`, `warning`, `error`. `warning` and `error` return empty matches in v1. |
| `scope` | `string` | One of `""`, `session`, `repairs`. |

### `cursor` and Pagination

- Matched as an exclusive decimal sequence number (`seq > cursor`).
- Must be a canonical decimal `uint64` string (e.g. `"0"`, `"1"`, `"42"`) without leading zeros or plus signs (non-canonical formats like `"01"`, `"+1"`, `" 1"` are rejected).
- When `cursor` is `""`, reading begins at the oldest available record in the buffer.
- The ring buffer retains up to 500 entries per session. When entries are overwritten on rollover, a non-empty `cursor` pointing before the retained window (`cursor < oldestSeq - 1`) causes `cursorExpired` to be set to `true`, and pagination transparently restarts at the oldest retained record.

### `limit`

- `0` or omitted defaults to `50`. Values outside `0..200` (or negative) are rejected with an error.

### `severity` and `scope`

- Accepted values for `severity` are `""`, `"info"`, `"warning"`, `"error"`. In v1, only `"info"` events are emitted; `"warning"` and `"error"` return 0 records and `hasMore: false`.
- Accepted values for `scope` are `""`, `"session"`, `"repairs"`.
- Unknown severity or scope values are rejected with a validation error.

## Output

```go
type DiagnosticRecord struct {
	Seq         uint64 `json:"seq"`
	Timestamp   string `json:"timestamp"`
	Severity    string `json:"severity"`
	Scope       string `json:"scope"`
	Event       string `json:"event"`
	Message     string `json:"message"`
	CharacterID *int   `json:"characterID,omitempty"`
	Revision    string `json:"revision,omitempty"`
}

type GetDiagnosticLogResult struct {
	SaveSessionID         string             `json:"saveSessionID"`
	Records               []DiagnosticRecord `json:"records"`
	NextCursor            string             `json:"nextCursor"`
	HasMore               bool               `json:"hasMore"`
	TotalBuffered         int                `json:"totalBuffered"`
	CursorExpired         bool               `json:"cursorExpired"`
	OldestAvailableCursor string             `json:"oldestAvailableCursor"`
}
```

### Result Fields

| Field | Type | Description |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the queried save session. |
| `records` | `[]DiagnosticRecord` | Slice of matching diagnostic records in chronological order. |
| `nextCursor` | `string` | Canonical decimal sequence string of the last record returned in this page, even on the final page (`hasMore: false`). If no records match, it equals the input `cursor` (or the fallback boundary if cursor expired or buffer was empty). |
| `hasMore` | `bool` | `true` if additional records matching the query filters exist in the buffer beyond this page; `false` otherwise. |
| `totalBuffered` | `int` | Total count of diagnostic records currently stored in the session's 500-entry ring buffer (0..500). |
| `cursorExpired` | `bool` | `true` if the requested `cursor` fell behind the retained window due to ring buffer rollover; `false` otherwise. |
| `oldestAvailableCursor` | `string` | Sequence token of the oldest available record (`oldestSeq`), or `""` if the buffer is empty. |

## V1 Diagnostic Emitters

SaveEngine v1 emits exactly three fixed, pre-vetted events:

| Event | Scope | Severity | Message | CharacterID | Revision | Trigger |
|---|---|---|---|---|---|---|
| `session_loaded` | `session` | `info` | `save session loaded and validated` | `nil` | `"0"` | Emitted upon successful `LoadSave`. |
| `save_written` | `session` | `info` | `save snapshot written and verified` | `nil` | new revision (e.g. `"1"`) | Emitted upon successful `WriteSave`. |
| `repairs_applied` | `repairs` | `info` | `repair plan actions executed` | `*int` (character slot) | new revision (e.g. `"1"`) | Emitted upon successful `ApplyRepairPlan` with 1+ executed actions (`Applied=true`). |
