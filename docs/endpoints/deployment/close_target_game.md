# CloseTargetGame

## Overview

`CloseTargetGame` runs the configured stop command of one target as an explicit
user action.

A stop command that found no process is a truthful outcome, not a save
corruption: it is reported through the exit code and the detail rather than as a
failure.

| | |
|---|---|
| EndpointID | `close_target_game` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none |

## Input

```go
func CloseTargetGame(
	ctx context.Context,
	service *deployment.Service,
	targetID string,
) (CommandOutcome, error)
```

## Output

```go
CommandOutcome // the same shape LaunchTargetGame returns
```

## Errors

| Condition | Result |
|---|---|
| the deployment service is not wired | `deployment service is required` |
| `targetID` names no target | `unknown deployment target …` |
| the target states no stop command | `configured: false` — an outcome, not an error |
| the command cannot be started at all | `the stop command could not be run` |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
