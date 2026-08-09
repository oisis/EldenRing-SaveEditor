# GetNetworkPresets

## Overview

`GetNetworkPresets` reports the network parameter presets the backend offers
today, together with the full parameter values of each preset.

The values live in
[`backend/gamecatalog/data/regulation/network_params.json`](../../../backend/gamecatalog/data/regulation/network_params.json).
That file is the authoritative source of the preset values for SaveForge 2.0 and
for this endpoint. `GameCatalog` loads and validates it once, when it is built,
and the endpoint only reads the already loaded presets: it opens no file per call
and it no longer calls the `backend/core` preset functions. The public contract
and the list of thirteen presets are unchanged.

`backend/core` still holds the older preset functions, but only temporarily and
only for its remaining legacy callers. It is no longer a data source for
SaveForge 2.0. Those functions will be removed together with `backend/core` in a
separate, later task.

| | |
|---|---|
| EndpointID | `get_network_presets` |
| Kind | Getter |
| Domain | `network` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/network/presets` of the local OpenAPI explorer (`backend/endpoints/swagger`). No Wails binding and no permanent CLI command reach it. |
| Implementation source | [../../../backend/endpoints/network/get_network_presets.go](../../../backend/endpoints/network/get_network_presets.go) |
| Test source | [../../../backend/endpoints/network/get_network_presets_test.go](../../../backend/endpoints/network/get_network_presets_test.go) |
| Data source | [`backend/gamecatalog/data/regulation/network_params.json`](../../../backend/gamecatalog/data/regulation/network_params.json), loaded and validated by `GameCatalog` |
| Save access | none — the endpoint never opens, reads, or writes a save |
| Mutation | none — the endpoint builds a result and modifies nothing |

## Input

The endpoint takes the loaded catalog and exactly one public parameter:

```go
func GetNetworkPresets(
	gameCatalog *gamecatalog.Catalog,
	presetID string,
) (GetNetworkPresetsResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `gameCatalog` | `*gamecatalog.Catalog` | The already loaded catalog, supplied by the backend caller. It owns the preset data; the endpoint never loads it. |
| `presetID` | `string` | Public preset identifier. Empty means every preset; a non-empty value selects exactly one preset. |

Matching rules:

- `presetID` is matched exactly and case-sensitively.
- It is never trimmed, never normalised, and never resolved through an alias.
  `Vanilla`, `" vanilla"`, and `faster_reds` are unknown identifiers, not the
  presets `vanilla` and `faster-reds`.
- An unknown identifier is an error, never a silent fallback to the full list or
  to a default preset.

The endpoint reads no other input. It takes no save session and no character.

## Output

The endpoint returns a typed result:

```go
type NetworkPreset = gamecatalog.NetworkPreset

type GetNetworkPresetsResult struct {
	Presets []NetworkPreset `json:"presets"`
}
```

`gamecatalog.NetworkPreset` carries `ID string` with the JSON name `id` and
`Parameters gamecatalog.NetworkParamValues` with the JSON name `parameters`, so
the public JSON shape is unchanged.

| Field | Type | Meaning |
|---|---|---|
| `presets` | array of `NetworkPreset` | Every backend preset in the order below, or exactly the one requested preset. Always non-nil. |
| `id` | `string` | The stable public identifier of the preset. |
| `parameters` | `gamecatalog.NetworkParamValues` | The complete parameter set of the preset, exactly as `network_params.json` stores it. |

`gamecatalog.NetworkParamValues` is owned by `GameCatalog`, which loads it from
`network_params.json`. Its field names and JSON names are unchanged; this
endpoint neither extends nor reshapes the type, and it copies no numeric value
of its own. The result carries no display name, no description, no tags, and no
category.

## Exposed presets

The endpoint exposes exactly thirteen presets, in this order:

| `id` | Value source in `network_params.json` |
|---|---|
| `vanilla` | the `default` object |
| `faster-reds` | `presets[0]` |
| `aggressive-reds` | `presets[1]` |
| `faster-summons` | `presets[2]` |
| `aggressive-summons` | `presets[3]` |
| `faster-blue` | `presets[4]` |
| `aggressive-blue` | `presets[5]` |
| `faster-summon-host` | `presets[6]` |
| `aggressive-summon-host` | `presets[7]` |
| `faster-summon-guest` | `presets[8]` |
| `aggressive-summon-guest` | `presets[9]` |
| `faster-hunter` | `presets[10]` |
| `aggressive-hunter` | `presets[11]` |

An empty `presetID` returns the list in exactly that order: the `default` object
first, then the `presets` array in file order.

### Presets that are deliberately not exposed

`backend/core` still holds legacy preset functions kept for older callers. They
are not stored in `network_params.json`, they are not part of this endpoint's
contract, and their identifiers are rejected as unknown:

- `fast-invasions` (`core.NetworkParamFastInvasions`)
- `light-invasions` (`core.NetworkParamLightInvasions`)
- `fast-summons` (`core.NetworkParamFastSummons`)
- `fast-blue` (`core.NetworkParamFastBlue`)
- `aggressive-host` (`core.NetworkParamAggressiveHost`)
- `defaults` — not an identifier of this endpoint; the vanilla values are
  `vanilla`
- `core.NetworkParamFast` — a legacy alias of `NetworkParamFastInvasions`, with
  no identifier here

## Processing flow

1. The getter reads the thirteen presets from the loaded `GameCatalog`, which
   validated `network_params.json` when it was built. A missing catalog or a
   catalog built without network parameters is an error, not an empty list.
2. An empty `presetID` returns all of them, in the contractual order.
3. A non-empty `presetID` is compared exactly and case-sensitively against the
   preset identifiers.
4. A match returns a result holding exactly that one preset.
5. No match returns the error below and an empty result.
6. The catalog returns an independent copy per call, so mutating one result never
   affects another call and never changes the catalog. The endpoint keeps no
   mutable shared state and re-reads no file.
7. The getter modifies nothing.

## Validation and errors

- An unknown `presetID` returns `unknown network preset "<value>"` and an empty
  result. The HTTP route maps it to `400`, because the value comes from the
  client.
- A `nil` catalog returns `game catalog is not loaded`, and a catalog built
  without network parameters returns `network parameters are not loaded`. Both
  are backend configuration errors, not client input errors.
- There is no other validation: a known identifier is always resolvable, because
  `GameCatalog` rejects an absent, malformed, or inconsistent
  `network_params.json` while it is being built.

## Dependencies

- The endpoint reads no save. It never opens a `.sl2`, a `.dat`, or a save
  session, and it performs no mutation of any kind.
- The endpoint reads exactly one thing: the network presets of the already
  loaded `GameCatalog`. It reads no manifest, no resource, and no other catalog
  data, and it never loads or reloads the catalog itself.
- The endpoint calls no other endpoint.
- The endpoint no longer imports `backend/core` and calls none of its
  `NetworkParam*` functions. Those functions stay in `backend/core` only for its
  remaining legacy callers, and they are removed together with `backend/core` in
  a separate, later task.
- The endpoint does not depend on the legacy `internal/application` package and
  does not import it. The `GetNetworkPreset` implementation that lives there is
  not reused.
- All parameter values come from
  `backend/gamecatalog/data/regulation/network_params.json`, which is their
  single source of truth. `GameCatalog` loads it once, validates it, and stores
  it read-only.

## Command-line verification

`GetNetworkPresets` is exposed over HTTP as `GET /api/v1/network/presets` by the
local OpenAPI explorer in `backend/endpoints/swagger`, a developer tool the
application neither imports nor starts. The route passes the catalog the
explorer loaded at start-up; it does not need the application version. A missing
or invalid `network_params.json` stops the explorer while it loads its data,
instead of leaving the route half working.

### Run tests

From the repository root:

```bash
go test ./backend/endpoints/network -run '^TestGetNetworkPresets' -count=1 -v
```

### Call the route

Start the explorer:

```bash
go run ./backend/endpoints/swagger
```

Then request every preset:

```bash
curl -s http://127.0.0.1:8788/api/v1/network/presets
```

Or exactly one preset:

```bash
curl -s "http://127.0.0.1:8788/api/v1/network/presets?presetID=faster-reds"
```

The filtered response holds a single entry whose `parameters` are the stored
values of the `faster-reds` preset:

```json
{
  "presets": [
    {
      "id": "faster-reds",
      "parameters": {
        "maxBreakInTargetListCount": 8,
        "breakInRequestIntervalTimeSec": 12,
        "breakInRequestTimeOutSec": 8,
        "breakInRequestAreaCount": 8
      }
    }
  ]
}
```

The excerpt above is shortened: the real response carries every field of
`gamecatalog.NetworkParamValues`.

An unknown identifier is rejected with `400`:

```bash
curl -s "http://127.0.0.1:8788/api/v1/network/presets?presetID=fast-invasions"
```

```json
{
  "error": "unknown network preset \"fast-invasions\""
}
```

No demonstration program or helper script for this endpoint is kept in the
repository.

## Current limitations

- The endpoint is not exposed through Wails.
- The only HTTP route is `GET /api/v1/network/presets` of the local OpenAPI
  explorer in `backend/endpoints/swagger`, a developer tool.
- There is no permanent CLI command for it.
- The result carries identifiers and parameter values only. There is no display
  name, description, tag, category, or role grouping.
- The endpoint only reports presets. Applying a preset to a save
  (`ApplyNetworkPreset`) and reading the current save settings
  (`GetNetworkSettings`) are separate endpoints and are not implemented here.
