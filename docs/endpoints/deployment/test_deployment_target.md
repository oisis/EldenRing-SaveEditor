# TestDeploymentTarget

## Overview

`TestDeploymentTarget` verifies that one target is reachable and that its save
location can be written, then reports the confirmed game status and whether a
save is already present there.

The check changes nothing: it creates and removes a private probe file in the
target directory and never touches, replaces or reads the save itself.

| | |
|---|---|
| EndpointID | `test_deployment_target` |
| Kind | Getter |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none |

## Input

```go
func TestDeploymentTarget(
	ctx context.Context,
	service *deployment.Service,
	targetID string,
) (TestDeploymentTargetResult, error)
```

## Output

```go
type TestTargetResult struct {
	TargetID       string     `json:"targetID"`
	Reachable      bool       `json:"reachable"`
	HostKeyTrusted bool       `json:"hostKeyTrusted"`
	GameStatus     GameStatus `json:"gameStatus"`
	SaveExists     bool       `json:"saveExists"`
}
```

## Errors

| Condition | Result |
|---|---|
| the deployment service is not wired | `deployment service is required` |
| `targetID` names no target | `unknown deployment target …` |
| the target directory is missing or not writable | `the target directory is not reachable` / `… is not writable` |
| the target is an SSH target | the fail-closed transport error; no connection is attempted |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
