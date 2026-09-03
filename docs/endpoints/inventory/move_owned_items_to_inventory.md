# MoveOwnedItemsToInventory

## Overview

`MoveOwnedItemsToInventory` moves every named Storage common record into
Inventory common as **one atomic mutation of one save revision**. Each record is
appended behind the records already there, in the order the caller listed them.

It never merges, rehandles, allocates or repacks: each move relocates one
complete physical record through the same writer the single-record
[`MoveOwnedItemToInventory`](move_owned_item_to_inventory.md) endpoint uses.

| | |
|---|---|
| EndpointID | `move_owned_items_to_inventory` |
| Kind | Mutation |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport status | desktop bridge only — bound as a Wails method on `desktop.Bridge` and reached by the SaveForge frontend. It is deliberately **not** registered as an HTTP route of the local explorer and therefore does not appear in `tools/swagger/openapi.json` or in the Scalar portal: an operation the explorer cannot perform must never be advertised there. |
| Implementation source | [../../../backend/endpoints/inventory/move_owned_items_to_inventory.go](../../../backend/endpoints/inventory/move_owned_items_to_inventory.go) |
| Save access | read-write on the session's private in-memory snapshot; no file is opened |
| Changed scopes | `inventory` and `storage`, exactly as the single-record move reports them |

## Input

```go
func MoveOwnedItemsToInventory(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	safetyProfile string,
	saveSessionID string,
	characterID int,
	ownedItemIDs []string,
	expectedRevision string,
) (MoveOwnedItemsToInventoryResult, error)
```

Every `ownedItemID` is read back through SaveEngine, its one save-side game ID is
resolved through GameCatalog, and the destination limit is derived from the
shared Safety Profile policy — the Inventory limit of that item under the active
profile. A record whose limit or record mode the catalog does not state rejects
the whole batch.


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

Rejected before the session is touched: an empty list, a repeated
`ownedItemID`, an identity outside the required source container or section, a
key record (no confirmed write contract), an item without a usable destination
limit, an unsupported record mode and a non-canonical `expectedRevision`.

Rejected during planning, which discards the whole candidate: a record that no
longer denotes the expected item, a record an Equipment, Quick Item or Pouch slot
references, a destination with no free row, and a move that would exceed the
destination limit.

## Local verification

```bash
go test -count=1 ./backend/saveengine ./backend/endpoints/inventory
```
