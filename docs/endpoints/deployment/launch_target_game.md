# LaunchTargetGame

## Overview

`LaunchTargetGame` runs the configured start command of one target as an
explicit user action.

The command line reaches the shell as one argument, exactly as the user
configured it. No path, target name or file name is ever concatenated into it,
and a command containing a newline is refused at configuration time. The
command's own output is never returned: it belongs to the target machine and may
carry paths, user names or secrets.

| | |
|---|---|
| EndpointID | `launch_target_game` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none |

## Input

```go
func LaunchTargetGame(
	ctx context.Context,
	service *deployment.Service,
	targetID string,
) (CommandOutcome, error)
```

## Output

```go
type CommandOutcome struct {
	Configured bool   `json:"configured"`
	Executed   bool   `json:"executed"`
	ExitCode   int    `json:"exitCode"`
	Detail     string `json:"detail,omitempty"`
}
```

## Errors

| Condition | Result |
|---|---|
| the deployment service is not wired | `deployment service is required` |
| `targetID` names no target | `unknown deployment target …` |
| the target states no start command | `configured: false` — an outcome, not an error |
| the command cannot be started at all | `the start command could not be run` |

A non-zero exit is reported as an outcome rather than raised as an error, so the
caller can see the real result.

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
