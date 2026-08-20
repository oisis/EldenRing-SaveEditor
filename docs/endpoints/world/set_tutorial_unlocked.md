# SetTutorialUnlocked

## Purpose

`SetTutorialUnlocked` sets the unlock state of one titled tutorial in one
character slot. The endpoint resolves the tutorial resource through GameCatalog
and passes only the private `TutorialParam` row ID to SaveEngine, which adds or
removes that ID in the `TutorialData` list under `expectedRevision` control.

The endpoint exposes no row ID and no save offset. The declared payload size, the
block header, unknown IDs and every byte outside the changed range remain
untouched, so the write is in place and never rebuilds the slot. Event flags,
Inventory, Storage and every other section are outside this contract.

The session must already exist through [`LoadSave`](../savesession/load_save.md).
The mutation changes only its private in-memory snapshot;
[`WriteSave`](../savesession/write_save.md) is still required to persist it.

| | |
|---|---|
| EndpointID | `set_tutorial_unlocked` |
| Kind | Mutation |
| Domain | `world` |
| Supported resource types | `TutorialDocument` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/tutorials/unlock` in the local explorer; the route is absent when it runs with `-allow-external-bind` |
| Implementation source | [../../../backend/endpoints/world/set_tutorial_unlocked.go](../../../backend/endpoints/world/set_tutorial_unlocked.go) |
| Endpoint tests | [../../../backend/endpoints/world/set_tutorial_unlocked_test.go](../../../backend/endpoints/world/set_tutorial_unlocked_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_tutorial_unlocked.go](../../../backend/saveengine/set_tutorial_unlocked.go) |
| SaveEngine tests | [../../../backend/saveengine/set_tutorial_unlocked_test.go](../../../backend/saveengine/set_tutorial_unlocked_test.go) |

## Input

```go
func SetTutorialUnlocked(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	tutorialKind string,
	tutorialKey string,
	unlocked bool,
	expectedRevision string,
) (SetTutorialUnlockedResult, error)
```

| Parameter | Meaning |
|---|---|
| `saveSessionID` | Existing save session identifier. |
| `characterID` | Physical character slot, `0` to `9`. |
| `tutorialKind` | Must be exactly `tutorial`. |
| `tutorialKey` | Exact public key of a `TutorialDocument`, which is its decimal `TutorialParam` row ID. |
| `unlocked` | Desired unlock state. |
| `expectedRevision` | Canonical decimal revision that must equal the current session revision. |

## Output

```go
type SetTutorialUnlockedResult struct {
	SaveSessionID string              `json:"saveSessionID"`
	SaveRevision  string              `json:"saveRevision"`
	CharacterID   int                 `json:"characterID"`
	TutorialKind  schema.ResourceKind `json:"tutorialKind"`
	TutorialKey   string              `json:"tutorialKey"`
	Unlocked      bool                `json:"unlocked"`
}
```

The result reports public catalog identity and the committed state. It does not
return the private `TutorialParam` row ID.

## Catalog resolution

The endpoint matches `tutorialKind == "tutorial"` and resolves the resource with
`ResourceByKindAndKey`. The resource must carry a `TutorialDocument` with a
confirmed, non-zero `TutorialID`; anything else is rejected before a save byte is
read. Only that row ID is handed to SaveEngine.

The `TutorialDocument` payloads themselves are validated when GameCatalog is
constructed. The setter performs its own exact `kind`/`key` lookup and its own
check for a confirmed, non-zero `TutorialID`; it does not call
[`GetTutorials`](get_tutorials.md) and depends on no other endpoint.

## SaveEngine mutation semantics

`TutorialData` stores an eight-byte header with the declared payload size,
followed by a `uint32` count and that many `TutorialParam` row IDs. The declared
size is read from the save, never assumed to be `0x400`, and is never changed:
the mutation writes the count and the part of the ID array either the old or the
new list uses as one in-place range.

- `unlocked: true`:
  - if the row ID is already present, nothing is written;
  - if the existing list is ascending, the row ID is inserted at its ascending
    position, which is what the native list of case `1590` shows;
  - if the existing list is not ascending, the row ID is appended at the end and
    the existing physical order is preserved verbatim. SaveForge 1.5.8 and 1.6.10
    appended every new ID, so a list one of them wrote may be unsorted; such a
    list is never sorted, deduplicated or normalised;
  - a list that already fills the declared payload capacity, or the hard cap of
    `0xFF` entries, is rejected fail-closed.
- `unlocked: false`:
  - if the row ID is absent, nothing is written;
  - every occurrence of exactly that row ID is removed, the remaining entries
    keep their exact order and values — including unknown and unsorted ones — the
    active part shifts left, the freed entries are zeroed and `count` drops by the
    number of removed occurrences.

A malformed layout — an unreadable header, a declared size outside the slot or
snapshot, a declared size too small to hold the `uint32` count itself, or a count
above the payload capacity or the hard cap — is rejected before any write.

## Residual gameplay risk of `unlocked: false`

Removing a tutorial ID is **not** a guaranteed inert operation. No build, tool or
documented experiment has ever removed one: SaveForge 1.5.8, 1.6.10 and every
reference parser only append or round-trip the block. Two effects are plausible
and unmeasured:

- the game may re-trigger the tutorial popup on the next matching event and
  reinsert the ID itself;
- `TutorialData` membership gates whether the game hands out a tutorial-bound
  item, so removing `1590` (Crimson Crystal Tear bundle) or `2010` (Crafting Kit /
  About Item Crafting) may make the world copy spawn again next to a player who
  already owns the inventory copy.

The endpoint deliberately blocks no specific ID and returns no warning field. The
open measurement is tracked in the project TODO under *TutorialData: gameplay side
effects of `unlocked=false`*.

## Rollback and atomicity

The count and the affected part of the ID array form a single contiguous write
range. `applyByteWrites` reads the previous bytes of that range before the write,
verifies it afterwards, and restores it if the write or the verification fails. A rejected mutation leaves the
snapshot, the revision, the dirty flag and the undo point exactly as they were.

An idempotent request writes no byte. It still advances `saveRevision` by `1` and
marks the session dirty under the existing revision contract, but it creates no
empty undo point.
