# SetOwnedItemQuantity

## Overview

`SetOwnedItemQuantity` sets the stored quantity of one owned item instance of an
existing SaveEngine session, addressed by the opaque `ownedItemID` a previous
[`GetInventory`](get_inventory.md), [`GetStorage`](get_storage.md) or
[`GetOwnedItem`](get_owned_item.md) result reported. It creates no record,
removes none, merges none, moves none and reorders none: it changes exactly the
four quantity bytes of the one addressed record.

**The mutation touches the session's private in-memory snapshot only.** There is
no file write inside this endpoint, so the user's save on disk is left
byte-for-byte unchanged until a separate [`WriteSave`](../savesession/write_save.md)
succeeds. A committed change is reported by
`SessionInfo.UnsavedChanges`, which means exactly "the private snapshot of this
session holds a committed change" and says nothing about the disk.

**Every previously issued `ownedItemID` of the session is invalidated by a
successful call.** The commit increments `saveRevision`, and an identity is valid
only for the revision that minted it. The `ownedItemID` this endpoint returns is
therefore already stale: it identifies the operation that was performed, not a
record the caller may address again. To address the record once more, re-read the
container under the new revision and use the freshly minted identity.

The endpoint owns exactly one decision SaveEngine cannot make: the two limits the
mutation is validated against. It reads them from the record's own ItemDocument
and passes them down. It opens no file, parses no save data of its own and calls
no other endpoint.

| | |
|---|---|
| EndpointID | `set_owned_item_quantity` |
| Kind | Mutation |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/quantity` of the local explorer. The route exists only without `-allow-external-bind`; no Wails binding, CLI command or frontend reaches it. |
| Implementation source | [../../../backend/endpoints/inventory/set_owned_item_quantity.go](../../../backend/endpoints/inventory/set_owned_item_quantity.go) |
| Test source | [../../../backend/endpoints/inventory/set_owned_item_quantity_test.go](../../../backend/endpoints/inventory/set_owned_item_quantity_test.go) |
| Save access | read-write on the session's private in-memory snapshot; no file is opened |
| Mutation | the four quantity bytes of one physical record, atomically, with rollback |

## Input

```go
func SetOwnedItemQuantity(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	ownedItemID string,
	quantity uint32,
	expectedRevision string,
) (SetOwnedItemQuantityResult, error)
```

The local HTTP request places the three identities in the path and the two
mutation values in a strict JSON body:

```http
PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/quantity
Content-Type: application/json
```

```json
{
  "quantity": 10,
  "expectedRevision": "1"
}
```

Both body fields are required and unknown fields are rejected. The transport
parses only the typed envelope: it does not trim or normalise strings, clamp the
quantity, resolve the identifier, read GameCatalog fields or own a mutation
rule.

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; the endpoint never creates one. A `nil` engine is rejected. |
| `gameCatalog` | `*gamecatalog.Catalog` | The already loaded catalog the two limits are read from. A `nil` catalog is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. It is passed to SaveEngine unchanged. |
| `characterID` | `int` | The physical slot index, `0` to `9`. |
| `ownedItemID` | `string` | The opaque identity of the owned instance, exactly as a getter of this session reported it under the current revision. |
| `quantity` | `uint32` | The new stored quantity. At least `1`. |
| `expectedRevision` | `string` | The `saveRevision` the caller believes the session is at. |

### `saveSessionID`, `ownedItemID` and `expectedRevision`

All three are passed through byte for byte. The endpoint never trims, normalises,
parses, splits or reconstructs them, and holds no rule of its own about their
shape. The full identity contract lives in
[`docs/owned-item-identity.md`](../../owned-item-identity.md).

### `quantity`

- It is never clamped. A value above a limit is rejected, not silently reduced to
  fit.
- `0` is an error, not a removal: this endpoint never removes a record.
- The stored high bit (`0x80000000`) is not part of the count. It is preserved by
  SaveEngine exactly as the game left it: never set here, never cleared here.
- Because that highest bit does not belong to the number, the count occupies the
  remaining 31 bits and the accepted range is `1..2147483647`, not the full
  `uint32` range. A larger value is rejected by SaveEngine before any byte
  changes, whatever the item and container limits are.

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
| `maxContainerTotal` | `item.storage.maxInventory` for a record in Inventory, `item.storage.maxStorage` for a record in Storage. |
| `maxPerRecord` | `min(item.capabilities.stack.rules.maxPerStack, maxContainerTotal)` |

`maxContainerTotal` bounds the sum of the addressed item across the whole
physical container — both of its sections — because the game counts what a
character holds there, not what one row holds. Records are summed by resolved
game ID, so two rows of one item cannot escape the sum by carrying different
handles.

**The rule is deliberately fail-closed.** In Storage a single record still never
exceeds `maxPerStack`, even when `maxStorage` is larger; the per-stack limit is
what one physical row is known to hold, and nothing is merged, split or spilled
into a second row to satisfy a larger request.

The endpoint accepts no mode, so it reads neither `safeModeMaxInventory` and
`safeModeMaxStorage` nor the `-sfv` fields. No limit is defaulted, invented,
widened or clamped. Unknown catalog data rejects the request instead.

## Output

```go
type SetOwnedItemQuantityResult struct {
	MutationReceipt
	OwnedItemID   string `json:"ownedItemID"`
	CharacterID   int    `json:"characterID"`
	Quantity      uint32 `json:"quantity"`
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
  `set_owned_item_quantity`.
- `changedScopes` are exactly `save.session`, `inventory`, `storage`,
  `equipment.loadout`, `diagnostics.report`, in that canonical order.

`equipment.loadout` is part of the list because a quantity change can address
a record a Quick Item or Pouch slot reports.

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the mutated session. It equals the requested value. |
| `saveRevision` | `string` | The revision the change committed under: the previous one plus exactly `1`. |
| `ownedItemID` | `string` | The requested identity, echoed back unchanged. It is **already stale** — see below. |
| `characterID` | `int` | The requested slot index. It equals the requested value. |
| `quantity` | `uint32` | The value now stored in the record. It equals the requested `quantity`. |

### The returned `ownedItemID` is stale by construction

The commit that produced `saveRevision` retired every identity of the previous
revision, including the one this call was made with. The returned value is the
identifier the operation was performed with — an operation receipt — and using it
again is an error, reported as a stale identity. Re-read the container to obtain
the identity the record carries under the new revision.

## Processing flow

1. A `nil` engine and a `nil` catalog are rejected by the endpoint itself.
2. `engine.GetOwnedItem` reads the one physical record the identity was minted
   for. The session, `characterID` and the identity are validated there.
3. `engine.ResolveGaItemIDs` resolves that record's one `GaItem` handle to a
   save-side game ID.
4. `gameCatalog.ItemByGameID` resolves that game ID to one ItemDocument. An
   unknown game ID rejects the request; no placeholder document is used.
5. `capabilities.stack` must be known and enabled and must carry rules with
   `maxPerStack` greater than zero.
6. `storage.recordMode` must be known and equal to `quantity_stack`. An unknown
   mode and `separate_instances` are both rejected.
7. The limit of the record's **own** container must be known and greater than
   zero: `maxInventory` for an Inventory record, `maxStorage` for a Storage
   record. An unknown container is an error; the other container's limit is never
   substituted.
8. `maxPerRecord` is computed as the minimum of `maxPerStack` and that limit.
9. `engine.SetOwnedItemQuantity` performs the mutation with `saveSessionID`,
   `characterID`, `ownedItemID`, `quantity` and `expectedRevision` unchanged, the
   resolved game ID as an anti-TOCTOU guard, and the two limits.
10. The SaveEngine result is copied into the endpoint result.

Steps 2–4 run outside the mutation lock, so the resolved game ID is re-checked
under the lock: SaveEngine rejects the request unless the addressed record still
denotes exactly that item. The limits therefore always belong to the item they
were read for.

The mutation itself is atomic. Every fallible check completes before the first
byte changes; the four-byte write is verified, and a failed verification restores
the exact previous bytes and reports an error without advancing the revision,
marking the session dirty or retiring an identity.

## Validation and errors

Every failure returns the zero result and changes nothing: no byte, no revision,
no `UnsavedChanges` flag and no identity.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `gameCatalog` is `nil` | `game catalog is not available` — a backend wiring error, not client input. |
| `saveSessionID` is empty, unknown or closed | The SaveEngine error of the read step, identical to `GetOwnedItem`. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. |
| `ownedItemID` is empty, unknown, retired, foreign or from another character | The SaveEngine identity error, with stale distinguishable from unknown. |
| The handle has no resolvable game ID, or the game ID is absent from GameCatalog | `owned item "<id>": game ID 0x<gameID> is not a known item`. |
| `capabilities.stack.known` is `false` | `owned item "<id>": item 0x<gameID> has an unknown stack capability`. |
| `capabilities.stack.enabled` is `false` | `owned item "<id>": item 0x<gameID> does not stack`. |
| The stack rules are missing or `maxPerStack` is `0` | `owned item "<id>": item 0x<gameID> carries no stack limit`. |
| `storage.recordMode` is unknown or `separate_instances` | `owned item "<id>": item 0x<gameID> does not store a quantity in one record`. |
| `storage.maxInventory` is unknown or `0`, for an Inventory record | `owned item "<id>": item 0x<gameID> carries no inventory limit`. |
| `storage.maxStorage` is unknown or `0`, for a Storage record | `owned item "<id>": item 0x<gameID> carries no storage limit`. |
| The record lives in an unknown container | `owned item "<id>" lives in unknown container "<container>"`. |
| `quantity` is `0` | `quantity must be at least 1; removing a record is a separate operation`. |
| `quantity` exceeds `2147483647` | `quantity <n> exceeds the 2147483647 the record can store`. The highest bit of the raw field is a preserved flag, so it is not available to the count. |
| `quantity` exceeds `maxPerRecord` | `quantity <n> exceeds the limit of <max> per record`. Nothing is clamped. |
| `quantity` would raise the container total above `maxContainerTotal` | The request is rejected with the resulting total and the limit named. Nothing is merged, deduplicated, moved or reindexed to make it fit. |
| `expectedRevision` is not a canonical decimal revision | `expectedRevision must be a canonical decimal saveRevision; got "<value>"`. |
| `expectedRevision` does not match the session | The error names the current `saveRevision`, so the caller can re-read the container once. |
| The addressed record is gone, or the record now denotes a different item | The request is rejected before any byte changes. |
| The four-byte write cannot be verified | The previous bytes are restored, the revision does not advance, and the error says the record is unchanged. |

## PC and PS4

Both platforms are supported and mutated identically. The record model is shared
across platforms; only the container around a slot differs, and that difference
stays owned by the platform entry points of the two container readers in
`backend/saveengine`. The quantity offset is derived from the same helper the
reader of that container uses, so the two can never disagree.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint. In
  particular it never calls [`GetOwnedItem`](get_owned_item.md),
  [`GetInventory`](get_inventory.md) or [`GetStorage`](get_storage.md) as
  endpoints; it uses the SaveEngine methods directly.
- It reads GameCatalog only for the fields listed under
  [Limits](#limits) and `recordMode`. It returns no document, name or synthetic
  value.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. Earlier SaveForge versions
  (1.5.8 and 1.6.10) capped a quantity by the container limit and **clamped** the
  requested value to fit; 2.0 rejects instead of clamping, and additionally bounds
  a single record by `maxPerStack`. No legacy code, helper or type was imported
  or reproduced.

## Command-line verification

From the repository root:

```bash
go test ./backend/endpoints/inventory -run '^TestSetOwnedItemQuantity' -count=1 -v
go test ./backend/endpoints/inventory -run '^TestSetOwnedItemQuantity' -race -count=1
go test ./backend/saveengine -run '^TestSetOwnedItemQuantity' -count=1 -v
go test ./backend/saveengine -run '^TestSetOwnedItemQuantity' -race -count=1
go test ./tools/swagger -run '^TestSetOwnedItemQuantityRoute$' -count=1
```

The endpoint tests build the synthetic Inventory and Storage containers of this
package inside `t.TempDir()` and rebuild the catalog with the addressed document
rewritten, so the catalog facts under test are stated rather than inherited. They
cover a committed mutation in each container, the selection of
`min(maxPerStack, maxStorage)` when `maxStorage` is larger, an unknown and a
disabled stack capability, `separate_instances` and an unknown `recordMode`, an
unknown and a zero limit of each container, a `nil` engine and a `nil` catalog,
and the exact declared contract variables. Every rejection additionally proves
that the stored quantity, the `saveRevision` and `UnsavedChanges` are unchanged.

A stack capability that is enabled without rules, or with `maxPerStack` `0`, is
rejected by GameCatalog schema validation itself, so no validated catalog can
carry one. The endpoint's guard against it is unreachable from the endpoint tests
by construction and exists to keep the projection fail-closed.

The SaveEngine tests own the rest of the matrix: both platforms, both containers,
both sections, the revision guard, the container total across sections, the
preserved high bit, the anti-TOCTOU game-ID guard, the rollback, and concurrent
access under `-race`.

## Current limitations

- **Local transport only.** The HTTP route is available only in the loopback
  explorer. It is absent under `-allow-external-bind`, and there is no Wails
  binding, CLI command or frontend.
- **Separate persistence.** The change remains in the session's private snapshot
  until `WriteSave` succeeds. Closing the session first discards it.
- The endpoint sets a quantity. It does not add, remove, move, merge, split or
  reorder a record, and it never creates the missing record for an item the
  character does not own.
- It accepts no mode, so Safe Mode and Chaos Mode limits are out of scope here.
