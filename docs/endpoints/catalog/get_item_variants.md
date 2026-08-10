# GetItemVariants

## Overview

`GetItemVariants` returns the variants stored in the item document of exactly one
resource of the already loaded GameCatalog, so a consumer never has to derive
upgrade levels or infusions itself. It returns only the variants, not the item
document that holds them; the full document belongs to `GetResource`.

| | |
|---|---|
| EndpointID | `get_item_variants` |
| Kind | Getter |
| Domain | `catalog` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/catalog/item-variants` of the local OpenAPI explorer (`backend/swagger`). No Wails binding and no permanent CLI command reach it. |
| Implementation source | [../../../backend/endpoints/catalog/get_item_variants.go](../../../backend/endpoints/catalog/get_item_variants.go) |
| Test source | [../../../backend/endpoints/catalog/get_item_variants_test.go](../../../backend/endpoints/catalog/get_item_variants_test.go) |
| Save access | none — the endpoint never opens, reads, or writes a save |
| Mutation | none — the endpoint reads catalog copies and modifies nothing |

## Input

The public contract of this endpoint has exactly two input parameters, `kind`
and `key`.

The Go signature is:

```go
func GetItemVariants(gameCatalog *gamecatalog.Catalog, kind string, key string) (GetItemVariantsResult, error)
```

### The resource identity is the pair `(kind, key)`

A resource is identified by the exact pair `schema.Resource.Kind` plus
`schema.Resource.Key`, for example `kind=item` and `key=000F4240`. There is no
numeric resource identifier in the public contract; `schema.ResourceID` and the
top-level `Resource.id` field no longer exist.

The pair is **not**:

- the item's `Item.GameID`;
- a variant ID;
- the name of the JSON file the resource was loaded from.

The lookup resolves the kind first and the key only inside that kind, performed
directly through `Catalog.ResourceByKindAndKey`:

- `kind` must be `item`. Only item resources carry variants, so any other kind is
  rejected before the key is looked at.
- `key` is matched exactly against `Resource.Key` inside the item kind. The same
  key may later exist under a different kind, so the key alone is not an
  identity.
- Neither value is trimmed, case-folded, parsed, or retried under another kind.
  An input with surrounding whitespace is reported as unknown rather than
  repaired.
- The pre-migration key form `item:000F4240` carried the kind as a prefix. It is
  now an unknown key and never an alias of `000F4240`.
- A variant is never addressable as input. There is no lookup of a single variant
  by its `GameID`, its affinity, or its upgrade level.

`schema.ValidateResource` requires an item key to be exactly eight uppercase
hexadecimal characters (`0-9`, `A-F`), so `000F4240` is well formed and
`000f4240` is not. `gamecatalog.New` rejects a catalog containing the same
`(kind, key)` pair twice, so at most one resource can match.

### The catalog argument

`*gamecatalog.Catalog` is a backend dependency, not a transport parameter:

- It is supplied by the backend caller, not by a client.
- The caller has to load and build the catalog itself before calling the getter.
  `GetItemVariants` never does that on its own and never reads or rescans the
  JSON data files.
- No caller exists in the runtime of the main application today. The endpoint is
  currently invoked only outside that runtime:
  - the unit tests call the getter directly with the prototype catalog;
  - the command-line example in
    [Print the real getter output](#print-the-real-getter-output) calls it
    directly with the real `backend/gamecatalog/data`.
- Once the endpoint is wired into a runtime, that runtime becomes responsible for
  owning the loaded catalog and passing it in.

## Output

The endpoint returns a typed result:

```go
type GetItemVariantsResult struct {
	Variants []schema.ItemVariant `json:"variants"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `variants` | `[]schema.ItemVariant` | The variants stored in `Item.Variants` of the resolved resource, in catalog order. |

`variants` is the only top-level field. The result carries no resource document,
no item document, and no relations.

Each entry is the complete `schema.ItemVariant`, not a reduced projection:

| Field | Type | Meaning |
|---|---|---|
| `gameID` | `Fact[uint32]` | The in-game item ID of this variant. |
| `kind` | `Fact[ItemVariantKind]` | `affinity`, `upgrade`, or `affinity_upgrade`. |
| `affinity` | `Fact[Affinity]` | The affinity of the variant, when it has one. |
| `upgradeLevel` | `Fact[uint8]` | The upgrade level of the variant. A base affinity variant carries a known level of `0` with its own provenance, not an unknown fact; `upgrade` and `affinity_upgrade` variants carry their actual level. |
| `sourceRowID` | `Fact[uint32]` | The regulation row the variant was derived from. |
| `data` | `VariantDocumentData` | The variant-specific document data (`presentation`, `storage`, `capabilities`, `safety`, `acquisition`, `modifiers`, `links`, `unlocks`, `relatedTechnicalRecords`, `weapon`, `spiritAsh`). |
| `sourceRecords` | `[]ParameterRecord` | The parameter records backing the variant. |

Provenance is preserved: every `Fact` keeps its `provenance` record, inside the
variant itself and inside `data`. The endpoint defines no second copy of the
schema models and flattens nothing into new DTOs.

### Only stored variants, never a synthesised base item

The result contains exactly the entries of `Item.Variants` and nothing else:

- The base item is **not** part of `Item.Variants` and is **never** synthesised
  into an extra variant. A caller that also needs the base item reads it through
  `GetResource`.
- Variants are returned as stored. They are not materialised through
  `schema.MaterializeVariant`, so `data` stays a `VariantDocumentData` block and
  is never merged into a full `ItemDocument`.
- Nothing is filtered, sorted, normalised, deduplicated, or recomputed. The
  order is the order the GameCatalog holds, which is the order of the generated
  data.

### An item without variants

An item whose document stores no variants is a valid case, not an error. The
getter returns a successful result whose `variants` is an empty array:

```json
{"variants":[]}
```

The field is never `null`. The catalog's clone layer yields a `nil` slice for an
item without variants, so the getter returns an explicitly empty
`[]schema.ItemVariant` to keep the JSON array contract. In the current data the
Determination Ash of War (`kind=item`, `key=8000EA60`) is such an item.

## Processing flow

This is what a caller that wants to invoke `GetItemVariants` has to do. It is not
a sequence the main application performs today.

1. The caller loads the catalog data through `loader.LoadDir`, for example from
   `backend/gamecatalog/data`.
2. The caller builds the catalog through `gamecatalog.New`, which validates the
   manifest and the resources and indexes every resource by its kind and, inside
   that kind, by its `Resource.Key`. A catalog that fails any of those checks is
   never constructed.
3. The caller passes that catalog, a `kind` and a `key` to `GetItemVariants`. The
   getter never loads, reloads, or rescans anything.
4. The getter validates `gameCatalog`, `kind` and `key`.
5. The getter resolves the pair through `Catalog.ResourceByKindAndKey`, which
   selects the map of the item kind and then the key inside it, using the
   two-level index built once during `gamecatalog.New`. There is no directory
   scan, no JSON read, and no linear search per call. The getter calls no other
   endpoint; in particular it does not call `GetResource`.
6. The catalog returns that one resource as an independent deep copy.
7. The getter takes `Item.Variants` of that copy, substitutes an empty slice for
   a missing one, puts it into `GetItemVariantsResult`, and returns it.

## Validation and errors

Every failure returns an empty `GetItemVariantsResult` together with the error. A
partial result is never returned alongside an error.

| Condition | Error |
|---|---|
| `gameCatalog` is `nil` | `game catalog is not loaded` |
| `kind` is empty | `resource kind is required` |
| `kind` is not `item` | `resource kind "…" has no item variants; only kind "item" is supported` |
| `key` is empty | `resource key is required` |
| `key` matches no `Resource.Key` inside the item kind | `unknown resource key "…" in kind "item"` |
| the resolved resource carries no `ItemDocument` | `resource kind "…" key "…" is not an item and has no variants` |

Notes:

- A missing kind, a kind other than `item`, a missing key and a key that is
  unknown inside the item kind are four distinguishable errors.
- The unknown-key error names both the key and the kind it was searched in.
- Whitespace is never trimmed. `" 000F4240"` is reported as an unknown key, not
  as a silent lookup of `"000F4240"`.
- The pre-migration key `"item:000F4240"` is reported as an unknown key under
  `kind=item`. There is no backward compatibility with the prefixed form.
- A numeric string such as `"1"` or `"1000000"` is not a `Resource.Key` and is
  reported as unknown. The `GameID` is not accepted as input.
- A zero-value `gamecatalog.Catalog` that never went through `gamecatalog.New`
  has no kind index, so every lookup against it reports an unknown kind.
- The missing-`ItemDocument` error is a defensive guard. `gamecatalog.New`
  rejects a resource without an item document, so a catalog built through the
  public constructor cannot produce it. Unknown data fails safely instead of
  being turned into an empty variant list.
- An item without variants is **not** an error; see
  [An item without variants](#an-item-without-variants).
- This endpoint does not use the shared `EndpointError` type; that type does not
  exist yet.

## Result immutability

The result is safe to modify. `GetItemVariants` hands out the variants of the
deep copy the catalog's existing query layer produced, and that layer already
deep-copies every variant, its `data` block, its capability rule slices, and its
parameter records. Mutating any part of `GetItemVariantsResult` does not change
the catalog, and a later call returns the original data. The getter itself
modifies nothing: it never adds, removes, repairs, or normalises catalog data,
and it never touches other application data.

## Save access

The endpoint never opens, reads, or writes a save, and it never uses
`SaveEngine`. It reads the GameCatalog only.

## Command-line verification

`GetItemVariants` is exposed over HTTP as `GET /api/v1/catalog/item-variants` by
the local OpenAPI explorer in `backend/swagger`, a developer tool the
application neither imports nor starts. It is not exposed through Wails and there
is no permanent CLI command that invokes it. There are two ways to verify it
locally.

### Run tests

From the repository root:

```bash
go test ./backend/endpoints/catalog -run '^TestGetItemVariants' -count=1 -v
```

The suite covers a valid `(kind, key)` pair returning every stored variant, full
variant data and provenance, the catalog order, an item without variants
returning an empty non-nil array, the JSON contract of the result, a `nil`
catalog, an empty kind, an empty key, a whitespace-only kind and key, values with
leading or trailing whitespace, an unsupported kind, an unknown key, a lowercase
key, the pre-migration prefixed key, a numeric string and a numeric `GameID`
passed as a string, the four distinguishable kind and key failures, and the
immutability of the catalog after the returned variants are mutated.

### Print the real getter output

The following runs the real getter against the real `backend/gamecatalog/data`.
It writes a temporary Go program outside the repository, runs it, and deletes it
afterwards. It is written for Bash or Zsh on macOS and Linux. Run it from the
repository root:

```bash
(
    set -eu

    resource_kind="item"
    resource_key="000F4240"

    variants_demo_dir=$(mktemp -d)
    trap 'rm -rf -- "$variants_demo_dir"' EXIT

    cat > "$variants_demo_dir/main.go" <<'EOF'
package main

import (
    "encoding/json"
    "log"
    "os"

    "github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
    "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
    "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
)

func main() {
    if len(os.Args) != 3 {
        log.Fatalf("usage: %s <kind> <key>", os.Args[0])
    }

    data, err := loader.LoadDir("backend/gamecatalog/data")
    if err != nil {
        log.Fatalf("load catalog data: %v", err)
    }

    gameCatalog, err := gamecatalog.New(data.Manifest, data.Resources())
    if err != nil {
        log.Fatalf("build catalog: %v", err)
    }

    variants, err := catalog.GetItemVariants(gameCatalog, os.Args[1], os.Args[2])
    if err != nil {
        log.Fatalf("GetItemVariants: %v", err)
    }

    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(variants); err != nil {
        log.Fatalf("encode result: %v", err)
    }
}
EOF

    go run "$variants_demo_dir/main.go" "$resource_kind" "$resource_key"
)
```

`resource_kind` and `resource_key` must be a real `(kind, key)` pair present in
the catalog data. `kind=item` with `key=000F4240` is the Dagger and exists in the
current data; any other existing pair works the same way. `kind=item` with
`key=8000EA60` is the Determination Ash of War and prints `{"variants": []}`. An
unknown kind or key makes the program exit with the corresponding error of the
getter.

The block runs in a subshell so `set -eu` does not leak into the calling shell. A
failure of `mktemp`, of writing the program, or of `go run` aborts the block and
produces a non-zero exit code. The `trap` removes the temporary directory on
every exit path, including failures, and because it runs during exit it does not
overwrite the exit code of the failing command.

The program is temporary on purpose. No demonstration program or helper script
for this endpoint is kept in the repository.

The real output is large, because every variant carries its own document data,
parameter records, and provenance entries. Its exact size depends on the catalog
data and changes when that data is regenerated. The example below is **heavily
abbreviated**; `…` marks omitted fields and entries:

```json
{
  "variants": [
    {
      "gameID": { "known": true, "value": 1000100, "provenance": { "source": "regulation_equip_param_weapon", "…": "…" } },
      "kind": { "known": true, "value": "affinity", "provenance": { "…": "…" } },
      "affinity": { "known": true, "value": "heavy", "provenance": { "…": "…" } },
      "upgradeLevel": { "known": true, "value": 0, "provenance": { "source": "regulation_equip_param_weapon", "…": "…" } },
      "sourceRowID": { "known": true, "value": 1000100, "provenance": { "…": "…" } },
      "data": { "presentation": { "name": { "known": true, "value": "Heavy Dagger", "…": "…" } }, "…": "…" },
      "sourceRecords": [ { "table": "EquipParamWeapon", "rowID": 1000100, "…": "…" } ]
    },
    "…"
  ]
}
```

`variants` is the only top-level key. The base item does not appear in the array.

A successful run prints the variants and exits without an error.

## Current limitations

- The endpoint is not exposed through Wails.
- The only HTTP route is `GET /api/v1/catalog/item-variants` of the local OpenAPI
  explorer in `backend/swagger`, a developer tool.
- There is no permanent CLI command for it.
- There is no caller in the runtime of the main application.
- The getter does not load the catalog. It requires an already loaded
  `*gamecatalog.Catalog` supplied by the caller.
- The getter never reads a save and never uses `SaveEngine`.
- Lookup is by the exact `(kind, key)` pair only. There is no numeric resource
  identifier any more, and there is no lookup by `GameID`, by name, or by
  variant.
- There is no filtering, sorting, searching, or pagination of the variants.
- Variants are returned as stored. The endpoint does not materialise a variant
  into a standalone item document or resource.
- The base item is never included. Reading it belongs to `GetResource`.
