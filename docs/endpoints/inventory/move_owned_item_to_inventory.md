# MoveOwnedItemToInventory

`MoveOwnedItemToInventory` atomically moves one existing physical record from
the common section of Storage to the common section of Inventory.

## Endpoint

```text
POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/move-to-inventory
```

The explorer registers this route only in local loopback mode. An explorer
started with `-allow-external-bind` answers `404`.

## Request

```json
{
  "targetPosition": 0,
  "expectedRevision": "7"
}
```

| Field | Meaning |
|---|---|
| `saveSessionID` | Exact identifier of the loaded private save session. |
| `characterID` | Physical character slot, `0..9`. |
| `ownedItemID` | Opaque, revision-scoped identity minted by a Storage read. |
| `targetPosition` | Zero-based position in the logical acquisition-index order of common Inventory; the current count appends. |
| `expectedRevision` | Current canonical decimal `saveRevision`. |

Unknown JSON fields are rejected. Strings are never trimmed or normalised.

## Catalog rule

The endpoint resolves the addressed handle to its exact save-side `gameID` and
then to one `ItemDocument`. `item.category` must be known and must not be
`key_items`; that broad category does not distinguish resources routed to
Inventory common from resources routed to Inventory key. The common-only
endpoint rejects the ambiguity instead of hardcoding item IDs.

`item.storage.maxInventory` must be known and greater than zero. The total
quantity of that `gameID` already present in both Inventory sections, plus the
complete quantity of the moved record, must not exceed that limit. No Safe Mode
or `-sfv` fallback limit is used.

`item.storage.recordMode` must be known and must be either `quantity_stack` or
`separate_instances`. An unknown or unsupported record mode is rejected fail-closed.

## Mutation contract

Only `Storage common -> Inventory common` is supported. Storage key records are
rejected because SaveForge 2.0 has no confirmed write contract for that source
section.

The operation moves the complete twelve-byte record:

- the GaItem handle is preserved;
- the raw quantity, including its high flag bit, is preserved;
- the source Storage row becomes twelve zero bytes;
- the Storage common count is lowered by one;
- the first free physical Inventory common row receives the moved record;
- the Inventory common count is raised by one;
- Inventory receives a fresh acquisition index from the confirmed allocator;
- Inventory `NextAcquisitionSortId` advances;
- Inventory `NextEquipIndex` remains byte-identical.

`targetPosition` describes the order the game derives from acquisition indices,
not a physical row. Existing common Inventory records are sorted by their raw
acquisition index, the moved record is inserted at the requested position, and
the sorted pool of existing indices plus one fresh index is assigned to that
logical order. Only index fields whose value changes are written.

Existing physical Inventory rows never move. Equipment, Quick Items and Pouch
refer to those rows, so all of their references remain valid and byte-identical.
Duplicate or allocator-inconsistent acquisition indices are rejected instead
of being silently normalised.

There is no quantity merge, handle rewrite, GaItem allocation, GaItemData
change, rehandle, repack or cascade. If the item's recordMode is `quantity_stack`
and Inventory already holds a record for that game ID, the move is rejected
fail-closed. Records of `separate_instances` items remain separate physical
instances, subject to the catalog Inventory limit.

All validation completes before the first write. The non-overlapping write
ranges are verified together and restored on failure. A rejection does not
advance the revision or mark the session dirty. A success advances
`saveRevision` exactly once and retires every `ownedItemID` from the previous
revision.

## Response

```json
{
  "operationID": "op-3f9c...",
  "operationKind": "move_owned_item_to_inventory",
  "saveSessionID": "session-id",
  "saveRevision": "8",
  "changedScopes": ["save.session", "inventory", "storage", "diagnostics.report"],
  "ownedItemID": "stale-source-token",
  "characterID": 0,
  "gameID": 268895456,
  "quantity": 1,
  "containerSection": "common",
  "targetPosition": 0,
  "physicalIndex": 0,
  "acquisitionIndex": 435
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
  `move_owned_item_to_inventory`.
- `changedScopes` are exactly `save.session`, `inventory`, `storage`,
  `diagnostics.report`, in that canonical order.

The echoed `ownedItemID` is already stale. Read Inventory under the returned
revision to obtain the destination record's new identity. `physicalIndex` is
the free row selected for this move and is not a stable identity.

## Fail-closed cases

The request is rejected before mutation when, among other cases:

- the session, character, revision or owned-item identity is invalid or stale;
- the record is not in Storage common;
- the item is absent from GameCatalog, has an unknown category, belongs to the
  ambiguous `key_items` category or has no positive `maxInventory`;
- the Inventory total would exceed `maxInventory`;
- the same item already has an Inventory key record;
- `targetPosition` is outside `0..commonRecordCount`;
- common Inventory has no free physical row;
- a handle cannot be resolved;
- the Storage count is inconsistent with the addressed record;
- acquisition indices are duplicated or exceed their stored high-water mark;
- a planned write cannot be verified.

## Legacy comparison

SaveForge 1.5.8 and 1.6.10 both ordered their Inventory workspace by raw
acquisition index, wrote a transfer into a free destination row and reassigned
the moved record's index. Version 1.6.10 corrected the Inventory allocator to
the native parity-stabilised high-water rule also used by SaveForge 2.0.

The legacy workspace could later wipe and replay a whole container. SaveForge
2.0 instead preserves physical Inventory rows and updates only the ranges needed
for this move, avoiding any need to rewrite Equipment, Quick Item or Pouch row
references.

## Verification

Focused coverage includes PC and PS4, insertion into logical acquisition order,
preservation of physical rows and raw quantity, allocator values, limit,
position and malformed-order rejection, ambiguous key-category rejection,
strict HTTP JSON decoding and serialize-reload validation.
