# DeleteCharacter

## Overview

`DeleteCharacter` permanently clears one active or residual character slot in
place. It never shifts another character. This differs from
[`SetCharacterActive(false)`](set_character_active.md), which only hides a slot
and deliberately preserves its character data.

The mutation changes the session's private snapshot. The source save remains
untouched until [`WriteSave`](../savesession/write_save.md) succeeds.

| | |
|---|---|
| EndpointID | `delete_character` |
| Kind | Mutation |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | transport-exposed — `DELETE /api/v1/save-sessions/{saveSessionID}/characters/{characterID}` of the loopback-only local explorer, and the `DeleteCharacter` Wails binding of the desktop application |
| Implementation source | [../../../backend/endpoints/character/delete_character.go](../../../backend/endpoints/character/delete_character.go) |
| Endpoint tests | [../../../backend/endpoints/character/delete_character_test.go](../../../backend/endpoints/character/delete_character_test.go) |
| SaveEngine source | [../../../backend/saveengine/delete_character.go](../../../backend/saveengine/delete_character.go) |
| SaveEngine tests | [../../../backend/saveengine/delete_character_test.go](../../../backend/saveengine/delete_character_test.go) |
| Mutation | atomic clearing of the target slot data, activity flag and full profile summary |

## Input

```go
func DeleteCharacter(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	expectedRevision string,
) (DeleteCharacterResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Exact identifier of an existing session. |
| `characterID` | `int` | Physical character slot, `0` to `9`. |
| `expectedRevision` | `string` | Canonical decimal revision that must equal the current session revision. |

The HTTP transport requires `expectedRevision` in a strict JSON body, rejects
unknown fields, and accepts only `application/json`.

## Output

```go
type DeleteCharacterResult struct {
	MutationReceipt
	CharacterID   int    `json:"characterID"`
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
receipt members and `characterID` all sit at the top level, and there is no
nested `receipt` object.

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
  `delete_character`.
- `changedScopes` are exactly `save.session`, `character.list`,
  `character.profile`, `character.stats`, `character.appearance`, `inventory`,
  `storage`, `equipment.loadout`, `world.flags`, `diagnostics.report`, in that
  canonical order.

The receipt identifies the deleted physical slot and the newly committed
revision. It contains no deleted name, account identifier, save bytes or
private offsets.

## Occupancy rules

An active slot is deletable from its exact activity flag alone. An inactive
slot is deletable only when a residual character name remains in PlayerGameData
or in its ProfileSummary. A fully zeroed slot is rejected as already empty.

An inactive slot with unknown nonzero data but no confirmed residual name is
not silently treated as either empty or occupied. Missing layout evidence and
activity bytes other than `0` or `1` fail closed without clearing anything.

## Save mutation

Deletion clears exactly three ranges owned by the selected physical slot:

- `0x280000` bytes of slot data;
- its one-byte activity flag in `UserData10`;
- its complete `0x24C`-byte ProfileSummary, including its opaque appearance and
  equipment snapshot.

The other nine slots, their flags and summaries, UserData11, and unrelated
UserData10 fields are untouched. Slots after the deleted slot are not shifted
or renumbered; gaps are valid in the game's positional model.

PC slot data begins after its 16-byte MD5 prefix. The private mutation leaves
that prefix untouched, as other SaveEngine mutations do. A later `WriteSave`
recalculates every required PC MD5. PS4 carries no corresponding prefix.

## Atomicity and revisions

SaveEngine reads all three complete original ranges before the first write.
Validation, writes, verification and any rollback run under the engine mutex.
If a write or verification fails, the three original ranges are restored and
verified; the revision and dirty state do not change.

A successful deletion advances `saveRevision` once, marks the session dirty and
invalidates identities minted under the previous revision. Deleting an already
empty slot is an error rather than an idempotent commit.

## Legacy comparison

SaveForge 1.5.8 and 1.6.10 use the same clear-in-place model and the same three
ranges. Both accept an active slot or an inactive slot with a residual name and
reject a slot with neither. The later positional implementation replaced an old
shift-down deletion that could desynchronize profile summaries; SaveForge 2.0
implements only the confirmed clear-in-place behavior.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It reads no GameCatalog resource.
- It creates no runtime or build dependency on the legacy SaveForge tree.

## Verification coverage

Synthetic PC and PS4 coverage verifies exact three-range clearing, preservation
of all neighbouring bytes, removal of active and residual characters, rejection
of empty and unknown slots, persistence through `WriteSave` and reload, strict
JSON transport, loopback-only route registration, and OpenAPI/Scalar
conformance.
