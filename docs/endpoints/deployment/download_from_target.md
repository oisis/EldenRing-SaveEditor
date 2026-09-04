# DownloadFromTarget

## Overview

`DownloadFromTarget` copies the save of one target into a local staging file.
`Download` and `Close & Download` are the same operation; `closeGameFirst`
states which.

The operation waits until the target save has stopped changing before it copies
anything, so a download never captures a file the game is still writing.

The staging file a completed download produces is deliberately kept: the caller
loads it as a **temporary** session, which is what keeps `Save` unavailable for
it until an explicit `Save As`. A blocked or failed download removes the staging
directory again and replaces no session.

| | |
|---|---|
| EndpointID | `download_from_target` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none of the open session; only the target's own file is read |

## Input

```go
func DownloadFromTarget(
	ctx context.Context,
	service *deployment.Service,
	request DownloadRequest,
) (OperationResult, error)

type DownloadRequest struct {
	OperationID    string `json:"operationID"`
	TargetID       string `json:"targetID"`
	CloseGameFirst bool   `json:"closeGameFirst,omitempty"`

	ContinueWithUnknownGameStatus bool `json:"continueWithUnknownGameStatus,omitempty"`
	ConfirmStopGame               bool `json:"confirmStopGame,omitempty"`
}
```

## Output

```go
type Stage struct {
	Stage     string `json:"stage"`
	Completed bool   `json:"completed"`
	Detail    string `json:"detail,omitempty"`
}

type OperationResult struct {
	OperationID string          `json:"operationID"`
	TargetID    string          `json:"targetID"`
	Completed   bool            `json:"completed"`
	Blocked     string          `json:"blocked,omitempty"`
	Failure     string          `json:"failure,omitempty"`
	TargetState string          `json:"targetState"`
	GameStatus  GameStatus      `json:"gameStatus"`
	Stages      []Stage         `json:"stages"`
	BackupID    string          `json:"backupID,omitempty"`
	LocalPath   string          `json:"localPath,omitempty"`
	Stop        *CommandOutcome `json:"stop,omitempty"`
	Launch      *CommandOutcome `json:"launch,omitempty"`
}
```

### Blocked outcomes

A blocked outcome is not an error. It always means the target was left exactly
as it was, and it is resolved by one explicit user decision passed back in the
next request.

| `blocked` | Meaning | Resolved by |
|---|---|---|
| `game_running` | plain `Upload` and `Download` are refused while the game runs | nothing — there is no Continue Anyway; use `Deploy & Launch` or `Close & Download` |
| `game_status_unknown` | the backend cannot confirm the game state | `continueWithUnknownGameStatus` |
| `remote_backup_confirmation_required` | the target already has a save and the policy is `ask` | `confirmRemoteBackup`; refusing cancels the operation, it can never continue without the backup |
| `stop_game_confirmation_required` | the running game has to be stopped first | `confirmStopGame`; refusing cancels the operation |
| `cancelled` | the operation was cancelled before the replacement point | starting it again |

The desktop bridge tracks the returned staging file until the frontend opens
or discards it. It removes the containing staging directory after either
outcome and removes any remaining tracked directories at application shutdown;
an arbitrary frontend path is never accepted for cleanup.

## Errors

| Condition | Result |
|---|---|
| the deployment service is not wired | `deployment service is required` |
| `targetID` names no target | `unknown deployment target …` |
| the target has no save | `the target has no save to download` |
| the copy fails | the driver's own copy failure; no session is replaced |
| the target is an SSH target | the fail-closed transport error |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
