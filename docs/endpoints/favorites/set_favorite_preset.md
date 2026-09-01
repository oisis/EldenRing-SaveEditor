# SetFavoritePreset

## Overview

`SetFavoritePreset` saves all appearance fields represented by Mirror Favorites
from an active character into the specified preset slot stored in `UserData10` of
an existing save session under `expectedRevision` control. It operates on the
session's private snapshot only.

The Mirror Favorites slots are shared across all 10 character slots of the save
file (they live globally in `UserData10`, not inside individual character
slots).

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `SetFavoritePreset` opens no source
file, reads no GameCatalog, requires no selection object, and persists nothing
directly; persistence is owned by a separate
[`WriteSave`](../savesession/write_save.md).

| | |
|---|---|
| EndpointID | `set_favorite_preset` |
| Kind | Mutation |
| Domain | `favorites` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/favorite-presets/{favoriteSlotID}` of the local OpenAPI explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`. There is no Wails binding, no frontend view and no CLI command. |
| Implementation source | [../../../backend/endpoints/favorites/set_favorite_preset.go](../../../backend/endpoints/favorites/set_favorite_preset.go) |
| Test source | [../../../backend/endpoints/favorites/set_favorite_preset_test.go](../../../backend/endpoints/favorites/set_favorite_preset_test.go) |
| Data source | the specified character's appearance in slot data and the global Mirror Favorites slot in `UserData10`, mutated by SaveEngine |
| Save access | private session snapshot mutation with verification and rollback; no file is opened |
| Mutation | writes the full 0x130-byte range of the target slot; advances `saveRevision`, sets `dirty` flag and invalidates the session's undo point |

## Input

```go
func SetFavoritePreset(
	engine *saveengine.Engine,
	saveSessionID string,
	favoriteSlotID int,
	sourceCharacterID int,
	expectedRevision string,
) (SetFavoritePresetResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance that owns the loaded sessions. A `nil` engine is rejected. |
| `saveSessionID` | `string` | Identifier of an already loaded session. It is passed to SaveEngine unchanged. |
| `favoriteSlotID` | `int` | Slot index in `0..14` identifying which of the 15 preset slots to write or replace. |
| `sourceCharacterID` | `int` | Slot index in `0..9` identifying the active character that supplies appearance data. |
| `expectedRevision` | `string` | Canonical decimal `saveRevision` matching the current revision of the session. |

### `saveSessionID`

- Matched exactly and case-sensitively by SaveEngine. It is never trimmed,
  normalised or guessed, so an empty, unknown or already closed identifier is
  rejected before any mutation occurs.

### `favoriteSlotID`

- An integer in `0..14`.
- Any value outside `0..14` (e.g. `-1` or `15`) is rejected.
- The parameter is never clamped or wrapped.

### `sourceCharacterID`

- An integer in `0..9`.
- Must point to an active character slot (`UserData10` active flag equals `1`).
- The source character's appearance anchor and `FACE` block must be well-formed.

### `expectedRevision`

- Must be the canonical decimal string representation of a non-negative integer
  without leading zeros (e.g. `"0"`, `"1"`).
- Checked under the session lock inside `commitRevision`. If it does not match
  the session's current `saveRevision`, the operation fails with no changes.

## Result

```go
type SetFavoritePresetResult struct {
	MutationReceipt
	FavoriteSlotID    int    `json:"favoriteSlotID"`
	SourceCharacterID int    `json:"sourceCharacterID"`
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
receipt members and `favoriteSlotID`, `sourceCharacterID` all sit at the top
level, and there is no nested `receipt` object.

The embedded `saveengine.MutationReceipt` is exactly the receipt the central
SaveEngine commit path produced for this execution. Nothing here is
reassembled from the EndpointID, the session, the revision or a scope lookup.

- `operationID` names this one execution. It is opaque and unpredictable.
  Identifiers do not repeat among the receipts issued by one running SaveEngine
  instance. That guarantee does not currently cover application restarts:
  uniqueness across restarts requires a persistent operation journal and stays
  outside this stage. A rejected call returns the complete zero result and no
  `operationID` at all.
- `operationKind` is the stable kind of the mutation and is always exactly
  `set_favorite_preset`.
- `changedScopes` are exactly `save.session`, `favorites`, `diagnostics.report`,
  in that canonical order.

| Field | Type | Meaning |
|---|---|---|
| `operationID` | `string` | Opaque identifier of this one execution. |
| `operationKind` | `string` | Stable kind of the mutation, exactly `set_favorite_preset`. |
| `saveSessionID` | `string` | Identifier of the session that was modified. |
| `saveRevision` | `string` | New canonical decimal save revision after the mutation (incremented by 1). |
| `changedScopes` | `[]string` | Backend read scopes this mutation invalidated, in the one canonical order. |
| `favoriteSlotID` | `int` | Physical slot index in `0..14` that was written. |
| `sourceCharacterID` | `int` | Source character slot index in `0..9` that supplied the appearance data. |

## Binary layout of the preset slot

The slot buffer is exactly `0x130` (304) bytes built by SaveEngine ([`backend/saveengine/set_favorite_preset.go`](../../../backend/saveengine/set_favorite_preset.go)):

- `+0x00` (2 bytes, u16 le): `0xFACE` magic
- `+0x02` (2 bytes): `0x00 0x00` pad
- `+0x04` (4 bytes, u32 le): `0x11D0` constant header value
- `+0x08` (1 byte, u8): body flag = `1`
- `+0x09` (1 byte, u8): body type (`0` for male, `1` for female; inverted relative to character slot gender `1`=male, `0`=female)
- `+0x0A` (10 bytes): `0x00` pad
- `+0x14` (4 bytes, i32 le): active marker (`0x00000000`)
- `+0x18` (4 bytes, ASCII): `"FACE"` magic
- `+0x1C` (4 bytes, u32 le): alignment (`4`)
- `+0x20` (4 bytes, u32 le): inner size (`0x120` / 288)
- `+0x24` (32 bytes): 8 model IDs (uint32 le each) copied from source character `FaceData[0x10..0x30]`
- `+0x44` (64 bytes): face shape sliders copied from source character `FaceData[0x30..0x70]`
- `+0x84` (64 bytes): opaque/unknown block copied from source character `FaceData[0x70..0xB0]`
- `+0xC4` (7 bytes): body proportions copied from source character `FaceData[0xB0..0xB7]`
- `+0xCB` (91 bytes): skin & cosmetics copied from source character `FaceData[0xB7..0x112]`
- `+0x126` (10 bytes): `0x00` trailing pad

VoiceType is not stored in Mirror Favorites and is not included.
If the target slot already contains identical bytes, the mutation is a byte-level no-op, but advances `saveRevision`, sets `dirty: true`, and clears undo as standard for `commitRevision`.

## Errors

| Situation | Result |
|---|---|
| `engine` is `nil` | `save engine is not available` |
| empty `saveSessionID` | `saveSessionID is required` |
| unknown or closed `saveSessionID` | `unknown save session "<id>"` |
| `favoriteSlotID` outside `0..14` | `favoriteSlotID <id> is outside the range 0..14` |
| `sourceCharacterID` outside `0..9` | `sourceCharacterID <id> is outside the range 0..9` |
| character is not active | `character <id> is not active` |
| character carries no appearance anchor | `character <id> carries no appearance player anchor` |
| character carries no appearance block | `character <id> carries no appearance block` |
| `expectedRevision` is non-canonical | `expectedRevision must be a canonical decimal saveRevision; got "<rev>"` |
| `expectedRevision` does not match | `expectedRevision "<rev>" does not match the current saveRevision "<current>"` |
| `UserData10` is truncated or does not cover the requested slot | `favorite preset slot <id> lies outside UserData10 bounds` |

## Transport

```
PUT /api/v1/save-sessions/{saveSessionID}/favorite-presets/{favoriteSlotID}
```

Request body (JSON, strict, `DisallowUnknownFields`):
```json
{
  "sourceCharacterID": 0,
  "expectedRevision": "0"
}
```

Response (`200 OK`):
```json
{
  "operationID": "op-0f1e2d3c4b5a69788796a5b4c3d2e1f0",
  "operationKind": "set_favorite_preset",
  "saveSessionID": "sess-12345678",
  "saveRevision": "1",
  "changedScopes": ["save.session", "favorites", "diagnostics.report"],
  "favoriteSlotID": 2,
  "sourceCharacterID": 0
}
```

The route belongs to the local developer explorer in `tools/swagger`. It is
registered only in loopback mode; an explorer started with `-allow-external-bind`
does not register it and answers `404`.

## Local verification

```bash
go test ./backend/saveengine -run '^TestSetFavoritePreset' -count=1
go test ./backend/endpoints/favorites -run '^TestSetFavoritePreset' -count=1
go test ./tools/swagger -run '^TestSetFavoritePresetRoute' -count=1
```
