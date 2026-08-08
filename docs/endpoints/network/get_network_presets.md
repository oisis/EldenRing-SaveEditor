# GetNetworkPresets

## Overview

`GetNetworkPresets` reports the network parameter presets the backend offers
today, together with the full parameter values of each preset.

| | |
|---|---|
| EndpointID | `get_network_presets` |
| Kind | Getter |
| Domain | `network` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/network/presets` of the local OpenAPI explorer (`backend/endpoints/swagger`). No Wails binding and no permanent CLI command reach it. |
| Implementation source | [../../../backend/endpoints/network/get_network_presets.go](../../../backend/endpoints/network/get_network_presets.go) |
| Test source | [../../../backend/endpoints/network/get_network_presets_test.go](../../../backend/endpoints/network/get_network_presets_test.go) |
| Save access | none — the endpoint never opens, reads, or writes a save |
| Mutation | none — the endpoint builds a result and modifies nothing |

## Input

The endpoint takes exactly one public parameter:

```go
func GetNetworkPresets(presetID string) (GetNetworkPresetsResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `presetID` | `string` | Public preset identifier. Empty means every preset; a non-empty value selects exactly one preset. |

Matching rules:

- `presetID` is matched exactly and case-sensitively.
- It is never trimmed, never normalised, and never resolved through an alias.
  `Vanilla`, `" vanilla"`, and `faster_reds` are unknown identifiers, not the
  presets `vanilla` and `faster-reds`.
- An unknown identifier is an error, never a silent fallback to the full list or
  to a default preset.

The endpoint reads no other input. It takes no save session, no character, and
no GameCatalog.

## Output

The endpoint returns a typed result:

```go
type NetworkPreset struct {
	ID         string                  `json:"id"`
	Parameters core.NetworkParamValues `json:"parameters"`
}

type GetNetworkPresetsResult struct {
	Presets []NetworkPreset `json:"presets"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `presets` | array of `NetworkPreset` | Every backend preset in the order below, or exactly the one requested preset. Always non-nil. |
| `id` | `string` | The stable public identifier of the preset. |
| `parameters` | `core.NetworkParamValues` | The complete parameter set of the preset, exactly as the owning `backend/core` function returns it. |

`core.NetworkParamValues` is owned by `backend/core`. Its fields and JSON names
are defined there; this endpoint neither extends nor reshapes the type, and it
copies no numeric value of its own. The result carries no display name, no
description, no tags, and no category.

## Exposed presets

The endpoint exposes exactly thirteen presets, in this order:

| `id` | Value source in `backend/core` |
|---|---|
| `vanilla` | `core.NetworkParamDefaults()` |
| `faster-reds` | `core.NetworkParamFasterReds()` |
| `aggressive-reds` | `core.NetworkParamAggressiveReds()` |
| `faster-summons` | `core.NetworkParamFasterSummons()` |
| `aggressive-summons` | `core.NetworkParamAggressiveSummons()` |
| `faster-blue` | `core.NetworkParamFasterBlue()` |
| `aggressive-blue` | `core.NetworkParamAggressiveBlue()` |
| `faster-summon-host` | `core.NetworkParamFasterSummonHost()` |
| `aggressive-summon-host` | `core.NetworkParamAggressiveSummonHost()` |
| `faster-summon-guest` | `core.NetworkParamFasterSummonGuest()` |
| `aggressive-summon-guest` | `core.NetworkParamAggressiveSummonGuest()` |
| `faster-hunter` | `core.NetworkParamFasterHunter()` |
| `aggressive-hunter` | `core.NetworkParamAggressiveHunter()` |

An empty `presetID` returns the list in exactly that order.

### Presets that are deliberately not exposed

`backend/core` still holds legacy preset functions kept for older callers. They
are not part of this endpoint's contract and their identifiers are rejected as
unknown:

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

1. The getter builds the thirteen presets from the `backend/core` functions
   listed above.
2. An empty `presetID` returns all of them, in the contractual order.
3. A non-empty `presetID` is compared exactly and case-sensitively against the
   preset identifiers.
4. A match returns a result holding exactly that one preset.
5. No match returns the error below and an empty result.
6. The result is built per call, so mutating one result never affects another
   call. The endpoint keeps no mutable shared state.
7. The getter modifies nothing.

## Validation and errors

- An unknown `presetID` returns `unknown network preset "<value>"` and an empty
  result. The HTTP route maps it to `400`, because the value comes from the
  client.
- There is no other validation: a known identifier is always resolvable, since
  the values come from compiled-in functions.

## Dependencies

- The endpoint reads no save. It never opens a `.sl2`, a `.dat`, or a save
  session, and it performs no mutation of any kind.
- The endpoint reads no GameCatalog. It takes no catalog instance and no
  manifest.
- The endpoint calls no other endpoint.
- The endpoint does not depend on the legacy `internal/application` package and
  does not import it. The `GetNetworkPreset` implementation that lives there is
  not reused.
- All parameter values come from `backend/core`, which remains their single
  source of truth.

## Command-line verification

`GetNetworkPresets` is exposed over HTTP as `GET /api/v1/network/presets` by the
local OpenAPI explorer in `backend/endpoints/swagger`, a developer tool the
application neither imports nor starts. The route needs neither the catalog nor
the application version.

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

The filtered response holds a single entry whose `parameters` are the values of
`core.NetworkParamFasterReds()`:

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
`core.NetworkParamValues`.

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
