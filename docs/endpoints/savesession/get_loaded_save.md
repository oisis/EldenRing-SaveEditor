# GetLoadedSave

## Overview

`GetLoadedSave` returns the safe metadata of a save session that already exists
in SaveEngine. It is a pure read of session state: it opens no file, reads no
snapshot, parses no save data, and changes neither the session nor the save.

The session must have been created earlier by
[`LoadSave`](load_save.md). `GetLoadedSave` never creates one, so calling it
before a successful `LoadSave` is an error, not an implicit load.

| | |
|---|---|
| EndpointID | `get_loaded_save` |
| Kind | Getter |
| Domain | `savesession` |
| Implementation status | implemented |
| Transport status | not exposed — callable only as a Go function. No Wails binding, no HTTP route, no CLI command, and no frontend reaches it. |
| Implementation source | [../../../backend/endpoints/savesession/get_loaded_save.go](../../../backend/endpoints/savesession/get_loaded_save.go) |
| Test source | [../../../backend/endpoints/savesession/get_loaded_save_test.go](../../../backend/endpoints/savesession/get_loaded_save_test.go) |
| Save access | none — no file is opened, and the session's private snapshot is not read |
| Mutation | none — the session map, the session, and the snapshot are left unchanged |

## Input

```go
func GetLoadedSave(engine *saveengine.Engine, saveSessionID string) (GetLoadedSaveResult, error)
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

```go
type GetLoadedSaveResult = saveengine.SessionInfo

type SessionInfo struct {
	SaveSessionID  string `json:"saveSessionID"`
	Platform       string `json:"platform"`
	Format         string `json:"format"`
	UnsavedChanges bool   `json:"unsavedChanges"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was read. It equals the requested value. |
| `platform` | `string` | The platform recognised when the session was created: `pc` or `ps4`. |
| `format` | `string` | The recognised container format: `bnd4` for PC and `ps4-container` for PS4. |
| `unsavedChanges` | `bool` | Always `false`. The session is read-only at this stage, so it can hold no pending change. |

The result is the same metadata model `LoadSave` returns, reused rather than
duplicated. It is an independent value: changing it does not change the metadata
SaveEngine keeps.

The result carries no absolute path, no handle, no offset, no raw save byte, and
no character, inventory, slot, `SteamID`, `UserData10`, `UserData11`, or MD5
data. None of that is read to produce it.

On any error the result is the zero value.

## Processing flow

1. The endpoint rejects a missing engine.
2. Everything else is delegated to SaveEngine: it validates `saveSessionID` and
   looks the session up under its own lock.
3. SaveEngine returns the session's public metadata by value. The session model
   and its private snapshot stay inside the package.

The endpoint is thin by design: it contains no SaveEngine rule, and there is no
shared endpoint helper behind it.

## Validation and errors

Every failure returns the zero result and changes nothing.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown | `unknown save session "<id>"`. A closed, expired, or never-created session is never resolved to a different one. |

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint. In
  particular it does not call `LoadSave`.
- It reads no GameCatalog data.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is
  greenfield.

## Command-line verification

`GetLoadedSave` has no transport, so it is verified through its tests. From the
repository root:

```bash
go test ./backend/saveengine -run '^TestGetSessionInfo' -count=1 -v
go test ./backend/endpoints/savesession -run '^TestGetLoadedSave' -count=1 -v
```

The tests build a synthetic PC container inside `t.TempDir()`. They use no real
save file and no repository fixture, so they depend on nothing outside the test
process. One test deletes the source file after the session exists and fails if
`GetLoadedSave` still needs it, which proves the endpoint neither reloads nor
reopens the save.

## Current limitations

- The endpoint is not exposed through Wails, HTTP, or a CLI, and there is no
  frontend for it. The local OpenAPI explorer does not route to it.
- It reports session metadata only. Characters, inventory, storage, equipment,
  world state, and slot content are not readable yet, because nothing exposes
  the snapshot.
- `unsavedChanges` is a constant `false` until a mutating stage exists.
- There is no way to list sessions. A loaded session is released with the
  implemented [`CloseSave`](close_save.md), which removes it from SaveEngine;
  afterwards `GetLoadedSave` no longer resolves it.
