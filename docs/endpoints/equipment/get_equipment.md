# GetEquipment

## Overview

`GetEquipment` returns the raw equipped state stored in one physical character
slot of a save session that already exists in SaveEngine. It reads the session's
private snapshot only and reports the 22 `ChrAsmEquipment` fields exactly as the
save stores them.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetEquipment` never creates one, so
calling it before a successful `LoadSave` is an error, not an implicit load. The
endpoint opens no source file, returns no raw save byte, and modifies nothing:
neither the save, nor the session, nor any application state.

| | |
|---|---|
| EndpointID | `get_equipment` |
| Kind | Getter |
| Domain | `equipment` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipment` of the local explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. There is no Wails binding, no CLI command and no frontend. |
| Implementation source | [../../../backend/endpoints/equipment/get_equipment.go](../../../backend/endpoints/equipment/get_equipment.go) |
| Test source | [../../../backend/endpoints/equipment/get_equipment_test.go](../../../backend/endpoints/equipment/get_equipment_test.go) |
| Save access | read-only — the session's private in-memory snapshot; no file is opened |
| Mutation | none — the snapshot, the session, and the save file are left unchanged |

## Input

```go
func GetEquipment(engine *saveengine.Engine, saveSessionID string, characterID int) (GetEquipmentResult, error)
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
type GetEquipmentResult = saveengine.CharacterEquipment

type CharacterEquipment struct {
	SaveSessionID string     `json:"saveSessionID"`
	SaveRevision  string     `json:"saveRevision"`
	CharacterID   int        `json:"characterID"`
	Active        bool       `json:"active"`
	Slots         [22]uint32 `json:"slots"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was read. It equals the requested value. |
| `saveRevision` | `string` | Opaque revision of the exact session snapshot that produced this result. Clients compare it exactly with the current session revision and discard a mismatch. |
| `characterID` | `int` | The requested slot index, `0` to `9`. It equals the requested value. |
| `active` | `bool` | `true` only when the slot's activity flag is exactly `1`. Any other flag value is not active. |
| `slots` | `[22]uint32` | The 22 raw equipment fields of an active slot, in stored order. Always all `0` for an inactive slot. |

### Slot index map

The order of `slots` is fixed and is the order the save stores the fields in:

| Index | Meaning | Index | Meaning |
|---|---|---|---|
| 0 | `leftHandArmament1` | 11 | `unknown0x2C` |
| 1 | `rightHandArmament1` | 12 | `head` |
| 2 | `leftHandArmament2` | 13 | `chest` |
| 3 | `rightHandArmament2` | 14 | `arms` |
| 4 | `leftHandArmament3` | 15 | `legs` |
| 5 | `rightHandArmament3` | 16 | `unknown0x40` |
| 6 | `arrows1` | 17 | `talisman1` |
| 7 | `bolts1` | 18 | `talisman2` |
| 8 | `arrows2` | 19 | `talisman3` |
| 9 | `bolts2` | 20 | `talisman4` |
| 10 | `unknown0x28` | 21 | `unknown0x54` |

All 22 fields are always returned, including the three `unknown` fields and
`unknown0x54`. Nothing is filtered out and no index is skipped. The raw getter
reports field 21, but `SetEquippedTalismans` does not treat that technical field
as a player-visible talisman slot and leaves it untouched.

### Raw values only

Every element of `slots` is the `uint32` stored in the save, reported exactly as
read. The endpoint computes nothing:

- No value is normalised, masked, clamped, repaired, validated or rejected. A
  value of `0`, a value of `0xFFFFFFFF` and a value with its high bit set are all
  returned unchanged.
- Type bits are never stripped and no value is converted into a handle, an index
  or any other derived form.
- Nothing is resolved to a name. **No GameCatalog lookup happens at this stage:**
  no `ItemDocument` is resolved, no item is named, no slot restriction is
  applied, and no value is checked for being a known item.

### JSON shape

`slots` is a fixed-size Go array, so it serialises as a JSON array of exactly 22
numbers — `[2147483948, 400, 0, …]`. It is never encoded as hex, base64, an
object or a string, and its length never varies.

### What is not returned

The result contains only the fields above. It carries no character name, level,
statistics, appearance, inventory, quick items, pouch, spells, physick, gestures,
offsets, and no raw bytes. None of that is read or computed to produce the
result.

### Inactive and residual slots

An inactive slot is a normal result, not an error. It reports `active: false`
with `saveSessionID` and `characterID` filled in and an all-zero `slots` array.

This holds for a residual slot too, where the equipment of a deleted character is
still present in the file: the activity flag alone decides what is reported. An
inactive slot's data is never searched and never read, so its residual equipment
is neither located nor decoded nor analysed.

On any error the result is the zero value.

## Processing flow

1. The endpoint rejects a missing engine.
2. Everything else is delegated to SaveEngine, in this order: `saveSessionID` is
   validated, the session is looked up under the engine's own lock, and
   `characterID` is checked against the slot range.
3. SaveEngine reads the slot's activity flag through the codec's bounded,
   copying reads. An inactive slot returns immediately, without touching the
   slot data.
4. For an active slot only, SaveEngine locates the confirmed 65-byte anchor
   inside the data of that one slot.
5. It reads the acquired-projectiles count at a fixed distance behind the anchor.
   That distance is the sum of the confirmed fixed structures between the two
   positions, so it is constant; the projectile section behind it is the one
   variable-length part of the chain.
6. The equipment block starts immediately behind the records the count declares.
   SaveEngine reads exactly 22 little-endian `uint32` values from there and
   returns them unchanged.
7. The result is returned by value. The snapshot and the session model stay
   inside the package.

PC and PS4 differ in the base of the slot data only — the PC container puts an
MD5 prefix in front of every slot and the PS4 container does not. The layout
behind that base, including the whole equipment chain, is identical, so both
platforms run the same search, the same dynamic skip and the same reads.

The endpoint is thin by design: it contains no SaveEngine rule, it holds no
knowledge of the save format, and there is no shared endpoint helper behind it.
It calls no other endpoint — in particular neither `LoadSave`, `GetLoadedSave`,
`CloseSave`, `GetSaveCharacters`, `GetCharacterProfile`, `GetCharacterStats`, nor
`GetCharacterAppearance`.

## Validation and errors

Every failure returns the zero result and changes nothing.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown or closed | `unknown save session "<id>"`. A closed or never-created session is never resolved to a different one. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. Checked only after the session resolves. |
| An active slot carries no anchor | `character <id> carries no equipment anchor`. |
| The projectile count would lie past the end of the slot | `projectile count of character <id> lies outside its slot`. |
| The projectile count is above the accepted maximum | `character <id> declares <n> projectile records, want at most 200000`. |
| The equipment block would reach past the end of the slot | `equipment block of character <id> does not fit into its slot`. |
| A required range lies outside the snapshot | The read is rejected before it happens, and the error names the character slot involved. |

The last five rows are fail-closed by design: for an active slot the equipment
must be present and complete where the game keeps it. A missing anchor, an
implausible declared length and any position reaching past the slot or the
snapshot all fail. There is no fallback offset, no second candidate position, no
partial result and no guessed value.

An inactive or residual slot is not in this table: it is a successful result.

Stored values are never an error. No equipment value is rejected for being
unknown, out of range or implausible.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It reads no GameCatalog data. No value is looked up, named, or validated
  against the catalog.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is
  greenfield.

## Command-line verification

`GetEquipment` is verified through its tests. From the repository root:

```bash
go test ./backend/saveengine -run '^TestGetEquipment' -count=1 -v
go test ./backend/endpoints/equipment -run '^TestGetEquipment' -count=1 -v
```

The tests build synthetic PC and PS4 containers inside `t.TempDir()`. They use no
real save file and no repository fixture, so they depend on nothing outside the
test process. The two platform fixtures place the anchor at different positions
and declare a different, non-zero number of acquired-projectile records, so
neither a fixed offset nor a fixed skip can pass both. Each fixture also writes a
second, equally well-formed 22-field block at the position the real block would
occupy if the declared count were zero, so a reader that ignores the dynamic
length reads the decoy and fails. The fixtures fill all 22 fields with distinct
values including `0`, `0xFFFFFFFF` and values with the high bit set, and the
assertion compares the whole array. They also cover a residual slot whose
equipment survives a cleared flag, the rejected `characterID` values `-1` and
`10`, a missing anchor, an invalid projectile count, and a block reaching past
the end of the slot.

## Current limitations

- The only transport is the local explorer route
  `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipment`,
  and it exists only while the explorer runs without `-allow-external-bind`.
  No Wails binding, no CLI command and no frontend reaches the endpoint.
- It reports raw state only. No value is resolved into an `ItemDocument`, no
  item name, icon or slot restriction is produced, and GameCatalog is not read.
- It is a getter. Changing the equipment is not possible: the session is
  read-only at this stage.
