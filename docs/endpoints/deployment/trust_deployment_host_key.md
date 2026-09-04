# TrustDeploymentHostKey

## Overview

`TrustDeploymentHostKey` is the internal storage operation for an SSH host key
fingerprint approved for one target's address. It is deliberately not exposed
until the SSH handshake can supply the fingerprint that was actually observed.

It must not be called with a manually entered value: that would only label an
arbitrary string as trusted, not implement Trust On First Use. There is no
equivalent of `InsecureIgnoreHostKey` anywhere in the module.

| | |
|---|---|
| EndpointID | `trust_deployment_host_key` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | not exposed — the current SSH driver cannot observe a remote fingerprint, so no caller may submit one as if it had been observed |
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

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
