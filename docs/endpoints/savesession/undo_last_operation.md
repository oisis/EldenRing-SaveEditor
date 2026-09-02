# UndoLastOperation

Atomically reverses the newest journal entry and moves it to the Redo stack.

| | |
|---|---|
| EndpointID | `undo_last_operation` |
| Kind | Mutation |
| Status | implemented; Wails `UndoLastOperation` |
| Source | [undo_last_operation.go](../../../backend/endpoints/savesession/undo_last_operation.go) |

Input is `saveSessionID` plus the exact canonical `expectedRevision`. SaveEngine
checks the replay preimage, validates the candidate, persists recovery and only
then commits one new revision and `session.changed` event. The result embeds the
shared mutation receipt and identifies the affected operation.
