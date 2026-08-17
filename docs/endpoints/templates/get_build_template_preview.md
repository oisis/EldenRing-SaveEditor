# GetBuildTemplatePreview

## Overview

`GetBuildTemplatePreview` builds a non-mutating preview plan of applying a Build
Template from the local library to a specified character in an active save
session.

It evaluates the template's target values against the character's live data
(profile, statistics, and equipped spells) and produces a deterministic diff
plan, reporting whether the plan is executable (`executable: true`) or
prevented by blocking issues (`executable: false`).

The getter is strictly non-mutating:

- It never modifies the save file, session snapshot, `saveRevision`, `dirty`
  flag, undo state, or `OwnedItemID` assignments;
- It never modifies, creates, or deletes template library files or `_index.json`;
- Save reads are protected by `saveRevision` verification before and after
  reading, returning `ErrSaveRevisionConflict` (HTTP 409) on concurrent mutation.

| | |
|---|---|
| EndpointID | `get_build_template_preview` |
| Kind | Getter |
| Domain | `templates` |
| Implementation status | implemented |
| Transport status | transport-exposed — `POST /api/v1/build-templates/{templateID}/preview` of the local OpenAPI explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`. |
| Implementation source | [../../../backend/endpoints/templates/get_build_template_preview.go](../../../backend/endpoints/templates/get_build_template_preview.go) |
| Test source | [../../../backend/endpoints/templates/get_build_template_preview_test.go](../../../backend/endpoints/templates/get_build_template_preview_test.go) |
| Data sources | `buildtemplates.Store`, `saveengine.Engine`, `gamecatalog.Catalog` |
| Save access | Read-only; protected by session revision check before and after reading |
| Mutation | none — strictly read-only |

## Input

```go
func GetBuildTemplatePreview(
	store *buildtemplates.Store,
	engine *saveengine.Engine,
	catalog *gamecatalog.Catalog,
	req GetBuildTemplatePreviewRequest,
) (GetBuildTemplatePreviewResult, error)
```

```go
type GetBuildTemplatePreviewRequest struct {
	SaveSessionID string                            `json:"saveSessionID"`
	CharacterID   int                               `json:"characterID"`
	TemplateID    string                            `json:"templateID"`
	Selection     *buildtemplates.TemplateSelection `json:"selection,omitempty"`
	Options       *buildtemplates.ApplyOptions      `json:"options,omitempty"`
}
```

| Parameter | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Required identifier of an active loaded save session. |
| `characterID` | `int` | Target character slot index in the range `0..9`. Must be active. |
| `templateID` | `string` | Unique identifier of the template in the local library. |
| `selection` | `*TemplateSelection` | Optional selection narrowing. If omitted (`nil`), the template's own `Selection` is used. |
| `options` | `*ApplyOptions` | Optional application settings (`items`, `inventoryLayout`, `storageLayout`, `weaponLevelOverride`). Non-empty unsupported fields produce `unsupported_option`. |

### Selection Narrowing & Validation Rules

1. **Narrowing-only rule**:
   - When `req.Selection` is provided (`!= nil`), it can only disable or pick a subset of the fields selected by the template's own `Selection`.
   - Selecting a section or field not selected in the template generates a blocking issue with `code: "selection_not_in_template"`.
   - `All: true` in a section of `req.Selection` is only valid if the template also has `All: true` for that section.
   - For v1 templates, the default base selection is `inventory.workspace`; `req.Selection` cannot expand to other sections.
   - An empty selection generates `code: "empty_selection"`.
2. **`profile`**:
   - `name`: compared against current character name.
   - `level`: cannot be applied without `stats`. If selected with `stats`, the template's target level must match the calculated level derived from the final 8 attributes (`level = sum(attributes) - 79`); otherwise generates `code: "level_mismatch"`.
   - `All: true` and unconfirmed profile fields generate `code: "unsupported_field"`.
3. **`stats`**:
   - Evaluates full 8 attributes using `saveengine.PlanCharacterStats`, checking starting-class minima and attribute ranges `1..99`.
   - Any attribute range violation or starting-class minima violation produces `code: "invalid_stats"`.
   - Reports resulting `resultLevel` and `resultSoulMemory`.
4. **`spells`**:
   - Confirms that physical save positions 13 and 14 are empty; if occupied, generates `code: "invalid_spell_loadout"`.
   - Constructs a 12-position state from current and template values, compacts it to remove gaps, and validates the compact loadout against GameCatalog.
   - Validates memory slot costs against character `availableMemorySlots` and checks uniqueness (no duplicates).
   - `Slots` in the plan compares current slots 1..12 with the final compact slots 1..12, ensuring exact alignment with `EquippedSpells`.
   - Invalid loadouts produce `code: "invalid_spell_loadout"`.
5. **Unsupported sections & options**:
   - Any selection of `inventory.workspace`, `equipment`, `items`, `inventoryLayout`, or `storageLayout` produces `code: "unsupported_section"`.
   - Any non-empty field in `tpl.ApplyOptions` or `req.Options` produces `code: "unsupported_option"`.

## Result

```go
type GetBuildTemplatePreviewResult struct {
	TemplateID       string                      `json:"templateID"`
	TemplateRevision string                      `json:"templateRevision"`
	CharacterID      int                         `json:"characterID"`
	SaveSessionID    string                      `json:"saveSessionID"`
	SaveRevision     string                      `json:"saveRevision"`
	Executable       bool                        `json:"executable"`
	Plan             BuildTemplatePreviewPlan    `json:"plan"`
	BlockingIssues   []BuildTemplatePreviewIssue `json:"blockingIssues,omitempty"`
}

type BuildTemplatePreviewIssue struct {
	Code    string `json:"code"`
	Section string `json:"section,omitempty"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}
```

| Stable Issue Code | Meaning |
|---|---|
| `empty_selection` | Selection contains no selected sections or fields |
| `unsupported_section` | Selected section is not supported for template application |
| `unsupported_field` | Selected field is not supported or profile.level was requested without stats |
| `selection_not_in_template` | Request attempts to expand selection beyond template scope |
| `missing_section` | Template payload is missing a section required by selection |
| `missing_value` | Template section is missing a specific required value |
| `unsupported_option` | Non-empty applyOption in template or request has no supported implementation |
| `invalid_stats` | Attribute values violate range 1..99 or starting-class minima |
| `level_mismatch` | Target profile level does not match calculated stats level |
| `invalid_spell_loadout` | Resulting spell loadout violates catalog rules, uniqueness, memory capacity, or physical slots 13-14 are occupied |

## HTTP Transport

When hosted in `tools/swagger`:

- **Route**: `POST /api/v1/build-templates/{templateID}/preview`
- **Loopback-only**: Available only in local loopback mode. An explorer started with `-allow-external-bind` does not register this route and returns 404.
- **Status codes**:
  - `200 OK`: Preview constructed and returned.
  - `400 Bad Request`: Malformed JSON, missing required fields, invalid character slot range, or inactive character.
  - `404 Not Found`: Template or save session not found.
  - `409 Conflict`: Save revision changed during preview construction.
