# GetOperationHistory

Returns the backend-owned ordered operation journal for one loaded save.

| | |
|---|---|
| EndpointID | `get_operation_history` |
| Kind | Getter |
| Status | implemented; Wails `GetOperationHistory` |
| Source | [get_operation_history.go](../../../backend/endpoints/savesession/get_operation_history.go) |

Input is the exact `saveSessionID`. The result contains the current
`saveRevision`, safe operation projections, and the global Undo/Redo counts.
Replay patches and save bytes never leave SaveEngine.

Every operation carries its stable ID and order, character when applicable,
area, description, before/after summary, risk, reason, changed byte count and
the exact shared `changedScopes`.
