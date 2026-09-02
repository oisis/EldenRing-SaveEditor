# Save

Persists the validated current revision to the session's durable local source.

| | |
|---|---|
| EndpointID | `save` |
| Kind | Mutation |
| Status | implemented; Wails `Save` |
| Source | [save.go](../../../backend/endpoints/savesession/save.go) |

Input is `saveSessionID`, `expectedRevision`, the exact token issued by
`ValidateReviewChanges`, `confirmWarnings` and the separate `confirmBanRisk`.
The backend rejects a required confirmation that is false. Save also fails if
the source changed outside SaveForge.
It creates a required automatic backup, validates serialization, atomically
replaces and rereads the target, then makes it the clean baseline and clears
history and recovery. The result includes the shared receipt, target, backup,
post-commit warnings and the one-time retention notice flag.
