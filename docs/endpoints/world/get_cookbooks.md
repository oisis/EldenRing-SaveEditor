# GetCookbooks

## Overview

`GetCookbooks` returns every cookbook GameCatalog declares, each one marked with
whether one physical character slot of a save session that already exists in
SaveEngine has unlocked it. It joins exactly two sources and nothing else:

| Source | What it provides |
|---|---|
| GameCatalog | the cookbook definitions: resource `kind`, `key` and `family`, the `eventFlagID`, the `name` and the `category` of the single `item.unlocks` entry of kind `cookbook` each declaring resource carries |
| SaveEngine | the state of those event flags in the requested slot |

A cookbook is reported as unlocked when **its own event flag is set**. Nothing
else decides the state.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetCookbooks` never creates one, so
calling it before a successful `LoadSave` is an error, not an implicit load. The
endpoint opens no source file, returns no raw save byte, and modifies nothing:
neither the save, nor the session, nor GameCatalog, nor any application state. It
also calls no other endpoint.

| | |
|---|---|
| EndpointID | `get_cookbooks` |
| Kind | Getter |
| Domain | `world` |
| Supported resource types | `ItemDocument: Cookbook` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/cookbooks` of the local explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. There is no Wails binding, no CLI command and no frontend. |
| Implementation source | [../../../backend/endpoints/world/get_cookbooks.go](../../../backend/endpoints/world/get_cookbooks.go) |
| Test source | [../../../backend/endpoints/world/get_cookbooks_test.go](../../../backend/endpoints/world/get_cookbooks_test.go) |
| Save access | read-only — the session's private in-memory snapshot; no file is opened |
| Mutation | none — the snapshot, the session, GameCatalog and the save file are left unchanged |

## Input

```go
func GetCookbooks(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	availabilityFilter string,
) (GetCookbooksResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; the endpoint never creates one. A `nil` engine is rejected. |
| `gameCatalog` | `*gamecatalog.Catalog` | The catalog the cookbook definitions come from. A `nil` catalog is rejected. |
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
| `""` | Every cookbook the catalog declares. |
| `"unlocked"` | Only the cookbooks whose event flag is set. |
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
type CookbookEntry struct {
	Kind     schema.ResourceKind `json:"kind"`
	Key      string              `json:"key"`
	Name     string              `json:"name"`
	Category string              `json:"category"`
	Unlocked bool                `json:"unlocked"`
}

type GetCookbooksResult struct {
	SaveSessionID string          `json:"saveSessionID"`
	SaveRevision  string          `json:"saveRevision"`
	CharacterID   int             `json:"characterID"`
	Active        bool            `json:"active"`
	Cookbooks     []CookbookEntry `json:"cookbooks"`
}
```

```json
{
  "saveSessionID": "9f1c…",
  "characterID": 0,
  "active": true,
  "cookbooks": [
    {
      "kind": "item",
      "key": "400024B8",
      "name": "Ancient Dragon Apostle's Cookbook [1]",
      "category": "Ancient Dragon Apostle's Cookbook",
      "unlocked": false
    },
    {
      "kind": "item",
      "key": "40002454",
      "name": "Nomadic Warrior's Cookbook [1]",
      "category": "Nomadic Warrior's Cookbook",
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
| `cookbooks` | `[]CookbookEntry` | The cookbooks that passed `availabilityFilter`, in the deterministic order below. Empty, never `null`. |

| Entry field | Type | Meaning |
|---|---|---|
| `kind` | `string` | The GameCatalog resource kind. Always `item` for a cookbook. |
| `key` | `string` | The GameCatalog resource key of the declaring `ItemDocument`. |
| `name` | `string` | The cookbook name stored in the unlock entry. Never empty. |
| `category` | `string` | The cookbook series stored in the unlock entry. Never empty. |
| `unlocked` | `bool` | `true` when the event flag of this cookbook is set in the requested slot. |

Nothing else is returned. The `eventFlagID` behind the state is **internal**: it
is used for one bulk `SaveEngine.GetEventFlags` call and never appears in the
result. No provenance, no `sourceRecords`, no bitfield offset, no bit position,
no `ItemDocument`, no raw save byte and no field unrelated to this getter is
exposed.

### What makes a resource a cookbook

A cookbook is an `ItemDocument` that carries an entry in `item.unlocks` whose
`kind` is `cookbook`. The declaring resource is the source of the public `kind`
and `key` identity; that entry is the single source of the cookbook's `name`,
`category` and `eventFlagID`.

- The unlock `kind` is the **only selector**: no name pattern, icon or inventory
  category takes part in it. A resource without a cookbook unlock is not a
  cookbook and is skipped.
- A resource that does carry one must be an item of family `goods` and must
  carry **exactly one** cookbook unlock. Another family, an unknown family or a
  second cookbook unlock is a fail-closed error naming the resource key and the
  reason — never a dropped entry and never a silently preferred unlock.
- `acquisition.worldPickupFlagID` is **not** a second source of truth and is
  never read here.
- The inventory is **not** consulted. Owning the cookbook item without the flag
  being set is not an unlock and is never reported as one.

One declaring resource produces exactly one `CookbookEntry`, so the public
`kind` and `key` identity is unique across a result. In the current stored
catalog, 104 goods `ItemDocument`s declare exactly 104 cookbook unlocks, one
each, and no two of them share an `eventFlagID`.

### Order

The order is deterministic and does not depend on the unlock state:

1. `category`, ascending;
2. then `name`, ascending;
3. then `key`, ascending, as the tie-breaker between two cookbooks that share a
   category and a name.

Because one resource is one entry, the resource key separates every pair of
entries that agrees on the first two keys. The sort is stable, so catalog order
remains the last tie-breaker instead of the internal `eventFlagID`. Sorting is
never done by `unlocked`, and applying a filter never reorders what remains.

## Where the unlock state comes from

### One bulk read, one flag per cookbook

The endpoint collects the `eventFlagID` of every declared cookbook first and
hands the whole list to `SaveEngine.GetEventFlags` in a **single call**, so the
slot is located and its chain walked once. Each entry is then marked from its own
identifier.

An identifier SaveEngine cannot place is its error, never a `false`. There is no
second lookup path, no cache, no BST table and no fallback position.

### Event flag blocks 67 and 68

Every cookbook flag of the stored catalog lies in block `67` or block `68` — the
two blocks the event flag reader has confirmed evidence for. A flag is addressed
inside the bitfield as:

| Part | Rule |
|---|---|
| block | `eventFlagID / 1000`; `67` maps to block position 17, `68` to block position 18 |
| byte | `blockPosition × 125 + (eventFlagID mod 1000) / 8` counted from the first byte of the bitfield |
| bit | `7 − (eventFlagID mod 1000) mod 8` |

Any other block is rejected by the reader by name instead of being answered from
a guessed position.

## How the event flag bitfield is located

The bitfield has **no fixed position** inside a slot: four lengths the save itself
declares sit in front of it. It is located from the confirmed 65-byte slot anchor
and walked forwards:

| Distance | Content |
|---|---|
| anchor `+ 0x931D` | `uint32` acquired-projectile count |
| `+ 4 + count × 8` | end of the projectile records |
| `+ 0x9C + 0x0C + 0x12F` | equipped armaments, `EquipPhysicsData`, face data |
| `+ 0x6010` | end of the Storage Box |
| `+ 0x100` | end of `GestureGameData` |
| here | `uint32` unlocked-region count |
| `+ 4 + count × 4` | end of the region records |
| `+ 0x29 + 0x4C` | `RideGameData` with its control byte, blood stain with its padding |
| `+ 8 + declared size` | the menu profile: an eight-byte header whose `uint32` payload size is always read from the header |
| `+ 0x34` | `TrophyEquipData` |
| `+ 8 + 7000 × 16` | `GaItemGameData` |
| `+ 8 + declared size` | the tutorial data, again sized from its own header |
| `+ 29` | the confirmed scalar block — **the first byte of the bitfield** |

The bitfield itself is `0x1BF99F` bytes with a single trailing terminator byte
whose value is not interpreted. Both must lie inside the slot **and** inside the
snapshot before one flag is read.

Every declared length is widened to `int64` before it is multiplied or added, so
a corrupt value can never wrap into a small, seemingly valid offset. A declared
projectile count above `200000`, a region count above `20000` and a dynamic
payload size above `0x10000` are treated as corrupt instead of followed. The
legacy assumption that the tutorial payload is always `0x400` bytes long is
deliberately not used.

The event flag reader in
[`backend/saveengine/event_flags.go`](../../../backend/saveengine/event_flags.go)
owns its own anchor, its own layout constants, its own location algorithm and its
own bounds checks. It calls no other getter and borrows no private constant or
helper from one.

## Active, inactive and residual slots

An inactive slot — including a residual one, whose deleted character's bitfield is
still present in the file — is a successful result, not an error:

- `active` is `false`;
- every catalog cookbook is reported with `unlocked: false`, subject to
  `availabilityFilter`;
- **the slot data is never searched or read.** The result comes from the
  UserData10 activity flag alone, so a residual bitfield is never located,
  decoded or exposed.

With `availabilityFilter` `"unlocked"` an inactive slot therefore returns an
empty, non-`null` list.

## PC and PS4

Both platforms are supported and read identically. The chain from the anchor to
the bitfield is the same on PC and PS4; the containers differ only in where a
slot begins:

| Platform | Slot data base |
|---|---|
| PC | slot block offset plus the `0x10`-byte MD5 prefix, which is skipped and never parsed |
| PS4 | the slot block offset itself; the PS4 container stores no MD5 prefix |

Those two bases live in
[`backend/saveengine/event_flags_pc.go`](../../../backend/saveengine/event_flags_pc.go)
and
[`backend/saveengine/event_flags_ps4.go`](../../../backend/saveengine/event_flags_ps4.go)
and are covered by the SaveEngine tests on both platforms. The endpoint tests use
a PC container only, because the endpoint adds no platform-specific behaviour of
its own.

## Processing flow

1. A `nil` engine, a `nil` catalog and an unknown `availabilityFilter` are
   rejected by the endpoint.
2. Every catalog resource of `kind` `item` is read in full. A resource that
   carries no `item.unlocks` entry of kind `cookbook` is skipped. A resource that
   carries one must be an item of family `goods` and must carry exactly one such
   entry; it then becomes one `CookbookEntry` with that entry's internal
   `eventFlagID`. A wrong family, a second cookbook unlock and any missing or
   conflicting catalog data fail here.
3. The entries are sorted by category, name and key, and their identifiers are
   collected in that same order.
4. SaveEngine receives the whole list in one `GetEventFlags` call. It validates
   `saveSessionID`, resolves the session, validates `characterID` and resolves
   every identifier before the slot is touched.
5. The UserData10 activity flag of the requested slot is read. A flag other than
   `1` ends the save-side read with an inactive result and an empty flag map.
6. For an active slot the bitfield is located along the chain above and one bit
   per distinct identifier is read.
7. Each entry is marked unlocked from its own identifier and filtered by
   `availabilityFilter`.

## Validation and errors

Every failure returns the zero result and changes nothing.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `gameCatalog` is `nil` | `game catalog is not available` — likewise a wiring error. |
| `availabilityFilter` is not `""`, `"unlocked"` or `"locked"` | `availabilityFilter must be "unlocked", "locked" or empty; got "<value>"`. |
| A cookbook resource carries no key | `a cookbook resource carries no key`. |
| A cookbook resource has no known item family | `cookbook "<key>" has no known item family, want "goods"`. |
| A cookbook resource belongs to another family | `cookbook "<key>" has item family "<family>", want "goods"`. |
| A resource declares more than one cookbook unlock | `cookbook "<key>" declares <n> cookbook unlocks, want exactly one`. No unlock is preferred and none is dropped. |
| A cookbook unlock has no known `eventFlagID` | `cookbook "<key>" unlock <index> has no known event flag ID`. |
| A cookbook unlock has no known or an empty name | `cookbook "<key>" unlock <index> has no known name`. |
| A cookbook unlock has no known or an empty category | `cookbook "<key>" unlock <index> has no known category`. |
| Two cookbooks declare the same `eventFlagID` | `cookbooks "<first key>" and "<second key>" both declare event flag <id>`. Neither is dropped, renamed or preferred. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown or closed | `unknown save session "<id>"`. A closed or never-created session is never resolved to a different one. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. Checked only after the session resolves. |
| A cookbook flag lies outside the supported blocks | `event flag <id> lies in block <block>, which this reader does not support`. Rejected before the slot is touched. |
| An active slot carries no anchor | `character <id> carries no event flag anchor`. |
| A declared count or payload size is above the accepted maximum | `character <id> declares a <field> of <n>, want at most <max>`. |
| A declared length header lies outside the slot | `<field> of character <id> lies outside its slot`. |
| The bitfield would reach past the end of the slot or of the file | `event flags of character <id> do not fit into their slot` / `… into the save file`. |

The save-side rows are fail-closed by design: for an active slot the required
structure must be present and complete where the game keeps it. A missing anchor,
an implausible declared length, a truncated chain and any position reaching past
the slot or the snapshot all fail. There is no fallback offset, no second
candidate position, no partial result and no guessed value.

The catalog-side rows are fail-closed for the same reason: a missing name,
category or identifier produces an error, never an empty string, a default
category or an invented flag; a resource that carries a cookbook unlock but is
not a single-unlock goods item is reported by key and reason instead of being
skipped or split into several entries; and a conflict between two cookbooks is
reported instead of being resolved.

An inactive or residual slot is not in this table: it is a successful result.

## Read-only behaviour

- The endpoint reads the session's private in-memory snapshot through the codec
  only. No file is opened, written, repaired, saved or reloaded.
- No snapshot byte leaves SaveEngine; the endpoint returns decoded values only.
- GameCatalog is read through its existing public methods `ResourceSummaries` and
  `ResourceByKindAndKey`. No catalog index, cache, provider or method is added,
  and the catalog is never modified.
- Nothing is normalised, repaired or resaved as a side effect of reading.

## Dependencies

- The endpoint delegates to `backend/saveengine` and `backend/gamecatalog` and
  calls no other endpoint.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is greenfield.
  Earlier SaveForge versions (1.5.8 and 1.6.10) were used as research material to
  confirm the flag blocks and the cookbook list only; no legacy code is imported,
  reused or depended on at runtime.

> **Legacy divergence.** SaveForge 1.5.8 and 1.6.10 report a cookbook as unlocked
> when its event flag is set **or** when the matching cookbook item is present in
> the inventory. SaveForge 2.0 deliberately does not reproduce that fallback: the
> event flag is the single source of truth, so an owned but unregistered cookbook
> item is reported as locked. Both legacy versions agree with each other and with
> the 104-cookbook list used here.

## Swagger route

```
GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/cookbooks
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
calls `world.GetCookbooks` and nothing else, returns `200` on success and `400`
on any error, and logs neither save data, nor character names, nor the source
path.

### Calling it from the Scalar portal

1. Start the local explorer without `-allow-external-bind` and open the portal.
2. Open **API Reference → World → GetCookbooks**, or the endpoint page under
   **World → GetCookbooks**.
3. Create a session first with **Save sessions → LoadSave** (`POST
   /api/v1/save-sessions`) and copy the returned `saveSessionID`.
4. Paste it into `saveSessionID`, set `characterID` to the slot you want, and
   optionally set `availabilityFilter`. Send the request.

The equivalent call outside the portal:

```bash
curl "http://127.0.0.1:8788/api/v1/save-sessions/<saveSessionID>/characters/0/cookbooks?availabilityFilter=unlocked"
```

## Command-line verification

`GetCookbooks` is verified through its tests. From the repository root:

```bash
go test ./backend/saveengine -run '^TestGetEventFlags' -count=1 -v
go test ./backend/endpoints/world -run '^TestGetCookbooks' -count=1 -v
go test ./tools/swagger -run '^TestCookbooksRoute' -count=1 -v
```

The tests build synthetic PC containers inside `t.TempDir()`. They use no real
save file and no repository fixture, so they depend on nothing outside the test
process. The endpoint fixture walks the whole chain with a non-zero projectile
count, a non-zero region count and a tutorial payload that is deliberately not
`0x400` bytes long, and it sets three real cookbook flags spread over both
supported blocks while leaving neighbouring cookbook flags of both blocks at
zero, so a shifted byte or an inverted bit direction cannot pass. The endpoint
tests run against the real stored catalog and cover the full 104-cookbook list,
the exact flag-to-cookbook assignment, an active and a residual slot, all three
filter values, a rejected padded, aliased and case-shifted filter, the
category/name/key order including a deliberately created tie, a `nil` engine, a
`nil` catalog, an empty, unknown and closed session, a `characterID` outside the
slot range, a cookbook whose name or category is missing, a cookbook unlock
placed in a resource of family `weapon`, two cookbook unlocks in one resource,
two cookbooks claiming one `eventFlagID`, that the 104 entries carry 104 distinct
`kind`/`key` identities, and that the source file is left byte-identical. The
Swagger tests compare the route body against a direct `world.GetCookbooks` call,
prove that `availabilityFilter` arrives unchanged, that an invalid filter answers
`400`, and that the route answers `404` without a SaveEngine.

## Current limitations

- It is a getter. Unlocking or locking a cookbook is a separate operation:
  [`SetCookbookUnlocked`](set_cookbook_unlocked.md).
- Only event flag blocks `67` and `68` are supported for cookbooks. Every
  cookbook of the current stored catalog lies in them, but a future cookbook
  outside those blocks would be rejected instead of answered.
- The inventory is not consulted, so a save whose cookbook items and event flags
  disagree is reported from the event flags alone.
- The only transport is the local explorer route
  `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/cookbooks`,
  and it exists only while the explorer runs without `-allow-external-bind`. No
  Wails binding, no CLI command and no frontend reaches the endpoint.

## Snapshot identity

The result includes `saveRevision`, the opaque revision of the exact session
snapshot used by this read. Clients compare it exactly with the current session
revision and discard a mismatch; they never parse, trim or order it.
