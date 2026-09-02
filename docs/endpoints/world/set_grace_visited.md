# SetGraceVisited

## Overview

`SetGraceVisited` sets or clears the visited state of one curated Site of Grace
in one physical character slot of a save session that already exists in
SaveEngine. It mutates the visit event flag bit declared by the grace resource,
the private door event flag bit when the grace guards a sealed dungeon entrance,
and — for the single confirmed exception Gatefront — the four companion flags the
game co-sets on a first authentic visit. Those companion flags are SaveEngine's
own closed rule, keyed on the Gatefront visit flag; no caller can name them.

| Source | What it provides |
|---|---|
| GameCatalog | the grace definition: resolves the resource by `graceKind` (`grace`) and `graceKey`, validates the complete curated list, and extracts its confirmed `grace.visitEventFlagID` and private `grace.doorEventFlagID` |
| SaveEngine | the atomic multi-bit mutation of the resolved plan in the requested slot under `expectedRevision` control |

`LastRestedGrace` is not written: the game owns that `BonfireId` scalar and sets
it when the player physically rests. No map flag, no region, no inventory item,
no quest flag and no Roundtable Hold flag is touched.

The session must have been created earlier by [`LoadSave`](../savesession/load_save.md). `SetGraceVisited` performs an atomic write to the session's in-memory snapshot. The original save file on disk remains untouched until [`WriteSave`](../savesession/write_save.md) is called.

| | |
|---|---|
| EndpointID | `set_grace_visited` |
| Kind | Mutation |
| Domain | `world` |
| Supported resource types | `GraceDocument` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/graces/visit` of the local explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. |
| Implementation source | [../../../backend/endpoints/world/set_grace_visited.go](../../../backend/endpoints/world/set_grace_visited.go) |
| Test source | [../../../backend/endpoints/world/set_grace_visited_test.go](../../../backend/endpoints/world/set_grace_visited_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_grace_visited.go](../../../backend/saveengine/set_grace_visited.go) |
| SaveEngine tests | [../../../backend/saveengine/set_grace_visited_test.go](../../../backend/saveengine/set_grace_visited_test.go) |
| Save access | read/write — mutates the session's private in-memory snapshot; no file on disk is opened |
| Mutation | atomic bit mutation in the event flag section; advances saveRevision by 1 and marks session dirty |

## Input

```go
func SetGraceVisited(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	graceKind string,
	graceKey string,
	visited bool,
	expectedRevision string,
) (SetGraceVisitedResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; a `nil` engine is rejected. |
| `gameCatalog` | `*gamecatalog.Catalog` | The catalog the grace definitions come from. A `nil` catalog is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. |
| `characterID` | `int` | The physical slot index, `0` to `9`. |
| `graceKind` | `string` | The GameCatalog resource kind. Must be exactly `"grace"`. |
| `graceKey` | `string` | The slug GameCatalog resource key of the grace (e.g. `"limgrave_west_gatefront"`). |
| `visited` | `bool` | `true` to set the visit bit and its confirmed dependencies, `false` to clear the visit and door bits. |
| `expectedRevision` | `string` | The canonical decimal save revision string expected by the caller. Must match the session's current revision. |

## Output

```go
type SetGraceVisitedResult struct {
	saveengine.MutationReceipt
	CharacterID   int                 `json:"characterID"`
	GraceKind     schema.ResourceKind `json:"graceKind"`
	GraceKey      string              `json:"graceKey"`
	Visited       bool                `json:"visited"`
}
```

```json
{
  "operationID": "op-3f9c…",
  "operationKind": "set_grace_visited",
  "saveSessionID": "9f1c…",
  "saveRevision": "1",
  "changedScopes": ["save.session", "world.flags", "diagnostics.report"],
  "characterID": 0,
  "graceKind": "grace",
  "graceKey": "limgrave_west_gatefront",
  "visited": true
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the mutated session. |
| `saveRevision` | `string` | The new decimal save revision string, which is the previous revision plus 1. |
| `characterID` | `int` | The physical slot index, `0` to `9`. |
| `graceKind` | `string` | The GameCatalog resource kind (`"grace"`). |
| `graceKey` | `string` | The GameCatalog resource key. |
| `visited` | `bool` | The new state of the grace visit flag. |

The internal visit flag, the door flag and the companion flags are absent from
the receipt; they stay save-format details.

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
  `set_grace_visited`.
- `changedScopes` are exactly `save.session`, `world.flags`, `diagnostics.report`, in that canonical order.
  This mutation writes World state only, so neither Inventory nor Storage is invalidated.

A committed request identical to the current state still advances `saveRevision`
and still returns a complete receipt with a fresh `operationID`: the central
commit path runs even when no byte changes.

## Grace resolution

The endpoint uses the same catalog rule and the same private resolver as
[`GetGraces`](get_graces.md), so a reader and a writer can never disagree about
which flag belongs to which grace. The shared resolver validates all 419 curated
graces — a missing document or two graces claiming one visit flag — before
SaveEngine is called. The endpoint never accepts a raw flag identifier from its
caller and never derives one from the key or name.

`graceKind` must be exactly `grace`. Any other kind is rejected before the
catalog is read, so a boss or an item can never be flipped through this endpoint.

## Written flags

| Case | `visited: true` | `visited: false` |
|---|---|---|
| Plain grace (`doorEventFlagID` is `0`) | visit bit `= 1` | visit bit `= 0` |
| Catacomb or Hero's Grave grace | visit bit `= 1`, door bit `= 1` | visit bit `= 0`, door bit `= 0` |
| Gatefront (`visitEventFlagID` `76111`) | visit bit `= 1`, plus `60100`, `4680`, `710520` and `4681` `= 1` | visit bit `= 0`; the four companion flags stay untouched |

Nineteen of the 419 curated graces declare a non-zero `doorEventFlagID`, spread
over eighteen event flag blocks. Gatefront is the only confirmed companion case:
its four flags reproduce the "initial Melina accord accepted" state, without
which the game can re-trigger the Melina dialogue or leave Torrent unusable. The
set is deliberately closed — the Roundtable Hold invitation flags, the transient
Melina cutscene triggers and every entry the legacy specification marks as
`needs verification` are excluded.

Companion flags are **SET-only**: they are never cleared on `visited: false`,
because normal progression or the Spectral Steed Whistle item path may also have
set them and clearing would regress a save.

## Atomic mutation

The endpoint hands SaveEngine two identifiers — the visit flag and the optional
door flag of the resolved grace:

```go
func (engine *Engine) SetGraceVisited(
	saveSessionID string,
	characterID int,
	visitEventFlagID uint32,
	doorEventFlagID uint32,
	visited bool,
	expectedRevision string,
) (SetGraceVisitedResult, error)
```

There is no companion parameter. SaveEngine holds the closed Gatefront rule
itself: when `visitEventFlagID` is `76111` and `visited` is `true` it adds
exactly `60100`, `4680`, `710520` and `4681`, and in every other case — any
other grace, and Gatefront on `visited: false` — it writes no companion flag at
all. A caller therefore cannot smuggle an arbitrary progression flag into a save
under the name of a grace visit.

SaveEngine restates the confirmed grace block bound and accepts a visit flag from
blocks `71`, `72`, `73`, `74` and `76` only. A non-zero `doorEventFlagID` must
lie in one of the eighteen confirmed door blocks the curated Graces table
declares; a flag of block `4`, `60`, `71`–`76`, `710` or any other non-door block
is rejected as an unconfirmed door even though `resolveEventFlag` could place it.
It then resolves every identifier through the single shared `resolveEventFlag`
before touching the session.

Under its single mutation lock it validates `characterID`, `expectedRevision`,
slot activity and the complete dynamic offset chain via the shared
`eventFlagSectionStart`, computed exactly once. Flags that share one byte — `4680`
and `4681` do — are merged into a single write, so the plan handed to
`applyByteWrites` covers each byte exactly once and contains no overlapping
range. `applyByteWrites` captures every previous byte, writes, reads back and
restores all of them if any byte fails verification, so no error can leave a
partially applied flag set.

A successful call advances `saveRevision` by exactly one and marks the private
snapshot dirty. Repeating a request that asks for the state already stored is a
successful idempotent state assignment and still advances the revision, like the
other explicit SaveEngine mutations. Every rejection leaves the snapshot,
revision and dirty state unchanged. The undo point is registered under the
operation identifier `set_grace_visited`.

## Validation and Errors

Every failure fails closed without modifying the snapshot, advancing the revision, or marking the session dirty.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` |
| `gameCatalog` is `nil` | `game catalog is not available` |
| `graceKind` is not `grace` | `resource kind "<kind>" is not "grace"` |
| `graceKey` is unknown | `unknown resource key "<key>" in kind "grace"` |
| A curated grace carries no document | `grace "<key>" carries no grace document` |
| Two graces declare one flag | `graces "<k1>" and "<k2>" both declare event flag <id>` |
| `saveSessionID` is empty | `saveSessionID is required` |
| `saveSessionID` is unknown | `unknown save session "<id>"` |
| `characterID` outside `0..9` | `characterID <id> is outside the range 0..9` |
| `expectedRevision` not canonical decimal | `expectedRevision must be a canonical decimal saveRevision; got "<rev>"` |
| `expectedRevision` mismatch | `expectedRevision "<exp>" does not match the current saveRevision "<curr>"` |
| Inactive character slot | `character <id> is not active` |
| Visit flag outside the grace blocks | `event flag <id> lies in block <block>, which is not a confirmed grace block [71 72 73 74 76]` |
| Door flag outside the confirmed door blocks | `event flag <id> lies in block <block>, which is not a confirmed grace door block` |
| A flag in an unsupported block | `event flag <id> lies in block <block>, which this reader does not support` |
| Missing anchor, corrupt declared length or out-of-slot bitfield | A field-specific fail-closed error; no fallback offset is used. |
| Write verification mismatch | `the plan could not be verified` (restores every previous byte) |

## Swagger Route

```
PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/graces/visit
```

JSON Body:
```json
{
  "graceKind": "grace",
  "graceKey": "limgrave_west_gatefront",
  "visited": true,
  "expectedRevision": "0"
}
```

The body is strict JSON. Unknown properties, a missing `visited` value and a
non-JSON media type are rejected before the endpoint is called.

## Legacy comparison

SaveForge 1.5.8, 1.6.10 and 1.7.1 carry semantically identical implementations
of this mutation. `applyGraceVisited` writes the grace flag through
`db.SetEventFlag`, follows it with the `GraceData.DoorFlag` when that field is
non-zero, and applies `CompanionEventFlagsForGrace` on activation only — the
same closed set SaveForge 2.0 keeps inside SaveEngine. `worldGraceFlagIDs` builds
the same flag list for the diagnostic journal. Historical comparisons use the
canonical Git tags directly.

The block-to-BST table these versions embed (`backend/db/data/eventflag_bst.txt`)
is byte-identical between `v1.5.8`, `v1.6.10` and `v1.7.1`, so every block
position this endpoint needs is confirmed in the canonical tags:

| Block | BST position | Used for |
|---|---|---|
| `4` | `4` | Gatefront companions `4680` and `4681` |
| `60` | `10` | Gatefront companion `60100` |
| `710` | `111` | Gatefront companion `710520` |
| `71`–`74`, `76` | `21`–`24`, `26` | visit flags |
| `1033438`, `1036518`, `1037538`, `1038528`, `1039418`, `1039488`, `1040528`, `1041378`, `1043338`, `1043388`, `1043398`, `1045348`, `1045518`, `1045528`, `1047408`, `1048368`, `1050538`, `1050558` | `2791`, `3687`, `3981`, `4254`, `4457`, `4506`, `4814`, `4989`, `5521`, `5556`, `5563`, `6088`, `6207`, `6214`, `6690`, `6942`, `7621`, `7635` | door flags |

Neither version writes `LastRestedGrace`, a map flag or a region, and neither
adds a Roundtable Hold flag; SaveForge 2.0 reproduces exactly that scope. The 1.x
versions differ only in surrounding infrastructure that 2.0 replaces on purpose:
their per-slot undo stack of depth five versus the single revision-based undo
point, and their diagnostic journal versus the `expectedRevision` receipt. Their
per-flag write also tolerated a failing companion flag with a printed warning;
2.0 fails the whole mutation closed instead.

The bit layout and the dynamic section walk are shared by the PC and PS4
containers after their platform-specific slot bases. Both paths are covered by
synthetic fixtures.

## Command-line Verification

```bash
go test ./backend/saveengine -run '^TestSetGraceVisited' -count=1 -v
go test -race ./backend/saveengine -run '^TestSetGraceVisited' -count=1
go test ./backend/endpoints/world -run '^TestSetGraceVisited' -count=1 -v
go test -race ./backend/endpoints/world -run '^TestSetGraceVisited' -count=1
go test ./tools/swagger -run 'SetGraceVisited|OpenAPIDocumentDescribesEveryRoute' -count=1 -v
make test
npm --prefix frontend run build
git diff --check
```
