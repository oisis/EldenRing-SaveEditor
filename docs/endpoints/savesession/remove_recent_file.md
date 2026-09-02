# RemoveRecentFile

Removes one exact path from the host-local Recent Files list.

| | |
|---|---|
| EndpointID | `remove_recent_file` |
| Kind | Mutation of application state; no save revision commit |
| Status | implemented; Wails `RemoveRecentFile` |
| Source | [remove_recent_file.go](../../../backend/endpoints/savesession/remove_recent_file.go) |

The path is matched exactly and is never resolved or normalized. The referenced
save is not touched. The result is the complete remaining list.
