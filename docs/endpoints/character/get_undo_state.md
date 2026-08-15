# GetUndoState

## Overview

`GetUndoState` reports whether the save session still holds a usable undo point
for one character, and which operation created it. It reads the session only: it
opens no file, reads no save byte, and changes no snapshot, revision, dirty flag
or `OwnedItemID` registry.

| | |
|---|---|
| EndpointID | `get_undo_state` |
| Kind | Getter |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/undo` of the loopback-only local explorer |
| Implementation source | [../../../backend/endpoints/character/get_undo_state.go](../../../backend/endpoints/character/get_undo_state.go) |
| Endpoint tests | [../../../backend/endpoints/character/get_undo_state_test.go](../../../backend/endpoints/character/get_undo_state_test.go) |
| SaveEngine source | [../../../backend/saveengine/undo.go](../../../backend/saveengine/undo.go) |
| SaveEngine tests | [../../../backend/saveengine/undo_test.go](../../../backend/saveengine/undo_test.go) |
| Mutation | none; the getter is non-mutating |

## Input

```go
func GetUndoState(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
) (CharacterUndoState, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Exact identifier of an existing session. |
| `characterID` | `int` | Physical character slot, `0` to `9`. The slot does not have to be active. |

## Output

```go
type CharacterUndoState struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	Available     bool   `json:"available"`
	UndoToken     string `json:"undoToken,omitempty"`
	OperationID   string `json:"operationID,omitempty"`
}
```

`Available` is true only when the session's single undo point belongs to this
character **and** its revision is the session's current revision. When it is
false, `UndoToken` and `OperationID` are empty and omitted from JSON.

`UndoToken` is an opaque SaveEngine identifier. It is not a GameResource
reference, carries no offset, no slot address and no save byte, and no consumer
parses it. `OperationID` is the `EndpointID` of the mutation the point would
revert, for example `set_character_stats` or `set_weapon_infusion`.

## The undo point

A save session holds **one** undo point, not a per-slot stack. It

- belongs to exactly one `characterID`;
- corresponds to exactly one successful mutation;
- is valid only for the revision that mutation created;
- stores the raw pre-image of the character's slot data (`0x280000` bytes), its
  `ProfileSummary` (`0x24C` bytes) and its one activity byte in `UserData10`,
  together with the session's `unsavedChanges` flag as it was before that
  mutation;
- is never serialized and never reaches a save file.

The PC per-entry MD5 prefixes are outside its scope, because `WriteSave`
regenerates all eleven of them from the data it is about to persist.

### Lifecycle

| Event | Effect on the undo point |
|---|---|
| `LoadSave` | The new session starts without one. |
| A rejected or rolled-back mutation | Unchanged. |
| A successful character mutation that changed slot data, the activity flag or the `ProfileSummary` | Creates a new point and replaces any earlier one, including one belonging to another character. |
| A successful character mutation that changed none of those bytes but still advanced the revision | Creates no point and drops the earlier one. |
| A successful global mutation, such as `SetNetworkSettings` or `SetSaveAccountID` | Drops the point. |
| A successful `WriteSave` | Drops the point. |
| A failed `WriteSave` | Unchanged. |
| `CloseSave` | Removed with the session. |
| A successful `UndoCharacterChanges` | Consumes the point; no redo point is created. |

One point per session is deliberate. `saveRevision` is global, so after any
further mutation an older point would already describe a revision that no longer
exists and could never be applied.

## Failure behavior

The endpoint fails without side effects for a missing engine, an empty or
unknown `saveSessionID`, and a `characterID` outside `0..9`. An absent undo point
is a normal result with `available: false`, not an error.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It uses no GameCatalog data.
- It creates no runtime or build dependency on the legacy SaveForge tree.

## Legacy comparison

SaveForge 1.5.8 and 1.6.8 are identical here: both kept ten independent undo
stacks of depth five, each entry holding the slot snapshot, the `ProfileSummary`
and the activity flag, and both cleared every stack on load and on a successful
save. `GetUndoDepth` returned the stack length. SaveForge 2.0 keeps the same
capture scope but not the stack, not the depth counter and no legacy type,
helper or structural snapshot.

## Verification coverage

SaveEngine coverage proves that the getter is non-mutating, that it reports the
token and operation identifier of a real mutation, and that `available` turns
false for the wrong character or a stale revision. Endpoint coverage proves the
delegation and the missing-engine rejection; transport coverage proves the GET
route, its loopback-only registration and its OpenAPI conformance.
