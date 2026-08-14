# SetWeaponInfusion

## Overview

`SetWeaponInfusion` changes the affinity of one existing weapon instance
addressed by an opaque `ownedItemID`. It preserves the current upgrade level:
a Dagger +5 changed from Standard to Heavy becomes Heavy Dagger +5.

The mutation changes the private in-memory snapshot only. `WriteSave` remains a
separate operation. A successful call advances `saveRevision` by one and makes
the echoed `ownedItemID` stale; re-read Inventory or Storage before addressing
the record again.

| | |
|---|---|
| EndpointID | `set_weapon_infusion` |
| Kind | Mutation |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport | `PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/infusion`, loopback explorer only |
| Save access | read-write on the session snapshot; no file is opened |

## Input

```go
func SetWeaponInfusion(
    engine *saveengine.Engine,
    gameCatalog *gamecatalog.Catalog,
    saveSessionID string,
    characterID int,
    ownedItemID string,
    affinity schema.Affinity,
    expectedRevision string,
) (SetWeaponInfusionResult, error)
```

The HTTP body is strict JSON:

```json
{
  "affinity": "heavy",
  "expectedRevision": "0"
}
```

The accepted vocabulary is `standard`, `heavy`, `keen`, `quality`, `fire`,
`flame_art`, `lightning`, `sacred`, `magic`, `cold`, `poison`, `blood` and
`occult`. The resolved weapon must declare a known enabled infusion capability,
the requested value in `allowedAffinities`, and exactly one stored affinity
anchor when the request is not `standard`. Unknown fields and omitted values
are rejected; values are never normalised or guessed.

## Processing

1. SaveEngine resolves `ownedItemID` to its exact Inventory or Storage common
   record and resolves the GaItem handle to the current weapon game ID.
2. GameCatalog finds the unique canonical or affinity anchor containing that
   ID, reads the current upgrade level, validates the requested affinity and
   derives `targetAffinityAnchor + currentUpgradeLevel`.
3. SaveEngine repeats identity, revision and current-ID checks under its lock.
4. One atomic plan changes the four-byte ID in the existing GaItem record,
   inserts the target ID in GaItemData when absent, and synchronises both stored
   item-ID representations of every matching equipped hand slot.
5. The handle, physical row, quantity, acquisition index, upgrade level,
   mounted Ash of War, old GaItemData entry and unrelated equipment remain
   unchanged. No GaItem record is allocated and no section is repacked.

Malformed or ambiguous GaItem data, a weapon equipped from outside Inventory
common, inconsistent equipped row/ID representations, or missing catalog
evidence rejects the complete plan before its first write. Applied writes are
verified and rolled back as one unit on failure.

The mounted Ash of War is deliberately not used to derive affinity. SaveForge
1.5.8 and 1.6.8 treated these edits independently, and GameCatalog currently
contains no confirmed relation that maps one mounted Ash of War to the subset of
affinities it permits.

## Result

```go
type SetWeaponInfusionResult struct {
    SaveSessionID    string
    SaveRevision     string
    OwnedItemID      string
    CharacterID      int
    Container        string
    PreviousGameID   uint32
    GameID           uint32
    Affinity         schema.Affinity
    UpgradeLevel     uint8
}
```

`Container` is `inventory` or `storage`. `GameID` is the exact infused ID now
stored; `PreviousGameID` records the value replaced. The returned `OwnedItemID`
describes the completed operation and is already stale under the new revision.

## Verification

Focused tests cover Standard-to-Heavy and Heavy-to-Standard target derivation,
upgrade-level preservation, disabled or unsupported infusion rejection, the
shared atomic weapon-ID writer and the strict HTTP route. The existing shared
writer tests cover PC and PS4, Inventory and Storage common, equipped-ID
synchronisation, GaItemData insertion, ambiguity rejection and reload.
