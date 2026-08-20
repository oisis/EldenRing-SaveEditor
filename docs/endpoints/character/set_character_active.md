# SetCharacterActive

## Overview

`SetCharacterActive` changes whether one physical character slot is visible to
the game. Deactivation preserves the character's slot and profile data so the
same residual slot can be reactivated later. The source save remains untouched
until [`WriteSave`](../savesession/write_save.md) is called.

| | |
|---|---|
| EndpointID | `set_character_active` |
| Kind | Mutation |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/active` of the loopback-only local explorer |
| Implementation source | [../../../backend/endpoints/character/set_character_active.go](../../../backend/endpoints/character/set_character_active.go) |
| Endpoint tests | [../../../backend/endpoints/character/set_character_active_test.go](../../../backend/endpoints/character/set_character_active_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_character_active.go](../../../backend/saveengine/set_character_active.go) |
| SaveEngine tests | [../../../backend/saveengine/set_character_active_test.go](../../../backend/saveengine/set_character_active_test.go) |
| Mutation | atomic assignment of one UserData10 activity byte |

## Input

```go
func SetCharacterActive(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	active bool,
	expectedRevision string,
) (SetCharacterActiveResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Exact identifier of an existing session. |
| `characterID` | `int` | Physical character slot, `0` to `9`. |
| `active` | `bool` | Target activity state. |
| `expectedRevision` | `string` | Canonical decimal revision that must equal the current session revision. |

The transport requires both body fields, rejects unknown JSON fields, and
accepts only `application/json`.

## Output

```go
type SetCharacterActiveResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	Active        bool   `json:"active"`
}
```

When the activity flag changes, `SaveRevision` is the newly committed revision.
When the stored flag already equals `active`, the operation is a successful
no-op and returns the unchanged current revision.

## Save mutation

The endpoint writes only the selected slot's one-byte activity flag in
`UserData10`. PC and PS4 use different container bases but the same confirmed
internal layout. Deactivation never clears, normalizes, or otherwise rewrites
the character slot or its profile summary.

Reactivation is fail-closed. The inactive slot must still contain the confirmed
statistics anchor, and at least one of its two confirmed character-name fields
must be non-empty. This rejects a truly empty slot and an orphaned summary whose
slot data cannot be located. An activity byte other than `0` or `1` is also
rejected instead of being normalized.

## Atomicity and revisions

Validation, mutation, verification, and rollback run under the SaveEngine
mutex. A changed flag advances `saveRevision` by one and marks the session
dirty. A validation or verification failure restores the prior byte and changes
neither revision nor dirty state. An idempotent request performs no write,
does not invalidate owned-item identities, and leaves the session clean when it
was clean before the call.

## Legacy comparison

SaveForge 1.5.8 and 1.6.10 both store slot activity as `0` or `1` in the same ten
`UserData10` bytes and preserve slot contents when toggling the flag. Their
residual-slot check accepts a deleted character when its PlayerGameData or
profile-summary name remains. SaveForge 2.0 additionally requires the existing
confirmed slot anchor before reactivation so malformed slot data is not made
game-visible.

## Failure behavior

The endpoint fails without mutation for a missing engine, empty or unknown
session, malformed or stale revision, character index outside `0..9`, unknown
activity-byte value, empty residual slot, missing slot anchor, truncated range,
or write/verification failure.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It uses no GameCatalog data.
- It creates no runtime or build dependency on the legacy SaveForge tree.

## Verification coverage

Synthetic PC and PS4 coverage verifies exact one-byte mutation, preservation of
residual data, safe reactivation, idempotence without a revision change,
rejection of empty and malformed slots, rejection of unknown flag values,
persistence through `WriteSave` and reload, strict JSON transport,
loopback-only route registration, and OpenAPI/Scalar conformance.
