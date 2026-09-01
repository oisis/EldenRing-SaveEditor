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
	saveengine.MutationReceipt
	PresetID        string                         `json:"presetID"`
	NetworkSettings gamecatalog.NetworkParamValues `json:"networkSettings"`
}

type MutationReceipt struct {
	OperationID   string   `json:"operationID"`
	OperationKind string   `json:"operationKind"`
	SaveSessionID string   `json:"saveSessionID"`
	SaveRevision  string   `json:"saveRevision"`
	ChangedScopes []string `json:"changedScopes"`
}
```

The receipt is embedded anonymously, so the JSON result is flat: the five
receipt members, `presetID` and `networkSettings` all sit at the top level, and
there is no nested `receipt` object.

```json
{
  "operationID": "op-0f1e2d3c4b5a69788796a5b4c3d2e1f0",
  "operationKind": "apply_network_preset",
  "saveSessionID": "save-session-1",
  "saveRevision": "2",
  "changedScopes": ["save.session", "network", "diagnostics.report"],
  "presetID": "faster-reds",
  "networkSettings": { "...": "the complete committed set of 22 values" }
}
```

The embedded `saveengine.MutationReceipt` is exactly the receipt the central
SaveEngine commit path produced for this execution. Nothing here is
reassembled from the EndpointID, the session, the revision or a scope lookup.

- `operationID` names this one execution. It is opaque and unpredictable.
  Identifiers do not repeat among the receipts issued by one running SaveEngine
  instance. That guarantee does not currently cover application restarts:
  uniqueness across restarts requires a persistent operation journal and stays
  outside stage 3b.1. A rejected call returns the zero result and no
  `operationID` at all.
- `operationKind` is the stable kind of the mutation and is always exactly
  `apply_network_preset`.
- `changedScopes` are exactly `save.session`, `network` and `diagnostics.report`,
  in that canonical order.

`presetID` identifies the selected preset and `networkSettings` echoes the
complete committed parameter set.

This endpoint and `SetNetworkSettings` share one SaveEngine writer, and the
writer receives its operation kind from the public entry point. A preset
therefore always reports `apply_network_preset` and never
`set_network_settings`.

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
