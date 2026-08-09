# GetCharacterProfile

## Overview

`GetCharacterProfile` returns the confirmed profile of one physical character
slot of a save session that already exists in SaveEngine. It reads the session's
private snapshot only.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetCharacterProfile` never creates
one, so calling it before a successful `LoadSave` is an error, not an implicit
load. The endpoint opens no source file, returns no raw save byte, and modifies
nothing: neither the save, nor the session, nor any application state.

| | |
|---|---|
| EndpointID | `get_character_profile` |
| Kind | Getter |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | not exposed — callable only as a Go function. No Wails binding, no HTTP route, no CLI command, and no frontend reaches it. |
| Implementation source | [../../../backend/endpoints/character/get_character_profile.go](../../../backend/endpoints/character/get_character_profile.go) |
| Test source | [../../../backend/endpoints/character/get_character_profile_test.go](../../../backend/endpoints/character/get_character_profile_test.go) |
| Save access | read-only — the session's private in-memory snapshot; no file is opened |
| Mutation | none — the snapshot, the session, and the save file are left unchanged |

## Input

```go
func GetCharacterProfile(engine *saveengine.Engine, saveSessionID string, characterID int) (GetCharacterProfileResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; the endpoint never creates one. A `nil` engine is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. It is passed to SaveEngine unchanged. |
| `characterID` | `int` | The physical slot index, `0` to `9`. It is the same index `GetSaveCharacters` reports positionally. |

### `saveSessionID`

- It is matched exactly and case-sensitively. It is never trimmed, normalised,
  or guessed, so `" <id>"`, `"<id> "`, and an upper-cased identifier are unknown
  values, not the session they resemble.
- Validation lives in SaveEngine. The endpoint holds no session-identifier rule
  of its own.

### `characterID`

- It is an index, not an identifier to search for: slot `n` is read directly.
- A value below `0` or above `9` is rejected. It is never clamped to the valid
  range and never resolved to a neighbouring slot.

## Output

```go
type GetCharacterProfileResult = saveengine.CharacterProfile

type CharacterProfile struct {
	SaveSessionID   string `json:"saveSessionID"`
	CharacterID     int    `json:"characterID"`
	Active          bool   `json:"active"`
	Name            string `json:"name"`
	Level           uint32 `json:"level"`
	StartingClassID uint8  `json:"startingClassID"`
	Gender          uint8  `json:"gender"`
	SecondsPlayed   uint32 `json:"secondsPlayed"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was read. It equals the requested value. |
| `characterID` | `int` | The requested slot index, `0` to `9`. It equals the requested value. |
| `active` | `bool` | `true` only when the slot's activity flag is exactly `1`. Any other flag value is not active. |
| `name` | `string` | The character name of an active slot. Always `""` for an inactive slot. |
| `level` | `uint32` | The character level of an active slot. Always `0` for an inactive slot. |
| `startingClassID` | `uint8` | The raw starting-class identifier of an active slot, reported exactly as stored. Always `0` for an inactive slot. |
| `gender` | `uint8` | The raw gender identifier of an active slot. The confirmed values are `0` for Type B and `1` for Type A. Always `0` for an inactive slot. |
| `secondsPlayed` | `uint32` | The raw play time of an active slot, in seconds. Always `0` for an inactive slot. |

### Raw identifiers

`startingClassID` and `gender` are raw save identifiers. They are passed through
unchanged: they are not mapped to a display name, they are not read through
GameCatalog, and an unknown value is neither rejected nor replaced. Turning them
into text is the caller's decision, not this endpoint's.

`secondsPlayed` is likewise a raw number of seconds. No formatted duration is
produced here.

### Inactive and residual slots

An inactive slot is a normal result, not an error. It reports `active: false`
with `saveSessionID` and `characterID` filled in and every other field zeroed.

This holds for a residual slot too, where the raw `UserData10` summary of a
deleted character is still present in the file: the activity flag alone decides
what is reported, and the summary of an inactive slot is never read at all, so
the residual name, level, class, gender, and play time are neither decoded nor
returned.

### What is not returned

The result contains only the confirmed fields above. It carries no `SteamID`, no
MD5, no `UserData11` data, no NG+ level, no location, no class name, no
inventory, no appearance data, no offsets, and no raw bytes. None of that is read
to produce it.

On any error the result is the zero value.

## Processing flow

1. The endpoint rejects a missing engine.
2. Everything else is delegated to SaveEngine, in this order: `saveSessionID` is
   validated, the session is looked up under the engine's own lock, and
   `characterID` is checked against the slot range.
3. SaveEngine reads the slot's activity flag through the codec's bounded,
   copying reads. An inactive slot returns immediately, without reading the
   summary.
4. For an active slot only, SaveEngine decodes the confirmed name, level,
   play time, gender, and starting class, and returns the result by value. The
   snapshot and the session model stay inside the package.

The endpoint is thin by design: it contains no SaveEngine rule, it holds no
knowledge of the save format, and there is no shared endpoint helper behind it.
It calls no other endpoint — in particular neither `LoadSave`, `GetLoadedSave`,
`CloseSave`, nor `GetSaveCharacters`.

## Validation and errors

Every failure returns the zero result and changes nothing.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown or closed | `unknown save session "<id>"`. A closed or never-created session is never resolved to a different one. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. Checked only after the session resolves. |
| A required range lies outside the snapshot | The read is rejected before it happens, and the error names the character slot involved. |

An inactive or residual slot is not in this table: it is a successful result.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It reads no GameCatalog data.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is
  greenfield.

## Command-line verification

`GetCharacterProfile` has no transport, so it is verified through its tests. From
the repository root:

```bash
go test ./backend/saveengine -run '^TestGetCharacterProfile' -count=1 -v
go test ./backend/endpoints/character -run '^TestGetCharacterProfile' -count=1 -v
```

The tests build synthetic PC and PS4 containers inside `t.TempDir()`. They use no
real save file and no repository fixture, so they depend on nothing outside the
test process. They cover an active profile on both platforms, a residual slot
whose raw values survive a cleared flag, and the rejected `characterID` values
`-1`, `10`, and `11`.

## Current limitations

- The endpoint is not exposed through Wails, HTTP, or a CLI, and there is no
  frontend for it. The local OpenAPI explorer does not route to it.
- It reports the confirmed profile fields only. Stats, appearance, inventory,
  storage, equipment, and world state are not readable yet.
- It is a getter. Changing a name, class, gender, or play time is not possible:
  the session is read-only at this stage.
