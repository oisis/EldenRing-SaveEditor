# SetEquippedSpells

## Overview

`SetEquippedSpells` atomically replaces the compact spell memory positions for
one active character. The request uses public GameCatalog resource identities;
raw save identifiers and physical slot indices 13–14 are not part of the input
contract.

| | |
|---|---|
| EndpointID | `set_equipped_spells` |
| Kind | Mutation |
| Domain | `equipment` |
| Implementation status | implemented |
| Transport | `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-spells` in the loopback-only local explorer |
| Save access | atomic mutation of the session's private snapshot |
| Persistence | a later `WriteSave` persists the changed snapshot |

The route is not registered when the explorer runs with `-allow-external-bind`.

## Request

```json
{
  "orderedResources": [
    {"kind": "item", "key": "40000FA0"},
    {"kind": "item", "key": "40000FB4"}
  ],
  "expectedRevision": "0"
}
```

`orderedResources` contains at most 12 spell resources in desired memory slot order. An empty array (`[]`) clears the entire public spell loadout with native empty sentinels (`0xFFFFFFFF`). `nil` entries, empty elements, or gaps in the array are rejected.

Each non-null entry is resolved by its exact `(kind, key)` pair and must satisfy all of these conditions:

- it has an `ItemDocument` with known family `spell`, a known game ID prefixed with `0x40000000`, and known `memorySlots > 0`;
- `item.capabilities.equipment` is known, enabled, has rules and allows the `spell_memory` slot;
- the same spell cannot be listed more than once;
- the sum of `memorySlots` for all listed spells cannot exceed the character's available Memory Slots capacity (or the game maximum of 12).

`expectedRevision` is the exact canonical decimal revision of the loaded session. A stale, padded, signed or otherwise non-canonical value is rejected.

## Response

```json
{
  "saveSessionID": "...",
  "saveRevision": "1",
  "characterID": 0,
  "orderedResources": [
    {"kind": "item", "key": "40000FA0"},
    {"kind": "item", "key": "40000FB4"}
  ],
  "usedMemorySlots": 2,
  "availableMemorySlots": 5
}
```

The response reports the new revision, the validated ordered public resources, the sum of Memory Slots used, and the total available Memory Slots capacity.

## Save mutation

SaveEngine validates the session, revision, active slot, physical positions 13–14, duplicate spells, and total Memory Slots cost before writing. The mutation works as follows:

1. find the confirmed slot anchor;
2. check physical positions 13 and 14 (bytes 96..111 of the EquippedSpells section). If either position is not natively empty (`spellID=0xFFFFFFFF, follower=0x00000000`), the mutation fails immediately with zero changes;
3. compute available Memory Slots capacity using Memory Stones and Moon of Nokstella;
4. compare total Memory Slots cost against available capacity;
5. write up to 12 records (bytes 0..95 of the EquippedSpells section) with raw MagicParam IDs and `0xFFFFFFFF` follower fields, filling remaining slots up to 12 with native empty pairs;
6. update the active spell index at `anchor + 0x9205 + 0x70`:
   - if `orderedResources` is empty, active index is set to `0xFFFFFFFF`;
   - if `orderedResources` is non-empty, preserve the existing numeric index if it falls within `0 .. len(orderedResources)-1`, otherwise set to `0`;
7. physical positions 13 and 14 remain completely untouched byte for byte.

The write is read back for verification. A verification failure restores original bytes; a rejected or rolled-back request does not advance revision and does not mark the session dirty.

## Validation failures

The endpoint fails closed for a missing engine or catalog, a list exceeding 12 spells, an unknown or ineligible resource, duplicate spells, an inactive or invalid character slot, a stale revision, total Memory Slots cost exceeding available capacity, or a non-empty physical position 13 or 14.

## Verification

```bash
go test ./backend/saveengine -run '^TestSetEquippedSpells' -count=1 -v
go test ./backend/endpoints/equipment -run '^TestSetEquippedSpells' -count=1 -v
go test ./tools/swagger -run '^TestEquippedSpells' -count=1 -v
```
