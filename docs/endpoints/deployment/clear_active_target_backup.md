# ClearActiveTargetBackup

## Overview

`ClearActiveTargetBackup` removes the active mark from the backups of one
target.

It is metadata only: no file on the target is touched and the target save is not
replaced.

| | |
|---|---|
| EndpointID | `clear_active_target_backup` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none |

## Input

```go
func ClearActiveTargetBackup(
	service *deployment.Service,
	store *deployment.Store,
	targetID string,
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

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
