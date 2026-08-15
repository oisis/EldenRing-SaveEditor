# GetRegions

## Overview

`GetRegions` returns the 274 curated invasion and blue-summon regions declared
by GameCatalog together with their state in one character's `UnlockedRegions`
list. It does not expose raw region IDs or unknown raw entries found in the
save.

| | |
|---|---|
| EndpointID | `get_regions` |
| Kind | Getter |
| Domain | `world` |
| Implementation status | implemented |
| Transport | `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/regions`, loopback explorer only |
| Save access | read-only session snapshot |

## Input

    func GetRegions(
        engine *saveengine.Engine,
        gameCatalog *gamecatalog.Catalog,
        saveSessionID string,
        characterID int,
    ) (GetRegionsResult, error)

`saveSessionID` is matched exactly. `characterID` is the physical slot index
`0..9`.

## Catalog contract

Each public entry comes from a `RegionDocument` with:

- a stable `(kind, key)` identity where kind is `region`;
- a known, non-zero internal region ID;
- a known, non-empty name;
- a known, non-empty area;
- provenance for all three facts.

The stored data is the 274-entry curated allowlist used by SaveForge 1.5.8 and
1.6.8, not the complete `PlayRegionParam` table. It contains 208 base-game and
66 Shadow of the Erdtree invasion or blue-summon contexts. Internal sub-areas
and network-only rows outside that allowlist are not presented as regions.

The source is the archived `backend/db/data/regions.go` table, recorded as
`legacy_db_data` with curated evidence. The two reference versions contain the
same table and use the same membership rule.

## Save contract

`UnlockedRegions` is not an Event Flags bitfield. It is a dynamic section:

    count: uint32
    regionIDs: count * uint32

The section starts immediately after the fixed `GestureGameData` block. The
reader locates that block through the confirmed per-slot anchor and the declared
projectile count, then validates the region count and the complete table against
the slot and snapshot bounds. PC and PS4 use the same section layout and differ
only in their container slot bases.

The reader preserves raw order, duplicates, zero and unknown IDs. The endpoint
builds a membership set and joins only IDs declared by GameCatalog. Unknown raw
IDs are ignored without changing the save.

An inactive or residual slot returns `active=false`, the complete catalog list
with every entry locked, and does not inspect its slot data.

## Result

    type RegionEntry struct {
        Kind     schema.ResourceKind
        Key      string
        Name     string
        Area     string
        Unlocked bool
    }

    type GetRegionsResult struct {
        SaveSessionID string
        CharacterID   int
        Active        bool
        Regions       []RegionEntry
    }

Entries are ordered by area, then name, then key. `regions` is always a
non-null array.

## Verification

SaveEngine tests cover PC and PS4, raw-value preservation, inactive residual
slots, invalid requests, excessive counts and a table outside the slot.
Endpoint tests cover the stored 274-resource catalog, locked and unlocked
membership, deterministic ordering, unknown raw IDs, inactive slots and a
duplicate catalog region ID. Transport coverage compares the HTTP response with
the typed getter result.

`SetRegionUnlocked` remains contract-only and is not exposed in OpenAPI or
Scalar.
