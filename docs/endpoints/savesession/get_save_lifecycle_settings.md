# GetSaveLifecycleSettings

Returns host-local Save lifecycle settings.

| | |
|---|---|
| EndpointID | `get_save_lifecycle_settings` |
| Kind | Getter |
| Status | implemented; Wails `GetSaveLifecycleSettings` |
| Source | [get_save_lifecycle_settings.go](../../../backend/endpoints/savesession/get_save_lifecycle_settings.go) |

The current contract contains `backupRetention` (default `10`) and the private
one-time `retentionNoticeShown` state. It reads no save file and creates no save
session.
