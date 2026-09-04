# DeployToTarget

## Overview

`DeployToTarget` prepares the current session state and safely replaces the
save of one target, optionally starting the game afterwards. `Upload` and
`Deploy & Launch` are the same operation; `launchAfter` states which.

The preparation is the shared safe phase of SaveEngine: the same review
authorisation, the same serialisation and the same reload validation an ordinary
`Save` performs, written into a private temporary directory that is removed
again whatever the outcome. A deployment can therefore never send a save an
ordinary `Save` would have refused.

The deployment does **not** write the user's local source file, does not mark the
session clean, does not advance its revision and does not clear `Review
Changes`, `Undo` or `Redo`.

The replacement itself is staged, verified, renamed atomically and verified
again. If the target cannot be replaced atomically the operation fails: there is
no fallback to a direct overwrite. Every failure before the rename leaves the
existing target save untouched, and past the rename the backend finishes its
verification and reports the real state rather than stopping half way.

| | |
|---|---|
| EndpointID | `deploy_to_target` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | the in-memory state of the named session, serialized and validated exactly as a `Save` would; the local source file is never written |

## Input

```go
func DeployToTarget(
	ctx context.Context,
	service *deployment.Service,
	engine *saveengine.Engine,
	request DeployRequest,
) (OperationResult, error)

type DeployRequest struct {
	OperationID      string `json:"operationID"`
	TargetID         string `json:"targetID"`
	SaveSessionID    string `json:"saveSessionID"`
	ExpectedRevision string `json:"expectedRevision"`
	ValidationToken  string `json:"validationToken"`
	ConfirmWarnings  bool   `json:"confirmWarnings,omitempty"`
	ConfirmBanRisk   bool   `json:"confirmBanRisk,omitempty"`
	LaunchAfter      bool   `json:"launchAfter,omitempty"`

	ContinueWithUnknownGameStatus bool `json:"continueWithUnknownGameStatus,omitempty"`
	ConfirmRemoteBackup           bool `json:"confirmRemoteBackup,omitempty"`
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

A blocked outcome is not an error. `targetState` is authoritative about whether
the replacement point was crossed: `unchanged`, `replaced_verified` or
`replaced_unverified`. In particular, cancelling the later launch step can
leave a verified replacement in place.

| `blocked` | Meaning | Resolved by |
|---|---|---|
| `game_running` | plain `Upload` and `Download` are refused while the game runs | nothing — there is no Continue Anyway; use `Deploy & Launch` or `Close & Download` |
| `game_status_unknown` | the backend cannot confirm the game state | `continueWithUnknownGameStatus` |
| `remote_backup_confirmation_required` | the target already has a save and the policy is `ask` | `confirmRemoteBackup`; refusing cancels the operation, it can never continue without the backup |
| `stop_game_confirmation_required` | the running game has to be stopped first | `confirmStopGame`; refusing cancels the operation |
| `cancelled` | the operation was cancelled before the replacement point | starting it again |

Progress is reported on the `deployment.progress` host event while the operation
runs, and the operation can be cancelled by its `operationID` up to the atomic
replacement. Past that point only the not-yet-performed steps, such as starting
the game, can still be cancelled.

## Errors

| Condition | Result |
|---|---|
| the deployment service or the engine is not wired | `deployment service is required` / `saveengine.Engine is required` |
| `expectedRevision` is not canonical or does not match the session | `invalid_revision` / `revision_conflict` |
| the validation token is missing or stale | `Review Changes validation is missing or stale` |
| warnings or ban risk are present but not confirmed | the matching explicit-confirmation refusal |
| the prepared file fails reload validation | `the prepared save failed final validation` — nothing is sent |
| the mandatory backup fails | `the mandatory target backup failed: …` — the target is not replaced |
| the transferred or replaced file does not match the prepared save | `the transferred file does not match the prepared save` / `the replaced target save does not match …` |
| the target is an SSH target | the fail-closed transport error |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
