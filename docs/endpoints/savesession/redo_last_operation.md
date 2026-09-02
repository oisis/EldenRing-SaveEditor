# RedoLastOperation

Atomically reapplies the newest entry from the global Redo stack.

| | |
|---|---|
| EndpointID | `redo_last_operation` |
| Kind | Mutation |
| Status | implemented; Wails `RedoLastOperation` |
| Source | [redo_last_operation.go](../../../backend/endpoints/savesession/redo_last_operation.go) |

Input is `saveSessionID` and `expectedRevision`. Replay requires the exact
stored preimage and a structurally valid candidate. Success returns the shared
mutation receipt and the affected operation identity; any later ordinary
mutation clears Redo.
