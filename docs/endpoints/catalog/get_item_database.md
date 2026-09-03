# GetItemDatabase

## Overview

`GetItemDatabase` returns one authoritative page of the general **Item
Database**: the item resources the active Safety Profile allows the interface to
present, filtered, sorted and paged by the backend.

It is the getter the Item Database screen reads. Unlike
[`GetResources`](get_resources.md), which stays the generic picker projection,
this endpoint:

- applies the global Safety Profile to visibility;
- accepts a category filter and a closed set of sort orders;
- reports the icon path, the save-side identifier and the presentation safety
  flags a list cell needs;
- reports the categories the current profile and family filter can reach, so the
  category control is built from backend data instead of a hardcoded list.

Filtering, sorting and paging all happen over the complete match set, so the
answer never depends on which page was requested. The catalog is global: this
endpoint works with no save session and reads no save.

| | |
|---|---|
| EndpointID | `get_item_database` |
| Kind | Getter |
| Domain | `catalog` |
| Implementation status | implemented |
| Transport status | desktop bridge only — bound as a Wails method on `desktop.Bridge` and reached by the SaveForge frontend. It is deliberately **not** registered as an HTTP route of the local explorer and therefore does not appear in `tools/swagger/openapi.json` or in the Scalar portal: an operation the explorer cannot perform must never be advertised there. |
| Implementation source | [../../../backend/endpoints/catalog/get_item_database.go](../../../backend/endpoints/catalog/get_item_database.go) |
| Save access | none |

## Input

```go
func GetItemDatabase(
	gameCatalog *gamecatalog.Catalog,
	safetyProfile string,
	family string,
	category string,
	search string,
	favoritesOnly bool,
	favorites []schema.ResourceRef,
	sortOrder string,
	page int,
	pageSize int,
) (GetItemDatabaseResult, error)
```

| Variable | Contract |
|---|---|
| `safetyProfile` | exactly one of `safe`, `expanded_limits`, `chaos`. The desktop bridge supplies the stored host setting; a caller never chooses it. |
| `family` | exact, case-sensitive match on `item.family`; empty never filters |
| `category` | exact, case-sensitive match on `item.category`; empty never filters |
| `search` | case-insensitive substring of the resource key or the resource name; empty never filters |
| `favoritesOnly` | restricts the result to the exact identities in `favorites` |
| `favorites` | the caller's presentational favourites as `(kind, key)` pairs. The backend matches them so a favourite outside the current page is never lost; it stores nothing. |
| `sortOrder` | `""` (catalog order: kind, then key), `name`, `category` or `game_id`. Every order falls back to the catalog order for equal rows, so it is total and stable. |
| `page` | 1-based; `0` selects page 1; negative is rejected |
| `pageSize` | `0` selects `GetItemDatabaseDefaultPageSize` (50); negative is rejected |

## Output

```go
type ItemDatabaseEntry struct {
	Kind        schema.ResourceKind `json:"kind"`
	Key         string              `json:"key"`
	GameID      uint32              `json:"gameID"`
	GameIDKnown bool                `json:"gameIDKnown"`
	Family      schema.ItemFamily   `json:"family"`
	Category    string              `json:"category"`
	Subcategory string              `json:"subcategory"`
	Name        string              `json:"name"`
	IconPath    string              `json:"iconPath"`
	BanRisk     bool                `json:"banRisk"`
	CutContent  bool                `json:"cutContent"`
	DLC         bool                `json:"dlc"`
	PreOrder    bool                `json:"preOrder"`
}

type GetItemDatabaseResult struct {
	SafetyProfile string                 `json:"safetyProfile"`
	Resources     []ItemDatabaseEntry    `json:"resources"`
	Categories    []ItemDatabaseCategory `json:"categories"`
	Total         int                    `json:"total"`
	Page          int                    `json:"page"`
	PageSize      int                    `json:"pageSize"`
}
```

An unknown fact stays empty or false. There is no fallback to the key, to a
category or to a placeholder: a synthesised value would be indistinguishable
from a real one. `Categories` is counted before the category filter is applied,
so choosing a category never erases the categories a user can switch to.

## Visibility

Visibility is the shared policy in `backend/safetyprofile` and nothing else:

- `noDatabase` known true → hidden under every profile;
- `banRisk` or `cutContent` known true → hidden unless the profile is `chaos`;
- `dlc` and `preOrder` never hide a row.

Only a **known** true hides a resource, so undecided catalog data never silently
removes an item from the list.

## Errors

| Condition | Result |
|---|---|
| no catalog is wired | `game catalog is not loaded` |
| unknown `safetyProfile` | `unknown safety profile %q; ...` |
| unknown `family` | `unknown item family %q` |
| unknown `sortOrder` | `unknown item database sort order %q` |
| negative `page` or `pageSize` | rejected with the offending value |
| a `favorites` entry missing a kind or a key | `favorites[i] must carry a kind and a key` |

## Local verification

```bash
go test -count=1 ./backend/endpoints/catalog
```
