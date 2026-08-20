# GetOwnedItem

## Overview

`GetOwnedItem` returns one owned item instance of an existing SaveEngine
session, addressed by the opaque `ownedItemID` a previous
[`GetInventory`](get_inventory.md) or [`GetStorage`](get_storage.md) result
reported. The record is read through the private `GaItem` table and resolved
through GameCatalog, so the result carries the canonical ItemDocument
`kind`/`key` pair and the exact resolved `gameID`. A stored variant preserves an
affinity, while a confirmed weapon-upgrade range resolves the exact level
without changing that canonical identity.

`ownedItemID` is **not** a stable item reference. It is opaque, session-scoped
and revision-scoped: it is valid only inside the session that minted it and only
while that session's `saveRevision` is unchanged. It is compared byte for byte
and is never parsed, trimmed, normalised or reconstructed anywhere on the path
from the transport to SaveEngine.

Only the container the identifier was minted in is read. A Storage identifier
never resolves into the Inventory record at the same
`containerSection`/`physicalIndex`, and there is no fallback search of the other
container, no fallback position and no zero-value success.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetOwnedItem` never creates one, so
calling it before a successful `LoadSave` is an error, not an implicit load. The
endpoint opens no source file, returns no raw save byte, and modifies nothing:
neither the save, nor the session, nor any application state. It also calls no
other endpoint.

| | |
|---|---|
| EndpointID | `get_owned_item` |
| Kind | Getter |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}` of the local explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. There is no Wails binding, no CLI command and no frontend. |
| Implementation source | [../../../backend/endpoints/inventory/get_owned_item.go](../../../backend/endpoints/inventory/get_owned_item.go) |
| Test source | [../../../backend/endpoints/inventory/get_owned_item_test.go](../../../backend/endpoints/inventory/get_owned_item_test.go) |
| Save access | read-only — the session's private in-memory snapshot; no file is opened |
| Mutation | none — the snapshot, the session, and the save file are left unchanged |

## Input

```go
func GetOwnedItem(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	ownedItemID string,
) (GetOwnedItemResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; the endpoint never creates one. A `nil` engine is rejected. |
| `gameCatalog` | `*gamecatalog.Catalog` | The already loaded catalog used to resolve the record. A `nil` catalog is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. It is passed to SaveEngine unchanged. |
| `characterID` | `int` | The physical slot index, `0` to `9`. It is the same index `GetSaveCharacters` reports positionally. |
| `ownedItemID` | `string` | The opaque identity of the owned instance, exactly as a container getter of this session reported it. |

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
- The slot must be the one the identifier was minted for. An identifier of
  another character is rejected with its own error, because the remedy differs
  from "this identifier does not exist".

### `ownedItemID`

The full contract lives in
[`docs/owned-item-identity.md`](../../owned-item-identity.md); what a caller of
this endpoint has to know:

- **Opaque.** It is compared byte for byte and never parsed, split, trimmed,
  normalised or reconstructed. `" <id>"` and `"<id>0"` are different, unknown
  strings, not the identifier they resemble. It encodes no handle, no
  acquisition index, no `physicalIndex` and no slot address, and its internal
  shape may change without notice.
- **Session-scoped and revision-scoped.** It is valid only inside the session
  that produced it and only while `saveRevision` is unchanged. An identifier from
  another session is unknown; an identifier retired by a revision increment is
  reported as stale, which is a distinguishable error because the remedy is to
  re-read the container instead of giving up.
- **Never persisted.** It is never stored in a template, a favorite or any other
  document, because it does not survive a reload or a restart.
- **Empty is an error.** There is no default identifier and no "first record"
  fallback.

## Output

```go
type GetOwnedItemResult struct {
	SaveSessionID    string              `json:"saveSessionID"`
	SaveRevision     string              `json:"saveRevision"`
	OwnedItemID      string              `json:"ownedItemID"`
	CharacterID      int                 `json:"characterID"`
	Kind             schema.ResourceKind `json:"kind"`
	Key              string              `json:"key"`
	GameID           uint32              `json:"gameID"`
	Container        string              `json:"container"`
	ContainerSection string              `json:"containerSection"`
	PhysicalIndex    int                 `json:"physicalIndex"`
	GaItemHandle     uint32              `json:"gaItemHandle"`
	Quantity         uint32              `json:"quantity"`
	AcquisitionIndex uint32              `json:"acquisitionIndex"`
}
```

`GetOwnedItemResult` belongs to `GetOwnedItem` alone. It is not the Inventory or
Storage record model reused: the three getters share no result type.

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was read. It equals the requested value. |
| `saveRevision` | `string` | The revision the record was read under, and the one `ownedItemID` is valid for. A non-empty decimal string. |
| `ownedItemID` | `string` | The requested identity, echoed back unchanged. |
| `characterID` | `int` | The requested slot index, `0` to `9`. It equals the requested value. |
| `kind` | `string` | GameCatalog resource kind. Always `item`. |
| `key` | `string` | Canonical GameCatalog resource key. |
| `gameID` | `uint32` | Exact game ID resolved from the save; it selects a catalog variant when applicable. |
| `container` | `string` | The physical container the instance lives in: `"inventory"` or `"storage"`. |
| `containerSection` | `string` | The physical section the record was read from: `"common"` or `"key"`. |
| `physicalIndex` | `int` | The position of the record inside its own section, counted from `0`. |
| `gaItemHandle` | `uint32` | The raw stored `GaItem` handle. |
| `quantity` | `uint32` | The stored quantity with the high bit masked off. |
| `acquisitionIndex` | `uint32` | The raw stored acquisition index. |

### `container`, `containerSection` and `physicalIndex` are a position

The three fields describe where the instance currently sits, so a caller can show
the row it came from. They are **not** the record's identity: the game moves a
physical row when it rewrites a section, so the triple identifies a position in
the current save state and nothing more. Do not persist it as an item reference —
`ownedItemID` is the only reference this endpoint accepts.

`container` is nevertheless part of the *internal* identity: a record in
Inventory and a record at the same `containerSection`/`physicalIndex` in Storage
are two different records with two different identifiers, and this getter reads
only the one container its identifier was minted in.

### Catalog resolution and physical values

- `gaItemHandle` and `acquisitionIndex` are returned exactly as stored. Neither
  is masked, normalised, validated or resolved.
- `quantity` is the stored value with the high bit masked off (`& 0x7FFFFFFF`),
  because that bit is not part of the count. That is the only transformation
  this endpoint performs.
- `kind`, `key` and `gameID` come only from the resolved ItemDocument. An
  unresolvable handle or a game ID absent from GameCatalog rejects the whole
  request; no partial identity, placeholder document or substitute item is
  returned.

## PC and PS4

Both platforms are supported and read identically. The identity is derived from
the record model, which both containers already share across platforms; only the
container around a slot differs, and that difference is owned by the platform
entry points of the two readers in
[`backend/saveengine/inventory_pc.go`](../../../backend/saveengine/inventory_pc.go),
[`backend/saveengine/inventory_ps4.go`](../../../backend/saveengine/inventory_ps4.go),
[`backend/saveengine/storage_pc.go`](../../../backend/saveengine/storage_pc.go)
and
[`backend/saveengine/storage_ps4.go`](../../../backend/saveengine/storage_ps4.go).

## Processing flow

1. A `nil` engine and a `nil` catalog are rejected by the endpoint. Everything
   else is delegated.
2. SaveEngine validates `saveSessionID`, resolves the session and validates
   `characterID`.
3. `ownedItemID` is resolved through the session's private identity registry. An
   empty, unknown, retired or foreign identifier, and one belonging to another
   character, all fail here — before any save data is read.
4. Only the container the identifier was minted in is located and decoded, by the
   same reader `GetInventory` or `GetStorage` uses. There is one source of truth
   for the anchors, the bounds checks, the section sizes, the sentinels, the
   quantity mask, the physical indexes and the minting.
5. The record at the exact `containerSection`/`physicalIndex` of the identifier is
   selected, and its own minted identifier must equal the requested one. A missing
   or non-matching record is a hard error.
6. The record's one `GaItem` handle is resolved to a save-side game ID and that ID
   is resolved to one ItemDocument. A failure at either step rejects the request.

The whole SaveEngine step runs under the existing process-wide engine lock, so
the reported `saveRevision` is the revision the record was read under and the one
the identifier was validated against.

## Validation and errors

Every failure returns the zero result and changes nothing.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `gameCatalog` is `nil` | `game catalog is not available` — a backend wiring error, not client input. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown or closed | `unknown save session "<id>"`. A closed or never-created session is never resolved to a different one. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. Checked only after the session resolves. |
| `ownedItemID` is empty | `ownedItemID is required`. |
| `ownedItemID` is unknown, fabricated, padded or from another session | `unknown ownedItemID`. It never falls back to a physical lookup. |
| `ownedItemID` was retired by a revision increment | `stale ownedItemID`, deliberately distinguishable from unknown: re-read the container. |
| `ownedItemID` belongs to another character | `ownedItemID belongs to character <a>, not to character <b>`. |
| The physical record is gone or no longer carries that identifier | `ownedItemID "<id>" no longer addresses a record of character <id>`. |
| The container cannot be located or decoded | The reader's own fail-closed error, identical to the one `GetInventory` or `GetStorage` reports. |
| The `GaItem` marker/table is missing, truncated or internally ambiguous | The request is rejected; no raw record is returned as a substitute. |
| The handle has no resolvable game ID or its game ID is absent from GameCatalog | The request is rejected; no placeholder item is returned. |

There is no successful empty result: this getter either returns one fully
resolved instance or an error.

## Read-only behaviour

- The endpoint reads the session's private in-memory snapshot through the codec
  only. No file is opened, written, repaired, saved or reloaded.
- No snapshot byte leaves SaveEngine; the endpoint returns decoded values only.
- Resolving an identifier of one container does not materialise the identities of
  the other container of the same slot: materialisation stays lazy per container.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint. In
  particular it never calls [`GetInventory`](get_inventory.md) or
  [`GetStorage`](get_storage.md).
- It joins the record with GameCatalog and returns the canonical `kind`/`key`
  plus exact resolved `gameID`; no full document, name or synthetic value is
  returned.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is
  greenfield. Earlier SaveForge versions (1.5.8 and 1.6.10) carried no owned-item
  identity and no equivalent endpoint at all, so no legacy behaviour was
  reproduced, imported or depended on here.

## Swagger route

```
GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}
```

The route is registered only by `registerSaveSessionRoutes`, which the explorer
calls only when it binds a loopback address. An explorer started with
`-allow-external-bind` does not register it and answers `404`.

`characterID` is parsed by the existing Swagger convention: decimal only, never
trimmed and never defaulted, so `"one"`, `" 0"` and `"0x1"` are rejected by the
route before SaveEngine sees them. `ownedItemID` is handed over exactly as it
arrived: the transport owns no identity rule and never parses, trims, normalises
or reconstructs it. The route calls `inventory.GetOwnedItem` and nothing else,
and logs neither save data nor the source path.

### Calling it from the Scalar portal

1. Start the local explorer without `-allow-external-bind` and open the portal.
2. Create a session with **Save sessions → LoadSave** (`POST
   /api/v1/save-sessions`) and copy the returned `saveSessionID`.
3. Call **Inventory → GetInventory** or **Inventory → GetStorage** for the slot
   you want and copy one `ownedItemID` from the result.
4. Open **API Reference → Inventory → GetOwnedItem**, paste `saveSessionID`, the
   same `characterID` and the copied `ownedItemID`, and send the request.

The equivalent call outside the portal:

```bash
curl "http://127.0.0.1:8788/api/v1/save-sessions/<saveSessionID>/characters/0/owned-items/<ownedItemID>"
```

## Command-line verification

`GetOwnedItem` is verified through its tests. From the repository root:

```bash
go test ./backend/saveengine -run '^TestGetOwnedItem' -count=1 -v
go test ./backend/saveengine -run '^TestGetOwnedItem' -race -count=1
go test ./backend/endpoints/inventory -run '^TestGetOwnedItem' -count=1 -v
go test ./tools/swagger -run '^TestOwnedItemRoute$' -count=1 -v
```

The tests build synthetic PC and PS4 containers inside `t.TempDir()`. They use no
real save file and no repository fixture.

The SaveEngine tests run on the one fixture that carries an `InventoryHeld`
section and a Storage Box in the same slot, with the same handle, quantity and
acquisition index at the same coordinates in both. That is what proves the two
containers never cross-resolve: only the container itself can separate the two
results. They also cover both platforms, lazy per-container materialisation, an
identifier whose physical record does not exist, an empty, unknown, fabricated
and cross-session identifier, an identifier of another character, a stale
identifier after a controlled revision increment together with the fresh
identifier that replaces it, and concurrent `GetInventory`/`GetStorage`/
`GetOwnedItem` reads of one session under `-race`.

The endpoint tests resolve every record of the Inventory and the Storage fixture
on both platforms and compare the complete result, so `kind`, `key`, `gameID`,
`container`, `containerSection`, `physicalIndex`, the raw handle, the masked
quantity and the acquisition index are all exact. They additionally cover a
residual slot, which mints nothing and therefore resolves nothing, a padded and
an extended form of a valid identifier, a `nil` engine, a `nil` catalog and a
catalog that does not know the item.

The route test fetches each record of the fixture through the route by its own
identifier and compares the body with the endpoint result, so every path
parameter is proved to reach the getter.

## Current limitations

- `ownedItemID` does not survive a reload or a restart. It is minted per session
  and per revision, and there is deliberately no persistent instance key,
  because no native evidence for one has been established.
- The result carries no name, `family`, equipped state, capacity or relation to
  the other container.
- The only transport is the local explorer route
  `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}`,
  and it exists only while the explorer runs without `-allow-external-bind`.
  No Wails binding, no CLI command and no frontend reaches the endpoint.
- It is a getter. Changing the instance is not possible: the session is read-only
  at this stage, and the setters that will consume the same identity are separate
  later tasks.
