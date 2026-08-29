# SetCharacterStats

## Overview

`SetCharacterStats` atomically assigns the eight attributes of one active
character and recalculates the two values the save keeps consistent with them:
the character level and, when it is too low, `TotalGetSoul`. It changes only the
session's private snapshot. The source save remains untouched until
[`WriteSave`](../savesession/write_save.md) is called.

| | |
|---|---|
| EndpointID | `set_character_stats` |
| Kind | Mutation |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/stats` of the loopback-only local explorer |
| Implementation source | [../../../backend/endpoints/character/set_character_stats.go](../../../backend/endpoints/character/set_character_stats.go) |
| Endpoint tests | [../../../backend/endpoints/character/set_character_stats_test.go](../../../backend/endpoints/character/set_character_stats_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_character_stats.go](../../../backend/saveengine/set_character_stats.go) |
| SaveEngine tests | [../../../backend/saveengine/set_character_stats_test.go](../../../backend/saveengine/set_character_stats_test.go) |
| Mutation | atomic assignment of one contiguous `PlayerGameData` range and the `ProfileSummary` level; advances `saveRevision` by 1 |

## Input

```go
func SetCharacterStats(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	attributes CharacterAttributes,
	levelPolicy string,
	expectedRevision string,
) (SetCharacterStatsResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Exact identifier of an existing session. |
| `characterID` | `int` | Physical character slot, `0` to `9`. The slot must be active. |
| `attributes` | `CharacterAttributes` | The complete writable attribute set. All eight fields are mandatory. |
| `levelPolicy` | `string` | Exactly `recalculate`. Any other value is rejected. |
| `expectedRevision` | `string` | Canonical decimal revision that must equal the current session revision. |

```go
type CharacterAttributes struct {
	Vigor        uint32 `json:"vigor"`
	Mind         uint32 `json:"mind"`
	Endurance    uint32 `json:"endurance"`
	Strength     uint32 `json:"strength"`
	Dexterity    uint32 `json:"dexterity"`
	Intelligence uint32 `json:"intelligence"`
	Faith        uint32 `json:"faith"`
	Arcane       uint32 `json:"arcane"`
}
```

The transport requires all three body fields, requires all eight attribute
fields and rejects unknown JSON fields at both levels. It uses nullable decoder
fields internally, so an omitted attribute is rejected instead of reaching
SaveEngine as the illegal value `0`.

`levelPolicy` is matched byte for byte. It is never trimmed, lower-cased or
otherwise normalised, so ` recalculate` and `Recalculate` are rejected. No
second policy exists: preserving a stored level, or supplying a level
explicitly, is not part of this contract.

The level is not an input. It is always derived, so a request cannot store a
level that contradicts the attributes it ships with.

## Output

```go
type SetCharacterStatsResult struct {
	SaveSessionID string              `json:"saveSessionID"`
	SaveRevision  string              `json:"saveRevision"`
	CharacterID   int                 `json:"characterID"`
	Attributes    CharacterAttributes `json:"attributes"`
	Level         uint32              `json:"level"`
	SoulMemory    uint32              `json:"soulMemory"`
}
```

The receipt returns the accepted attributes, the revision created by the
mutation and the two derived values as they were actually stored. It exposes no
save offset, no private save byte and no starting class.

## Validation

1. `expectedRevision` must be a canonical decimal revision, and `levelPolicy`
   must be exactly `recalculate`.
2. Every attribute must lie in `1..99`. Both boundaries are valid; `0` and `100`
   are rejected, never clamped.
3. The recalculated level must lie in `1..713`.
4. The character's own `StartingClassID` is read from its `PlayerGameData`, and
   every attribute must be at or above that class's base value — the same floor
   the game enforces when respeccing. The minima are resolved directly from the
   GameCatalog `class` resources (`0` through `11`), which serve as their single
   source of truth. A `StartingClassID` outside the twelve confirmed classes
   is a hard rejection: an unknown class carries no known minima, so its save
   is not written.

   `PlayerGameData` is the authoritative copy of the class, read relative to the
   same character anchor as the attributes. The second copy in the character's
   `ProfileSummary` is menu data that can be stale — for example after an edit by
   an older SaveForge — and does not take part in this mutation: it is neither
   read for the minima, nor compared, nor synchronised, nor repaired.

## Derived values

**Level.** SaveEngine always recalculates it:

```
level = vigor + mind + endurance + strength + dexterity + intelligence + faith + arcane - 79
```

The sum is taken in a type that cannot overflow. The minimum is `1` (Wretch with
every attribute at its base `10`) and the maximum is `713` (every attribute at
`99`).

**SoulMemory.** `TotalGetSoul` is the lifetime-runes field of `PlayerGameData`,
called `SoulMemory` by the legacy implementation. The minimum a character must
carry is the sum of the per-level costs

```
cost(n) = floor(0.02*n^3 + 3.06*n^2 + 105.6*n - 895)
```

clamped to zero, summed for `n = classBaseLevel+1 .. level`, where
`classBaseLevel` is the base Rune Level of the character's own starting class as
declared by its GameCatalog class document.

The minimum is therefore **relative to the class, not absolute from level 1**. A
character at or below its class base level owes nothing, so the minimum is `0`.
This is not a tolerance: the native vanilla save stores every one of the ten
freshly created classes at its own base level `1..10` with `TotalGetSoul` exactly
`0`, and a floor summed from level 1 would report those untouched characters as
illegal.

When the stored value is below that minimum it is raised to exactly the minimum;
when it already equals or exceeds it, it is left untouched. The endpoint never
lowers it.

The sum is evaluated in integer arithmetic, so the result cannot depend on the
host's floating-point behaviour. Six per-level costs are corrected by one to
reproduce the established results at the boundaries where the historical
floating-point evaluation rounded down. The reference vectors are those of the
1.6.10 integer evaluation and its own reference tests; the 1.5.8 `float64` sum is
not a reference, because its results depend on the host. They are level `1` →
`0`, `9` → `473`, `50` → `256598`, `150` → `7106585` and `713` → `1692560963`,
each counted from base level `1`; the maximum fits into `uint32`, so no clamp is
needed. Each per-level correction stays attached to its own level, so a sum that
starts above level 1 applies exactly the corrections it covers.

## Save mutation

The mutation writes exactly two ranges:

- one contiguous `PlayerGameData` range that starts at the first attribute and
  ends with `TotalGetSoul`. It is read, modified in memory and written back as a
  whole, so the held-runes field and the unknown words inside it are preserved
  byte for byte;
- the four-byte level of the character's `ProfileSummary` in `UserData10`, so
  the character list and the in-game slot summary cannot disagree with
  `PlayerGameData`.

The `PlayerGameData` range is located relative to the same bounded character
anchor used by the statistics reader, which stays the single source of that
position. PC and PS4 use their existing platform-specific slot bases; no
absolute offset is assumed.

Nothing else changes. In particular the endpoint does not touch `HP`, `MaxHP`,
`BaseMaxHP`, `FP`, `MaxFP`, `BaseMaxFP`, `SP`, `MaxSP`, `BaseMaxSP`, the runes
held by the character, the starting class, the name, the appearance, the
inventory, the equipment or any hash. The combat statistics the game derives
from the attributes are not recomputed here: the game recomputes them itself.

## Atomicity and revisions

SaveEngine validates the level policy, the attribute ranges and the recalculated
level before the session is opened at all, then validates the session, the
canonical revision, the slot index, the active flag, the anchor and the starting
class. Both target ranges are resolved and read before the first byte is
written, so a truncated range fails the whole mutation untouched.

Both writes are verified under the existing engine mutex. A failed write or
verification restores both ranges and reports the failure, without advancing the
revision or marking the session dirty; a rollback that cannot itself be verified
is reported as such rather than presented as an unchanged save.

A successful assignment advances `saveRevision` by exactly one and marks the
session dirty. This includes an idempotent assignment where the stored values
already equal the requested ones, matching the other mutation endpoints.

## Legacy comparison

SaveForge 1.5.8 and 1.6.10 agree on the field layout, the level formula, the
`1..99` attribute range, the `1..713` level range and the ten base starting
classes with their base attributes; the two class tables are byte-identical.
Neither carries the Regulation 1.17 classes `10` Idus Knight and `11` Heavy
Knight, which this build resolves from its own GameCatalog documents. Both also
validate the minima against the `PlayerGameData` class at offset `-248` from the
character anchor, never against the `ProfileSummary` copy. This endpoint keeps
that source.

They differ in how the SoulMemory minimum is evaluated. 1.5.8 summed the
per-level cost in `float64`, which made the result depend on the host's
floating-point behaviour. 1.6.10 replaced it with integer arithmetic that
reproduces the established results. This endpoint reimplements the 1.6.10
integer evaluation and pins its confirmed reference vectors in a test.

Both legacy versions summed that cost from level 1 regardless of class. This
endpoint does not: the sum starts above the base level of the character's own
class, because the native save proves a freshly created class owes nothing at its
base level. The reference vectors are unchanged for a base level of `1`.

They also differ on an unknown `StartingClassID`. 1.x downgraded it to a warning
and skipped the class check. This endpoint rejects it: an unknown class has no
confirmed minima, and silently skipping the check would write a save under a
rule that was never verified.

No legacy implementation, helper, type or package is imported or called.

## Failure behavior

The endpoint fails without mutation for a missing engine, an empty or unknown
session, a malformed or stale revision, a character index outside `0..9`, an
inactive slot, a `levelPolicy` other than `recalculate`, an attribute outside
`1..99`, an attribute below its starting class's base value, a recalculated
level outside `1..713`, an unknown starting class, a missing anchor, a truncated
range, or a write/verification failure. Negative JSON values fail decoding
because the public values are unsigned.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It uses embedded GameCatalog data: the starting-class minima are resolved from
  the GameCatalog `class` resources (`backend/gamecatalog/data/classes/`), which
  also supply the base level the SoulMemory minimum is counted from.
- It creates no runtime or build dependency on the legacy SaveForge tree.

## Verification coverage

Synthetic PC and PS4 coverage verifies the exact set of mutated bytes,
preservation of all unrelated bytes including the held runes, the required
SoulMemory raise counted from the class base level, an already sufficient
SoulMemory left untouched, a class sitting at its own base level with
`SoulMemory` `0` accepted as legal, the maximum
attributes with level `713`, rejection of `0`, `100` and a value below the class
minimum, rejection of a value that is legal for a stale `ProfileSummary` class
but below the real `PlayerGameData` class minimum, rejection of an unknown class
and of a `levelPolicy` other than
`recalculate`, invalid sessions/slots/revisions, inactive and anchorless slots
with the snapshot, revision and dirty state unchanged, idempotent assignment,
persistence through `WriteSave`/`LoadSave` re-read through `GetCharacterStats`
and `GetCharacterProfile`, the confirmed SoulMemory reference vectors, strict
JSON transport including a missing attribute and unknown fields at both levels,
loopback-only route registration, and OpenAPI/Scalar conformance.

Controlled native before/after evidence has not yet been collected on either
platform, so in-game semantic validation remains a recorded limitation beyond
the matching legacy layout and the synthetic persistence coverage.
