# SetDiagnosticMode

## Overview

`SetDiagnosticMode` turns the extended diagnostics of one running application
instance on or off.

Debug Mode is global instance state, not save state and not a persistent
setting. It starts disabled on every launch, is never written to disk, and
covers the current session, every later session and every operation performed
without an open save. Disabling it stops new `debug` records; it deletes
nothing that was already recorded, and `info`, `warning` and `error` records
are produced regardless of the flag.

The single owner is the diagnostic service the composition root builds and
injects. There is no package-level singleton, no second flag in the host
settings store and no authoritative copy in the frontend.

Although the contract is a `Mutation`, it mutates application runtime state
only. It needs no `saveSessionID` and no `expectedRevision`, it advances no
`saveRevision`, it changes no dirty flag, operation history, Undo/Redo stack or
recovery journal, and it produces neither a `MutationReceipt` nor a
`session.changed` event.

| | |
|---|---|
| EndpointID | `set_diagnostic_mode` |
| Kind | Mutation |
| Domain | `diagnostics` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; the flag governs one running desktop instance, so it is deliberately not an HTTP route of the local explorer and is absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/diagnostics](../../../backend/endpoints/diagnostics) |
| Save access | none |

## Input

```go
func SetDiagnosticMode(
	service *diagnostics.Service, enabled bool,
) (DiagnosticModeResult, error)
```

## Output

```go
type DiagnosticModeResult = diagnostics.State

type State struct {
	Enabled               bool `json:"enabled"`
	LogDirectoryExists    bool `json:"logDirectoryExists"`
	LocalLoggingAvailable bool `json:"localLoggingAvailable"`
	DroppedRecords        int  `json:"droppedRecords"`
}
```

The result is the state actually in effect, so the interface never renders a
value the backend did not confirm.

`localLoggingAvailable` is false once the local sink has failed. That is a
reported state and never an error of the operation that produced the record:
the in-memory buffer keeps working and no edit, save or deployment result
changes because the file could not be written. `droppedRecords` counts what
never reached the file — rejected entries, a full queue and oversized records —
so a failing logger is described in state rather than in another record.

## Errors

| Condition | Result |
|---|---|
| the diagnostic service is not wired | `the diagnostic service is not available` |
