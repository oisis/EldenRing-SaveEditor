# LoadSave

## Overview

`LoadSave` reads a local save file read-only, recognises its container,
validates the structure the current stage depends on, and creates a new save
session holding a private snapshot of the file. It never modifies the file it
opened, and it closes that file before returning.

This document describes the first, deliberately minimal stage of the endpoint
and of `backend/saveengine`. The endpoint recognises PC and PS4 containers and
reports session metadata. It does not yet read characters, inventory, storage,
equipment, world state, `SteamID`, `UserData11`, or any slot content.

| | |
|---|---|
| EndpointID | `load_save` |
| Kind | Mutation |
| Domain | `savesession` |
| Implementation status | implemented |
| Transport status | transport-exposed — `POST /api/v1/save-sessions` of the local OpenAPI explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind it does not exist and answers 404. Also exposed through the Wails bridge method `LoadSave(source, expectedPlatform, sourceKind)`, which the frontend reaches through its save-session port. No CLI command. |
| Implementation source | [../../../backend/endpoints/savesession/load_save.go](../../../backend/endpoints/savesession/load_save.go) |
| Test source | [../../../backend/endpoints/savesession/load_save_test.go](../../../backend/endpoints/savesession/load_save_test.go) |
| Save access | read-only — the file is opened for reading, and no byte of it is written |
| Mutation | application state only — a new session is registered in SaveEngine; the input file is unchanged |

`LoadSave` is a mutation because it creates a session, not because it changes a
save.

## Input

```go
func LoadSave(
	engine *saveengine.Engine,
	source string,
	expectedPlatform string,
	sourceKind string,
) (LoadSaveResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the session; the endpoint never creates one. A `nil` engine is rejected. |
| `source` | `string` | Path of a local save file. It is passed to SaveEngine unchanged: it is never rewritten, resolved against a search path, or guessed from a file name. |
| `expectedPlatform` | `string` | The platform the caller expects, or an empty value for no expectation. |
| `sourceKind` | `string` | What `source` is: `local` or `temporary`. Required; there is no empty form and no default. |

The local HTTP request carries all three values as a JSON body:

```http
POST /api/v1/save-sessions
Content-Type: application/json
```

The `Content-Type` header is required and must be `application/json`. A valid
media-type parameter is accepted, so `application/json; charset=utf-8` is the
same media type. A missing header, a malformed header, and any other media type
are rejected with `400` before the body is decoded and before the endpoint or
SaveEngine is called, so a refused request opens no file and creates no session.
The rule exists because a `POST` carrying `text/plain` or no `Content-Type` is a
CORS simple request, which a browser sends without a preflight.

### `sourceKind`

- The accepted values are exactly `local` and `temporary`. There is no third
  value and no empty form.
- `local` is a durable file the user owns. `temporary` is a working copy that is
  not the user's durable save; it exists for the later deployment flow and
  carries no behaviour of its own at this stage.
- Matching is exact and case-sensitive and the value is never trimmed. `Local`,
  `" local"`, `local `, and `temp` are unknown values.
- It is validated before the file system is touched, so a rejected value opens
  no file and creates no session.
- There is deliberately no default. A session records where its snapshot came
  from, and it must never claim an origin nobody stated: a caller that omits the
  field is rejected rather than silently treated as `local`.
- The native file dialog of the desktop host supplies `local`.

### `expectedPlatform`

- The accepted values are the empty string, `pc`, and `ps4`.
- Matching is exact, case-sensitive, and never trimmed. `PC`, `" pc"`, and
  `pc ` are unknown values, not the platform `pc`.
- Any other value is an error, and no file is opened for it.
- An empty value means the caller states no expectation and the recognised
  platform is accepted.
- A non-empty value that differs from the recognised platform is a fail-closed
  error: no session is created, and the file stays untouched.

## Output

```go
type LoadSaveResult = saveengine.SessionInfo

type SessionInfo struct {
	SaveSessionID  string `json:"saveSessionID"`
	Platform       string `json:"platform"`
	Format         string `json:"format"`
	SourcePath     string `json:"sourcePath"`
	SourceKind     string `json:"sourceKind"`
	SaveRevision   string `json:"saveRevision"`
	UnsavedChanges bool   `json:"unsavedChanges"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the created session. Always non-empty on success and unique per session. A later `GetLoadedSave` will read a session by this value. |
| `platform` | `string` | The recognised platform: `pc` or `ps4`. Never guessed and never taken from the file name. |
| `format` | `string` | The recognised container format: `bnd4` for PC and `ps4-container` for PS4. |
| `sourcePath` | `string` | The exact `source` the snapshot was created from, carried verbatim. Never trimmed, recased, resolved, or guessed. |
| `sourceKind` | `string` | The `sourceKind` the caller stated, echoed back unchanged. |
| `saveRevision` | `string` | The canonical decimal revision of the session, always `"0"` for a newly loaded one. A string and not a number, so no consumer can round, increment, or reorder it. |
| `unsavedChanges` | `bool` | Always `false` for a newly loaded session. Later mutations may set it, and a successful `WriteSave` clears it. |

On any error the result is the zero value: `saveSessionID` is empty and no
session exists.

### `sourcePath` is metadata, not permission to re-read

`sourcePath` records where the snapshot came from so a desktop session can name
the file it is editing. It is not a licence to reopen that file:

- SaveEngine reads the file exactly once, during `LoadSave`, and closes it
  before returning.
- Every later read — `GetLoadedSave`, every getter, every mutation — goes to the
  private in-memory snapshot.
- Removing, replacing, or rewriting the file after a successful load changes
  nothing about the existing session, `sourcePath` included: the recorded path
  is not re-resolved and not checked for existence.

The result still carries no `SteamID`, no offset, no handle, no raw save bytes,
and no character data, and the `Session` model inside SaveEngine stores none of
them either.

The file content itself lives in a private snapshot that SaveEngine
keeps next to the session, under the same `saveSessionID`. The snapshot is
package-private: no public type, field, or method of SaveEngine or of this
endpoint exposes it, its size, or the path it came from. The getters read save
data exclusively from the snapshot bound to a `saveSessionID`, never
by reopening the user's file; mutations and `WriteSave` operate on that same
session-owned snapshot.

## Processing flow

1. The endpoint rejects a missing engine and delegates everything else. It holds
   no magic, no offset, no size, and no platform rule of its own.
2. SaveEngine validates `expectedPlatform` and then `sourceKind` before touching
   the file system.
3. SaveEngine opens `source` read-only. A directory or any other non-regular
   file is rejected before anything is read. The regular file is then read once
   into a private in-memory snapshot and closed immediately, so no handle to the
   user's file outlives the call and a later change to that file cannot alter an
   existing session.
4. The container is recognised from its leading magic only: `BND4` for PC and
   `CB 01 9C 2C` for PS4. Nothing is decrypted, and no other container is
   accepted.
5. A non-empty `expectedPlatform` that differs from the recognised platform ends
   the call here, before structural validation and before a session exists.
6. The platform-specific validation runs: `backend/saveengine/pc.go` for PC and
   `backend/saveengine/ps4.go` for PS4.
7. A session is created with a fresh identifier, the exact `source` and
   `sourceKind` it was given, and revision `0`, and is registered in the engine
   together with its private snapshot. Not a single byte of the source file was
   written, and the file is already closed at this point.

## Container validation

Only the structure this stage depends on is validated. No slot content, no
`SteamID`, no `UserData11`, and no MD5 checksum is parsed or verified.

### PC

- The leading magic is `BND4`.
- The file holds the complete `0x300`-byte header.
- The header declares exactly 12 BND4 entries.
- The ten slot blocks fit inside the file. Each block is `0x280010` bytes:
  a `0x10` MD5 prefix followed by `0x280000` bytes of slot data.
- The `UserData10` block of `0x60010` bytes fits after them.

The smallest accepted PC file is therefore
`0x300 + 10 × 0x280010 + 0x60010` bytes. `UserData11` follows it and has a
variable length, so it is not part of this bound.

### PS4

- The leading magic is `CB 01 9C 2C`.
- The file holds the complete `0x70`-byte header.
- The header carries its twelve fixed entry descriptors: at `0x10` and every
  8 bytes after it, a consecutive entry index from `0x07` to `0x12`, each
  followed by the `0x7F7F7F7F` marker. The header is validated structurally, not
  by byte equality against one captured header, so a native save is not rejected
  over a field this stage does not interpret.
- The ten slots fit inside the file. Each is `0x280000` bytes, stored without
  the MD5 prefix the PC container uses.
- The `UserData10` block of `0x60000` bytes fits after them.

The smallest accepted PS4 file is therefore
`0x70 + 10 × 0x280000 + 0x60000` bytes.

## Validation and errors

Every failure returns an empty result and creates no session.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `expectedPlatform` is not `""`, `pc`, or `ps4` | Rejected before the file is opened. |
| `sourceKind` is not exactly `local` or `temporary` | Rejected before the file is opened, the empty value included. No default is applied. |
| `source` cannot be opened, or is not a regular file | Rejected with the underlying reason. |
| The leading magic is neither PC nor PS4, or the file is shorter than a magic | `unsupported save container: the file is neither a native PC nor a native PS4 save`. An encrypted or unknown container is never decrypted to find out what it holds. |
| The recognised platform differs from a non-empty `expectedPlatform` | Fail-closed error naming both platforms. |
| The container is recognised but too small or inconsistent with its layout | Rejected with the failing structure named. |

Unknown data always fails safely: it is never repaired, normalised, or
reinterpreted as the other platform.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It reads no GameCatalog data.
- `backend/saveengine/codec.go` is the only component that touches raw save
  bytes. It opens the file read-only, performs bounded reads, and implements no
  write of any kind.
- The endpoint does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is greenfield.

## Command-line verification

`LoadSave` is verified through its tests. Its only transport is the local
OpenAPI explorer route `POST /api/v1/save-sessions`, which exists solely
when the explorer runs without `-allow-external-bind`. From the repository root:

```bash
go test ./backend/saveengine -run '^Test' -count=1 -v
go test ./backend/endpoints/savesession -run '^TestLoadSave' -count=1 -v
```

The tests build synthetic PC and PS4 containers in `t.TempDir()`. They use no
real save file, and no repository fixture, so they depend on nothing outside the
test process. One test hashes the source file before and after a successful load
and fails if its content, size, or modification time changed. Another overwrites
the source after a successful load and fails if the session's private snapshot
followed that change.

## Current limitations

- The transports are the local developer explorer route
  `POST /api/v1/save-sessions`, registered only without `-allow-external-bind`,
  and the Wails bridge method `LoadSave`. There is no CLI command.
- `temporary` is accepted and recorded but carries no behaviour yet: the
  deployment flow that produces such a file is not implemented.
- Only native PC and PS4 containers are recognised. Encrypted containers are
  rejected, and no format conversion exists.
- Structural recognition only: characters, inventory, storage, equipment, world
  state, slot content, `SteamID`, `UserData11`, and MD5 checksums are not read
  or verified.
- The public result carries metadata only. `GetLoadedSave` reads that metadata,
  `WriteSave` persists the session snapshot, and `CloseSave` releases it through
  separate endpoints.
- Each open session holds the full file content until `CloseSave` removes the
  session or the engine itself is released.
