# GetResource

## Overview

`GetResource` returns the complete document of exactly one resource stored in the
already loaded GameCatalog, including its capabilities, variants, presentation,
and provenance. It returns no relations; those belong to `GetResourceRelations`,
which is not implemented yet.

| | |
|---|---|
| EndpointID | `get_resource` |
| Kind | Getter |
| Domain | `catalog` |
| Implementation status | implemented |
| Transport status | not exposed — no Wails binding, HTTP route, or permanent CLI command reaches it |
| Implementation source | [../../../backend/endpoints/catalog/get_resource.go](../../../backend/endpoints/catalog/get_resource.go) |
| Test source | [../../../backend/endpoints/catalog/get_resource_test.go](../../../backend/endpoints/catalog/get_resource_test.go) |
| Save access | none — the endpoint never opens, reads, or writes a save |
| Mutation | none — the endpoint reads catalog copies and modifies nothing |

## Input

The public contract of this endpoint has exactly one input parameter,
`resourceID`.

The Go signature is:

```go
func GetResource(gameCatalog *gamecatalog.Catalog, resourceID string) (GetResourceResult, error)
```

### `resourceID` is the exact `Resource.Key`

`resourceID` is the stable `schema.Resource.Key` of the wanted resource, for
example `item:000F4240`.

It is **not**:

- the numeric `schema.ResourceID` (`Resource.ID`);
- the item's `Item.GameID`;
- a variant ID;
- the name of the JSON file the resource was loaded from.

The lookup is an exact string match against `Resource.Key`:

- The endpoint declares no key format of its own, because `schema` declares none
  either. `resource_validation.go` requires only that `Key` is non-empty.
- The endpoint never parses the key, never splits it into a kind and an ID, and
  never derives a `GameID` from it. The current data happens to use
  `item:<uppercase-hex-game-id>`, but that shape is a property of the generated
  data, not a contract this endpoint enforces or relies on.
- The key is never normalised. Case is not folded and whitespace is not trimmed;
  an input with surrounding whitespace is rejected rather than repaired.

`Resource.Key` is the single source of truth for this lookup. `gamecatalog.New`
already rejects a catalog with duplicate keys, so at most one resource can match.

### The catalog argument

`*gamecatalog.Catalog` is a backend dependency, not a transport parameter:

- It is supplied by the backend caller, not by a client.
- The caller has to load and build the catalog itself before calling the getter.
  `GetResource` never does that on its own.
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
type GetResourceResult struct {
	Resource schema.Resource `json:"resource"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `resource` | `schema.Resource` | The complete resource document stored in the catalog. |

`resource` is the only top-level field. The result carries no relations of any
kind.

`resource` is the full `schema.Resource`, not a reduced projection:

| Field | Type | Meaning |
|---|---|---|
| `id` | `uint32` | Numeric `ResourceID` inside the loaded catalog. It is an internal index, not the public `resourceID` parameter. |
| `key` | `string` | The stable `Resource.Key` the lookup matched. |
| `kind` | `string` | Resource kind. `item` is the only kind the current schema supports. |
| `item` | `ItemDocument` | The complete item document. |

The `item` document is returned whole, exactly as `schema.ItemDocument` defines
it, including `presentation`, `capabilities`, `safety`, `storage`, `acquisition`,
`modifiers`, `links`, `variants`, `aliases`, `unlocks`,
`relatedTechnicalRecords`, `sourceRecords`, and the family-specific block
(`weapon`, `armor`, `talisman`, `ashOfWar`, `spell`, `spiritAsh`, `goods`, or
`gesture`). Provenance is preserved: every `Fact` keeps its `provenance` record,
and so do variants, aliases, and parameter records. The endpoint defines no
second copy of the schema models and flattens nothing into new DTOs.

## Processing flow

This is what a caller that wants to invoke `GetResource` has to do. It is not a
sequence the main application performs today.

1. The caller loads the catalog data through `loader.LoadDir`, for example from
   `backend/gamecatalog/data`.
2. The caller builds the catalog through `gamecatalog.New`, which validates the
   manifest and the resources and indexes every resource by its `Resource.Key`. A
   catalog that fails any of those checks is never constructed.
3. The caller passes that catalog and a `resourceID` to `GetResource`. The getter
   never loads, reloads, or rescans anything.
4. The getter validates `gameCatalog` and `resourceID`.
5. The getter resolves the key through `Catalog.ResourceByKey`, which reads the
   key index built once during `gamecatalog.New`. There is no directory scan, no
   JSON read, and no linear search per call.
6. The catalog returns that one resource as an independent deep copy.
7. The getter puts the resource into `GetResourceResult` and returns it.

## Validation and errors

Every failure returns an empty `GetResourceResult` together with the error. A
partial result is never returned alongside an error.

| Condition | Error |
|---|---|
| `gameCatalog` is `nil` | `game catalog is not loaded` |
| `resourceID` is empty | `resource ID is required` |
| `resourceID` is whitespace only | `resource ID "…" must not contain leading or trailing whitespace` |
| `resourceID` has leading or trailing whitespace | `resource ID "…" must not contain leading or trailing whitespace` |
| `resourceID` matches no `Resource.Key` | `resource ID "…" was not found in the game catalog` |

Notes:

- The not-found error names the exact `resourceID` that was not found.
- Whitespace is rejected, never trimmed. `" item:000F4240"` is an error, not a
  silent lookup of `"item:000F4240"`.
- A numeric string such as `"1"` or `"1000000"` is not a `Resource.Key` and is
  reported as not found. The numeric `ResourceID` and the `GameID` are not
  accepted as input.
- A zero-value `gamecatalog.Catalog` that never went through `gamecatalog.New`
  has no key index, so every lookup against it is reported as not found.
- This endpoint does not use the shared `EndpointError` type; that type does not
  exist yet.

## Result immutability

The result is safe to modify. `GetResource` returns what the catalog's existing
query layer produced, and that layer already deep-copies the resource, its item
document, its variants, its capability rule slices, and its parameter records.
Mutating any part of `GetResourceResult` — including a variant or an affinity
slice — does not change the catalog, and a later call returns the original data.
The getter itself modifies nothing: it does not add, remove, repair, or normalise
catalog data.

## Command-line verification

`GetResource` is not exposed through Wails, HTTP, or a permanent CLI command, so
there is no `curl` call or application command that invokes it. It is currently
reachable only as a Go function. There are two ways to verify it locally.

### Run tests

From the repository root:

```bash
go test ./backend/endpoints/catalog -run '^TestGetResource' -count=1 -v
```

The suite covers a valid key returning the expected resource, the full item
document with its variants, capabilities, presentation, and provenance, the JSON
contract of the result, a `nil` catalog, an empty key, a whitespace-only key,
keys with leading or trailing whitespace, an unknown key, a numeric `ResourceID`
and a numeric `GameID` passed as strings, and the immutability of the returned
result.

### Print the real getter output

The following runs the real getter against the real `backend/gamecatalog/data`.
It writes a temporary Go program outside the repository, runs it, and deletes it
afterwards. It is written for Bash or Zsh on macOS and Linux. Run it from the
repository root:

```bash
(
    set -eu

    resource_key="item:000F4240"

    resource_demo_dir=$(mktemp -d)
    trap 'rm -rf -- "$resource_demo_dir"' EXIT

    cat > "$resource_demo_dir/main.go" <<'EOF'
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

    resource, err := catalog.GetResource(gameCatalog, os.Args[1])
    if err != nil {
        log.Fatalf("GetResource: %v", err)
    }

    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(resource); err != nil {
        log.Fatalf("encode result: %v", err)
    }
}
EOF

    go run "$resource_demo_dir/main.go" "$resource_key"
)
```

`resource_key` must be a real `Resource.Key` present in the catalog data.
`item:000F4240` is the Dagger and exists in the current data; any other existing
key works the same way. An unknown key makes the program exit with the
not-found error of the getter.

The block runs in a subshell so `set -eu` does not leak into the calling shell. A
failure of `mktemp`, of writing the program, or of `go run` aborts the block and
produces a non-zero exit code. The `trap` removes the temporary directory on
every exit path, including failures, and because it runs during exit it does not
overwrite the exit code of the failing command.

The program is temporary on purpose. No demonstration program or helper script
for this endpoint is kept in the repository.

The real output is large, because the resource document contains every variant,
every parameter record, and every provenance entry. Its exact size depends on the
catalog data and changes when that data is regenerated. The example below is
**heavily abbreviated**; `…` marks omitted fields and entries:

```json
{
  "resource": {
    "id": 2,
    "key": "item:000F4240",
    "kind": "item",
    "item": {
      "gameID": { "known": true, "value": 1000000, "provenance": { "source": "<source-id>", "method": "<method>" } },
      "family": { "known": true, "value": "weapon", "provenance": { "…": "…" } },
      "presentation": { "name": { "known": true, "value": "Dagger", "provenance": { "…": "…" } }, "…": "…" },
      "capabilities": { "upgrade": { "known": true, "enabled": true, "rules": { "maxLevel": 25, "…": "…" } }, "…": "…" },
      "variants": [ { "gameID": { "known": true, "value": 1000100, "…": "…" }, "…": "…" } ],
      "sourceRecords": [ { "table": "EquipParamWeapon", "rowID": 1000000, "…": "…" } ],
      "…": "…"
    }
  }
}
```

`resource` is the only top-level key. Neither `outgoingRelations`,
`incomingRelations`, nor `relatedResources` appears in the output.

A successful run prints the resource and exits without an error.

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
- There is no filtering, searching, or pagination. Listing resources belongs to
  `GetResources`, which is not implemented yet.
- The endpoint returns no relations. Neither outgoing nor incoming relations are
  part of the result, and it never returns the documents of related resources.
  Relations, together with `relationType` and `direction` filtering, belong to
  `GetResourceRelations`, which is not implemented yet.
- Variants are returned inside the item document as stored. The endpoint does not
  materialise a variant into a standalone resource.
