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
| Transport status | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}` of the local OpenAPI explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind it does not exist and answers 404. Also exposed through the Wails bridge method `GetLoadedSave(saveSessionID)`, which the frontend reaches through its save-session port. No CLI command. |
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
	SourcePath     string `json:"sourcePath"`
	SourceKind     string `json:"sourceKind"`
	SaveRevision   string `json:"saveRevision"`
	UnsavedChanges bool   `json:"unsavedChanges"`
	EventSequence  string `json:"eventSequence"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was read. It equals the requested value. |
| `platform` | `string` | The platform recognised when the session was created: `pc` or `ps4`. |
| `format` | `string` | The recognised container format: `bnd4` for PC and `ps4-container` for PS4. |
| `sourcePath` | `string` | The exact path the session's snapshot was created from, as recorded by `LoadSave`. |
| `sourceKind` | `string` | `local` or `temporary`, as stated at load time. |
| `saveRevision` | `string` | The session's current canonical decimal revision: `"0"` after `LoadSave`, and the value the last accepted mutation returned afterwards. A refused mutation does not advance it. |
| `unsavedChanges` | `bool` | Whether the session's private snapshot carries a committed mutation. `false` after `LoadSave` and after a successful `WriteSave`. |
| `eventSequence` | `string` | The session's canonical decimal position in its `session.changed` stream: `"0"` after `LoadSave`, advanced by exactly one per committed mutation. A subscriber reads it to establish or re-establish its baseline after a start, a lost event or a reconnect. A refused mutation, a rollback and a success that commits nothing do not advance it. |

The result is the same metadata model `LoadSave` returns, reused rather than
duplicated. It is an independent value: changing it does not change the metadata
SaveEngine keeps.

`sourcePath` is reported from what the session recorded. It is not re-resolved
and not checked for existence, so a source file removed, replaced, or rewritten
after the load changes no field of this result: the session answers from its
private snapshot alone.

The result carries no handle, no offset, no raw save byte, and no character,
inventory, slot, `SteamID`, `UserData10`, `UserData11`, or MD5 data. None of
that is read to produce it.

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

`GetLoadedSave` is verified through its tests. Its only transport is the local
OpenAPI explorer route `GET /api/v1/save-sessions/{saveSessionID}`, which exists solely
when the explorer runs without `-allow-external-bind`. From the repository root:

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

- The only transport is the local developer explorer route `GET /api/v1/save-sessions/{saveSessionID}`,
  which is registered only without `-allow-external-bind`. There is no Wails
  binding, no CLI command, and no frontend for the endpoint.
- It reports session metadata only. Characters, inventory, storage, equipment,
  world state, and slot content are not readable yet, because nothing exposes
  the snapshot.
- `unsavedChanges` is a constant `false` until a mutating stage exists.
- There is no way to list sessions. A loaded session is released with the
  implemented [`CloseSave`](close_save.md), which removes it from SaveEngine;
  afterwards `GetLoadedSave` no longer resolves it.
