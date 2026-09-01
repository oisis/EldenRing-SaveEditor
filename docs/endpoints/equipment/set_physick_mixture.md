# SetPhysickMixture

## Overview

`SetPhysickMixture` atomically replaces both positions of the Flask of Wondrous
Physick mixture for one active character. The request uses public GameCatalog
resource identities; raw save identifiers and inventory handles are not part of
the input contract.

| | |
|---|---|
| EndpointID | `set_physick_mixture` |
| Kind | Mutation |
| Domain | `equipment` |
| Implementation status | implemented |
| Transport | `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/physick-mixture` in the loopback-only local explorer |
| Save access | atomic mutation of the session's private snapshot |
| Persistence | a later `WriteSave` persists the changed snapshot |

The route is not registered when the explorer runs with
`-allow-external-bind`.

## Request

```json
{
  "crystalTearResources": [
    {"kind": "item", "key": "40002AF9"},
    null
  ],
  "expectedRevision": "0"
}
```

`crystalTearResources` must contain exactly two entries in stored order. A
`null` entry clears that exact position with the native `0xFFFFFFFF` value; the
other entry is never moved or left-packed.

Each non-null entry is resolved by its exact `(kind, key)` pair and must satisfy
all of these conditions:

- it has an `ItemDocument` with known family `goods` and a known game ID;
- `item.capabilities.equipment` is known, enabled, has rules and allows the
  `physick` slot;
- the character owns the exact resolved game ID with positive quantity in the
  common or key section of Inventory;
- it is not also selected for the other position.

The character must additionally own either physical save form of the Flask of
Wondrous Physick (`0x400000FA` or `0x400000FB`) in Inventory. Storage is not
consulted: an item stored there is not carried and cannot be equipped.

`expectedRevision` is the exact canonical decimal revision of the loaded
session. A stale, padded, signed or otherwise non-canonical value is rejected.

## Response

```json
{
  "operationID": "op-3f9c...",
  "operationKind": "set_physick_mixture",
  "saveSessionID": "...",
  "saveRevision": "1",
  "changedScopes": ["save.session", "equipment.loadout", "diagnostics.report"],
  "characterID": 0,
  "crystalTearResources": [
    {"kind": "item", "key": "40002AF9"},
    null
  ]
}
```

The response reports the new revision and the two public resource positions.
It exposes no inventory handle, raw offset or private snapshot data.

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
  `set_physick_mixture`.
- `changedScopes` are exactly `save.session`, `equipment.loadout`,
  `diagnostics.report`, in that canonical order. This mutation writes only the
  loadout fields of the slot, so neither Inventory nor Storage is invalidated.

A committed assignment identical to the current one still advances
`saveRevision` and still returns a complete receipt with a fresh `operationID`:
the central commit path runs even when no byte changes.

## Save mutation

SaveEngine validates the session, revision, active slot, Inventory ownership
and complete final pair before writing. The two mixture fields are located by
the same dynamic chain used by `GetPhysickMixture`:

1. find the confirmed slot anchor;
2. read the acquired-projectile count at `anchor + 0x931D`;
3. skip the count header, `count * 8` projectile bytes and the `0x9C`
   equipped-armaments block;
4. replace exactly the next eight bytes, which are the first two `uint32`
   values of `EquipPhysicsData`.

The third `uint32` of `EquipPhysicsData` and every other save field remain
untouched. The write is read back for verification. A verification failure
restores the original eight bytes; a rejected or rolled-back request does not
advance the revision and does not mark the session dirty.

PC and PS4 use the same slot-internal layout. Their only difference here is the
platform-specific slot-data base: PC has a per-slot MD5 prefix and PS4 does not.

## Validation failures

The endpoint fails closed for a missing engine or catalog, a selection whose
length is not two, an unknown or ineligible resource, duplicate tears, an
unowned tear or Flask, an inactive or invalid character slot, a stale revision,
or a malformed save-side layout. No resource is guessed, normalised or replaced
with a similarly named tear.

The two Crimson Crystal Tear records `0x40002AFA` and `0x40002AFB` remain
distinct physical items. Either may be selected when that exact ID is owned;
the endpoint does not canonicalise one into the other.

## Evidence and scope

SaveForge 1.5.8 had no separate Physick mixture writer. SaveForge 1.6.10
confirmed the two writable fields, the `0xFFFFFFFF` empty value, preservation of
position order, duplicate rejection, exact Crimson variant handling, Inventory
ownership and Flask ownership. SaveForge 2.0 reimplements those confirmed
invariants through its own GameCatalog, SaveEngine session and revision model;
it imports no legacy package.

## Verification

```bash
go test ./backend/saveengine -run 'Test(Set|Get)PhysickMixture' -count=1
go test ./backend/endpoints/equipment -run 'TestSetPhysickMixture' -count=1
go test ./tools/swagger -run 'Test(SetPhysickMixtureRoute|OpenAPIDocumentMatchesRoutes)$' -count=1
```

The focused SaveEngine test uses synthetic PC and PS4 containers, verifies
in-memory read-back, explicit `WriteSave`, reload and preservation of the source
file and the trailing Physick field. No real save is modified.
