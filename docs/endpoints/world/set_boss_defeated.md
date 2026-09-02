# SetBossDefeated

## Overview

`SetBossDefeated` sets or clears the defeated state of one curated boss in one
physical character slot of a save session that already exists in SaveEngine. It
mutates the single synchronized defeat event flag bit declared by the boss
resource and nothing else.

| Source | What it provides |
|---|---|
| GameCatalog | the boss definition: resolves the resource by `bossKind` (`boss`) and `bossKey`, validates the complete curated list, and extracts its confirmed `boss.defeatEventFlagID` |
| SaveEngine | the single bit mutation of that flag in the requested slot under `expectedRevision` control |

Setting `defeated: true` sets the event flag bit to `1`. Setting
`defeated: false` clears it to `0`. No reward, no rune award, no Remembrance
item, no arena state flag, no quest flag, no grace flag and no second event flag
is touched. The endpoint represents exactly the state of the synchronized flag
the `BossDocument` declares; it is not a full replay or full undo of a boss
fight.

The session must have been created earlier by [`LoadSave`](../savesession/load_save.md). `SetBossDefeated` performs an atomic write to the session's in-memory snapshot. The original save file on disk remains untouched until [`WriteSave`](../savesession/write_save.md) is called.

| | |
|---|---|
| EndpointID | `set_boss_defeated` |
| Kind | Mutation |
| Domain | `world` |
| Supported resource types | `BossDocument` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/bosses/defeat` of the local explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. |
| Implementation source | [../../../backend/endpoints/world/set_boss_defeated.go](../../../backend/endpoints/world/set_boss_defeated.go) |
| Test source | [../../../backend/endpoints/world/set_boss_defeated_test.go](../../../backend/endpoints/world/set_boss_defeated_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_boss_defeated.go](../../../backend/saveengine/set_boss_defeated.go) |
| SaveEngine tests | [../../../backend/saveengine/set_boss_defeated_test.go](../../../backend/saveengine/set_boss_defeated_test.go) |
| Save access | read/write — mutates the session's private in-memory snapshot; no file on disk is opened |
| Mutation | atomic bit mutation in the event flag section; advances saveRevision by 1 and marks session dirty |

## Input

```go
func SetBossDefeated(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	bossKind string,
	bossKey string,
	defeated bool,
	expectedRevision string,
) (SetBossDefeatedResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; a `nil` engine is rejected. |
| `gameCatalog` | `*gamecatalog.Catalog` | The catalog the boss definitions come from. A `nil` catalog is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. |
| `characterID` | `int` | The physical slot index, `0` to `9`. |
| `bossKind` | `string` | The GameCatalog resource kind. Must be exactly `"boss"`. |
| `bossKey` | `string` | The slug GameCatalog resource key of the boss (e.g. `"stormveil_castle_godrick_the_grafted"`). |
| `defeated` | `bool` | `true` to set the defeat bit, `false` to clear it. |
| `expectedRevision` | `string` | The canonical decimal save revision string expected by the caller. Must match the session's current revision. |

## Output

```go
type SetBossDefeatedResult struct {
	saveengine.MutationReceipt
	CharacterID   int                 `json:"characterID"`
	BossKind      schema.ResourceKind `json:"bossKind"`
	BossKey       string              `json:"bossKey"`
	Defeated      bool                `json:"defeated"`
}
```

```json
{
  "operationID": "op-3f9c…",
  "operationKind": "set_boss_defeated",
  "saveSessionID": "9f1c…",
  "saveRevision": "1",
  "changedScopes": ["save.session", "world.flags", "diagnostics.report"],
  "characterID": 0,
  "bossKind": "boss",
  "bossKey": "stormveil_castle_godrick_the_grafted",
  "defeated": true
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the mutated session. |
| `saveRevision` | `string` | The new decimal save revision string, which is the previous revision plus 1. |
| `characterID` | `int` | The physical slot index, `0` to `9`. |
| `bossKind` | `string` | The GameCatalog resource kind (`"boss"`). |
| `bossKey` | `string` | The GameCatalog resource key. |
| `defeated` | `bool` | The new state of the boss defeat flag. |

The result embeds the shared `MutationReceipt` anonymously, so the JSON stays
flat: `operationID`, `operationKind`, `saveSessionID`, `saveRevision` and
`changedScopes` are top-level members beside the domain fields, and there is no
nested `receipt` object.

The embedded receipt is exactly the one the central SaveEngine commit path
produced for this execution. Nothing here is reassembled from the EndpointID,
the session, the revision or a scope lookup.

- `operationID` names this one execution. It is opaque and unpredictable.
  Identifiers do not repeat among the receipts issued by one running SaveEngine
  instance. That guarantee does not currently cover application restarts:
  uniqueness across restarts requires a persistent operation journal and stays
  outside this stage. A rejected call returns the complete zero result and no
  `operationID` at all.
- `operationKind` is the stable kind of the mutation and is always exactly
  `set_boss_defeated`.
- `changedScopes` are exactly `save.session`, `world.flags`, `diagnostics.report`, in that canonical order.
  This mutation writes World state only, so neither Inventory nor Storage is invalidated.

A committed request identical to the current state still advances `saveRevision`
and still returns a complete receipt with a fresh `operationID`: the central
commit path runs even when no byte changes.

## Boss resolution

The endpoint uses the same catalog rule as [`GetBosses`](get_bosses.md) and
shares its resolver, so a reader and a writer can never disagree about which flag
belongs to which boss. A resource is a boss only when it is a `boss` resource
carrying a complete `BossDocument`; catalog validation already requires a known
non-empty name, a known non-empty region label and a known
`defeatEventFlagID` inside the confirmed block `9000..9999`.

The shared resolver validates all 110 curated bosses before SaveEngine is called,
so a missing document or two bosses claiming one defeat flag fails before the
slot is touched. The endpoint never accepts a raw flag identifier from its caller
and never derives one from the key or name. The flag stays internal and is absent
from the receipt.

The curated table covers only bosses with a confirmed synchronized defeat flag.
The roughly 97 open-world field bosses that carry a per-map flag only are outside
this contract and are never synthesised, so they cannot be addressed here.

`bossKind` must be exactly `boss`. Any other kind is rejected before the catalog
is read, so a summoning pool or an item can never be flipped through this
endpoint.

## Atomic mutation

SaveEngine restates the confirmed block bound and accepts only block `9`, then
resolves the flag through the single shared `resolveEventFlag` before touching
the session. Under its single mutation lock it validates `characterID`,
`expectedRevision`, slot activity and the complete dynamic offset chain via the
shared `eventFlagSectionStart`. It changes exactly one byte with a bitwise set or
clear, preserving the other seven bits, reads that byte back and rolls it back if
verification fails.

A successful call advances `saveRevision` by exactly one and marks the private
snapshot dirty. Repeating a request that asks for the state already stored is a
successful idempotent state assignment and still advances the revision, like the
other explicit SaveEngine mutations. Every rejection leaves the snapshot,
revision and dirty state unchanged. The undo point is registered under the
operation identifier `set_boss_defeated`.

## Validation and Errors

Every failure fails closed without modifying the snapshot, advancing the revision, or marking the session dirty.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` |
| `gameCatalog` is `nil` | `game catalog is not available` |
| `bossKind` is not `boss` | `resource kind "<kind>" is not "boss"` |
| `bossKey` is unknown | `unknown resource key "<key>" in kind "boss"` |
| A curated boss carries no document | `boss "<key>" carries no boss document` |
| Two bosses declare one flag | `bosses "<k1>" and "<k2>" both declare event flag <id>` |
| `saveSessionID` is empty | `saveSessionID is required` |
| `saveSessionID` is unknown | `unknown save session "<id>"` |
| `characterID` outside `0..9` | `characterID <id> is outside the range 0..9` |
| `expectedRevision` not canonical decimal | `expectedRevision must be a canonical decimal saveRevision; got "<rev>"` |
| `expectedRevision` mismatch | `expectedRevision "<exp>" does not match the current saveRevision "<curr>"` |
| Inactive character slot | `character <id> is not active` |
| Event flag outside block 9 | `event flag <id> lies in block <block>, which is not the confirmed boss block 9` |
| Missing anchor, corrupt declared length or out-of-slot bitfield | A field-specific fail-closed error; no fallback offset is used. |
| Write verification mismatch | `event flag mutation could not be verified; the save is unchanged` (restores original byte) |

## Swagger Route

```
PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/bosses/defeat
```

JSON Body:
```json
{
  "bossKind": "boss",
  "bossKey": "stormveil_castle_godrick_the_grafted",
  "defeated": true,
  "expectedRevision": "0"
}
```

The body is strict JSON. Unknown properties, a missing `defeated` value and a
non-JSON media type are rejected before the endpoint is called.

## Legacy comparison

SaveForge 1.5.8, 1.6.10 and 1.7.1 carry byte-identical implementations of this
mutation; the only difference between the 1.5.8 and later files is the package
name (`main` in 1.5.8, `application` later). `applyBossDefeated` writes exactly
one event flag — the boss identifier itself — through `db.SetEventFlag` and has
no further effect: no arena state flag, no reward, no Remembrance drop and no
world state flag. Historical comparisons use the canonical Git tags directly.

Both versions ship the identical design document `spec/38-boss-multiflag.md`,
which describes a multi-flag kill rework with status `Planned`. It was never
implemented in either release, so it is not part of the confirmed contract and
SaveForge 2.0 does not reproduce it. SaveForge 2.0 therefore reproduces the same
single-flag semantics, with the flag coming from `BossDocument` instead of a raw
caller argument.

The 1.x versions differ only in surrounding infrastructure that 2.0 replaces on
purpose: their per-slot undo stack of depth five versus the single revision-based
undo point, and their diagnostic journal versus the `expectedRevision` receipt.

The bit layout and the dynamic section walk are shared by the PC and PS4
containers after their platform-specific slot bases. Both paths are covered by
synthetic fixtures.

## Command-line Verification

```bash
go test ./backend/saveengine -run '^TestSetBossDefeated' -count=1 -v
go test -race ./backend/saveengine -run '^TestSetBossDefeated' -count=1
go test ./backend/endpoints/world -run '^TestSetBossDefeated' -count=1 -v
go test -race ./backend/endpoints/world -run '^TestSetBossDefeated' -count=1
go test ./tools/swagger -run 'SetBossDefeated|OpenAPIDocumentDescribesEveryRoute' -count=1 -v
make test
npm --prefix frontend run build
git diff --check
```
