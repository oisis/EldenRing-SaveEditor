# UndoCharacterChanges

## Overview

`UndoCharacterChanges` reverts the last committed mutation of one character by
restoring the bytes captured before it. It replays no domain mutation in
reverse: it writes back the stored pre-image of the three ranges a character
mutation owns. It changes only the session's private snapshot; the source save
remains untouched until [`WriteSave`](../savesession/write_save.md) is called.

| | |
|---|---|
| EndpointID | `undo_character_changes` |
| Kind | Mutation |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | transport-exposed — `POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/undo` of the loopback-only local explorer |
| Implementation source | [../../../backend/endpoints/character/undo_character_changes.go](../../../backend/endpoints/character/undo_character_changes.go) |
| Endpoint tests | [../../../backend/endpoints/character/undo_character_changes_test.go](../../../backend/endpoints/character/undo_character_changes_test.go) |
| SaveEngine source | [../../../backend/saveengine/undo.go](../../../backend/saveengine/undo.go) |
| SaveEngine tests | [../../../backend/saveengine/undo_test.go](../../../backend/saveengine/undo_test.go) |
| Mutation | atomic restore of one character's slot data, `ProfileSummary` and activity flag; advances `saveRevision` by 1 |

## Input

```go
func UndoCharacterChanges(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	undoToken string,
	expectedRevision string,
) (UndoCharacterChangesResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Exact identifier of an existing session. |
| `characterID` | `int` | Physical character slot, `0` to `9`. It must be the slot the undo point belongs to. |
| `undoToken` | `string` | The exact token reported by [`GetUndoState`](get_undo_state.md). It is required and compared byte for byte. |
| `expectedRevision` | `string` | Canonical decimal revision that must equal the current session revision. |

`undoToken` is an opaque SaveEngine identifier, **not** a GameResource
reference. The endpoint supports no catalog resource type and reads no
GameCatalog variable.

The transport requires both body fields and rejects unknown JSON fields:

```json
{
  "undoToken": "...",
  "expectedRevision": "..."
}
```

## Output

```go
type UndoCharacterChangesResult struct {
	SaveSessionID     string `json:"saveSessionID"`
	SaveRevision      string `json:"saveRevision"`
	CharacterID       int    `json:"characterID"`
	UndoneOperationID string `json:"undoneOperationID"`
}
```

`UndoneOperationID` is the `EndpointID` of the mutation that was reverted. The
receipt exposes no save offset, no undo byte and no private session state.

## Save mutation

The undo point stores the raw pre-image of exactly three physical ranges of one
character:

- the character's slot data, `0x280000` bytes at the platform's slot base;
- the character's `ProfileSummary`, `0x24C` bytes in `UserData10`;
- the character's single activity byte in the `UserData10` flag array.

No other slot, no other range and no container-level structure is read or
written. The PC per-entry MD5 prefixes are outside the scope, because `WriteSave`
regenerates all eleven of them from the data it persists.

## Atomicity and revisions

The whole operation runs under the existing engine mutex.

1. `expectedRevision` must be canonical and equal to the current revision.
2. `characterID` must be inside `0..9`.
3. The session's single undo point must exist and belong to this session, this
   character and this revision.
4. `undoToken` must match it exactly.
5. All three current ranges are read **before** the first byte is written.
6. The three stored ranges are written, then all three are verified.
7. A write or verification failure restores the state the call started from and
   reports the failure. The undo point, the revision and the dirty flag are left
   untouched, so a failed undo consumes nothing.

A successful undo consumes the point, restores the `unsavedChanges` flag that
was in force before the undone mutation, advances `saveRevision` by exactly one
and retires every `OwnedItemID` minted under the previous revision. It creates no
redo point and no new undo point, so undo cannot be repeated.

## Failure behavior

The endpoint fails without mutation for a missing engine, an empty or unknown
session, a malformed or stale `expectedRevision`, a `characterID` outside `0..9`,
an empty or mismatched `undoToken`, an absent or foreign undo point, an
unreadable range, and a write or verification failure.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It uses no GameCatalog data.
- It creates no runtime or build dependency on the legacy SaveForge tree.

## Legacy comparison

SaveForge 1.5.8 and 1.6.8 implemented `RevertSlot` identically: ten independent
per-slot stacks of depth five, each entry holding the activity flag, the
`ProfileSummary` and a structural slot snapshot, cleared on load and on a
successful save. SaveForge 2.0 keeps the capture scope and the lifecycle
boundaries, and deliberately drops the stack, the depth cap and the structural
snapshot: `saveRevision` is global here, so any second entry would already be
unusable. 2.0 additionally requires an `expectedRevision` and an unpredictable
`undoToken`, which the legacy implementation had no equivalent of. No legacy
type, helper or implementation is imported or called.

## Verification coverage

SaveEngine coverage proves that a real mutation creates a point, that a correct
undo restores the slot data, the activity flag and the `ProfileSummary`, consumes
the point, advances the revision by one and restores the earlier dirty flag, that
a wrong token, character or revision changes nothing, that a later mutation
replaces or invalidates the point, that a successful `WriteSave` clears it while
a rejected one keeps it, and that the behaviour holds on both PC and PS4
fixtures. Endpoint coverage proves the delegation and the missing-engine
rejection; transport coverage proves the POST route, the strict JSON body, the
loopback-only registration and the OpenAPI conformance.
