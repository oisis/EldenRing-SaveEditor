# SetEquippedArmaments

## Overview

`SetEquippedArmaments` atomically replaces all six hand-armament slots of one
active character. Public callers identify existing Inventory records with
revision-scoped `ownedItemID` values and never send GaItem handles or offsets.

| | |
|---|---|
| EndpointID | `set_equipped_armaments` |
| Kind | Mutation |
| Domain | `equipment` |
| Implementation status | implemented |
| Transport | `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-armaments` in the loopback-only local explorer |
| Save access | one atomic mutation of the session's private snapshot |
| Persistence | a later `WriteSave` persists the changed snapshot |

The route is not registered when the explorer runs with
`-allow-external-bind`.

## Request

```json
{
  "slotAssignments": [
    "owned-left-1-token",
    "owned-right-1-token",
    null,
    null,
    "owned-left-3-token",
    "owned-right-3-token"
  ],
  "expectedRevision": "0"
}
```

`slotAssignments` always contains exactly six positions in stored order:

1. left hand 1;
2. right hand 1;
3. left hand 2;
4. right hand 2;
5. left hand 3;
6. right hand 3.

A string must be an `ownedItemID` minted by `GetInventory` for the requested
character and current revision. It must still address a positive-quantity
weapon record in Inventory `common`. The resolved GameCatalog item must have
family `weapon` and a known, enabled equipment capability allowing the exact
left-hand or right-hand position. One physical weapon record cannot be assigned
to more than one position.

`null` clears that position by selecting the native `Unarmed` item
`0x0001ADB0`. Native saves may use one shared physical `Unarmed` record for
several empty hand slots. That record must already exist in Inventory `common`
and GaItem. SaveForge 2.0 does not allocate it or repack GaItem, so its absence
is a hard error.

## Response

```json
{
  "saveSessionID": "...",
  "saveRevision": "1",
  "characterID": 0,
  "slotAssignments": [
    {"kind": "item", "key": "000F4240", "gameID": 1000000},
    {"kind": "item", "key": "000F4240", "gameID": 1000000},
    null,
    null,
    {"kind": "item", "key": "003085E0", "gameID": 3180000},
    {"kind": "item", "key": "003085E0", "gameID": 3180000}
  ]
}
```

The response uses the canonical public catalog identity together with the exact
materialized `gameID`, so an upgraded or infused weapon variant remains
distinguishable. A cleared position is `null`; the technical `Unarmed` item
never becomes part of the public assignment.

## Save mutation

SaveEngine validates the complete existing state and target plan before the
first write. For equipment fields 0–5 it keeps four native representations
synchronized:

1. `EquipedItemIndex`: `0x180 + Inventory common row`;
2. `EquipedItemsID`: the weapon game ID;
3. `ActiveEquipedItemsGa`: the exact `0x8...` Inventory handle;
4. the dynamic equipped-armaments block: the complete weapon game ID.

The four ranges are written and read back as one operation. Failure restores
their previous bytes and leaves the revision and dirty state unchanged.
Ammunition, armor, talismans, Inventory rows, Storage, GaItem records,
equipment hashes and every unrelated field remain untouched.

## Validation failures

The endpoint fails closed for a missing engine or catalog, an assignment count
other than six, an inactive character, an invalid or stale revision, an empty,
unknown, foreign or stale token, a Storage or key record, zero quantity, an
unresolved or non-weapon handle, a catalog item of the wrong family or hand
slot, a duplicated physical weapon record, a missing native `Unarmed` record,
or inconsistent existing equipment representations.

## Evidence and scope

SaveForge 1.5.8 wrote one equipment representation and recomputed an equipment
hash. SaveForge 1.6.10 corrected that behavior from native cases T544, T547 and
T548: the game maintains four representations, preserves the existing hash and
allows cleared hand slots to share one `Unarmed` record. SaveForge 2.0 follows
that native-backed contract without restoring the retired allocator or slot
repacker used by 1.6.10 when the technical record was absent.

PC and PS4 use the same slot-internal representation. Platform handling remains
owned by SaveEngine and the save codecs. Ammunition slots 6–9 are not part of
this endpoint.

## Local verification

```text
go test ./backend/saveengine -run TestSetEquippedArmaments -count=1
go test ./backend/endpoints/equipment -run TestSetEquippedArmaments -count=1
go test ./tools/swagger -run TestSetEquippedArmamentsRoute -count=1
```
