# GetRecentFiles

Returns at most ten durable local saves in most-recently-opened order.

| | |
|---|---|
| EndpointID | `get_recent_files` |
| Kind | Getter |
| Status | implemented; Wails `GetRecentFiles` |
| Source | [get_recent_files.go](../../../backend/endpoints/savesession/get_recent_files.go) |

Each entry contains the exact recorded path, platform, format and RFC3339
timestamp. The getter does not open, validate or remove the referenced file.
