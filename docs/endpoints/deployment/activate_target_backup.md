# ActivateTargetBackup

## Overview

`ActivateTargetBackup` makes one backup the current save of its target.

It is a replacement of the target file, so it obeys exactly the same safety
rules as a deployment: a confirmed running game blocks it, the current target
save is backed up first, the replacement is staged and renamed atomically, and
the result is verified.

A backup the store does not know can never be activated: the endpoint refuses
rather than falling back to a file name the caller supplied.

| | |
|---|---|
| EndpointID | `activate_target_backup` |
| Kind | Mutation |
| Domain | `deployment` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/deployment](../../../backend/endpoints/deployment) |
| Domain source | [../../../backend/deployment](../../../backend/deployment) |
| Save access | none of the open session; only the target's own file is replaced |

## Input

```go
func ActivateTargetBackup(
	ctx context.Context,
	service *deployment.Service,
	store *deployment.Store,
	operationID string,
	targetID string,
	backupID string,
	continueWithUnknownGameStatus bool,
	confirmRemoteBackup bool,
) (ActivateTargetBackupResult, error)
```

## Output

```go
type ActivateTargetBackupResult struct {
	Operation OperationResult        `json:"operation"`
	Backups   GetTargetBackupsResult `json:"backups"`
}
```

### Blocked outcomes

A blocked outcome is not an error. `targetState` is authoritative about whether
the target remained `unchanged`, was `replaced_verified`, was
`replaced_unverified`, or ended in `replacement_undetermined` after the
irreversible replacement point.

`replacement_undetermined` is the third answer and not a synonym for either of
the other two: the replacement was requested and its result was never
established, so the target may or may not carry the new save. It is never
retried automatically, the game is never started after it, and it is reported
with `failure` `replacement_undetermined`.

| `blocked` | Meaning | Resolved by |
|---|---|---|
| `game_running` | plain `Upload` and `Download` are refused while the game runs | nothing — there is no Continue Anyway; use `Deploy & Launch` or `Close & Download` |
| `game_status_unknown` | the backend cannot confirm the game state, before or after the stop command | `continueWithUnknownGameStatus`; confirming the stop is not a confirmation for this |
| `remote_backup_confirmation_required` | the target already has a save and the policy is `ask` | `confirmRemoteBackup`; refusing cancels the operation, it can never continue without the backup |
| `stop_game_confirmation_required` | the running game has to be stopped first | `confirmStopGame`; refusing cancels the operation |
| `cancelled` | the operation was cancelled before the replacement point | starting it again |

At most one backup of a target carries the active mark. The result carries the
library as it stands afterwards, so a blocked activation never leaves the caller
guessing whether anything changed.

## Errors

| Condition | Result |
|---|---|
| the deployment service is not wired | `deployment service is required` |
| `targetID` names no target | `unknown deployment target …` |
| `backupID` is not in the library | `the selected backup does not exist` |
| the backup file is gone from the target | `the selected backup is missing on the target` |
| the mandatory backup of the current save fails | `the mandatory target backup failed: …` — the target is not replaced |
| the target is an SSH target | the fail-closed transport error |

## Local verification

```bash
go test -count=1 ./backend/deployment ./backend/endpoints/deployment
```
