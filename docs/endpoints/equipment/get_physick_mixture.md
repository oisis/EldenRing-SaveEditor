# GetPhysickMixture

## Overview

`GetPhysickMixture` returns the raw Flask of Wondrous Physick mixture stored in
one physical character slot of a save session that already exists in SaveEngine.
It reads the session's private snapshot only and reports the two Crystal Tear
identifiers exactly as the save stores them.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetPhysickMixture` never creates one,
so calling it before a successful `LoadSave` is an error, not an implicit load.
The endpoint opens no source file, returns no raw save byte, and modifies
nothing: neither the save, nor the session, nor any application state.

| | |
|---|---|
| EndpointID | `get_physick_mixture` |
| Kind | Getter |
| Domain | `equipment` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/physick-mixture` of the local explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. There is no Wails binding, no CLI command and no frontend. |
| Implementation source | [../../../backend/endpoints/equipment/get_physick_mixture.go](../../../backend/endpoints/equipment/get_physick_mixture.go) |
| Test source | [../../../backend/endpoints/equipment/get_physick_mixture_test.go](../../../backend/endpoints/equipment/get_physick_mixture_test.go) |
| Save access | read-only — the session's private in-memory snapshot; no file is opened |
| Mutation | none — the snapshot, the session, and the save file are left unchanged |

## Input

```go
func GetPhysickMixture(engine *saveengine.Engine, saveSessionID string, characterID int) (GetPhysickMixtureResult, error)
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
type GetPhysickMixtureResult = saveengine.CharacterPhysickMixture

type CharacterPhysickMixture struct {
	SaveSessionID string    `json:"saveSessionID"`
	CharacterID   int       `json:"characterID"`
	Active        bool      `json:"active"`
	Tears         [2]uint32 `json:"tears"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was read. It equals the requested value. |
| `characterID` | `int` | The requested slot index, `0` to `9`. It equals the requested value. |
| `active` | `bool` | `true` only when the slot's activity flag is exactly `1`. Any other flag value is not active. |
| `tears` | `[2]uint32` | The two raw Crystal Tear identifiers of an active slot, in stored order. Always two zeros for an inactive slot. |

### Tear order and layout

The mixture lives at the very start of the `EquipPhysicsData` block. It is
exactly two little-endian `uint32` values, read in this order:

| Index | Meaning |
|---|---|
| `tears[0]` | The first mixture position, as the save stores it. |
| `tears[1]` | The second mixture position, as the save stores it. |

The order is the stored order, which is the order of the two mixture positions
in the game. Neither value is sorted, swapped, deduplicated or filtered out, and
the count never varies: there are always exactly two values, including when both
positions hold the same identifier or none at all.

Everything behind those eight bytes in `EquipPhysicsData` is not read.

### Raw values only

Both identifiers are the values stored in the save, reported exactly as read.
The endpoint computes nothing:

- No value is normalised, masked, clamped, repaired, validated or rejected. A
  value of `0`, a value of `0xFFFFFFFF` and a value with its high bit set are all
  returned unchanged, including whatever the game writes for an unfilled
  position.
- Type bits are never stripped and no value is converted into a handle, an index
  or any other derived form.
- Nothing is resolved to a name. **No GameCatalog lookup happens at this stage:**
  no `ItemDocument` is resolved, no Crystal Tear is named, no mixture rule is
  applied, and no value is checked for being a known item.

### JSON shape

`tears` is a fixed-size Go array, so it serialises as a JSON array of exactly two
numbers. It is never encoded as hex, base64 or a string, and its length never
varies.

### What is not returned

The result contains only the fields above. It carries no character name, level,
statistics, appearance, equipment, inventory, quick items, pouch items, spells,
gestures, offsets, and no raw bytes. None of that is read or computed to produce
the result.

### Active, inactive and residual slots

An active slot — activity flag exactly `1` — reports `active: true` and the two
identifiers read from its own slot data.

An inactive slot is a normal result, not an error. It reports `active: false`
with `saveSessionID` and `characterID` filled in and two zeros.

This holds for a residual slot too, where the mixture of a deleted character is
still present in the file: the activity flag alone decides what is reported. An
inactive slot's data is never searched and never read, so its residual mixture is
neither located nor decoded nor analysed.

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
   inside the data of that one slot. The anchor is stated locally in
   `backend/saveengine/physick_mixture.go`; the dynamic locator is shared only
   with `SetPhysickMixture`, so the reader and writer cannot drift apart.
5. The projectile count is read as a raw `uint32` at the constant distance
   `0x931D` behind the anchor. That distance is the sum of the confirmed fixed
   structures between the two positions — `SpEffect` `0x00D0`,
   `EquipedItemIndex` `0x0058`, `ActiveEquipedItems` `0x001C`,
   `EquipedItemsID` `0x0058`, `ActiveEquipedItemsGa` `0x0058`, `InventoryHeld`
   `0x9011`, `EquippedSpells` `0x0074`, `EquipItemData` `0x008C` and
   `EquippedGestures` `0x0018`.
6. The acquired-projectiles section behind that count is variable length: it
   holds exactly `projectileCount` records of eight bytes each. **This is why the
   block cannot be read at a fixed offset:** how far `EquipPhysicsData` sits from
   the anchor depends on how many projectiles the character has acquired, so a
   fixed distance would land inside the projectile records of one save and behind
   the mixture of another.
7. The equipped-armaments block starts at
   `projectileCountOffset + 4 + projectileCount * 8` and is `0x9C` bytes long.
   `EquipPhysicsData` starts immediately behind it, and its first eight bytes are
   the two Tear identifiers.
8. SaveEngine reads exactly those eight bytes and returns both values unchanged.
9. The result is returned by value. The snapshot and the session model stay
   inside the package.

PC and PS4 differ in the base of the slot data only — the PC container puts an
MD5 prefix in front of every slot and the PS4 container does not. The layout
behind that base, including the whole Physick chain, is identical, so both
platforms run the same search and the same reads.

The endpoint is thin by design: it contains no SaveEngine rule and holds no
knowledge of the save format. The shared locator remains private to SaveEngine.
It calls no other endpoint — in particular neither `LoadSave`, `GetLoadedSave`,
`CloseSave`, `GetSaveCharacters`, `GetCharacterProfile`, `GetCharacterStats`,
`GetCharacterAppearance`, `GetEquipment`, `GetQuickItems`, nor `GetPouchItems`.

## Validation and errors

Every failure returns the zero result and changes nothing.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown or closed | `unknown save session "<id>"`. A closed or never-created session is never resolved to a different one. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. Checked only after the session resolves. |
| An active slot carries no anchor | `character <id> carries no physick anchor`. |
| The projectile count itself would lie past the end of the slot | `projectile count of character <id> lies outside its slot`. |
| The slot declares more than `200000` projectile records | `character <id> declares <n> projectile records, want at most 200000`. The count is widened to `int64` before it is multiplied, so a declared length can never wrap into a small, seemingly valid offset. |
| The two Tears would reach past the end of the slot | `physick mixture of character <id> does not fit into its slot`. |
| A required range lies outside the snapshot | The read is rejected before it happens, and the error names the character slot involved. |

The last five rows are fail-closed by design: for an active slot the mixture must
be present and complete where the game keeps it. A missing marker, an implausible
declared length and any position reaching past the slot or the snapshot all fail.
There is no fallback offset, no second candidate position, no partial result and
no guessed value.

An inactive or residual slot is not in this table: it is a successful result.

Stored values are never an error. No Tear identifier is rejected for being
unknown, out of range or implausible.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It reads no GameCatalog data. No value is looked up, named, or validated
  against the catalog.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is
  greenfield.

## Command-line verification

`GetPhysickMixture` is verified through its tests. From the repository root:

```bash
go test ./backend/saveengine -run '^TestGetPhysickMixture' -count=1 -v
go test ./backend/endpoints/equipment -run '^TestGetPhysickMixture' -count=1 -v
```

The tests build synthetic PC and PS4 containers inside `t.TempDir()`. They use no
real save file and no repository fixture, so they depend on nothing outside the
test process. The two platform fixtures place the anchor at different positions
and declare a different, non-zero projectile count, so a reader that depends on a
fixed position inside the slot cannot pass both. Each fixture also writes a decoy
pair at the position the mixture would occupy if the declared count were ignored,
so skipping the dynamic length fails the assertion. Between the two cases the
expected identifiers cover `0`, `0xFFFFFFFF` and values with the high bit set,
and the assertion compares the whole array. The tests also cover a residual slot
whose mixture survives a cleared flag, an empty and an unknown `saveSessionID`,
the rejected `characterID` values `-1` and `10`, a missing anchor, a declared
count of `200001`, and a mixture range reaching past the end of the slot.

## Current limitations

- The only transport is the local explorer route
  `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/physick-mixture`,
  and it exists only while the explorer runs without `-allow-external-bind`.
  No Wails binding, no CLI command and no frontend reaches the endpoint.
- It reports raw state only. No value is resolved into an `ItemDocument`, no
  Crystal Tear name, icon or effect is produced, and GameCatalog is not read.
- Changing the mixture is a separate operation documented under
  [`SetPhysickMixture`](set_physick_mixture.md); this getter itself remains
  read-only.
