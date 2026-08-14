# ApplyNetworkPreset

## Overview

`ApplyNetworkPreset` selects one backend network preset from the already loaded
GameCatalog and applies its complete parameter set through the same SaveEngine
operation as `SetNetworkSettings`. It owns no second validation rule, binary
writer or platform-specific path.

| | |
|---|---|
| EndpointID | `apply_network_preset` |
| Kind | Mutation |
| Domain | `network` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/network-settings/preset` of the local OpenAPI explorer, registered only without `-allow-external-bind` |
| Implementation source | [../../../backend/endpoints/network/apply_network_preset.go](../../../backend/endpoints/network/apply_network_preset.go) |
| Test source | [../../../backend/endpoints/network/apply_network_preset_test.go](../../../backend/endpoints/network/apply_network_preset_test.go) |
| GameCatalog access | one preset from `regulation/network_params.json` |
| Save access | exactly one call to `SaveEngine.SetNetworkSettings`; persistence remains a separate `WriteSave` operation |

## Input

```go
func ApplyNetworkPreset(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	presetID string,
	expectedRevision string,
) (ApplyNetworkPresetResult, error)
```

The HTTP body is:

```json
{
  "presetID": "faster-reds",
  "expectedRevision": "0"
}
```

`presetID` is required and matched exactly and case-sensitively. It is never
trimmed, normalised, treated as an alias or replaced with `vanilla`. IDs such as
`defaults`, `fast-invasions` and `aggressive-host` existed only in the legacy
application and are rejected because they are absent from GameCatalog.

`expectedRevision` must be the canonical decimal revision currently held by the
session. The transport rejects unknown JSON fields.

## Processing

The endpoint performs only three steps:

1. Obtain an independent preset list from the loaded GameCatalog.
2. Resolve exactly one `presetID` through the same private resolver used by
   `GetNetworkPresets` when it filters by ID.
3. Pass the preset's complete `NetworkParamValues` and `expectedRevision` to
   `SaveEngine.SetNetworkSettings`.

SaveEngine validates all 22 fields and their relationships before mutation. It
owns atomicity, revision handling, rollback and the PC/PS4 DFLT/ZSTD write
paths. The endpoint never merges a preset with the current save and never
applies only the fields that differ.

## Result

```go
type ApplyNetworkPresetResult struct {
	SaveSessionID   string                         `json:"saveSessionID"`
	SaveRevision    string                         `json:"saveRevision"`
	PresetID        string                         `json:"presetID"`
	NetworkSettings gamecatalog.NetworkParamValues `json:"networkSettings"`
}
```

The receipt identifies the selected preset, echoes the complete committed
parameter set and returns the new revision required by the next mutation.

## Errors and atomicity

The whole call fails when the engine or GameCatalog is unavailable, `presetID`
is empty or unknown, the revision is stale, the selected values fail the shared
SaveEngine validation, or `UserData11` cannot be safely rewritten. Every such
failure returns an empty result and leaves the snapshot, revision and dirty flag
unchanged.

There is no fallback to `vanilla`, legacy alias support or partial preset
application.

## Transport

```text
PUT /api/v1/save-sessions/{saveSessionID}/network-settings/preset
```

The route is available only in the explorer's local loopback mode. Success
returns `200`; request and endpoint errors return `400`. There is no Wails,
frontend or CLI binding.

## Local verification

```text
go test ./backend/endpoints/network -run 'TestApplyNetworkPreset' -count=1
go test ./tools/swagger -run 'TestNetworkSettingsRoute' -count=1
```

The endpoint test verifies a catalog preset through the existing network
fixture and confirms that `GetNetworkSettings` observes the complete preset.
Binary platform and compression behavior remains covered by the dedicated
`SetNetworkSettings` SaveEngine tests and is not duplicated here.
