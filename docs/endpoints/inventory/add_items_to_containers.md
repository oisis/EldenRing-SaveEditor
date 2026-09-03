# AddItemsToContainers

## Overview

`AddItemsToContainers` adds several catalog resources to the common section of
Inventory, of Storage, or of both, as **one atomic mutation of one save
revision**. It is the endpoint behind the shared *Add* dialog, in which the user
states an Inventory amount and a Storage amount per resource.

The two lists are independent requests for two containers, not one list behind a
destination switch. The receipt's `changedScopes` therefore name exactly the
containers the call actually wrote: `inventory` only, `storage` only, or both.
Those extra scopes are resolved through the same shared scope table every other
mutation uses; nothing assembles a scope list by hand.

| | |
|---|---|
| EndpointID | `add_items_to_containers` |
| Kind | Mutation |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport status | desktop bridge only — bound as a Wails method on `desktop.Bridge` and reached by the SaveForge frontend. It is deliberately **not** registered as an HTTP route of the local explorer and therefore does not appear in `tools/swagger/openapi.json` or in the Scalar portal: an operation the explorer cannot perform must never be advertised there. |
| Implementation source | [../../../backend/endpoints/inventory/add_items_to_containers.go](../../../backend/endpoints/inventory/add_items_to_containers.go) |
| Save access | read-write on the session's private in-memory snapshot; no file is opened |
| Mutation | per addition: a quantity top-up or a new common row, the common count of the affected section, the two trailing allocators, and one active GaItemData entry per newly owned item |

## Input

```go
type AddItemsRequestEntry struct {
	Kind              string  `json:"kind"`
	Key               string  `json:"key"`
	VariantID         *uint32 `json:"variantID,omitempty"`
	InventoryQuantity uint32  `json:"inventoryQuantity"`
	StorageQuantity   uint32  `json:"storageQuantity"`
}

func AddItemsToContainers(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	safetyProfile string,
	saveSessionID string,
	characterID int,
	items []AddItemsRequestEntry,
	confirmBanRisk bool,
	expectedRevision string,
) (AddItemsToContainersResult, error)
```

A quantity of zero means the entry does not address that container at all; an
entry addressing neither container is rejected rather than silently dropped.

## Catalog contract

Every entry passes the same common-only add contract the single-record
endpoints prove: a known family that is `goods` or `talisman`, a known game ID
whose prefix agrees with that family, a known category that is not `key_items`,
and a known record mode. A Storage amount additionally requires the resolved
subcategory not to be `Flasks` and, for goods, `item.goods.isDepositable`.

The two limits come from the shared Safety Profile policy:

```
maxContainerTotal = profile Inventory limit  /  profile Storage limit
maxPerRecord      = min(capabilities.stack.rules.maxPerStack, maxContainerTotal)
```

## Safety

`safetyprofile.AllowMutation` runs per entry, before anything is written:

- under `safe` and `expanded_limits` a resource marked `banRisk` or
  `cutContent` is refused outright;
- a resource marked `banRisk` additionally requires `confirmBanRisk`.

`confirmBanRisk` is the user's explicit confirmation and can never substitute
for a profile that forbids the resource. A call that bypasses the interface is
refused by exactly the same rule the interface renders.


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

Rejected before the session is touched: an empty `items`, an entry addressing
neither container, a repeated game ID inside one container list (two additions
of one item are never silently summed), a quantity of zero, a quantity above
what the record can store, a quantity other than 1 for a `separate_instances`
item, a missing limit, and a non-canonical `expectedRevision`.

## Local verification

```bash
go test -count=1 ./backend/saveengine ./backend/endpoints/inventory
```
