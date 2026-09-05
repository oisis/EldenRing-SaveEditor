# TrustDeploymentHostKey

## Overview

`TrustDeploymentHostKey` stores the SSH host key fingerprint the user approved
for one target's address. It is the second half of Trust On First Use; the first
half is the handshake `TestDeploymentTarget` performs, which records what the
host actually presented and refuses the connection until the user decides.

The approval is bound to that observation. The store accepts a fingerprint only
when a handshake with the target's exact host and port presented it in this
process, so a caller cannot approve an invented value, cannot approve a value
observed for a different host, and cannot approve anything at all for a target
that was never contacted. There is no equivalent of `InsecureIgnoreHostKey`
anywhere in the module.

| | |
|---|---|
| EndpointID | `trust_deployment_host_key` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none |

## Input

```go
func TrustDeploymentHostKey(
	store *deployment.Store,
	targetID string,
	fingerprint string,
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
| the target is not an SSH target | `only an SSH target has a host key` |
| the address or the fingerprint is empty | `trusting a host key needs the address and the fingerprint` |
| no handshake with this address ever observed a key | `this host key was never observed; test the target first and approve what it presented` |
| the fingerprint is not the one the host presented | `this is not the host key the target presented` |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
