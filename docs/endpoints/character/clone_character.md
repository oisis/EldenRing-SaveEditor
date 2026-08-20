# CloneCharacter

## Overview

`CloneCharacter` copies one active character into a specified, completely empty
physical slot. It does not shift or renumber any character. The clone receives
the first unused suffixed name, starting with ` 2`.

The mutation changes the session's private snapshot. The source save remains
untouched until [`WriteSave`](../savesession/write_save.md) succeeds.

| | |
|---|---|
| EndpointID | `clone_character` |
| Kind | Mutation |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | transport-exposed — `POST /api/v1/save-sessions/{saveSessionID}/characters/{sourceCharacterID}/clone` of the loopback-only local explorer |
| Implementation source | [../../../backend/endpoints/character/clone_character.go](../../../backend/endpoints/character/clone_character.go) |
| Endpoint tests | [../../../backend/endpoints/character/clone_character_test.go](../../../backend/endpoints/character/clone_character_test.go) |
| SaveEngine source | [../../../backend/saveengine/clone_character.go](../../../backend/saveengine/clone_character.go) |
| SaveEngine tests | [../../../backend/saveengine/clone_character_test.go](../../../backend/saveengine/clone_character_test.go) |
| Mutation | atomic replacement of one empty target slot, activity flag and complete profile summary |

## Input

```go
func CloneCharacter(
	engine *saveengine.Engine,
	saveSessionID string,
	sourceCharacterID int,
	targetSlotID int,
	expectedRevision string,
) (CloneCharacterResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Exact identifier of an existing session. |
| `sourceCharacterID` | `int` | Active physical source slot, `0` to `9`. |
| `targetSlotID` | `int` | Different physical target slot, `0` to `9`. |
| `expectedRevision` | `string` | Canonical decimal revision that must equal the current session revision. |

The HTTP transport carries `sourceCharacterID` in the path. Its strict JSON
body requires `targetSlotID` and `expectedRevision`, rejects unknown fields and
accepts only `application/json`.

## Output

```go
type CloneCharacterResult struct {
	SaveSessionID     string `json:"saveSessionID"`
	SaveRevision      string `json:"saveRevision"`
	SourceCharacterID int    `json:"sourceCharacterID"`
	TargetSlotID      int    `json:"targetSlotID"`
	Name              string `json:"name"`
}
```

`Name` is the exact unique name written into both the cloned PlayerGameData and
ProfileSummary. The receipt contains no save bytes, account identifier or
private offset.

## Source and target validation

The source must have activity flag `1`, a confirmed statistics anchor and a
non-empty PlayerGameData name. The target must have activity flag `0`, an
entirely zero `0x280000`-byte slot data range and an entirely zero `0x24C`-byte
ProfileSummary. The source and target indices must differ.

Activity values other than `0` and `1`, residual target data, missing layout
evidence and stale revisions fail closed before the first write. This strict
empty-state requirement prevents unknown target data from being overwritten.

## Unique name

Naming starts with the source name followed by ` 2`, then tries ` 3`, ` 4` and
so on. Active and residual character names in every physical slot reserve their
suffix. Residual names are read from PlayerGameData, with ProfileSummary used
only as its fallback.

The result is at most 16 UTF-16 code units. The source-name prefix is shortened
at a complete Unicode code-point boundary to make room for the suffix; surrogate
pairs are never split.

## Save mutation

The complete source slot data and complete source ProfileSummary are copied to
the target. Only the two target name fields are replaced with the unique name,
then the target activity flag becomes `1`.

The source slot and all unrelated slots, flags and summaries remain byte-exact.
On PC, the target slot's 16-byte MD5 prefix is not part of the private mutation;
a later `WriteSave` recalculates every required checksum. PS4 has no equivalent
prefix.

## Atomicity and revisions

SaveEngine reads the complete original target ranges before writing. Validation,
writes, verification and rollback run under the engine mutex. If a write or
verification fails, the original target slot, summary and flag are restored and
verified; the revision and dirty state do not change.

A successful clone advances `saveRevision` once, marks the session dirty and
invalidates identities minted under the previous revision.

## Legacy comparison

SaveForge 1.5.8 and 1.6.10 use the same positional clone and unique-suffix rules,
including UTF-16-aware shortening and residual-name collisions. SaveForge 2.0
deliberately strengthens target validation: the target must be physically zero
instead of relying only on legacy residual-slot classification.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It reads no GameCatalog resource.
- It creates no runtime or build dependency on the legacy SaveForge tree.

## Verification coverage

Synthetic PC and PS4 coverage verifies exact target-only copying, unique names,
residual-name collisions, the UTF-16 boundary, rejection of active, residual and
unknown target states, persistence through `WriteSave` and reload, strict JSON
transport, loopback-only route registration, and OpenAPI/Scalar conformance.
