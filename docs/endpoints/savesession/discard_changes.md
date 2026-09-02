# DiscardChanges

Restores the last loaded or successfully saved baseline and clears operation
history, Undo, Redo and the recovery journal.

| | |
|---|---|
| EndpointID | `discard_changes` |
| Kind | Mutation |
| Status | implemented; Wails `DiscardChanges` |
| Source | [discard_changes.go](../../../backend/endpoints/savesession/discard_changes.go) |

Input is `saveSessionID` and `expectedRevision`. The baseline is validated
before commit. Success returns the shared receipt and `discardedOperations`;
the session becomes clean while its monotonic revision still advances.
