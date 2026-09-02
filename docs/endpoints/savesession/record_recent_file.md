# RecordRecentFile

Moves one accepted durable local session to the head of Recent Files.

| | |
|---|---|
| EndpointID | `record_recent_file` |
| Kind | Mutation of application state; no save revision commit |
| Status | implemented; Wails `RecordRecentFile` |
| Source | [record_recent_file.go](../../../backend/endpoints/savesession/record_recent_file.go) |

Input is `saveSessionID`. Temporary sessions are rejected. The exact backend
session metadata is stored atomically, duplicate paths collapse to one entry,
and the oldest entry is dropped when the list would exceed ten.
