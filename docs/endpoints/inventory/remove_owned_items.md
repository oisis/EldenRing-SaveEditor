# RemoveOwnedItems

## Overview

`RemoveOwnedItems` removes every named owned instance of one character as **one
atomic mutation of one save revision**. It creates no record, moves none, merges
none, reorders none and reindexes none: each removal clears exactly the twelve
bytes of the addressed record the way that record's own section is confirmed to
be cleared, through the same planner
[`RemoveOwnedItem`](remove_owned_item.md) uses.

A removal is addressed by identity, so **no GameCatalog is read and none is
required**, and the Safety Profile plays no part: nothing is added and no limit
applies.

| | |
|---|---|
| EndpointID | `remove_owned_items` |
| Kind | Mutation |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport status | desktop bridge only — bound as a Wails method on `desktop.Bridge` and reached by the SaveForge frontend. It is deliberately **not** registered as an HTTP route of the local explorer and therefore does not appear in `tools/swagger/openapi.json` or in the Scalar portal: an operation the explorer cannot perform must never be advertised there. |
| Implementation source | [../../../backend/endpoints/inventory/remove_owned_items.go](../../../backend/endpoints/inventory/remove_owned_items.go) |
| Save access | read-write on the session's private in-memory snapshot; no file is opened |
| Changed scopes | `inventory` and `storage`, exactly as `RemoveOwnedItem` reports them |

## Input

```go
func RemoveOwnedItems(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	ownedItemIDs []string,
	expectedRevision string,
) (RemoveOwnedItemsResult, error)
```

`saveSessionID`, every `ownedItemID` and `expectedRevision` are passed through
byte for byte.

## Output

`RemoveOwnedItemsResult` embeds the shared `MutationReceipt` flat and adds
`characterID` plus one `{ownedItemID, gameID}` entry per removal. Every echoed
identity is already stale twice over: the revision moved on, and the record it
addressed no longer exists.


## Atomicity

The batch is validated and applied against a **private candidate image** of the
session snapshot. Only after the last step succeeded does the candidate replace
the session's own snapshot, in one assignment. A step that fails therefore
leaves nothing behind: there is no partial result, no rollback of a half-written
batch and no per-item retry.

One successful batch produces:

- exactly one committed `saveRevision`;
- exactly one operation-history entry;
- exactly one `MutationReceipt` with one `operationID`;
- exactly one `session.changed` event.

It participates in Undo, Redo and the recovery journal like every other
save-session mutation, because it goes through the same central commit path.

A stale `expectedRevision` is refused before anything is read. The backend never
retries a revision conflict; the caller re-reads and confirms again.


## Validation

Rejected before the session is touched: an empty list, an empty entry, a
repeated `ownedItemID` and a non-canonical `expectedRevision`.

Rejected during planning, which discards the whole candidate: an unknown or
stale identity, an identity of another character, a Storage key record (no
confirmed write contract) and any record an Equipment, Quick Item or Pouch slot
references. One referenced record rejects the whole batch, and no earlier step of
it is kept.

## Local verification

```bash
go test -count=1 ./backend/saveengine ./backend/endpoints/inventory
```
