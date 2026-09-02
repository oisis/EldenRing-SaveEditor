# SetGestureUnlocked

## Overview

`SetGestureUnlocked` assigns the unlock state of one logical gesture resource in
one active character slot. The caller supplies the public GameCatalog identity;
the physical gesture slot ID remains internal.

| Source | What it provides |
|---|---|
| GameCatalog | exact `gestureKind` and `gestureKey` resolution, the `gesture` family, and the canonical ID in `item.gesture.slots[].slotID` |
| SaveEngine | atomic interpretation and mutation of the 64-record `GestureGameData` block under `expectedRevision` control |

The endpoint changes only the session's private snapshot. The source save stays
untouched until [`WriteSave`](../savesession/write_save.md) is called.

| | |
|---|---|
| EndpointID | `set_gesture_unlocked` |
| Kind | Mutation |
| Domain | `world` |
| Supported resource types | `ItemDocument: Gesture` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gestures/unlock` of the loopback-only local explorer |
| Implementation source | [../../../backend/endpoints/world/set_gesture_unlocked.go](../../../backend/endpoints/world/set_gesture_unlocked.go) |
| Endpoint tests | [../../../backend/endpoints/world/set_gesture_unlocked_test.go](../../../backend/endpoints/world/set_gesture_unlocked_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_gesture_unlocked.go](../../../backend/saveengine/set_gesture_unlocked.go) |
| SaveEngine tests | [../../../backend/saveengine/set_gesture_unlocked_test.go](../../../backend/saveengine/set_gesture_unlocked_test.go) |
| Mutation | atomic assignment in `GestureGameData`; advances `saveRevision` by 1 |

## Input

```go
func SetGestureUnlocked(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	gestureKind string,
	gestureKey string,
	unlocked bool,
	expectedRevision string,
) (SetGestureUnlockedResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Exact identifier of an existing session. |
| `characterID` | `int` | Physical character slot, `0` to `9`. The slot must be active. |
| `gestureKind` | `string` | Exact GameCatalog resource kind. It must resolve to `item`. |
| `gestureKey` | `string` | Exact GameCatalog key of the logical gesture resource. |
| `unlocked` | `bool` | Desired logical state, not an operation or toggle. |
| `expectedRevision` | `string` | Canonical decimal revision that must equal the current session revision. |

The transport requires all four body fields and rejects unknown JSON fields. It
uses a pointer only while decoding `unlocked`, so an omitted `false` cannot be
mistaken for an explicit request to lock the gesture.

## Output

```go
type SetGestureUnlockedResult struct {
	saveengine.MutationReceipt
	CharacterID   int                 `json:"characterID"`
	GestureKind   schema.ResourceKind `json:"gestureKind"`
	GestureKey    string              `json:"gestureKey"`
	Unlocked      bool                `json:"unlocked"`
}
```

```json
{
  "operationID": "op-3f9c…",
  "operationKind": "set_gesture_unlocked",
  "saveSessionID": "9f1c…",
  "saveRevision": "1",
  "changedScopes": ["save.session", "world.flags", "diagnostics.report"],
  "characterID": 0,
  "gestureKind": "item",
  "gestureKey": "401EA7AB",
  "unlocked": true
}
```

The receipt contains the public logical identity only. It never exposes the raw
slot ID, an offset, a record index, or save bytes.

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
  `set_gesture_unlocked`.
- `changedScopes` are exactly `save.session`, `world.flags`, `diagnostics.report`, in that canonical order.
  This mutation writes World state only, so neither Inventory nor Storage is invalidated.

A committed request identical to the current state still advances `saveRevision`
and still returns a complete receipt with a fresh `operationID`: the central
commit path runs even when no byte changes.

## Catalog resolution

The endpoint resolves exactly `(gestureKind, gestureKey)` through GameCatalog
and then requires:

- an `ItemDocument` whose known family is `gesture`;
- a gesture document with one known, non-zero, odd canonical slot ID; or
- the one confirmed two-slot alias set `{227, 233}` for Ring of Miquella.

Any other multi-slot resource fails closed. The endpoint does not derive an ID
from the key or name, scan other resources for a substitute, or accept a public
`slotID` parameter.

## Native record assignment

Each of the 64 little-endian `uint32` records has one of these roles:

| Stored value | Meaning for this mutation |
|---|---|
| odd canonical ID `n` | gesture `n` is unlocked |
| non-zero preceding even value `n - 1` | mutable gesture `n` is locked |
| `0xFFFFFFFE` | native empty record available for a new unlock |
| zero or any unrelated value | preserved exactly |

For an ordinary gesture:

- unlock is a no-op when the odd ID already exists;
- otherwise the first exact `n - 1` record becomes `n`, except that protected
  slot `1` never treats zero as a writable predecessor;
- if no exact locked record exists, the first `0xFFFFFFFE` becomes `n`;
- a full block with neither location is rejected;
- lock changes every duplicate exact `n` into `n - 1`.

The mutation never sorts, compacts, deduplicates, purges, or repairs the block.
Unknown values, zeroes, unrelated canonical IDs, and all untouched sentinels stay
byte-for-byte unchanged.

## Protected starting gestures

The confirmed base starting set cannot be locked:

| Slot ID | Gesture |
|---:|---|
| 1 | Bow |
| 13 | Warm Welcome |
| 15 | Wave |
| 41 | Point Forwards |
| 43 | Point Upwards |
| 45 | Point Downwards |
| 47 | Beckon |
| 49 | Wait! |
| 101 | Rallying Cry |
| 141 | Jump for Joy |
| 161 | Dejection |
| 185 | Rest |

A lock request is rejected without mutation when the protected odd record is
present. Unlocking a protected gesture still uses the ordinary assignment rule,
which can restore a missing record without creating a second policy.

## Ring of Miquella alias

Ring of Miquella is the only special alias:

- `227` is the protected pre-order grant;
- `233` is the version earned by helping a player who owns the pre-order grant;
- either stored odd ID means the logical resource is unlocked;
- unlock changes `232` to `233`, or uses the first empty sentinel when `232` is
  absent;
- unlock never creates or modifies `227` or its preceding value `226`;
- lock changes every `233` to `232` only when `227` is absent;
- if `227` exists, lock is rejected and `233` is also left untouched.

This exception is local to the confirmed pair. There is no generic alias
framework and no GameCatalog schema extension.

## Atomicity and revisions

SaveEngine performs validation, location, mutation, verification, and any
rollback under the existing engine mutex. Before the first write it validates:

- canonical `expectedRevision` and exact revision match;
- `characterID` and the active-slot flag;
- the confirmed anchor and projectile count;
- the complete `GestureGameData` range inside the character slot and snapshot;
- the protected-gesture and capacity rules.

The whole 0x100-byte block is read before mutation. If a changed block cannot be
verified, SaveEngine restores the original block and does not advance the
revision or mark the session dirty. A successful state assignment, including an
idempotent one, advances the revision by exactly one and retires identities from
the previous revision, consistently with the other mutation endpoints.

## Platform evidence

PC and PS4 use the same confirmed layout inside the character slot and differ
only in their existing slot-bound entry points. Both paths are covered by
synthetic mutation, rejection, persistence, reload, and race tests.

The odd-unlocked/even-locked semantics and the protected starting set were also
verified against a controlled native PC/Steam Deck starting save. PS4 semantic
behavior is not yet backed by a controlled console before/after save pair; it
remains a recorded validation risk rather than an assumption hidden by the
synthetic coverage.

## Failure behavior

The endpoint fails without mutation for an unknown session or resource, an
inactive slot, a stale or malformed revision, a non-gesture resource, unknown or
unsupported slot declarations, malformed layout, a full block, and a protected
lock request. Error messages identify the rejected field or rule without
including private save contents.

## Verification coverage

Regression coverage includes ordinary unlock and lock, exact placeholder
preference, sentinel insertion, duplicate locking, idempotent assignment, full
capacity, preservation of zero and unknown records, every protected starting
gesture, Ring of Miquella alias behavior, invalid session/character/revision data,
PC and PS4 persistence, source-file immutability, endpoint/catalog validation,
strict JSON transport, and OpenAPI/Scalar conformance.
