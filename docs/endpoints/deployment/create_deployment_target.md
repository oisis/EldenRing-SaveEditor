# CreateDeploymentTarget

## Overview

`CreateDeploymentTarget` validates and stores a new deployment target, then
returns the whole library as it now stands.

The identifier is assigned by the store and a caller-supplied one is ignored, so
a create can never overwrite an existing target by claiming its identifier.

| | |
|---|---|
| EndpointID | `create_deployment_target` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none |

## Input

```go
func CreateDeploymentTarget(
	store *deployment.Store,
	input TargetInput,
) (GetDeploymentTargetsResult, error)

type TargetInput struct {
	TargetID     string `json:"targetID,omitempty"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	SavePath     string `json:"savePath"`
	StartCommand  string `json:"startCommand,omitempty"`
	StopCommand   string `json:"stopCommand,omitempty"`
	StatusCommand string `json:"statusCommand,omitempty"`
	Host          string `json:"host,omitempty"`
	Port          int    `json:"port,omitempty"`
	User          string `json:"user,omitempty"`
	KeyPath       string `json:"keyPath,omitempty"`
}
```


`statusCommand` is the only source of the target's game state. Its exit code is
the whole contract: 0 means the game is running, 1 means it is stopped, and any
other outcome — no configured command, another exit code, a timeout or a
transport fault — leaves the state `unknown`. A target stored before this field
existed simply keeps reporting `unknown`. The backend never derives the state
from a process name, from the start command or from the kind of target.

## Output

```go
GetDeploymentTargetsResult // the same shape GetDeploymentTargets returns
```

## Errors

| Condition | Result |
|---|---|
| the deployment store is not wired | `deployment store is required` |
| `kind` is not `local` or `ssh` | `unknown deployment target kind …` |
| `name` or `savePath` is empty | `a deployment target needs a name` / `… needs the save path on the target system` |
| a start, stop or status command contains a newline or a NUL byte | `the … must be a single command line` — one configured command may never become several |
| an SSH target states no host, user or key path | the matching `an SSH target needs …` message |
| an SSH port is outside 0–65535 | `an SSH port must be between 0 and 65535` |

Nothing is written until validation succeeds, so a refused target leaves the
configuration exactly as it was.

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
