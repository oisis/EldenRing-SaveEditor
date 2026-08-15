# GetCatalogInfo

## Overview

`GetCatalogInfo` reports the manifest of the already loaded GameCatalog: the
schema version, the data version, the game version, whether the manifest is
valid, and the list of data sources the catalog was built from.

| | |
|---|---|
| EndpointID | `get_catalog_info` |
| Kind | Getter |
| Domain | `catalog` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/catalog/info` of the local OpenAPI explorer (`tools/swagger`). No Wails binding and no permanent CLI command reach it. |
| Implementation source | [../../../backend/endpoints/catalog/get_catalog_info.go](../../../backend/endpoints/catalog/get_catalog_info.go) |
| Test source | [../../../backend/endpoints/catalog/get_catalog_info_test.go](../../../backend/endpoints/catalog/get_catalog_info_test.go) |
| Save access | none — the endpoint never opens, reads, or writes a save |
| Mutation | none — the endpoint reads a manifest copy and modifies nothing |

## Input

The public contract of this endpoint has no input parameters. There is no
filter, no pagination, and no identifier to resolve.

The Go signature takes a `*gamecatalog.Catalog`:

```go
func GetCatalogInfo(gameCatalog *gamecatalog.Catalog) (GetCatalogInfoResult, error)
```

That argument is a backend dependency, not a transport parameter:

- `*gamecatalog.Catalog` is supplied by the backend caller, not by a client.
- The current caller has to load and build the catalog itself before calling
  the getter. `GetCatalogInfo` never does that on its own.
- No caller exists in the runtime of the main application today. The endpoint is
  currently invoked only outside that runtime:
  - the unit tests call the getter directly with catalogs built for the test,
    not with the shipped catalog data;
  - the command-line example in
    [Print the real getter output](#print-the-real-getter-output) calls it
    directly with the real `backend/gamecatalog/data`.
- Once the endpoint is wired into a runtime, that runtime becomes responsible
  for owning the loaded catalog and passing it in.
- The frontend never passes this parameter directly and never influences which
  catalog is inspected.

## Output

The endpoint returns a typed result:

```go
type GetCatalogInfoResult struct {
	SchemaVersion uint32              `json:"schemaVersion"`
	DataVersion   string              `json:"dataVersion"`
	GameVersion   string              `json:"gameVersion"`
	Valid         bool                `json:"valid"`
	Sources       []schema.DataSource `json:"sources"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `schemaVersion` | `uint32` | Version of the GameCatalog schema the loaded data conforms to. Accepted only inside the range supported by `schema.ValidateManifest` (`schema.MinimumSchemaVersion` to `schema.CurrentSchemaVersion`). |
| `dataVersion` | `string` | Version identifier of the generated catalog data. In the shipped catalog this is a content hash produced by the data generator, not a human-readable release number. |
| `gameVersion` | `string` | The version label recorded by the catalog generator for the supplied regulation dump. It may be a game version, a version class, or any other dataset identifier — it is not guaranteed to be an official Elden Ring release number. The endpoint returns the stored label verbatim, without interpreting, parsing, or modifying it. |
| `valid` | `boolean` | `true` only when `schema.ValidateManifest` accepted the manifest. It is never returned as `false`: a rejected manifest produces an error and an empty result instead. |
| `sources` | array of `DataSource` | The manifest of every data source the catalog was built from, in manifest order. |

Each `DataSource` entry describes one input the catalog data was derived from:

| Field | Type | Meaning |
|---|---|---|
| `id` | `string` | Stable source identifier referenced by resource provenance records. Unique within the manifest. |
| `kind` | `string` | Category of the source, for example a regulation parameter CSV, a game text extract, or a legacy curated catalog. |
| `location` | `string` | Logical location of the source. It is a provenance label validated by `schema.ValidateManifest`, not a guaranteed path on the current machine. |
| `version` | `string` | Version identifier of that specific source. In the shipped catalog these are content hashes of the extracted input. |
| `evidence` | `string` | Evidence level of the source: `regulation`, `game_data`, `verified_research`, `curated`, or `unknown`. |
| `reviewed` | `boolean` | Whether the source has been reviewed by a human. |

## Processing flow

This is what a caller that wants to invoke `GetCatalogInfo` has to do. It is not
a sequence the main application performs at startup today.

1. The caller loads the catalog data through `loader.LoadDir`, for example from
   `backend/gamecatalog/data`.
2. The caller builds the catalog through `gamecatalog.New`, which validates the
   manifest, the resources, and the relations. A catalog that fails any of those
   checks is never constructed.
3. The caller passes the resulting catalog to `GetCatalogInfo`. The getter never
   loads or reloads the catalog itself.
4. The getter takes a copy of the manifest through `Catalog.Manifest()`.
5. The manifest copy is checked with the existing `schema.ValidateManifest`
   validator. Because `gamecatalog.Catalog` is an exported type, a caller can
   pass a zero-value catalog that never went through `gamecatalog.New`, so the
   manifest is validated instead of trusted.
6. The getter returns a typed `GetCatalogInfoResult`.
7. The getter modifies nothing. The result holds a copy, so mutating the
   returned struct — including its `sources` entries — does not affect the
   catalog.

## Validation and errors

- A `nil` catalog returns the error `game catalog is not loaded` and an empty
  result.
- An empty or otherwise invalid manifest returns an error containing
  `game catalog manifest is invalid`, wrapping the exact reason reported by
  `schema.ValidateManifest`, and an empty result.
- `valid=true` is returned only after `schema.ValidateManifest` accepted the
  manifest. It is not a claim about the catalog's resources or relations.
- The endpoint defines no validation rules of its own. Every rule it enforces
  comes from `schema.ValidateManifest`, which stays the single source of truth
  for manifest validity.
- Invalid data is never repaired or normalised. A rejected manifest produces an
  error; it is never silently corrected, defaulted, or partially returned.

## Command-line verification

`GetCatalogInfo` is exposed over HTTP as `GET /api/v1/catalog/info` by the local
OpenAPI explorer in `tools/swagger`, a developer tool the application
neither imports nor starts. It is not exposed through Wails and there is no
permanent CLI command that invokes it. There are two ways to verify it
locally.

### Run tests

From the repository root:

```bash
go test ./backend/endpoints/catalog -run '^TestGetCatalogInfo' -count=1 -v
```

The suite covers four cases: a valid manifest returns the expected fields with
`valid=true`, a `nil` catalog returns an error, a catalog whose manifest was
rejected by `schema.ValidateManifest` returns an error and an empty result, and
mutating the returned result does not mutate the catalog.

### Print the real getter output

The following runs the real getter against the real `backend/gamecatalog/data`.
It writes a temporary Go program outside the repository, runs it, and deletes it
afterwards. It is written for Bash or Zsh on macOS and Linux. Run it from the
repository root:

```bash
(
    set -eu

    catalog_demo_dir=$(mktemp -d)
    trap 'rm -rf -- "$catalog_demo_dir"' EXIT

    cat > "$catalog_demo_dir/main.go" <<'EOF'
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
    data, err := loader.LoadDir("backend/gamecatalog/data")
    if err != nil {
        log.Fatalf("load catalog data: %v", err)
    }

    gameCatalog, err := gamecatalog.New(data.Manifest, data.Resources())
    if err != nil {
        log.Fatalf("build catalog: %v", err)
    }

    info, err := catalog.GetCatalogInfo(gameCatalog)
    if err != nil {
        log.Fatalf("GetCatalogInfo: %v", err)
    }

    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(info); err != nil {
        log.Fatalf("encode result: %v", err)
    }
}
EOF

    go run "$catalog_demo_dir/main.go"
)
```

The block runs in a subshell so `set -eu` does not leak into the calling shell.
A failure of `mktemp`, of writing the program, or of `go run` aborts the block
and produces a non-zero exit code. The `trap` removes the temporary directory on
every exit path, including failures, and because it runs during exit it does not
overwrite the exit code of the failing command.

The program is temporary on purpose. No demonstration program or helper script
for this endpoint is kept in the repository.

The command prints the full result, including **every** source in the current
manifest. The example below is abbreviated to a single source, and its values
are placeholders — the real output contains the current versions and hashes,
which change whenever the catalog data is regenerated:

```json
{
  "schemaVersion": 9,
  "dataVersion": "<current-data-hash>",
  "gameVersion": "<current-game-version>",
  "valid": true,
  "sources": [
    {
      "id": "<source-id>",
      "kind": "<source-kind>",
      "location": "<logical-source-location>",
      "version": "<source-version>",
      "evidence": "<evidence-level>",
      "reviewed": true
    }
  ]
}
```

A successful run reports `"valid": true` and exits without an error.

## Current limitations

- The endpoint is not exposed through Wails.
- The only HTTP route is `GET /api/v1/catalog/info` of the local OpenAPI explorer
  in `tools/swagger`, a developer tool.
- There is no permanent CLI command for it.
- The getter does not load the catalog. It requires an already loaded
  `*gamecatalog.Catalog` supplied by the caller.
- The result does not include resources or a resource count. It returns only the
  fields of the current contract: `schemaVersion`, `dataVersion`, `gameVersion`,
  `valid`, and `sources`.
