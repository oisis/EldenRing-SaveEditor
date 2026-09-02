# GetRecoveryJournal

Returns one safe recovery-journal summary by its exact opaque identifier.

| | |
|---|---|
| EndpointID | `get_recovery_journal` |
| Kind | Getter |
| Status | implemented; Wails `GetRecoveryJournal` |
| Source | [get_recovery_journal.go](../../../backend/endpoints/savesession/get_recovery_journal.go) |

Identifiers containing path separators are rejected. Reading does not replay,
discard or modify the journal or its source save, and replay patches remain
private to SaveEngine.
