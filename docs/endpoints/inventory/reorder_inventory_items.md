# ReorderInventoryItems

## Overview

`ReorderInventoryItems` moves a selected group of supported Inventory instances
to a new position, anchored on one member of that group, as **one atomic
mutation of one save revision**.

The placement rule is stated once, here:

> The anchor lands on `targetPosition` of the resulting order. The selected
> records that were in front of it stay in front of it, and the selected records
> that were behind it stay behind it. The group keeps its internal order, and
> every record outside the group keeps its relative order too.

The result is one complete supported order, written through the same planner
[`SetInventoryOrder`](set_inventory_order.md) uses, so the acquisition
allocator, the retained buckets and the unsafe-index bound have exactly one
implementation. Physical rows, handles, quantities, key records,
`NextEquipIndex`, Equipment, Storage and the GaItem table all stay unchanged.

Storage has no manual order and is deliberately not addressable here.

| | |
|---|---|
| EndpointID | `reorder_inventory_items` |
| Kind | Mutation |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport status | desktop bridge only — bound as a Wails method on `desktop.Bridge` and reached by the SaveForge frontend. It is deliberately **not** registered as an HTTP route of the local explorer and therefore does not appear in `tools/swagger/openapi.json` or in the Scalar portal: an operation the explorer cannot perform must never be advertised there. |
| Implementation source | [../../../backend/endpoints/inventory/reorder_inventory_items.go](../../../backend/endpoints/inventory/reorder_inventory_items.go) |
| Save access | read-write on the session's private in-memory snapshot; no file is opened |
| Changed scopes | `inventory` |

## Input

```go
func ReorderInventoryItems(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	anchorOwnedItemID string,
	groupOwnedItemIDs []string,
	targetPosition int,
	expectedRevision string,
) (ReorderInventoryItemsResult, error)
```

`targetPosition` is the zero-based position the anchor takes in the resulting
**supported** Inventory order. [`GetOwnedItems`](get_owned_items.md) reports each
record's current rank in that order as `orderPosition`, counted over the whole
container, so a caller never derives a position from a page, an acquisition
index or a physical row.

Which records take part in the order is the same classification
`SetInventoryOrder` uses: a known category in the confirmed order set, excluding
the technical Unarmed record.

## Output

`ReorderInventoryItemsResult` embeds the shared `MutationReceipt` flat and adds
`characterID`, the complete `orderedResources` in stable catalog terms and the
`acquisitionIndices` the plan assigned.


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

Rejected before the session is touched: an empty `anchorOwnedItemID`, an empty
`groupOwnedItemIDs`, a repeated identity, an anchor outside the group, a
negative `targetPosition` and a non-canonical `expectedRevision`.

Rejected during planning, which discards the whole candidate: an identity that is
not a supported Inventory common record, a `targetPosition` outside the supported
order, and a position the anchored group cannot occupy — the group needs
`anchorIndexInGroup` records in front of the anchor, so a target closer to the
start than that is refused instead of being clamped.

## Local verification

```bash
go test -count=1 ./backend/saveengine ./backend/endpoints/inventory
```
