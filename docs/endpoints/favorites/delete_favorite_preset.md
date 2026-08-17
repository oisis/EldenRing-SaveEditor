# DeleteFavoritePreset

## Overview

`DeleteFavoritePreset` clears the specified Mirror Favorites appearance preset
slot stored in `UserData10` of an existing save session under `expectedRevision`
control. It operates on the session's private snapshot only.

The Mirror Favorites slots are shared across all 10 character slots of the save
file (they live globally in `UserData10`, not inside individual character
slots). Consequently, the endpoint accepts no `characterID`.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `DeleteFavoritePreset` opens no source
file, reads no GameCatalog, and persists nothing directly; persistence is
owned by a separate [`WriteSave`](../savesession/write_save.md).

| | |
|---|---|
| EndpointID | `delete_favorite_preset` |
| Kind | Mutation |
| Domain | `favorites` |
| Implementation status | implemented |
| Transport status | transport-exposed — `DELETE /api/v1/save-sessions/{saveSessionID}/favorite-presets/{favoriteSlotID}` of the local OpenAPI explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`. There is no Wails binding, no frontend view and no CLI command. |
| Implementation source | [../../../backend/endpoints/favorites/delete_favorite_preset.go](../../../backend/endpoints/favorites/delete_favorite_preset.go) |
| Test source | [../../../backend/endpoints/favorites/delete_favorite_preset_test.go](../../../backend/endpoints/favorites/delete_favorite_preset_test.go) |
| Data source | the specified global Mirror Favorites slot in `UserData10`, mutated by SaveEngine |
| Save access | private session snapshot mutation with verification and rollback; no file is opened |
| Mutation | clears the full 0x130-byte range of the active slot; advances `saveRevision`, sets `dirty` flag and invalidates the session's undo point |

## Input

```go
func DeleteFavoritePreset(
	engine *saveengine.Engine,
	saveSessionID string,
	favoriteSlotID int,
	expectedRevision string,
) (DeleteFavoritePresetResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance that owns the loaded sessions. A `nil` engine is rejected. |
| `saveSessionID` | `string` | Identifier of an already loaded session. It is passed to SaveEngine unchanged. |
| `favoriteSlotID` | `int` | Slot index in `0..14` identifying which of the 15 preset slots to clear. |
| `expectedRevision` | `string` | Canonical decimal `saveRevision` matching the current revision of the session. |

### `saveSessionID`

- Matched exactly and case-sensitively by SaveEngine. It is never trimmed,
  normalised or guessed, so an empty, unknown or already closed identifier is
  rejected before any mutation occurs.

### `favoriteSlotID`

- An integer in `0..14`.
- Any value outside `0..14` (e.g. `-1` or `15`) is rejected.
- The parameter is never clamped or wrapped.

### `expectedRevision`

- Must be the canonical decimal string representation of a non-negative integer
  without leading zeros (e.g. `"0"`, `"1"`).
- Checked under the session lock inside `commitRevision`. If it does not match
  the session's current `saveRevision`, the operation fails with no changes.

## Result

```go
type DeleteFavoritePresetResult struct {
	SaveSessionID  string `json:"saveSessionID"`
	SaveRevision   string `json:"saveRevision"`
	FavoriteSlotID int    `json:"favoriteSlotID"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was modified. |
| `saveRevision` | `string` | New canonical decimal save revision after the mutation (incremented by 1). |
| `favoriteSlotID` | `int` | Physical slot index in `0..14` that was deleted. |

## How the slot is deleted

The binary layout is owned by SaveEngine ([`backend/saveengine/favorites.go`](../../../backend/saveengine/favorites.go)):

- **Base offset**: `0x154` from the start of `UserData10` data on both PC and PS4 (PC skips the 0x10 MD5 prefix; PS4 has no prefix).
- **Slot stride and size**: `0x130` bytes (304 bytes) per slot.
- **Active detection**:
  - The slot is active if and only if the four bytes at slot offset `0x18` equal ASCII `"FACE"`.
- **Zeroing active slot**:
  - When active, exactly all `0x130` bytes of the slot (`slotAt .. slotAt+0x130`) are overwritten with `0x00` via `applyByteWrites`.
  - No neighbouring slot, character data, profile summary or other `UserData10` section is modified.
- **Idempotent no-op for inactive slot**:
  - If the slot is already empty (magic at `+0x18` is not `"FACE"`), no byte writes are executed.
  - In accordance with the global `commitRevision` contract, a successful call still increments `saveRevision`, sets `dirty: true`, and clears any previous character undo point.

## Errors

| Situation | Result |
|---|---|
| `engine` is `nil` | `save engine is not available` |
| empty `saveSessionID` | `saveSessionID is required` |
| unknown or closed `saveSessionID` | `unknown save session "<id>"` |
| `favoriteSlotID` outside `0..14` | `favoriteSlotID <id> is outside the range 0..14` |
| `expectedRevision` is non-canonical | `expectedRevision must be a canonical decimal saveRevision; got "<rev>"` |
| `expectedRevision` does not match | `expectedRevision "<rev>" does not match the current saveRevision "<current>"` |
| `UserData10` is truncated or does not cover the requested slot | `favorite preset slot <id> lies outside UserData10 bounds` |

## Transport

```
DELETE /api/v1/save-sessions/{saveSessionID}/favorite-presets/{favoriteSlotID}
```

Request body (JSON, strict, `DisallowUnknownFields`):
```json
{
  "expectedRevision": "0"
}
```

Response (`200 OK`):
```json
{
  "saveSessionID": "sess-12345678",
  "saveRevision": "1",
  "favoriteSlotID": 2
}
```

The route belongs to the local developer explorer in `tools/swagger`. It is
registered only in loopback mode; an explorer started with `-allow-external-bind`
does not register it and answers `404`.

## Local verification

```bash
go test ./backend/saveengine -run '^TestDeleteFavoritePreset' -count=1
go test ./backend/endpoints/favorites -run '^TestDeleteFavoritePreset' -count=1
go test ./tools/swagger -run '^TestDeleteFavoritePresetRoute' -count=1
```
