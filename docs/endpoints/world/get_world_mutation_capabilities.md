# GetWorldMutationCapabilities

## Purpose

`GetWorldMutationCapabilities` publishes the World mutation contract: which World
mutations this build supports, what risk the backend attaches to each of them,
why it carries that risk, and which one commits a whole set atomically.

It exists so that no client has to carry that knowledge itself. A frontend
renders a World writer only when the matching capability is present, shows the
risk and the reason it received, and never derives, defaults or upgrades either.

## Contract

| Property | Value |
| --- | --- |
| Endpoint ID | `get_world_mutation_capabilities` |
| Kind | Getter |
| Transport | `GET /api/v1/world/mutation-capabilities` |
| Resource identity | Operation kinds; no catalog resource is named |
| Concurrency | None; the getter never mutates |

The endpoint takes no input at all. It names no save session and no character:
which World mutations exist and what risk they carry is a property of the build,
not of a save. It opens no session, reads no save data and reads no GameCatalog
document.

The result carries one field, `capabilities`, an array of:

| Field | Meaning |
| --- | --- |
| `operationKind` | The `EndpointID` of the World mutation endpoint, which is also the SaveEngine operation kind it commits under |
| `risk` | The backend risk level of every execution of that kind |
| `riskReason` | The backend's own wording of why the kind carries that risk |
| `supportsBulk` | Whether the mutation changes a whole set in one atomic commit |

## Supported operations

Fifteen capabilities are returned, ordered by `operationKind`:

`lock_all_spectral_steed_attires`, `set_bell_bearing_unlocked`,
`set_boss_defeated`, `set_colosseum_unlocked`, `set_cookbook_unlocked`,
`set_fog_of_war_removed`, `set_gesture_unlocked`, `set_grace_visited`,
`set_map_region_revealed`, `set_quest_step`, `set_region_unlocked`,
`set_spectral_steed_attire`, `set_summoning_pool_activated`,
`set_tutorial_unlocked` and `set_whetblade_unlocked`.

The presence of an entry is what declares the operation supported. There is no
`available` flag and no capability for an operation this build does not
implement.

## Risk ownership

`risk` and `riskReason` are not a second risk table. They are read from the same
SaveEngine description the operation history and Review Changes present for a
committed operation, so a World action shows the identical wording before and
after it is applied.

Two properties of that source matter to a client:

- the risk belongs to the *kind*, not to one execution. The elevated ban risk a
  single execution can acquire from GameCatalog data is a property of that
  execution and is reported by the operation history, not here;
- an operation kind SaveEngine does not know is rejected with an error rather
  than published with a default risk. The contract fails closed.

## Bulk support

`supportsBulk` is true for `lock_all_spectral_steed_attires` only. That endpoint
removes the three attire items and resets the appearance flags in one verified
plan, so a failure cannot leave the save wearing an appearance whose item is
gone.

Every other capability is a single-target write. A client that wants to change
many entries performs many mutations; it never emulates a bulk operation by
looping single setters, and there is no generic batch endpoint for World.
