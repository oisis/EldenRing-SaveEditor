# GetDeploymentGameStatus

## Overview

`GetDeploymentGameStatus` reports the confirmed state of the game on one
target: `running`, `stopped` or `unknown`.

The state comes from one place only: the status command the user explicitly
configured on the target. Its exit code is the whole contract.

| Outcome of the status command | Reported state |
|---|---|
| exit code 0 | `running` |
| exit code 1 | `stopped` |
| no status command is configured | `unknown` |
| any other exit code | `unknown` |
| a timeout, a transport fault, or a command that could not be started | `unknown` |

`unknown` is a first-class answer, not a failure: the interface applies the
explicit warning and confirmation the deployment specification defines for
exactly this case. The backend never derives the state from a process name, from
the start command, from the kind of operating system or from the save's
modification time, and the mere success of a start or stop command is not
evidence of a state either.

| | |
|---|---|
| EndpointID | `get_deployment_game_status` |
| Kind | Getter |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none |

## Input

```go
func GetDeploymentGameStatus(
	ctx context.Context,
	service *deployment.Service,
	targetID string,
) (GetDeploymentGameStatusResult, error)
```

## Output

```go
type GetDeploymentGameStatusResult struct {
	TargetID   string `json:"targetID"`
	GameStatus string `json:"gameStatus"`
}
```

## Errors

| Condition | Result |
|---|---|
| the deployment service is not wired | `deployment service is required` |
| `targetID` names no target | `unknown deployment target …` |
| the SSH connection cannot be opened or its host key is not approved | the transport's own refusal |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
