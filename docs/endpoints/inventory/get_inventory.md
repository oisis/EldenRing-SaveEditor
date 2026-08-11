# GetInventory

## Overview

`GetInventory` returns one page of the raw, native `InventoryHeld` records stored
in one physical character slot of a save session that already exists in
SaveEngine. It reads the session's private snapshot only and reports every
non-empty record exactly as the save stores it.

**This is phase 1 of the Inventory surface: raw native inventory reading.** The
result deliberately contains nothing that a raw record cannot prove. It does
**not** return:

- item names or any other presentation data;
- GameCatalog identity — no `kind`, no `key`, no resource reference;
- `family`;
- variants, infusion, upgrade level or Ash of War;
- equipped state;
- a stable `OwnedItemID`;
- capacity;
- Storage (`inventory_storage_box`) records.

All of those require a verified `GaItem` parser and belong to a later phase. A
raw handle is not an item identity, so nothing here is resolved, named or
classified.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetInventory` never creates one, so
calling it before a successful `LoadSave` is an error, not an implicit load. The
endpoint opens no source file, returns no raw save byte, and modifies nothing:
neither the save, nor the session, nor any application state.

| | |
|---|---|
| EndpointID | `get_inventory` |
| Kind | Getter |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory` of the local explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. There is no Wails binding, no CLI command and no frontend. |
| Implementation source | [../../../backend/endpoints/inventory/get_inventory.go](../../../backend/endpoints/inventory/get_inventory.go) |
| Test source | [../../../backend/endpoints/inventory/get_inventory_test.go](../../../backend/endpoints/inventory/get_inventory_test.go) |
| Save access | read-only — the session's private in-memory snapshot; no file is opened |
| Mutation | none — the snapshot, the session, and the save file are left unchanged |

## Input

```go
func GetInventory(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	containerSection string,
	page int,
	pageSize int,
) (GetInventoryResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; the endpoint never creates one. A `nil` engine is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. It is passed to SaveEngine unchanged. |
| `characterID` | `int` | The physical slot index, `0` to `9`. It is the same index `GetSaveCharacters` reports positionally. |
| `containerSection` | `string` | Physical section filter: `""`, `"common"` or `"key"`. |
| `page` | `int` | One-based page number. `0` means page 1. |
| `pageSize` | `int` | Entries per page. `0` means 50. |

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

### `containerSection`

| Value | Meaning |
|---|---|
| `""` | Both sections: the common records first, then the key records. |
| `"common"` | The `0xA80` physical common records only. |
| `"key"` | The `0x180` physical key records only. |

Any other value is rejected. The comparison is exact and case-sensitive, the
value is never trimmed, and there is no alias: `"Common"`, `" key"`, `"keys"`
and `"storage"` are all errors, not the section they resemble.

### `page` and `pageSize`

They follow the same convention as
[`GetResources`](../catalog/get_resources.md):

- a negative `page` or `pageSize` is rejected;
- `page` `0` becomes `1`;
- `pageSize` `0` becomes `50`;
- there is deliberately no maximum `pageSize`;
- a valid page beyond `total` returns an empty, non-`nil` list with the real
  `total`.

## Output

```go
type GetInventoryResult = saveengine.CharacterInventory

type InventoryRecord struct {
	ContainerSection string `json:"containerSection"`
	PhysicalIndex    int    `json:"physicalIndex"`
	GaItemHandle     uint32 `json:"gaItemHandle"`
	Quantity         uint32 `json:"quantity"`
	AcquisitionIndex uint32 `json:"acquisitionIndex"`
}

type CharacterInventory struct {
	SaveSessionID string            `json:"saveSessionID"`
	CharacterID   int               `json:"characterID"`
	Active        bool              `json:"active"`
	Records       []InventoryRecord `json:"records"`
	Total         int               `json:"total"`
	Page          int               `json:"page"`
	PageSize      int               `json:"pageSize"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was read. It equals the requested value. |
| `characterID` | `int` | The requested slot index, `0` to `9`. It equals the requested value. |
| `active` | `bool` | `true` only when the slot's activity flag is exactly `1`. Any other flag value is not active. |
| `records` | `[]InventoryRecord` | The requested page, in physical native order. Empty, never `null`. |
| `total` | `int` | Number of non-empty records that passed the section filter, counted before paging. |
| `page` | `int` | The effective one-based page, so a requested `0` is reported as `1`. |
| `pageSize` | `int` | The effective page size, so a requested `0` is reported as `50`. |

| Record field | Type | Meaning |
|---|---|---|
| `containerSection` | `string` | The physical section the record was read from: `"common"` or `"key"`. |
| `physicalIndex` | `int` | The position of the record inside its own section, counted from `0`. |
| `gaItemHandle` | `uint32` | The raw stored `GaItem` handle. |
| `quantity` | `uint32` | The stored quantity with the high bit masked off. |
| `acquisitionIndex` | `uint32` | The raw stored acquisition index. |

### `physicalIndex` is a row position, not an identity

`containerSection` together with `physicalIndex` identifies the physical row the
record was read from, so the original row identity stays visible even after empty
rows have been filtered out. It is **not** a stable `OwnedItemID`: the game moves
a physical row when it rewrites the section, so the pair identifies a position in
the current save state and nothing more. Do not persist it as an item reference.

### Record order and layout

The order is the physical native order and is never sorted, grouped or
re-numbered:

1. the common section, in stored row order;
2. then the key section, in stored row order.

After the section filter, each remaining section keeps its own physical order.

The native layout inside the slot, measured from the confirmed slot anchor:

| Distance from the anchor | Content |
|---|---|
| `505` | first common record |
| `505 + 0xA80 × 12` | four-byte key-item count |
| `505 + 0xA80 × 12 + 4` | first key record |
| behind `0x180` key records | two trailing counters (`NextEquipIndex`, `NextAcquisitionSortId`) |

One physical record is 12 bytes: `gaItemHandle` `uint32`, `quantity` `uint32`,
`acquisitionIndex` `uint32`, all little-endian. The whole section, including the
key-item count header and both trailing counters, must fit inside the slot;
otherwise the read fails.

### Empty records

A record counts as absent when its handle is one of the two established native
sentinels:

| Handle | Meaning |
|---|---|
| `0x00000000` | empty slot |
| `0xFFFFFFFF` | invalid slot |

Absent records are skipped and never reported. **No other handle is
reinterpreted.** A handle this phase cannot explain stays visible exactly as
stored instead of being dropped, normalised or turned into a different value.

### Raw values only

- `gaItemHandle` and `acquisitionIndex` are returned exactly as stored. Neither
  is masked, normalised, validated or resolved.
- `quantity` is the stored value with the high bit masked off (`& 0x7FFFFFFF`),
  because that bit is not part of the count. That is the only transformation
  this endpoint performs.
- No name is created, no GameCatalog resource is resolved and no unknown value
  is hidden.

### Active, inactive and residual slots

An inactive slot — including a residual one, whose deleted character's inventory
is still present in the file — is a successful result, not an error:

- `active` is `false`;
- `total` is `0`;
- `records` is an empty, non-`nil` list;
- `page` and `pageSize` report the effective values;
- **the slot data is never searched or read.** The result comes from the
  UserData10 activity flag alone, so residual inventory is never located,
  decoded or exposed.

## PC and PS4

Both platforms are supported and read identically. The `InventoryHeld` layout
inside a character slot is the same on PC and PS4; the containers differ only in
where a slot begins:

| Platform | Slot data base |
|---|---|
| PC | slot block offset plus the `0x10`-byte MD5 prefix, which is skipped and never parsed |
| PS4 | the slot block offset itself; the PS4 container stores no MD5 prefix |

Those two bases live in
[`backend/saveengine/inventory_pc.go`](../../../backend/saveengine/inventory_pc.go)
and
[`backend/saveengine/inventory_ps4.go`](../../../backend/saveengine/inventory_ps4.go).
Everything inside the slot is decoded by the shared reader in
[`backend/saveengine/inventory.go`](../../../backend/saveengine/inventory.go),
which owns its own anchor and its own layout constants and borrows no position,
helper or parsing function from another getter.

## Processing flow

1. A `nil` engine is rejected by the endpoint. Everything else is delegated to
   SaveEngine.
2. SaveEngine validates `saveSessionID`, resolves the session, validates
   `characterID`, `containerSection`, `page` and `pageSize`, and normalises the
   two paging values.
3. The UserData10 activity flag of the requested slot is read. A flag other than
   `1` ends the read with an inactive result.
4. For an active slot the slot data bounds come from the platform entry point,
   and the confirmed 65-byte anchor is searched inside that one slot only.
5. The whole `InventoryHeld` section is read at the constant distance behind the
   anchor and decoded in place.
6. Non-empty records of the requested sections are collected in physical order,
   counted into `total` and cut into the requested page.

## Validation and errors

Every failure returns the zero result and changes nothing.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown or closed | `unknown save session "<id>"`. A closed or never-created session is never resolved to a different one. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. Checked only after the session resolves. |
| `containerSection` is not `""`, `"common"` or `"key"` | `containerSection must be "common", "key" or empty; got "<value>"`. |
| `page` is negative | `page must not be negative; got <value>`. |
| `pageSize` is negative | `pageSize must not be negative; got <value>`. |
| An active slot carries no anchor | `character <id> carries no inventory anchor`. |
| The section would reach past the end of the slot | `inventory of character <id> does not fit into its slot`. |
| A required range lies outside the snapshot | The read is rejected before it happens, and the error names the character slot involved. |

The last three rows are fail-closed by design: for an active slot the whole
`InventoryHeld` section must be present and complete where the game keeps it. A
missing anchor, a truncated section and any position reaching past the slot or
the snapshot all fail. There is no fallback offset, no second candidate position,
no partial result and no guessed value.

An inactive or residual slot is not in this table: it is a successful result. A
valid page beyond `total` is not in this table either: it is a successful,
empty page.

Stored values are never an error. No `gaItemHandle`, `quantity` or
`acquisitionIndex` is rejected for being unknown, out of range or implausible.

## Read-only behaviour

- The endpoint reads the session's private in-memory snapshot through the codec
  only. No file is opened, written, repaired, saved or reloaded.
- No snapshot byte leaves SaveEngine; the endpoint returns decoded values only.
- Nothing is normalised, repaired or resaved as a side effect of reading.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It reads no GameCatalog data. No value is looked up, named, or validated
  against the catalog.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is
  greenfield.

## Swagger route

```
GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory
    ?containerSection=&page=&pageSize=
```

The route is registered only by `registerSaveSessionRoutes`, which the explorer
calls only when it binds a loopback address. An explorer started with
`-allow-external-bind` does not register it and answers `404`.

`characterID` is parsed by the existing Swagger convention: decimal only, never
trimmed and never defaulted, so `"one"`, `" 0"` and `"0x1"` are rejected by the
route before SaveEngine sees them. `page` and `pageSize` use the existing paging
parser, so a non-integer value is rejected by the route and an absent value stays
`0`, which is the getter's "use the default". Everything else, including the
`containerSection` rule, is decided by the getter.

## Command-line verification

`GetInventory` is verified through its tests. From the repository root:

```bash
go test ./backend/saveengine -run '^TestGetInventory' -count=1 -v
go test ./backend/endpoints/inventory -run '^TestGetInventory' -count=1 -v
go test ./tools/swagger -run '^TestInventoryRoute$' -count=1 -v
```

The tests build synthetic PC and PS4 containers inside `t.TempDir()`. They use no
real save file and no repository fixture, so they depend on nothing outside the
test process. The two platform fixtures place the anchor at different positions,
so a reader that depends on a fixed position inside the slot cannot pass both.
The sections mix occupied rows with both native sentinels and leave gaps between
them, so `physicalIndex`, the sentinel rule, the section order and the quantity
high-bit mask are all covered. The tests also cover section filtering, paging
including a page beyond the total, a residual slot whose inventory survives a
cleared flag, an empty, unknown and closed `saveSessionID`, the rejected
`characterID` values `-1` and `10`, a rejected and a case-shifted
`containerSection`, negative paging values, a missing anchor, a truncated
section and a `nil` engine.

## Current limitations

- This is phase 1. The result is raw: no names, no GameCatalog identity, no
  `family`, no variants, no equipped state, no stable `OwnedItemID`, no capacity
  and no Storage records. Those need a separate, verified `GaItem` parser.
- The only transport is the local explorer route
  `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory`,
  and it exists only while the explorer runs without `-allow-external-bind`.
  No Wails binding, no CLI command and no frontend reaches the endpoint.
- It is a getter. Changing the inventory is not possible: the session is
  read-only at this stage.
