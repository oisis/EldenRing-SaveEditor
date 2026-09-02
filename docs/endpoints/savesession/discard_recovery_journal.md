# DiscardRecoveryJournal

Permanently removes one selected protected recovery journal.

| | |
|---|---|
| EndpointID | `discard_recovery_journal` |
| Kind | Mutation of application state; no save revision commit |
| Status | implemented; Wails `DiscardRecoveryJournal` |
| Source | [discard_recovery_journal.go](../../../backend/endpoints/savesession/discard_recovery_journal.go) |

The exact journal identifier is required. Its source save and any live session
are not modified. Missing or invalid identifiers fail instead of selecting a
different file.
