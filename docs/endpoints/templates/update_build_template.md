# UpdateBuildTemplate

## Overview

`UpdateBuildTemplate` modifies the metadata or contents of an existing Build
Template in the **local templates library**: it updates the template payload file
and its entry in `_index.json` under optimistic concurrency control.

The endpoint is a thin adapter. The whole mutation — revision validation, the
Store lock, index and payload validation, the atomic payload and index writes,
and in-process rollback — belongs to `buildtemplates.Store.UpdateTemplate`,
which is the single owner of those rules.

The library is a purely local directory. This endpoint therefore:

- accepts **no `saveSessionID`** and **no `characterID`**;
- reads **no `SaveEngine`** and takes **no save `expectedRevision`**;
- reads **no `GameCatalog`**;
- never touches a save file.

Everything it changes lives under the store directory
(`os.UserConfigDir()/EldenRing-SaveEditor/templates` by default).

## Input

```go
func UpdateBuildTemplate(
	store *buildtemplates.Store,
	templateID string,
	req UpdateBuildTemplateRequest,
) (UpdateBuildTemplateResult, error)
```

```go
type UpdateBuildTemplateRequest struct {
	TemplateRevision string                       `json:"templateRevision"`
	Metadata         *UpdateBuildTemplateMetadata `json:"metadata,omitempty"`
	Content          *BuildTemplate               `json:"content,omitempty"`
}

type UpdateBuildTemplateMetadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}
```

| Parameter | Type | Meaning |
|---|---|---|
| `store` | `*buildtemplates.Store` | The local templates library. A `nil` store is rejected. |
| `templateID` | `string` | Required, non-empty identifier of the entry in `_index.json`. |
| `req.templateRevision` | `string` | Required generation token of the entry, echoed back exactly as a getter reported it. |
| `req.metadata` | `*UpdateBuildTemplateMetadata` | Optional full replacement of editable metadata (`name`, `description`, `tags`). |
| `req.content` | `*BuildTemplate` | Optional complete replacement `BuildTemplate` document. |

At least one of `metadata` or `content` must be provided.

### Request semantics

- **`metadata`-only**: The existing payload is loaded from disk. Only its `name`,
  `description`, and `tags` are replaced; all other payload fields (e.g. `author`,
  `sourceCharacterIndex`, `sourceCharacterName`, item selections) remain untouched.
  The index entry's `name`, `description`, and `tags` are updated to match.
- **`content`-only**: The replacement `BuildTemplate` document is validated and
  written as the new payload. The index entry fields (`name`, `description`,
  `tags`, `inventoryItems`, `storageItems`, `version`, `selectedSections`) are
  derived from the new document.
- **Both `metadata` and `content`**: The `metadata` fields override `name`,
  `description`, and `tags` in the replacement document; remaining metadata fields
  in `content.metadata` (such as `author` and character provenance) are preserved.
  The index entry reflects the overridden metadata and the structure of `content`.

`templateID`, `filename`, and `createdAt` of the library entry remain immutable.

### `templateRevision`

- It is the same opaque generation token [`GetBuildTemplate`](get_build_template.md)
  and [`GetBuildTemplates`](get_build_templates.md) return for the template.
- Only the **canonical decimal form** is accepted. `""`, `"01"`, `"+1"`, `"-1"`,
  `" 1"`, `"1.0"` and any value above `uint64` are rejected **before the first
  file is touched**.
- An index entry written before the revision counter existed reports **`"0"`**.
  Passing `"0"` updates such an entry.
- The comparison is made against the canonical rendering of the counter stored
  in the index entry. A token that does not match yields `ErrStaleRevision` and
  **nothing is written**.
- If the current revision counter is `math.MaxUint64`, the mutation is rejected
  before writing to prevent overflow.
- On success, the revision counter increments by **exactly 1**.

## Result

```go
type UpdateBuildTemplateResult struct {
	TemplateID       string `json:"templateID"`
	TemplateRevision string `json:"templateRevision"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `templateID` | `string` | Identifier of the updated template. |
| `templateRevision` | `string` | New canonical revision token of the updated library entry. |

The receipt deliberately exposes no filename. Every error path returns the
**zero** `UpdateBuildTemplateResult` together with the error.

## How the template is updated

The store performs the mutation in exactly this order:

1. Validate `templateID`, `templateRevision`, and ensure at least one of
   `metadata` or `content` is non-nil.
2. Take the `Store` mutex.
3. Read `_index.json` and validate the **complete** index fail-closed (strict
   JSON decoding with no unknown fields, supported version, no duplicate IDs,
   safe filenames).
4. Find the target entry. A missing entry is `ErrNotFound`.
5. Compare `templateRevision` with the entry's canonical token (`ErrStaleRevision`
   on mismatch) and ensure revision is not `math.MaxUint64`.
6. Validate the target filename, reject symlinks (via `os.Lstat`), reject
   symlink escapes and verify the payload file exists inside the store directory.
7. Read, decode (`DecodeTemplate`), and strictly validate the existing payload
   against the index entry fail-closed (`version`, `name`, `description`, `tags`).
   If the existing payload or index entry is corrupt or inconsistent, the update
   is rejected without modifying any file.
8. Build the target `BuildTemplate` document and validate it fail-closed (`ValidateTemplate`).
9. Construct the updated `indexEntry` with bumped `revision`, fresh `updatedAt`
   timestamp and preserved `warnings`, and validate the candidate new index in
   memory before writing.
10. Commit the new payload file atomically through a temporary file in the same
    directory (`CreateTemp`, write, `Sync`, `Close`, `Chmod 0644`, `Rename`).
11. Commit the new `_index.json` file atomically through the same procedure.
12. If step 11 fails, the store attempts an in-process rollback by atomically
    restoring the previous payload file. If rollback succeeds, the index write
    error is returned; if rollback also fails, the error explicitly reports both
    failures without exposing system paths.
13. Individual file writes are atomic, but the two-step sequence is not protected
    by a journal or WAL: a process crash between the two rename operations remains
    a known constraint of the journal-free local store design.

## Error mapping

The HTTP explorer maps errors to status codes as follows:

| Condition | Status | Error reason |
|---|---|---|
| Successful update | `200 OK` | — |
| Invalid JSON body, missing both metadata and content, trailing data, unknown fields, non-canonical revision, invalid template schema | `400 Bad Request` | Descriptive error message |
| `templateID` not found in index, or payload file missing | `404 Not Found` | `template not found` |
| Supplied `templateRevision` does not match entry | `409 Conflict` | `stale template revision` |
| Explorer started with `-allow-external-bind` | `404 Not Found` | Route is not registered |

## Verification

```bash
go test ./backend/buildtemplates -run '^TestStore_UpdateTemplate' -count=1
go test ./backend/endpoints/templates -run '^TestUpdateBuildTemplate' -count=1
go test ./tools/swagger -run '^TestUpdateBuildTemplateRoute' -count=1
```
