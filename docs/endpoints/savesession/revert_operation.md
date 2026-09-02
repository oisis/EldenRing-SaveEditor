# RevertOperation

Selectively removes one operation by rebuilding from the durable baseline and
replaying every retained entry in order.

| | |
|---|---|
| EndpointID | `revert_operation` |
| Kind | Mutation |
| Status | implemented; Wails `RevertOperation` |
| Source | [revert_operation.go](../../../backend/endpoints/savesession/revert_operation.go) |

Input is the exact `saveSessionID`, `operationID` and `expectedRevision`. A
retained operation whose preimage depends on the removed entry makes the whole
request fail without changing the snapshot, history, recovery journal or
revision. Success clears Redo and returns the shared receipt.
