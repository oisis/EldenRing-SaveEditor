# GetResources

## Overview

`GetResources` returns one page of the resources stored in the already loaded
GameCatalog, reduced to the four fields a list or a picker needs: `kind`, `key`,
`family`, and `name`. It is deliberately not a document getter: capabilities,
variants, relations, provenance, and the full `schema.ItemDocument` stay the
responsibility of `GetResource`, `GetItemVariants`, and `GetResourceRelations`.

The point of the endpoint is that a picker for any item category is one call with
typed filters, instead of one dedicated getter per category.

| | |
|---|---|
| EndpointID | `get_resources` |
| Kind | Getter |
| Domain | `catalog` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/catalog/resources` of the local OpenAPI explorer (`tools/swagger`). No Wails binding and no permanent CLI command reach it. |
| Implementation source | [../../../backend/endpoints/catalog/get_resources.go](../../../backend/endpoints/catalog/get_resources.go) |
| Test source | [../../../backend/endpoints/catalog/get_resources_test.go](../../../backend/endpoints/catalog/get_resources_test.go) |
| Save access | none — the endpoint never opens, reads, or writes a save |
| Mutation | none — the endpoint reads catalog copies and modifies nothing |

## Input

The public contract of this endpoint has seven input parameters: `resourceType`,
`family`, `capability`, `endpointId`, `search`, `page`, and `pageSize`.

The Go signature is:

```go
func GetResources(
    gameCatalog *gamecatalog.Catalog,
    resourceType string,
    family string,
    capability string,
    endpointID string,
    search string,
    page int,
    pageSize int,
) (GetResourcesResult, error)
```

### The catalog argument

`gameCatalog` is an already loaded `*gamecatalog.Catalog`, supplied by the backend
caller and not by a client. The endpoint never loads, reloads, or rescans catalog
data, never opens a file, and never reads a save. A `nil` catalog is an error.

The whole resource set is read through one catalog method:

```go
func (catalog *Catalog) ResourceSummaries() []ResourceSummary
```

`ResourceSummaries` returns a value-only snapshot of exactly the fields this list
needs — kind, key, family, name, and the `Known`/`Enabled` pair of the five
capabilities — ordered by kind and only then by key. It copies scalars only:
the full documents, their variants, provenance, and capability rules are never
deep-copied, so listing does not pay for data the projection discards. The
summary types hold no pointer, map, or slice, so no mutable catalog state is
exposed.

The method offers no filtering, search, or paging on purpose: selecting and
slicing belong to this endpoint, and deciding what a `Known`/`Enabled` pair means
stays in the getter.

## Filters

Every filter is independent and combined with AND. An empty filter never filters.
No filter value is trimmed, case-folded, or normalised, except `search`, which is
explicitly case-insensitive.

### `resourceType`

Matched exactly and case-sensitively against `schema.Resource.Kind`.

- Empty means every kind, which today mixes items, colosseums and regions in one
  page.
- `item`, `colosseum` and `region` are the kinds the current schema declares, so
  they are the only accepted non-empty values.
- Any other value, including `Item`, is rejected with
  `unsupported resource type "…"`. It is not silently ignored and it does not
  fall back to the unfiltered list.

### `family`

Matched exactly and case-sensitively against `Item.Family`.

- Empty means every family.
- The accepted values are the eight existing `schema.ItemFamily` constants:
  `weapon`, `armor`, `talisman`, `ash_of_war`, `spell`, `spirit_ash`, `goods`,
  `gesture`. No family is invented here.
- Any other value is rejected with `unknown item family "…"`.
- A resource matches only when it carries an item document **and** its family is
  `Known`. A resource whose family was never established, and every resource of
  a non-item kind, is never reported as a member of the requested family.
- A valid family that no resource carries yields an empty page and `total` 0, not
  the unfiltered list.

### `capability`

Selects one field of `schema.ItemCapabilities`.

- Empty means no capability filter.
- The accepted values mirror the struct fields one by one: `upgrade`, `infusion`,
  `ashOfWarMount`, `stack`, `equipment`. There is no alias.
- Any other value is rejected with `unknown capability "…"`.
- A resource matches **only** when that capability has both `Known == true` and
  `Enabled == true`. A known but disabled capability does not match, and an
  unknown capability is not treated as an enabled one, so undecided data can
  never widen a picker.

### `endpointId`

Not supported.

GameCatalog declares no resource-to-endpoint relation today: there is no field,
no relation kind, and no derived mapping that could answer the question "which
resources does endpoint X apply to". The endpoint therefore refuses the filter
instead of guessing:

- an empty `endpointId` means the filter is simply absent;
- any non-empty `endpointId` returns the error
  `the endpointId filter is not supported because GameCatalog does not declare endpoint relations yet; got "…"`.

No mapping is invented in the getter, in the route, or in the schema. Supporting
this filter requires the catalog to declare the relation first.

### `search`

A case-insensitive substring search.

- Empty means no search.
- The search runs against `Resource.Key` and against the resource name: the item
  name (`Item.Presentation.Name.Value`), colosseum name
  (`Colosseum.Name.Value`) or region name (`Region.Name.Value`).
- An unknown name is the empty string in the projection, so it is never searched
  as a placeholder value.
- A resource matches when the substring occurs in either field.

## Paging

- `page` is one-based. `0` means page 1.
- `pageSize` `0` means the default of `50` (`catalog.GetResourcesDefaultPageSize`).
- A negative `page` or `pageSize` is an error; it is never clamped to a default.
- There is deliberately **no** maximum `pageSize`. The projection is small and the
  caller decides how much of the catalog it wants.
- A page beyond the last one returns an empty, non-nil `resources` array together
  with the real `total`, so the payload is `[]` and never `null`.
- The order is the catalog order — kind first and only then key — so paging is
  stable across calls and a resource never appears on two pages.

## Output

The endpoint returns a typed result:

```go
type GetResourcesEntry struct {
    Kind   schema.ResourceKind `json:"kind"`
    Key    string              `json:"key"`
    Family schema.ItemFamily   `json:"family"`
    Name   string              `json:"name"`
}

type GetResourcesResult struct {
    Resources []GetResourcesEntry `json:"resources"`
    Total     int                 `json:"total"`
    Page      int                 `json:"page"`
    PageSize  int                 `json:"pageSize"`
}
```

`resources`, `total`, `page`, and `pageSize` are the only four top-level fields,
and `kind`, `key`, `family`, and `name` are the only four fields of an entry.

- `total` counts every resource that passed the filters, **before** paging, so a
  caller can size a picker without walking every page.
- `page` and `pageSize` report the **effective** values, so a requested `0` comes
  back as `1` and `50`.
- `family` is empty when the resource carries no item document or its family is
  not `Known`.
- `name` is empty when the name of the resource is not `Known`. There is no
  fallback to the key, to a category, or to a placeholder: a synthesised name
  would be indistinguishable from a real one.

The result carries no relations, no variants, no provenance, no capabilities, and
no `ItemDocument`.

## Processing flow

1. Reject a `nil` catalog.
2. Validate `resourceType`, `family`, and `capability` against the values the
   schema declares.
3. Reject any non-empty `endpointId`.
4. Reject a negative `page` or `pageSize`, then apply the defaults for `0`.
5. Read every resource summary through `Catalog.ResourceSummaries`, already
   ordered by kind and then key.
6. Apply `resourceType`, `family`, and `capability`, project the entry, then
   apply `search` to the projected key and name.
7. Count the matches into `total`.
8. Slice the requested page, returning an empty array when the page lies beyond
   the last one.

## Validation and errors

| Condition | Message |
|---|---|
| `gameCatalog` is `nil` | `game catalog is not loaded` |
| `resourceType` is neither empty nor `item`, `colosseum` or `region` | `unsupported resource type "…"` |
| `family` is neither empty nor a `schema.ItemFamily` | `unknown item family "…"` |
| `capability` is neither empty nor a capability name | `unknown capability "…"` |
| `endpointId` is not empty | `the endpointId filter is not supported because GameCatalog does not declare endpoint relations yet; got "…"` |
| `page` is negative | `page must not be negative; got …` |
| `pageSize` is negative | `pageSize must not be negative; got …` |

An empty result is never an error: a filter that excludes everything returns
`{"resources": [], "total": 0, …}`.

## Result immutability

`ResourceSummaries` returns value-only summaries that contain no pointer, map, or
slice, and the entries are plain scalar projections of those summaries. Mutating
the returned result can never reach the catalog, and the getter modifies neither
the catalog nor the data it read.

## Save access

None. The endpoint never opens, reads, or writes a save, and it calls no other
endpoint.

## Calling it from Go

```go
data, err := loader.LoadDir("backend/gamecatalog/data")
if err != nil {
    log.Fatalf("load catalog data: %v", err)
}
gameCatalog, err := gamecatalog.New(data.Manifest, data.Resources())
if err != nil {
    log.Fatalf("build catalog: %v", err)
}

result, err := catalog.GetResources(gameCatalog, "item", "weapon", "infusion", "", "", 1, 20)
if err != nil {
    log.Fatalf("GetResources: %v", err)
}
log.Printf("page %d/%d of %d matches", result.Page, result.PageSize, result.Total)
```

## Calling it over HTTP

The endpoint is exposed by the local, read-only OpenAPI explorer. Start it from
the repository root; it binds loopback only:

```bash
go run ./tools/swagger -addr 127.0.0.1:8788 -data ./backend/gamecatalog/data
```

It serves the OpenAPI document at `http://127.0.0.1:8788/openapi.json` and no
page of its own; the browser interface is the Scalar portal started by
`tools/run_swagger.sh start`.

```bash
curl -s "http://127.0.0.1:8788/api/v1/catalog/resources"
```

Filtered variants:

```bash
# every weapon
curl -s "http://127.0.0.1:8788/api/v1/catalog/resources?resourceType=item&family=weapon"

# only resources whose infusion capability is known and enabled
curl -s "http://127.0.0.1:8788/api/v1/catalog/resources?capability=infusion"

# case-insensitive search on key and name
curl -s "http://127.0.0.1:8788/api/v1/catalog/resources?search=dagger"

# second page of ten
curl -s "http://127.0.0.1:8788/api/v1/catalog/resources?page=2&pageSize=10"
```

The HTTP route reads the seven query parameters, converts `page` and `pageSize`
to integers, calls `GetResources`, and serialises its result. It holds no
filtering of its own. Every getter error becomes HTTP 400 carrying the exact
message of the getter in the shared `{"error": "…"}` envelope. A `page` or
`pageSize` that is not an integer never reaches the getter, so the route reports
it itself as `page must be an integer; got "…"`.

## Example output

A page of the two prototype resources, the Dagger and Ash of War: Determination:

```json
{
  "resources": [
    { "kind": "item", "key": "000F4240", "family": "weapon", "name": "Dagger" },
    { "kind": "item", "key": "8000EA60", "family": "ash_of_war", "name": "Ash of War: Determination" }
  ],
  "total": 2,
  "page": 1,
  "pageSize": 50
}
```

A page beyond the last one keeps the real total and returns an empty array:

```bash
curl -s "http://127.0.0.1:8788/api/v1/catalog/resources?page=99&pageSize=1"
```

```json
{
  "resources": [],
  "total": 2,
  "page": 99,
  "pageSize": 1
}
```

The rejected `endpointId` filter:

```bash
curl -s "http://127.0.0.1:8788/api/v1/catalog/resources?endpointId=get_resource"
```

```json
{
  "error": "the endpointId filter is not supported because GameCatalog does not declare endpoint relations yet; got \"get_resource\""
}
```

## Test verification

From the repository root:

```bash
go test ./backend/endpoints/catalog -run '^TestGetResources' -count=1 -v
go test ./tools/swagger -run '^TestResourcesRoute' -count=1 -v
```

The getter suite covers the unfiltered catalog order, the four-field projection,
`resourceType` and `family` filtering including a valid but unused family, a
known-and-enabled capability against a known-but-disabled one, case-insensitive
search on both the key and the name, deterministic paging with `total`, a page
beyond the last one serialised as `[]` instead of `null`, the rejected
`endpointId` filter with its exact message, a `nil` catalog, and the unknown or
negative filter and paging values.

## Current limitations

- The endpoint is not exposed through Wails and has no permanent CLI command.
  Only the local explorer reaches it.
- There is no caller in the runtime of the main application.
- The getter does not load the catalog. It requires an already loaded
  `*gamecatalog.Catalog` supplied by the caller.
- `endpointId` is accepted only as an empty value. GameCatalog declares no
  endpoint relations yet, so the filter cannot be answered from data.
- `item`, `colosseum` and `region` are the resource kinds the schema declares
  today. The `family` and `capability` filters describe items only and never
  match a colosseum or region.
- The result is a projection for lists and pickers. A caller that needs the full
  document calls `GetResource` for the selected `(kind, key)` pair.
- Filtering and paging run over the whole resource set on every call; there is no
  index and no cache.
