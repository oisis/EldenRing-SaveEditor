# GetSaveCharacters

## Overview

`GetSaveCharacters` returns a summary of all ten physical character slots of a
save session that already exists in SaveEngine. It reads the session's private
snapshot only, and it always reports exactly ten slots in slot order.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetSaveCharacters` never creates
one, so calling it before a successful `LoadSave` is an error, not an implicit
load. The endpoint opens no source file, returns no raw save byte, and modifies
nothing: neither the save, nor the session, nor any application state.

| | |
|---|---|
| EndpointID | `get_save_characters` |
| Kind | Getter |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters` of the local OpenAPI explorer (`tools/swagger`). The route is registered only when the explorer runs without `-allow-external-bind`; with an external bind it does not exist and answers 404. No Wails binding, no CLI command, and no frontend reaches the endpoint. |
| Implementation source | [../../../backend/endpoints/character/get_save_characters.go](../../../backend/endpoints/character/get_save_characters.go) |
| Test source | [../../../backend/endpoints/character/get_save_characters_test.go](../../../backend/endpoints/character/get_save_characters_test.go) |
| Save access | read-only — the session's private in-memory snapshot; no file is opened |
| Mutation | none — the snapshot, the session, and the save file are left unchanged |

## Input

```go
func GetSaveCharacters(engine *saveengine.Engine, saveSessionID string) (GetSaveCharactersResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; the endpoint never creates one. A `nil` engine is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. It is passed to SaveEngine unchanged. |

### `saveSessionID`

- It is matched exactly and case-sensitively. It is never trimmed, normalised,
  or guessed, so `" <id>"`, `"<id> "`, and an upper-cased identifier are unknown
  values, not the session they resemble.
- Validation lives in SaveEngine. The endpoint holds no session-identifier rule
  of its own.

## Output

```go
type GetSaveCharactersResult = saveengine.SaveCharacters

type SaveCharacters struct {
	SaveSessionID string             `json:"saveSessionID"`
	SaveRevision  string             `json:"saveRevision"`
	Characters    []CharacterSummary `json:"characters"`
}

type CharacterSummary struct {
	CharacterID int    `json:"characterID"`
	Active      bool   `json:"active"`
	Name        string `json:"name"`
	Level       uint32 `json:"level"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was read. It equals the requested value. |
| `saveRevision` | `string` | Opaque revision of the exact session snapshot that produced this result. Clients compare it exactly with the current session revision and discard a mismatch. |
| `characters` | `[]CharacterSummary` | Always exactly ten entries, one per physical slot, in slot order `0..9`. The slice is never `nil` on success. |
| `characterID` | `int` | The slot index, `0` to `9`. It is positional, so no separate slot field exists and none is needed. |
| `active` | `bool` | `true` only when the slot's activity flag is exactly `1`. Any other flag value is not active. |
| `name` | `string` | The character name of an active slot. Always `""` for an inactive slot. |
| `level` | `uint32` | The character level of an active slot. Always `0` for an inactive slot. |

### Inactive and residual slots

An inactive slot always reports `name: ""` and `level: 0`. This holds even for a
residual slot, where the raw `UserData10` summary of a deleted character is
still present in the file: the activity flag alone decides what is reported, and
the residual name and level are neither decoded nor returned.

### What is not returned

The result contains only the confirmed fields above. It carries no `SteamID`,
no play time, no class, no NG+ level, no location, no save version, no
appearance data, no inventory, no offsets, and no raw bytes. None of that is
read to produce it.

On any error the result is the zero value.

## Processing flow

1. The endpoint rejects a missing engine.
2. Everything else is delegated to SaveEngine: it validates `saveSessionID`,
   looks the session up under its own lock, and reads the snapshot through the
   codec's bounded, copying reads.
3. SaveEngine reads the ten slot activity flags, then decodes the name and level
   of the active slots only, and returns the result by value. The snapshot and
   the session model stay inside the package.

The endpoint is thin by design: it contains no SaveEngine rule, it holds no
knowledge of the save format, and there is no shared endpoint helper behind it.
It calls no other endpoint — in particular neither `LoadSave`, `GetLoadedSave`,
nor `CloseSave`.

## Validation and errors

Every failure returns the zero result and changes nothing.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown or closed | `unknown save session "<id>"`. A closed or never-created session is never resolved to a different one. |
| A required range lies outside the snapshot | The read is rejected before it happens, and the error names the character slot involved. |

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It reads no GameCatalog data.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is
  greenfield.

## Command-line verification

`GetSaveCharacters` is verified through its tests. Its only transport is the local
OpenAPI explorer route `GET /api/v1/save-sessions/{saveSessionID}/characters`, which exists solely
when the explorer runs without `-allow-external-bind`. From the repository root:

```bash
go test ./backend/saveengine -run '^TestGetSaveCharacters' -count=1 -v
go test ./backend/endpoints/character -run '^TestGetSaveCharacters' -count=1 -v
```

The tests build synthetic PC and PS4 containers inside `t.TempDir()`. They use
no real save file and no repository fixture, so they depend on nothing outside
the test process. They cover an active slot, an inactive slot, a residual slot
whose raw name and level survive a cleared flag, and an active name that fills
the whole field without a NUL terminator.

## Current limitations

- The only transport is the local developer explorer route `GET /api/v1/save-sessions/{saveSessionID}/characters`,
  which is registered only without `-allow-external-bind`. There is no Wails
  binding, no CLI command, and no frontend for the endpoint.
- It reports name, level, and activity only. Stats, appearance, inventory,
  storage, equipment, and world state are not readable yet.
- It is a getter. Creating, deleting, cloning, or renaming a character is not
  possible: the session is read-only at this stage.
