# GetRecoveryJournals

Returns safe startup summaries of every protected recovery journal.

| | |
|---|---|
| EndpointID | `get_recovery_journals` |
| Kind | Getter |
| Status | implemented; Wails `GetRecoveryJournals` |
| Source | [get_recovery_journals.go](../../../backend/endpoints/savesession/get_recovery_journals.go) |

Each journal is classified as `compatible`, `incompatible` or `corrupt`.
Compatibility requires the exact source fingerprint. Summaries expose safe
operation metadata but never their private replay byte patches.
