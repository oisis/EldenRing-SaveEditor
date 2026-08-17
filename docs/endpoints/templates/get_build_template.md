# GetBuildTemplate

## Overview

`GetBuildTemplate` returns one complete Build Template from the local templates
library by its unique `templateID`. It resolves the template file exclusively
through `_index.json`, decodes the payload fail-closed, and enforces all
structural schema invariants before returning the portable template document
together with the library generation token `templateRevision`.

The getter accepts no `saveSessionID` and no `characterID`, and does not access
`SaveEngine` or `GameCatalog`. It is strictly read-only: it never creates the
library directory, writes or updates `_index.json`, rebuilds an index, or
modifies files.

| | |
|---|---|
| EndpointID | `get_build_template` |
| Kind | Getter |
| Domain | `templates` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/build-templates/{templateID}` of the local OpenAPI explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`. There is no Wails binding, no frontend view and no CLI command. |
| Implementation source | [../../../backend/endpoints/templates/get_build_template.go](../../../backend/endpoints/templates/get_build_template.go) |
| Test source | [../../../backend/endpoints/templates/get_build_template_test.go](../../../backend/endpoints/templates/get_build_template_test.go) |
| Data source | local template payload file resolved via `_index.json` in the user config directory (`$UserConfigDir/EldenRing-SaveEditor/templates/`), read by `buildtemplates.Store` |
| Save access | none — this getter does not interact with save files or `SaveEngine` |
| Mutation | none — strictly read-only; no files or directories are created or modified |

## Input

```go
func GetBuildTemplate(
	store *buildtemplates.Store,
	templateID string,
) (GetBuildTemplateResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `store` | `*buildtemplates.Store` | The templates store instance that reads the library directory. A `nil` store is rejected. |
| `templateID` | `string` | The unique identifier of the template in `_index.json`. Must not be empty. |

### `templateID`

- Resolved exclusively through `_index.json`. It is never interpreted as a filename directly.
- An empty `templateID` is rejected as an error.
- An unknown `templateID` or a missing library returns `ErrNotFound` (transported as HTTP 404).

## Result

The endpoint returns a wrapper carrying the complete portable `BuildTemplate`
document and the library generation token of the template:

```go
type GetBuildTemplateResult struct {
	Template         *BuildTemplate `json:"template"`
	TemplateRevision string         `json:"templateRevision"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `template` | `*BuildTemplate` | The complete decoded and validated portable template document described below. |
| `templateRevision` | `string` | Opaque generation token of the template in the local library. |

Every error path returns the zero `GetBuildTemplateResult` (`template` is `nil`
and `templateRevision` is empty) together with the error.

### `templateRevision`

- It is an **opaque generation token of the template**, not a semantic version,
  not a timestamp and not a content hash. Callers must treat it as a string to
  be echoed back, never parsed, compared as a number, or ordered.
- On the wire it is always a JSON string, never a JSON number.
- It is the canonical decimal form of a persistent, monotonic per-template
  counter stored in the template's `_index.json` entry.
- One generation is the state committed by the Store writers, covering both the
  `_index.json` entry and the payload file it points to. A future update
  increments the counter for a metadata change and for a payload content change
  alike. Because the token is not a content hash, it does not detect edits made
  to the library outside the Store.
- An entry written before the counter existed carries no revision field and
  therefore reports `"0"`. `"0"` is a valid token, not a missing value.
- **The revision is not part of the portable template document.** It never
  appears in `BuildTemplate`, in `TemplateDocMetadata`, or in the payload file
  on disk. It belongs to the local library, so a template exported from one
  library and imported into another does not carry its revision with it.

### `template`

The `template` field carries the complete portable `BuildTemplate` document:

```go
type BuildTemplate struct {
	Schema       string               `json:"schema"`
	Version      int                  `json:"version"`
	CreatedAt    string               `json:"createdAt"`
	AppVersion   string               `json:"appVersion,omitempty"`
	Metadata     *TemplateDocMetadata `json:"metadata,omitempty"`
	Selection    *TemplateSelection   `json:"selection,omitempty"`
	Sections     TemplateSections     `json:"sections"`
	ApplyOptions *ApplyOptions        `json:"applyOptions,omitempty"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `schema` | `string` | Schema identifier string (`saveforge.build-template`). |
| `version` | `int` | Supported schema version (`1` or `2`). |
| `createdAt` | `string` | RFC3339 creation timestamp. |
| `appVersion` | `string` | Optional version string of the SaveForge instance that built the template. |
| `metadata` | `*TemplateDocMetadata` | Optional human-readable metadata (`name`, `description`, `author`, `tags`, `sourceCharacterIndex`, `sourceCharacterName`). |
| `selection` | `*TemplateSelection` | Section/field selection object (required for v2). |
| `sections` | `TemplateSections` | Payload sections: `inventory.workspace` (v1), `profile`, `stats`, `equipment`, `spells`, `items`, `inventoryLayout`, `storageLayout` (v2). |
| `applyOptions` | `*ApplyOptions` | Optional apply configuration (`items`, `inventoryLayout`, `storageLayout`, `weaponLevelOverride`). |

## Error handling

- **HTTP 404 (Not Found)**:
  - Unknown `templateID` not present in `_index.json`.
  - Missing library directory or missing `_index.json`.
  - Target payload file listed in `_index.json` does not exist on disk.
- **HTTP 400 (Bad Request)**:
  - Empty `templateID`.
  - Corrupt or malformed `_index.json` (syntax error, unsupported version, duplicate template IDs).
  - Malformed payload JSON or unknown JSON fields (fail-closed decoder).
  - Schema constraint violations (invalid ranges, invalid item IDs, invalid slot keys, duplicate entry IDs, etc.).
  - Metadata mismatch between `_index.json` entry and payload file.
  - Path traversal attempts or symlinks escaping the store directory.
