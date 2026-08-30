# GetQuests

## Overview

`GetQuests` returns the curated NPC questlines GameCatalog declares, together
with the match state of every supported step for one character slot. The stored
catalog declares 36 questlines with 908 supported steps.

A step is `matched` when **every** event flag of its canonical plan currently
holds the target value the plan declares. That is a statement about the save
state alone. The endpoint deliberately does **not** name a current step and does
**not** declare allowed transitions — see
[No current step and no transitions](#no-current-step-and-no-transitions).

The getter reads only the private session snapshot. It never opens or writes a
save file and does not mutate the session, its revision, its dirty state, its
undo history or the `OwnedItemID` registry.

| | |
|---|---|
| EndpointID | `get_quests` |
| Kind | Getter |
| Domain | `world` |
| Implementation status | implemented |
| Transport | `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/quests`, loopback explorer only |
| Save access | read-only session snapshot |

## Input

```go
func GetQuests(
    engine *saveengine.Engine,
    gameCatalog *gamecatalog.Catalog,
    saveSessionID string,
    characterID int,
    questKind string,
    questKey string,
) (GetQuestsResult, error)
```

- `saveSessionID` is matched exactly and `characterID` is the slot index `0..9`.
  The contract draft did not list `saveSessionID`; it is required here, because
  the reported state belongs to one session snapshot, exactly as in every other
  implemented world getter.
- `questKind` is matched exactly and case-sensitively against `quest`. An empty
  or different value is rejected; it is never defaulted.
- `questKey` selects one curated questline. Omitted or empty returns every
  declared questline; a non-empty unknown key is rejected instead of being
  answered with an empty result.

## No current step and no transitions

The contract draft described "allowed transitions derived from the catalog". No
such graph exists in any confirmed source, and none was introduced:

- The curated quest data is a flat list of independently addressable steps. Each
  step declares only a key, a description, a location and a canonical event flag
  plan. There is no predecessor, successor, precondition or exclusion field.
- SaveForge 1.5.8, 1.6.10 and 1.7.1 carry a **byte-identical**
  `GetQuestProgress` in their canonical Git tags. It evaluates every step
  independently and marks it
  `Complete` when all of its flags equal their targets. It never selects a
  current step, never resolves ties between several complete steps, never uses
  step order for anything but display, and represents "no match" simply as every
  step being incomplete. Their UI lets the user apply any step at any time; the
  "transitions" of the draft never existed as behaviour.
- Several plans of one questline legitimately hold at the same time. On a
  freshly cleared event flag bitfield, for example, 5 Fia steps and 7 Melina
  steps match, because their plans declare only cleared flags. Naming one of
  them the current step would require a rule no source provides.

Deriving order or adjacency semantics from the step index is therefore
forbidden, and `matched` is intentionally not called `current`, `complete` or
`progress`.

## Catalog contract

GameCatalog is the only source of public identity. `schema.ValidateResource`
already enforces, so an offending catalog never loads:

- kind `quest` and a key of lowercase letters, digits and underscores;
- no document of any other kind in the resource union;
- a known, non-empty `quest.name`;
- at least one supported step;
- per step: a valid, unique step key, a known non-empty description, a validated
  location fact and a non-empty plan of unique, non-zero event flag IDs.

The endpoint additionally fails closed on a `quest` resource whose quest document
is missing. It contains no fixed quest list.

The 34 legacy steps whose plans reach unconfirmed event flag blocks are not part
of the catalog and are therefore neither reported nor synthesised. A step that is
not in the catalog is simply not part of this endpoint's answer.

## Save contract

The step plans are resolved through the single existing
`saveengine.resolveEventFlag` and read through one bulk `saveengine.GetEventFlags`
call. No second resolver, no separate BST table and no `eventFlagID/8` fallback
were introduced, and no new SaveEngine method was needed.

The endpoint collects the **distinct** flag identifiers of every selected
questline first — 2034 of them for the full catalog — and performs exactly one
`GetEventFlags` call for all of them, instead of one read per step. It never
locates the `EventFlags` section itself and never decodes a bit itself. PC and
PS4 differ only in the container that locator already owns, so both platforms
share this contract unchanged.

An inactive or residual slot is reported from its activity flag alone: `active`
is `false`, every `matched` is `false`, and not one byte of that slot's data is
read. The unmatched result is asserted explicitly rather than inferred from an
empty flag map, because a plan of only cleared flags would otherwise appear to
match a bitfield that was never looked at.

Event flag IDs, their target values, their BST positions, offsets and masks are
not part of any result field. The complete step plan stays available through the
general [`GetResource`](../catalog/get_resource.md) getter, as its contract
defines. Provenance is carried by the curated facts only — `quest.name` and each
step's `description` and `location`. A single plan entry declares exactly an
event flag ID and its target value and has no per-flag provenance; none is
synthesised here.

## Result

```go
type GetQuestsResult struct {
    SaveSessionID string       `json:"saveSessionID"`
    CharacterID   int          `json:"characterID"`
    Active        bool         `json:"active"`
    Quests        []QuestEntry `json:"quests"`
}

type QuestEntry struct {
    Kind  schema.ResourceKind `json:"kind"`
    Key   string              `json:"key"`
    Name  string              `json:"name"`
    Steps []QuestStepEntry    `json:"steps"`
}

type QuestStepEntry struct {
    StepKind    string `json:"stepKind"`
    StepKey     string `json:"stepKey"`
    Description string `json:"description"`
    Location    string `json:"location"`
    Matched     bool   `json:"matched"`
}
```

`Quests` is always a non-nil array, empty rather than `null`. Questlines are
ordered by `name`, then `key`, so paging and diffing are stable across calls.
Steps keep the declaration order of their catalog document; that order is stable
but carries no confirmed progression meaning. `StepKind` is always `quest_step`,
the same value [`SetQuestStep`](set_quest_step.md) accepts. `Location` is empty
when the curated source declares none.

## Errors

| Condition | Behaviour |
|---|---|
| `engine` or `gameCatalog` is nil | error before any read |
| `questKind` other than `quest` | error before any read |
| non-empty unknown `questKey` | error before any read |
| a `quest` resource without a quest document | error |
| empty or unknown `saveSessionID` | error |
| `characterID` outside `0..9` | error |
| an event flag in an unsupported block | error from `resolveEventFlag` |
| an unreadable or truncated event flag section | error |

Every failure happens before a partial result is produced; there is no fallback
value and no partially populated answer.

## Related endpoints

- [`SetQuestStep`](set_quest_step.md) applies exactly one step plan. It reads
  nothing from this getter and this getter calls nothing from it; both resolve
  the same catalog documents independently.
- [`GetBosses`](get_bosses.md), [`GetGraces`](get_graces.md) and
  [`GetSummoningPools`](get_summoning_pools.md) share the same
  catalog-plus-one-bulk-flag-read shape.
