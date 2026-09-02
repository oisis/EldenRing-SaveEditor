# SetSaveLifecycleSettings

Atomically updates the host-local automatic-backup retention count.

| | |
|---|---|
| EndpointID | `set_save_lifecycle_settings` |
| Kind | Mutation of application settings; no save revision commit |
| Status | implemented; Wails `SetSaveLifecycleSettings` |
| Source | [set_save_lifecycle_settings.go](../../../backend/endpoints/savesession/set_save_lifecycle_settings.go) |

Input is an integer `backupRetention` in the supported range `1..1000`. The
private settings file is written atomically with owner-only permissions. A
failure restores the previous in-memory value.
