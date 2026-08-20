# ApplyRepairs

## Overview

`ApplyRepairs` re-derives the selected [`GetRepairPlan`](get_repair_plan.md)
result at its original save revision, checks that `planToken` seals the same
executable actions, then applies all of those actions atomically. It never
accepts actions from the client.

| | |
|---|---|
| EndpointID | `apply_repairs` |
| Kind | Mutation |
| Domain | `diagnostics` |
| Implementation status | implemented |
| Transport status | transport-exposed — `POST /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/repairs/apply` of the local explorer, registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. |
| Implementation source | [../../../backend/endpoints/diagnostics/apply_repairs.go](../../../backend/endpoints/diagnostics/apply_repairs.go) |
| Test source | [../../../backend/endpoints/diagnostics/apply_repairs_test.go](../../../backend/endpoints/diagnostics/apply_repairs_test.go) |
| Save access | mutation of the session's private snapshot only; no source file is opened |

## Input

`issueIDs` are the exact selected findings previously supplied to
`GetRepairPlan`. A digest can verify actions but cannot reconstruct their
selection, so the identifiers are required. Their order is ignored; duplicate,
empty and unknown identifiers are rejected. `expectedRevision` must exactly
equal the current revision.

`planToken` must equal the token of the freshly regenerated executable action
set. It seals actions only: rejected findings carry no mutation and therefore do
not affect it.

## Supported actions

Only actions already produced by `GetRepairPlan` are executable:

- `remove_owned_item` for a record with quantity zero;
- `set_owned_item_quantity` for a record above its confirmed per-record limit;
- `set_character_stats` for the single derived legal eight-attribute set.

Every other finding remains a `Rejected` result. A mixed selection commits its
actions and returns its rejections. A selection containing only rejections is a
successful non-mutating result: the revision, dirty state and undo point do not
change.

## Atomicity

SaveEngine resolves every addressed record, validates every action and prepares
all physical byte writes before the first write. It then applies non-overlapping
writes as one rollback-capable transaction. Any failure leaves the snapshot,
revision, dirty state and undo state unchanged. A successful non-empty plan
advances the revision once and creates at most one `apply_repairs` undo point.

## Result

The receipt returns `saveSessionID`, `saveRevision`, `characterID`, `applied`,
and the freshly derived `actions` and `rejected`. `applied` is false only for a
rejected-only selection.

## Errors

The endpoint rejects a missing engine or catalog, malformed or stale revision,
an inactive slot, invalid selected identifiers, a mismatched token, an unknown
session, or a physical target that no longer matches the regenerated plan. It
never falls back to a different record, target value or repair policy.
