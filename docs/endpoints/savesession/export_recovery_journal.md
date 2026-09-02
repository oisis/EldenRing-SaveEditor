# ExportRecoveryJournal

Copies one protected recovery journal to an explicit user-selected target for
inspection or support.

| | |
|---|---|
| EndpointID | `export_recovery_journal` |
| Kind | Mutation of external output only; no save revision commit |
| Status | implemented; Wails `ExportRecoveryJournal` after a native target chooser |
| Source | [export_recovery_journal.go](../../../backend/endpoints/savesession/export_recovery_journal.go) |

The export is atomic and does not replay the journal or modify its source. It
does not change permissions of the user-selected parent directory. The caller
is responsible for the exported copy after it leaves private application data.
