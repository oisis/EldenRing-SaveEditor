# GetGestures

## Overview

`GetGestures` returns every gesture GameCatalog knows, each one marked with
whether one physical character slot of a save session that already exists in
SaveEngine has unlocked it. It joins exactly two sources and nothing else:

| Source | What it provides |
|---|---|
| GameCatalog | the gesture definitions: resource `kind` and `key`, the canonical `slotID`, the `name` and the `category` |
| SaveEngine | the raw 64-record `GestureGameData` block of the requested slot |

A gesture is reported as unlocked when its canonical `slotID` is present in that
block as an **exact `uint32` match**. Nothing else decides the state.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetGestures` never creates one, so
calling it before a successful `LoadSave` is an error, not an implicit load. The
endpoint opens no source file, returns no raw save byte, and modifies nothing:
neither the save, nor the session, nor GameCatalog, nor any application state. It
also calls no other endpoint.

| | |
|---|---|
| EndpointID | `get_gestures` |
| Kind | Getter |
| Domain | `world` |
| Supported resource types | `ItemDocument: Gesture` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gestures` of the local explorer (`backend/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. There is no Wails binding, no CLI command and no frontend. |
| Implementation source | [../../../backend/endpoints/world/get_gestures.go](../../../backend/endpoints/world/get_gestures.go) |
| Test source | [../../../backend/endpoints/world/get_gestures_test.go](../../../backend/endpoints/world/get_gestures_test.go) |
| Save access | read-only — the session's private in-memory snapshot; no file is opened |
| Mutation | none — the snapshot, the session, GameCatalog and the save file are left unchanged |

## Input

```go
func GetGestures(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	availabilityFilter string,
) (GetGesturesResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; the endpoint never creates one. A `nil` engine is rejected. |
| `gameCatalog` | `*gamecatalog.Catalog` | The catalog the gesture definitions come from. A `nil` catalog is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. It is passed to SaveEngine unchanged. |
| `characterID` | `int` | The physical slot index, `0` to `9`. It is the same index `GetSaveCharacters` reports positionally. |
| `availabilityFilter` | `string` | Unlock-state filter: `""`, `"unlocked"` or `"locked"`. |

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

### `availabilityFilter`

| Value | Meaning |
|---|---|
| `""` | Every gesture the catalog knows. |
| `"unlocked"` | Only the gestures whose canonical `slotID` is present in the block. |
| `"locked"` | Only the rest. |

Any other value is rejected. The comparison is exact and case-sensitive, the
value is never trimmed, and there is no alias: `"Unlocked"`, `" unlocked"`,
`"unlocked "`, `"LOCKED"` and `"all"` are all errors, not the filter they
resemble.

The filter can only **remove** entries. It never sorts, regroups or renumbers
them, so the entries it keeps appear in exactly the order and with exactly the
state the unfiltered result gives them.

## Output

```go
type GestureEntry struct {
	Kind     schema.ResourceKind `json:"kind"`
	Key      string              `json:"key"`
	SlotID   uint32              `json:"slotID"`
	Name     string              `json:"name"`
	Category string              `json:"category"`
	Unlocked bool                `json:"unlocked"`
}

type GetGesturesResult struct {
	SaveSessionID string         `json:"saveSessionID"`
	CharacterID   int            `json:"characterID"`
	Active        bool           `json:"active"`
	Gestures      []GestureEntry `json:"gestures"`
}
```

```json
{
  "saveSessionID": "9f1c…",
  "characterID": 0,
  "active": true,
  "gestures": [
    {
      "kind": "item",
      "key": "4000233E",
      "slotID": 105,
      "name": "By My Sword",
      "category": "Battle",
      "unlocked": false
    },
    {
      "kind": "item",
      "key": "40002328",
      "slotID": 1,
      "name": "Bow",
      "category": "Greetings",
      "unlocked": true
    }
  ]
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was read. It equals the requested value. |
| `characterID` | `int` | The requested slot index, `0` to `9`. It equals the requested value. |
| `active` | `bool` | `true` only when the slot's activity flag is exactly `1`. Any other flag value is not active. |
| `gestures` | `[]GestureEntry` | The gestures that passed `availabilityFilter`, in the deterministic order below. Empty, never `null`. |

| Entry field | Type | Meaning |
|---|---|---|
| `kind` | `string` | The GameCatalog resource kind. Always `item` for a gesture. |
| `key` | `string` | The GameCatalog resource key. |
| `slotID` | `uint32` | The canonical save slot ID of this one gesture. |
| `name` | `string` | The gesture name stored in GameCatalog. Never empty. |
| `category` | `string` | The gesture category stored in GameCatalog. Never empty. |
| `unlocked` | `bool` | `true` when this canonical `slotID` is present in the raw block. |

Nothing else is returned. No provenance, no `sourceRecords`, no icon content, no
`ItemDocument`, no raw save byte and no field unrelated to this getter.

### `kind` and `key` are the resource, `slotID` is the gesture

`kind` together with `key` is the public GameCatalog identity of the resource.
`slotID` identifies the gesture **inside** that resource, because one resource
may declare more than one entry in `gesture.slots`. Two entries then share `kind`
and `key` and differ in `slotID`, `name` and `category`. In the current stored
catalog, 56 gesture documents declare 57 gesture slots for exactly this reason.

### Order

The order is deterministic and does not depend on the unlock state:

1. `category`, ascending;
2. then `name`, ascending;
3. then `slotID`, ascending, as the tie-breaker between two slots that share a
   category and a name.

Sorting is never done by `unlocked`, and applying a filter never reorders what
remains.

## Where the unlock state comes from

### `GestureGameData`, not the equipped gesture ring

Two different structures hold gesture identifiers, and this getter reads only one
of them:

| Structure | Size | Meaning | Read here |
|---|---|---|---|
| `EquippedGestures` (the gesture ring) | `0x18` bytes, six `uint32` | the gestures currently assigned to the quick-access wheel | **no** |
| `GestureGameData` | `0x100` bytes, 64 `uint32` | every gesture the character has unlocked | **yes** |

The gesture ring is a small subset the player has mapped to buttons; it says
nothing about ownership. `GestureGameData` is the full unlock list, and it is the
only structure `GetGestures` reads.

### Event Flags are not used

This getter reads **no** event flag. Unlock state is decided entirely by the
presence of a canonical `slotID` inside `GestureGameData`.

> **Specification divergence.** `spec/08-spells-gestures.md` states that gesture
> unlock is controlled by Event Flags in the range 60800–60849, and its gesture
> table lists the even `GestureParam` row IDs rather than the canonical save slot
> IDs. That contradicts the verified working read path of SaveForge 1.6.8, which
> touches no event flag and matches the odd canonical IDs. The behaviour
> implemented here follows the verified read path. The specification is not
> modified by this endpoint and the divergence remains open.

### Canonical `slotID`

The canonical save slot ID is the value GameCatalog stores in
`item.gesture.slots[].slotID`, derived from the `GestureParam` row ID as
`rowID × 2 + 1`. Every canonical gesture slot ID is therefore odd.

- The match is an exact `uint32` comparison against the stored records.
- There is **no** even/odd "body type" theory. An earlier SaveForge build wrote
  even identifiers that the game silently ignored; that model is wrong and is not
  reproduced here.
- An even stored value is **never** converted into the odd value next to it, and
  the odd value next to it is **not** unlocked because of it.

### Sentinels, zero, unknown values and duplicates

| Stored value | Effect |
|---|---|
| `0xFFFFFFFE` | The native empty-slot sentinel. Unlocks nothing. |
| `0x00000000` | An untouched record. Unlocks nothing. |
| any value matching no canonical `slotID` | Unlocks nothing, whether odd or even. |
| the same canonical `slotID` stored twice | Its one catalog entry is `unlocked: true`. The entry is never duplicated. |

Nothing is sanitised, repaired, normalised or rewritten. SaveEngine returns all
64 records exactly as stored, including sentinels, zeros, duplicates and values
this stage cannot explain; the endpoint only decides which catalog entries those
values match.

## How `GestureGameData` is located

The block has **no fixed position** inside a slot. It sits directly behind the
Storage Box, which sits behind the face data, which sits behind
`EquipPhysicsData` and the equipped-armaments block, which themselves sit behind
the variable-length acquired-projectiles section. It is therefore located from
the confirmed 65-byte slot anchor across the one dynamic length the save itself
declares:

| Distance | Content |
|---|---|
| anchor `+ 0x931D` | `uint32` acquired-projectile count |
| `+ 4 + count × 8` | end of the projectile records |
| `+ 0x9C` | end of the equipped-armaments block |
| `+ 0x0C` | end of `EquipPhysicsData` |
| `+ 0x12F` | end of the face data — the first byte of the Storage Box |
| `+ 0x6010` | end of the Storage Box — **the first byte of `GestureGameData`** |

The `0x931D` distance is the sum of the confirmed fixed structures between the
anchor and the projectile count: `SpEffect` `0xD0`, `EquipedItemIndex` `0x58`,
`ActiveEquipedItems` `0x1C`, `EquipedItemsID` `0x58`, `ActiveEquipedItemsGa`
`0x58`, `InventoryHeld` `0x9011`, `EquippedSpells` `0x74`, `EquipItemData` `0x8C`
and `EquippedGestures` `0x18`. The `0x6010` distance is the confirmed size of the
Storage Box; nothing inside it is read, decoded or validated here.

The block itself is `0x100` bytes: 64 little-endian `uint32` records, always read
whole. The declared projectile count is widened to `int64` before it is
multiplied, so a declared length can never wrap into a small, seemingly valid
offset, and a count above `200000` is treated as corrupt instead of followed.

The gesture reader in
[`backend/saveengine/gestures.go`](../../../backend/saveengine/gestures.go) owns
its own anchor, its own layout constants, its own location algorithm and its own
bounds checks. It calls neither `GetStorage` nor any other getter, and it borrows
no private constant or helper from one.

## Active, inactive and residual slots

An inactive slot — including a residual one, whose deleted character's gesture
block is still present in the file — is a successful result, not an error:

- `active` is `false`;
- every catalog gesture is reported with `unlocked: false`, subject to
  `availabilityFilter`;
- **the slot data is never searched or read.** The result comes from the
  UserData10 activity flag alone, so a residual gesture block is never located,
  decoded or exposed.

With `availabilityFilter` `"unlocked"` an inactive slot therefore returns an
empty, non-`null` list.

## PC and PS4

Both platforms are supported and read identically. The `GestureGameData` layout
inside a character slot is the same on PC and PS4; the containers differ only in
where a slot begins:

| Platform | Slot data base |
|---|---|
| PC | slot block offset plus the `0x10`-byte MD5 prefix, which is skipped and never parsed |
| PS4 | the slot block offset itself; the PS4 container stores no MD5 prefix |

Those two bases live in
[`backend/saveengine/gestures_pc.go`](../../../backend/saveengine/gestures_pc.go)
and
[`backend/saveengine/gestures_ps4.go`](../../../backend/saveengine/gestures_ps4.go).
The platform split exists from the start; it is not deferred to a later refactor.

## Processing flow

1. A `nil` engine, a `nil` catalog and an unknown `availabilityFilter` are
   rejected by the endpoint.
2. SaveEngine validates `saveSessionID`, resolves the session and validates
   `characterID`.
3. The UserData10 activity flag of the requested slot is read. A flag other than
   `1` ends the save-side read with an inactive result.
4. For an active slot the slot data bounds come from the platform entry point,
   the confirmed 65-byte anchor is searched inside that one slot, and the block is
   located across the projectile count, the three fixed blocks and the Storage
   Box. All 64 records are returned raw.
5. The endpoint builds the set of stored values, skipping only `0xFFFFFFFE` and
   `0`, because neither can be a canonical `slotID`.
6. Every catalog resource of `kind` `item` whose family is known and equal to
   `gesture` is read in full, and each of its declared gesture slots becomes one
   entry.
7. Each entry is marked unlocked when its canonical `slotID` is in the set,
   sorted by category, name and `slotID`, and filtered by `availabilityFilter`.

## Validation and errors

Every failure returns the zero result and changes nothing.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `gameCatalog` is `nil` | `game catalog is not available` — likewise a wiring error. |
| `availabilityFilter` is not `""`, `"unlocked"` or `"locked"` | `availabilityFilter must be "unlocked", "locked" or empty; got "<value>"`. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown or closed | `unknown save session "<id>"`. A closed or never-created session is never resolved to a different one. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. Checked only after the session resolves. |
| An active slot carries no anchor | `character <id> carries no gesture anchor`. |
| The projectile count lies outside the slot | `projectile count of character <id> lies outside its slot`. |
| The declared projectile count is above the accepted maximum | `character <id> declares <n> projectile records, want at most 200000`. |
| The block would reach past the end of the slot | `gestures of character <id> do not fit into their slot`. |
| A gesture resource has no key, no item document or no gesture document | fail-closed error naming the resource. |
| A gesture slot has no known `slotID`, name or category | fail-closed error naming the resource and the slot index. |

The save-side rows are fail-closed by design: for an active slot the required
structure must be present and complete where the game keeps it. A missing anchor,
an implausible declared length, a truncated block and any position reaching past
the slot or the snapshot all fail. There is no fallback offset, no second
candidate position, no partial result and no guessed value.

The catalog-side rows are fail-closed for the same reason: a missing name,
category or `slotID` produces an error, never an empty string, a default category
or an invented identifier.

An inactive or residual slot is not in this table: it is a successful result.

Stored values are never an error. No record is rejected for being unknown, even,
duplicated or implausible; it simply matches no canonical `slotID`.

## Read-only behaviour

- The endpoint reads the session's private in-memory snapshot through the codec
  only. No file is opened, written, repaired, saved or reloaded.
- No snapshot byte leaves SaveEngine; the endpoint returns decoded values only.
- GameCatalog is read through its existing public methods `ResourceSummaries` and
  `ResourceByKindAndKey`. No catalog index, cache or method is added, and the
  catalog is never modified.
- Nothing is normalised, repaired or resaved as a side effect of reading.

## Dependencies

- The endpoint delegates to `backend/saveengine` and `backend/gamecatalog` and
  calls no other endpoint.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is greenfield.
  Earlier SaveForge versions (1.5.8 and 1.6.8) were used as research material to
  confirm the binary format and the unlock rule only; no legacy code is imported,
  reused or depended on at runtime.

## Swagger route

```
GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gestures
    ?availabilityFilter=
```

The route is registered only by `registerSaveSessionRoutes`, which the explorer
calls only when it binds a loopback address. An explorer started with
`-allow-external-bind` does not register it and answers `404`.

`characterID` is parsed by the existing Swagger convention: decimal only, never
trimmed and never defaulted, so `"one"`, `" 0"` and `"0x1"` are rejected by the
route before SaveEngine sees them. `availabilityFilter` is handed over **exactly
as it arrived** — it is never trimmed, lower-cased or defaulted by the transport,
so the getter owns both the rule and the wording of its rejection. The route
calls `world.GetGestures` and nothing else, returns `200` on success and `400` on
any error, and logs neither save data, nor character names, nor the source path.

### Calling it from the Scalar portal

1. Start the local explorer without `-allow-external-bind` and open the portal.
2. Open **API Reference → World → GetGestures**, or the endpoint page under
   **World → GetGestures**.
3. Create a session first with **Save sessions → LoadSave** (`POST
   /api/v1/save-sessions`) and copy the returned `saveSessionID`.
4. Paste it into `saveSessionID`, set `characterID` to the slot you want, and
   optionally set `availabilityFilter`. Send the request.

The equivalent call outside the portal:

```bash
curl "http://127.0.0.1:8788/api/v1/save-sessions/<saveSessionID>/characters/0/gestures?availabilityFilter=unlocked"
```

## Command-line verification

`GetGestures` is verified through its tests. From the repository root:

```bash
go test ./backend/saveengine -run '^TestGetGestures' -count=1 -v
go test ./backend/endpoints/world -run '^TestGetGestures' -count=1 -v
go test ./backend/swagger -run '^TestGesturesRoute' -count=1 -v
```

The tests build synthetic PC and PS4 containers inside `t.TempDir()`. They use no
real save file and no repository fixture, so they depend on nothing outside the
test process. The two platform fixtures place the anchor at different positions
and declare a different number of acquired projectiles, so a reader that depends
on a fixed position inside the slot, or that ignores the declared length in front
of the block, cannot pass both. The written block mixes canonical odd slot IDs, a
repeated one, an even value, an explicit zero, an unknown odd value and the
`0xFFFFFFFE` sentinel, so the sentinel rule, the exact-match rule, the "no even
normalisation" rule and the duplicate rule are all covered. The endpoint tests run
against the real stored catalog and cover the full 57-slot list from 56
documents, the two-slot resource, the category/name/`slotID` order including a
deliberately created tie, all three filter values, a rejected, padded and
case-shifted filter, an inactive slot, a `nil` engine and a `nil` catalog. The
Swagger tests compare the route body against a direct `world.GetGestures` call,
prove that `availabilityFilter` arrives unchanged, that an invalid filter answers
`400`, and that the route answers `404` without a SaveEngine.

## Current limitations

- It is a getter. Unlocking or locking a gesture is not possible: the session is
  read-only at this stage, and `SetGestureUnlocked` is a contract only.
- The equipped gesture ring (`EquippedGestures`) is not exposed by this endpoint.
- No event-flag state is read or reported, so a save whose gesture block and
  event flags disagree is reported from the gesture block alone.
- The specification divergence described above is not resolved here.
- The only transport is the local explorer route
  `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gestures`,
  and it exists only while the explorer runs without `-allow-external-bind`. No
  Wails binding, no CLI command and no frontend reaches the endpoint.
