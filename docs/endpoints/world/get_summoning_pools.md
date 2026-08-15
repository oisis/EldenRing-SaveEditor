# GetSummoningPools

## Overview

`GetSummoningPools` returns every summoning pool resource GameCatalog declares,
together with its activation state for one character slot. A summoning pool is a
Martyr Effigy the character has activated. The stored catalog declares 213 pools,
covering the base game and Shadow of the Erdtree.

The getter reads only the private session snapshot. It never opens or writes a
save file and does not mutate the session, its revision, its dirty state or its
undo history.

| | |
|---|---|
| EndpointID | `get_summoning_pools` |
| Kind | Getter |
| Domain | `world` |
| Implementation status | implemented |
| Transport | `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/summoning-pools`, loopback explorer only |
| Save access | read-only session snapshot |

## Input

```go
func GetSummoningPools(
    engine *saveengine.Engine,
    gameCatalog *gamecatalog.Catalog,
    saveSessionID string,
    characterID int,
) (GetSummoningPoolsResult, error)
```

The public input is exactly `saveSessionID` and `characterID`. There is no
region filter: `regionLabel` is a presentation label, not a selector, and
filtering by it would imply a resource relation the catalog does not declare.
`saveSessionID` is matched exactly and `characterID` is the slot index `0..9`.

## Catalog contract

GameCatalog is the only source of public identity and meaning. A declaring
resource must have:

- kind `summoning_pool` and a key of lowercase letters, digits and underscores;
- no document of any other kind in the resource union;
- a known, non-empty `summoningPool.name`;
- a known, non-empty `summoningPool.regionLabel`;
- a known, non-zero `summoningPool.activationEventFlagID` inside the confirmed
  block `670000..670999`.

All of these are enforced by `schema.ValidateResource`, so a catalog that
violates them never loads. The endpoint additionally rejects two pools that
declare the same activation event flag, which no single document can rule out.
It contains no fixed pool list, and `GetSummoningPoolsResult` carries neither
the event flag identifier nor any save-layout detail. The complete catalog
document, including `activationEventFlagID` and its provenance, stays available
through the general [`GetResource`](../catalog/get_resource.md) getter, as its
contract defines.

### `regionLabel` is not a resource reference

`regionLabel` is the curated grouping label of the legacy pool table — for
example `Stormveil Castle` or `Land of Shadow`. It is a confirmed presentation
and grouping fact, and nothing more:

- it is not a `ResourceRef` and carries no kind or key;
- it does not resolve to a `RegionDocument` and creates no catalog relation;
- the two vocabularies are separate curated lists with different granularity, so
  joining them would invent evidence that does not exist.

The catalog `region` resources of [`GetRegions`](get_regions.md) remain a
completely independent surface.

### Data provenance

Each of the three facts carries its own provenance. All three come from the
curated legacy SaveForge data (`legacy_db_data`, manifest kind
`legacy_saveforge_data`), evidence level `curated`: the provenance names the
source table `SummoningPools`, the exact record, and the field the value was
read from (`map key`, `Name` or `Region`). The manifest version of that source
hashes the table together with the rest of the legacy snapshot.

The three arenas of the separate legacy `Colosseums` table share the same Go row
type but are a different curated list with their own resource kind. They are
deliberately not migrated as summoning pools and never appear in this result;
they stay owned by [`GetColosseums`](get_colosseums.md).

## Activation state

An entry is activated only when its declared activation event flag is set. Every
flag of the curated table lies in block `670`, whose position in the bitfield
(BST position `107`) was added to the existing `resolveEventFlag` of SaveEngine.
No second resolver, no second Event Flags locator and no `id/8` fallback was
introduced. The endpoint performs exactly one bulk `GetEventFlags` read for all
213 flags and decodes no bit itself.

An inactive or residual slot returns `active=false` and every entry deactivated.
Its event flag bitfield is not located, read or decoded.

## Result

```go
type SummoningPoolEntry struct {
    Kind        schema.ResourceKind
    Key         string
    Name        string
    RegionLabel string
    Activated   bool
}

type GetSummoningPoolsResult struct {
    SaveSessionID  string
    CharacterID    int
    Active         bool
    SummoningPools []SummoningPoolEntry
}
```

Entries are ordered by `regionLabel`, then `name`, then `key`, so the order is
deterministic even where a label and a name repeat. `summoningPools` is always a
non-null array.

## Verification

Endpoint tests cover the 213 stored declarations, one set and one adjacent clear
pool, the exclusion of the three colosseums, the inactive residual slot that
reports everything deactivated without reading its bitfield, an invalid engine,
catalog and session, deterministic ordering including the key tie-break, and
rejection of a duplicate activation flag. SaveEngine tests cover block `670` on
PC and PS4 including the boundaries `670000` and `670999` with their neighbours.
Catalog tests cover fail-closed rejection of a missing document, a foreign
document in the union, an unknown name, an empty region label, a zero flag and a
flag outside block `670`, plus the generic `GetResource`/`GetResources` surface.
The migrator test proves the 213 migrated rows, their block bounds, key
uniqueness, key determinism and the exclusion of the colosseum rows. The
transport test compares the HTTP response with the typed getter result.

`SetSummoningPoolActivated` remains contract-only and is deliberately not
exposed in OpenAPI or Scalar.
