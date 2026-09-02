# SetColosseumUnlocked

## Overview

`SetColosseumUnlocked` sets or clears the unlock state of one colosseum in one
physical character slot of a save session that already exists in SaveEngine. The
public state of a colosseum is its activation event flag, the same flag
[`GetColosseums`](get_colosseums.md) reports, but a working unlock consists of
four flags per arena plus three global flags. That complete set is SaveEngine's
own closed rule for the three confirmed arenas; no caller can name a flag.

| Source | What it provides |
|---|---|
| GameCatalog | the colosseum definition: resolves the resource by `colosseumKind` (`colosseum`) and `colosseumKey`, validates the complete declared list, and extracts its confirmed `colosseum.unlockEventFlagID` |
| SaveEngine | the closed four-flag arena set, the three global flags, and the atomic multi-bit mutation in the requested slot under `expectedRevision` control |

The physical gate of a colosseum is not an event flag: that state lives in the
`WorldGeom` blob and is deliberately not written. No summoning pool, grace, map
region, item or quest flag is touched.

The session must have been created earlier by [`LoadSave`](../savesession/load_save.md). `SetColosseumUnlocked` performs an atomic write to the session's in-memory snapshot. The original save file on disk remains untouched until [`WriteSave`](../savesession/write_save.md) is called.

| | |
|---|---|
| EndpointID | `set_colosseum_unlocked` |
| Kind | Mutation |
| Domain | `world` |
| Supported resource types | `ColosseumDocument` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/colosseums/unlock` of the local explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. |
| Implementation source | [../../../backend/endpoints/world/set_colosseum_unlocked.go](../../../backend/endpoints/world/set_colosseum_unlocked.go) |
| Test source | [../../../backend/endpoints/world/set_colosseum_unlocked_test.go](../../../backend/endpoints/world/set_colosseum_unlocked_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_colosseum_unlocked.go](../../../backend/saveengine/set_colosseum_unlocked.go) |
| SaveEngine tests | [../../../backend/saveengine/set_colosseum_unlocked_test.go](../../../backend/saveengine/set_colosseum_unlocked_test.go) |
| Save access | read/write — mutates the session's private in-memory snapshot; no file on disk is opened |
| Mutation | atomic bit mutation in the event flag section; advances saveRevision by 1 and marks session dirty |

## Input

```go
func SetColosseumUnlocked(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	colosseumKind string,
	colosseumKey string,
	unlocked bool,
	expectedRevision string,
) (SetColosseumUnlockedResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; a `nil` engine is rejected. |
| `gameCatalog` | `*gamecatalog.Catalog` | The catalog the colosseum definitions come from. A `nil` catalog is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. |
| `characterID` | `int` | The physical slot index, `0` to `9`. |
| `colosseumKind` | `string` | The GameCatalog resource kind. Must be exactly `"colosseum"`. |
| `colosseumKey` | `string` | The slug GameCatalog resource key of the arena (`caelid_colosseum`, `limgrave_colosseum` or `royal_colosseum`). |
| `unlocked` | `bool` | `true` to write the four arena flags and the three global flags, `false` to clear the four arena flags only. |
| `expectedRevision` | `string` | The canonical decimal save revision string expected by the caller. Must match the session's current revision. |

## Output

```go
type SetColosseumUnlockedResult struct {
	saveengine.MutationReceipt
	CharacterID   int                 `json:"characterID"`
	ColosseumKind schema.ResourceKind `json:"colosseumKind"`
	ColosseumKey  string              `json:"colosseumKey"`
	Unlocked      bool                `json:"unlocked"`
}
```

```json
{
  "operationID": "op-3f9c…",
  "operationKind": "set_colosseum_unlocked",
  "saveSessionID": "9f1c…",
  "saveRevision": "1",
  "changedScopes": ["save.session", "world.flags", "diagnostics.report"],
  "characterID": 0,
  "colosseumKind": "colosseum",
  "colosseumKey": "royal_colosseum",
  "unlocked": true
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the mutated session. |
| `saveRevision` | `string` | The new decimal save revision string, which is the previous revision plus 1. |
| `characterID` | `int` | The physical slot index, `0` to `9`. |
| `colosseumKind` | `string` | The GameCatalog resource kind (`"colosseum"`). |
| `colosseumKey` | `string` | The GameCatalog resource key. |
| `unlocked` | `bool` | The new unlock state of the colosseum. |

No event flag identifier appears in the request or in the receipt; the flags stay
save-format details.

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
  `set_colosseum_unlocked`.
- `changedScopes` are exactly `save.session`, `world.flags`, `diagnostics.report`, in that canonical order.
  This mutation writes World state only, so neither Inventory nor Storage is invalidated.

A committed request identical to the current state still advances `saveRevision`
and still returns a complete receipt with a fresh `operationID`: the central
commit path runs even when no byte changes.

## Colosseum resolution

The endpoint uses the same catalog rule and the same private resolver as
[`GetColosseums`](get_colosseums.md), so a reader and a writer can never disagree
about which flag belongs to which arena. The shared resolver validates all three
declared colosseums — a missing document or two colosseums claiming one flag —
before SaveEngine is called. The endpoint never accepts a raw flag identifier
from its caller and never derives one from the key or name.

`colosseumKind` must be exactly `colosseum`. Any other kind is rejected before
the catalog is read.

## Written flags

| Colosseum | Activation | Map POI | NPC / event memory | Matchmaking gate |
|---|---|---|---|---|
| Caelid Colosseum | `60350` | `62720` | `69450` | `710850` |
| Limgrave Colosseum | `60360` | `62730` | `69460` | `710860` |
| Royal Colosseum | `60370` | `62740` | `69470` | `710870` |

The three global flags are `6080` (gameman "any colosseum unlocked"), `60100`
(shared event/map system flag) and `69480` (block 69 global).

| Case | Effect |
|---|---|
| `unlocked: true` | the four flags of the requested arena `= 1`, plus `6080`, `60100` and `69480` `= 1` |
| `unlocked: false` | the four flags of the requested arena `= 0`; the three global flags stay untouched |

The global flags are **SET-only**. They are never cleared, because another
colosseum may still be unlocked and own them, and `60100` is additionally shared
with the Torrent progression that [`SetGraceVisited`](set_grace_visited.md) also
sets for Gatefront. Clearing them would regress a save.

The gate marker `710xxx` is the matchmaking gate flag, not the physical door.
The physical open state of the arena gate is stored in the `WorldGeom` binary
blob and cannot be written from a save editor; the player opens the gate once
in-game and the state persists from then on.

## Atomic mutation

The endpoint hands SaveEngine exactly one identifier — the activation flag of the
resolved colosseum:

```go
func (engine *Engine) SetColosseumUnlocked(
	saveSessionID string,
	characterID int,
	unlockEventFlagID uint32,
	unlocked bool,
	expectedRevision string,
) (SetColosseumUnlockedResult, error)
```

SaveEngine accepts `60350`, `60360` and `60370` only and derives the rest from
its own closed table. There is no fallback that would treat an arbitrary flag of
the supported block `60` as an arena, and a derivative member of a set — for
example `62720` — is not an entry point either. A caller can therefore neither
extend nor shorten the confirmed set.

Every identifier is resolved through the single shared `resolveEventFlag` before
the session is touched. Under its single mutation lock SaveEngine validates
`characterID`, `expectedRevision`, slot activity and the complete dynamic offset
chain via the shared `eventFlagSectionStart`, computed exactly once. Targets
sharing one byte are merged into a single write, so the plan handed to
`applyByteWrites` covers each byte exactly once and contains no overlapping
range. `applyByteWrites` captures every previous byte, writes, reads back and
restores all of them if any byte fails verification, so no error can leave a
partially applied flag set.

A successful call advances `saveRevision` by exactly one and marks the private
snapshot dirty. Repeating a request that asks for the state already stored is a
successful idempotent state assignment and still advances the revision, like the
other explicit SaveEngine mutations. Every rejection leaves the snapshot,
revision and dirty state unchanged. The undo point is registered under the
operation identifier `set_colosseum_unlocked`.

## Validation and Errors

Every failure fails closed without modifying the snapshot, advancing the revision, or marking the session dirty.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` |
| `gameCatalog` is `nil` | `game catalog is not available` |
| `colosseumKind` is not `colosseum` | `resource kind "<kind>" is not "colosseum"` |
| `colosseumKey` is unknown | `unknown resource key "<key>" in kind "colosseum"` |
| A colosseum carries no document | `colosseum "<key>" carries no colosseum document` |
| Two colosseums declare one flag | `colosseums "<k1>" and "<k2>" both declare event flag <id>` |
| `saveSessionID` is empty | `saveSessionID is required` |
| `saveSessionID` is unknown | `unknown save session "<id>"` |
| `characterID` outside `0..9` | `characterID <id> is outside the range 0..9` |
| `expectedRevision` not canonical decimal | `expectedRevision must be a canonical decimal saveRevision; got "<rev>"` |
| `expectedRevision` mismatch | `expectedRevision "<exp>" does not match the current saveRevision "<curr>"` |
| Inactive character slot | `character <id> is not active` |
| Flag is not a confirmed activation flag | `event flag <id> is not a confirmed colosseum unlock flag [60350 60360 60370]` |
| A flag in an unsupported block | `event flag <id> lies in block <block>, which this reader does not support` |
| Missing anchor, corrupt declared length or out-of-slot bitfield | A field-specific fail-closed error; no fallback offset is used. |
| Write verification mismatch | `the plan could not be verified` (restores every previous byte) |

## Swagger Route

```
PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/colosseums/unlock
```

JSON Body:
```json
{
  "colosseumKind": "colosseum",
  "colosseumKey": "royal_colosseum",
  "unlocked": true,
  "expectedRevision": "0"
}
```

The body is strict JSON. Unknown properties, a missing `unlocked` value and a
non-JSON media type are rejected before the endpoint is called.

## Legacy comparison

SaveForge 1.5.8, 1.6.10 and 1.7.1 carry byte-identical implementations of this mutation:
`backend/db/data/summoning_pools.go` declares `ColosseumFlagSets` with the same
three arenas and the same four members each, and `ColosseumGlobalFlags` with
`6080`, `60100` and `69480`; `applyColosseumUnlocked` in `app_world.go`
(`internal/application/app_world.go` in 1.6.10) writes the four arena flags
following `unlocked` and applies the globals on activation only. The canonical
tags do not differ in either file. All three versions document the same
`WorldGeom` limitation for the physical gate.

One legacy behaviour is deliberately **not** reproduced: when the identifier is
not one of the three known activation flags, 1.x falls back to
`ColosseumFlagSet{Activate: colosseumID}` and writes that single unknown flag.
SaveForge 2.0 rejects an unconfirmed identifier instead, so no arbitrary flag of
block `60` can be written under the name of a colosseum.

The block-to-BST table these versions embed (`backend/db/data/eventflag_bst.txt`)
is byte-identical between `v1.5.8`, `v1.6.10` and `v1.7.1`, so every block
position this endpoint needs is confirmed in the canonical tags:

| Block | BST position | Used for |
|---|---|---|
| `6` | `6` | global flag `6080` |
| `60` | `10` | activation flags and the global `60100` |
| `62` | `12` | map POI flags |
| `69` | `19` | NPC flags and the global `69480` |
| `710` | `111` | matchmaking gate flags |

Blocks `60`, `62` and `710` were already resolved by the shared
`resolveEventFlag`; only the confirmed blocks `6` and `69` were added, without a
second resolver, a separate BST table or an `eventFlagID/8` fallback. No current
GameCatalog resource or endpoint requires the neighbouring blocks `5`, `7` or
`70`, so they stay rejected.

The bit layout and the dynamic section walk are shared by the PC and PS4
containers after their platform-specific slot bases. Both paths are covered by
synthetic fixtures.

## Command-line Verification

```bash
go test ./backend/saveengine -run '^TestSetColosseumUnlocked' -count=1 -v
go test -race ./backend/saveengine -run '^TestSetColosseumUnlocked' -count=1
go test ./backend/endpoints/world -run '^TestSetColosseumUnlocked' -count=1 -v
go test -race ./backend/endpoints/world -run '^TestSetColosseumUnlocked' -count=1
go test ./tools/swagger -run 'SetColosseumUnlocked|OpenAPIDocumentDescribesEveryRoute' -count=1 -v
make test
npm --prefix frontend run build
git diff --check
```
