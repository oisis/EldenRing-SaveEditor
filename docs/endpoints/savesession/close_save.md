# CloseSave

## Overview

`CloseSave` removes a save session that already exists in SaveEngine. It is a
pure in-memory lifecycle operation: the session entry is deleted from
SaveEngine's session map, which releases SaveEngine's only references to the
session model and to its private snapshot.

The save file is not involved. `CloseSave` opens no file, reads no snapshot,
writes nothing, and does not delete, move, or normalise the source save.

The session must have been created earlier by [`LoadSave`](load_save.md).
`CloseSave` never creates one, so closing an unknown or already closed session
is an error, not a silent success.

| | |
|---|---|
| EndpointID | `close_save` |
| Kind | Mutation |
| Domain | `savesession` |
| Implementation status | implemented |
| Transport status | transport-exposed — `DELETE /api/v1/save-sessions/{saveSessionID}` of the local OpenAPI explorer (`backend/endpoints/swagger`). The route is registered only when the explorer runs without `-allow-external-bind`; with an external bind it does not exist and answers 404. No Wails binding, no CLI command, and no frontend reaches the endpoint. |
| Implementation source | [../../../backend/endpoints/savesession/close_save.go](../../../backend/endpoints/savesession/close_save.go) |
| Test source | [../../../backend/endpoints/savesession/close_save_test.go](../../../backend/endpoints/savesession/close_save_test.go) |
| Save access | none — no file is opened, and the session's private snapshot is not read |
| Mutation | in-memory only — one entry is removed from SaveEngine's session map. No save file is changed. |

## Input

```go
func CloseSave(engine *saveengine.Engine, saveSessionID string) error
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; the endpoint never creates one. A `nil` engine is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. It is passed to SaveEngine unchanged. |

### `saveSessionID`

- It is matched exactly and case-sensitively. It is never trimmed, normalised,
  or guessed, so `" <id>"`, `"<id> "`, and an upper-cased identifier are unknown
  values, not the session they resemble.
- Validation lives in SaveEngine. The endpoint holds no session-identifier rule
  of its own.

## Output

There is no result value. A `nil` error means the named session no longer exists
in SaveEngine.

## Memory behaviour

Closing a session removes SaveEngine's last references to the session model and
to the private snapshot the session was created from. Those objects then become
eligible for the ordinary Go garbage collector.

Reclaiming the memory is not immediate and not a manual operation. Neither the
endpoint nor SaveEngine forces a collection, frees a buffer explicitly, or gives
any timing guarantee: process memory may stay flat for a while after a
successful close. The contract is "the reference is dropped", not "the memory is
freed now".

## Processing flow

1. The endpoint rejects a missing engine.
2. Everything else is delegated to SaveEngine: it validates `saveSessionID` and,
   under its own lock, checks that the session exists and deletes exactly that
   one map entry.
3. No other session is touched, no snapshot byte is read, and no file is opened.

The endpoint is thin by design: it contains no SaveEngine rule, and there is no
shared endpoint helper behind it.

## Validation and errors

Every failure leaves the session map unchanged.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup and no deletion is attempted. |
| `saveSessionID` is unknown | `unknown save session "<id>"`. An already closed or never-created session is never resolved to a different one, so a second close of the same identifier fails. |

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint. In
  particular it does not call `LoadSave` or `GetLoadedSave`.
- It reads no GameCatalog data.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is
  greenfield.

## Command-line verification

`CloseSave` is verified through its tests. Its only transport is the local
OpenAPI explorer route `DELETE /api/v1/save-sessions/{saveSessionID}`, which exists solely
when the explorer runs without `-allow-external-bind`. From the repository root:

```bash
go test ./backend/saveengine -run '^TestCloseSession' -count=1 -v
go test ./backend/endpoints/savesession -run '^TestCloseSave' -count=1 -v
```

The tests build a synthetic PC container inside `t.TempDir()`. They use no real
save file and no repository fixture, so they depend on nothing outside the test
process.

## Current limitations

- The only transport is the local developer explorer route `DELETE /api/v1/save-sessions/{saveSessionID}`,
  which is registered only without `-allow-external-bind`. There is no Wails
  binding, no CLI command, and no frontend for the endpoint.
- There is no unsaved-changes handling, no expected revision, and no write path,
  because the current stage is read-only: a session can hold no pending change.
  A later mutating stage will need its own explicit contract change.
- There is no way to list sessions and no way to close all of them at once. Each
  session is closed by its own identifier.
