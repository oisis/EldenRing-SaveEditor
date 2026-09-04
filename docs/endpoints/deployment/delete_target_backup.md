# DeleteTargetBackup

## Overview

`DeleteTargetBackup` removes one backup from its target after an explicit
confirmation.

The file is deleted on the target first and the record is dropped afterwards, so
a record never survives the file it describes and never presents a backup the
user can no longer restore.

| | |
|---|---|
| EndpointID | `delete_target_backup` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none |

## Input

```go
func DeleteTargetBackup(
	ctx context.Context,
	service *deployment.Service,
	store *deployment.Store,
	targetID string,
	backupID string,
) (GetTargetBackupsResult, error)
```

## Output

```go
GetTargetBackupsResult
```

## Errors

| Condition | Result |
|---|---|
| the deployment service is not wired | `deployment service is required` |
| `targetID` names no target | `unknown deployment target …` |
| `backupID` is not in the library | `the selected backup does not exist` |
| the file cannot be removed | the driver's own removal failure; the record is kept |
| the target is an SSH target | the fail-closed transport error |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
