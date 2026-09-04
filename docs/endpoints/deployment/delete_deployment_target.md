# DeleteDeploymentTarget

## Overview

`DeleteDeploymentTarget` removes one target from the configuration together
with its approved host key and its backup metadata, and returns the remaining
library.

Nothing on the target itself is touched. Removing a configuration entry may
never delete a user's files.

| | |
|---|---|
| EndpointID | `delete_deployment_target` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none |

## Input

```go
func DeleteDeploymentTarget(
	store *deployment.Store,
	targetID string,
) (GetDeploymentTargetsResult, error)
```

## Output

```go
GetDeploymentTargetsResult
```

## Errors

| Condition | Result |
|---|---|
| the deployment store is not wired | `deployment store is required` |
| `targetID` names no target | `unknown deployment target …` |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
