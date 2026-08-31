# GetPouchItems

## Overview

`GetPouchItems` returns the raw Pouch state stored in one physical character slot
of a save session that already exists in SaveEngine. It reads the session's
private snapshot only and reports the six `EquipItemData` pouch records exactly
as the save stores them.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetPouchItems` never creates one, so
calling it before a successful `LoadSave` is an error, not an implicit load. The
endpoint opens no source file, returns no raw save byte, and modifies nothing:
neither the save, nor the session, nor any application state.

| | |
|---|---|
| EndpointID | `get_pouch_items` |
| Kind | Getter |
| Domain | `equipment` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/pouch-items` of the local explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. There is no Wails binding, no CLI command and no frontend. |
| Implementation source | [../../../backend/endpoints/equipment/get_pouch_items.go](../../../backend/endpoints/equipment/get_pouch_items.go) |
| Test source | [../../../backend/endpoints/equipment/get_pouch_items_test.go](../../../backend/endpoints/equipment/get_pouch_items_test.go) |
| Save access | read-only — the session's private in-memory snapshot; no file is opened |
| Mutation | none — the snapshot, the session, and the save file are left unchanged |

## Input

```go
func GetPouchItems(engine *saveengine.Engine, saveSessionID string, characterID int) (GetPouchItemsResult, error)
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
type GetPouchItemsResult = saveengine.CharacterPouchItems

type PouchItemSlot struct {
	ItemID     uint32 `json:"itemID"`
	EquipIndex uint32 `json:"equipIndex"`
}

type CharacterPouchItems struct {
	SaveSessionID string           `json:"saveSessionID"`
	SaveRevision  string           `json:"saveRevision"`
	CharacterID   int              `json:"characterID"`
	Active        bool             `json:"active"`
	Items         [6]PouchItemSlot `json:"items"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was read. It equals the requested value. |
| `saveRevision` | `string` | Opaque revision of the exact session snapshot that produced this result. Clients compare it exactly with the current session revision and discard a mismatch. |
| `characterID` | `int` | The requested slot index, `0` to `9`. It equals the requested value. |
| `active` | `bool` | `true` only when the slot's activity flag is exactly `1`. Any other flag value is not active. |
| `items` | `[6]PouchItemSlot` | The six raw Pouch records of an active slot, in stored order. Always six zeroed records for an inactive slot. |

### Record order and layout

The Pouch records live in the `EquipItemData` section, behind the ten Quick Items
records and the four-byte active-quick value that follow the start of that
section. There are exactly six records and their order is the order the save
stores them in, which is the order of the six Pouch positions in the game. The
first record is position one; no record is skipped, reordered or filtered out,
and the count never varies.

Each record is two little-endian `uint32` fields, read in this order:

| Field | Meaning |
|---|---|
| `itemID` | The stored item field of the record. |
| `equipIndex` | The stored equip-index field of the record. |

The first record therefore starts at offset `0x54` from the start of
`EquipItemData`, that is `0x50` of Quick Items records plus the four-byte
active-quick value.

### Raw values only

Every `itemID` and every `equipIndex` is the value stored in the save, reported
exactly as read. The endpoint computes nothing:

- No value is normalised, masked, clamped, repaired, validated or rejected. A
  value of `0`, a value of `0xFFFFFFFF` and a value with its high bit set are all
  returned unchanged, including the empty-slot combination the game writes.
- Type bits are never stripped and no value is converted into a handle, an index
  or any other derived form.
- Nothing is resolved to a name. **No GameCatalog lookup happens at this stage:**
  no `ItemDocument` is resolved, no item is named, no slot restriction is
  applied, and no value is checked for being a known item.

### JSON shape

`items` is a fixed-size Go array, so it serialises as a JSON array of exactly six
objects, each with the two number fields `itemID` and `equipIndex`. It is never
encoded as hex, base64 or a string, and its length never varies.

### What is not returned

The result contains only the fields above. It carries no character name, level,
statistics, appearance, equipment, inventory, quick items, spells, physick,
gestures, offsets, and no raw bytes. None of that is read or computed to produce
the result. In particular the ten Quick Items records and the active-quick value
that precede the pouch records in the same section are not reported by this
getter.

### Active, inactive and residual slots

An active slot — activity flag exactly `1` — reports `active: true` and the six
records read from its own slot data.

An inactive slot is a normal result, not an error. It reports `active: false`
with `saveSessionID` and `characterID` filled in and six zeroed records.

This holds for a residual slot too, where the Pouch items of a deleted character
are still present in the file: the activity flag alone decides what is reported.
An inactive slot's data is never searched and never read, so its residual Pouch
items are neither located nor decoded nor analysed.

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
   `backend/saveengine/pouch_items.go`; this getter shares no state, no helper
   and no reader with any other endpoint.
5. The pouch records start at a constant distance behind the anchor. That
   distance is the sum of the confirmed fixed structures between the anchor and
   `EquipItemData` — `SpEffect` `0x00D0`, `EquipedItemIndex` `0x0058`,
   `ActiveEquipedItems` `0x001C`, `EquipedItemsID` `0x0058`,
   `ActiveEquipedItemsGa` `0x0058`, `InventoryHeld` `0x9011` and
   `EquippedSpells` `0x0074`, that is `0x9279` — plus the fixed head of that
   section, `0x50` of Quick Items records and the four-byte active-quick value,
   giving `0x92CD` in total. Every one of them has a fixed size, so nothing
   variable-length lies in front of the pouch records.
6. SaveEngine reads exactly six eight-byte records from there and returns them
   unchanged.
7. The result is returned by value. The snapshot and the session model stay
   inside the package.

PC and PS4 differ in the base of the slot data only — the PC container puts an
MD5 prefix in front of every slot and the PS4 container does not. The layout
behind that base, including the whole pouch chain, is identical, so both
platforms run the same search and the same reads.

The endpoint is thin by design: it contains no SaveEngine rule, it holds no
knowledge of the save format, and there is no shared endpoint helper behind it.
It calls no other endpoint — in particular neither `LoadSave`, `GetLoadedSave`,
`CloseSave`, `GetSaveCharacters`, `GetCharacterProfile`, `GetCharacterStats`,
`GetCharacterAppearance`, `GetEquipment`, nor `GetQuickItems`.

## Validation and errors

Every failure returns the zero result and changes nothing.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown or closed | `unknown save session "<id>"`. A closed or never-created session is never resolved to a different one. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. Checked only after the session resolves. |
| An active slot carries no anchor | `character <id> carries no pouch-items anchor`. |
| The required range would reach past the end of the slot | `pouch items of character <id> do not fit into its slot`. |
| A required range lies outside the snapshot | The read is rejected before it happens, and the error names the character slot involved. |

The last three rows are fail-closed by design: for an active slot the Pouch items
must be present and complete where the game keeps them. A missing marker and any
position reaching past the slot or the snapshot both fail. There is no fallback
offset, no second candidate position, no partial result and no guessed value.

An inactive or residual slot is not in this table: it is a successful result.

Stored values are never an error. No `itemID` or `equipIndex` is rejected for
being unknown, out of range or implausible.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It reads no GameCatalog data. No value is looked up, named, or validated
  against the catalog.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is
  greenfield.

## Command-line verification

`GetPouchItems` is verified through its tests. From the repository root:

```bash
go test ./backend/saveengine -run '^TestGetPouchItems' -count=1 -v
go test ./backend/endpoints/equipment -run '^TestGetPouchItems' -count=1 -v
```

The tests build synthetic PC and PS4 containers inside `t.TempDir()`. They use no
real save file and no repository fixture, so they depend on nothing outside the
test process. The two platform fixtures place the anchor at different positions,
so a reader that depends on a fixed position inside the slot cannot pass both.
The fixtures fill all six records with distinct pairs including `0`,
`0xFFFFFFFF` and values with the high bit set, and the assertion compares the
whole array. The tests also cover a residual slot whose Pouch items survive a
cleared flag, an empty and an unknown `saveSessionID`, the rejected `characterID`
values `-1` and `10`, a missing anchor, and a required range reaching past the
end of the slot.

## Current limitations

- The only transport is the local explorer route
  `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/pouch-items`,
  and it exists only while the explorer runs without `-allow-external-bind`.
  No Wails binding, no CLI command and no frontend reaches the endpoint.
- It reports raw state only. No value is resolved into an `ItemDocument`, no
  item name, icon or slot restriction is produced, and GameCatalog is not read.
- It is a getter. Changing the Pouch items is not possible: the session is
  read-only at this stage.
