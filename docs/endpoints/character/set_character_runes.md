# SetCharacterRunes

## Overview

`SetCharacterRunes` assigns the exact number of runes currently held by one
active character. It changes only the session's private snapshot. The source
save remains untouched until [`WriteSave`](../savesession/write_save.md) is
called.

| | |
|---|---|
| EndpointID | `set_character_runes` |
| Kind | Mutation |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/runes` of the loopback-only local explorer |
| Implementation source | [../../../backend/endpoints/character/set_character_runes.go](../../../backend/endpoints/character/set_character_runes.go) |
| Endpoint tests | [../../../backend/endpoints/character/set_character_runes_test.go](../../../backend/endpoints/character/set_character_runes_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_character_runes.go](../../../backend/saveengine/set_character_runes.go) |
| SaveEngine tests | [../../../backend/saveengine/set_character_runes_test.go](../../../backend/saveengine/set_character_runes_test.go) |
| Mutation | atomic assignment of one four-byte PlayerGameData field; advances `saveRevision` by 1 |

## Input

```go
func SetCharacterRunes(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	runes uint32,
	expectedRevision string,
) (SetCharacterRunesResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Exact identifier of an existing session. |
| `characterID` | `int` | Physical character slot, `0` to `9`. The slot must be active. |
| `runes` | `uint32` | Exact held-runes value, from `0` through `999999999` inclusive. |
| `expectedRevision` | `string` | Canonical decimal revision that must equal the current session revision. |

Both range boundaries are valid. `1000000000` and every larger value are
rejected before the session is mutated. The endpoint never clamps an invalid
request to the maximum.

The transport requires both body fields and rejects unknown JSON fields. It
uses a nullable decoder field internally so an omitted `runes` cannot be
mistaken for the valid explicit value `0`.

## Output

```go
type SetCharacterRunesResult struct {
	MutationReceipt
	CharacterID   int    `json:"characterID"`
	Runes         uint32 `json:"runes"`
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
receipt members and `characterID`, `runes` all sit at the top level, and there
is no nested `receipt` object.

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
  `set_character_runes`.
- `changedScopes` are exactly `save.session`, `diagnostics.report`, in that
  canonical order.

The receipt returns the exact accepted value and the revision created by the
mutation. It exposes no save offset or private save bytes.

## Save mutation

Held runes occupy one confirmed little-endian `uint32` in `PlayerGameData`. The
field is located relative to the same bounded character anchor used by the
statistics reader. PC and PS4 use their existing platform-specific slot bases;
the relative field layout is the same.

Only those four bytes are assigned. In particular, the endpoint does not
change:

- `TotalGetSoul`, the lifetime-runes statistic next to the field, called
  `SoulMemory` by the legacy implementation;
- any level or other progression value;
- a bloodstain's recoverable runes;
- attributes, level, profile summary, inventory, or world state.

## Atomicity and revisions

SaveEngine validates the range, session, canonical revision, active slot,
anchor, and complete field range before writing. The original four bytes are
read first, and the write is verified under the existing engine mutex. A failed
verification restores the original bytes without advancing the revision or
marking the session dirty.

A successful assignment advances `saveRevision` by exactly one and marks the
session dirty. This includes an idempotent assignment where the stored value
already equals the requested value, matching the other mutation endpoints.

## Legacy comparison

SaveForge 1.5.8 and 1.6.10 agree on the field type, relative offset and legal
maximum. Their Safe Mode frontend clamped input above `999999999`; this endpoint
instead rejects it because silent input correction would hide a failed API
contract. No legacy implementation is imported or called.

## Failure behavior

The endpoint fails without mutation for a missing engine, empty or unknown
session, malformed or stale revision, character index outside `0..9`, inactive
slot, runes above `999999999`, missing anchor, truncated range, or
write/verification failure. Negative JSON values fail decoding because the
public value is unsigned.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It uses no GameCatalog data.
- It creates no runtime or build dependency on the legacy SaveForge tree.

## Verification coverage

Synthetic PC and PS4 coverage verifies exact four-byte mutation, preservation
of all unrelated bytes, zero, the legal maximum, rejection immediately above
the maximum, invalid sessions/slots/revisions, inactive and malformed slots,
idempotent assignment, persistence through `WriteSave`/`LoadSave`, source-file
immutability, strict JSON transport, loopback-only route registration, and
OpenAPI/Scalar conformance.

Controlled native PS4 before/after evidence has not yet been collected, so PS4
semantic validation remains a recorded limitation beyond the matching legacy
layout and synthetic persistence coverage.
