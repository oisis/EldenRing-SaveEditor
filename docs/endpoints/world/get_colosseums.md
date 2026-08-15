# GetColosseums

## Overview

`GetColosseums` returns every colosseum resource GameCatalog declares, together
with its unlock state for one character slot. The stored catalog declares the
three base-game arenas: Caelid Colosseum, Limgrave Colosseum and Royal
Colosseum.

The getter reads only the private session snapshot. It never opens or writes a
save file and does not mutate the session or its revision.

| | |
|---|---|
| EndpointID | `get_colosseums` |
| Kind | Getter |
| Domain | `world` |
| Implementation status | implemented |
| Transport | `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/colosseums`, loopback explorer only |
| Save access | read-only session snapshot |

## Input

```go
func GetColosseums(
    engine *saveengine.Engine,
    gameCatalog *gamecatalog.Catalog,
    saveSessionID string,
    characterID int,
) (GetColosseumsResult, error)
```

There is no availability filter. `saveSessionID` is matched exactly and
`characterID` is the slot index `0..9`.

## Catalog contract

GameCatalog is the only source of public identity and meaning. A declaring
resource must have:

- kind `colosseum` and a non-empty key;
- no item document;
- a known, non-empty `colosseum.name`;
- a known, non-zero `colosseum.unlockEventFlagID`.

The first four rules are enforced by `schema.ValidateResource`, so a catalog
that violates them never loads. The endpoint additionally rejects two
colosseums that declare the same event flag, which no single document can rule
out. It contains no fixed colosseum list, and `GetColosseumsResult` carries
neither the event flag identifier nor any save-layout detail. The complete
catalog document, including `unlockEventFlagID` and its provenance, stays
available through the general
[`GetResource`](../catalog/get_resource.md) getter, as its contract defines.

### Data provenance

Each fact carries its own provenance because the two come from different
sources:

- `name` — the official English `PlaceName` FMG of `item.msgbnd`
  (entries 450200, 450100 and 450000), evidence level `game_data`;
- `unlockEventFlagID` — the curated legacy SaveForge data
  (`legacy_db_data`, manifest kind `legacy_saveforge_data`), evidence level
  `curated`. The provenance names the source table `Colosseums` and the exact
  record the flag was read from, and the manifest version of that source hashes
  that table together with the rest of the legacy snapshot.

The activation flags are **not** regulation data and are not presented as such.
Only the Limgrave flag (`60360`) was confirmed against a native save; the Caelid
and Royal flags follow the stride verified alongside it and remain curated
research, identical in SaveForge 1.5.8 and 1.6.8.

## Unlock state

An entry is unlocked only when its declared activation event flag is set. All
three flags lie in block `60`, which `SaveEngine.GetEventFlags` already
supports; the endpoint performs exactly one bulk read for all of them and
decodes no bit itself.

An inactive or residual slot returns `active=false` and every entry locked. Its
event flag bitfield is not read.

## Result

```go
type ColosseumEntry struct {
    Kind     schema.ResourceKind
    Key      string
    Name     string
    Unlocked bool
}

type GetColosseumsResult struct {
    SaveSessionID string
    CharacterID   int
    Active        bool
    Colosseums    []ColosseumEntry
}
```

Entries are ordered by name and then key. `colosseums` is always a non-null
array.

## Verification

Endpoint tests cover the three stored declarations with their names and event
flags, the locked and unlocked states, the inactive slot, an invalid session and
characterID, deterministic ordering including the key tie-break, rejection of a
duplicate event flag, and delegation of flag placement to SaveEngine. Catalog
tests cover fail-closed rejection of a missing name, a missing event flag, a
missing document and a contradictory resource union. The transport test compares
the HTTP response with the typed getter result.

The matching writer is [`SetColosseumUnlocked`](set_colosseum_unlocked.md). It
shares this resolver, so the reported state and the written activation flag can
never drift apart.
