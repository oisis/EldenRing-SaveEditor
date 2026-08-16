# SetQuestStep

## Purpose

`SetQuestStep` sets an NPC questline to an explicitly specified supported step
in one character slot of an existing save session. The endpoint resolves the
`quest` resource and the requested `quest_step` through GameCatalog, prepares the
canonical event flag plan, and delegates one atomic event flag mutation to
SaveEngine under `expectedRevision` control.

The endpoint does not expose raw event flag IDs, bit offsets or binary save
structures. Unknown quest or step keys fail closed without touching the save.
Fog of War, map regions, Map Fragments, Inventory and Storage are outside this
contract and remain untouched.

The session must already exist through [`LoadSave`](../savesession/load_save.md).
The mutation changes only its private in-memory snapshot; [`WriteSave`](../savesession/write_save.md)
is still required to persist it.

| | |
|---|---|
| EndpointID | `set_quest_step` |
| Kind | Mutation |
| Domain | `world` |
| Supported resource types | `QuestDocument` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/quests/step` in the local explorer; the route is absent when it runs with `-allow-external-bind` |
| Implementation source | [../../../backend/endpoints/world/set_quest_step.go](../../../backend/endpoints/world/set_quest_step.go) |
| Endpoint tests | [../../../backend/endpoints/world/set_quest_step_test.go](../../../backend/endpoints/world/set_quest_step_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_quest_step.go](../../../backend/saveengine/set_quest_step.go) |
| SaveEngine tests | [../../../backend/saveengine/set_quest_step_test.go](../../../backend/saveengine/set_quest_step_test.go) |

## Input

```go
func SetQuestStep(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	questKind string,
	questKey string,
	stepKind string,
	stepKey string,
	expectedRevision string,
) (SetQuestStepResult, error)
```

| Parameter | Meaning |
|---|---|
| `saveSessionID` | Existing save session identifier. |
| `characterID` | Physical character slot, `0` to `9`. |
| `questKind` | Must be exactly `quest`. |
| `questKey` | Exact public snake_case slug of a `QuestDocument`. |
| `stepKind` | Must be exactly `quest_step`. |
| `stepKey` | Exact public key of a supported `QuestStepDocument` (e.g. `legacy_000`). |
| `expectedRevision` | Canonical decimal revision that must equal the current session revision. |

## Output

```go
type SetQuestStepResult struct {
	SaveSessionID string              `json:"saveSessionID"`
	SaveRevision  string              `json:"saveRevision"`
	CharacterID   int                 `json:"characterID"`
	QuestKind     schema.ResourceKind `json:"questKind"`
	QuestKey      string              `json:"questKey"`
	StepKind      string              `json:"stepKind"`
	StepKey       string              `json:"stepKey"`
}
```

The result reports public catalog identity and the committed state. It does not
return private event flags or offsets.

## Catalog resolution and supported steps

The endpoint matches `questKind == "quest"` and exact `questKey`, then verifies
that `stepKind == "quest_step"` and matches `stepKey` within `resource.Quest.Steps`.

SaveForge 2.0 supports exactly 908 quest steps across 36 NPC questlines whose
flags resolve to confirmed Binary Search Tree (BST) blocks. The 34 legacy steps
that touched unmapped BST blocks are excluded fail-closed. Conflicting duplicate
flags in legacy steps are canonicalized with last-write-wins before mutation.

The endpoint implements no transition graph, no prerequisites and no automatic
step progression: it applies exactly the requested step plan.
[`GetQuests`](get_quests.md) reports the match state of the same step plans and
likewise declares no transitions; neither endpoint calls the other.

## SaveEngine mutation semantics

`SetQuestStep` resolves each flag ID in the step plan to its confirmed byte offset
and bit position in `confirmedEventFlagBlocks`. Flags sharing the same byte are
merged into a single write mask, combining bitwise SET (`|`) and CLEAR (`&^`)
operations.

Before applying changes, the original bytes are captured. `applyByteWrites` writes
and verifies every byte. If verification fails, all modified bytes are rolled back
to their original values.

The operation advances `saveRevision` by `1` and marks the session dirty;
an undo point (`set_quest_step`) is recorded only when at least one byte in the
slot data actually changed.
