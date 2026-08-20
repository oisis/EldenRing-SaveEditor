# SetSummoningPoolActivated

## Overview

`SetSummoningPoolActivated` sets or clears the activation state of one curated
Summoning Pool (Martyr Effigy) in one physical character slot of a save session
that already exists in SaveEngine. It mutates the single activation event flag
bit associated with the pool resource and nothing else.

| Source | What it provides |
|---|---|
| GameCatalog | the pool definition: resolves the resource by `summoningPoolKind` (`summoning_pool`) and `summoningPoolKey`, validates the complete curated list, and extracts its confirmed `summoningPool.activationEventFlagID` |
| SaveEngine | the single bit mutation of that flag in the requested slot under `expectedRevision` control |

Setting `activated: true` sets the event flag bit to `1`. Setting
`activated: false` clears it to `0`. No item, no map state and no second flag is
touched.

The session must have been created earlier by [`LoadSave`](../savesession/load_save.md). `SetSummoningPoolActivated` performs an atomic write to the session's in-memory snapshot. The original save file on disk remains untouched until [`WriteSave`](../savesession/write_save.md) is called.

| | |
|---|---|
| EndpointID | `set_summoning_pool_activated` |
| Kind | Mutation |
| Domain | `world` |
| Supported resource types | `SummoningPoolDocument` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/summoning-pools/activate` of the local explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. |
| Implementation source | [../../../backend/endpoints/world/set_summoning_pool_activated.go](../../../backend/endpoints/world/set_summoning_pool_activated.go) |
| Test source | [../../../backend/endpoints/world/set_summoning_pool_activated_test.go](../../../backend/endpoints/world/set_summoning_pool_activated_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_summoning_pool_activated.go](../../../backend/saveengine/set_summoning_pool_activated.go) |
| SaveEngine tests | [../../../backend/saveengine/set_summoning_pool_activated_test.go](../../../backend/saveengine/set_summoning_pool_activated_test.go) |
| Save access | read/write — mutates the session's private in-memory snapshot; no file on disk is opened |
| Mutation | atomic bit mutation in the event flag section; advances saveRevision by 1 and marks session dirty |

## Input

```go
func SetSummoningPoolActivated(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	summoningPoolKind string,
	summoningPoolKey string,
	activated bool,
	expectedRevision string,
) (SetSummoningPoolActivatedResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; a `nil` engine is rejected. |
| `gameCatalog` | `*gamecatalog.Catalog` | The catalog the pool definitions come from. A `nil` catalog is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. |
| `characterID` | `int` | The physical slot index, `0` to `9`. |
| `summoningPoolKind` | `string` | The GameCatalog resource kind. Must be exactly `"summoning_pool"`. |
| `summoningPoolKey` | `string` | The slug GameCatalog resource key of the pool (e.g. `"stormveil_castle_liftside_chamber"`). |
| `activated` | `bool` | `true` to set the activation bit, `false` to clear it. |
| `expectedRevision` | `string` | The canonical decimal save revision string expected by the caller. Must match the session's current revision. |

## Output

```go
type SetSummoningPoolActivatedResult struct {
	SaveSessionID     string              `json:"saveSessionID"`
	SaveRevision      string              `json:"saveRevision"`
	CharacterID       int                 `json:"characterID"`
	SummoningPoolKind schema.ResourceKind `json:"summoningPoolKind"`
	SummoningPoolKey  string              `json:"summoningPoolKey"`
	Activated         bool                `json:"activated"`
}
```

```json
{
  "saveSessionID": "9f1c…",
  "saveRevision": "1",
  "characterID": 0,
  "summoningPoolKind": "summoning_pool",
  "summoningPoolKey": "stormveil_castle_liftside_chamber",
  "activated": true
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the mutated session. |
| `saveRevision` | `string` | The new decimal save revision string, which is the previous revision plus 1. |
| `characterID` | `int` | The physical slot index, `0` to `9`. |
| `summoningPoolKind` | `string` | The GameCatalog resource kind (`"summoning_pool"`). |
| `summoningPoolKey` | `string` | The GameCatalog resource key. |
| `activated` | `bool` | The new activation state of the pool flag. |

## Summoning Pool resolution

The endpoint uses the same catalog rule as
[`GetSummoningPools`](get_summoning_pools.md) and shares its resolver, so a
reader and a writer can never disagree about which flag belongs to which pool.
A resource is a Summoning Pool only when it is a `summoning_pool` resource
carrying a complete `SummoningPoolDocument`; catalog validation already requires
a known non-empty name, a known non-empty region label and a known non-zero
`activationEventFlagID` inside the confirmed block `670000..670999`.

The shared resolver validates all 213 curated pools before SaveEngine is called,
so a missing document or two pools claiming one activation flag fails before the
slot is touched. The endpoint never accepts a raw flag identifier from its
caller and never derives one from the key or name. The flag stays internal and
is absent from the receipt.

`summoningPoolKind` must be exactly `summoning_pool`. Any other kind is rejected
before the catalog is read, so a colosseum or an item can never be activated
through this endpoint.

## Atomic mutation

SaveEngine restates the confirmed block bound and accepts only block `670`, then
resolves the flag through the single shared `resolveEventFlag` before touching
the session. Under its single mutation lock it validates `characterID`,
`expectedRevision`, slot activity and the complete dynamic offset chain via the
shared `eventFlagSectionStart`. It changes exactly one byte with a bitwise set
or clear, preserving the other seven bits, reads that byte back and rolls it
back if verification fails.

A successful call advances `saveRevision` by exactly one and marks the private
snapshot dirty. Repeating a request that asks for the state already stored is a
successful idempotent state assignment and still advances the revision, like the
other explicit SaveEngine mutations. Every rejection leaves the snapshot,
revision and dirty state unchanged. The undo point is registered under the
operation identifier `set_summoning_pool_activated`.

## Validation and Errors

Every failure fails closed without modifying the snapshot, advancing the revision, or marking the session dirty.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` |
| `gameCatalog` is `nil` | `game catalog is not available` |
| `summoningPoolKind` is not `summoning_pool` | `resource kind "<kind>" is not "summoning_pool"` |
| `summoningPoolKey` is unknown | `unknown resource key "<key>" in kind "summoning_pool"` |
| A curated pool carries no document | `summoning pool "<key>" carries no summoning pool document` |
| Two pools declare one flag | `summoning pools "<k1>" and "<k2>" both declare event flag <id>` |
| `saveSessionID` is empty | `saveSessionID is required` |
| `saveSessionID` is unknown | `unknown save session "<id>"` |
| `characterID` outside `0..9` | `characterID <id> is outside the range 0..9` |
| `expectedRevision` not canonical decimal | `expectedRevision must be a canonical decimal saveRevision; got "<rev>"` |
| `expectedRevision` mismatch | `expectedRevision "<exp>" does not match the current saveRevision "<curr>"` |
| Inactive character slot | `character <id> is not active` |
| Event flag outside block 670 | `event flag <id> lies in block <block>, which is not the confirmed summoning pool block 670` |
| Missing anchor, corrupt declared length or out-of-slot bitfield | A field-specific fail-closed error; no fallback offset is used. |
| Write verification mismatch | `event flag mutation could not be verified; the save is unchanged` (restores original byte) |

## Swagger Route

```
PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/summoning-pools/activate
```

JSON Body:
```json
{
  "summoningPoolKind": "summoning_pool",
  "summoningPoolKey": "stormveil_castle_liftside_chamber",
  "activated": true,
  "expectedRevision": "0"
}
```

The body is strict JSON. Unknown properties, a missing `activated` value and a
non-JSON media type are rejected before the endpoint is called.

## Legacy comparison

SaveForge 1.5.8 and 1.6.10 carry byte-identical implementations of this
mutation. `applySummoningPoolActivated` writes exactly one event flag — the pool
identifier itself — through `db.SetEventFlag` and has no further effect: no
derivative flag, no global flag, no item and no map state. The local read-only
copy in `tmp/er-sf-1.6.10/internal/application/app_world.go` is identical to the
`v1.6.10` tag. SaveForge 2.0 therefore reproduces the same single-flag semantics,
with the flag coming from `SummoningPoolDocument` instead of a raw caller
argument.

The 1.x versions differ only in surrounding infrastructure that 2.0 replaces on
purpose: their per-slot undo stack of depth five versus the single revision-based
undo point, and their diagnostic journal versus the `expectedRevision` receipt.

The bit layout and the dynamic section walk are shared by the PC and PS4
containers after their platform-specific slot bases. Both paths are covered by
synthetic fixtures.

## Command-line Verification

```bash
go test ./backend/saveengine -run '^TestSetSummoningPoolActivated' -count=1 -v
go test -race ./backend/saveengine -run '^TestSetSummoningPoolActivated' -count=1
go test ./backend/endpoints/world -run '^TestSetSummoningPoolActivated' -count=1 -v
go test -race ./backend/endpoints/world -run '^TestSetSummoningPoolActivated' -count=1
go test ./tools/swagger -run 'SetSummoningPoolActivated|OpenAPIDocumentDescribesEveryRoute' -count=1 -v
make test
npm --prefix frontend run build
git diff --check
```
