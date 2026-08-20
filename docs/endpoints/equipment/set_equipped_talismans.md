# SetEquippedTalismans

## Overview

`SetEquippedTalismans` atomically replaces the compact talisman loadout of one
active character. The request identifies owned physical Inventory records;
raw handles, save offsets and the technical equipment field 21 are not public.

| | |
|---|---|
| EndpointID | `set_equipped_talismans` |
| Kind | Mutation |
| Domain | `equipment` |
| Implementation status | implemented |
| Transport | `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-talismans` in the loopback-only local explorer |
| Save access | atomic mutation of the session's private snapshot |
| Persistence | a later `WriteSave` persists the changed snapshot |

The route is not registered when the explorer runs with
`-allow-external-bind`.

## Request

```json
{
  "orderedOwnedItemIDs": ["oi-sess-0-1", "oi-sess-0-2"],
  "expectedRevision": "0"
}
```

The list contains at most four `ownedItemID` tokens in desired slot order and
cannot be longer than the character's unlocked talisman-slot count. It is
compact: there are no `null` gaps. An empty array clears all four visible
talisman fields.

Every token must identify a positive-quantity record in Inventory `common` for
the requested character and current revision. Its GameCatalog document must
have family `talisman` and a known, enabled equipment rule allowing the
`talisman` slot. The same talisman cannot appear twice.

## Response

```json
{
  "saveSessionID": "...",
  "saveRevision": "1",
  "characterID": 0,
  "orderedResources": [
    {"kind": "item", "key": "200003E8"},
    {"kind": "item", "key": "20000474"}
  ],
  "unlockedSlots": 4
}
```

The receipt replaces revision-scoped input tokens with public catalog
identities and reports the character's effective slot count.

## Save mutation

SaveEngine validates the complete current and target state before writing. For
the four player-visible talisman fields 17–20 it synchronises:

1. `EquipedItemIndex`: `0x180 + Inventory common row`;
2. `EquipedItemsID`: the bare talisman ID;
3. `ActiveEquipedItemsGa`: the exact `0xA...` inventory handle;
4. the dynamic equipped-armaments block: the canonical `0x2...` game ID.

Empty fields use the confirmed native tuple
`{0xFFFFFFFF, 0xFFFFFFFF, 0, 0xFFFFFFFF}`. Field 21, equipment hashes,
Inventory, Storage and every unrelated byte remain untouched. All four written
ranges are read back; a failure restores their original bytes and does not
advance the revision.

PC and PS4 use the same slot-internal representation.

## Validation failures

The endpoint fails closed for a missing engine or catalog, an inactive
character, invalid or stale revision, too many entries for the unlocked slots,
an empty, unknown, foreign or stale token, a Storage or Inventory key record,
zero quantity, a non-talisman or ineligible catalog item, a duplicate talisman,
or inconsistent existing representations.

## Evidence and scope

SaveForge 1.6.10 and native cases T544/T547/T548 confirm four player-visible
talisman fields, four representations, native empty sentinels and preservation
of the equipment hashes. SaveForge 1.5.8 included the unverified fifth field;
SaveForge 2.0 deliberately excludes field 21 and leaves it byte-for-byte
unchanged.
