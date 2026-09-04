# GetDeploymentGameStatus

## Overview

`GetDeploymentGameStatus` reports the confirmed state of the game on one
target: `running`, `stopped` or `unknown`.

`unknown` is a first-class answer, not a failure. Identifying the game process
needs a contract that does not exist — the target configuration carries a start
and a stop command and nothing that names a process — so the backend states the
truth instead of guessing from a process list or from the save's modification
time. The interface then applies the explicit warning and confirmation the
deployment specification defines for exactly this case.

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
| the target is an SSH target | the fail-closed transport error |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
