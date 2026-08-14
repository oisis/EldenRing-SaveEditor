# GetBellBearings

## Overview

`GetBellBearings` returns every goods resource that GameCatalog declares through
exactly one `item.unlocks` entry of kind `bell_bearing`, together with its
acquisition state for one character slot. The stored catalog currently declares
62 Bell Bearings.

The getter reads only the private session snapshot. It never opens or writes a
save file and does not mutate the session or its revision.

| | |
|---|---|
| EndpointID | `get_bell_bearings` |
| Kind | Getter |
| Domain | `world` |
| Implementation status | implemented |
| Transport | `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/bell-bearings`, loopback explorer only |
| Save access | read-only session snapshot |

## Input

```go
func GetBellBearings(
    engine *saveengine.Engine,
    gameCatalog *gamecatalog.Catalog,
    saveSessionID string,
    characterID int,
    availabilityFilter string,
) (GetBellBearingsResult, error)
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
- exactly one unlock of known kind `bell_bearing`;
- a known event flag ID in block `11109`;
- a non-empty known name and category in that unlock.

Duplicate event flags are rejected. The endpoint contains no fixed Bell Bearing
list and does not expose the physical flag identifier.

## Unlock state

An entry is unlocked only when its acquisition event flag is set. This matches
SaveForge 1.5.8 and 1.6.8, where the flag represents a Bell Bearing handed to
the Twin Maiden Husks. A matching item still present in Inventory or Storage is
the pre-handover state and is not treated as unlocked.

All current Bell Bearing flags belong to block `11109`, whose confirmed BST
position is `11129` in SaveForge 1.5.8, 1.6.8 and the local legacy reference.
Unknown blocks still fail explicitly; there is no generic BST table or fallback.

An inactive or residual slot returns `active=false` and every entry locked. Its
event flag bitfield is not read.

## Result

```go
type BellBearingEntry struct {
    Kind     schema.ResourceKind
    Key      string
    Name     string
    Category string
    Unlocked bool
}

type GetBellBearingsResult struct {
    SaveSessionID string
    CharacterID   int
    Active        bool
    BellBearings  []BellBearingEntry
}
```

Entries are ordered by category, name and key. Filtering never changes their
relative order. `bellBearings` is always a non-null array.

## Verification

SaveEngine tests cover the block `11109` mapping on PC and PS4 as well as its
first and last flags. Endpoint tests cover all 62 stored declarations,
deterministic order, filtering, inactive state and rejection of a foreign event
flag block. The transport test compares the HTTP response with the typed getter
result.
