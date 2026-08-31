# GetMapRegions

## Overview

`GetMapRegions` returns the 263 safe map visibility regions declared by
GameCatalog together with their current visibility for one character slot. The
catalog is generated only from the curated legacy `MapVisible` table.

The getter reads the private session snapshot. It never opens or writes a save
file and does not change the session revision, dirty state or undo history.

| | |
|---|---|
| EndpointID | `get_map_regions` |
| Kind | Getter |
| Domain | `world` |
| Implementation status | implemented |
| Transport | `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/map-regions`, loopback explorer only |
| Save access | read-only session snapshot |

## Input

```go
func GetMapRegions(
    engine *saveengine.Engine,
    gameCatalog *gamecatalog.Catalog,
    saveSessionID string,
    characterID int,
) (GetMapRegionsResult, error)
```

The input is exactly `saveSessionID` and `characterID`. The earlier contract
draft also declared `parentRegionKind` and `parentRegionKey`. They were removed
because the source provides only a plain-text area label and declares no
relation to a `RegionDocument`.

## Catalog contract

One public entry is generated for each row of `MapVisible`. A valid
`MapRegionDocument` has:

- kind `map_region` and a lowercase slug key;
- no other document in the resource union;
- a known, non-empty `name`;
- a known, non-empty `areaLabel`;
- a known, non-zero `visibleEventFlagID` in block `62`.

The endpoint validates the complete catalog projection before reading the save
and rejects two resources that claim the same visibility flag. Raw flag IDs and
their provenance remain available through
[`GetResource`](../catalog/get_resource.md), but are not returned by this
getter.

`areaLabel` is only a curated presentation and grouping label. It is not a
`ResourceRef`, does not resolve to a `RegionDocument` and creates no catalog
relation.

## Excluded legacy map tables

The other legacy map tables have different meanings and are deliberately not
projected as map regions:

- `MapSystem` contains global map display switches, not regions;
- `MapUnsafe` contains sub-region flags that can produce black map tiles when
  set outside the game's discovery flow;
- `MapAcquired` contains transient map-fragment pickup notification triggers.
  The game clears them after displaying the notification. They do not record
  visibility, exploration or item ownership.

The existing item-side `MapFragmentMetadata` remains the source of map-fragment
item metadata. `GetMapRegions` does not duplicate or infer that relationship.

## Save contract

Visibility flags are resolved by the existing `saveengine.resolveEventFlag`.
Block `62` maps to BST position `12`, as declared by the event-flag tables used
by both SaveForge 1.5.8 and 1.6.10. Blocks `63` and `82` remain unsupported by
this reader contract.

The endpoint resolves every catalog flag before touching the slot and then
performs one bulk `GetEventFlags` call. It contains no bitfield decoder and no
fallback based on `eventFlagID/8`.

For an inactive or residual slot, `active` is `false` and every entry has
`visible=false`. SaveEngine determines that from the activity flag without
reading the residual slot contents.

## Result

```go
type GetMapRegionsResult struct {
    SaveSessionID string           `json:"saveSessionID"`
    SaveRevision  string           `json:"saveRevision"`
    CharacterID   int              `json:"characterID"`
    Active        bool             `json:"active"`
    MapRegions    []MapRegionEntry `json:"mapRegions"`
}

type MapRegionEntry struct {
    Kind      schema.ResourceKind `json:"kind"`
    Key       string              `json:"key"`
    Name      string              `json:"name"`
    AreaLabel string              `json:"areaLabel"`
    Visible   bool                `json:"visible"`
}
```

`mapRegions` is non-nil and ordered by `areaLabel`, then `name`, then `key`.

## Errors

| Condition | Behaviour |
|---|---|
| `engine` or `gameCatalog` is nil | error before any save read |
| unknown `saveSessionID` | error |
| `characterID` outside `0..9` | error |
| incomplete map region document | catalog validation error |
| duplicate visibility event flag | endpoint error before the save read |
| event flag outside supported block `62` | SaveEngine error |
| unreadable or truncated event flag section | SaveEngine error |

No failure produces a partial result or substitutes a guessed state.

## Related endpoints

- [`GetRegions`](get_regions.md) reads the separate `UnlockedRegions` list and
  returns invasion and blue-summon regions. It is not a parent catalog for this
  endpoint.
- [`SetMapRegionRevealed`](set_map_region_revealed.md) writes the same curated
  visibility state and keeps the catalog-linked Map Fragment in step.
- [`SetFogOfWarRemoved`](set_fog_of_war_removed.md) removes the separate global
  Fog of War overlay. It names no map region and changes no visibility flag.

## Snapshot identity

The result includes `saveRevision`, the opaque revision of the exact session
snapshot used by this read. Clients compare it exactly with the current session
revision and discard a mismatch; they never parse, trim or order it.
