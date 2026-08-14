# SetWeaponAshOfWar

## Overview

`SetWeaponAshOfWar` mounts an existing free Ash of War copy on one owned
weapon, changes the mounted copy, or removes the custom Ash. It never creates a
copy: when the requested Ash has no unique free GaItem, the request fails.

The mutation changes the private session snapshot only. `WriteSave` remains a
separate operation. Success advances `saveRevision` once and makes the supplied
`weaponOwnedItemID` stale.

| | |
|---|---|
| EndpointID | `set_weapon_ash_of_war` |
| Kind | Mutation |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport | `PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/ash-of-war`, loopback explorer only |
| Save access | read-write on the session snapshot; no file is opened |

## Input

```go
func SetWeaponAshOfWar(
    engine *saveengine.Engine,
    gameCatalog *gamecatalog.Catalog,
    saveSessionID string,
    characterID int,
    weaponOwnedItemID string,
    ashOfWarKind *string,
    ashOfWarKey *string,
    expectedRevision string,
) (SetWeaponAshOfWarResult, error)
```

Mount or change:

```json
{
  "ashOfWarKind": "item",
  "ashOfWarKey": "8000EA60",
  "expectedRevision": "0"
}
```

Remove:

```json
{
  "ashOfWarKind": null,
  "ashOfWarKey": null,
  "expectedRevision": "0"
}
```

Both selector fields are required. They must both be strings or both be null;
omitting either field or mixing a string with null is rejected. Strings are
matched exactly and are never trimmed or normalised.

The owned item must be a positive-quantity weapon record in the common section
of Inventory or Storage.

For mounting, the weapon must declare a known, enabled `ashOfWarMount`
capability with mode `custom`. The selected resource must be a known
`ash_of_war` item with a valid `0x8...` game ID and known compatibility data.
GameCatalog must contain the exact `compatible_with_aow` relation from the
current weapon resource to the selected Ash resource. Removal needs no
compatibility or mount-capability check, matching the confirmed 1.5.8 and 1.6.8
clear contract.

## Existing-copy rule

The endpoint uses only a physical Ash of War GaItem already present in the
character slot. SaveEngine scans GaItems in physical order and selects the
first copy that:

- has the requested game ID;
- has exactly one GaItem record for its handle;
- is not referenced by another weapon.

If the weapon already carries the requested Ash through one valid, uniquely
owned handle, that same handle is retained. Otherwise, absence of a free copy
is a hard error. The endpoint does not allocate a GaItem, write GaItemData,
repack the table, or restore any retired allocator.

## Binary mutation

A weapon GaItem stores its custom Ash handle four bytes into the final part of
its 21-byte record. A successful change writes only those four bytes at
`weaponRecord + 16`.

Both `0x00000000` and historical `0xFFFFFFFF` are read as no custom Ash.
Removal always writes the native canonical value `0x00000000`. The weapon game
ID, affinity, upgrade level, handle, owned row, quantity, acquisition index,
equipment data, GaItemData and the Ash records themselves remain unchanged.

Before writing, SaveEngine repeats the revision and identity checks under its
lock and requires:

- exactly one GaItem record for the owned weapon handle;
- a weapon-shaped game ID and a valid current Ash reference;
- exactly one GaItem record for the currently attached Ash, when present;
- no sharing of the current or selected Ash handle between weapons;
- a unique free target copy when the requested Ash differs from the current
  one.

Malformed references and unsupported allocation fail before mutation. The
four-byte write is read back and verified; a failed write or verification is
rolled back without advancing the revision.

The in-place contract and zero removal sentinel are identical in SaveForge
1.5.8 and 1.6.8. Their separate path that allocated a fresh Ash copy depended
on the retired GaItem repacker and is deliberately not part of this endpoint.

## Result

```go
type SetWeaponAshOfWarResult struct {
    SaveSessionID          string
    SaveRevision           string
    WeaponOwnedItemID      string
    CharacterID            int
    Container              string
    WeaponGameID           uint32
    PreviousAshOfWarGameID uint32
    AshOfWarGameID         uint32
}
```

`Container` is `inventory` or `storage`. A zero Ash game ID means that side of
the change had no custom Ash. No raw GaItem handle is exposed. The returned
`weaponOwnedItemID` describes the completed operation but is stale under the
new revision; re-read Inventory or Storage before another owned-item mutation.

## Verification

SaveEngine tests cover attach, change, removal, canonical sentinel output,
shared-handle rejection, missing free-copy rejection, PC and PS4 layouts, exact
byte scope, `WriteSave` and reload. Endpoint and transport tests cover catalog
compatibility, explicit nullable selectors, dependency failures and the public
HTTP contract.
