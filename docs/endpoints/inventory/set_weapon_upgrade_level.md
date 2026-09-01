# SetWeaponUpgradeLevel

## Overview

`SetWeaponUpgradeLevel` changes the target upgrade level of one existing weapon
instance addressed by an opaque `ownedItemID`. It preserves the weapon base and
its current affinity: a Heavy Dagger remains Heavy, and only the level suffix of
its save-side game ID changes.

The mutation changes the private in-memory snapshot only. `WriteSave` remains a
separate operation. A successful call advances `saveRevision` by one and makes
the echoed `ownedItemID` stale; re-read Inventory or Storage before addressing
the record again.

| | |
|---|---|
| EndpointID | `set_weapon_upgrade_level` |
| Kind | Mutation |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport | `PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/upgrade-level`, loopback explorer only |
| Save access | read-write on the session snapshot; no file is opened |

## Input

```go
func SetWeaponUpgradeLevel(
    engine *saveengine.Engine,
    gameCatalog *gamecatalog.Catalog,
    saveSessionID string,
    characterID int,
    ownedItemID string,
    upgradeLevel uint8,
    expectedRevision string,
) (SetWeaponUpgradeLevelResult, error)
```

The HTTP body is strict JSON:

```json
{
  "upgradeLevel": 10,
  "expectedRevision": "0"
}
```

Unknown fields and an omitted `upgradeLevel` are rejected. Level zero is valid.
Standard weapons accept at most their catalog maximum, normally `25`; somber
weapons accept at most their catalog maximum, normally `10`. Unknown, disabled,
missing or non-weapon upgrade rules fail closed. Values are never clamped.

## Processing

1. SaveEngine resolves `ownedItemID` to its exact Inventory or Storage common
   record and resolves the GaItem handle to the current weapon game ID.
2. GameCatalog finds the unique canonical or affinity anchor containing that
   exact ID, validates a known enabled `standard` or `somber` upgrade model,
   derives `anchor + upgradeLevel`, and computes the 0..25 matchmaking level.
3. SaveEngine repeats identity, revision and current-ID checks under its lock.
4. One atomic plan changes the four-byte ID in the existing GaItem record,
   inserts the target ID in GaItemData when absent, synchronises both stored
   item-ID representations of every matching equipped hand slot, and raises the
   durable matchmaking weapon level byte at `MagicOffset - 0xD5` if the target
   level exceeds the character's stored value (strictly monotonic).
5. The handle, physical row, quantity, acquisition index, affinity, old
   GaItemData entry and unrelated equipment remain unchanged. No GaItem record
   is allocated and no section is repacked.

Malformed or ambiguous GaItem data, a weapon equipped from outside Inventory
common, or inconsistent equipped row/ID representations rejects the complete
plan before its first write. Applied writes are verified and rolled back as one
unit on failure.

## Result

```go
type SetWeaponUpgradeLevelResult struct {
    MutationReceipt
    OwnedItemID    string
    CharacterID    int
    Container      string
    PreviousGameID uint32
    GameID         uint32
    UpgradeLevel   uint8
}

type MutationReceipt struct {
    OperationID   string
    OperationKind string
    SaveSessionID string
    SaveRevision  string
    ChangedScopes []string
}
```

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
  `set_weapon_upgrade_level`.
- `changedScopes` are exactly `save.session`, `inventory`, `storage`,
  `equipment.loadout`, `diagnostics.report`, in that canonical order.

This endpoint commits through the SaveEngine writer `setOwnedWeaponGameID`,
which it shares with exactly one other endpoint,
[`SetWeaponInfusion`](set_weapon_infusion.md). The writer receives its kind from
the public entry point, so the receipt of this endpoint always reports
`set_weapon_upgrade_level` and never `set_weapon_infusion`.
[`SetWeaponAshOfWar`](set_weapon_ash_of_war.md) does not use that writer at
all.

`Container` is `inventory` or `storage`. `GameID` is the exact upgraded ID now
stored; `PreviousGameID` records the value replaced. The returned
`OwnedItemID` describes the completed operation and is already stale under the
new revision.

## Verification

Focused tests cover standard and somber catalog limits, affinity-anchor
preservation, PC and PS4 save layouts, equipped-representation synchronisation,
GaItemData insertion, ambiguous GaItem rejection, atomic failure and
write/reload/read-back semantics.
