# SetCharacterStartingClass

## Overview

`SetCharacterStartingClass` atomically changes a character's starting class,
raising any deficient attributes to meet the new class's minima, recalculating
the character level and, when too low, `TotalGetSoul` (`SoulMemory`). It keeps
both the `PlayerGameData` and `ProfileSummary` class copies synchronised. It
changes only the session's private snapshot. The source save remains untouched
until [`WriteSave`](../savesession/write_save.md) is called.

| | |
|---|---|
| EndpointID | `set_character_starting_class` |
| Kind | Mutation |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/starting-class` of the loopback-only local explorer |
| Implementation source | [../../../backend/endpoints/character/set_character_starting_class.go](../../../backend/endpoints/character/set_character_starting_class.go) |
| Endpoint tests | [../../../backend/endpoints/character/set_character_starting_class_test.go](../../../backend/endpoints/character/set_character_starting_class_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_character_starting_class.go](../../../backend/saveengine/set_character_starting_class.go) |
| SaveEngine tests | [../../../backend/saveengine/set_character_starting_class_test.go](../../../backend/saveengine/set_character_starting_class_test.go) |
| Mutation | atomic assignment of the `PlayerGameData` statistics block, the `PlayerGameData` class byte, the `ProfileSummary` level and the `ProfileSummary` class byte; advances `saveRevision` by 1 |

## Input

```go
func SetCharacterStartingClass(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	startingClassID uint8,
	expectedRevision string,
) (SetCharacterStartingClassResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Exact identifier of an existing session. |
| `characterID` | `int` | Physical character slot, `0` to `9`. The slot must be active. |
| `startingClassID` | `uint8` | Target starting class identifier (`0`..`9`). Must resolve in GameCatalog class resources. |
| `expectedRevision` | `string` | Canonical decimal revision that must equal the current session revision. |

```json
{
  "startingClassID": 4,
  "expectedRevision": "0"
}
```

The transport requires both body fields and rejects unknown JSON fields. It uses
a nullable pointer for `startingClassID` internally, so an omitted class is
rejected instead of reaching SaveEngine as the valid Vagabond class `0`.

## Output

```go
type SetCharacterStartingClassResult struct {
	SaveSessionID    string              `json:"saveSessionID"`
	SaveRevision     string              `json:"saveRevision"`
	CharacterID      int                 `json:"characterID"`
	StartingClassID  uint8               `json:"startingClassID"`
	Attributes       CharacterAttributes `json:"attributes"`
	Level            uint32              `json:"level"`
	SoulMemory       uint32              `json:"soulMemory"`
	AttributesRaised bool                `json:"attributesRaised"`
}
```

The receipt returns the session ID, the new revision, the character ID, the
applied `startingClassID`, the eight resulting attributes, the resulting level,
the resulting `SoulMemory`, and an `attributesRaised` boolean indicating whether
any attribute was raised to meet the new class's minima.

## Validation

1. `expectedRevision` must be a canonical decimal revision equal to the current
   session revision.
2. `characterID` must be `0..9` and the slot must be active.
3. `startingClassID` must resolve to one of the ten confirmed starting classes
   (`0` Vagabond through `9` Wretch) via GameCatalog class resources. An unknown
   class ID is a hard rejection: never defaulted, never skipped, never partially
   written.
4. The recalculated level must lie in `1..713`, and every resulting attribute
   must lie in `1..99`.

## Attribute and level rule

For each of the eight attributes:

```
new = max(current, minimum of the new class)
```

The character level is recalculated from the resulting attributes:

```
level = sum(the eight new attributes) - 79
```

Consequences of the rule:
- When every current attribute already meets the new class's minima, `max()` is a
  no-op: attributes and level remain unchanged, and `attributesRaised` is `false`.
- An attribute above its new minimum is never lowered. A character never loses
  levels, and stats are never reset to base values.
- Because the level can rise, stored `TotalGetSoul` (`SoulMemory`) may fall below
  the minimum required for the new level. When below, it is raised to exactly
  that minimum; when already sufficient, it is left untouched. It is never
  lowered.

## Save mutation

The mutation writes exactly four ranges:

1. the contiguous `PlayerGameData` block from the first attribute through
   `TotalGetSoul` (`statsWritableBlockOffset` / `statsWritableBlockSize`),
   carrying the eight attributes, the level and `TotalGetSoul`, with held runes
   and unknown words preserved byte for byte;
2. the `PlayerGameData` class byte at `anchor - 248` (`statsClassOffset`);
3. the `ProfileSummary` level in `UserData10` at `summary + 0x22`;
4. the `ProfileSummary` class byte in `UserData10` at `summary + 0x243`.

The `PlayerGameData` block and class byte are located relative to the bounded
character anchor. PC and PS4 use their existing platform-specific slot bases; no
absolute offset is assumed.

Nothing else changes: not the name, not the gender, not the appearance, not the
held runes, not the inventory, not the equipment, not any hash.

## Atomicity and revisions

SaveEngine resolves and reads all four target ranges before writing any byte.
All four writes are verified under the engine mutex. A failed write or
verification restores all four ranges and reports the failure without advancing
the revision or marking the session dirty. A rollback that cannot itself be
verified is reported as such.

A successful mutation advances `saveRevision` by exactly one and marks the session
dirty, including when the requested class equals the stored one.

## Legacy comparison

SaveForge 1.5.8 and 1.6.10 implemented starting class changes in the frontend
(`handleClassChange` in `CharacterTab.tsx`), byte-identically between versions.
SaveForge 2.0 reimplements this rule in the Go backend.

Both legacy versions used the same `max(current, min)` and `level = sum - 79`
rule. An unknown starting class is rejected fail-closed in 2.0 rather than
silently ignored.

No legacy code, types or structure are imported or called.

## Failure behavior

The endpoint fails without mutation for a missing engine, an empty or unknown
session, a malformed or stale revision, a character index outside `0..9`, an
inactive slot, an unknown `startingClassID`, a missing anchor, a truncated
range, or a write/verification failure.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It uses embedded GameCatalog data: starting-class minima are resolved from the
  GameCatalog `class` resources (`backend/gamecatalog/data/classes/`).
- It creates no runtime or build dependency on the legacy SaveForge tree.

## Verification coverage

Synthetic PC and PS4 coverage verifies:
- no-collision class change where level is unchanged and only the two class bytes
  differ;
- collision class change where only deficient attributes are raised, level is
  recalculated, and `attributesRaised` is `true`;
- both class copies written and agreeing;
- `TotalGetSoul` raised when needed, untouched when sufficient, never lowered;
- unknown `startingClassID` rejected without mutation or revision advance;
- invalid revisions, inactive and anchorless slots failing cleanly;
- truncated ranges failing without partial writes;
- persistence through `WriteSave`/`LoadSave` re-read through `GetCharacterProfile`
  and `GetCharacterStats`;
- strict JSON transport including missing and unknown fields;
- loopback-only route registration and OpenAPI/Scalar conformance.
