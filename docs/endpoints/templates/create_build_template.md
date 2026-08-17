# CreateBuildTemplate

## Overview

`CreateBuildTemplate` constructs a new schema version 2 Build Template from an
explicit validated data selection of a character in an active save session and
persists it to the **local templates library**.

The endpoint coordinates between three layers:

- **SaveEngine & GameCatalog**: read the character's profile, statistics, and
  equipped spells from the private snapshot of the loaded session, resolving
  occupied spell records through the catalog;
- **Endpoints layer**: validates inputs, verifies session revision consistency
  before and after reading character data, enforces supported section boundaries,
  and constructs the complete `BuildTemplate` document;
- **`buildtemplates.Store.CreateTemplate`**: validates the document fail-closed,
  generates an unpredictable cryptographically secure `templateID`, derives the
  filename solely from that ID, assigns revision 1, updates `_index.json`, and
  commits the payload file atomically followed by the index file atomically.

The endpoint is non-mutating on the save: it never modifies save state, save
revision, the session dirty flag, undo history, or `OwnedItemID` assignments.

## Input

```go
func CreateBuildTemplate(
	store *buildtemplates.Store,
	engine *saveengine.Engine,
	catalog *gamecatalog.Catalog,
	appVersion string,
	req CreateBuildTemplateRequest,
) (CreateBuildTemplateResult, error)
```

```go
type CreateBuildTemplateRequest struct {
	SaveSessionID     string                         `json:"saveSessionID"`
	SourceCharacterID int                            `json:"sourceCharacterID"`
	Selection         buildtemplates.TemplateSelection `json:"selection"`
	Name              string                         `json:"name"`
	Description       string                         `json:"description,omitempty"`
	Tags              []string                       `json:"tags,omitempty"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Required identifier of an active loaded save session. |
| `sourceCharacterID` | `int` | Character slot index in the range `0..9`. Must belong to an active slot. |
| `selection` | `TemplateSelection` | Explicit, non-empty selection of template sections and fields. |
| `name` | `string` | Display name of the new template. Must not be empty. |
| `description` | `string` | Optional detailed description. |
| `tags` | `[]string` | Optional list of categorization tags. |

### Supported Selection & Field Mapping

SaveForge 2.0 strictly enforces fail-closed validation on the selection:

1. **`profile`**:
   - Only confirmed fields `name` and `level` are supported.
   - The boolean shortcut (`All: true`) is **rejected** because unconfirmed
     profile fields (`runes`, `soulMemory`, `class`, `clearCount`,
     `scadutreeBlessing`, `shadowRealmBlessing`, `talismanSlots`) cannot be
     exported.
   - Selecting any unconfirmed profile field is rejected with an error.
2. **`stats`**:
   - All 8 character attributes (`vigor`, `mind`, `endurance`, `strength`,
     `dexterity`, `intelligence`, `faith`, `arcane`) are supported via boolean
     shortcut (`All: true`) or field map.
3. **`spells`**:
   - The boolean shortcut (`All: true`) is **rejected** because it would select
     all fields in `SpellsSection`, including unconfirmed slots `spell13` and
     `spell14`. Callers must explicitly select specific slots (`spell1`..`spell12`).
   - Equipped spell memory slots 1 through 12 (`spell1`..`spell12`) are supported.
   - For each occupied slot, `baseItemID` is formatted as
     `SpellItemIDPrefix | rawMagicParamID` (`0x4XXXXXXX`) with its presentation
     `name` resolved from `GameCatalog`. Empty slots remain `nil`.
   - Selecting `spell13` or `spell14` is rejected.
4. **Unsupported sections**:
   - `inventory.workspace` (v1 format), `equipment`, `items` (v2 items),
     `inventoryLayout`, and `storageLayout` do not currently have confirmed,
     complete export mappings in 2.0 and selecting any of them is rejected.

### Save Read Consistency

To guarantee that exported character data represents a single consistent snapshot:

- The session's `saveRevision` is read via `engine.GetUndoState` before any
  character reader executes.
- After all selected sections are built, `saveRevision` is checked again.
- If the revision changed between the reads, the request is rejected with
  `ErrSaveRevisionConflict` (HTTP 409 Conflict) and nothing is written to the
  library.

## Result

```go
type CreateBuildTemplateResult struct {
	TemplateID       string `json:"templateID"`
	TemplateRevision string `json:"templateRevision"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `templateID` | `string` | Unpredictable, cryptographically generated identifier (`tpl-<hex>`). |
| `templateRevision` | `string` | Canonical decimal revision token of the created entry, always `"1"`. |

## HTTP Transport

When hosted in `tools/swagger`:

- **Route**: `POST /api/v1/build-templates`
- **Loopback-only**: Available only in local loopback mode. An explorer started
  with `-allow-external-bind` does not register this route and returns 404.
- **Status codes**:
  - `201 Created`: Template successfully created and persisted.
  - `400 Bad Request`: Malformed JSON, empty name, empty selection, unsupported
    sections/fields, inactive character, or template schema validation failure.
  - `404 Not Found`: Save session not found.
  - `409 Conflict`: Save revision changed during template construction.
