# GetSaveValidationReport

## Overview

`GetSaveValidationReport` runs one non-mutating validation pass over a single
character slot of a save session that already exists in SaveEngine and returns
the problems it found.

It reports defects. It does not resolve them: this getter emits no action, no
default action and no repair proposal. Proposing a repair belongs to
`GetRepairPlan`, and performing one belongs to `ApplyRepairs`; both are still
contract-only.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetSaveValidationReport` never
creates one, so calling it before a successful `LoadSave` is an error, not an
implicit load. The endpoint opens no source file, returns no raw save byte, and
modifies nothing: neither the save, nor the session, nor the catalog, nor any
application state.

| | |
|---|---|
| EndpointID | `get_save_validation_report` |
| Kind | Getter |
| Domain | `diagnostics` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/validation-report` of the local explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. There is no Wails binding, no CLI command and no frontend. |
| Implementation source | [../../../backend/endpoints/diagnostics/get_save_validation_report.go](../../../backend/endpoints/diagnostics/get_save_validation_report.go) |
| Test source | [../../../backend/endpoints/diagnostics/get_save_validation_report_test.go](../../../backend/endpoints/diagnostics/get_save_validation_report_test.go) |
| Save access | read-only — the session's private in-memory snapshot, read under one lock and one save revision; no file is opened |
| Catalog access | read-only — one `ItemByGameID` lookup per resolved container record and per occupied spell record |
| Mutation | none — the snapshot, the session, the catalog, and the save file are left unchanged |

## Input

```go
func GetSaveValidationReport(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	scope string,
) (GetSaveValidationReportResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; the endpoint never creates one. A `nil` engine is rejected. |
| `gameCatalog` | `*gamecatalog.Catalog` | The catalog the container records and the equipped spells are judged against. A `nil` catalog is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. It is passed to SaveEngine unchanged. |
| `characterID` | `int` | The physical slot index, `0` to `9`. It is the same index `GetSaveCharacters` reports positionally. |
| `scope` | `string` | One validation scope, or the empty string for all of them. |

### `saveSessionID`

- It is matched exactly and case-sensitively. It is never trimmed, normalised,
  or guessed, so `" <id>"`, `"<id> "`, and an upper-cased identifier are unknown
  values, not the session they resemble.
- Validation lives in SaveEngine. The endpoint holds no session-identifier rule
  of its own.

### `characterID`

- It is an index, not an identifier to search for: slot `n` is read directly.
- A value below `0` or above `9` is rejected. It is never clamped to the valid
  range and never resolved to a neighbouring slot.

### `scope`

- Accepted values are `inventory`, `storage`, `stats`, `equipment` and `spells`.
  The empty string means all five.
- It is matched exactly. It is never trimmed or lower-cased, so `"Inventory"` is
  an unknown scope, not the scope it resembles, and it is rejected instead of
  silently reporting nothing.
- A narrowed scope changes only which findings are reported, never how a scope is
  judged. The whole pass is read either way, so two reports can never disagree
  about the same data.

## Output

```go
type SaveValidationIssue struct {
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Scope       string `json:"scope"`
	Message     string `json:"message"`
	OwnedItemID string `json:"ownedItemID"`
}

type SaveValidationScopeCoverage struct {
	Scope             string `json:"scope"`
	Checked           bool   `json:"checked"`
	Reason            string `json:"reason"`
	RecordsChecked    int    `json:"recordsChecked"`
	UnresolvedRecords int    `json:"unresolvedRecords"`
}

type GetSaveValidationReportResult struct {
	SaveSessionID string                        `json:"saveSessionID"`
	SaveRevision  string                        `json:"saveRevision"`
	CharacterID   int                           `json:"characterID"`
	Active        bool                          `json:"active"`
	Coverage      []SaveValidationScopeCoverage `json:"coverage"`
	Issues        []SaveValidationIssue         `json:"issues"`
	ErrorCount    int                           `json:"errorCount"`
	WarningCount  int                           `json:"warningCount"`
}
```

`Issues` are ordered by scope in the order `Coverage` lists them, and inside a
scope in the stored order of the data they were found in, so two reports of the
same revision are identical.

`SaveRevision` is the revision the whole pass was read under. Every scope
describes that one revision, because SaveEngine gathers all facts under a single
lock; a concurrent mutation cannot land between two scopes and leave a report
that mixes two states.

`OwnedItemID` identifies the container record a record-scoped issue was found in.
It is valid for `SaveRevision` and for nothing else. It is empty for an issue
that names no record.

An inactive or residual slot reports `active` false, no issue, and every
requested scope unchecked with that reason.

## Coverage

`Coverage` exists so an empty `Issues` list can be read correctly.

- `checked` true means the scope was decoded and judged.
- `checked` false with a `reason` means the scope could not be decoded at all.
  Its data was therefore not judged, and the save is **not** thereby clean.
- `unresolvedRecords` counts records inside a checked scope whose stored data
  this build could not resolve. Those records were reported as warnings and not
  judged against limits that were never established for them.

A scope that cannot be decoded never fails the whole report. The remaining scopes
are still judged, and the undecodable one is reported as a gap.

## Checks

### Scope `inventory` and scope `storage`

Each stored record of the container is resolved through the GaItem table and then
judged against the same GameCatalog limits the mutating endpoints enforce.

| Code | Severity | Meaning |
|---|---|---|
| `unresolved_item` | warning | The stored GaItem handle resolves to no record, or the resolved item carries no confirmed limit for this container. Nothing is invented for it. |
| `unknown_item` | warning | The handle resolved to a game ID GameCatalog does not know. |
| `quantity_zero` | error | A non-empty record stores the quantity `0`. |
| `quantity_above_stack_limit` | error | One record holds more than `min(maxPerStack, container total)`, or more than one instance of a `separate_instances` item. |
| `quantity_above_container_limit` | error | The sum of one game ID across the container exceeds `maxInventory` or `maxStorage`. Judged once per item, never once per record. |
| `item_not_allowed_in_container` | error | The item's confirmed limit for this container is `0`, so it does not belong there at all. |

### Scope `stats`

The eight attributes, the stored level, the stored lifetime runes and the
starting class are read, and the expected values are derived by the same
functions `SetCharacterStats` enforces. The report therefore can never accept a
state the setter rejects, or reject a state the setter accepts.

| Code | Severity | Meaning |
|---|---|---|
| `attribute_out_of_range` | error | An attribute lies outside `1..99`, or the level the formula derives lies outside the legal range. Without a legal attribute set there is no expected level, so no level mismatch is reported on top of it. |
| `level_mismatch` | error | The stored level differs from `sum(attributes) - 79`. |
| `attribute_below_class_minimum` | error | An attribute is below the base value of the character's own starting class, or the stored starting class is not one of the ten confirmed classes and therefore carries no known minima. |
| `soul_memory_below_minimum` | error | The stored lifetime runes are below the minimum the recalculated level requires. |

### Scope `equipment`

Every stored `{GaItem handle, InventoryHeld common row}` pair of the slot is
decoded — the 22 Equipment fields, the 10 Quick Item pairs and the 6 Pouch pairs
— and compared with the row it names.

| Code | Severity | Meaning |
|---|---|---|
| `dangling_equipment_reference` | error | The referenced common row is empty, or it carries a different handle than the pair states. |

The invalid-row sentinel and every row below the InventoryHeld base mean the slot
references nothing and are not findings. This is the same rule `RemoveOwnedItem`
relies on when it refuses to empty a referenced row.

### Scope `spells`

| Code | Severity | Meaning |
|---|---|---|
| `reserved_spell_position_occupied` | error | One of the two physical positions the game keeps reserved (13 and 14) is not empty. |
| `unresolved_equipped_spell` | warning | An occupied identifier is not a raw MagicParam ID, or does not resolve to a known spell with a confirmed Memory Slots cost. |
| `memory_slots_exceeded` | error | The equipped spells consume more memory than the capacity SaveEngine derived from the slot. |

An unresolved spell is left out of the memory sum and suppresses the capacity
check, so unknown data can never create a capacity error of its own.

## Deliberate non-checks

The following defects the 1.x repair scanner reported are **not** checked here,
and each exclusion is a decision, not an omission:

- **Duplicate GaItem handles.** SaveForge 2.0 confirms that one handle may
  legitimately be shared by several records, which `RemoveOwnedItem` documents
  and depends on. Reporting a shared handle would classify known-good native data
  as corrupt.
- **Duplicate acquisition indices.** SaveForge 1.5.8 matched indices exactly and
  1.6.8 replaced that with a mixed-range bucket rule. Both describe the index
  allocator of the 1.x editor, which SaveForge 2.0 does not reproduce; applied to
  native data alone, both produce false positives.
- **GaItem repack, allocator and container-overuse findings.** That behaviour was
  retired. Reviving it requires new native evidence and explicit approval, not a
  new report.

## Errors

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `gameCatalog` is `nil` | `game catalog is not available` — likewise a wiring error. |
| `scope` is not one of the five accepted values | `scope must be one of [inventory storage stats equipment spells] or empty; got "<scope>"`. Checked before the session is read. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown or closed | `unknown save session "<id>"`. A closed or never-created session is never resolved to a different one. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. Checked only after the session resolves. |
| The activity flag of the slot cannot be read | `cannot read activity of character <id>: …`. |

Everything else is a finding, not an error. A slot whose inventory, storage,
statistics, references or spell records cannot be decoded produces an unchecked
coverage entry naming the reason, and the remaining scopes are still judged.

An inactive or residual slot is not in this table: it is a successful result.

## Dependencies

- The endpoint delegates to `backend/saveengine` through
  `GetSaveValidationFacts` and reads `backend/gamecatalog` through
  `ItemByGameID` and `ValidateSpellResource` only. It calls no other endpoint.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is
  greenfield.

## Command-line verification

`GetSaveValidationReport` is verified through its tests. From the repository
root:

```bash
go test ./backend/saveengine -run '^TestGetSaveValidationFacts' -count=1 -v
go test ./backend/endpoints/diagnostics -count=1 -v
go test ./tools/swagger -run '^TestSaveValidationReportRoute' -count=1 -v
```

The tests build synthetic PC and PS4 containers inside `t.TempDir()`. They use no
real save file and no repository fixture, so they depend on nothing outside the
test process. The endpoint tests resolve against the stored catalog data, so
every limit they assert is the real document rather than a local copy of itself.

The SaveEngine tests cover a known-good slot on both platforms, an inactive slot,
an unresolvable handle beside a resolvable one, all three reference shapes plus a
key-section record that must not satisfy a common-row reference, every statistics
rule, the reserved spell positions, a spell record pair the game never writes,
and that the snapshot, the revision and the unsaved-changes flag do not move.

The endpoint tests cover a clean report on both platforms, an inactive slot whose
scopes are all reported unchecked, every container limit, unknown data staying a
warning with zero errors, every statistics code, the dangling reference, all
three spell codes, scope narrowing and rejection, a `nil` engine, a `nil`
catalog, an unknown session, an invalid `characterID`, and that two consecutive
reports of the same session are byte-for-byte identical.

## Current limitations

- The only transport is the local explorer route
  `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/validation-report`,
  and it exists only while the explorer runs without `-allow-external-bind`.
  No Wails binding, no CLI command and no frontend reaches the endpoint.
- It reports one character slot per call. Scanning a whole save is a loop on the
  caller's side, so a report never has to decide what an inactive slot means in
  the middle of a combined result.
- It reports defects only. `GetRepairPlan` and `ApplyRepairs` remain
  contract-only, so nothing in this build proposes or performs a repair.
