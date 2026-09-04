# ForgetDeploymentHostKey

## Overview

`ForgetDeploymentHostKey` drops the fingerprint approved for one target's
address, so the next connection asks for an explicit approval again.

| | |
|---|---|
| EndpointID | `forget_deployment_host_key` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none |

## Input

```go
func ForgetDeploymentHostKey(
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
