# DeleteBuildTemplate

## Overview

`DeleteBuildTemplate` removes one Build Template from the **local templates
library**: it drops the entry from `_index.json` and unlinks the payload file
that entry pointed at.

The endpoint is a thin adapter. The whole mutation — revision validation, the
Store lock, index validation, the atomic index write and the payload removal —
belongs to `buildtemplates.Store.DeleteTemplate`, which is the single owner of
those rules.

The library is a purely local directory. This endpoint therefore:

- accepts **no `saveSessionID`** and **no `characterID`**;
- reads **no `SaveEngine`** and takes **no save `expectedRevision`**;
- reads **no `GameCatalog`**;
- never touches a save file.

It is the first writer in the Build Templates domain. Everything it changes lives
under the store directory (`os.UserConfigDir()/EldenRing-SaveEditor/templates` by
default).

## Input

```go
func DeleteBuildTemplate(
	store *buildtemplates.Store,
	templateID string,
	templateRevision string,
) (DeleteBuildTemplateResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `store` | `*buildtemplates.Store` | The local templates library. A `nil` store is rejected. |
| `templateID` | `string` | Required, non-empty identifier of the entry in `_index.json`. |
| `templateRevision` | `string` | Required generation token of the entry, echoed back exactly as a getter reported it. |

### `templateRevision`

- It is the same opaque generation token
  [`GetBuildTemplate`](get_build_template.md) and
  [`GetBuildTemplates`](get_build_templates.md) return for the template. Callers
  echo it back unchanged; they never parse, order, or compute it.
- Only the **canonical decimal form** is accepted. `""`, `"01"`, `"+1"`, `"-1"`,
  `" 1"`, `"1.0"` and any value above `uint64` are rejected **before the first
  file is touched**. `"18446744073709551615"` (`math.MaxUint64`) is a valid
  token.
- An index entry written before the revision counter existed carries no
  `revision` field and its canonical token is **`"0"`**. Passing `"0"` therefore
  deletes such a legacy entry; `"0"` is a valid token, not a missing value.
- The comparison is made against the canonical rendering of the counter stored
  in the index entry. A token that does not match means the caller acted on a
  stale view: the delete is refused and **nothing is written**.
- Delete never advances the counter. It removes the entry, so there is no
  counter left to bump.

## Result

```go
type DeleteBuildTemplateResult struct {
	TemplateID string `json:"templateID"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `templateID` | `string` | Identifier of the template that was removed from the library. |

The receipt deliberately exposes no filename and no internal revision counter.
Every error path returns the **zero** `DeleteBuildTemplateResult` together with
the error.

## How the template is deleted

The store performs the mutation in exactly this order:

1. Validate `templateID` and the canonical form of `templateRevision`.
2. Take the `Store` mutex. Reads and writes on one `Store` instance share the
   same lock.
3. Read `_index.json` and validate the **complete** index fail-closed: strict
   JSON decoding with no unknown fields, a supported index version, no
   duplicate `templateID`, a non-empty and safe filename on every entry (which
   cannot be `.` or `..`), and no filename shared by two entries. A library the
   writer cannot fully account for is never rewritten.
4. Find the target entry. A missing entry is `ErrNotFound`.
5. Compare `templateRevision` with the entry's canonical token.
6. Build the new index in memory, without the target entry. The remaining
   entries keep their order and their contents.
7. Serialise the new index and re-validate the serialised document before the
   first write.
8. Commit the new `_index.json` atomically: a temporary file in the same
   directory, write, `Sync`, `Close`, `Rename`.
9. **Only then** unlink the payload file.

The index is committed **before** the payload is removed. An interrupted delete
can therefore leave an unreferenced payload behind, but can never leave an index
entry pointing at a file that is already gone.

### Missing payload

A payload file that is already absent is **not** an error. The index entry is
removed either way, which keeps the library consistent when a user deleted the
file by hand.

### Orphan payloads

If the index was committed but the payload could not be unlinked, the delete is
already done and is reported as a success. The leftover file is an invisible
orphan: no getter can reach it, because templates resolve exclusively through
`_index.json`. There is **no automatic cleanup** of orphan files.

### Durability

The temporary index file is synced before the rename, so a crash leaves either
the complete old index or the complete new one. The **containing directory is
not fsynced**, so a power loss can still lose the rename itself. The library
gives no full power-loss durability guarantee.

After a successful delete, deleting the same `templateID` again returns
`ErrNotFound`.

## Errors

| Situation | Result |
|---|---|
| `store` is `nil` | `templates store is not available` |
| empty `templateID` | `templateID must not be empty` |
| non-canonical `templateRevision` | `templateRevision "<token>" is not a canonical decimal revision token` |
| missing `_index.json` | `template "<id>": template not found` (`ErrNotFound`) |
| unsupported index version | `unsupported index version <n>; expected 1` |
| duplicate `templateID` in the index | `index contains duplicate template ID "<id>"` |
| empty filename on any entry | `template "<id>" index entry has empty filename` |
| unsafe filename on any entry | `template "<id>" index entry has invalid filename "<name>"` |
| two entries sharing one filename | `index entry "<id>" shares filename "<name>" with another entry` |
| `templateID` not in the index | `template "<id>": template not found` (`ErrNotFound`) |
| `templateRevision` does not match | `template "<id>": stale template revision` (`ErrStaleRevision`) |

Errors never expose full system paths. Every error above leaves `_index.json`
and every payload byte-for-byte unchanged.

## Transport

```
DELETE /api/v1/build-templates/{templateID}
```

Request body (JSON, strict, `DisallowUnknownFields`):
```json
{
  "templateRevision": "0"
}
```

A missing body, a missing `templateRevision` and an unknown field are all
rejected.

Response (`200 OK`):
```json
{
  "templateID": "tpl-123"
}
```

| Status | Situation |
|---|---|
| `200` | The template was deleted. |
| `400` | Invalid request body, non-canonical `templateRevision`, or a corrupt index. |
| `404` | Unknown `templateID`. |
| `409` | Stale `templateRevision`; nothing was deleted. |

The route belongs to the local developer explorer in `tools/swagger`. It is
registered only in loopback mode; an explorer started with
`-allow-external-bind` does not register it and answers `404`.

## Local verification

```bash
go test ./backend/buildtemplates -run '^TestStore_DeleteTemplate' -count=1
go test ./backend/endpoints/templates -run '^TestDeleteBuildTemplate' -count=1
go test ./tools/swagger -run '^TestDeleteBuildTemplateRoute' -count=1
```
