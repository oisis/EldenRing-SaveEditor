# AddItemToInventory

## Overview

`AddItemToInventory` adds one catalog resource or variant to the **common
section** of `InventoryHeld` of one character of an existing SaveEngine session.
It either tops up the first common record the character already holds of that
item, or opens a new record in the first free common row. It moves no record,
merges none, reorders none, reindexes none and never writes the key section, the
Storage Box, Equipment or any other structure.

**`quantity` is a delta.** It is the amount added, never a target total. Setting
an existing record to an exact value is [`SetOwnedItemQuantity`](set_owned_item_quantity.md).

**The mutation touches the session's private in-memory snapshot only.** There is
no file write inside this endpoint, so the user's save on disk is left
byte-for-byte unchanged until a separate [`WriteSave`](../savesession/write_save.md)
succeeds. A committed change is reported by `SessionInfo.UnsavedChanges`, which
means exactly "the private snapshot of this session holds a committed change" and
says nothing about the disk.

**Every previously issued `ownedItemID` of the session is invalidated by a
successful call.** The commit increments `saveRevision`, and an identity is valid
only for the revision that minted it. This endpoint therefore returns **no**
`ownedItemID`: a token minted inside the mutation would be the only identity
alive under the new revision, which would leave the registry inconsistent with
every other record. The receipt reports the physical coordinates of the written
row instead; to address it, re-read the container under the new revision.

The endpoint owns exactly the decisions SaveEngine cannot make: which families
may be added at all, which ambiguous category must be rejected, the record mode
and the two limits. It opens no file, parses no save data of its own and calls no
other endpoint.

| | |
|---|---|
| EndpointID | `add_item_to_inventory` |
| Kind | Mutation |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport status | transport-exposed — `POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory/items` of the local explorer. The route exists only without `-allow-external-bind`; no Wails binding, CLI command or frontend reaches it. |
| Implementation source | [../../../backend/endpoints/inventory/add_item_to_inventory.go](../../../backend/endpoints/inventory/add_item_to_inventory.go) |
| Test source | [../../../backend/endpoints/inventory/add_item_to_inventory_test.go](../../../backend/endpoints/inventory/add_item_to_inventory_test.go) |
| Save access | read-write on the session's private in-memory snapshot; no file is opened |
| Mutation | one common record plus, for a new record, the common item count, `NextEquipIndex`, `NextAcquisitionSortId` and one active `GaItemData` entry — atomically, with rollback |

## Supported resources

Only two item families are accepted: **`goods`** and **`talisman`**.

Those are the two families for which this endpoint has a confirmed
`InventoryHeld` write contract. Their save-side `GaItem` handles are derived from
the game ID alone (`0xB0000000 | id` and `0xA0000000 | id`). Weapons, armour and
Ashes of War need records in the variable-length `GaItem` table, and allocating
one means repacking that section and shifting everything behind it. Spells and
spirit ashes share the goods `0x4` prefix, but they do not share this mutation's
confirmed inventory semantics. The family gate therefore rejects them even
though the prefix alone would produce a handle.

The family gate is the **primary** rule and is checked before anything else. A
game-ID prefix alone would not do: spells and spirit ashes carry the same
`0x4` prefix goods do.

Additionally, every resource whose `item.category` is **`key_items`** is
rejected. That catalog category does not distinguish common-section resources
from the confirmed subset routed to the key section: SaveForge 1.x sent only the
Crafting Kit, the cookbooks, the Cracked Pot and the Crimson Crystal Tear to key,
while maps, bell bearings and the rest stayed in common. This endpoint writes
common only and refuses the whole ambiguous category fail-closed. That costs
reach but needs no hardcoded ID, no schema field and no catalog regeneration;
writing a key-routed item into the wrong section would cost correctness.

## Input

```go
func AddItemToInventory(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	kind string,
	key string,
	variantID *uint32,
	quantity uint32,
	expectedRevision string,
) (AddItemToInventoryResult, error)
```

The local HTTP request places the two identities in the path and everything else
in a strict JSON body:

```http
POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory/items
Content-Type: application/json
```

```json
{
  "kind": "item",
  "key": "400006A4",
  "variantID": null,
  "quantity": 3,
  "expectedRevision": "1"
}
```

`kind`, `key`, `quantity` and `expectedRevision` are required, `variantID` is
optional, and unknown fields are rejected. The `Content-Type` has to be
`application/json`: this is a session-mutating `POST`, so a CORS simple request
is refused before the body is decoded. The transport parses only the typed
envelope: it does not trim or normalise strings, clamp the quantity, resolve the
resource, read GameCatalog fields or own a mutation rule.

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; the endpoint never creates one. A `nil` engine is rejected. |
| `gameCatalog` | `*gamecatalog.Catalog` | The already loaded catalog every decision below is read from. A `nil` catalog is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. It is passed to SaveEngine unchanged. |
| `characterID` | `int` | The physical slot index, `0` to `9`. The slot has to be active. |
| `kind` | `string` | Resource kind, matched exactly. |
| `key` | `string` | Resource key inside that kind, matched exactly. |
| `variantID` | `*uint32` | Optional variant game ID. `nil` selects the base ItemDocument. |
| `quantity` | `uint32` | The amount **added**. At least `1`. |
| `expectedRevision` | `string` | The `saveRevision` the caller believes the session is at. |

### `kind`, `key` and `variantID`

`gamecatalog.Catalog.ResourceByKindKeyAndVariant` is the single implementation of
the selection rule, shared with every other item endpoint: `nil` selects the base
document, and a value selects only the stored variant of that same `(kind, key)`
pair whose `ItemVariant.GameID` is known and exactly equal to it. No variant is
synthesised, and neither the base game ID nor an alias is a variant here.

### `quantity`

- It is a **delta**. The record ends up holding its previous value plus this one.
- It is never clamped. A value above a limit is rejected, not silently reduced to
  fit, and never spilled into a second record.
- `0` is an error. Removing a record is [`RemoveOwnedItem`](remove_owned_item.md).
- For a `separate_instances` resource it has to be exactly `1`, because every
  copy is its own physical record.
- The stored high bit (`0x80000000`) is not part of the count. It is preserved by
  SaveEngine exactly as the game left it: never set here, never cleared here. The
  count therefore occupies 31 bits and the accepted range is `1..2147483647`.

### `expectedRevision`

- It must be a canonical decimal `saveRevision` — no sign, no prefix, no padding,
  no separator, no whitespace — and `"0"` is a valid value.
- It is compared byte for byte against the session's current revision. A
  malformed value and a mismatched value are distinct errors, and the mismatch
  names the current revision so the caller can re-read without a second round
  trip. Neither changes a byte.

## Limits

The endpoint derives two limits from the resolved ItemDocument and hands them to
SaveEngine, which enforces them exactly as supplied:

| Limit | Value |
|---|---|
| `maxContainerTotal` | `item.storage.maxInventory` |
| `maxPerRecord` | `min(item.capabilities.stack.rules.maxPerStack, maxContainerTotal)` for `quantity_stack`, `1` for `separate_instances` |

`maxContainerTotal` bounds the sum of the item across the whole `InventoryHeld`
container — both of its sections — because the game counts what a character holds
there, not what one row holds. Records are summed by resolved game ID, so two
rows of one item cannot escape the sum by carrying different handles.

`maxStorage` is never read: this endpoint writes `InventoryHeld` only.

The endpoint accepts no mode, so it reads neither `safeModeMaxInventory` and
`safeModeMaxStorage` nor the `-sfv` fields. No limit is defaulted, invented,
widened or clamped. Unknown catalog data rejects the request instead.

## Output

```go
type AddItemToInventoryResult struct {
	SaveSessionID    string `json:"saveSessionID"`
	SaveRevision     string `json:"saveRevision"`
	CharacterID      int    `json:"characterID"`
	GameID           uint32 `json:"gameID"`
	Added            uint32 `json:"added"`
	Quantity         uint32 `json:"quantity"`
	CreatedRecord    bool   `json:"createdRecord"`
	ContainerSection string `json:"containerSection"`
	PhysicalIndex    int    `json:"physicalIndex"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the mutated session. It equals the requested value. |
| `saveRevision` | `string` | The revision the change committed under: the previous one plus exactly `1`. |
| `characterID` | `int` | The requested slot index. |
| `gameID` | `uint32` | The save-side game ID of the resolved resource or variant. |
| `added` | `uint32` | The delta that was applied. It equals the requested `quantity`. |
| `quantity` | `uint32` | The value the written record stores now. It equals `added` for a new record and is larger for a top-up. |
| `createdRecord` | `bool` | `true` when a new record was opened, `false` when an existing one was topped up. |
| `containerSection` | `string` | Always `"common"`. |
| `physicalIndex` | `int` | The physical row of the written record inside the common section. |

### There is deliberately no `ownedItemID`

The commit that produced `saveRevision` retired every identity of the previous
revision. A token minted for the new row inside that same commit would either be
minted under the revision that is being retired, or be the only identity alive
under the new one while every other record of the container has none — an
inconsistent registry either way. [`SetOwnedItemQuantity`](set_owned_item_quantity.md)
and [`RemoveOwnedItem`](remove_owned_item.md) echo a token that is stale for the
same reason; this endpoint receives no token to echo, so it reports the physical
coordinates instead. Re-read the container under the new revision to obtain the
identity of the row.

## What the mutation writes

For a **top-up** of an existing common record, exactly four bytes:

| Range | Value |
|---|---|
| The quantity field of that record | previous value plus `quantity`, with the high bit preserved |

Nothing else moves. The common item count, `NextEquipIndex`,
`NextAcquisitionSortId` and the whole `GaItemData` block stay byte-identical: the
section already holds the row, the counters are allocators of the game rather
than a population count, and the item already has its `GaItemData` entry.

For a **new record**:

| Range | Value |
|---|---|
| The twelve bytes of the first free common row | `{handle, quantity, acquisitionIndex}` |
| The four-byte common item count in front of the section | previous value plus `1` |
| `NextEquipIndex`, the first trailing counter | previous value plus `1` |
| `NextAcquisitionSortId`, the second trailing counter | `acquisitionIndex + 1` |
| One active `GaItemData` entry | `{gameID, 1}`, only when the character owns no physical record of that item yet |

`acquisitionIndex` is derived the way SaveForge 1.5.8 and 1.6.8 derived it, byte
for byte identically in both: `NextAcquisitionSortId` is a **high-water mark**,
not the index to assign. It is raised to the reserved-equipment floor `434` when
it is below it, stabilised to an even value, and the new record takes the odd
index one past it — which keeps consecutive adds in distinct game-side sort
buckets, because the game sorts by `index >> 1`.

The greatest safe stored mark is `0xFFFFFFFC`: it produces acquisition index
`0xFFFFFFFD` and leaves `0xFFFFFFFE` in the counter. Every greater stored value is
rejected before mutation because advancing it would wrap `uint32`.

The first free row is the lowest common row carrying one of the two native absent
sentinels, `0x00000000` or `0xFFFFFFFF`. A section without one is a rejection,
never an overwrite of an occupied row.

### `GaItemData`

`GaItemGameData` is a fixed `8 + 7000 × 16` block. Its length never changes, so
nothing behind it moves and no section is repacked. Its first four bytes are the
number of distinct active entries — stored signed, so anything at or above
`7000`, which includes every negative `int32`, is treated as corrupt instead of
followed. The four bytes behind the count are not interpreted and are never
written.

The active prefix is that many **eight-byte** `{itemID, 1}` records. Its capacity
is `7000` records, so the highest byte this mutation can reach is `8 + 7000 × 8`
— the **first half** of the block. The second half and the whole unknown tail
behind the active prefix stay byte for byte as the game left them.

Ash of War entries — `itemID >> 28 == 8` — form one contiguous segment at the end
of the active prefix. This mutation adds ordinary entries only, so that segment
is only ever shifted right as a whole, with its order and contents preserved; it
is never sorted and never entered.

Inside the ordinary entries only the **last ascending run** is treated as sorted.
An insert is placed by lower bound inside that run, so an unsorted legacy prefix
in front of it is left exactly as it is rather than "repaired".

An item ID the section already carries receives no second entry: the insert is a
no-op and the count does not move.

## Processing flow

1. A `nil` engine and a `nil` catalog are rejected by the endpoint itself.
2. `gameCatalog.ResourceByKindKeyAndVariant` resolves `(kind, key, variantID)` to
   one document.
3. `item.family` must be known and must be `goods` or `talisman`.
4. `item.gameID` must be known, and its prefix must agree with the family
   (`0x4` for goods, `0x2` for talisman). A disagreement is a hard error and is
   never silently corrected into one of the two facts.
5. `item.category` must be known and must not be `key_items`.
6. `item.storage.maxInventory` must be known and greater than zero.
7. `item.storage.recordMode` must be known. `quantity_stack` additionally
   requires a known, enabled `capabilities.stack` with `maxPerStack` greater than
   zero; `separate_instances` sets `maxPerRecord` to `1`.
8. `engine.AddItemToInventory` performs the mutation with `saveSessionID`,
   `characterID`, `quantity` and `expectedRevision` unchanged, the resolved game
   ID, the record mode and the two limits.
9. Under the lock, SaveEngine validates the whole plan before the first byte
   changes: the revision, the `characterID` range, the activity flag of the slot,
   the derived handle, the container total across both sections, the absence of a
   key record for that item in Inventory and Storage, whether Storage common
   already holds a physical copy, the free row, the count header, both allocators
   and the `GaItemData` position.
10. Every write of the plan is applied, then verified. A write or a verification
    that fails restores the exact previous bytes of everything the plan changed,
    in reverse order, and reports an error without advancing the revision,
    marking the session dirty or retiring an identity.

## Validation and errors

Every failure returns the zero result and changes nothing: no byte, no revision,
no `UnsavedChanges` flag and no identity.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `gameCatalog` is `nil` | `game catalog is not available` — a backend wiring error, not client input. |
| `kind`, `key` or `variantID` is unknown | The GameCatalog resolution error, identical to every other item endpoint. |
| The resolved resource carries no item document | `resource kind "<kind>" key "<key>" has no item document`. |
| `item.family` is unknown | `resource kind "<kind>" key "<key>" has an unknown family`. |
| `item.family` is not `goods` or `talisman` | `... is of family "<family>"; this endpoint adds only "goods" and "talisman"`. Spells and spirit ashes are rejected here despite sharing the goods game-ID prefix. |
| `item.gameID` is unknown | `resource kind "<kind>" key "<key>" has an unknown game ID`. |
| The family and the game-ID prefix disagree | `... declares family "<family>" and game ID 0x<gameID>, which disagree`. |
| `item.category` is unknown | `resource kind "<kind>" key "<key>" has an unknown category`. |
| `item.category` is `key_items` | `... is in category "key_items", which does not distinguish common from key routing; this common-only endpoint rejects the category fail-closed`. |
| `storage.maxInventory` is unknown or `0` | `resource kind "<kind>" key "<key>" carries no inventory limit`. |
| `storage.recordMode` is unknown | `resource kind "<kind>" key "<key>" has an unknown record mode`. |
| `capabilities.stack` is unknown, disabled, or carries no limit | `... has an unknown stack capability` / `... stores a quantity but does not stack` / `... carries no stack limit`. |
| `quantity` is `0` | `quantity must be at least 1; it is the amount added, not a target total`. |
| `quantity` exceeds `2147483647` | `quantity <n> exceeds the 2147483647 the record can store`. |
| `quantity` is not `1` for a `separate_instances` resource | `item 0x<gameID> stores every copy in its own record, so quantity must be 1; got <n>`. |
| `quantity` exceeds `maxPerRecord` | `quantity <n> exceeds the limit of <max> per record`. Nothing is clamped. |
| A top-up would exceed `maxPerRecord` | The error names the existing value, the result and the limit. Nothing is clamped and no second stack is opened for the remainder. |
| The add would raise the container total above `maxContainerTotal` | The error names the resulting total and the limit. Nothing is merged, deduplicated, moved or reindexed to make it fit. |
| `expectedRevision` is not a canonical decimal revision | `expectedRevision must be a canonical decimal saveRevision; got "<value>"`. |
| `expectedRevision` does not match the session | The error names the current `saveRevision`. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. |
| The slot is inactive or residual | `character <id> is not active and receives no item`. Its slot data is never located or written. |
| The derived handle would need a `GaItem` record | `game ID 0x<gameID> needs a record in the GaItem table, which this mutation never allocates`. |
| A record of the container carries an unresolvable handle | The resolution error. An add never proceeds past data the engine cannot decode. |
| The item already holds an Inventory or Storage key record | The error names the container, item and character and explains that the mutation writes common records only. |
| The Storage Box cannot be located or decoded | The Storage reader error. Storage is checked before both top-up and create, so malformed data never lets the mutation bypass key routing or first-copy `GaItemData` rules. |
| The common section holds no free record | `the common inventory section of character <id> holds no free record`. |
| The common item count already reads `0xA80` | `the inventory of character <id> declares 2688 of 2688 common records and receives no item`. |
| `GaItemData` declares `7000` or more active entries | `character <id> declares <n> active GaItemData entries, want fewer than 7000`. |
| An allocator cannot be advanced | `NextEquipIndex of character <id> cannot be advanced` / `NextAcquisitionSortId of character <id> is <n> and cannot be advanced`. |
| Any write of the plan cannot be verified | The previous bytes of everything the plan changed are restored in reverse order, the revision does not advance, and the error says the inventory is unchanged. |

## PC and PS4

Both platforms are supported and mutated identically. The record model, the
section layout, the counters and the `GaItemData` block are shared across
platforms; only the container around a slot differs, and that difference stays
owned by the platform entry points in `backend/saveengine`. Every position is
derived from the same helpers the readers of those sections use, so a reader and
this writer can never disagree about where a row lives.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint. In
  particular it never calls [`GetInventory`](get_inventory.md) or
  [`SetOwnedItemQuantity`](set_owned_item_quantity.md) as endpoints.
- It reads GameCatalog only for the fields listed under
  [Supported resources](#supported-resources) and [Limits](#limits). It returns no
  document, name or synthetic value.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 1.5.8 and 1.6.8
  are byte-identical in `addToInventory`, `addToKeyItems`, `classifyItemAdd` and
  `nextAcquisitionWriteIndex`; their entire difference sits in the `GaItem` table
  allocator, which this endpoint does not have and never reaches. The one
  deliberate API-level divergence is the meaning of `quantity`: legacy **set** the
  target total, 2.0 **adds** a delta. The byte contract is identical either way.
  No legacy code, helper, type or package structure was imported or reproduced.

## Command-line verification

From the repository root:

```bash
go test ./backend/endpoints/inventory -run '^TestAddItemToInventory' -count=1 -v
go test ./backend/endpoints/inventory -run '^TestAddItemToInventory' -race -count=1
go test ./backend/saveengine -run '^TestAddItemToInventory|^TestGaItemData' -count=1 -v
go test ./backend/saveengine -run '^TestAddItemToInventory|^TestGaItemData' -race -count=1
go test ./tools/swagger -run '^TestAddItemToInventoryRoute$' -count=1
```

The endpoint tests build the synthetic Inventory container of this package inside
`t.TempDir()` and rebuild the catalog with the addressed document rewritten, so
the catalog facts under test are stated rather than inherited. They cover a
committed top-up, both limit derivations, a real talisman document end to end, a
real key-routed document, a real spell and a real spirit ash behind the goods
prefix, an unknown category, record mode, inventory limit and stack capability, a
`nil` engine and a `nil` catalog, and an unknown kind, key and variant. Every
rejection additionally proves that the stored quantity, the record count, the
`saveRevision` and `UnsavedChanges` are unchanged.

Three endpoint guards are unreachable from a validated catalog and are therefore
not exercised: an unknown family, a family whose family-specific document is
missing, and a stack capability enabled without rules are all rejected by
GameCatalog schema validation itself. They stay in the endpoint as the fail-closed
default for a catalog assembled another way.

The SaveEngine tests own the rest of the matrix: both platforms, a new record and
a top-up, the first free row, the last free record and a full section, the
preserved high bit, the three counters, the `GaItemData` ordering rules including
the Ash of War segment and an unsorted legacy prefix, the idempotent insert, the
untouched second half of the block, an inactive slot, every rejection with proof
that no byte, revision or dirty flag moved, and a full parse → mutate → serialize
→ reload round trip that re-reads the record and re-resolves its handle.

## Current limitations

- **Local transport only.** The HTTP route is available only in the loopback
  explorer. It is absent under `-allow-external-bind`, and there is no Wails
  binding, CLI command or frontend.
- **Separate persistence.** The change remains in the session's private snapshot
  until `WriteSave` succeeds. Closing the session first discards it.
- **Goods and talismans only.** Weapons, armour, Ashes of War, spells, spirit
  ashes and gestures need a `GaItem` table record and are out of scope until this
  project has a confirmed, evidence-backed contract for allocating one.
- **Common section only.** Every `key_items` resource is refused, including the
  ones SaveForge 1.x kept in the common section.
- **Inventory only.** Adding to the Storage Box is a separate endpoint.
- **Storage is still validated.** Both add shapes read Storage to reject a matching
  key record and to decide whether a common record is the first physical copy;
  malformed Storage therefore rejects an Inventory add without mutation.
- It accepts no mode, so Safe Mode and Chaos Mode limits are out of scope here.
