# SetRegionUnlocked

## Purpose

`SetRegionUnlocked` sets the unlock state of one curated invasion or blue-summon
region in one character slot. The endpoint resolves the region resource through
GameCatalog and passes only the private internal `RegionID` to SaveEngine, which
rebuilds the variable-length `UnlockedRegions` slot section under `expectedRevision`
control.

The endpoint does not expose raw region IDs or raw save offsets. Unknown raw IDs,
zeros, duplicates of other IDs, and their exact physical order are preserved in the
save. The operation does not sort or deduplicate the whole list. Fog of War, event
flags, map regions, Map Fragments, Inventory and Storage are outside this contract
and remain untouched.

The session must already exist through [`LoadSave`](../savesession/load_save.md).
The mutation changes only its private in-memory snapshot; [`WriteSave`](../savesession/write_save.md)
is still required to persist it.

| | |
|---|---|
| EndpointID | `set_region_unlocked` |
| Kind | Mutation |
| Domain | `world` |
| Supported resource types | `RegionDocument` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/regions/unlock` in the local explorer; the route is absent when it runs with `-allow-external-bind` |
| Implementation source | [../../../backend/endpoints/world/set_region_unlocked.go](../../../backend/endpoints/world/set_region_unlocked.go) |
| Endpoint tests | [../../../backend/endpoints/world/set_region_unlocked_test.go](../../../backend/endpoints/world/set_region_unlocked_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_region_unlocked.go](../../../backend/saveengine/set_region_unlocked.go) |
| SaveEngine tests | [../../../backend/saveengine/set_region_unlocked_test.go](../../../backend/saveengine/set_region_unlocked_test.go) |

## Input

```go
func SetRegionUnlocked(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	regionKind string,
	regionKey string,
	unlocked bool,
	expectedRevision string,
) (SetRegionUnlockedResult, error)
```

| Parameter | Meaning |
|---|---|
| `saveSessionID` | Existing save session identifier. |
| `characterID` | Physical character slot, `0` to `9`. |
| `regionKind` | Must be exactly `region`. |
| `regionKey` | Exact public key of a `RegionDocument`. |
| `unlocked` | Desired unlock state. |
| `expectedRevision` | Canonical decimal revision that must equal the current session revision. |

## Output

```go
type SetRegionUnlockedResult struct {
	SaveSessionID string              `json:"saveSessionID"`
	SaveRevision  string              `json:"saveRevision"`
	CharacterID   int                 `json:"characterID"`
	RegionKind    schema.ResourceKind `json:"regionKind"`
	RegionKey     string              `json:"regionKey"`
	Unlocked      bool                `json:"unlocked"`
}
```

The result reports public catalog identity and the committed state. It does not
return the private internal `RegionID`.

## Catalog resolution

The endpoint uses the shared `catalogRegions` resolver used by
[`GetRegions`](get_regions.md). The complete curated table is validated before
any save byte is read or modified. Duplicate region IDs in catalog data fail
closed.

The endpoint matches `regionKind == "region"` and exact `regionKey`. It then
supplies the private `RegionID` to SaveEngine.

## SaveEngine mutation semantics

`SetRegionUnlocked` modifies only the membership of the targeted `RegionID`:

- `unlocked: true`:
  - if the target `RegionID` is absent from `UnlockedRegions`, it is appended
    exactly once at the end;
  - if it is already present at least once, the list remains unchanged;
  - existing duplicates of the target `RegionID` are not removed on unlock.
- `unlocked: false`:
  - all occurrences of the target `RegionID` are removed;
  - all other entries are preserved in their exact order and multiplicity.

All other raw entries (including non-curated or unknown IDs, zeros and duplicates)
are preserved verbatim. The whole list is deliberately not sorted or deduplicated,
which differs intentionally from SaveForge 1.5.8 / 1.6.8 legacy behavior.

The slot payload (`0x280000` bytes) is rebuilt through the private
`rebuildSlotWithRegions` foundation. All post-region structures shift by the
length delta, absorbing the shift in tail padding before the fixed end-of-slot
DLC and PlayerGameDataHash blocks.

## Rollback and atomicity

Before writing the rebuilt slot to the snapshot, the original slot data is
captured. After writing, SaveEngine verifies:

1. exact byte match of the rewritten slot against the rebuild buffer;
2. re-reading the unlocked regions list through `readUnlockedRegions`;
3. valid resolution of `eventFlagSectionStart`.

If any write or verification step fails, the entire original slot is restored and
verified. The revision, dirty state, undo point and OwnedItemID registries remain
untouched on failure.

An idempotent unlock advances `saveRevision` by `1` and marks the session dirty,
but creates no empty undo point because no byte in the slot changed.
