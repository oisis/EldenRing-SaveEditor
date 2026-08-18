# SetStorageOrder

`SetStorageOrder` replaces the complete logical order of the supported records
in the common Storage section. It changes acquisition indices, not physical
rows.

## Endpoint

```text
PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/storage/order
```

The explorer registers this route only in local loopback mode. An explorer
started with `-allow-external-bind` answers `404`.

## Request

```json
{
  "orderedOwnedItemIDs": ["current-token-a", "current-token-b"],
  "expectedRevision": "7"
}
```

| Field | Meaning |
|---|---|
| `saveSessionID` | Exact identifier of the loaded private save session. |
| `characterID` | Physical character slot, `0..9`. |
| `orderedOwnedItemIDs` | Complete permutation of the current supported common Storage instances. |
| `expectedRevision` | Current canonical decimal `saveRevision`. |

The list must contain every supported instance exactly once. Empty, duplicate,
stale, foreign, Inventory, key and unsupported identities are rejected. Unknown
JSON fields are rejected, and strings are never trimmed or normalised.

## Supported records

The endpoint resolves every common Storage handle to its exact save-side
`gameID`, then reads the corresponding ItemDocument from GameCatalog. A record
participates only when `item.category` is known and equals one of:

- `melee_armaments`;
- `ranged_and_catalysts`;
- `shields`;
- `talismans`;
- `head`, `chest`, `arms` or `legs`.

The technical Unarmed resource with key `0001ADB0` is excluded. Every other
category and every Storage key record is retained byte-for-byte but omitted
from the request. Unknown items or categories reject the whole operation
instead of being guessed or skipped.

The order is global across the supported records. The game filters those
records into its Storage tabs, so the relative order visible within each tab is
the corresponding subsequence of this list.

## Mutation contract

Storage uses acquisition-sort buckets derived from `acquisitionIndex >> 1`.
Consequently the endpoint assigns fresh even indices with a stride of two. The
range:

- starts at or above `2 * Storage NextAcquisitionSortId` (or index `2` when the counter is zero), without any reserved range;
- skips ranges whose buckets collide with retained common or key records;
- must end below `10000`, the native unsafe boundary established by the legacy
  save experiments.

Storage `NextAcquisitionSortId` becomes the next free sort bucket (`lastAssignedIndex / 2 + 1`).
Storage `NextEquipIndex` is a separate native counter and remains unchanged.

Only the four acquisition-index bytes of supported common records whose values
change, plus the four bytes of Storage `NextAcquisitionSortId`, may change.
Physical rows, handles, quantities, unsupported and key records, Inventory,
Equipment, GaItem data and every other slot byte remain untouched.

The operation mutates only the session's private snapshot. `WriteSave` remains
a separate endpoint. Every validation finishes before the first write, and a
failed write is rolled back. A rejection changes no bytes, revision, dirty flag
or identity. A success advances `saveRevision` exactly once and invalidates all
`ownedItemID` values minted under the previous revision.

## Response

```json
{
  "saveSessionID": "session-id",
  "saveRevision": "8",
  "characterID": 0,
  "orderedResources": [
    {"kind": "item", "key": "100704E0"}
  ],
  "acquisitionIndices": [14]
}
```

`orderedResources` reports the committed order using stable GameCatalog
identities. `acquisitionIndices` reports the newly assigned native indices in
the same order. The request tokens are deliberately not echoed because they are
already stale after the revision advance.

## Legacy comparison

SaveForge 1.5.8 and 1.6.8 use the same supported Storage categories, exclude
Unarmed and require a complete order. Controlled native Storage additions in
the 1.6.8 research workspace establish the same even stride-two acquisition
indices used for Inventory.

The old writer also changed Storage `NextEquipIndex`. Later native evidence
established that it is an independent input counter. SaveForge 2.0 therefore
preserves it.

## Verification coverage

Focused coverage exercises PC and PS4, complete reversed order, preservation of
Storage `NextEquipIndex`, key records, Inventory and unrelated bytes,
serialize-reload semantics, incomplete and duplicate lists, Inventory
identities, unsupported categories, strict HTTP JSON decoding, OpenAPI and
Scalar synchronization.
