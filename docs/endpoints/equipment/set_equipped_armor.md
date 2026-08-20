# SetEquippedArmor

## Overview

`SetEquippedArmor` atomically replaces the head, chest, arms and legs armor of
one active character. Public callers identify existing Inventory records with
revision-scoped `ownedItemID` values and never send GaItem handles or offsets.

| | |
|---|---|
| EndpointID | `set_equipped_armor` |
| Kind | Mutation |
| Domain | `equipment` |
| Implementation status | implemented |
| Transport | `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-armor` in the loopback-only local explorer |
| Save access | one atomic mutation of the session's private snapshot |
| Persistence | a later `WriteSave` persists the changed snapshot |

The route is not registered when the explorer runs with
`-allow-external-bind`.

## Request

```json
{
  "slotAssignments": [
    "owned-head-token",
    "owned-chest-token",
    null,
    "owned-legs-token"
  ],
  "expectedRevision": "0"
}
```

`slotAssignments` always contains exactly four positions in this order:

1. head;
2. chest;
3. arms;
4. legs.

A string must be an `ownedItemID` minted by `GetInventory` for the requested
character and current revision. It must still address a positive-quantity
record in Inventory `common`. The resolved GameCatalog item must have family
`armor` and a known, enabled equipment capability allowing the exact position.

`null` removes visible armor from that position by selecting its native
bare-body item. The native record must already exist in Inventory `common` and
the GaItem table. SaveForge 2.0 does not allocate that technical armor or
repack GaItem, so a missing record is a hard error instead of a synthesized
fallback.

## Response

```json
{
  "saveSessionID": "...",
  "saveRevision": "1",
  "characterID": 0,
  "slotAssignments": [
    {"kind": "item", "key": "10009C40"},
    {"kind": "item", "key": "10009CA4"},
    null,
    {"kind": "item", "key": "10009D6C"}
  ]
}
```

The response uses public catalog identities. A cleared position is `null`; the
technical bare-body item never becomes part of the public assignment.

## Save mutation

SaveEngine validates the complete existing state and target plan before the
first write. For equipment fields 12–15 it keeps four native representations
synchronized:

1. `EquipedItemIndex`: `0x180 + Inventory common row`;
2. `EquipedItemsID`: the game ID without its high family nibble;
3. `ActiveEquipedItemsGa`: the exact `0x9...` Inventory handle;
4. the dynamic equipped-armaments block: the complete armor game ID.

The four ranges are written and read back as one operation. Failure restores
their previous bytes and leaves the revision and dirty state unchanged.
Weapons, ammunition, talismans, Inventory rows, Storage, GaItem records,
equipment hashes and every unrelated field remain untouched.

## Validation failures

The endpoint fails closed for a missing engine or catalog, an assignment count
other than four, an inactive character, an invalid or stale revision, an empty,
unknown, foreign or stale token, a Storage or key record, zero quantity, an
unresolved or non-armor handle, a catalog item of the wrong family or slot, a
duplicate physical record, a missing native bare-body record, or inconsistent
existing equipment representations.

## Evidence and scope

SaveForge 1.5.8 wrote one equipment representation and recomputed an equipment
hash. SaveForge 1.6.10 corrected that behavior from native cases T544, T547 and
T548: the game maintains four representations and the existing hash is
preserved. SaveForge 2.0 follows the later native-backed contract and does not
restore the retired allocator or slot repacker used by 1.6.10 to provision a
missing bare-body record.

PC and PS4 use the same slot-internal representation. Platform handling remains
owned by SaveEngine and the save codecs.

## Local verification

```text
go test ./backend/saveengine -run TestSetEquippedArmor -count=1
go test ./backend/endpoints/equipment -run TestSetEquippedArmor -count=1
go test ./tools/swagger -run TestSetEquippedArmorRoute -count=1
```
