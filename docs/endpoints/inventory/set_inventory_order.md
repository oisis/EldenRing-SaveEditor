# SetInventoryOrder

`SetInventoryOrder` replaces the complete logical order of the supported
records in the common Inventory section. It changes acquisition indices, not
physical rows, so Equipment, Quick Items and Pouch references remain valid.

## Endpoint

```text
PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/inventory/order
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
| `orderedOwnedItemIDs` | Complete permutation of the current supported common Inventory instances. |
| `expectedRevision` | Current canonical decimal `saveRevision`. |

The list must contain every supported instance exactly once. Empty, duplicate,
stale, foreign, Storage, key and unsupported identities are rejected. Unknown
JSON fields are rejected, and strings are never trimmed or normalised.

## Supported records

The endpoint resolves every common Inventory handle to its exact save-side
`gameID`, then reads the corresponding ItemDocument from GameCatalog. A record
participates only when `item.category` is known and equals one of:

- `melee_armaments`;
- `ranged_and_catalysts`;
- `shields`;
- `talismans`;
- `head`, `chest`, `arms` or `legs`.

The technical Unarmed resource with key `0001ADB0` is excluded. Every other
category and every Inventory key record is retained byte-for-byte but omitted
from the request. Unknown items or categories reject the whole operation
instead of being guessed or skipped.

The order is global across the supported records. The game filters those
records into its Inventory tabs, so the relative order visible within each tab
is the corresponding subsequence of this list.

## Mutation contract

The game compares acquisition-sort buckets derived from
`acquisitionIndex >> 1`. Consequently the endpoint assigns fresh even indices
with a stride of two. The range:

- starts at or above `434`, after the native reserved range `0..432`;
- starts at or above the current `NextAcquisitionSortId`;
- skips ranges whose buckets collide with retained common or key records;
- must end below `10000`, the native unsafe boundary established by the legacy
  save experiments.

`NextAcquisitionSortId` becomes the last assigned index plus one.
`NextEquipIndex` is a separate native counter and remains unchanged.

Only the four acquisition-index bytes of supported common records whose values
change, plus the four bytes of `NextAcquisitionSortId`, may change. Physical
rows, handles, quantities, unsupported and key records, Equipment, Storage,
GaItem data and every other slot byte remain untouched.

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
  "acquisitionIndices": [434]
}
```

`orderedResources` reports the committed order using stable GameCatalog
identities. `acquisitionIndices` reports the newly assigned native indices in
the same order. The request tokens are deliberately not echoed because they are
already stale after the revision advance.

## Legacy comparison

SaveForge 1.5.8 and 1.6.8 use the same supported Inventory categories, exclude
Unarmed and require a complete order. Native save experiments in the 1.6.8
workspace established the even stride-two rule and the unsafe `10000`
boundary.

The old writer also changed `NextEquipIndex`. Later native evidence established
that it is independent from `NextAcquisitionSortId` and that forcing them to
track each other can corrupt a save. SaveForge 2.0 therefore preserves
`NextEquipIndex`.

## Verification coverage

Focused coverage exercises PC and PS4, complete reversed order, retained-bucket
collision avoidance, preservation of `NextEquipIndex`, key records and all
unrelated bytes, serialize-reload semantics, incomplete and duplicate lists,
Storage identities, unsupported categories, the unsafe index boundary, strict
HTTP JSON decoding, OpenAPI and Scalar synchronization.
