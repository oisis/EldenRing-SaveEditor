# GetDiagnosticMode

## Overview

`GetDiagnosticMode` reports whether extended diagnostics are enabled for this
running instance and whether local diagnostic logging works.

It is the getter half of the state `SetDiagnosticMode` writes, and it exists so
`Tools → Settings` can render the backend's own value instead of a local guess.
It reads no save, opens no session and mutates nothing.

| | |
|---|---|
| EndpointID | `get_diagnostic_mode` |
| Kind | Getter |
| Domain | `diagnostics` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; it describes one running desktop instance, so it is deliberately not an HTTP route of the local explorer and is absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/diagnostics](../../../backend/endpoints/diagnostics) |
| Save access | none |

## Input

```go
func GetDiagnosticMode(service *diagnostics.Service) (DiagnosticModeResult, error)
```

## Output

The `DiagnosticModeResult` of
[set_diagnostic_mode.md](set_diagnostic_mode.md), unchanged. The setter and the
getter share one type so the two can never describe the state differently.

## Errors

| Condition | Result |
|---|---|
| the diagnostic service is not wired | `the diagnostic service is not available` |
