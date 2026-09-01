# SetCharacterName

## Overview

`SetCharacterName` assigns the exact name of one active character slot. It
synchronizes the two confirmed name copies in the save and changes only the
session's private snapshot. The source save remains untouched until
[`WriteSave`](../savesession/write_save.md) is called.

| | |
|---|---|
| EndpointID | `set_character_name` |
| Kind | Mutation |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/name` of the loopback-only local explorer |
| Implementation source | [../../../backend/endpoints/character/set_character_name.go](../../../backend/endpoints/character/set_character_name.go) |
| Endpoint tests | [../../../backend/endpoints/character/set_character_name_test.go](../../../backend/endpoints/character/set_character_name_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_character_name.go](../../../backend/saveengine/set_character_name.go) |
| SaveEngine tests | [../../../backend/saveengine/set_character_name_test.go](../../../backend/saveengine/set_character_name_test.go) |
| Mutation | atomic assignment of both UTF-16LE name fields; advances `saveRevision` by 1 |

## Input

```go
func SetCharacterName(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	name string,
	expectedRevision string,
) (SetCharacterNameResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Exact identifier of an existing session. |
| `characterID` | `int` | Physical character slot, `0` to `9`. The slot must be active. |
| `name` | `string` | Exact name to store. It is neither trimmed nor normalised. |
| `expectedRevision` | `string` | Canonical decimal revision that must equal the current session revision. |

The name must be non-empty valid UTF-8, contain no U+0000, and encode to at
most 16 UTF-16 code units. The unit limit is intentional: a supplementary
Unicode character such as an emoji consumes two units. The endpoint does not
truncate, replace, case-fold, normalise, trim, or apply a character whitelist.

The transport requires both body fields, rejects unknown JSON fields, and
accepts only `application/json`.

## Output

```go
type SetCharacterNameResult struct {
	MutationReceipt
	CharacterID   int    `json:"characterID"`
	Name          string `json:"name"`
}

type MutationReceipt struct {
	OperationID   string   `json:"operationID"`
	OperationKind string   `json:"operationKind"`
	SaveSessionID string   `json:"saveSessionID"`
	SaveRevision  string   `json:"saveRevision"`
	ChangedScopes []string `json:"changedScopes"`
}
```

The receipt is embedded anonymously, so the JSON result is flat: the five
receipt members and `characterID`, `name` all sit at the top level, and there is
no nested `receipt` object.

The embedded `saveengine.MutationReceipt` is exactly the receipt the central
SaveEngine commit path produced for this execution. Nothing here is
reassembled from the EndpointID, the session, the revision or a scope lookup.

- `operationID` names this one execution. It is opaque and unpredictable.
  Identifiers do not repeat among the receipts issued by one running SaveEngine
  instance. That guarantee does not currently cover application restarts:
  uniqueness across restarts requires a persistent operation journal and stays
  outside this stage. A rejected call returns the complete zero result and no
  `operationID` at all.
- `operationKind` is the stable kind of the mutation and is always exactly
  `set_character_name`.
- `changedScopes` are exactly `save.session`, `character.list`,
  `character.profile`, `diagnostics.report`, in that canonical order.

The receipt returns the exact accepted name and the revision created by the
mutation. It exposes no save offsets, raw UTF-16 units, or private save bytes.

## Save mutation

SaveEngine encodes one zero-padded 32-byte UTF-16LE field and writes it to both
confirmed locations:

- the character name in `PlayerGameData` inside the active slot;
- the matching `UserData10` profile-summary name.

Only those two 32-byte fields are assigned. The two bytes immediately following
each field are not part of the name and remain untouched. If the old copies
disagree, a successful call synchronizes both to the requested value.

The PC and PS4 paths use their existing platform-specific slot and summary
bases; the two name fields have the same confirmed layout relative to those
bases.

## Atomicity and revisions

Validation, location, mutation, verification, and rollback run under the
SaveEngine mutex. Before writing, SaveEngine validates the session, canonical
revision, active slot, name encoding, anchor, and both complete destination
ranges. It reads both original fields before the first write.

If either write or verification fails, both original fields are restored and
the revision and dirty state do not change. A successful assignment advances
`saveRevision` by exactly one and marks the session dirty. This includes an
idempotent assignment where both fields already contain the requested bytes,
matching the revision contract of other mutation endpoints.

## Failure behavior

The endpoint fails without mutation for a missing engine, empty or unknown
session, malformed or stale revision, character index outside `0..9`, inactive
slot, invalid name, missing anchor, truncated range, or write/verification
failure. Error messages identify the rejected field or rule without including
the submitted name or private save contents.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It uses no GameCatalog data.
- It creates no runtime or build dependency on the legacy SaveForge tree.

## Verification coverage

Synthetic PC and PS4 coverage verifies exact two-field mutation, preservation
of adjacent bytes, synchronization of mismatched copies, the 16-unit boundary,
supplementary Unicode characters, invalid UTF-8, U+0000, inactive and malformed
slots, stale revisions, idempotent assignment, persistence through
`WriteSave`/`LoadSave`, strict JSON transport, loopback-only route registration,
and OpenAPI/Scalar conformance.

The field locations and encoding behavior match SaveForge 1.5.8 and 1.6.10.
Controlled native PS4 before/after evidence has not yet been collected, so PS4
semantic validation remains a recorded limitation beyond the synthetic layout
and persistence coverage.
