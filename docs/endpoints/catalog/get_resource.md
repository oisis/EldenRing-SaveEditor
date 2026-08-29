# GetResource

## Overview

`GetResource` returns the complete document of exactly one resource stored in the
already loaded GameCatalog, whatever its kind: an `ItemDocument` for kind `item`
with its capabilities, variants, presentation and provenance, a
`ColosseumDocument` for kind `colosseum` with its name, its unlock event flag ID
and the provenance of both, a `RegionDocument` for kind `region` with its
internal region ID, name and area, a `SummoningPoolDocument` for kind
`summoning_pool` with its name, curated region label and activation event flag
ID, a `GraceDocument` for kind `grace` with its name, curated region label,
visit event flag ID, boss-arena fact, dungeon type and door event flag ID,
a `BossDocument` for kind `boss` with its name, curated region label, encounter
type, remembrance fact and synchronized defeat event flag ID,
a `MapRegionDocument` for kind `map_region` with its name, area label and safe
visibility event flag ID, a `TutorialDocument` for kind `tutorial` with its
`TutorialParam` row ID and official title, a `QuestDocument` for kind `quest`
with its name, supported steps, locations, descriptions and canonical event
flag plans, or a `ClassDocument` for kind `class` with its starting-class ID,
official name, base Rune Level and eight base attributes. It returns no relations; those belong to
[`GetResourceRelations`](get_resource_relations.md).

| | |
|---|---|
| EndpointID | `get_resource` |
| Kind | Getter |
| Domain | `catalog` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/catalog/resource` of the local OpenAPI explorer (`tools/swagger`). No Wails binding and no permanent CLI command reach it. |
| Implementation source | [../../../backend/endpoints/catalog/get_resource.go](../../../backend/endpoints/catalog/get_resource.go) |
| Test source | [../../../backend/endpoints/catalog/get_resource_test.go](../../../backend/endpoints/catalog/get_resource_test.go) |
| Save access | none — the endpoint never opens, reads, or writes a save |
| Mutation | none — the endpoint reads catalog copies and modifies nothing |

## Input

The public contract of this endpoint has exactly two input parameters, `kind`
and `key`.

The Go signature is:

```go
func GetResource(gameCatalog *gamecatalog.Catalog, kind string, key string) (GetResourceResult, error)
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

The lookup resolves the kind first and the key only inside that kind:

- `kind` is matched exactly against `Resource.Kind`. `item`, `colosseum`,
  `region`, `summoning_pool`, `grace`, `boss`, `map_region`, `tutorial`,
  `quest` and `class` are the kinds the current schema supports.
- `key` is matched exactly against `Resource.Key` inside the resolved kind. The
  same key may later exist under a different kind, so the key alone is not an
  identity.
- Neither value is trimmed, case-folded, parsed, or retried under another kind.
  An input with surrounding whitespace is reported as unknown rather than
  repaired.
- The pre-migration key form `item:000F4240` carried the kind as a prefix. It is
  now an unknown key and never an alias of `000F4240`.

`schema.ValidateResource` requires an item key to be exactly eight uppercase
hexadecimal characters (`0-9`, `A-F`), so `000F4240` is well formed and
`000f4240` is not. Colosseum, region, summoning pool, grace, boss, map region
and quest keys use lowercase letters, digits and underscores, for example
`royal_colosseum`, `limgrave_the_first_step`,
`stormveil_castle_gateside_chamber`,
`weeping_peninsula_tombsward_catacombs` and `brother_corhyn`. A tutorial key is
the decimal form of its `TutorialParam` row ID, for example `2010`. A class key
is the decimal starting-class ID `0`..`11`, for example `0` or `11`.
`gamecatalog.New` rejects a catalog containing the same `(kind, key)` pair
twice, so at most one resource can match.

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
| `key` | `string` | The stable `Resource.Key` the lookup matched, for an item eight uppercase hexadecimal characters. |
| `kind` | `string` | Resource kind, `item`, `colosseum`, `region`, `summoning_pool`, `grace`, `boss`, `map_region`, `tutorial`, `quest` or `class`. |
| `item` | `ItemDocument` | The complete item document. Present only for kind `item`. |
| `colosseum` | `ColosseumDocument` | The complete colosseum document. Present only for kind `colosseum`. |
| `region` | `RegionDocument` | The complete curated region document. Present only for kind `region`. |
| `summoningPool` | `SummoningPoolDocument` | The complete curated summoning pool document. Present only for kind `summoning_pool`. |
| `grace` | `GraceDocument` | The complete curated Site of Grace document. Present only for kind `grace`. |
| `boss` | `BossDocument` | The complete curated boss encounter document. Present only for kind `boss`. |
| `mapRegion` | `MapRegionDocument` | The complete curated safe map visibility document. Present only for kind `map_region`. |
| `tutorial` | `TutorialDocument` | The complete user-facing tutorial document. Present only for kind `tutorial`. |
| `quest` | `QuestDocument` | The complete curated quest document. Present only for kind `quest`. |
| `class` | `ClassDocument` | The complete playable starting class document: starting-class ID, name, base Rune Level (`level`, the `CharaInitParam` `soulLv` fact, never derived from the attribute sum) and the eight base attributes, each with its own provenance. Present only for kind `class`. |

`schema.Resource` is a union over those kinds: exactly one document field is
present and the others are omitted from the JSON entirely.

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
   manifest and the resources and indexes every resource by its kind and, inside
   that kind, by its `Resource.Key`. A catalog that fails any of those checks is
   never constructed.
3. The caller passes that catalog, a `kind` and a `key` to `GetResource`. The
   getter never loads, reloads, or rescans anything.
4. The getter validates `gameCatalog`, `kind` and `key`.
5. The getter resolves the pair through `Catalog.ResourceByKindAndKey`, which
   selects the map of the requested kind and then the key inside it, using the
   two-level index built once during `gamecatalog.New`. There is no directory
   scan, no JSON read, and no linear search per call.
6. The catalog returns that one resource as an independent deep copy.
7. The getter puts the resource into `GetResourceResult` and returns it.

## Validation and errors

Every failure returns an empty `GetResourceResult` together with the error. A
partial result is never returned alongside an error.

| Condition | Error |
|---|---|
| `gameCatalog` is `nil` | `game catalog is not loaded` |
| `kind` is empty | `resource kind is required` |
| `kind` matches no catalog kind | `unknown resource kind "…"` |
| `key` is empty | `resource key is required` |
| `key` matches no `Resource.Key` inside the resolved kind | `unknown resource key "…" in kind "…"` |

Notes:

- A missing kind, an unknown kind, a missing key and a key that is unknown inside
  an existing kind are four distinguishable errors.
- The unknown-key error names both the key and the kind it was searched in.
- Whitespace is never trimmed. `" 000F4240"` is reported as an unknown key, not
  as a silent lookup of `"000F4240"`.
- The pre-migration key `"item:000F4240"` is reported as an unknown key under
  `kind=item`. There is no backward compatibility with the prefixed form.
- A numeric string such as `"1"` or `"1000000"` is not a `Resource.Key` and is
  reported as unknown. The `GameID` is not accepted as input.
- A zero-value `gamecatalog.Catalog` that never went through `gamecatalog.New`
  has no kind index, so every lookup against it reports an unknown kind.
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

`GetResource` is exposed over HTTP as `GET /api/v1/catalog/resource` by the local
OpenAPI explorer in `tools/swagger`, a developer tool the application
neither imports nor starts. It is not exposed through Wails and there is no
permanent CLI command that invokes it. There are two ways to verify it locally.

### Run tests

From the repository root:

```bash
go test ./backend/endpoints/catalog -run '^TestGetResource' -count=1 -v
```

The suite covers a valid `(kind, key)` pair returning the expected resource, the
full item document with its variants, capabilities, presentation, and provenance,
the JSON contract of the result, a `nil` catalog, an empty kind, an empty key, a
whitespace-only kind and key, values with leading or trailing whitespace, an
unknown kind, an unknown key, a lowercase key, the pre-migration prefixed key, a
numeric string and a numeric `GameID` passed as a string, the four distinguishable
kind and key failures, and the immutability of the returned result. They also
cover colosseum, region, summoning pool, grace, boss, map region, tutorial,
quest and class resources: their complete typed documents and
provenance, the absence of documents from other kinds, independent returned
copies, and JSON bodies containing only the matching union field.

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

    resource, err := catalog.GetResource(gameCatalog, os.Args[1], os.Args[2])
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

    go run "$resource_demo_dir/main.go" "$resource_kind" "$resource_key"
)
```

`resource_kind` and `resource_key` must be a real `(kind, key)` pair present in
the catalog data. `kind=item` with `key=000F4240` is the Dagger and exists in the
current data; any other existing pair works the same way. An unknown kind or key
makes the program exit with the corresponding error of the getter.

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
    "key": "000F4240",
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
- The only HTTP route is `GET /api/v1/catalog/resource` of the local OpenAPI
  explorer in `tools/swagger`, a developer tool.
- There is no permanent CLI command for it.
- There is no caller in the runtime of the main application.
- The getter does not load the catalog. It requires an already loaded
  `*gamecatalog.Catalog` supplied by the caller.
- The getter never reads a save and never uses `SaveEngine`.
- Lookup is by the exact `(kind, key)` pair only. There is no numeric resource
  identifier any more, and there is no lookup by `GameID`, by name, or by
  variant.
- There is no filtering, searching, or pagination. Listing resources belongs to
  `GetResources`, which is not implemented yet.
- The endpoint returns no relations. Neither outgoing nor incoming relations are
  part of the result, and it never returns the documents of related resources.
  Relations, together with `relationType` and `direction` filtering, belong to
  [`GetResourceRelations`](get_resource_relations.md).
- Variants are returned inside the item document as stored. The endpoint does not
  materialise a variant into a standalone resource.
