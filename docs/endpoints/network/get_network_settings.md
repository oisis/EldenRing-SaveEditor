# GetNetworkSettings

## Overview

`GetNetworkSettings` returns the 22 network parameters stored in `UserData11` of
an already loaded save session. It reports the state of that save and nothing
else: the values are the ones the save holds, not the ones a preset recommends.

The endpoint reads no GameCatalog. It never loads
`regulation/network_params.json`, never compares a value against a preset range
and never calls `GetNetworkPresets`; `gamecatalog.NetworkParamValues` is reused
as the shared typed model so that a stored parameter set and a preset parameter
set have exactly the same 22 fields and JSON names.

| | |
|---|---|
| EndpointID | `get_network_settings` |
| Kind | Getter |
| Domain | `network` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/network-settings` of the local OpenAPI explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`. No Wails binding, no frontend view and no CLI command reach it. |
| Implementation source | [../../../backend/endpoints/network/get_network_settings.go](../../../backend/endpoints/network/get_network_settings.go) |
| Test source | [../../../backend/endpoints/network/get_network_settings_test.go](../../../backend/endpoints/network/get_network_settings_test.go) |
| Data source | the `UserData11` regulation of the loaded save session, read by SaveEngine |
| Save access | read-only — SaveEngine reads the private session snapshot; no file is opened by this endpoint |
| Mutation | none — this getter changes no save, snapshot, session or catalog state; `SetNetworkSettings` is the separate write endpoint |

## Input

```go
func GetNetworkSettings(
	engine *saveengine.Engine,
	saveSessionID string,
) (GetNetworkSettingsResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance that owns the loaded sessions. A `nil` engine is the only input the endpoint validates itself. |
| `saveSessionID` | `string` | Identifier of an already loaded session. It is passed to SaveEngine unchanged. |

The session must already exist. The endpoint never creates one: it calls neither
`LoadSave` nor any other endpoint.

`saveSessionID` is matched exactly by SaveEngine. It is never trimmed,
normalised or guessed, so an empty, unknown or already closed identifier is
rejected instead of resolving to a session.

## Result

```go
type GetNetworkSettingsResult struct {
	SaveSessionID string                         `json:"saveSessionID"`
	Parameters    gamecatalog.NetworkParamValues `json:"parameters"`
}
```

`Parameters` holds the 22 values in the field order and with the JSON names of
`gamecatalog.NetworkParamValues`: `maxBreakInTargetListCount`,
`breakInRequestIntervalTimeSec`, `breakInRequestTimeOutSec`,
`breakInRequestAreaCount`, `summonTimeoutTime`, `reloadSignIntervalTime2`,
`reloadSignTotalCount`, `reloadSignCellCount`, `updateSignIntervalTime`,
`singGetMax`, `signDownloadSpan`, `signUpdateSpan`, `reloadVisitListCoolTime`,
`maxCoopBlueSummonCount`, `maxVisitListCount`, `reloadSearchCoopBlueMin`,
`reloadSearchCoopBlueMax`, `allAreaSearchRateCoopBlue`,
`allAreaSearchRateVsBlue`, `visitorListMax`, `visitorTimeOutTime` and
`visitorDownloadSpan`. Ten of them are `int32` and twelve are `float32`, exactly
as stored.

## How the values are read

The whole binary path belongs to SaveEngine
([`backend/saveengine/network_settings.go`](../../../backend/saveengine/network_settings.go),
[`network_settings_pc.go`](../../../backend/saveengine/network_settings_pc.go),
[`network_settings_ps4.go`](../../../backend/saveengine/network_settings_ps4.go)).
The endpoint holds no offset, no key and no format rule of its own.

Both platforms are supported and their container rules stay separate:

| | PC | PS4 |
|---|---|---|
| `UserData11` position | behind `UserData10`, to the end of the file | behind `UserData10`, to the end of the file |
| Prefix inside `UserData11` | `0x10` MD5 prefix, never parsed or verified, then the `0x10` regulation header | the `0x10` regulation header only |
| Regulation blob | behind the header, identical on both platforms | behind the header, identical on both platforms |

From the blob inwards the path is shared: the leading initialisation vector and
AES-256-CBC decryption, the `DCX` archive in one of its two confirmed
compression variants, the `BND4` archive inside it, the `NetworkParam.param`
entry located by name, and row 0 of that parameter file, from which the 22
values are read at their fixed offsets.

Everything happens on the session's private, read-only snapshot: no file is
opened, no byte is written, no snapshot, session, catalog or application state
changes, and no raw save byte leaves SaveEngine.

## Reported, not corrected

The values are returned exactly as stored:

- no value is validated against a preset range;
- no value is clamped, rounded or normalised;
- no missing or implausible value is replaced by a default or by a preset;
- no preset is loaded, compared or suggested.

Structural decoding, in contrast, is strict and fail-closed.

## Errors

| Situation | Result |
|---|---|
| `engine` is `nil` | error — the endpoint rejects it before delegating |
| empty, unknown or already closed `saveSessionID` | error from SaveEngine, exactly like every other SaveEngine getter |
| the container carries no `UserData11` | error |
| `UserData11` does not start with the confirmed regulation header | error |
| the regulation blob is truncated or not block aligned | error |
| the blob does not decrypt into a `DCX` archive | error |
| the `DCX` uses an unsupported compression variant | error |
| the archive is not a `BND4` or holds no `NetworkParam.param` | error |
| the parameter file uses an unsupported layout or a row too short for the 22 values | error |

Every one of them fails the whole call. No partial parameter set, no default set
and no guessed value is ever returned.

## Transport

```
GET /api/v1/save-sessions/{saveSessionID}/network-settings
```

The route belongs to the local developer explorer in `tools/swagger`. It is
registered in `registerSaveSessionRoutes` only, so it exists exclusively in the
local loopback mode: an explorer started with `-allow-external-bind` does not
register it and answers `404`. The path segment is passed to
`network.GetNetworkSettings` unchanged; a successful call returns `200` and every
endpoint error returns `400`.

There is no Wails binding, frontend view or CLI command for this endpoint. The
separate `SetNetworkSettings` endpoint owns the write path.

## Local verification

```
go test ./backend/saveengine -run '^TestGetNetworkSettings' -count=1
go test ./backend/endpoints/network -run '^TestGetNetworkSettings' -count=1
go test ./tools/swagger -run '^TestNetworkSettingsRoute' -count=1
```

The tests build synthetic PC and PS4 containers in `t.TempDir()` from the format
rules above. No real save is read and no fixture blob is stored in the
repository.
