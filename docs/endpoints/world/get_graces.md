# GetGraces

## Overview

`GetGraces` returns every grace resource GameCatalog declares, together with its
visit state for one character slot. A grace is a Site of Grace the character has
rested at. The stored catalog declares 419 graces, covering the base game and
Shadow of the Erdtree.

The getter reads only the private session snapshot. It never opens or writes a
save file and does not mutate the session, its revision, its dirty state or its
undo history.

| | |
|---|---|
| EndpointID | `get_graces` |
| Kind | Getter |
| Domain | `world` |
| Implementation status | implemented |
| Transport | `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/graces`, loopback explorer only |
| Save access | read-only session snapshot |

## Input

```go
func GetGraces(
    engine *saveengine.Engine,
    gameCatalog *gamecatalog.Catalog,
    saveSessionID string,
    characterID int,
) (GetGracesResult, error)
```

The public input is exactly `saveSessionID` and `characterID`. There is no
region filter: `regionLabel` is a presentation label, not a selector, and
filtering by it would imply a resource relation the catalog does not declare.
`saveSessionID` is matched exactly and `characterID` is the slot index `0..9`.

## Catalog contract

GameCatalog is the only source of public identity and meaning. A declaring
resource must have:

- kind `grace` and a key of lowercase letters, digits and underscores;
- no document of any other kind in the resource union;
- a known, non-empty `grace.name`;
- a known, non-empty `grace.regionLabel`;
- a known, non-zero `grace.visitEventFlagID` inside a block the curated table
  confirms (`71`, `72`, `73`, `74` or `76`);
- a known `grace.bossArena`;
- a known `grace.dungeonType` out of the confirmed set `""`, `"catacomb"` and
  `"hero_grave"`;
- a known `grace.doorEventFlagID`, which may only be non-zero when a dungeon type
  is set.

All of these are enforced by `schema.ValidateResource`, so a catalog that
violates them never loads. The endpoint additionally rejects two graces that
declare the same visit event flag, which no single document can rule out. It
contains no fixed grace list, and `GetGracesResult` carries neither event flag
identifier nor any save-layout detail. The complete catalog document, including
`visitEventFlagID`, `doorEventFlagID` and their provenance, stays available
through the general [`GetResource`](../catalog/get_resource.md) getter, as its
contract defines.

### `regionLabel` is not a resource reference

`regionLabel` is the curated grouping label the legacy grace name carries in its
trailing parenthesis — for example `Stormveil Castle` or `Mountaintops of the
Giants West`. It is a confirmed presentation and grouping fact, and nothing more:

- it is not a `ResourceRef` and carries no kind or key;
- it does not resolve to a `RegionDocument` and creates no catalog relation;
- the two vocabularies are separate curated lists with different granularity, so
  joining them would invent evidence that does not exist.

The catalog `region` resources of [`GetRegions`](get_regions.md) remain a
completely independent surface. The legacy 1.x rewrites that renamed grace
regions and reassigned Roundtable Hold to another region were deliberately not
carried over: the label stays exactly the one the curated record declares.

### `doorEventFlagID` is private

`doorEventFlagID` is the overworld ObjAct flag that opens the sealed entrance of
a catacomb or hero's grave. It belongs to the same confirmed record, so the
catalog document stores it, but it is a save-format detail and no field of
`GraceEntry` exposes it. Zero is its confirmed value for a grace with no
dependent door, which includes every grace without a dungeon type and the four
dungeon graces the curated table records without one.

### Data provenance

Each of the six facts carries its own provenance. All of them come from the
curated legacy SaveForge data (`legacy_db_data`, manifest kind
`legacy_saveforge_data`), evidence level `curated`: the provenance names the
source table `Graces`, the exact record, and the field the value was read from
(`map key`, `Name`, `BossArena`, `DungeonType` or `DoorFlag`). `name` and
`regionLabel` share the source field `Name` and are separated by the generator:
the trailing parenthesis is the region label and everything in front of it is the
name. Every one of the 419 curated names declares that suffix, so the split needs
no fallback and a row without it is rejected instead of guessed.

### Version difference between SaveForge 1.5.8 and 1.6.10

The two legacy versions disagree about exactly one record: `Castle Sol Main Gate`
(flag `76522`) is a boss-arena grace in 1.5.8 and a regular grace in 1.6.10. The
1.6.10 value is the migrated one, because 1.6.10 introduced the change together
with a dedicated regression test recording why: the boss of Castle Sol sits at
`Castle Sol Rooftop` (flag `76524`), which stays a boss-arena grace in both
versions. The newer value was not preferred for being newer.

## Visit state

An entry is visited only when its declared visit event flag is set. The flags of
the curated table lie in the blocks `71`, `72`, `73`, `74` and `76`, whose
positions in the bitfield (BST positions `21`, `22`, `23`, `24` and `26`) were
added to the existing `resolveEventFlag` of SaveEngine. Block `75` has a position
of its own but carries no grace of the curated table, so it was deliberately not
added and stays rejected. No second resolver, no second Event Flags locator and
no `id/8` fallback was introduced. The endpoint performs exactly one bulk
`GetEventFlags` read for all 419 flags and decodes no bit itself.

An inactive or residual slot returns `active=false` and every entry unvisited.
Its event flag bitfield is not located, read or decoded.

## Result

```go
type GraceEntry struct {
    Kind        schema.ResourceKind
    Key         string
    Name        string
    RegionLabel string
    BossArena   bool
    DungeonType string
    Visited     bool
}

type GetGracesResult struct {
    SaveSessionID string
    CharacterID   int
    Active        bool
    Graces        []GraceEntry
}
```

Entries are ordered by `regionLabel`, then `name`, then `key`, so the order is
deterministic even where a label and a name repeat. `graces` is always a non-null
array.

## Verification

Endpoint tests cover the 419 stored declarations, one set and one adjacent clear
grace, the presence of all three dungeon types and the 50 boss-arena graces, the
absence of raw flags and of the source name formatting, the inactive residual
slot that reports everything unvisited without reading its bitfield, an invalid
engine, catalog and session, deterministic ordering including the key tie-break,
and rejection of a duplicate visit flag. SaveEngine tests cover the five grace
blocks on PC and PS4 with a set flag and its clear neighbour each, and the
continued rejection of block `75`. Catalog tests cover fail-closed rejection of a
missing document, a foreign document in the union, an unknown name, an empty
region label, a zero flag, a flag in the unused block `75`, a flag outside every
grace block, an unknown boss-arena fact, an unknown and an unsupported dungeon
type, an unknown door flag and a door flag without a dungeon type, plus the
generic `GetResource`/`GetResources` surface. The authored catalog and endpoint
tests prove the 419 stored rows, their block bounds, flag and key uniqueness,
deterministic ordering, the name and region-label split including the one curated
name that carries its own separator, and the resolved Castle Sol boss-arena
difference. The transport test compares the HTTP response with the typed getter
result.

`SetGraceVisited` remains contract-only and is deliberately not exposed in
OpenAPI or Scalar.
