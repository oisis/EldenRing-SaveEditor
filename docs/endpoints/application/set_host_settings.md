# SetHostSettings

## Overview

`SetHostSettings` stores the complete host settings value and returns the
settings now in effect.

It stores the whole value rather than a patch: a partial write would need a
second, implicit source of truth for the fields it left out. The policy is
validated before anything is written, and a failed write restores the previous
value, so the endpoint never reports a setting the host did not keep.

Neither setting weakens a safety rule. `skipReviewForNormalRisk` can only hide
the Review Changes modal for an operation the completed validation reported with
no warning, no ban risk and no critical finding; validation itself always runs.
`remoteBackupPolicy` only chooses whether the mandatory target backup is
confirmed each time.

| | |
|---|---|
| EndpointID | `set_host_settings` |
| Kind | Mutation |
| Domain | `application` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/application](../../../backend/endpoints/application) |
| Save access | none |

## Input

```go
func SetHostSettings(
	store *hostsettings.Store,
	skipReviewForNormalRisk bool,
	remoteBackupPolicy string,
) (HostSettingsResult, error)
```

## Output

```go
HostSettingsResult // the same shape GetHostSettings returns
```

## Errors

| Condition | Result |
|---|---|
| the settings store is not wired | `host settings store is required` |
| `remoteBackupPolicy` is not `ask` or `always` | `unknown remote backup policy …` — nothing is written |
| the settings file cannot be written | `cannot write host settings` — the previous value stays in effect |

This mutation advances no save revision and produces no mutation receipt: it is
host state, not save state.
