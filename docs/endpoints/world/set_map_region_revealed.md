# SetMapRegionRevealed

## Purpose

`SetMapRegionRevealed` sets the visibility of one curated map region in one
character slot. When GameCatalog links the visibility flag to a Map Fragment,
the same atomic mutation also makes that goods item present or absent in
`InventoryHeld`.

The endpoint does not expose raw event flags or game IDs. The transient
Map Fragment acquisition flags in block `63`, system map flags, unsafe
sub-region flags, Fog of War and Storage are outside the contract.

The session must already exist through [`LoadSave`](../savesession/load_save.md).
The mutation changes only its private in-memory snapshot; [`WriteSave`](../savesession/write_save.md)
is still required to persist it.

| | |
|---|---|
| EndpointID | `set_map_region_revealed` |
| Kind | Mutation |
| Domain | `world` |
| Supported resource types | `MapRegionDocument` plus the goods `ItemDocument` declaring the matching map unlock, when one exists |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/map-regions/reveal` in the local explorer; the route is absent when it runs with `-allow-external-bind` |
| Implementation source | [../../../backend/endpoints/world/set_map_region_revealed.go](../../../backend/endpoints/world/set_map_region_revealed.go) |
| Endpoint tests | [../../../backend/endpoints/world/set_map_region_revealed_test.go](../../../backend/endpoints/world/set_map_region_revealed_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_map_region_revealed.go](../../../backend/saveengine/set_map_region_revealed.go) |
| SaveEngine tests | [../../../backend/saveengine/set_map_region_revealed_test.go](../../../backend/saveengine/set_map_region_revealed_test.go) |

## Input

```go
func SetMapRegionRevealed(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	mapRegionKind string,
	mapRegionKey string,
	revealed bool,
	expectedRevision string,
) (SetMapRegionRevealedResult, error)
```

| Parameter | Meaning |
|---|---|
| `saveSessionID` | Existing save session identifier. |
| `characterID` | Physical character slot, `0` to `9`. |
| `mapRegionKind` | Must be exactly `map_region`. |
| `mapRegionKey` | Exact public key of a `MapRegionDocument`. |
| `revealed` | Desired visibility state. |
| `expectedRevision` | Canonical decimal revision that must equal the current session revision. |

## Output

```go
type SetMapRegionRevealedResult struct {
	SaveSessionID string              `json:"saveSessionID"`
	SaveRevision  string              `json:"saveRevision"`
	CharacterID   int                 `json:"characterID"`
	MapRegionKind schema.ResourceKind `json:"mapRegionKind"`
	MapRegionKey  string              `json:"mapRegionKey"`
	Revealed      bool                `json:"revealed"`
}
```

The result reports public catalog identity and the committed state. It does not
return the private visibility event flag or Map Fragment game ID.

## Catalog resolution

The endpoint first uses the same `catalogMapRegions` resolver as
[`GetMapRegions`](get_map_regions.md). The complete curated map-region table is
validated before SaveEngine reads the session, including duplicate visibility
flags.

Map Fragments are resolved only through existing GameCatalog data:

1. an `ItemDocument` declares exactly one `item.unlocks` entry whose kind is
   `map`;
2. that entry carries a known event flag declared by exactly one map region;
3. the item has a known goods game ID;
4. no second Map Fragment may claim the same visibility flag.

A map region without such an item relation has no fragment. Its mutation writes
only the visibility bit. Names, area labels, ordering and arithmetic on flag IDs
are never used to guess the relation.

## Mutation semantics

| Request | Visibility flag | Linked Map Fragment |
|---|---|---|
| `revealed: true` | set | present in `InventoryHeld` |
| `revealed: false` | clear | absent from `InventoryHeld` |

The item operation is a presence assignment, not a quantity operation. An
existing positive-quantity record is preserved. A missing item is created with
quantity `1` in the common Inventory section. Removal accepts the existing raw
key-item ID or derived goods handle, rejects duplicate and zero-quantity
records, and applies the normal owned-item reference protection.

The endpoint never reads or writes Storage. It also never writes block `63`
Map Fragment acquisition flags. Those flags are transient pickup triggers and
were not part of this mutation in SaveForge 1.5.8 or 1.6.10.

SaveEngine resolves the block `62` position before entering the session
mutation. Under one revision-controlled operation it validates slot activity,
plans the Inventory change, plans the flag byte and applies the combined byte
plan. Any validation or verification error leaves the snapshot, dirty state and
revision unchanged. Success advances `saveRevision` by exactly one and creates
an undo point under `set_map_region_revealed`.

## HTTP route

```text
PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/map-regions/reveal
```

```json
{
  "mapRegionKind": "map_region",
  "mapRegionKey": "limgrave_weeping_peninsula",
  "revealed": true,
  "expectedRevision": "0"
}
```

The body must use `application/json`, rejects unknown fields, and requires an
explicit boolean `revealed` value.

## Validation and errors

Every rejection is fail-closed.

| Condition | Behaviour |
|---|---|
| nil SaveEngine or GameCatalog | rejected before any save read |
| wrong `mapRegionKind` or unknown key | rejected before mutation |
| duplicate or incomplete map-region declarations | rejected before mutation |
| invalid or conflicting Map Fragment declaration | rejected before mutation |
| non-canonical or stale `expectedRevision` | rejected without advancing revision |
| inactive or out-of-range character slot | rejected without reading residual slot data |
| visibility flag outside block `62` | rejected by SaveEngine |
| non-goods fragment game ID | rejected by SaveEngine |
| duplicate, malformed or referenced fragment record | rejected before the flag write |
| truncated layout or failed write verification | the complete plan is rejected or rolled back |

## Legacy comparison

SaveForge 1.5.8 and 1.6.10 have the same `MapFragmentItems` table and the same
`applyMapRegionUnlock` behavior: the safe map visibility flag follows the
requested state, and a mapped Map Fragment is added or removed with it. Both
versions deliberately leave the block `63` acquisition flag untouched.

Their generic item writer receives the forced-common option for this flow.
SaveForge 2.0 reimplements that confirmed result through its existing common
Inventory record planner and combines the item and flag writes into one atomic
plan. It does not import legacy code or restore retired allocation or repacking
behavior.

The dynamic event-flag layout after the platform-specific slot base is shared
by PC and PS4. Synthetic fixtures cover both codecs and verify persistence by
write and reload.

## Verification

```bash
go test ./backend/saveengine -run 'SetMapRegionRevealed|SetWhetbladeUnlocked' -count=1
go test -race ./backend/saveengine -run 'SetMapRegionRevealed|SetWhetbladeUnlocked' -count=1
go test ./backend/endpoints/world -run 'SetMapRegionRevealed|GetMapRegions' -count=1
go test ./tools/swagger -run 'SetMapRegionRevealed|OpenAPIDocumentDescribesEveryRoute' -count=1
make test
npm --prefix frontend run build
git diff --check
```
