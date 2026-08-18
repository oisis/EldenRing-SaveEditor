# GetRepairPlan

## Overview

`GetRepairPlan` turns selected findings of a
[`GetSaveValidationReport`](get_save_validation_report.md) into a non-mutating
plan of the changes that would resolve them, bound to the exact save version the
findings were observed at.

It plans. It does not repair: building a plan writes nothing, reserves nothing
and changes no session state. Performing a plan belongs to `ApplyRepairs`, which
is still contract-only.

A finding becomes an action only when confirmed data determines a single target
state for it. Every other requested finding is returned as an explicit
*rejection* carrying the reason it carries no plan. This asymmetry is the point
of the endpoint: a defect whose resolution would need a policy this project has
not confirmed stays a reported defect, because inventing that policy here would
turn a heuristic into a save mutation.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetRepairPlan` never creates one,
opens no source file, returns no raw save byte, and modifies nothing.

| | |
|---|---|
| EndpointID | `get_repair_plan` |
| Kind | Getter |
| Domain | `diagnostics` |
| Implementation status | implemented |
| Transport status | transport-exposed — `POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/repair-plan` of the local explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. There is no Wails binding, no CLI command and no frontend. |
| Implementation source | [../../../backend/endpoints/diagnostics/get_repair_plan.go](../../../backend/endpoints/diagnostics/get_repair_plan.go) |
| Test source | [../../../backend/endpoints/diagnostics/get_repair_plan_test.go](../../../backend/endpoints/diagnostics/get_repair_plan_test.go) |
| Save access | read-only — one `GetSaveValidationFacts` pass over the session's private in-memory snapshot, read under one lock and one save revision; no file is opened |
| Catalog access | read-only — the same container limits and stack rules `GetSaveValidationReport` reads, reused to derive a clamping target |
| Mutation | none — the snapshot, the session, the catalog, and the save file are left unchanged |

`POST` is the transport verb because the request carries a list of identifiers,
not because the operation mutates anything. The endpoint is a getter.

## Input

```go
func GetRepairPlan(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	saveRevision string,
	issueIDs []string,
) (GetRepairPlanResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; the endpoint never creates one. A `nil` engine is rejected. |
| `gameCatalog` | `*gamecatalog.Catalog` | The catalog the container records are judged and re-derived against. A `nil` catalog is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. It is passed to SaveEngine unchanged. |
| `characterID` | `int` | The physical slot index, `0` to `9`. A plan is per-character. |
| `saveRevision` | `string` | The revision the findings were observed at, exactly as returned in `saveRevision` by the report. Required. |
| `issueIDs` | `[]string` | The `id` values of the `SaveValidationIssue` entries to plan for. At least one is required. |

### `characterID`

The endpoint map lists `saveSessionID`, `saveRevision` and `issueIDs` for this
endpoint and omits `characterID`. It is carried here because a validation issue
identifier is unique inside one character slot and nowhere else: two slots of the
same save both produce `inventory:quantity_zero:0`. `ApplyRepairs` already
carries the same variable, so the pair stays consistent.

### `saveRevision`

- It is required and matched exactly. It is what binds a plan to a save version.
- An issue identifier describes a position inside a report of one revision. A
  plan derived from identifiers of an older revision would address findings that
  have since moved, so a revision that no longer matches the session is rejected
  rather than reinterpreted.

### `issueIDs`

- Each identifier must be the `id` of a finding in the report of this same
  character at this same revision.
- An empty list, a repeated identifier and an identifier naming no current
  finding are all rejected. The endpoint never guesses which defect a caller
  meant and never silently returns a smaller plan than was asked for.
- The order of the request is ignored. Actions and rejections are ordered by the
  order the validation report lists their findings, so the same request always
  produces the same plan.

## Output

```go
type GetRepairPlanResult struct {
	SaveSessionID string            `json:"saveSessionID"`
	SaveRevision  string            `json:"saveRevision"`
	CharacterID   int               `json:"characterID"`
	PlanToken     string            `json:"planToken"`
	Actions       []RepairAction    `json:"actions"`
	Rejected      []RepairRejection `json:"rejected"`
}

type RepairAction struct {
	IssueIDs    []string                        `json:"issueIDs"`
	Scope       string                          `json:"scope"`
	Operation   string                          `json:"operation"`
	OwnedItemID string                          `json:"ownedItemID,omitempty"`
	TargetValue uint32                          `json:"targetValue,omitempty"`
	Attributes  *saveengine.CharacterAttributes `json:"attributes,omitempty"`
	Description string                          `json:"description"`
}

type RepairRejection struct {
	IssueID string `json:"issueID"`
	Code    string `json:"code"`
	Scope   string `json:"scope"`
	Reason  string `json:"reason"`
}
```

`Actions` and `Rejected` together account for every requested identifier exactly
once. A finding is planned or it is refused; it is never omitted, so an absent
action can never be misread as "already fine" or "silently handled".

`IssueIDs` is a list because an action may be **shared**. The statistics block is
written by exactly one confirmed operation over all eight attributes, so every
attribute finding of one character hangs off a single `set_character_stats`
action; planning them separately would mean two competing writes to the same
block. A container action always names exactly one finding. Each requested
identifier still points at exactly one action or one rejection.

## The plan token

`PlanToken` is the SHA-256 digest, hex-encoded, of the session identifier, the
save revision, the character, and every action in order.

It is derived rather than random on purpose. `ApplyRepairs` can derive the plan
itself, recompute the token and compare, so a plan needs no server-side storage,
no expiry and no reservation. Any change to the save, the character, the
selection or a single action target produces a different token, and a token can
therefore never authorise a mutation that was not the one described.

## Planned repairs

These are the findings whose target state confirmed data determines uniquely.

| Issue code | Operation | Target |
|---|---|---|
| `quantity_zero` | `remove_owned_item` | The record occupying a container slot with quantity `0` is removed. |
| `quantity_above_stack_limit` | `set_owned_item_quantity` | The quantity is set to the confirmed per-record limit of the item in that container, re-derived through the same `containerLimits` rule that produced the finding. |
| `attribute_out_of_range`, `attribute_below_class_minimum` | `set_character_stats` | One shared action carrying the nearest attribute set satisfying both the absolute range `1..99` and the per-attribute minima of the character's own starting class, each attribute moved the smallest legal distance. |

The per-record limit is re-derived from the facts rather than carried on the
issue, because the report states what is wrong and nothing about how to resolve
it: a limit is a repair target, not a defect.

### The statistics action and its unavoidable consequence

The corrected attribute set comes from `saveengine.LegalAttributesFor`, which
owns both the absolute range and the ten confirmed class minima. The plan keeps
no second copy of either, so it can never propose a set `SetCharacterStats` would
reject. An unknown starting class carries no confirmed minima and is therefore
refused, not guessed.

Applying the action **also moves the level and the lifetime runes**. This is not
a choice the plan makes: `SetCharacterStats` accepts exactly one `levelPolicy`,
`recalculate`, and always writes the level derived from the attributes into both
PlayerGameData and the ProfileSummary, then raises `TotalGetSoul` to that level's
minimum (it is never lowered). There is no contract in this build that writes
attributes without doing so. The action description states this explicitly rather
than leaving it to be discovered on apply.

## Refused repairs

| Issue code | Why no plan |
|---|---|
| `unresolved_item`, `unknown_item`, `unresolved_equipped_spell` | This build cannot resolve the stored data. Deriving a repair would mean guessing what it was meant to be, and unknown data must never become a deletion or a different valid item. |
| `quantity_above_container_limit` | The finding names a container total, not one record. No confirmed rule selects which of the records holding that item is reduced. |
| `item_not_allowed_in_container` | No confirmed rule states whether an item stored in a container that does not accept it is moved or destroyed. |
| `memory_slots_exceeded` | No confirmed rule selects which of the equipped spells is unequipped to fit the available capacity. |
| `level_mismatch`, `soul_memory_below_minimum` | Neither value has a repair contract of its own. SaveEngine writes the stored level and the lifetime runes only as derived consequences of `SetCharacterStats`. Whether this build may rewrite a stored level to match its attributes is an unresolved contract decision: SaveForge 1.5.8 and 1.6.8 both rated the mismatch a *warning* and marked it explicitly not automatically repairable, while 2.0 rates it an error. Until that is settled, a plan does not settle it. A selected attribute repair still moves both values as the side effect described above. |
| `dangling_equipment_reference`, `reserved_spell_position_occupied` | Clearing the offending position is not yet a confirmed repair contract of this build. |

A refusal is a result, not an error. The plan is returned with the remaining
actions and every refusal listed beside them.

## Errors

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `gameCatalog` is `nil` | `game catalog is not available` — likewise a wiring error. |
| `saveRevision` is empty | `saveRevision is required`. Checked before the session is read. |
| `issueIDs` is empty | `issueIDs is required and must name at least one finding`. |
| `issueIDs` contains an empty string | `issueIDs must not contain an empty identifier`. |
| `issueIDs` repeats an identifier | `issueIDs repeats "<id>"`. |
| `saveSessionID` is empty, unknown or closed | The SaveEngine error, unchanged. No session is ever resolved to a different one. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. |
| `saveRevision` is not the current revision | `saveRevision "<given>" does not match the current revision "<current>" of session <id>`. |
| The slot is inactive or residual | `character <id> is not active, so it has nothing to repair`. Residual data of a deleted character is never planned against. |
| An identifier names no current finding | `issueIDs [...] name no finding of the current report of character <id> at revision <rev>`. The unknown identifiers are listed in a stable order. |

## Dependencies

- The endpoint reads `backend/saveengine` through `GetSaveValidationFacts` only,
  and joins those facts with `backend/gamecatalog` through the same
  `ItemByGameID` lookup the report uses.
- It shares the report builder with `GetSaveValidationReport` rather than
  re-running it, so the plan and the findings it explains always come from one
  lock, one read and one revision. Two derivations of the same save can never
  disagree.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is
  greenfield.

## Command-line verification

`GetRepairPlan` is verified through its tests. From the repository root:

```bash
go test ./backend/endpoints/diagnostics -run '^TestGetRepairPlan' -count=1 -v
go test ./backend/endpoints/diagnostics -run '^TestSaveValidationReport_IssueIdentity' -count=1 -v
go test ./backend/saveengine -run '^TestLegalAttributesFor' -count=1 -v
go test ./tools/swagger -run 'RepairPlanRoute' -count=1 -v
go test -race ./backend/endpoints/diagnostics -count=1
```

The tests build synthetic PC containers inside `t.TempDir()` and resolve against
the stored catalog data, so every limit they assert is the real document rather
than a local copy of itself. They use no real save file.

They cover the identifier contract the plan addresses findings by, every planned
repair with its exact target values, both attribute findings collapsing into one
shared `set_character_stats` action rather than two competing writes, an unknown
starting class producing no write at all, refusal of every unresolvable and
policy-dependent finding, binding to the save revision including a revision a
real mutation advanced past, every input rejection, an inactive slot, the token
being stable across request order and different across a different selection, and
that building a plan for real defects moves neither the session revision nor the
unsaved-changes flag.

`TestLegalAttributesFor` in `backend/saveengine` covers the rule application
below the endpoint boundary: a legal set left untouched, each attribute moving
the smallest legal distance with the class minimum winning where it is stricter,
every one of the ten classes summing high enough that a corrected set is always
writable, and an unknown class rejected.

## Current limitations

- The only transport is the local explorer route
  `POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/repair-plan`,
  and it exists only while the explorer runs without `-allow-external-bind`.
  No Wails binding, no CLI command and no frontend reaches the endpoint.
- Nothing executes a plan. `ApplyRepairs` remains contract-only, so this build
  can describe a repair but not perform one.
- Six of the fourteen issue codes carry no repair contract. Extending the planned
  set is a separate task per code, each needing the confirmed rule that makes its
  target state unique.
- `quantity_above_container_limit` is refused partly because 2.0 and 1.x disagree
  about what `maxInventory`/`maxStorage` means. 1.5.8 and 1.6.8 treated it as a
  per-record cap only; 2.0 also enforces it as a cross-record container total, in
  `AddItemToInventory` and therefore in the report. Resolving that divergence is
  tracked in `tmp/app-se/TODO.md` and is not this endpoint's to settle.
- It plans one character slot per call, matching the report it consumes.
