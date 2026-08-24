# ApplyBuildTemplate

## Overview

`ApplyBuildTemplate` builds a full execution plan for a Build Template from the
local templates library and atomically applies the confirmed target values
(profile name, statistics, and equipped spells) to the specified character in an
active save session.

The endpoint executes in two stages:

1. **Deterministic Planning & Preflight**: evaluates the template against live
   save data, selection narrowing, and `GameCatalog` using the shared
   `planBuildTemplate` planner. If any blocking issue is detected, the request
   is rejected without mutating save or session state.
2. **Atomic SaveEngine Mutation**: if the plan is executable and
   `expectedRevision` matches the save revision of the plan, delegates all
   target byte writes to a single atomic `SaveEngine.ApplyCharacterTemplate`
   operation under `Engine.mutex`. Exactly one `saveRevision` is advanced and
   at most one undo point (`opApplyBuildTemplate`) is recorded.

| | |
|---|---|
| EndpointID | `apply_build_template` |
| Kind | Mutation |
| Domain | `templates` |
| Implementation status | implemented |
| Transport status | transport-exposed — `POST /api/v1/build-templates/{templateID}/apply` of the local OpenAPI explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`. |
| Implementation source | [../../../backend/endpoints/templates/apply_build_template.go](../../../backend/endpoints/templates/apply_build_template.go) |
| Test source | [../../../backend/endpoints/templates/apply_build_template_test.go](../../../backend/endpoints/templates/apply_build_template_test.go) |
| Data sources | `buildtemplates.Store`, `saveengine.Engine`, `gamecatalog.Catalog` |
| Save access | Read-write; protected by `expectedRevision` control and atomic commit |
| Mutation | Atomic character profile name, statistics, and equipped spells mutation |

## Input

```go
func ApplyBuildTemplate(
	store *buildtemplates.Store,
	engine *saveengine.Engine,
	catalog *gamecatalog.Catalog,
	req ApplyBuildTemplateRequest,
) (ApplyBuildTemplateResult, error)
```

```go
type ApplyBuildTemplateRequest struct {
	SaveSessionID    string                            `json:"saveSessionID"`
	CharacterID      int                               `json:"characterID"`
	TemplateID       string                            `json:"templateID"`
	Selection        *buildtemplates.TemplateSelection `json:"selection,omitempty"`
	Options          *buildtemplates.ApplyOptions      `json:"options,omitempty"`
	ExpectedRevision string                            `json:"expectedRevision"`
}
```

| Parameter | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Required identifier of an active loaded save session. |
| `characterID` | `int` | Target character slot index in the range `0..9`. Must be active. |
| `templateID` | `string` | Unique identifier of the template in the local library. |
| `selection` | `*TemplateSelection` | Optional selection narrowing. If omitted (`nil`), the template's own `Selection` is used. |
| `options` | `*ApplyOptions` | Optional application settings (`items`, `inventoryLayout`, `storageLayout`, `weaponLevelOverride`). Non-empty unsupported fields produce a blocking issue. |
| `expectedRevision` | `string` | Canonical decimal save revision expected before mutation. |

### Scope & Invariants

1. **Shared planner with `GetBuildTemplatePreview`**:
   - Uses the exact same planner (`planBuildTemplate`) for loading, selection narrowing, options validation, and preflight rules.
   - Any blocking issue (`executable == false`) prevents mutation entirely.
2. **Supported Sections**:
   - `profile.name`: writes both 16-unit UTF-16LE copies (PlayerGameData and ProfileSummary).
   - `stats`: writes the 8 attributes, recalculated level (`sum(attributes) - 79`), SoulMemory (TotalGetSoul raised to the minimum the levels above the base level of the character's own starting class require, never lowered), and ProfileSummary level. Attributes below starting-class minima are rejected.
   - `spells`: replaces the first 12 spell memory positions with the validated compact sequence and updates active index. Physical positions 13-14 must be empty in the native save and remain byte-for-byte untouched.
3. **Unsupported Sections & Options**:
   - Selecting `inventory.workspace`, `equipment`, `items`, `inventoryLayout`, `storageLayout`, or any non-empty `ApplyOptions` produces a blocking issue and aborts the request before any byte is written.

## Result

```go
type ApplyBuildTemplateResult struct {
	TemplateID       string                   `json:"templateID"`
	TemplateRevision string                   `json:"templateRevision"`
	SaveSessionID    string                   `json:"saveSessionID"`
	SaveRevision     string                   `json:"saveRevision"`
	CharacterID      int                      `json:"characterID"`
	Plan             BuildTemplatePreviewPlan `json:"plan"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `templateID` | `string` | Identifier of the applied template. |
| `templateRevision` | `string` | Revision of the template in the library used during application. |
| `saveSessionID` | `string` | Identifier of the save session. |
| `saveRevision` | `string` | New save revision advanced by the atomic mutation. |
| `characterID` | `int` | Character slot index. |
| `plan` | `BuildTemplatePreviewPlan` | The exact execution plan that was applied. |

## HTTP Transport

When hosted in `tools/swagger`:

- **Route**: `POST /api/v1/build-templates/{templateID}/apply`
- **Loopback-only**: Available only in local loopback mode. An explorer started with `-allow-external-bind` does not register this route and returns 404.
- **Status codes**:
  - `200 OK`: Template successfully applied; returns mutation receipt with new revision and applied plan.
  - `400 Bad Request`: Malformed JSON body, missing required fields, invalid character slot range, inactive character, or non-executable plan.
  - `404 Not Found`: Template or save session not found.
  - `409 Conflict`: `expectedRevision` mismatch or save revision changed concurrently during planning/application.
