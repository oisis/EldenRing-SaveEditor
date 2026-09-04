# CreateTargetBackup

## Overview

`CreateTargetBackup` copies the current save of one target into a new manual
backup beside it and records its metadata.

The copy is made on the target, so the data does not travel through this
machine. The record is written only after the file is confirmed to exist, so a
record can never describe a backup that was never taken.

Manual backups are exempt from automatic retention: this path never prunes.

| | |
|---|---|
| EndpointID | `create_target_backup` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none of the open session; only the target's own file is copied |

## Input

```go
func CreateTargetBackup(
	ctx context.Context,
	service *deployment.Service,
	store *deployment.Store,
	targetID string,
	tags []string,
	description string,
) (GetTargetBackupsResult, error)
```

## Output

```go
GetTargetBackupsResult // the same shape GetTargetBackups returns
```

## Errors

| Condition | Result |
|---|---|
| the deployment service is not wired | `deployment service is required` |
| `targetID` names no target | `unknown deployment target …` |
| the target has no save | `the target has no save to back up` |
| the copy is not written | `the backup was not written on the target` |
| the target is an SSH target | the fail-closed transport error |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
