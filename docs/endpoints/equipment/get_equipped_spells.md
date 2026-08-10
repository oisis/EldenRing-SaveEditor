# GetEquippedSpells

## Overview

`GetEquippedSpells` returns the fourteen physical `EquippedSpells` records of one
character slot of a save session that already exists in SaveEngine, resolves
every occupied record through GameCatalog, and reports how many Memory Slots the
loadout consumes and how many the character may fill.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetEquippedSpells` never creates one,
so calling it before a successful `LoadSave` is an error, not an implicit load.
The endpoint opens no source file, returns no raw save byte, and modifies
nothing: neither the save, nor the session, nor the catalog, nor any application
state.

| | |
|---|---|
| EndpointID | `get_equipped_spells` |
| Kind | Getter |
| Domain | `equipment` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-spells` of the local explorer (`backend/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. There is no Wails binding, no CLI command and no frontend. |
| Implementation source | [../../../backend/endpoints/equipment/get_equipped_spells.go](../../../backend/endpoints/equipment/get_equipped_spells.go) |
| Test source | [../../../backend/endpoints/equipment/get_equipped_spells_test.go](../../../backend/endpoints/equipment/get_equipped_spells_test.go) |
| Save access | read-only — the session's private in-memory snapshot; no file is opened |
| Catalog access | read-only — one `ItemByGameID` lookup per occupied record |
| Mutation | none — the snapshot, the session, the catalog, and the save file are left unchanged |

## Input

```go
func GetEquippedSpells(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
) (GetEquippedSpellsResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; the endpoint never creates one. A `nil` engine is rejected. |
| `gameCatalog` | `*gamecatalog.Catalog` | The catalog the occupied records are resolved against. A `nil` catalog is rejected. |
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
type EquippedSpellSlot struct {
	RawMagicParamID uint32 `json:"rawMagicParamID"`
	ResourceKey     string `json:"resourceKey"`
	Name            string `json:"name"`
	MemorySlots     int    `json:"memorySlots"`
}

type GetEquippedSpellsResult struct {
	SaveSessionID        string              `json:"saveSessionID"`
	CharacterID          int                 `json:"characterID"`
	Active               bool                `json:"active"`
	Spells               []EquippedSpellSlot `json:"spells"`
	UsedMemorySlots      int                 `json:"usedMemorySlots"`
	AvailableMemorySlots int                 `json:"availableMemorySlots"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was read. It equals the requested value. |
| `characterID` | `int` | The requested slot index, `0` to `9`. It equals the requested value. |
| `active` | `bool` | `true` only when the slot's activity flag is exactly `1`. Any other flag value is not active. |
| `spells` | array of exactly 14 entries | The fourteen physical records in stored order. |
| `usedMemorySlots` | `int` | Sum of the Memory Slots costs of the occupied records. |
| `availableMemorySlots` | `int` | Memory Slots the character may fill. |

Every entry of `spells` carries:

| Field | Type | Meaning |
|---|---|---|
| `rawMagicParamID` | `uint32` | The identifier the save stores, exactly as read, including the empty sentinel `0xFFFFFFFF`. |
| `resourceKey` | `string` | GameCatalog resource key of the resolved spell. Empty only for the native empty record. |
| `name` | `string` | GameCatalog presentation name of the resolved spell. Empty only for the native empty record. |
| `memorySlots` | `int` | GameCatalog Memory Slots cost of the resolved spell. Zero only for the native empty record. |

### The fourteen physical records

The save always keeps fourteen `EquippedSpells` records, and all fourteen are
returned in stored order. The count never varies with how many spells the
character has memorised or how many memory slots the character has unlocked:
records the game cannot use are still physical records and are still reported.

Each record is a pair of two little-endian `uint32`: the raw MagicParam ID and
the follower field behind it. The game writes exactly two combinations:

| Record | Raw MagicParam ID | Follower |
|---|---|---|
| empty | `0xFFFFFFFF` | `0x00000000` |
| occupied | the stored identifier | `0xFFFFFFFF` |

Both are accepted and the stored identifier is reported unchanged for either of
them. Any other pair is corrupt state: it is a fail-closed error and is never
reinterpreted as an empty record, as an occupied record, or as a repairable one.
The follower field itself is not part of the result, and the active-index
`uint32` behind the fourteen records is not read.

### Raw MagicParam IDs and the empty sentinel

`rawMagicParamID` is the value stored in the save, not a full item ID. It is
never normalised, masked, clamped or replaced, and an empty record keeps the
sentinel `0xFFFFFFFF` instead of becoming a zero or disappearing from the array.

A record holding the sentinel resolves nothing: its `resourceKey` and `name` stay
empty, its `memorySlots` stays zero, and no GameCatalog lookup is attempted for
it.

### GameCatalog resolution

An occupied record is resolved in one step, and only after its raw format has
been validated:

1. A raw MagicParam ID must be non-zero and must carry no item-family bits, so it
   must be below `0x10000000`. A stored value outside that range is not a raw ID
   and is rejected instead of being prefixed into some other item.
2. The full spell game ID is the raw ID with the spell item-family prefix:
   `0x40000000 | rawMagicParamID`. Glintstone Pebble, MagicParam row `4000`
   (`0x0FA0`), is therefore catalog game ID `0x40000FA0`.
3. `ItemByGameID` resolves that game ID to one `ItemDocument`. The document must
   declare the `spell` family, a known presentation name, and a known
   `spell.memorySlots` value.

`resourceKey`, `name` and `memorySlots` come from that document and from nowhere
else. If any of the three steps fails the whole call fails: no name, key or cost
is invented, no neighbouring item is substituted, and no partial result is
returned.

Duplicates are not an error. The same spell in two records is reported twice and
counted twice; enforcing a loadout rule belongs to a future setter, not to this
reader.

### Used and available Memory Slots

`usedMemorySlots` is the sum of the `memorySlots` values GameCatalog states for
the occupied records. It is computed from resolved costs only, so an unresolved
record fails the call instead of being counted as free.

`availableMemorySlots` is the capacity the character may fill, and it is smaller
than the fourteen physical records:

| Term | Value |
|---|---|
| base capacity | `2` |
| Memory Stones | the effective count the character holds, capped at `8` |
| Moon of Nokstella | `+2`, only while the talisman sits in an **unlocked** talisman field |
| game maximum | the sum is capped at `12` |

The effective Memory Stone count is the quantity of the Memory Stone stack in the
common inventory records, falling back to the key-item records only when no
common stack holds the stone. The stored quantity keeps a high bit that is not
part of the count, so it is masked off before the count is used. A character that
holds no Memory Stone at all is a normal result of zero stones, not an error.

The number of unlocked talisman fields is one more than the stored
additional-talisman-slots byte, which the game keeps in `0..3`. Only unlocked
fields are inspected: Moon of Nokstella in a field the character has not unlocked
grants nothing, exactly as in the game.

### JSON shape

```json
{
  "saveSessionID": "…",
  "characterID": 0,
  "active": true,
  "spells": [
    { "rawMagicParamID": 4000, "resourceKey": "40000FA0", "name": "Glintstone Pebble", "memorySlots": 1 },
    { "rawMagicParamID": 4294967295, "resourceKey": "", "name": "", "memorySlots": 0 }
  ],
  "usedMemorySlots": 1,
  "availableMemorySlots": 5
}
```

`spells` always holds exactly fourteen entries; the example is abbreviated. Every
identifier is a JSON number and is never encoded as hex, base64 or a string.

### What is not returned

The result contains only the fields above. It carries no character name, level,
statistics, appearance, equipment, inventory, quick items, pouch items, physick
mixture, gestures, active spell index, follower fields, offsets, and no raw
bytes. None of that is part of the result.

### Active, inactive and residual slots

An active slot — activity flag exactly `1` — reports `active: true`, the fourteen
records read from its own slot data, and both counts.

An inactive slot is a normal result, not an error. It reports `active: false`
with `saveSessionID` and `characterID` filled in, fourteen zero-value entries and
both counts zero.

This holds for a residual slot too, where the spells of a deleted character are
still present in the file: the activity flag alone decides what is reported. An
inactive slot's data is never searched and never read, so its residual spells are
neither located nor decoded nor resolved.

On any error the result is the zero value.

## Processing flow

1. The endpoint rejects a missing engine and a missing catalog.
2. Reading the save is delegated to SaveEngine, in this order: `saveSessionID` is
   validated, the session is looked up under the engine's own lock, and
   `characterID` is checked against the slot range.
3. SaveEngine reads the slot's activity flag through the codec's bounded,
   copying reads. An inactive slot returns immediately, without touching the
   slot data.
4. For an active slot only, SaveEngine locates the confirmed 65-byte anchor
   inside the data of that one slot. The anchor is stated locally in
   `backend/saveengine/equipped_spells.go`; this getter shares no state, no
   helper and no reader with any other endpoint.
5. The `EquippedSpells` section starts at the constant distance `0x9205` behind
   the anchor. That distance is the sum of the confirmed fixed structures between
   the two positions — `SpEffect` `0x00D0`, `EquipedItemIndex` `0x0058`,
   `ActiveEquipedItems` `0x001C`, `EquipedItemsID` `0x0058`,
   `ActiveEquipedItemsGa` `0x0058` and `InventoryHeld` `0x9011`. Nothing
   variable-length lies in front of it, so the distance is constant.
6. The fourteen records are read as `14 × 8` bytes and every pair is validated
   against the two native combinations.
7. The Memory Stone stack is read from the `InventoryHeld` section, whose common
   records start `505` bytes behind the anchor: `0xA80` records of twelve bytes
   (GaItem handle, quantity, acquisition index), a four-byte key count, then
   `0x180` key records of the same shape. The stack is the record whose handle is
   `0xB000272E`.
8. The additional-talisman-slots byte is read `241` bytes **in front of** the
   anchor, and the talisman fields are read from the equipped-armaments block:
   the acquired-projectiles count sits `0x931D` behind the anchor, the block
   starts at `projectileCountOffset + 4 + projectileCount * 8`, and its fields
   `17` to `21` are the five talisman fields. **The block cannot be read at a
   fixed offset:** the projectile section in front of it is variable length, so a
   fixed distance would land inside the projectile records of one save and behind
   the talismans of another.
9. The endpoint resolves every occupied record through GameCatalog and sums the
   resolved costs. The capacity SaveEngine computed is copied into the result
   unchanged.
10. The result is returned by value. The snapshot and the session model stay
    inside SaveEngine, and the catalog documents stay inside GameCatalog.

PC and PS4 differ in the base of the slot data only — the PC container puts an
MD5 prefix in front of every slot and the PS4 container does not. The layout
behind that base, including the whole spell and capacity chain, is identical, so
both platforms run the same search and the same reads.

The endpoint is thin by design: it contains no SaveEngine rule, it holds no
knowledge of the save format, it adds no catalog query method, it duplicates no
catalog state, and there is no shared endpoint helper behind it. It calls no
other endpoint — in particular neither `LoadSave`, `GetLoadedSave`, `CloseSave`,
`GetSaveCharacters`, `GetCharacterProfile`, `GetCharacterStats`,
`GetCharacterAppearance`, `GetEquipment`, `GetQuickItems`, `GetPouchItems`, nor
`GetPhysickMixture`.

## Validation and errors

Every failure returns the zero result and changes nothing.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `gameCatalog` is `nil` | `game catalog is not available` — likewise a wiring error. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown or closed | `unknown save session "<id>"`. A closed or never-created session is never resolved to a different one. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. Checked only after the session resolves. |
| An active slot carries no anchor | `character <id> carries no equipped-spells anchor`. |
| The fourteen records would reach past the end of the slot | `equipped spells of character <id> do not fit into its slot`. |
| A record pair is neither empty nor occupied | `spell record <n> of character <id> stores the pair (0x…, 0x…), which is neither empty nor occupied`. |
| The Memory Stone records would reach past the end of the slot | `memory stones of character <id> do not fit into its slot`. |
| The additional-talisman-slots byte would lie in front of the slot | `talisman slot count of character <id> lies outside its slot`. |
| The projectile count itself would lie past the end of the slot | `projectile count of character <id> lies outside its slot`. |
| The slot declares more than `200000` projectile records | `character <id> declares <n> projectile records, want at most 200000`. The count is widened to `int64` before it is multiplied, so a declared length can never wrap into a small, seemingly valid offset. |
| The talisman fields would reach past the end of the slot | `talisman fields of character <id> do not fit into its slot`. |
| An occupied identifier is zero or carries family bits | `spell slot <n>: 0x… is not a raw MagicParam ID`. |
| An occupied identifier resolves to no catalog item | `spell slot <n>: game ID 0x… is not a known item`. |
| An occupied identifier resolves to an item of another family | `spell slot <n>: game ID 0x… is not a spell`. |
| The resolved spell has no known name | `spell slot <n>: spell 0x… has no known name`. |
| The resolved spell has no known Memory Slots cost | `spell slot <n>: spell 0x… has no known memory slots`. |
| A required range lies outside the snapshot | The read is rejected before it happens, and the error names the character slot involved. |

These rows are fail-closed by design: for an active slot the records and the
capacity inputs must be present and complete where the game keeps them, and an
occupied record must resolve to a real spell with a real cost. A missing marker,
a pair the game never writes, an implausible declared length, a position reaching
past the slot or the snapshot, and an unresolvable identifier all fail. There is
no fallback offset, no second candidate position, no partial result and no
guessed value.

An inactive or residual slot is not in this table: it is a successful result.

Duplicate spells are not an error either. This is a reader, not the future
setter.

## Dependencies

- The endpoint delegates to `backend/saveengine` and reads `backend/gamecatalog`
  through `ItemByGameID` only. It calls no other endpoint.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is
  greenfield.

## Command-line verification

`GetEquippedSpells` is verified through its tests. From the repository root:

```bash
go test ./backend/saveengine -run '^TestGetEquippedSpells' -count=1 -v
go test ./backend/endpoints/equipment -run '^TestGetEquippedSpells' -count=1 -v
go test ./backend/swagger -run '^TestEquippedSpellsRoute' -count=1 -v
```

The tests build synthetic PC and PS4 containers inside `t.TempDir()`. They use no
real save file and no repository fixture, so they depend on nothing outside the
test process. The two platform fixtures place the anchor at different positions
and declare a different, non-zero projectile count, so a reader that depends on a
fixed position inside the slot cannot pass both. The SaveEngine tests cover all
fourteen records, both native pairs, a full and a partial loadout, duplicate
spells, a residual slot, the Memory Stone capacity from the common and from the
key records, the stone cap, the game maximum, Moon of Nokstella in an unlocked
and in a locked talisman field, both malformed pair directions, a missing anchor,
a section that no longer fits into the slot, an anchor without room for the byte
in front of it, an implausible projectile count, and that neither the source file
nor the result changes when the getter is called twice. The endpoint tests cover
successful catalog resolution against the stored catalog data, empty records, an
unknown raw identifier, a known identifier of another family, an identifier that
already carries family bits, an unknown Memory Slots cost, a `nil` engine, a
`nil` catalog, an invalid `characterID`, and a missing and an unknown session.

## Current limitations

- The only transport is the local explorer route
  `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-spells`,
  and it exists only while the explorer runs without `-allow-external-bind`.
  No Wails binding, no CLI command and no frontend reaches the endpoint.
- It reports the loadout only. The active spell index, the follower fields and
  the icon, effect, FP cost and requirement facts of a spell are not part of the
  result.
- It is a getter. Changing the loadout is not possible: the session is read-only
  at this stage.
