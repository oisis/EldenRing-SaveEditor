# GetCharacterStats

## Overview

`GetCharacterStats` returns the raw statistics stored in one physical character
slot of a save session that already exists in SaveEngine. It reads the session's
private snapshot only.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetCharacterStats` never creates one,
so calling it before a successful `LoadSave` is an error, not an implicit load.
The endpoint opens no source file, returns no raw save byte, and modifies
nothing: neither the save, nor the session, nor any application state.

| | |
|---|---|
| EndpointID | `get_character_stats` |
| Kind | Getter |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/stats` of the local OpenAPI explorer (`tools/swagger`). The route is registered only when the explorer runs without `-allow-external-bind`; with an external bind it does not exist and answers 404. No Wails binding, no CLI command, and no frontend reaches the endpoint. |
| Implementation source | [../../../backend/endpoints/character/get_character_stats.go](../../../backend/endpoints/character/get_character_stats.go) |
| Test source | [../../../backend/endpoints/character/get_character_stats_test.go](../../../backend/endpoints/character/get_character_stats_test.go) |
| Save access | read-only — the session's private in-memory snapshot; no file is opened |
| Mutation | none — the snapshot, the session, and the save file are left unchanged |

## Input

```go
func GetCharacterStats(engine *saveengine.Engine, saveSessionID string, characterID int) (GetCharacterStatsResult, error)
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
type GetCharacterStatsResult = saveengine.CharacterStats

type CharacterStats struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	Active        bool   `json:"active"`

	Vigor        uint32 `json:"vigor"`
	Mind         uint32 `json:"mind"`
	Endurance    uint32 `json:"endurance"`
	Strength     uint32 `json:"strength"`
	Dexterity    uint32 `json:"dexterity"`
	Intelligence uint32 `json:"intelligence"`
	Faith        uint32 `json:"faith"`
	Arcane       uint32 `json:"arcane"`
	Level        uint32 `json:"level"`

	HP        uint32 `json:"hp"`
	MaxHP     uint32 `json:"maxHP"`
	BaseMaxHP uint32 `json:"baseMaxHP"`
	FP        uint32 `json:"fp"`
	MaxFP     uint32 `json:"maxFP"`
	BaseMaxFP uint32 `json:"baseMaxFP"`
	SP        uint32 `json:"sp"`
	MaxSP     uint32 `json:"maxSP"`
	BaseMaxSP uint32 `json:"baseMaxSP"`

	Runes      uint32 `json:"runes"`
	SoulMemory uint32 `json:"soulMemory"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was read. It equals the requested value. |
| `saveRevision` | `string` | Opaque revision of the exact session snapshot that produced this result. Clients compare it exactly with the current session revision and discard a mismatch. |
| `characterID` | `int` | The requested slot index, `0` to `9`. It equals the requested value. |
| `active` | `bool` | `true` only when the slot's activity flag is exactly `1`. Any other flag value is not active. |
| `vigor`, `mind`, `endurance`, `strength`, `dexterity`, `intelligence`, `faith`, `arcane` | `uint32` | The eight stored attributes of an active slot. Always `0` for an inactive slot. |
| `level` | `uint32` | The stored character level of an active slot. Always `0` for an inactive slot. |
| `hp`, `maxHP`, `baseMaxHP` | `uint32` | The stored current, maximum and base maximum HP of an active slot. Always `0` for an inactive slot. |
| `fp`, `maxFP`, `baseMaxFP` | `uint32` | The stored current, maximum and base maximum FP of an active slot. Always `0` for an inactive slot. |
| `sp`, `maxSP`, `baseMaxSP` | `uint32` | The stored current, maximum and base maximum stamina of an active slot. Always `0` for an inactive slot. |
| `runes` | `uint32` | The stored held runes of an active slot, the field `SetCharacterRunes` writes. Always `0` for an inactive slot. |
| `soulMemory` | `uint32` | The stored lifetime runes (`TotalGetSoul`) of an active slot. Read-only: no endpoint writes it. Always `0` for an inactive slot. |

### Raw values only

Every numeric field is the `uint32` stored in the save, reported exactly as read.
The endpoint computes nothing:

- No value is validated, normalised, clamped, repaired, or rejected. An attribute
  above the in-game cap, a `level` that does not match the sum of the attributes,
  and an `hp` above `maxHP` are all returned as stored.
- Nothing is derived from anything else. `maxHP` is read, not calculated from
  `vigor`; `level` is read, not calculated from the attributes.

### What is not returned

The result contains only the fields above. It carries no resistances, no equip
load, no spell slots, no level-up cost, no predicted or simulated value, no
starting-class name, no name, no play time, no appearance, no inventory, no
offsets, and no raw bytes. None of that is read or computed to produce it.

### Inactive and residual slots

An inactive slot is a normal result, not an error. It reports `active: false`
with `saveSessionID` and `characterID` filled in and every other field zeroed.

This holds for a residual slot too, where the raw statistics of a deleted
character are still present in the file: the activity flag alone decides what is
reported. An inactive slot's data is never searched and never read, so its
residual attributes, level, HP/FP/SP, runes and `TotalGetSoul` values are neither
located nor decoded.

On any error the result is the zero value.

## Processing flow

1. The endpoint rejects a missing engine.
2. Everything else is delegated to SaveEngine, in this order: `saveSessionID` is
   validated, the session is looked up under the engine's own lock, and
   `characterID` is checked against the slot range.
3. SaveEngine reads the slot's activity flag through the codec's bounded,
   copying reads. An inactive slot returns immediately, without touching the
   slot data.
4. For an active slot only, SaveEngine locates the confirmed statistics anchor
   inside the data of that one slot and reads each confirmed field backwards
   from it. PC and PS4 differ in the base of the slot data only; the layout
   behind it is the same.
5. The result is returned by value. The snapshot and the session model stay
   inside the package.

The endpoint is thin by design: it contains no SaveEngine rule, it holds no
knowledge of the save format, and there is no shared endpoint helper behind it.
It calls no other endpoint — in particular neither `LoadSave`, `GetLoadedSave`,
`CloseSave`, `GetSaveCharacters`, nor `GetCharacterProfile`.

## Validation and errors

Every failure returns the zero result and changes nothing.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown or closed | `unknown save session "<id>"`. A closed or never-created session is never resolved to a different one. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. Checked only after the session resolves. |
| An active slot carries no statistics anchor | `character <id> carries no statistics anchor`. This is a hard error: there is no fallback position, no default offset, and no guessed value. |
| A required range lies outside the snapshot | The read is rejected before it happens, and the error names the character slot involved. |

An inactive or residual slot is not in this table: it is a successful result.

Stored values are never an error. No attribute, level, HP, FP, or SP value is
rejected for being out of range, inconsistent, or implausible.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It reads no GameCatalog data. No field is looked up, named, or validated
  against the catalog.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is
  greenfield.

## Command-line verification

`GetCharacterStats` is verified through its tests. Its only transport is the local
OpenAPI explorer route `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/stats`, which exists solely
when the explorer runs without `-allow-external-bind`. From the repository root:

```bash
go test ./backend/saveengine -run '^TestGetCharacterStats' -count=1 -v
go test ./backend/endpoints/character -run '^TestGetCharacterStats' -count=1 -v
```

The tests build synthetic PC and PS4 containers inside `t.TempDir()`. They use no
real save file and no repository fixture, so they depend on nothing outside the
test process. The two platform fixtures place the anchor at a different position
inside the slot, so a fixed offset instead of a search cannot pass both. They
also cover a residual slot whose raw values survive a cleared flag, the rejected
`characterID` values `-1`, `10`, and `11`, and an active slot without an anchor.

## Current limitations

- The only transport is the local developer explorer route `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/stats`,
  which is registered only without `-allow-external-bind`. There is no Wails
  binding, no CLI command, and no frontend for the endpoint.
- It reports the raw stored statistics only. Runes, derived values, resistances,
  equip load, and spell slots are not readable, and nothing is computed here.
- It is a getter. Changing an attribute, level, HP, FP, or SP is not possible:
  the session is read-only at this stage.
