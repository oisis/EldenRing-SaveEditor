# RestoreRecoveryJournal

Loads the unchanged source, verifies its fingerprint and replays every logical
operation into a new private session.

| | |
|---|---|
| EndpointID | `restore_recovery_journal` |
| Kind | Mutation of application session state; no save file write |
| Status | implemented; Wails `RestoreRecoveryJournal` |
| Source | [restore_recovery_journal.go](../../../backend/endpoints/savesession/restore_recovery_journal.go) |

Replay is ordered and fail-closed: every patch requires its exact preimage, the
final candidate must validate, and operation IDs may not collide with the
running engine. Failure creates no recovered session and never writes the
source. Success returns a normal `SessionInfo` for the recovered dirty session.
