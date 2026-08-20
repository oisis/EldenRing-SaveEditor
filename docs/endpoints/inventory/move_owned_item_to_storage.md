# MoveOwnedItemToStorage

`MoveOwnedItemToStorage` atomically moves one existing physical record from the
common section of Inventory to the common section of Storage.

## Endpoint

```text
POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/move-to-storage
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
| `ownedItemID` | Opaque, revision-scoped identity minted by an Inventory read. |
| `targetPosition` | Zero-based position among non-empty common Storage records; the current count appends. |
| `expectedRevision` | Current canonical decimal `saveRevision`. |

Unknown JSON fields are rejected. Strings are never trimmed or normalised.

## Catalog rule

The endpoint resolves the addressed handle to its exact save-side `gameID` and
then to one `ItemDocument`. `item.storage.maxStorage` must be known and greater
than zero. The total quantity of that `gameID` already present in both Storage
sections, plus the complete quantity of the moved record, must not exceed that
limit.

No Safe Mode, `-sfv` or inferred fallback limit is used.

`item.storage.recordMode` must be known and must be either `quantity_stack` or
`separate_instances`. An unknown or unsupported record mode is rejected fail-closed.

## Mutation contract

Only `Inventory common -> Storage common` is supported. Inventory key records
are rejected because SaveForge 2.0 has no confirmed write contract for the
Storage key section.

The operation moves the complete twelve-byte record:

- the GaItem handle is preserved;
- the raw quantity, including its high flag bit, is preserved;
- Storage receives a fresh acquisition index;
- the source row becomes `{0, 0, physicalRow}`;
- the Inventory common count is lowered when it is non-zero;
- the Storage common count is raised by one;
- Storage `NextAcquisitionSortId` advances to the next free bucket;
- Storage `NextEquipIndex` is set to `128 + highestOccupiedPhysicalRow` of the
  common section as the insertion leaves it, so a rotation that consumes a hole
  below the highest occupied row keeps the counter unchanged.

The destination insertion rotates one contiguous physical span needed to place
the record at `targetPosition`. Existing gaps and rows outside that span are
not normalised.

There is no quantity merge, duplicate-handle rewrite, GaItem allocation,
rehandle, repack or cascade. If the item's recordMode is `quantity_stack`
and Storage already holds a record for that game ID, the move is rejected
fail-closed. Records of `separate_instances` items remain separate physical
records and still participate in the catalog storage limit.

Equipment, Quick Items and Pouch references address an Inventory common
instance by its physical row and handle. If any of those structures still
references the source row, the move is rejected and the caller must unequip it
first. Nothing is silently cleared.

All validation completes before the first write. The non-overlapping write
ranges are verified together and restored on failure. A rejection does not
advance the revision or mark the session dirty. A success advances
`saveRevision` exactly once and retires every `ownedItemID` from the previous
revision.

## Response

```json
{
  "saveSessionID": "session-id",
  "saveRevision": "8",
  "ownedItemID": "stale-source-token",
  "characterID": 0,
  "gameID": 268895456,
  "quantity": 1,
  "containerSection": "common",
  "targetPosition": 0,
  "physicalIndex": 0,
  "acquisitionIndex": 2
}
```

The echoed `ownedItemID` is already stale. Read Storage under the returned
revision to obtain the destination record's new identity. `physicalIndex` is
the row selected after insertion; it is informative and not a stable identity.

## Fail-closed cases

The request is rejected before mutation when, among other cases:

- the session, character, revision or owned-item identity is invalid or stale;
- the record is not in Inventory common;
- the item is absent from GameCatalog or has no positive `maxStorage`;
- the Storage total would exceed `maxStorage`;
- `targetPosition` is outside `0..commonRecordCount`;
- common Storage has no free physical row;
- the handle cannot be resolved;
- Equipment, Quick Items or Pouch still references the source row;
- a planned write cannot be verified.

## Legacy comparison

SaveForge 1.5.8 and 1.6.10 agree on the relevant direct-transfer rules: an
Inventory-to-Storage move preserves the record handle and quantity, rejects an
equipped item, assigns a fresh Storage index and advances `NextAcquisitionSortId`.
In 2.0, every deposit into Storage also updates `NextEquipIndex`, which follows
the physical layout of the common section rather than the number of deposits, in
accordance with the native Test 3 evidence confirmed in 1.6.10. Version 1.6.10
changed allocator details for duplicate instance handles, but this endpoint
deliberately does not invoke that retired allocation/repack path.

The legacy workspace accepted a requested list position and later rewrote a
whole container. SaveForge 2.0 instead applies one direct, verified mutation to
the private snapshot and keeps unrelated Storage bytes unchanged.

## Verification

Focused coverage includes PC and PS4, insertion between sparse Storage rows,
raw quantity preservation, allocator values, active-reference rejection,
position and limit rejection, the catalog `maxStorage` gate, strict HTTP
JSON decoding and serialize-reload validation.
