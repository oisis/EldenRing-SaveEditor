# GetResourceRelations

## Overview

`GetResourceRelations` returns the outgoing and incoming relations of exactly one
resource stored in the already loaded GameCatalog. It returns relations only: the
documents of the related resources belong to `GetResource`, which the caller can
invoke separately for each relation endpoint.

| | |
|---|---|
| EndpointID | `get_resource_relations` |
| Kind | Getter |
| Domain | `catalog` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/catalog/resource-relations` of the local OpenAPI explorer (`backend/endpoints/swagger`). No Wails binding and no permanent CLI command reach it. |
| Implementation source | [../../../backend/endpoints/catalog/get_resource_relations.go](../../../backend/endpoints/catalog/get_resource_relations.go) |
| Test source | [../../../backend/endpoints/catalog/get_resource_relations_test.go](../../../backend/endpoints/catalog/get_resource_relations_test.go) |
| Save access | none — the endpoint never opens, reads, or writes a save |
| Mutation | none — the endpoint reads catalog copies and modifies nothing |

## Input

The public contract of this endpoint has four input parameters: `kind`, `key`,
`relationType`, and `direction`.

The Go signature is:

```go
func GetResourceRelations(
    gameCatalog *gamecatalog.Catalog,
    kind string,
    key string,
    relationType string,
    direction string,
) (GetResourceRelationsResult, error)
```

### The resource identity is the pair `(kind, key)`

A resource is identified by the exact pair `schema.Resource.Kind` plus
`schema.Resource.Key`, for example `kind=item` and `key=000F4240`. There is no
numeric resource identifier in the public contract; `schema.ResourceID` no longer
exists.

The pair is **not** the item's `Item.GameID`, a variant ID, or the name of the
JSON file the resource was loaded from.

The lookup resolves the kind first and the key only inside that kind:

- `kind` is matched exactly against `Resource.Kind`. `item` is the only kind the
  current schema supports.
- `key` is matched exactly against `Resource.Key` inside the resolved kind.
- Neither value is trimmed, case-folded, parsed, or retried under another kind.
- The pre-migration key form `item:000F4240` carried the kind as a prefix. It is
  now an unknown key and never an alias of `000F4240`.

The pair is resolved through the same `Catalog` helper `GetResource` uses, so an
unknown kind and an unknown key produce the identical two errors in both getters.

### The catalog argument

`gameCatalog` is an already loaded `*gamecatalog.Catalog`, supplied by the backend
caller and not by a client. The endpoint never loads, reloads, or rescans catalog
data, never opens a file, and never reads a save. A `nil` catalog is an error.

## Output

The endpoint returns a typed result:

```go
type GetResourceRelationsResult struct {
	Outgoing []schema.Relation `json:"outgoing"`
	Incoming []schema.Relation `json:"incoming"`
}
```

`outgoing` and `incoming` are the only two top-level fields. Each entry is a
`schema.Relation` passed through unchanged:

| Field | Type | Meaning |
|---|---|---|
| `from` | `schema.ResourceRef` | The resource the relation starts at, as `{kind, key}`. |
| `to` | `schema.ResourceRef` | The resource the relation points at, as `{kind, key}`. |
| `kind` | `schema.RelationKind` | `compatible_with_aow` or `requires_container`. |
| `provenance` | `schema.Provenance` | The evidence the catalog derived the relation from. |

`from` and `to` are `ResourceRef` objects carrying `kind` and `key`, never numeric
identifiers, because that is how the catalog stores a relation after the
kind/key migration. Each `ResourceRef` can be passed straight back into
`GetResource` to fetch the document of the related resource.

Relations are returned in catalog order. The endpoint never sorts, normalises,
deduplicates, or synthesises a relation, and it never returns the document of a
related resource.

A resource without relations is a valid result: both fields are empty JSON arrays
`[]`, never `null`. The same holds for a direction that was filtered out.

## Filters

Both filters are matched exactly and case-sensitively. They are never trimmed and
never normalised.

| Parameter | Empty value | Accepted values |
|---|---|---|
| `relationType` | every relation type | `compatible_with_aow`, `requires_container` |
| `direction` | both directions | `outgoing`, `incoming` |

- `direction=outgoing` returns only outgoing relations; `incoming` stays an empty
  array.
- `direction=incoming` returns only incoming relations; `outgoing` stays an empty
  array.
- There is no `direction=both`. Both directions are requested with an empty
  `direction`.
- `relationType` and `direction` combine: each surviving relation must match both
  filters.
- A filter that matches nothing is not an error; it returns empty arrays.

## Processing flow

1. The caller loads the catalog data (for example through `loader.LoadDir` from
   `backend/gamecatalog/data`) and builds the catalog through `gamecatalog.New`,
   which validates the manifest and the resources, indexes every resource by its
   kind and, inside that kind, by its `Resource.Key`, and derives the relations
   once.
2. The caller passes that catalog and the four parameters to
   `GetResourceRelations`.
3. The getter validates `gameCatalog`, `kind`, `key`, `relationType`, and
   `direction`. Both filters are validated before the lookup, so an unsupported
   filter fails even for a resource that exists.
4. The getter resolves the pair through `Catalog.RelationsByKindAndKey`, which
   reads the two-level kind/key index and the relation maps built once during
   `gamecatalog.New`. There is no directory scan, no JSON read, and no linear
   search per call. The getter calls no other endpoint; in particular it does not
   call `GetResource`, `GetItemVariants`, or `GetCatalogInfo`.
5. The catalog returns two independent copies of the stored relations.
6. The getter copies the relations that match both filters into the result,
   preserving their catalog order, and returns it.

## Validation and errors

Every failure returns an empty `GetResourceRelationsResult` together with the
error. A partial result is never returned alongside an error.

| Condition | Error |
|---|---|
| `gameCatalog` is `nil` | `game catalog is not loaded` |
| `kind` is empty | `resource kind is required` |
| `key` is empty | `resource key is required` |
| `relationType` is not a supported `schema.RelationKind` | `relation type "…" is not supported; use "compatible_with_aow", "requires_container" or an empty value for every relation type` |
| `direction` is not `outgoing`, `incoming`, or empty | `direction "…" is not supported; use "outgoing", "incoming" or an empty value for both directions` |
| `kind` matches no catalog kind | `unknown resource kind "…"` |
| `key` matches no `Resource.Key` inside the resolved kind | `unknown resource key "…" in kind "…"` |

Notes:

- A missing kind, an unknown kind, a missing key and a key that is unknown inside
  an existing kind are four distinguishable errors, worded exactly as in
  `GetResource`.
- Whitespace is never trimmed. `" 000F4240"` is reported as an unknown key, not
  as a silent lookup of `"000F4240"`.
- A numeric string such as `"1"` or `"1000000"` is not a `Resource.Key` and is
  reported as unknown. The `GameID` is not accepted as input.
- A zero-value `gamecatalog.Catalog` that never went through `gamecatalog.New`
  has no kind index, so every lookup against it reports an unknown kind.
- The validation of this endpoint is implemented in its own file. It shares no
  validation helper, no DTO, and no error type with another endpoint.
- This endpoint does not use a shared `EndpointError` type; that type does not
  exist yet.

## Result immutability

The result is safe to modify. `Catalog.RelationsByKindAndKey` returns independent
copies of the relation slices, and `schema.Relation` holds only value fields, so a
copy is complete. Mutating any part of `GetResourceRelationsResult`, including
appending to either slice or overwriting a `from`/`to` reference, does not change
the catalog, and a later call returns the original data. The getter itself
modifies nothing: it never adds, removes, repairs, or normalises catalog data.

## Save access

The endpoint never opens, reads, or writes a save, and it never uses
`SaveEngine`. It reads the GameCatalog only.

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

result, err := catalog.GetResourceRelations(gameCatalog, "item", "000F4240", "", "")
if err != nil {
    log.Fatalf("GetResourceRelations: %v", err)
}
log.Printf("outgoing=%d incoming=%d", len(result.Outgoing), len(result.Incoming))
```

## Calling it over HTTP

The endpoint is exposed by the local, read-only OpenAPI explorer. Start it from
the repository root; it binds loopback only:

```bash
go run ./backend/endpoints/swagger -addr 127.0.0.1:8788 -data ./backend/gamecatalog/data
```

The browser explorer is at `http://127.0.0.1:8788/docs` and reads
`http://127.0.0.1:8788/openapi.json`.

```bash
curl -s "http://127.0.0.1:8788/api/v1/catalog/resource-relations?kind=item&key=000F4240"
```

Filtered variants:

```bash
# only outgoing relations
curl -s "http://127.0.0.1:8788/api/v1/catalog/resource-relations?kind=item&key=000F4240&direction=outgoing"

# only incoming relations
curl -s "http://127.0.0.1:8788/api/v1/catalog/resource-relations?kind=item&key=8000EA60&direction=incoming"

# one relation type
curl -s "http://127.0.0.1:8788/api/v1/catalog/resource-relations?kind=item&key=000F4240&relationType=compatible_with_aow"
```

The HTTP route only reads the four query parameters, calls
`GetResourceRelations`, and serialises its result. It holds no validation and no
filtering of its own. Every input error becomes HTTP 400 carrying the exact
message of the getter in the shared `{"error": "…"}` envelope.

## Example output

`kind=item` with `key=000F4240` is the Dagger; the catalog derives one
`compatible_with_aow` relation per compatible Ash of War. The example is
abbreviated; `…` marks omitted entries:

```json
{
  "outgoing": [
    {
      "from": { "kind": "item", "key": "000F4240" },
      "to": { "kind": "item", "key": "8000EA60" },
      "kind": "compatible_with_aow",
      "provenance": {
        "source": "regulation_equip_param_gem_raw",
        "method": "parsed the full 44-bit compatibility field",
        "table": "EquipParamGem",
        "row": "60000",
        "field": "canMountWep[0:44]"
      }
    }
  ],
  "incoming": []
}
```

The same relation appears as an incoming relation of the Ash of War it points at:

```bash
curl -s "http://127.0.0.1:8788/api/v1/catalog/resource-relations?kind=item&key=8000EA60&direction=incoming"
```

```json
{
  "outgoing": [],
  "incoming": [
    {
      "from": { "kind": "item", "key": "000F4240" },
      "to": { "kind": "item", "key": "8000EA60" },
      "kind": "compatible_with_aow",
      "provenance": { "…": "…" }
    }
  ]
}
```

A resource without any relation returns two empty arrays:

```json
{
  "outgoing": [],
  "incoming": []
}
```

A rejected filter:

```bash
curl -s "http://127.0.0.1:8788/api/v1/catalog/resource-relations?kind=item&key=000F4240&direction=both"
```

```json
{
  "error": "direction \"both\" is not supported; use \"outgoing\", \"incoming\" or an empty value for both directions"
}
```

## Test verification

From the repository root:

```bash
go test ./backend/endpoints/catalog -run '^TestGetResourceRelations' -count=1 -v
go test ./backend/endpoints/swagger -run '^TestResourceRelations' -count=1 -v
```

The getter suite covers the outgoing relations of the Dagger, the incoming
relations of Determination, `relationType` filtering, `direction` filtering, the
`from`/`to` `ResourceRef` carrying both `kind` and `key`, a `nil` catalog, an
empty kind, an empty key, an unknown kind, an unknown key, an unsupported
`relationType`, an unsupported `direction`, the empty directions serialised as
`[]` instead of `null`, and the immutability of the catalog after the result is
mutated.

## Current limitations

- The endpoint is not exposed through Wails and has no permanent CLI command.
  Only the local explorer reaches it.
- There is no caller in the runtime of the main application.
- The getter does not load the catalog. It requires an already loaded
  `*gamecatalog.Catalog` supplied by the caller.
- Lookup is by the exact `(kind, key)` pair only. There is no lookup by `GameID`,
  by name, or by variant.
- The result carries `from` and `to` as references. Resolving them to documents is
  the caller's job through `GetResource`.
- Only the two relation kinds the catalog derives today exist:
  `compatible_with_aow` and `requires_container`. The endpoint accepts no other
  relation type.
- There is no pagination and no search. A resource compatible with many Ashes of
  War returns every relation in one response.
- There is no `direction=both` value; an empty `direction` is the way to ask for
  both.
