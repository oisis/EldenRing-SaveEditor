# GetInventory

## Overview

`GetInventory` returns one page of native `InventoryHeld` records stored in one
physical character slot of an existing SaveEngine session. Every listed record
is resolved through the private `GaItem` table and GameCatalog.

Each record carries the canonical ItemDocument `kind`/`key` pair and the exact
resolved `gameID`; the latter selects the materialised catalog variant for an
upgraded or infused item. `ownedItemID` remains the separate opaque identity of
the owned physical record. Native handles, acquisition indices, quantity masks,
physical order and paging remain visible unchanged.

This phase deliberately does not add a name, `family` filter, equipped state,
capacity or Storage records. A malformed GaItem table, an instance-backed handle
without a GaItem record, or a game ID absent from GameCatalog rejects the whole
request; no partial result or substitute item is returned.

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
    gameCatalog *gamecatalog.Catalog,
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
| `gameCatalog` | `*gamecatalog.Catalog` | The already loaded catalog used to resolve each record. A `nil` catalog is rejected. |
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
type InventoryRecord struct {
	OwnedItemID      string `json:"ownedItemID"`
	Kind             schema.ResourceKind `json:"kind"`
	Key              string `json:"key"`
	GameID           uint32 `json:"gameID"`
	ContainerSection string `json:"containerSection"`
	PhysicalIndex    int    `json:"physicalIndex"`
	GaItemHandle     uint32 `json:"gaItemHandle"`
	Quantity         uint32 `json:"quantity"`
	AcquisitionIndex uint32 `json:"acquisitionIndex"`
}

type GetInventoryResult struct {
	SaveSessionID string            `json:"saveSessionID"`
	SaveRevision  string            `json:"saveRevision"`
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
| `saveRevision` | `string` | The revision the result was read under, and the one every `ownedItemID` in it was minted under. A non-empty decimal string. |
| `characterID` | `int` | The requested slot index, `0` to `9`. It equals the requested value. |
| `active` | `bool` | `true` only when the slot's activity flag is exactly `1`. Any other flag value is not active. |
| `records` | `[]InventoryRecord` | The requested page, in physical native order. Empty, never `null`. |
| `total` | `int` | Number of non-empty records that passed the section filter, counted before paging. |
| `page` | `int` | The effective one-based page, so a requested `0` is reported as `1`. |
| `pageSize` | `int` | The effective page size, so a requested `0` is reported as `50`. |

| Record field | Type | Meaning |
|---|---|---|
| `ownedItemID` | `string` | Opaque identity of this physical record under `saveRevision`. Non-empty, unique inside one result. |
| `kind` | `string` | GameCatalog resource kind. Always `item`. |
| `key` | `string` | Canonical GameCatalog resource key. |
| `gameID` | `uint32` | Exact game ID resolved from the save; it selects a catalog variant when applicable. |
| `containerSection` | `string` | The physical section the record was read from: `"common"` or `"key"`. |
| `physicalIndex` | `int` | The position of the record inside its own section, counted from `0`. |
| `gaItemHandle` | `uint32` | The raw stored `GaItem` handle. |
| `quantity` | `uint32` | The stored quantity with the high bit masked off. |
| `acquisitionIndex` | `uint32` | The raw stored acquisition index. |

### `physicalIndex` is a row position, not an identity

`containerSection` together with `physicalIndex` identifies the physical row the
record was read from, so the original row identity stays visible even after empty
rows have been filtered out. It is **not** the record's identity: the game moves
a physical row when it rewrites the section, so the pair identifies a position in
the current save state and nothing more. Do not persist it as an item reference —
use `ownedItemID` to refer to a record.

### `ownedItemID` and `saveRevision`

`ownedItemID` is the opaque identity SaveEngine mints for one physical record,
and `saveRevision` is the revision every identifier in the result was minted
under. The full contract lives in
[`docs/owned-item-identity.md`](../../owned-item-identity.md); what a caller of
this endpoint has to know:

- **Opaque.** Compare it byte for byte. Never parse, split, trim, normalise or
  reconstruct it. It encodes no handle, no acquisition index, no `physicalIndex`
  and no slot address, and its internal shape may change without notice.
- **Session-scoped and revision-scoped.** It is valid only inside the session
  that produced it and only while `saveRevision` is unchanged. Closing the
  session, reloading the file or restarting the application invalidates it. It is
  never persisted in a template, a favorite or any other stored document.
- **Per physical record.** Two rows of the same item get two identifiers, and an
  Inventory record never shares its identifier with a Storage record at the same
  `containerSection`/`physicalIndex`.
- **Independent of the request.** `containerSection`, `page` and `pageSize`
  select *which* records come back; they never change *which* identifier a record
  gets. Within one revision, a filtered read, a paged read and a repeated read
  all report the same identifier for the same physical record.
- **Only for real records.** The two native absent sentinels are not records, so
  they are neither listed nor identified. A record whose handle this phase cannot
  explain is still listed and still identified.
- **`saveRevision` is a string.** It is the decimal rendering of an internal
  `uint64`, deliberately not a JSON number, because a value above 2^53−1 would
  round in a JavaScript client. Carry it back unchanged where a later mutation
  endpoint asks for `expectedRevision`; never parse, increment or order it.

An inactive or residual slot mints nothing but still reports the current
`saveRevision`.

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

### Catalog resolution and physical values

- `gaItemHandle` and `acquisitionIndex` are returned exactly as stored. Neither
  is masked, normalised, validated or resolved.
- `quantity` is the stored value with the high bit masked off (`& 0x7FFFFFFF`),
  because that bit is not part of the count. That is the only transformation
  this endpoint performs.
- `kind`, `key` and `gameID` come only from the resolved ItemDocument. A missing
  catalog item rejects the whole request; no name, key or variant is invented.
- `gaItemHandle` and `acquisitionIndex` remain raw physical values and are not
  accepted as a public item reference.

### Active, inactive and residual slots

An inactive slot — including a residual one, whose deleted character's inventory
is still present in the file — is a successful result, not an error:

- `active` is `false`;
- `total` is `0`;
- `records` is an empty, non-`nil` list;
- `page` and `pageSize` report the effective values;
- `saveRevision` reports the current revision, because an empty container is not
  a reason to omit it;
- **the slot data is never searched or read.** The result comes from the
  UserData10 activity flag alone, so residual inventory is never located,
  decoded, identified or exposed.

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
6. **Both** physical sections are decoded and every non-empty record of the
   container is identified, before any filter or page applies. That is what makes
   `ownedItemID` independent of `containerSection`, `page` and `pageSize`.
7. Records of the requested sections are then collected in physical order,
   counted into `total` and cut into the requested page.

## Validation and errors

Every failure returns the zero result and changes nothing.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `gameCatalog` is `nil` | `game catalog is not available` — a backend wiring error, not client input. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown or closed | `unknown save session "<id>"`. A closed or never-created session is never resolved to a different one. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. Checked only after the session resolves. |
| `containerSection` is not `""`, `"common"` or `"key"` | `containerSection must be "common", "key" or empty; got "<value>"`. |
| `page` is negative | `page must not be negative; got <value>`. |
| `pageSize` is negative | `pageSize must not be negative; got <value>`. |
| An active slot carries no anchor | `character <id> carries no inventory anchor`. |
| The section would reach past the end of the slot | `inventory of character <id> does not fit into its slot`. |
| The GaItem marker/table is missing, truncated or internally ambiguous | The request is rejected; no raw record is returned as a substitute. |
| A listed handle has no resolvable game ID or its game ID is absent from GameCatalog | The request is rejected; no partial page is returned. |
| A required range lies outside the snapshot | The read is rejected before it happens, and the error names the character slot involved. |

The last three rows are fail-closed by design: for an active slot the whole
`InventoryHeld` section must be present and complete where the game keeps it. A
missing anchor, a truncated section and any position reaching past the slot or
the snapshot all fail. There is no fallback offset, no second candidate position,
no partial result and no guessed value.

An inactive or residual slot is not in this table: it is a successful result. A
valid page beyond `total` is not in this table either: it is a successful,
empty page.

The physical fields themselves remain raw and are never normalised. Resolution
is different: an unresolvable handle or unknown catalog item rejects the whole
request.

## Read-only behaviour

- The endpoint reads the session's private in-memory snapshot through the codec
  only. No file is opened, written, repaired, saved or reloaded.
- No snapshot byte leaves SaveEngine; the endpoint returns decoded values only.
- Nothing is normalised, repaired or resaved as a side effect of reading.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It joins the raw records with GameCatalog and returns the canonical `kind`/`key`
  plus exact resolved `gameID`; no full document, name or synthetic value is returned.
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

The identity contract is covered as well: every returned record carries a
non-empty `ownedItemID`, two records of one read never share one, a repeated,
filtered or paged read of the same revision reports the same identifier for the
same physical record, a residual slot mints none, and the absent sentinels
receive none. The resolved-record coverage additionally exercises
handle-encoded goods, an armor GaItem and an Ash-of-War GaItem on both synthetic
platforms; an unresolved table, handle or catalog item fails closed. That the
two containers never share an identity is proved in
`backend/saveengine/owned_item_containers_test.go`, on the one fixture that
carries an `InventoryHeld` section and a Storage Box in the same slot.

## Current limitations

- The result is semantically resolved to ItemDocument identities, but does not
  add names, `family` filtering, equipped state, capacity or Storage data.
- `ownedItemID` does not survive a reload or a restart. It is minted per session
  and per revision, and there is deliberately no persistent instance key,
  because no native evidence for one has been established.
- The only transport is the local explorer route
  `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory`,
  and it exists only while the explorer runs without `-allow-external-bind`.
  No Wails binding, no CLI command and no frontend reaches the endpoint.
- It is a getter. Changing the inventory is not possible: the session is
  read-only at this stage.
