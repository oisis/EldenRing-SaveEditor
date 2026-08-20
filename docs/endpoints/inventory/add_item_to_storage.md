# AddItemToStorage

## Overview

`AddItemToStorage` adds one resolved goods or talisman resource to the common
section of a character's Storage Box. The change is made in the loaded
session's private snapshot; [`WriteSave`](../savesession/write_save.md) remains the
separate persistence operation.

| | |
|---|---|
| EndpointID | `add_item_to_storage` |
| Kind | Mutation |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport | `POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/storage/items` in the local explorer |
| Save access | atomic private-snapshot mutation with rollback |
| Implementation | [../../../backend/endpoints/inventory/add_item_to_storage.go](../../../backend/endpoints/inventory/add_item_to_storage.go) |
| Tests | [../../../backend/endpoints/inventory/add_item_to_storage_test.go](../../../backend/endpoints/inventory/add_item_to_storage_test.go) and [../../../backend/saveengine/add_item_to_storage_test.go](../../../backend/saveengine/add_item_to_storage_test.go) |

## Supported resources

The endpoint accepts only `ItemDocument` resources of family `goods` or
`talisman`. These families have save handles derived from their game IDs and do
not require allocation in the variable-length GaItem table.

The following requests are rejected before mutation:

- every other family, including weapons, armour, Ashes of War, spells and
  spirit ashes;
- category `key_items`, whose common/key routing subset is not represented in
  GameCatalog;
- subcategory `Flasks`, whose shared charge limit is unresolved;
- goods whose `item.goods.isDepositable` is unknown or false;
- an unknown or zero `item.storage.maxStorage`;
- an unknown or inconsistent family, game ID, record mode or stack capability.

The endpoint does not restore the retired GaItem allocator or repacker and never
writes the key section.

## Request

```http
POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/storage/items
Content-Type: application/json
```

```json
{
  "kind": "item",
  "key": "400006A4",
  "variantID": null,
  "quantity": 3,
  "expectedRevision": "0"
}
```

Unknown JSON fields are rejected. `kind`, `key`, `quantity` and
`expectedRevision` are required; `variantID` is optional. Strings are passed
unchanged and are never trimmed or normalised.

| Parameter | Type | Rule |
|---|---|---|
| `saveSessionID` | string | Existing session identifier, matched exactly. |
| `characterID` | integer | Active physical slot `0..9`. |
| `kind` | string | Exact GameCatalog resource kind. |
| `key` | string | Exact key inside that kind. |
| `variantID` | optional uint32 | Exact stored variant game ID; absence selects the base document. |
| `quantity` | uint32 | Positive amount added, not a target total. Separate instances accept `1` only. |
| `expectedRevision` | string | Canonical decimal current `saveRevision`. |

## Quantity and limits

For a `quantity_stack`, the first matching common Storage record is increased
by `quantity`. The result is rejected rather than clamped when it exceeds
`item.storage.maxStorage`. No overflow is moved into another record.

For `separate_instances`, every call creates one physical row and `quantity`
must equal `1`.

Storage uses `item.storage.maxStorage` as both the per-record and total
container limit. This matches the repository-quantity rule shared by SaveForge
1.5.8 and 1.6.8. The total is calculated across all matching Storage records by
resolved game ID. A matching record in either Inventory or Storage key section
rejects the common-only operation.

## Mutation rules

A top-up changes the four quantity bytes and the four AcquisitionIndex bytes of
the selected row, and advances Storage NextAcquisitionSortId to the next free
bucket. The stored high bit of the quantity is preserved exactly; common count,
NextEquipIndex, physical row and GaItemData remain unchanged.

A new record changes:

- the first free common Storage row;
- the common Storage count;
- Storage acquisition metadata;
- one active GaItemData entry only when the game ID has no physical record in
  Inventory or Storage and no existing active entry.

No section changes length and no GaItem table record is allocated.

### Storage allocation rule

Depositing a new record into common Storage follows the unified SaveForge 2.0
allocator confirmed by native T310 and T330 evidence:

1. Derives the effective bucket:
   `effectiveBucket = max(storedNextAcquisitionSortId, maxExistingBucket + 1, 1)`
   where `maxExistingBucket` is the highest `acquisitionIndex / 2` among active
   common records below `50000`.
2. Assigns the even index `2 * effectiveBucket` with stride 2 to the new record.
3. Advances `NextAcquisitionSortId` to `effectiveBucket + 1`.
4. Sets `NextEquipIndex` to `128 + highestOccupiedPhysicalRow`, where the row is
   the highest physically occupied index of the common section after the
   insertion. The stored value is never carried over: a record filling a hole
   below that row leaves the counter unchanged, and only a record extending the
   table raises it by one. An initial deposit into row 0 therefore yields `128`.
   Removing a record never recomputes this counter.

An acquisition allocator that cannot advance without wrapping is rejected before
mutation.

## Response

```json
{
  "saveSessionID": "6bd5...",
  "saveRevision": "1",
  "characterID": 0,
  "gameID": 1073743524,
  "added": 3,
  "quantity": 3,
  "createdRecord": true,
  "containerSection": "common",
  "physicalIndex": 0
}
```

`added` is the requested delta. `quantity` is the resulting record quantity.
`createdRecord` distinguishes a new row from a top-up.

The response deliberately carries no `ownedItemID`. A successful commit
advances the revision and retires identities from the previous one. Read
[`GetStorage`](get_storage.md) under the returned revision to obtain the new
identity.

## Atomicity

SaveEngine validates the revision, activity, both containers, key conflicts,
limits, free row, count header, allocators and GaItemData plan while holding the
session lock. Only after the complete plan succeeds are its non-overlapping byte
ranges written and verified. A failed write or verification restores the exact
previous bytes and does not advance the revision, mark the session dirty or
retire an identity.

PC and PS4 use their platform-specific slot bases. The Storage record layout and
mutation rules inside a slot are identical.
