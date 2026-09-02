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
	saveengine.MutationReceipt
	CharacterID   int                 `json:"characterID"`
	RegionKind    schema.ResourceKind `json:"regionKind"`
	RegionKey     string              `json:"regionKey"`
	Unlocked      bool                `json:"unlocked"`
}
```

The result reports public catalog identity and the committed state. It does not
return the private internal `RegionID`.

The result embeds the shared `MutationReceipt` anonymously, so the JSON stays
flat: `operationID`, `operationKind`, `saveSessionID`, `saveRevision` and
`changedScopes` are top-level members beside the domain fields, and there is no
nested `receipt` object.

The embedded receipt is exactly the one the central SaveEngine commit path
produced for this execution. Nothing here is reassembled from the EndpointID,
the session, the revision or a scope lookup.

- `operationID` names this one execution. It is opaque and unpredictable.
  Identifiers do not repeat among the receipts issued by one running SaveEngine
  instance. That guarantee does not currently cover application restarts:
  uniqueness across restarts requires a persistent operation journal and stays
  outside this stage. A rejected call returns the complete zero result and no
  `operationID` at all.
- `operationKind` is the stable kind of the mutation and is always exactly
  `set_region_unlocked`.
- `changedScopes` are exactly `save.session`, `world.flags`, `diagnostics.report`, in that canonical order.
  This mutation writes World state only, so neither Inventory nor Storage is invalidated.

A committed request identical to the current state still advances `saveRevision`
and still returns a complete receipt with a fresh `operationID`: the central
commit path runs even when no byte changes.

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
which differs intentionally from SaveForge 1.5.8 / 1.6.10 legacy behavior.

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
