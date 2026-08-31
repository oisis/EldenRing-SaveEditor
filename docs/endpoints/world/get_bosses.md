# GetBosses

## Overview

`GetBosses` returns every boss resource GameCatalog declares, together with its
defeat state for one character slot. The stored catalog declares 110 bosses,
covering the base game and Shadow of the Erdtree.

The getter reads only the private session snapshot. It never opens or writes a
save file and does not mutate the session, its revision, its dirty state or its
undo history.

| | |
|---|---|
| EndpointID | `get_bosses` |
| Kind | Getter |
| Domain | `world` |
| Implementation status | implemented |
| Transport | `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/bosses`, loopback explorer only |
| Save access | read-only session snapshot |

## Input

```go
func GetBosses(
    engine *saveengine.Engine,
    gameCatalog *gamecatalog.Catalog,
    saveSessionID string,
    characterID int,
) (GetBossesResult, error)
```

The public input is exactly `saveSessionID` and `characterID`. The earlier
contract draft declared `regionKind` and `regionKey`; that pair was removed
because the curated source declares a plain text region and no region resource
identity at all, so the parameters would have promised a relation nothing
supports. `saveSessionID` is matched exactly and `characterID` is the slot index
`0..9`.

## Scope of the curated table

The catalog holds only the bosses with a **confirmed synchronized defeat flag**.
The roughly 97 open-world field bosses — Night's Cavalry, Deathbirds, dragons,
Evergaols and the rest — record their defeat solely in a per-map flag that lies
far outside the event flag bitfield this reader addresses. They are absent from
the source, they are not guessed here, and no entry is ever synthesised for them.
A boss that is not in the table is simply not part of this endpoint's answer; it
is never reported as undefeated.

## Catalog contract

GameCatalog is the only source of public identity and meaning. A declaring
resource must have:

- kind `boss` and a key of lowercase letters, digits and underscores;
- no document of any other kind in the resource union;
- a known, non-empty `boss.name`;
- a known, non-empty `boss.regionLabel`;
- a known `boss.encounterType` out of the confirmed set `"main"` and `"field"`;
- a known `boss.remembrance`;
- a known, non-zero `boss.defeatEventFlagID` inside block `9`, the only block the
  curated table confirms.

All of these are enforced by `schema.ValidateResource`, so a catalog that
violates them never loads. The endpoint additionally rejects two bosses that
declare the same defeat event flag, which no single document can rule out. It
contains no fixed boss list, and `GetBossesResult` carries neither event flag
identifier nor any save-layout detail. The complete catalog document, including
`defeatEventFlagID` and its provenance, stays available through the general
[`GetResource`](../catalog/get_resource.md) getter, as its contract defines.

### `regionLabel` is not a resource reference

`regionLabel` is the curated grouping label the legacy row carries in its
`Region` field — for example `Stormveil Castle` or `Crumbling Farum Azula`. It is
a confirmed presentation and grouping fact, and nothing more:

- it is not a `ResourceRef` and carries no kind or key;
- it does not resolve to a `RegionDocument` and creates no catalog relation;
- the two vocabularies are separate curated lists with different granularity, so
  joining them would invent evidence that does not exist.

The catalog `region` resources of [`GetRegions`](get_regions.md) remain a
completely independent surface. This matches
[`GetGraces`](get_graces.md) and [`GetSummoningPools`](get_summoning_pools.md).

## Save contract

The defeat flags are resolved through the single existing
`saveengine.resolveEventFlag`. Block `9` was added there, mapped to BST position
`9`, which both SaveForge 1.5.8 and 1.6.10 confirm through `eventflag_bst.txt`.
No second resolver, no separate BST table, no `eventFlagID/8` fallback and no
cache were introduced. The neighbouring blocks `8` and `10` carry no curated
resource and stay rejected by name.

The endpoint resolves the complete set of identifiers from the catalog **before**
any slot is read and then performs exactly one bulk `GetEventFlags` call. It
never locates the `EventFlags` section itself and never decodes a bit itself.

An inactive or residual slot is reported from its activity flag alone: `active`
is `false`, every `defeated` is `false`, and not one byte of that slot's data is
read, so the residual bitfield of a deleted character can never leak a state.

## Result

```go
type GetBossesResult struct {
    SaveSessionID string      `json:"saveSessionID"`
    SaveRevision  string      `json:"saveRevision"`
    CharacterID   int         `json:"characterID"`
    Active        bool        `json:"active"`
    Bosses        []BossEntry `json:"bosses"`
}

type BossEntry struct {
    Kind          schema.ResourceKind `json:"kind"`
    Key           string              `json:"key"`
    Name          string              `json:"name"`
    RegionLabel   string              `json:"regionLabel"`
    EncounterType string              `json:"encounterType"`
    Remembrance   bool                `json:"remembrance"`
    Defeated      bool                `json:"defeated"`
}
```

`Bosses` is always a non-nil array, empty rather than `null`. Entries are ordered
by `regionLabel`, then `name`, then `key`, so paging and diffing are stable
across calls. `encounterType` is `main` or `field` and nothing else.

## Errors

| Condition | Behaviour |
|---|---|
| `engine` or `gameCatalog` is nil | error before any read |
| empty or unknown `saveSessionID` | error |
| `characterID` outside `0..9` | error |
| a `boss` resource without a boss document | error |
| two bosses declaring the same defeat event flag | error |
| an event flag outside block `9` | error from `resolveEventFlag` |
| an unreadable or truncated event flag section | error |

Every failure happens before a partial result is produced; there is no fallback
value and no partially populated answer.

## Related endpoints

- [`GetGraces`](get_graces.md) and [`GetSummoningPools`](get_summoning_pools.md)
  share the same catalog-plus-bulk-flag shape and the same `regionLabel` rule.
- `SetBossDefeated` stays a contract-only endpoint. It has no handler, no route,
  no OpenAPI operation and no Scalar page, and this document does not present it
  as available.

## Snapshot identity

The result includes `saveRevision`, the opaque revision of the exact session
snapshot used by this read. Clients compare it exactly with the current session
revision and discard a mismatch; they never parse, trim or order it.
