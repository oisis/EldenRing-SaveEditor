# GetBuildTemplates

## Overview

`GetBuildTemplates` returns one page of lightweight metadata for Build Templates
stored in the user's local template library. It reads exclusively from the
library index file (`_index.json`) and does not load or parse full template
payload files.

The getter accepts no `saveSessionID` and no `characterID`, and does not access
`SaveEngine` or `GameCatalog`. It is strictly read-only: it never creates the
library directory, writes or updates `_index.json`, rebuilds an index, or
modifies file timestamps.

| | |
|---|---|
| EndpointID | `get_build_templates` |
| Kind | Getter |
| Domain | `templates` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/build-templates` of the local OpenAPI explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`. There is no Wails binding, no frontend view and no CLI command. |
| Implementation source | [../../../backend/endpoints/templates/get_build_templates.go](../../../backend/endpoints/templates/get_build_templates.go) |
| Test source | [../../../backend/endpoints/templates/get_build_templates_test.go](../../../backend/endpoints/templates/get_build_templates_test.go) |
| Data source | local `_index.json` metadata file in the user config directory (`$UserConfigDir/EldenRing-SaveEditor/templates/`), read by `buildtemplates.Store` |
| Save access | none — this getter does not interact with save files or `SaveEngine` |
| Mutation | none — strictly read-only; no files or directories are created or modified |

## Input

```go
func GetBuildTemplates(
	store *buildtemplates.Store,
	search string,
	tags []string,
	page int,
	pageSize int,
) (GetBuildTemplatesResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `store` | `*buildtemplates.Store` | The templates store instance that reads the library directory. A `nil` store is rejected. |
| `search` | `string` | Optional case-insensitive substring search matched against template `name` and `description`. Omitted or empty string means no search filtering. |
| `tags` | `[]string` | Optional list of tags. When non-empty, every tag in the list must be present on the template (AND semantics). Tags are matched case-sensitively and exactly. An empty tag element is rejected. |
| `page` | `int` | 1-based page number. Must not be negative. Passing `0` defaults to page `1`. |
| `pageSize` | `int` | Number of entries per page. Must not be negative. Passing `0` defaults to `50` (`GetBuildTemplatesDefaultPageSize`). |

### `search`

- Case-insensitive substring match against `name` and `description`.
- Never trimmed or modified before matching.

### `tags`

- Nil or empty slice performs no tag filtering.
- Every tag in `tags` must be present in the template's `tags` slice (AND semantics).
- Tags are compared exactly and case-sensitively.
- An empty tag element (e.g. `""`) is rejected as an error.

### `page` and `pageSize`

- `page < 0` or `pageSize < 0` is rejected with an error.
- `page == 0` defaults to `1`.
- `pageSize == 0` defaults to `50`.
- If `page` exceeds the available pages, the endpoint returns an empty `templates` slice (never `nil`) along with the correct `total`, `page`, and `pageSize`.

## Result

```go
type BuildTemplateEntry struct {
	TemplateID       string   `json:"templateID"`
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
	SchemaVersion    int      `json:"schemaVersion,omitempty"`
	SelectedSections []string `json:"selectedSections,omitempty"`
	InventoryItems   int      `json:"inventoryItems"`
	StorageItems     int      `json:"storageItems"`
	Warnings         int      `json:"warnings"`
	TemplateRevision string   `json:"templateRevision"`
}

type GetBuildTemplatesResult struct {
	Templates []BuildTemplateEntry `json:"templates"`
	Total     int                  `json:"total"`
	Page      int                  `json:"page"`
	PageSize  int                  `json:"pageSize"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `templates` | `[]BuildTemplateEntry` | Matching template metadata entries in the requested page. Never `nil`. |
| `templates[].templateID` | `string` | Unique identifier of the template. |
| `templates[].name` | `string` | User-facing display name of the template. |
| `templates[].description` | `string` | Optional description of the template. Omitted when empty. |
| `templates[].tags` | `[]string` | Tag list associated with the template. Omitted when empty. |
| `templates[].createdAt` | `string` | RFC3339 creation timestamp. |
| `templates[].updatedAt` | `string` | RFC3339 last update timestamp. |
| `templates[].schemaVersion` | `int` | Schema version of the template (e.g. `1` or `2`). |
| `templates[].selectedSections` | `[]string` | Sections included in the template (e.g. `["inventory.workspace"]`). Omitted when empty. |
| `templates[].inventoryItems` | `int` | Count of inventory items in the template. |
| `templates[].storageItems` | `int` | Count of storage items in the template. |
| `templates[].warnings` | `int` | Count of warnings recorded for the template. |
| `templates[].templateRevision` | `string` | Opaque generation token of the library entry, always present. It is the canonical decimal form of the persistent per-template revision counter stored in `_index.json`, and it is the same token [`GetBuildTemplate`](get_build_template.md) returns for that `templateID`. An entry written before the counter existed carries no revision field and reports `"0"`, which is a valid token and not a missing value. The revision is not part of the portable template document. |
| `total` | `int` | Total number of templates matching the search and tag filters before paging. |
| `page` | `int` | The effective 1-based page number. |
| `pageSize` | `int` | The effective page size. |

### What is not returned

The result contains only index metadata. It returns:
- no `filename` or relative file paths;
- no `rootDir` or absolute filesystem paths;
- no item definitions, weapon details, stats, or full template JSON payload contents.

## Storage and index handling

The library index is read from `_index.json` in the library directory:

- **Missing library directory or missing `_index.json`**: Treated as an empty library; returns `total: 0`, `templates: []`, with no error and without creating any file or directory.
- **Malformed `_index.json`**: Hard fail-closed error. The getter never runs an automatic rebuild.
- **Unsupported index version**: If `_index.json` declares a version other than `1`, the getter fails closed with an error.
- **Sort order**: Entries are sorted by `updatedAt` descending (newest first). When `updatedAt` is identical, a deterministic tie-break is performed on `templateID` ascending.

## Verification

Focused tests:
```bash
go test ./backend/buildtemplates/... -v -count=1
go test ./backend/endpoints/templates -run '^TestGetBuildTemplates' -v -count=1
go test ./tools/swagger -run '^TestGetBuildTemplates|^TestOpenAPI' -v -count=1
```
