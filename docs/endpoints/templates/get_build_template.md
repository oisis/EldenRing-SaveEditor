# GetBuildTemplate

## Overview

`GetBuildTemplate` returns one complete Build Template from the local templates
library by its unique `templateID`. It resolves the template file exclusively
through `_index.json`, decodes the payload fail-closed, and enforces all
structural schema invariants before returning the portable template document.

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
) (*buildtemplates.BuildTemplate, error)
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

The endpoint returns the complete portable `BuildTemplate` document:

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
