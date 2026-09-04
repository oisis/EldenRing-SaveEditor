# UpdateTargetBackup

## Overview

`UpdateTargetBackup` replaces the tags and the description of one backup.

It is metadata only: no file on the target is touched.

| | |
|---|---|
| EndpointID | `update_target_backup` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none |

## Input

```go
func UpdateTargetBackup(
	store *deployment.Store,
	targetID string,
	backupID string,
	tags []string,
	description string,
) (GetTargetBackupsResult, error)
```

## Output

```go
GetTargetBackupsResult
```

## Errors

| Condition | Result |
|---|---|
| the deployment store is not wired | `deployment store is required` |
| `targetID` names no target | `unknown deployment target …` |
| `backupID` is not in the library | `unknown backup …` |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
