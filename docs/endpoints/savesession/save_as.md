# SaveAs

Persists the validated revision to an explicit target and makes that target the
durable local source of the existing session.

| | |
|---|---|
| EndpointID | `save_as` |
| Kind | Mutation |
| Status | implemented; Wails `SaveAs` after the native target chooser |
| Source | [save_as.go](../../../backend/endpoints/savesession/save_as.go) |

Input is `saveSessionID`, `expectedRevision`, `validationToken`, the independent
`confirmWarnings` and `confirmBanRisk` decisions, and `target`.
An existing regular target is backed up before replacement. The same isolated
serialization, validation, atomic write, final reread and rollback guarantees
as Save apply. Success updates `sourcePath`, clears history and recovery, and
returns the shared receipt plus lifecycle metadata.
