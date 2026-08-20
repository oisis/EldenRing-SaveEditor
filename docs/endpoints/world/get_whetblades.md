# GetWhetblades

## Overview

`GetWhetblades` returns every goods resource that GameCatalog declares through
exactly one `item.unlocks` entry of kind `whetblade`, together with its unlock
state for one character slot. The stored catalog currently declares six:
Whetstone Knife and the five Whetblades.

The getter reads only the private session snapshot. It never opens or writes a
save file and does not mutate the session or its revision.

| | |
|---|---|
| EndpointID | `get_whetblades` |
| Kind | Getter |
| Domain | `world` |
| Implementation status | implemented |
| Transport | `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/whetblades`, loopback explorer only |
| Save access | read-only session snapshot |

## Input

```go
func GetWhetblades(
    engine *saveengine.Engine,
    gameCatalog *gamecatalog.Catalog,
    saveSessionID string,
    characterID int,
    availabilityFilter string,
) (GetWhetbladesResult, error)
```

`availabilityFilter` is matched exactly:

- empty returns every entry;
- `unlocked` returns unlocked entries;
- `locked` returns locked entries.

Every other value is rejected without trimming, case folding or aliases.

## Catalog contract

GameCatalog is the only source of public identity and meaning. A declaring
resource must have:

- kind `item` and a non-empty key;
- known family `goods`;
- one known goods game ID with prefix `0x4`;
- exactly one unlock of known kind `whetblade`;
- a known event flag ID in block `60` or `65` and a non-empty name in that
  unlock.

Duplicate game IDs or event flags across two Whetblades are rejected. The
endpoint contains no fixed item list and exposes neither identifier.

## Unlock state

An entry is unlocked when either:

1. its main event flag is set; or
2. a positive-quantity record for its goods game ID exists in common or key
   `InventoryHeld`.

This is the state rule shared by SaveForge 1.5.8 and 1.6.10. Their getter used a
fixed five-entry table. SaveForge 2.0 instead follows the six declarations in
GameCatalog, which also includes Whetstone Knife.

SaveEngine recognises the two confirmed goods representations: a direct `0x4`
game ID used by game-placed key items and the handle-encoded `0xB` form. Storage
is not consulted.

The Event Flags reader supports only confirmed blocks. This endpoint adds the
two mappings required by its catalog data: block `60` at position `10` and
block `65` at position `15`. Cookbook mappings `67 → 17` and `68 → 18` remain
unchanged. Unknown blocks still fail explicitly; there is no `eventFlagID/8`
fallback or general BST table.

An inactive or residual slot returns `active=false` and every entry locked.
Neither its InventoryHeld records nor its event flag bitfield is read.

## Result

```go
type WhetbladeEntry struct {
    Kind     schema.ResourceKind
    Key      string
    Name     string
    Unlocked bool
}

type GetWhetbladesResult struct {
    SaveSessionID string
    CharacterID   int
    Active        bool
    Whetblades    []WhetbladeEntry
}
```

Entries are ordered by name and then key. Filtering never changes their
relative order. `whetblades` is always a non-null array.

## Verification

SaveEngine tests cover PC and PS4 event-flag mappings, direct and encoded goods
records, zero quantities and residual slots. Endpoint tests cover all six
stored declarations, both unlock signals, filtering and inactive state. The
transport test compares the HTTP response with the typed getter result.
