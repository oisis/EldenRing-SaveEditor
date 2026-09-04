# GetDeploymentTargets

## Overview

`GetDeploymentTargets` returns the configured deployment targets together with
the two facts the interface enables its actions from: whether this build can
move a save to and from that kind of target, and whether an SSH host key
fingerprint has already been approved for it.

No target is contacted. The endpoint reads host configuration only.

The SSH key is reported as the path the user stated. Its contents are never read
into the configuration, never copied and never written to a log or the
diagnostic report.

| | |
|---|---|
| EndpointID | `get_deployment_targets` |
| Kind | Getter |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none |

## Input

```go
func GetDeploymentTargets(store *deployment.Store) (GetDeploymentTargetsResult, error)
```

## Output

```go
type TargetEntry struct {
	deployment.Target
	HostKeyTrusted     bool   `json:"hostKeyTrusted"`
	HostKeyFingerprint string `json:"hostKeyFingerprint,omitempty"`
	TransferSupported  bool   `json:"transferSupported"`
	UnsupportedReason  string `json:"unsupportedReason,omitempty"`
}

type GetDeploymentTargetsResult struct {
	Targets        []TargetEntry `json:"targets"`
	AvailableKinds []string      `json:"availableKinds"`
}
```

`transferSupported` is `false` for an SSH target in this build. The reason
string states exactly why, and the interface disables the operations instead of
offering an action that must fail. See
[`../../../backend/deployment/driver.go`](../../../backend/deployment/driver.go).

## Errors

| Condition | Result |
|---|---|
| the deployment store is not wired | `deployment store is required` |
| the configuration document is unreadable or of an unknown version | `the deployment configuration is invalid` — it never degrades into an empty configuration, because that would silently drop every target and every trusted host key |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
