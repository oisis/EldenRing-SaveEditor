# SetCharacterStartingClass

## Overview

`SetCharacterStartingClass` atomically changes a character's starting class as a
**destructive build reset**. The eight attributes become exactly the base
attributes of the target class and the level becomes exactly its base Rune Level,
both read from the GameCatalog class document. Distributed points are discarded,
attributes are lowered as freely as they are raised, and the level is never
derived from the attribute sum here.

`TotalGetSoul` (`SoulMemory`) and the held runes are preserved unchanged: a class
change earns and spends nothing.

Because the reset destroys the current build, the request must carry
`confirmReset: true`. It keeps both the `PlayerGameData` and `ProfileSummary`
class copies synchronised, and likewise both level copies. It changes only the
session's private snapshot. The source save remains untouched until
[`WriteSave`](../savesession/write_save.md) is called.

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
	confirmReset bool,
	expectedRevision string,
) (SetCharacterStartingClassResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Exact identifier of an existing session. |
| `characterID` | `int` | Physical character slot, `0` to `9`. The slot must be active. |
| `startingClassID` | `uint8` | Target starting class identifier (`0`..`9`). Must resolve in GameCatalog class resources. |
| `confirmReset` | `bool` | Explicit confirmation of the destructive reset. Only `true` performs the mutation. |
| `expectedRevision` | `string` | Canonical decimal revision that must equal the current session revision. |

```json
{
  "startingClassID": 4,
  "confirmReset": true,
  "expectedRevision": "0"
}
```

The transport requires all three body fields and rejects unknown JSON fields. It
uses nullable pointers for `startingClassID` and `confirmReset` internally, so an
omitted class is rejected instead of reaching SaveEngine as the valid Vagabond
class `0`, and an omitted confirmation is rejected by name instead of silently
reading as a refusal.

### The `confirmReset` gate

`confirmReset` is checked first, before the revision, the slot, the class and any
read of the snapshot. A request with `confirmReset` absent or `false` is rejected
without any mutation and without advancing `saveRevision`.

A frontend must present a confirmation modal before it sends `true`. The modal
contract is:

- state that **all eight attributes and the Rune Level will be reset to the base
  values of the selected class**;
- state that **SoulMemory and held runes stay unchanged**;
- offer **Confirm** and **Cancel**;
- send the request **only** after **Confirm**. **Cancel** sends nothing.

No frontend exists in this repository yet; this is the contract the future one
must implement.

### Undo

The reset is destructive for the build, not irreversible inside the session. A
successful mutation leaves the ordinary single undo point of the session under
the operation ID `set_character_starting_class`, and
[`UndoCharacterChanges`](undo_character_changes.md) restores the exact prior
state: the starting class, the level, all eight attributes, `SoulMemory` and the
held runes.

That point is one level deep and not durable, exactly as for every other
character mutation: the next mutation replaces it, and
[`WriteSave`](../savesession/write_save.md) ends the possibility of undoing. The
confirmation gate exists because the build is discarded, not because the session
could not roll it back.

## Output

```go
type SetCharacterStartingClassResult struct {
	SaveSessionID   string              `json:"saveSessionID"`
	SaveRevision    string              `json:"saveRevision"`
	CharacterID     int                 `json:"characterID"`
	StartingClassID uint8               `json:"startingClassID"`
	Attributes      CharacterAttributes `json:"attributes"`
	Level           uint32              `json:"level"`
	SoulMemory      uint32              `json:"soulMemory"`
}
```

The receipt returns the session ID, the new revision, the character ID, the
applied `startingClassID`, the eight base attributes of that class, its base
level, and the `SoulMemory` the mutation left unchanged.

There is no `attributesRaised` field. The reset always writes all eight
attributes, so a flag reporting whether some of them moved carries no
information.

## Validation

1. `expectedRevision` must be a canonical decimal revision equal to the current
   session revision.
2. `characterID` must be `0..9` and the slot must be active.
3. `startingClassID` must resolve to one of the ten confirmed starting classes
   (`0` Vagabond through `9` Wretch) via GameCatalog class resources. An unknown
   class ID is a hard rejection: never defaulted, never skipped, never partially
   written.

`confirmReset` is validated before all of these.

## Attribute and level rule

The reset copies the target class document verbatim:

```
attributes = the eight base attributes of the target class
level      = ClassDocument.Level of the target class
```

Consequences of the rule:
- The stored attributes are never read as input. Whatever they were — a developed
  build, or a value an external editor left outside `1..99` — they are replaced.
- Attributes are lowered as freely as they are raised. A character does lose
  levels when it moves to a class with a lower base level.
- The level is **not** recomputed as `sum(attributes) - 79` here. That formula is
  the rule of [`SetCharacterStats`](set_character_stats.md); this endpoint takes
  the confirmed `soulLv` fact of the class instead. The two agree for every
  confirmed class, but the fact is the source.
- `TotalGetSoul` (`SoulMemory`) is never read as a constraint and never written.
  It is neither raised to a level floor nor lowered. Held runes are likewise
  untouched. [`SetCharacterStats`](set_character_stats.md) remains the only path
  that raises `TotalGetSoul`.

### Confirmed class definitions

| ID | Class | Base level | VIG | MND | END | STR | DEX | INT | FTH | ARC |
|---|---|---|---|---|---|---|---|---|---|---|
| 0 | Vagabond | 9 | 15 | 10 | 11 | 14 | 13 | 9 | 9 | 7 |
| 1 | Warrior | 8 | 11 | 12 | 11 | 10 | 16 | 10 | 8 | 9 |
| 2 | Hero | 7 | 14 | 9 | 12 | 16 | 9 | 7 | 8 | 11 |
| 3 | Bandit | 5 | 10 | 11 | 10 | 9 | 13 | 9 | 8 | 14 |
| 4 | Astrologer | 6 | 9 | 15 | 9 | 8 | 12 | 16 | 7 | 9 |
| 5 | Prophet | 7 | 10 | 14 | 8 | 11 | 10 | 7 | 16 | 10 |
| 6 | Confessor | 10 | 10 | 13 | 10 | 12 | 12 | 9 | 14 | 9 |
| 7 | Samurai | 9 | 12 | 11 | 13 | 12 | 15 | 9 | 8 | 8 |
| 8 | Prisoner | 9 | 11 | 12 | 11 | 11 | 14 | 14 | 6 | 9 |
| 9 | Wretch | 1 | 10 | 10 | 10 | 10 | 10 | 10 | 10 | 10 |

The `6` Confessor, `7` Samurai and `8` Prisoner mapping is confirmed against a
native vanilla save holding all ten freshly created classes. The create-character
menu display order is a different order and must never be used as the ID.

## Save mutation

The mutation writes exactly four ranges:

1. the contiguous `PlayerGameData` block from the first attribute through
   `TotalGetSoul` (`statsWritableBlockOffset` / `statsWritableBlockSize`),
   carrying the eight attributes and the level, with `TotalGetSoul`, the held
   runes and the unknown words preserved byte for byte;
2. the `PlayerGameData` class byte at `anchor - 248` (`statsClassOffset`);
3. the `ProfileSummary` level in `UserData10` at `summary + 0x22`;
4. the `ProfileSummary` class byte in `UserData10` at `summary + 0x243`.

The `PlayerGameData` block and class byte are located relative to the bounded
character anchor. PC and PS4 use their existing platform-specific slot bases; no
absolute offset is assumed.

Nothing else changes: not the name, not the gender, not the appearance, not
`TotalGetSoul`, not the held runes, not the inventory, not the storage, not the
equipment, not the spells, not any hash.

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

Both legacy versions used a `max(current, min)` raise with `level = sum - 79`.
SaveForge 2.0 deliberately does **not** reproduce that: a class change is a build
reset, the level comes from the class `soulLv` fact, `SoulMemory` is preserved,
and the mutation requires an explicit confirmation. An unknown starting class is
rejected fail-closed in 2.0 rather than silently ignored.

The legacy class table also mapped IDs `6`, `7` and `8` incorrectly. That mapping
is not restored; see the confirmed table above.

No legacy code, types or structure are imported or called.

## Failure behavior

The endpoint fails without mutation for a missing engine, an empty or unknown
session, a malformed or stale revision, a character index outside `0..9`, an
inactive slot, an unknown `startingClassID`, a missing or refused `confirmReset`,
a missing anchor, a truncated range, or a write/verification failure.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It uses embedded GameCatalog data: the base attributes and the base level of
  every class are resolved from the GameCatalog `class` resources
  (`backend/gamecatalog/data/classes/`), which are SaveEngine's single source of
  truth for both.
- It creates no runtime or build dependency on the legacy SaveForge tree.

## Verification coverage

Synthetic PC and PS4 coverage verifies:
- a developed character reset to the exact base build of each of the ten classes,
  with every other byte of the snapshot proven unchanged;
- the base level moving both down (Confessor 10 to Wretch 1) and up (Wretch 1 to
  Confessor 10), taken from the class document, not from the attribute sum;
- a stored attribute outside `1..99` overwritten by the class base rather than
  carried forward;
- both class copies and both level copies written and agreeing;
- `SoulMemory` identical after the reset at `0` and at a high stored value, and
  held runes identical;
- `confirmReset: true` committing, `false` and an omitted field rejected without
  mutation or revision advance;
- a repeated identical reset keeping the revision semantics of the endpoint;
- unknown `startingClassID` rejected without mutation or revision advance;
- invalid revisions, inactive and anchorless slots failing cleanly;
- truncated ranges failing without partial writes;
- persistence through `WriteSave`/`LoadSave` re-read through `GetCharacterProfile`
  and `GetCharacterStats`;
- strict JSON transport including missing and unknown fields;
- loopback-only route registration and OpenAPI/Scalar conformance.
