# SetQuickItems

## Overview

`SetQuickItems` atomically replaces all ten Quick Items positions for one active
character. Occupied positions use public `ownedItemID` tokens and `null` clears
that exact position. Raw handles and physical indices are not public input.

| | |
|---|---|
| EndpointID | `set_quick_items` |
| Kind | Mutation |
| Domain | `equipment` |
| Implementation status | implemented |
| Transport | `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/quick-items` in the loopback-only local explorer |
| Save access | atomic mutation of the session's private snapshot |
| Persistence | a later `WriteSave` persists the changed snapshot |

The route is not registered when the explorer runs with `-allow-external-bind`.

## Request

```json
{
  "slotAssignments": [
    "oi-sess-0-1",
    null,
    "oi-sess-0-2",
    null,
    null,
    null,
    null,
    null,
    null,
    null
  ],
  "expectedRevision": "0"
}
```

`slotAssignments` contains exactly ten entries in stored order. `null` clears
the position; gaps remain gaps and positions are never compacted.

Each non-null token must identify a positive-quantity record in the common
section of Inventory for the requested character. Its GameCatalog document must
have family `goods` and a known, enabled `item.capabilities.equipment` rule that
allows `quick_item`. Storage and Inventory key records are rejected. One goods
item cannot occupy two Quick Items positions, while the same item may also
appear in the Pouch.

`expectedRevision` must be the session's current canonical decimal revision.

## Response

```json
{
  "saveSessionID": "...",
  "saveRevision": "1",
  "characterID": 0,
  "slotAssignments": [
    {"kind": "item", "key": "400006A4"},
    null,
    {"kind": "item", "key": "40000065"},
    null,
    null,
    null,
    null,
    null,
    null,
    null
  ]
}
```

The receipt uses public catalog identities and exposes no raw handle, offset,
physical index or stale `ownedItemID`.

## Save mutation

SaveEngine validates the complete current and target state before writing. It
synchronises two native representations:

1. ten `{handle, equipIndex}` pairs at `anchor + 0x9279`, where an occupied
   `equipIndex` is `0x180 + Inventory common row` and an empty pair is
   `{0, 0xFFFFFFFF}`;
2. ten direct `GoodsParam` IDs at `armamentsAt + 0x58`, where an empty position
   is `0xFFFFFFFF`.

The four-byte active Quick Item index at `anchor + 0x92C9`, all Pouch fields,
Inventory quantities and every unrelated byte remain unchanged. Both written
ranges are read back. A failure restores their original bytes; a rejected or
rolled-back request does not advance the revision or mark the session dirty.

PC and PS4 use the same slot-internal layout.

## Validation failures

The endpoint fails closed for a missing engine or catalog, a position count
other than ten, an inactive character, an invalid or stale revision, an
unowned, zero-quantity, non-goods or unequipable item, a duplicate Quick Items
assignment, or inconsistent existing Quick Items data.

## Evidence and scope

SaveForge 1.5.8 had no Quick Items writer. SaveForge 1.6.10 confirms the dual
representation, native empty sentinels, within-family duplicate rejection,
cross-family allowance and preservation of the active Quick Item index.
SaveForge 2.0 reimplements those invariants through its own GameCatalog,
SaveEngine session and revision model.

## Verification

`SetQuickItems` is verified through its SaveEngine, endpoint and transport tests.
