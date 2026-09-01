# SetPouchItems

## Overview

`SetPouchItems` atomically replaces all six Pouch slot positions for one active
character. The request uses public `ownedItemID` tokens for occupied slots and
`null` for empty positions; raw handles, offset calculations and physical inventory
indices are not part of the public input contract.

| | |
|---|---|
| EndpointID | `set_pouch_items` |
| Kind | Mutation |
| Domain | `equipment` |
| Implementation status | implemented |
| Transport | `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/pouch-items` in the loopback-only local explorer |
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
    null
  ],
  "expectedRevision": "0"
}
```

`slotAssignments` must contain exactly six entries in stored order. A `null`
entry clears that exact slot using the native empty sentinels; positions between
occupied slots are allowed and slots are never shifted or compacted.

Each non-null entry is resolved by its `ownedItemID` token and must satisfy:

- it resolves to an active record in the common section of Inventory for the requested character;
- its quantity is greater than zero;
- its GameCatalog item document has family `goods`;
- `item.capabilities.equipment` is known, enabled, has rules and allows the `pouch` slot;
- the same `ownedItemID` or physical inventory record is not assigned to multiple Pouch slots.

Storage records are not carried and cannot be equipped.

`expectedRevision` is the exact canonical decimal revision of the loaded session.
A stale or non-canonical value is rejected.

## Response

```json
{
  "operationID": "op-3f9c...",
  "operationKind": "set_pouch_items",
  "saveSessionID": "...",
  "saveRevision": "1",
  "changedScopes": ["save.session", "equipment.loadout", "diagnostics.report"],
  "characterID": 0,
  "slotAssignments": [
    {"kind": "item", "key": "40000064"},
    null,
    {"kind": "item", "key": "40000065"},
    null,
    null,
    null
  ]
}
```

The response reports the new revision and the six public catalog resource positions.
It exposes no inventory handle, raw offset, physical index or stale `ownedItemID`.

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
  `set_pouch_items`.
- `changedScopes` are exactly `save.session`, `equipment.loadout`,
  `diagnostics.report`, in that canonical order. This mutation writes only the
  loadout fields of the slot, so neither Inventory nor Storage is invalidated.

A committed assignment identical to the current one still advances
`saveRevision` and still returns a complete receipt with a fresh `operationID`:
the central commit path runs even when no byte changes.

## Save mutation

SaveEngine validates the session, revision, active slot, Inventory ownership,
catalog rules and complete final configuration before writing. The six Pouch
slots require synchronising two representations:

1. `EquipItemData` pouch section: six pairs of `{handle, equipIndex}` starting at `anchor + 0x92CD` (where `equipIndex` is `0x180 + row` for occupied slots and `0xFFFFFFFF` for empty slots);
2. `equipped-armaments` tail: six 4-byte `GoodsParam ID` fields starting at `armamentsOff + 0x80` (where empty slot is `0xFFFFFFFF`).

Quick Items (the preceding 10 pairs and active index) and all other fields remain
untouched. Writes are verified by readback. Verification failure restores the
original bytes; a rejected or rolled-back request does not advance the revision and
does not mark the session dirty.

PC and PS4 use the same slot-internal layout.

## Validation failures

The endpoint fails closed for a missing engine or catalog, a selection whose
length is not six, an unowned, unequipable or zero-quantity item, duplicate slots,
an inactive slot, a stale revision or a malformed save-side layout.

## Evidence and scope

SaveForge 1.5.8 had no pouch writer. SaveForge 1.6.10 confirmed dual representation
writing, empty slot sentinels, position preservation and duplicate checks. SaveForge 2.0
reimplements those confirmed invariants through its own GameCatalog, SaveEngine
session, and revision model.

## Verification

`SetPouchItems` is verified through its unit tests.
