# TestDeploymentTarget

## Overview

`TestDeploymentTarget` verifies that one target is reachable and that its save
location can be written, then reports the confirmed game status and whether a
save is already present there.

The check changes nothing: it creates and removes a private probe file in the
target directory and never touches, replaces or reads the save itself.

For an SSH target this is also the handshake that observes the host key. The
connection authenticates with the configured private key only and verifies the
host under Trust On First Use:

- a host whose key was never approved refuses the connection and is reported as
  `hostKeyPending` together with the `observedFingerprint` the host presented,
  which is the value `TrustDeploymentHostKey` will accept and nothing else;
- a host presenting a different key than the approved one refuses the connection
  and is reported as `hostKeyChanged`; it is never re-approved from here.

Neither is returned as an error: both are states the interface answers with an
explicit user decision.

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
	TargetID            string     `json:"targetID"`
	Reachable           bool       `json:"reachable"`
	HostKeyTrusted      bool       `json:"hostKeyTrusted"`
	GameStatus          GameStatus `json:"gameStatus"`
	SaveExists          bool       `json:"saveExists"`
	HostKeyPending      bool       `json:"hostKeyPending"`
	HostKeyChanged      bool       `json:"hostKeyChanged"`
	ObservedFingerprint string     `json:"observedFingerprint,omitempty"`
}
```

## Errors

| Condition | Result |
|---|---|
| the deployment service is not wired | `deployment service is required` |
| `targetID` names no target | `unknown deployment target …` |
| the target directory is missing or not writable | `the target directory is not reachable` / `… is not writable` |
| the configured SSH key is missing or unusable | `the configured SSH private key could not be read` / `… could not be used; encrypted keys are not supported`; no connection is attempted |
| the SSH host is unreachable | `the SSH target could not be reached` |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
