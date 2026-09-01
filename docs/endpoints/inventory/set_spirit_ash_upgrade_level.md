# SetSpiritAshUpgradeLevel

## Overview

`SetSpiritAshUpgradeLevel` changes the target upgrade level of one existing
Spirit Ash addressed by an opaque `ownedItemID`. GameCatalog selects the exact
stored grave- or ghost-glovewort variant; the endpoint never derives an item ID
by arithmetic and never changes the Spirit Ash identity.

The mutation changes the private in-memory snapshot only. `WriteSave` remains a
separate operation. A successful call advances `saveRevision` by one and makes
the echoed `ownedItemID` stale; re-read Inventory or Storage before addressing
the record again.

| | |
|---|---|
| EndpointID | `set_spirit_ash_upgrade_level` |
| Kind | Mutation |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport | `PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}/spirit-ash-upgrade-level`, loopback explorer only |
| Save access | read-write on the session snapshot; no file is opened |

## Input

```go
func SetSpiritAshUpgradeLevel(
    engine *saveengine.Engine,
    gameCatalog *gamecatalog.Catalog,
    saveSessionID string,
    characterID int,
    ownedItemID string,
    upgradeLevel uint8,
    expectedRevision string,
) (SetSpiritAshUpgradeLevelResult, error)
```

The HTTP body is strict JSON:

```json
{
  "upgradeLevel": 10,
  "expectedRevision": "0"
}
```

Unknown fields and an omitted `upgradeLevel` are rejected. Level zero is valid.
The requested level must not exceed the catalog maximum, currently `10` for the
supported grave- and ghost-glovewort models. Unknown, disabled, incomplete or
non-Spirit-Ash upgrade data fails closed. Values are never clamped.

## Processing

1. SaveEngine resolves `ownedItemID` to its exact Inventory or Storage common
   record and resolves its record-free goods handle to the current game ID.
2. GameCatalog validates the `spirit_ash` family, the known enabled grave- or
   ghost-glovewort capability and the requested limit, then selects exactly one
   stored upgrade variant.
3. SaveEngine repeats identity, revision, quantity and current-handle checks
   under its lock.
4. One atomic plan changes the four-byte handle in the existing record, inserts
   the target ID in GaItemData when absent and, for an Inventory record, updates
   every Quick Items or Pouch position that points to that physical row.
5. Quantity, acquisition index, physical row, the old GaItemData entry and
   unrelated loadout state remain unchanged. No GaItem record is allocated and
   no section is repacked.

An inconsistent Quick Items or Pouch representation rejects the complete plan
before its first write. Applied writes are verified and rolled back as one unit
on failure. Storage records have no active loadout references and change only
their handle and GaItemData.

## Result

```go
type SetSpiritAshUpgradeLevelResult struct {
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
  `set_spirit_ash_upgrade_level`.
- `changedScopes` are exactly `save.session`, `inventory`, `storage`,
  `equipment.loadout`, `diagnostics.report`, in that canonical order.

`Container` is `inventory` or `storage`. `GameID` is the exact stored variant
now referenced; `PreviousGameID` records the replaced value. The returned
`OwnedItemID` describes the completed operation and is already stale under the
new revision.

## Verification

Focused tests cover both glovewort models and their level limit, PC and PS4 slot
layouts, Inventory reference synchronisation, GaItemData insertion, atomic
failure on an inconsistent reference, transport validation and write/reload
read-back.
