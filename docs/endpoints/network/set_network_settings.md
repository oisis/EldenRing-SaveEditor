# SetNetworkSettings

## Overview

`SetNetworkSettings` atomically replaces the complete set of 22 supported
network parameters in `UserData11` of an already loaded save session. It accepts
direct values; it does not load or apply a GameCatalog preset.
`ApplyNetworkPreset` resolves a preset and delegates to this same SaveEngine
operation.

| | |
|---|---|
| EndpointID | `set_network_settings` |
| Kind | Mutation |
| Domain | `network` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/network-settings` of the local OpenAPI explorer, registered only without `-allow-external-bind` |
| Implementation source | [../../../backend/endpoints/network/set_network_settings.go](../../../backend/endpoints/network/set_network_settings.go) |
| Test source | [../../../backend/endpoints/network/set_network_settings_test.go](../../../backend/endpoints/network/set_network_settings_test.go) |
| GameCatalog access | none; `gamecatalog.NetworkParamValues` is only the shared typed model |
| Save access | atomic mutation of the private session snapshot; persistence remains a separate `WriteSave` operation |

## Input

```go
func SetNetworkSettings(
	engine *saveengine.Engine,
	saveSessionID string,
	networkSettings gamecatalog.NetworkParamValues,
	expectedRevision string,
) (SetNetworkSettingsResult, error)
```

The HTTP body is:

```json
{
  "networkSettings": {
    "maxBreakInTargetListCount": 8,
    "breakInRequestIntervalTimeSec": 12,
    "breakInRequestTimeOutSec": 8,
    "breakInRequestAreaCount": 8,
    "summonTimeoutTime": 45,
    "reloadSignIntervalTime2": 20,
    "reloadSignTotalCount": 40,
    "reloadSignCellCount": 20,
    "updateSignIntervalTime": 15,
    "singGetMax": 64,
    "signDownloadSpan": 15,
    "signUpdateSpan": 20,
    "reloadVisitListCoolTime": 8,
    "maxCoopBlueSummonCount": 2,
    "maxVisitListCount": 10,
    "reloadSearchCoopBlueMin": 10,
    "reloadSearchCoopBlueMax": 40,
    "allAreaSearchRateCoopBlue": 60,
    "allAreaSearchRateVsBlue": 30,
    "visitorListMax": 10,
    "visitorTimeOutTime": 60,
    "visitorDownloadSpan": 60
  },
  "expectedRevision": "0"
}
```

The JSON decoder rejects unknown fields. All 22 fields are required by the
`NetworkParamValues` schema; an omitted numeric field becomes zero in a direct
Go call and is rejected by SaveEngine's lower bounds.

## Validation

Every floating-point value must be finite. SaveEngine applies these confirmed
write limits before reading or changing the snapshot:

| Field | Inclusive range |
|---|---:|
| `maxBreakInTargetListCount` | 1..20 |
| `breakInRequestIntervalTimeSec` | 2..30 |
| `breakInRequestTimeOutSec` | 3..20 |
| `breakInRequestAreaCount` | 1..50 |
| `summonTimeoutTime` | 1..999 |
| `reloadSignIntervalTime2` | 1..1000 |
| `reloadSignTotalCount` | 1..128 |
| `reloadSignCellCount` | 1..99 |
| `updateSignIntervalTime` | 1..1000 |
| `singGetMax` | 1..128 |
| `signDownloadSpan` | 1..1000 |
| `signUpdateSpan` | 1..1000 |
| `reloadVisitListCoolTime` | 1..1000 |
| `maxCoopBlueSummonCount` | 1..10 |
| `maxVisitListCount` | 1..50 |
| `reloadSearchCoopBlueMin` | 1..999 |
| `reloadSearchCoopBlueMax` | 1..999 |
| `allAreaSearchRateCoopBlue` | 0..100 |
| `allAreaSearchRateVsBlue` | 0..100 |
| `visitorListMax` | 1..100 |
| `visitorTimeOutTime` | 1..600 |
| `visitorDownloadSpan` | 1..600 |

Three relationships must also hold:

- `reloadSignCellCount <= reloadSignTotalCount`;
- `reloadSignTotalCount <= singGetMax`;
- `reloadSearchCoopBlueMin <= reloadSearchCoopBlueMax`.

`expectedRevision` must be the canonical decimal revision currently held by the
session. A validation or revision failure leaves the snapshot, revision and
dirty flag unchanged.

## Write path

SaveEngine decrypts the existing regulation blob, locates
`NetworkParam.param` by its BND4 name and row 0 by the confirmed long-offset
layout, then writes exactly the 22 fields. It builds and decodes the complete
replacement before changing the private snapshot and verifies the written
values afterward. A verification failure restores the original `UserData11`.

The container rules remain platform-specific:

| Platform and format | Write rule |
|---|---|
| PC DFLT | recompress the DCX payload, encrypt it with the original IV and update the `UserData11` MD5 |
| PC ZSTD | recompress the DCX payload, encrypt it with the original IV and update the `UserData11` MD5 |
| PS4 DFLT | recompress and re-encrypt within the existing fixed blob capacity |
| PS4 ZSTD | preserve the native ZSTD frame and untouched blocks; replace only blocks covering the edited row fields with RAW blocks before re-encryption |

The PS4 ZSTD rule is required because full recompression was confirmed to make
the console reject the save. A replacement that exceeds the existing encrypted
blob capacity fails before the snapshot is touched. No other BND4 entry,
parameter row or save field is intentionally changed.

## Result

```go
type SetNetworkSettingsResult struct {
	saveengine.MutationReceipt
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
receipt members and `networkSettings` all sit at the top level, and there is no
nested `receipt` object.

```json
{
  "operationID": "op-0f1e2d3c4b5a69788796a5b4c3d2e1f0",
  "operationKind": "set_network_settings",
  "saveSessionID": "save-session-1",
  "saveRevision": "1",
  "changedScopes": ["save.session", "network", "diagnostics.report"],
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
  `set_network_settings`.
- `changedScopes` are exactly `save.session`, `network` and `diagnostics.report`,
  in that canonical order.

`saveRevision` is the new revision clients must use for the next mutation.
`networkSettings` echoes the complete committed set.

`ApplyNetworkPreset` shares this SaveEngine writer but reports its own
`operationKind`; see [apply_network_preset.md](apply_network_preset.md).

## Transport

```
PUT /api/v1/save-sessions/{saveSessionID}/network-settings
```

The route is available only in the explorer's local loopback mode. Success
returns `200`; malformed JSON, validation failures, revision conflicts, unknown
sessions and unsafe or malformed save layouts return `400`. The route has no
Wails, frontend or CLI binding.

## Local verification

```text
go test ./backend/saveengine -run '^TestSetNetworkSettings' -count=1
go test ./backend/endpoints/network -run '^TestSetNetworkSettings' -count=1
go test ./tools/swagger -run '^TestNetworkSettingsRoute' -count=1
```

The SaveEngine tests use synthetic PC/PS4 DFLT/ZSTD fixtures in `t.TempDir()`,
persist through `WriteSave`, reload the result and verify all 22 values. No real
save file is read or modified.
