# GetFavoritePresets

## Overview

`GetFavoritePresets` returns the occupancy state of the 15 global Mirror
Favorites appearance preset slots stored in `UserData10` of an existing save
session. It reads the session's private snapshot only.

The Mirror Favorites slots are shared across all 10 character slots of the save
file (they live globally in `UserData10`, not inside individual character
slots). Consequently, the endpoint accepts no `characterID`.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetFavoritePresets` never creates one,
opens no source file, reads no GameCatalog, and modifies nothing: neither the
save, nor the session, nor any application state.

| | |
|---|---|
| EndpointID | `get_favorite_presets` |
| Kind | Getter |
| Domain | `favorites` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/favorite-presets` of the local OpenAPI explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`. There is no Wails binding, no frontend view and no CLI command. |
| Implementation source | [../../../backend/endpoints/favorites/get_favorite_presets.go](../../../backend/endpoints/favorites/get_favorite_presets.go) |
| Test source | [../../../backend/endpoints/favorites/get_favorite_presets_test.go](../../../backend/endpoints/favorites/get_favorite_presets_test.go) |
| Data source | the 15 global Mirror Favorites slots in `UserData10`, read by SaveEngine |
| Save access | read-only — SaveEngine reads the private session snapshot; no file is opened |
| Mutation | none — this getter changes no save, snapshot, session or catalog state |

## Input

```go
func GetFavoritePresets(
	engine *saveengine.Engine,
	saveSessionID string,
	favoriteSlotID *int,
) (GetFavoritePresetsResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance that owns the loaded sessions. A `nil` engine is rejected. |
| `saveSessionID` | `string` | Identifier of an already loaded session. It is passed to SaveEngine unchanged. |
| `favoriteSlotID` | `*int` | Optional filter: `nil` returns all 15 slots in order `0..14`; a value in `0..14` returns exactly that one slot. Any other value is rejected. |

### `saveSessionID`

- Matched exactly and case-sensitively by SaveEngine. It is never trimmed,
  normalised or guessed, so an empty, unknown or already closed identifier is
  rejected instead of resolving to a session.

### `favoriteSlotID`

- `nil` returns all 15 preset slots in physical index order `0..14`.
- A pointer to an integer `0..14` returns a one-element slice containing that exact slot.
- Any value outside `0..14` (e.g. `-1` or `15`) is rejected as an error.
- The parameter is never clamped, wrapped or resolved to a neighbouring slot.

## Result

```go
type FavoritePreset struct {
	FavoriteSlotID int  `json:"favoriteSlotID"`
	Active         bool `json:"active"`
}

type GetFavoritePresetsResult struct {
	SaveSessionID string           `json:"saveSessionID"`
	Presets       []FavoritePreset `json:"presets"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was read. Equals the requested value. |
| `presets` | `[]FavoritePreset` | Non-nil slice of preset occupancy records. Contains 15 entries when `favoriteSlotID` is `nil`, or exactly 1 entry when filtered. |
| `presets[].favoriteSlotID` | `int` | Physical slot index in `0..14`. |
| `presets[].active` | `bool` | `true` when the slot contains the exact 4-byte ASCII signature `"FACE"` at offset `0x18`; `false` when empty or containing any other bytes. |

### What is not returned

The result contains only occupancy information. It returns:
- no preset names, images or descriptions (which were session caches or catalog matches in 1.x, not stored save data);
- no raw 304-byte preset blobs;
- no appearance model IDs, face shape, body or skin cosmetics;
- no legacy `safe` flags (all 15 slots `0..14` are fully accessible in SaveForge 2.0).

## How the values are read

The binary layout is owned by SaveEngine ([`backend/saveengine/favorites.go`](../../../backend/saveengine/favorites.go)):

- **Base offset**: `0x154` from the start of `UserData10` data on both PC and PS4 (PC skips the 0x10 MD5 prefix; PS4 has no prefix).
- **Slot count**: 15 slots (indices `0..14`).
- **Slot size**: `0x130` bytes (304 bytes) per slot; total section span `0x154..0x1323` (4560 bytes).
- **Active detection**:
  - `active` is `true` if and only if the four bytes at slot offset `0x18` are exactly equal to ASCII `"FACE"`.
  - If the magic at `0x18` is absent or differs (e.g. all `0x00`), the slot is reported as `active: false`.
  - The getter reports occupancy only: it does not validate alignment (`+0x1C`), inner size (`+0x20`), face structure or preset integrity.
  - Alignment (`4`) and inner size (`0x120` / 288 bytes) at `0x1C` and `0x20`, as well as outer header values at `0x00..0x13` (`0xFACE` at `0x00`, `0x11D0` at `0x04`), are observed characteristics in native saves and writer targets, but are not preconditions for reading slot occupancy.

Everything happens on the session's private, read-only snapshot: no file is
opened, no byte is written, and revision, dirty state, undo and registries remain
untouched.

## Errors

| Situation | Result |
|---|---|
| `engine` is `nil` | `save engine is not available` |
| empty `saveSessionID` | `saveSessionID is required` |
| unknown or closed `saveSessionID` | `unknown save session "<id>"` |
| `favoriteSlotID` outside `0..14` | `favoriteSlotID <id> is outside the range 0..14` |
| `UserData10` is truncated or does not cover the requested slot | `favorite preset slot <id> lies outside UserData10 bounds` |

## Transport

```
GET /api/v1/save-sessions/{saveSessionID}/favorite-presets
```

Query parameters:
- `favoriteSlotID` (optional integer `0..14`): when provided, filters output to that single slot.

The route belongs to the local developer explorer in `tools/swagger`. It is
registered only in loopback mode; an explorer started with `-allow-external-bind`
does not register it and answers `404`.

## Local verification

```bash
go test ./backend/saveengine -run '^TestGetFavoritePresets' -count=1
go test ./backend/endpoints/favorites -run '^TestGetFavoritePresets' -count=1
go test ./tools/swagger -run '^TestGetFavoritePresetsRoute' -count=1
```
