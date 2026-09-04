# DownloadTargetBackup

## Overview

`DownloadTargetBackup` copies one backup off its target into a local path the
user chose in the native Save As dialog.

The destination is never derived here: a download can only ever write where the
host dialog already agreed to, so it can never silently overwrite a file.
Cancelling the dialog never reaches this endpoint — the bridge treats the empty
path as an ordinary outcome and copies nothing.

Opening the downloaded file in the editor is a separate, explicit user action.

| | |
|---|---|
| EndpointID | `download_target_backup` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none |

## Input

```go
func DownloadTargetBackup(
	ctx context.Context,
	service *deployment.Service,
	targetID string,
	backupID string,
	target string,
) (DownloadTargetBackupResult, error)
```

## Output

```go
type DownloadTargetBackupResult struct {
	TargetID string `json:"targetID"`
	BackupID string `json:"backupID"`
	Target   string `json:"target"`
}
```

## Errors

| Condition | Result |
|---|---|
| the deployment service is not wired | `deployment service is required` |
| `target` is empty | `a backup download needs a target path` |
| `targetID` names no target | `unknown deployment target …` |
| `backupID` is not in the library | `the selected backup does not exist` |
| the target is an SSH target | the fail-closed transport error |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
