# WriteSave

## Overview

`WriteSave` persists the current private snapshot of one SaveEngine session to
an explicitly supplied local path. It serializes the complete container,
reloads and validates the candidate in isolation, and writes it through a
same-directory temporary file before replacing the target.

The endpoint does not infer the destination from `LoadSave`, resolve catalog
resources, repair unknown data, or write through another endpoint.

| | |
|---|---|
| EndpointID | `write_save` |
| Kind | Mutation |
| Domain | `savesession` |
| Implementation status | implemented |
| Transport status | transport-exposed — `POST /api/v1/save-sessions/{saveSessionID}/write` of the local OpenAPI explorer. The route exists only without `-allow-external-bind`; no Wails binding, CLI command, or frontend reaches it. |
| Implementation source | [../../../backend/endpoints/savesession/write_save.go](../../../backend/endpoints/savesession/write_save.go) |
| Test source | [../../../backend/endpoints/savesession/write_save_test.go](../../../backend/endpoints/savesession/write_save_test.go) |
| Save access | writes the validated session snapshot to the explicit `target` |
| GameCatalog access | none |

## Input

```go
func WriteSave(
	engine *saveengine.Engine,
	saveSessionID string,
	expectedRevision string,
	target string,
) (WriteSaveResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | Existing SaveEngine. A `nil` engine is rejected. |
| `saveSessionID` | `string` | Existing session identifier, matched exactly. |
| `expectedRevision` | `string` | Current canonical decimal revision. It is compared byte for byte. |
| `target` | `string` | Explicit destination path, passed unchanged to SaveEngine. |

The local HTTP request uses the session identifier from the path:

```http
POST /api/v1/save-sessions/{saveSessionID}/write
Content-Type: application/json
```

```json
{
  "expectedRevision": "0",
  "target": "/explicit/path/ER0000.sl2"
}
```

The JSON body is strict. Both fields are required, and unknown fields are
rejected. Neither value is trimmed, normalised, parsed by the transport, or
given a default.

The `Content-Type` header is required and must be `application/json`. A valid
media-type parameter is accepted, so `application/json; charset=utf-8` is the
same media type. A missing header, a malformed header, and any other media type
are rejected with `400` before the body is decoded and before the endpoint or
SaveEngine is called, so a refused request writes nothing and advances no
revision. The rule exists because a `POST` carrying `text/plain` or no
`Content-Type` is a CORS simple request, which a browser sends without a
preflight.

## Output

```go
type WriteSaveResult = saveengine.WriteSaveResult

type WriteSaveResult struct {
	saveengine.MutationReceipt
}

type MutationReceipt struct {
	OperationID   string   `json:"operationID"`
	OperationKind string   `json:"operationKind"`
	SaveSessionID string   `json:"saveSessionID"`
	SaveRevision  string   `json:"saveRevision"`
	ChangedScopes []string `json:"changedScopes"`
}
```

The receipt is embedded anonymously, so the JSON result is flat and carries the
five receipt members at its top level. There is no nested `receipt` object and
no domain field of its own:

```json
{
  "operationID": "op-0f1e2d3c4b5a69788796a5b4c3d2e1f0",
  "operationKind": "write_save",
  "saveSessionID": "save-session-1",
  "saveRevision": "1",
  "changedScopes": ["save.session", "diagnostics.report"]
}
```

The embedded `saveengine.MutationReceipt` is exactly the receipt the central
SaveEngine commit path produced for this execution. Nothing here is
reassembled from the EndpointID, the session, the revision or a scope lookup.

- `operationID` names this one execution. It is opaque and unpredictable.
  Identifiers do not repeat among the receipts issued by one running SaveEngine
  instance. That guarantee does not currently cover application restarts:
  uniqueness across restarts requires a persistent operation journal and stays
  outside stage 3b.1. A rejected call returns the zero result and no
  `operationID` at all.
- `operationKind` is the stable kind of the mutation and is always exactly
  `write_save`.
- `changedScopes` are exactly `save.session` and `diagnostics.report`,
  in that canonical order.

`saveRevision` is the new revision after the write. Every successful
`WriteSave`, including a write with no prior in-memory mutation, advances it by
exactly one. The result carries no target path and no save bytes.

The write persists the snapshot and changes no domain value, so it invalidates
the session scope and the validation report pinned to the previous revision,
and nothing else.

## Processing flow

SaveEngine performs the complete operation under its process-wide mutex:

1. Reject a missing or unknown session and a malformed or stale revision.
2. Copy the current snapshot into an independent serialization candidate.
3. For PC, refresh the MD5 prefixes of all ten character slots and
   `UserData10`. PS4 has no corresponding checksum layer.
4. Recognise and structurally validate the serialized container as the same
   platform as the session.
5. Reload every active slot through the current Inventory, Storage, and GaItem
   readers and verify that every visible container handle resolves.
6. Create a temporary file in the target directory, write the complete
   candidate, flush and close it, then rename it into place.
7. Make the validated candidate the active session snapshot.
8. Advance `saveRevision`, retire all previous `ownedItemID` values, and clear
   `unsavedChanges`.

The source path used by `LoadSave` is not stored in the session. It changes only
when the caller explicitly supplies that same path as `target`.

## Atomicity and errors

Before the target rename succeeds, every failure leaves these values unchanged:

- the active session snapshot;
- `saveRevision`;
- `unsavedChanges`;
- the OwnedItemID registry;
- an existing regular target file.

An empty target, a missing parent directory, a directory, a symlink, or another
non-regular existing target is rejected. Existing regular-file permissions are
preserved; a new target uses `0644`. Temporary files are removed on both success
and failure.

After the rename succeeds, the remaining in-memory commit cannot return an
error. The persisted candidate and the active snapshot therefore advance as one
successful operation from the caller's perspective.

## Revision and identity semantics

- `expectedRevision` must be a non-empty canonical decimal string such as `0`
  or `17`. Values such as `00`, `+1`, ` 1`, and `1 ` are rejected.
- A stale revision fails before serialization or file access.
- A successful write always advances the revision, even when the snapshot was
  already clean.
- Every OwnedItemID minted before the write becomes stale. A caller must reread
  Inventory or Storage to obtain identifiers for the new revision.
- `unsavedChanges` becomes `false` only after the target has been written
  successfully.

## Platform scope

The implementation follows the confirmed container behavior of SaveForge
1.5.8 and 1.6.10 and is covered by synthetic PC and PS4 tests. It does not convert
between platforms, decrypt encrypted PC saves, or infer a platform from the
file extension.

No real PC, PS4, console, or game-load test is claimed for this stage. Those
controlled checks remain deferred in `TODO.md`.

## Dependencies and transport safety

- The endpoint delegates directly to `backend/saveengine` and calls no other
  endpoint.
- It reads no GameCatalog data and accepts no `GameResource` reference.
- The HTTP route is registered only for the loopback explorer. Starting the
  explorer with `-allow-external-bind` withholds SaveEngine and the route answers
  404.
- There is no automatic backup or retention policy yet.
- Atomic replacement of an existing target on Windows remains a deferred
  platform test in `TODO.md`.

## Verification

```bash
go test ./backend/saveengine -run '^TestWriteSave' -count=1
go test -race ./backend/saveengine -count=1
go test ./backend/endpoints/savesession -run '^TestWriteSave' -count=1
go test ./tools/swagger -run 'TestSaveSessionLifecycleRoutes|TestOpenAPIDocumentDescribesEveryRoute' -count=1
```

The tests use only synthetic files created under `t.TempDir()`.
