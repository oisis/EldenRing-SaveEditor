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
| Transport status | not exposed — no Wails binding, HTTP route, or permanent CLI command reaches it |
| Implementation source | [../../../backend/endpoints/catalog/get_item_variants.go](../../../backend/endpoints/catalog/get_item_variants.go) |
| Test source | [../../../backend/endpoints/catalog/get_item_variants_test.go](../../../backend/endpoints/catalog/get_item_variants_test.go) |
| Save access | none — the endpoint never opens, reads, or writes a save |
| Mutation | none — the endpoint reads catalog copies and modifies nothing |

## Input

The public contract of this endpoint has exactly one input parameter,
`resourceID`.

The Go signature is:

```go
func GetItemVariants(gameCatalog *gamecatalog.Catalog, resourceID string) (GetItemVariantsResult, error)
```

### `resourceID` is the exact `Resource.Key`

`resourceID` is the stable `schema.Resource.Key` of the resource whose variants
are wanted, for example `item:000F4240`.

It is **not**:

- the numeric `schema.ResourceID` (`Resource.ID`);
- the item's `Item.GameID`;
- a variant ID;
- the name of the JSON file the resource was loaded from.

The lookup is an exact string match against `Resource.Key`, performed directly
through `Catalog.ResourceByKey`:

- The endpoint declares no key format of its own, because `schema` declares none
  either. `resource_validation.go` requires only that `Key` is non-empty.
- The endpoint never parses the key, never splits it into a kind and an ID, and
  never derives a `GameID` from it. The current data happens to use
  `item:<uppercase-hex-game-id>`, but that shape is a property of the generated
  data, not a contract this endpoint enforces or relies on.
- The key is never normalised. Case is not folded and whitespace is not trimmed;
  an input with surrounding whitespace is rejected rather than repaired.
- A variant is never addressable as input. There is no lookup of a single variant
  by its `GameID`, its affinity, or its upgrade level.

`Resource.Key` is the single source of truth for this lookup. `gamecatalog.New`
already rejects a catalog with duplicate keys, so at most one resource can match.

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
| `upgradeLevel` | `Fact[uint8]` | The upgrade level of the variant, when it has one. |
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
Determination Ash of War (`item:8000EA60`) is such an item.

## Processing flow

This is what a caller that wants to invoke `GetItemVariants` has to do. It is not
a sequence the main application performs today.

1. The caller loads the catalog data through `loader.LoadDir`, for example from
   `backend/gamecatalog/data`.
2. The caller builds the catalog through `gamecatalog.New`, which validates the
   manifest and the resources and indexes every resource by its `Resource.Key`. A
   catalog that fails any of those checks is never constructed.
3. The caller passes that catalog and a `resourceID` to `GetItemVariants`. The
   getter never loads, reloads, or rescans anything.
4. The getter validates `gameCatalog` and `resourceID`.
5. The getter resolves the key through `Catalog.ResourceByKey`, which reads the
   key index built once during `gamecatalog.New`. There is no directory scan, no
   JSON read, and no linear search per call. The getter calls no other endpoint;
   in particular it does not call `GetResource`.
6. The catalog returns that one resource as an independent deep copy.
7. The getter takes `Item.Variants` of that copy, substitutes an empty slice for
   a missing one, puts it into `GetItemVariantsResult`, and returns it.

## Validation and errors

Every failure returns an empty `GetItemVariantsResult` together with the error. A
partial result is never returned alongside an error.

| Condition | Error |
|---|---|
| `gameCatalog` is `nil` | `game catalog is not loaded` |
| `resourceID` is empty | `resource ID is required` |
| `resourceID` is whitespace only | `resource ID "…" must not contain leading or trailing whitespace` |
| `resourceID` has leading or trailing whitespace | `resource ID "…" must not contain leading or trailing whitespace` |
| `resourceID` matches no `Resource.Key` | `resource ID "…" was not found in the game catalog` |
| the resolved resource carries no `ItemDocument` | `resource ID "…" is not an item and has no variants` |

Notes:

- The not-found error names the exact `resourceID` that was not found.
- Whitespace is rejected, never trimmed. `" item:000F4240"` is an error, not a
  silent lookup of `"item:000F4240"`.
- A numeric string such as `"1"` or `"1000000"` is not a `Resource.Key` and is
  reported as not found. The numeric `ResourceID` and the `GameID` are not
  accepted as input.
- A zero-value `gamecatalog.Catalog` that never went through `gamecatalog.New`
  has no key index, so every lookup against it is reported as not found.
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

`GetItemVariants` is not exposed through Wails, HTTP, or a permanent CLI command,
so there is no `curl` call or application command that invokes it. It is
currently reachable only as a Go function. There are two ways to verify it
locally.

### Run tests

From the repository root:

```bash
go test ./backend/endpoints/catalog -run '^TestGetItemVariants' -count=1 -v
```

The suite covers a valid key returning every stored variant, full variant data
and provenance, the catalog order, an item without variants returning an empty
non-nil array, the JSON contract of the result, a `nil` catalog, an empty key, a
whitespace-only key, keys with leading or trailing whitespace, an unknown key, a
numeric `ResourceID` and a numeric `GameID` passed as strings, and the
immutability of the catalog after the returned variants are mutated.

### Print the real getter output

The following runs the real getter against the real `backend/gamecatalog/data`.
It writes a temporary Go program outside the repository, runs it, and deletes it
afterwards. It is written for Bash or Zsh on macOS and Linux. Run it from the
repository root:

```bash
(
    set -eu

    resource_key="item:000F4240"

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
    if len(os.Args) != 2 {
        log.Fatalf("usage: %s <resourceID>", os.Args[0])
    }

    data, err := loader.LoadDir("backend/gamecatalog/data")
    if err != nil {
        log.Fatalf("load catalog data: %v", err)
    }

    gameCatalog, err := gamecatalog.New(data.Manifest, data.Resources())
    if err != nil {
        log.Fatalf("build catalog: %v", err)
    }

    variants, err := catalog.GetItemVariants(gameCatalog, os.Args[1])
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

    go run "$variants_demo_dir/main.go" "$resource_key"
)
```

`resource_key` must be a real `Resource.Key` present in the catalog data.
`item:000F4240` is the Dagger and exists in the current data; any other existing
key works the same way. `item:8000EA60` is the Determination Ash of War and
prints `{"variants": []}`. An unknown key makes the program exit with the
not-found error of the getter.

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
      "upgradeLevel": { "known": false, "value": 0, "provenance": { "…": "…" } },
      "sourceRowID": { "known": true, "value": 1000100, "provenance": { "…": "…" } },
      "data": { "presentation": { "displayName": { "known": true, "value": "Dagger", "…": "…" } }, "…": "…" },
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
- There is no HTTP endpoint for it.
- There is no permanent CLI command for it.
- There is no caller in the runtime of the main application.
- The getter does not load the catalog. It requires an already loaded
  `*gamecatalog.Catalog` supplied by the caller.
- The getter never reads a save and never uses `SaveEngine`.
- Lookup is by exact `Resource.Key` only. There is no lookup by `ResourceID`, by
  `GameID`, by name, or by variant.
- There is no filtering, sorting, searching, or pagination of the variants.
- Variants are returned as stored. The endpoint does not materialise a variant
  into a standalone item document or resource.
- The base item is never included. Reading it belongs to `GetResource`.
