# GetOwnedItems

## Overview

`GetOwnedItems` returns one authoritative page of the Inventory or of the
Storage Box of one character: the records, resolved against GameCatalog and the
active Safety Profile, together with the mutations each record actually allows.

It is the getter the Inventory & Storage workspace reads. Unlike the raw
[`GetInventory`](get_inventory.md) and [`GetStorage`](get_storage.md), which stay
the physical projections, this endpoint:

- reads the **complete** container first and only then filters, sorts and
  slices, so the search, the category, the favourites and the sort order always
  describe the whole container instead of one served page;
- resolves the name, the icon path, the classification and the record mode;
- reports the container limit under the active profile as `maxQuantity`;
- reports, per record, which mutations the backend accepts (`actions`), so the
  interface renders no action the backend did not declare;
- reports the record's rank inside the manual Inventory order
  (`orderPosition`), counted over the whole container.

| | |
|---|---|
| EndpointID | `get_owned_items` |
| Kind | Getter |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport status | desktop bridge only — bound as a Wails method on `desktop.Bridge` and reached by the SaveForge frontend. It is deliberately **not** registered as an HTTP route of the local explorer and therefore does not appear in `tools/swagger/openapi.json` or in the Scalar portal: an operation the explorer cannot perform must never be advertised there. |
| Implementation source | [../../../backend/endpoints/inventory/get_owned_items.go](../../../backend/endpoints/inventory/get_owned_items.go) |
| Save access | read-only on the session's private in-memory snapshot |

## Input

```go
func GetOwnedItems(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	safetyProfile string,
	saveSessionID string,
	characterID int,
	container string,
	containerSection string,
	search string,
	category string,
	favoritesOnly bool,
	favorites []schema.ResourceRef,
	sortOrder string,
	page int,
	pageSize int,
) (GetOwnedItemsResult, error)
```

| Variable | Contract |
|---|---|
| `safetyProfile` | one of `safe`, `expanded_limits`, `chaos`; supplied by the bridge from the host setting |
| `container` | exactly `inventory` or `storage` |
| `containerSection` | forwarded unchanged to the raw getter below; `common`, `key` or empty |
| `search`, `category` | as in [`GetItemDatabase`](../catalog/get_item_database.md) |
| `favoritesOnly`, `favorites` | presentational preference, matched by the backend over the whole container |
| `sortOrder` | `""` (the container's own physical order), `name`, `category` or `quantity`; every order falls back to the physical order, so it is total and stable |
| `page`, `pageSize` | 1-based; `0` selects page 1 and `GetOwnedItemsDefaultPageSize` (30 — one 5 × 6 Grid card) |

## Output

```go
type OwnedItemActions struct {
	MoveToStorage   bool `json:"moveToStorage"`
	MoveToInventory bool `json:"moveToInventory"`
	Remove          bool `json:"remove"`
	SetQuantity     bool `json:"setQuantity"`
	Reorder         bool `json:"reorder"`
}
```

`OwnedItemRow` carries the physical fields verbatim (`ownedItemID`, `kind`,
`key`, `gameID`, `container`, `containerSection`, `physicalIndex`,
`acquisitionIndex`, `quantity`), the resolved catalog fields (`family`,
`category`, `subcategory`, `name`, `iconPath`, `recordMode` and the four
presentation safety flags), the profile-resolved `maxQuantity` with its
`maxQuantityKnown` flag, `orderPosition` with `orderPositionKnown`, and
`actions`.

`GetOwnedItemsResult` adds `saveSessionID`, `saveRevision`, `characterID`,
`active`, `safetyProfile`, `container`, `categories`, `total`, `page` and
`pageSize`.

`maxQuantityKnown` false means the catalog states no usable limit for that
container; the interface then shows no maximum rather than a substituted one.

## The action flags

Each flag is a **necessary condition the backend checked**, never a promise: the
mutation still validates the complete plan and may reject it.

| Flag | Condition |
|---|---|
| `remove` | the record is in a section with a confirmed removal contract (Inventory common or key) |
| `moveToStorage` | Inventory common, a known Storage limit under the profile, and depositable goods |
| `moveToInventory` | Storage common and a known Inventory limit under the profile |
| `setQuantity` | record mode `quantity_stack` with a known, enabled stack capability carrying a positive `maxPerStack` |
| `reorder` | Inventory common and a category the Inventory order contract supports |

Whether an Equipment, Quick Item or Pouch slot references the record is decided
by the shared removal planner inside the mutation, which fails closed;
duplicating that scan in a list getter would be a second implementation of the
same rule.

## Errors

Every failure of the underlying raw getter is reported unchanged. In addition:
an unknown `container`, an unknown `sortOrder`, an unknown `safetyProfile`,
negative paging and a `favorites` entry missing a kind or a key are all
rejected.

## Local verification

```bash
go test -count=1 ./backend/endpoints/inventory
```
