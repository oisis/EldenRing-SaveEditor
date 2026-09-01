# RemoveOwnedItem

## Overview

`RemoveOwnedItem` removes one owned item instance of an existing SaveEngine
session, addressed by the opaque `ownedItemID` a previous
[`GetInventory`](get_inventory.md), [`GetStorage`](get_storage.md) or
[`GetOwnedItem`](get_owned_item.md) result reported. It removes exactly one
physical record — the one that identity was minted for — and creates none, moves
none, merges none, splits none and reorders none.

**The mutation touches the session's private in-memory snapshot only.** There is
no file write inside this endpoint, so the user's save on disk is left
byte-for-byte unchanged until a separate
[`WriteSave`](../savesession/write_save.md) succeeds. A committed change is
reported by `SessionInfo.UnsavedChanges`, which means exactly "the private
snapshot of this session holds a committed change" and says nothing about the
disk.

**Every previously issued `ownedItemID` of the session is invalidated by a
successful call.** The commit increments `saveRevision`, and an identity is valid
only for the revision that minted it. The `ownedItemID` this endpoint returns is
therefore already stale twice over: the revision moved on, and the record it
addressed no longer exists.

The endpoint owns no decision at all. Unlike
[`SetOwnedItemQuantity`](set_owned_item_quantity.md) a removal needs no stack
rule, no record mode and no container limit, so **no GameCatalog is read and
none is required**. The endpoint opens no file, parses no save data of its own
and calls no other endpoint.

| | |
|---|---|
| EndpointID | `remove_owned_item` |
| Kind | Mutation |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport status | transport-exposed — `DELETE /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}` of the local explorer. The route exists only without `-allow-external-bind`; no Wails binding, CLI command or frontend reaches it. |
| Implementation source | [../../../backend/endpoints/inventory/remove_owned_item.go](../../../backend/endpoints/inventory/remove_owned_item.go) |
| Test source | [../../../backend/endpoints/inventory/remove_owned_item_test.go](../../../backend/endpoints/inventory/remove_owned_item_test.go) |
| Save access | read-write on the session's private in-memory snapshot; no file is opened |
| Mutation | the twelve bytes of one physical record plus, where that section's count is confirmed to move, the non-empty count of the section, atomically, with rollback |

## Input

```go
func RemoveOwnedItem(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	ownedItemID string,
	expectedRevision string,
) (RemoveOwnedItemResult, error)
```

The local HTTP request places the three identities in the path and the one
mutation value in a strict JSON body:

```http
DELETE /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}
Content-Type: application/json
```

```json
{
  "expectedRevision": "1"
}
```

The body field is required and unknown fields are rejected. The path is the one
[`GetOwnedItem`](get_owned_item.md) reads from; the method is what separates
reading an instance from removing it. The transport parses only the typed
envelope: it does not trim or normalise strings, resolve the identifier or own a
mutation rule.

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; the endpoint never creates one. A `nil` engine is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. It is passed to SaveEngine unchanged. |
| `characterID` | `int` | The physical slot index, `0` to `9`. |
| `ownedItemID` | `string` | The opaque identity of the owned instance, exactly as a getter of this session reported it under the current revision. |
| `expectedRevision` | `string` | The `saveRevision` the caller believes the session is at. |

### `saveSessionID`, `ownedItemID` and `expectedRevision`

All three are passed through byte for byte. The endpoint never trims, normalises,
parses, splits or reconstructs them, and holds no rule of its own about their
shape. The full identity contract lives in
[`docs/owned-item-identity.md`](../../owned-item-identity.md).

### `ownedItemID` selects one instance, not an item

- It resolves through the session's identity registry only. There is **no lookup
  by game ID**, so several records of the same item are several instances and the
  identity picks exactly one of them.
- It is bound to the character and the physical container it was minted in.
  A Storage identity never removes the Inventory record at the same coordinates,
  and there is **no fallback into the other container**.
- An empty, unknown, fabricated, retired, foreign-session or other character's
  identity is rejected before anything is read or changed, with stale
  distinguishable from unknown.

### `expectedRevision`

- It must be a canonical decimal `saveRevision` — no sign, no prefix, no padding,
  no separator, no whitespace — and `"0"` is a valid value.
- It is compared byte for byte against the session's current revision. A
  malformed value and a mismatched value are distinct errors, and the mismatch
  names the current revision so the caller can re-read without a second round
  trip. Neither changes a byte.

## What a removal changes, and what it does not

### Each physical section is cleared its own confirmed way

The four sections do **not** share one write, because the native formats do not
agree on one. Only three of them are supported at all.

| Section | Cleared record | Section count |
|---|---|---|
| `InventoryHeld` common | handle `0`, quantity `0`, **third field = the physical row number** | lowered by exactly one |
| `InventoryHeld` key | handle `0`, quantity `0`, **third field = the physical row number** | **`key_count` is not touched** |
| Storage common | all twelve bytes zeroed, third field included | lowered by exactly one |
| Storage key | — | — rejected as unsupported, see below |

A count that already reads `0` is left at `0` rather than wrapped around to
`0xFFFFFFFF`: the save is then already inconsistent with its own content, and a
removal is not the place to repair or to reject it.

**Why the `InventoryHeld` third field keeps the row number.** SaveForge 1.5.8 and
1.6.10 both wrote `{handle 0, quantity 0, index = physical row}` when they cleared
an inventory row (`backend/core/writer.go`, `backend/core/transfer.go`), and
`backend/core/wondrous_physick_repair.go` did the same. Zeroing that field too
would be a new, unevidenced write.

**Why `key_count` stays.** `v1.6.10:backend/core/remove_key_item_test.go` is a
regression test that explicitly protects the `InventoryHeld` `key_count` header
across a key-item removal, and the removal path there lowered only the common
count. No native evidence in this project contradicts it, so 2.0 inherits the
confirmed behaviour instead of introducing a decrement.

**Why Storage key is rejected.** `v1.6.10:spec/10-storage.md` records that the
Storage key section is not exposed by the 1.x runtime model at all and that its
semantics are `needs verification`; 1.6.10 never wrote it. A synthetic fixture is
not evidence of a write contract, so the removal fails closed with an explicit
`not supported` error and changes nothing at all. It becomes supported only after
native evidence, not before.

### What is never touched

| Structure | Behaviour |
|---|---|
| Every other record | Untouched, including a second record of the same item, the record at the same coordinates in the other container and every neighbouring row. Nothing is merged, compacted, moved or reindexed to close the gap. |
| The GaItem table | Untouched. |
| `NextEquipIndex` and `NextAcquisitionSortId` | Untouched. |
| Equipment, Quick Items, the Pouch, the physick, equipped spells | **Never written.** The Equipment, Quick Item and Pouch reference pairs are *read* as a fail-closed guard, described below; nothing is unequipped, cleared or cascaded. |

**Why the GaItem table stays.** Deleting a GaItem record means repacking a
variable-length section and shifting everything behind it — the retired SaveForge
1.x rebuild path, which is not revived here. Leaving the table intact also means
no reference into it can dangle: a weapon that names an Ash of War handle, and a
handle a second record still uses, keep resolving exactly as before. The removal
only requires that the handle of the record it is about resolves at all, so
undecodable data is never silently deleted.

### A referenced instance is refused, never unequipped

**The physical `InventoryHeld` common row identifies the referenced instance —
the GaItem handle does not.**
[`docs/owned-item-identity.md`](../../owned-item-identity.md) records this twice:
fact **L1** states that one handle is legitimately shared by several physical
records, in one container or split across Inventory and Storage, and *Variant B —
`GaItemHandle` alone* is rejected for exactly that reason. A guard matching on
the handle alone would therefore refuse removals that reference nothing, and
would contradict the rule that a removal deletes exactly one instance. Legacy's
own reference check (`v1.6.10:backend/core/equipment_writer_test.go`,
`nativeHandleMatches`) resolves the row first and compares the handle **of that
row** — the same rule 2.0 applies.

Three structures carry such a reference, and all three are read, because all
three would otherwise be left pointing at a row this removal empties:

| Structure | Pairs | Position | Layout |
|---|---|---|---|
| Equipment | 22 | `EquipedItemIndex` at anchor + `0x00D0` + 1, `ActiveEquipedItemsGa` at anchor + `0x019C` + 1 | two parallel `uint32` blocks; the third block ends exactly on the four-byte common count of `InventoryHeld`, i.e. the same `anchor + 505` the container reader already measures — the two independent constants agree, which is what pins the layout down |
| Quick Items | 10 | `EquipItemData` at anchor + `0x9279` | one interleaved 8-byte pair per slot |
| Pouch | 6 | `EquipItemData` at anchor + `0x9279` + `0x50` + 4 | one interleaved 8-byte pair per slot |

Every pair is `{GaItem handle, 0x180 + physical InventoryHeld common row}`, which
is what `v1.6.10:backend/core/quick_pouch_writer.go` writes for the two
`EquipItemData` families and what `equipmentRepresentationOffsets` documents for
the equipment blocks. (`EquipedItemsID` at anchor + `0x0144` + 1 carries the bare
item ID and takes no part in this guard.) The three `EquipItemData` positions,
slot counts and record sizes are the ones
[`GetQuickItems`](../equipment/get_quick_items.md) and
[`GetPouchItems`](../equipment/get_pouch_items.md) already own; the guard borrows
them rather than restating them.

**The guard applies to `InventoryHeld` common records only.** Every row field
above is counted in that one section, so an `InventoryHeld` key record and both
Storage sections can never be named by one — sharing a handle with a referenced
common row does **not** make them look equipped, and for them the guard reads
nothing at all.

For an `InventoryHeld` common record, each of the 38 pairs decides as follows:

| Pair | Outcome |
|---|---|
| names the addressed row, carrying that row's own handle | **Rejected** — an exact reference to this instance. |
| names the addressed row, carrying a different handle | **Rejected fail-closed** — an inconsistent reference is refused, never repaired and never ignored. The message is distinct from the exact-reference one. |
| names another row, whatever its handle — including this row's handle | **Allowed.** A shared handle at a different row is a reference to a *different* instance. |
| carries `0xFFFFFFFF` or any row value below `0x180` | **Allowed** — it names no row at all. |

The guard reads those structures and writes none of them; no unequip and no
cascade is implemented, because inventing one would need evidence this project
does not have.

## Output

```go
type RemoveOwnedItemResult struct {
	MutationReceipt
	OwnedItemID   string `json:"ownedItemID"`
	CharacterID   int    `json:"characterID"`
	GameID        uint32 `json:"gameID"`
}

type MutationReceipt struct {
	OperationID   string   `json:"operationID"`
	OperationKind string   `json:"operationKind"`
	SaveSessionID string   `json:"saveSessionID"`
	SaveRevision  string   `json:"saveRevision"`
	ChangedScopes []string `json:"changedScopes"`
}
```

The result embeds the shared `MutationReceipt` anonymously, so the JSON stays
flat: `operationID`, `operationKind`, `saveSessionID`, `saveRevision` and
`changedScopes` are top-level members beside the domain fields, and there is no
nested `receipt` object.

The embedded receipt is exactly the one the central SaveEngine commit path
produced for this execution. Nothing here is reassembled from the EndpointID,
the session, the revision or a scope lookup.

- `operationID` names this one execution. It is opaque and unpredictable.
  Identifiers do not repeat among the receipts issued by one running SaveEngine
  instance. That guarantee does not currently cover application restarts:
  uniqueness across restarts requires a persistent operation journal and stays
  outside this stage. A rejected call returns the complete zero result and no
  `operationID` at all.
- `operationKind` is the stable kind of the mutation and is always exactly
  `remove_owned_item`.
- `changedScopes` are exactly `save.session`, `inventory`, `storage`,
  `diagnostics.report`, in that canonical order.

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the mutated session. It equals the requested value. |
| `saveRevision` | `string` | The revision the removal committed under: the previous one plus exactly `1`. |
| `ownedItemID` | `string` | The requested identity, echoed back unchanged. It is **already stale** — see below. |
| `characterID` | `int` | The requested slot index. It equals the requested value. |
| `gameID` | `uint32` | The save-side game ID the removed record's GaItem handle resolved to under the mutation lock, so the caller learns what was removed without addressing the record again. |

### The returned `ownedItemID` is stale by construction

The commit that produced `saveRevision` retired every identity of the previous
revision, including the one this call was made with, and the record it addressed
is gone. The returned value is an operation receipt; using it again is an error,
reported as a stale identity.

## Processing flow

1. A `nil` engine is rejected by the endpoint itself. No catalog is needed, so
   there is no second dependency to check.
2. `engine.RemoveOwnedItem` receives `saveSessionID`, `characterID`,
   `ownedItemID` and `expectedRevision` unchanged and performs everything else
   under one critical section of the process-wide engine lock:
   1. `expectedRevision` must be canonical and must equal the session's current
      revision.
   2. The identity is resolved to the one physical record it was minted for.
   3. The section that record lives in must be one with a confirmed write
      contract. A Storage key record is refused here, before anything is read.
   4. That record's container — and only that container — is read through its own
      reader, and the record must still sit at those coordinates with that exact
      identity.
   5. The record's GaItem handle must still resolve to a game ID.
   6. For an `InventoryHeld` common record only, the 22 Equipment, 10 Quick Item
      and 6 Pouch reference pairs of the slot are read, and a pair naming this
      record's physical row rejects the removal. No other section is checked,
      because no row field can name one.
   7. The record is cleared as its section prescribes, then the section's count
      is lowered where that count participates. Both writes are read back and
      verified; a failure restores **every byte the removal changed** — the
      record and, if it had already moved, the count — and confirms both before
      reporting that the record is unchanged.
   8. Only a fully successful mutation advances `saveRevision` by one, marks the
      session dirty and retires the identities of the previous revision.

## Validation and errors

Every failure returns the zero result and changes nothing: no byte, no revision,
no `UnsavedChanges` flag and no identity.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `saveSessionID` is empty, unknown or closed | The SaveEngine session error. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. |
| `ownedItemID` is empty | `ownedItemID is required`. |
| `ownedItemID` is unknown, fabricated or from another session | `unknown ownedItemID`. |
| `ownedItemID` was minted under an earlier revision | `stale ownedItemID`, deliberately distinct from unknown: the remedy is to re-read the container. |
| `ownedItemID` belongs to another character | `ownedItemID belongs to character <a>, not to character <b>`. |
| `expectedRevision` is not a canonical decimal revision | `expectedRevision must be a canonical decimal saveRevision; got "<value>"`. |
| `expectedRevision` does not match the session | The error names the current `saveRevision`, so the caller can re-read the container once. |
| The addressed record is gone | `ownedItemID "<id>" no longer addresses a record of character <n>`. |
| `ownedItemID` addresses a Storage **key** record | `... addresses a Storage key record, and removing one is not supported: this project has no confirmed native write contract for that section`. Nothing is read further and nothing is changed. |
| An Equipment, Quick Item or Pouch pair references the record's row with that row's handle | `ownedItemID "<id>" is referenced by <structure> <n> of character <c> and is not removed; unequip it first`. Nothing is unequipped and no byte changes. |
| Such a pair references the record's row with a *different* handle | `ownedItemID "<id>" sits in the inventory row <structure> <n> of character <c> references, and that reference carries the different handle 0x…; the removal fails closed rather than emptying a referenced row`. Deliberately distinct from the message above. |
| The record's handle resolves to no item | The GaItem error is named, and nothing is removed: undecodable data is never turned into a deletion. |
| A write cannot be verified | Every byte the removal had already changed — the record, and the section count if it had moved — is restored and confirmed, and the error says the record is unchanged. |
| The slot is inactive or residual | Its data is never read and mints no identity, so nothing in it can be addressed at all. |

## PC and PS4

Both platforms are supported and mutated identically. The record model is shared
across platforms; only the container around a slot differs, and that difference
stays owned by the platform entry points of the two container readers in
`backend/saveengine`. The record offset and the section count offset are derived
from the same helper the reader of that container uses, so a reader and a writer
can never disagree.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint. In
  particular it never calls [`GetOwnedItem`](get_owned_item.md),
  [`GetInventory`](get_inventory.md) or [`GetStorage`](get_storage.md) as
  endpoints.
- It reads no GameCatalog and holds no catalog rule.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. Earlier SaveForge versions
  (1.5.8 and 1.6.10) removed **by handle**, clearing every row that carried it in
  the selected containers, deleted the GaItem record once no list referenced the
  handle any more and rebuilt the whole slot around the shortened table. 2.0
  removes one instance, keeps the table and rebuilds nothing. No legacy code,
  helper or type was imported or reproduced.

## Command-line verification

From the repository root:

```bash
go test ./backend/endpoints/inventory -run '^TestRemoveOwnedItem' -count=1 -v
go test ./backend/endpoints/inventory -run '^TestRemoveOwnedItem' -race -count=1
go test ./backend/saveengine -run '^TestRemoveOwnedItem|^TestWriteOwnedItemRemoval' -count=1 -v
go test ./backend/saveengine -run '^TestRemoveOwnedItem' -race -count=1
go test ./tools/swagger -run '^TestRemoveOwnedItemRoute$' -count=1
```

The endpoint tests build the synthetic Inventory and Storage containers of this
package inside `t.TempDir()` and cover a committed removal in each container, the
survival of the other two records, the refusal of the unsupported Storage key
section, the missing-engine guard, the declared contract variables and the
pass-through of every rejection.

The SaveEngine tests own the rest of the matrix: both platforms and both
containers, the exact instance among two records of the same item, the untouched
twin record in the other container, the per-section clearing rule including the
preserved `InventoryHeld` physical row number, the untouched `key_count` header,
the common counts including the `0` boundary, the Storage key refusal on PC and
PS4, the reference guard over Equipment, Quick Items and the Pouch — the exact
pair, the inconsistent-handle pair, the unrelated-slot negatives including a
shared handle at another row, two common rows sharing one handle, and the
Inventory key and Storage records that a shared handle may not make look
equipped — the empty, unknown, fabricated, foreign-session, wrong-character,
stale and cross-container identities, the malformed and stale revision, the
unresolvable handle, the inactive slot, the rollback of a failed count write and
of a count that had already been lowered, the single revision increment, the
retired identity, the byte-identical source file with a separate `WriteSave`, and
concurrent access under `-race`.

## Current limitations

- **Local transport only.** The HTTP route is available only in the loopback
  explorer. It is absent under `-allow-external-bind`, and there is no Wails
  binding, CLI command or frontend.
- **Separate persistence.** The change remains in the session's private snapshot
  until `WriteSave` succeeds. Closing the session first discards it.
- The endpoint removes one instance. It does not add, move, merge, split or
  reorder a record, does not free its GaItem record and does not remove several
  records in one call.
- **Storage key records cannot be removed.** The section has no confirmed native
  write contract, so the request is refused rather than guessed. Lifting this
  needs native evidence of how the game writes that section.
- **A referenced instance cannot be removed.** An Equipment, Quick Item or Pouch
  pair that names the record's physical row blocks it; the caller has to unequip
  it first, and this endpoint never does it for them.
- It accepts no mode, so Safe Mode and Chaos Mode rules are out of scope here.
