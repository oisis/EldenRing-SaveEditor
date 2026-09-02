# ClearRecentFiles

Clears the host-local Recent Files list without touching any save file.

| | |
|---|---|
| EndpointID | `clear_recent_files` |
| Kind | Mutation of application state; no save revision commit |
| Status | implemented; Wails `ClearRecentFiles` |
| Source | [clear_recent_files.go](../../../backend/endpoints/savesession/clear_recent_files.go) |

The empty list is persisted atomically. Loaded sessions, operation history,
recovery journals and save revisions are unchanged.
