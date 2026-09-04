# GetTargetBackups

## Overview

`GetTargetBackups` returns the backup library of one deployment target, newest
first, together with whether this build can act on it.

Save Manager and Deployment read the same library: there is one backup model and
one target model in the backend, not two.

A record deliberately carries no size. The Save Manager table has no `Size`
column, and a value nothing renders would only be one more thing that can go
stale.

| | |
|---|---|
| EndpointID | `get_target_backups` |
| Kind | Getter |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none |

## Input

```go
func GetTargetBackups(
	store *deployment.Store,
	targetID string,
) (GetTargetBackupsResult, error)
```

## Output

```go
type BackupRecord struct {
	ID          string   `json:"id"`
	TargetID    string   `json:"targetID"`
	FileName    string   `json:"fileName"`
	CreatedAt   string   `json:"createdAt"`
	Manual      bool     `json:"manual"`
	Active      bool     `json:"active"`
	Tags        []string `json:"tags,omitempty"`
	Description string   `json:"description,omitempty"`
}

type GetTargetBackupsResult struct {
	TargetID          string         `json:"targetID"`
	Backups           []BackupRecord `json:"backups"`
	TransferSupported bool           `json:"transferSupported"`
	UnsupportedReason string         `json:"unsupportedReason,omitempty"`
}
```

`manual` marks a backup the user asked for. Manual backups are exempt from
automatic retention and the pruning path cannot even see them.

## Errors

| Condition | Result |
|---|---|
| the deployment store is not wired | `deployment store is required` |
| `targetID` names no target | `unknown deployment target …` |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
