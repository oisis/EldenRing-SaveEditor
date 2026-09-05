# GetSaveLifecycleSettings

Returns host-local Save lifecycle settings.

| | |
|---|---|
| EndpointID | `get_save_lifecycle_settings` |
| Kind | Getter |
| Status | implemented; Wails `GetSaveLifecycleSettings` |
| Source | [get_save_lifecycle_settings.go](../../../backend/endpoints/savesession/get_save_lifecycle_settings.go) |

The current contract contains `backupRetention` (default `10`), the private
one-time `retentionNoticeShown` state, the configured `backupNamePattern`
(default `{filename}.{timestamp}`) and the derived `backupNameExample`.

`backupNameExample` is what the pattern in effect produces for a sample save; it
is always recomputed on read, never stored, and it is what the Settings screen
previews so that the grammar is not reimplemented in the frontend. The grammar
and its validation are documented in
[set_save_lifecycle_settings.md](set_save_lifecycle_settings.md).

It reads no save file and creates no save session.
