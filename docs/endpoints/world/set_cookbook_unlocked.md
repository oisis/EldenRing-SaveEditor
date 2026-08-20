# SetCookbookUnlocked

## Overview

`SetCookbookUnlocked` sets or clears the unlock state of one catalog cookbook
in one physical character slot of a save session that already exists in
SaveEngine. It mutates the single event flag bit associated with the cookbook
resource.

| Source | What it provides |
|---|---|
| GameCatalog | the cookbook definition: resolves the resource by `cookbookKind` (`item`) and `cookbookKey`, validates that it is a goods item declaring exactly one cookbook unlock, and extracts its confirmed `eventFlagID` |
| SaveEngine | the single bit mutation of that `eventFlagID` in the requested slot under `expectedRevision` control |

Setting `unlocked: true` sets the event flag bit to `1`. Setting
`unlocked: false` clears the event flag bit to `0`. No physical inventory or
storage items are created or deleted.

The session must have been created earlier by [`LoadSave`](../savesession/load_save.md). `SetCookbookUnlocked` performs an atomic write to the session's in-memory snapshot. The original save file on disk remains untouched until [`WriteSave`](../savesession/write_save.md) is called.

| | |
|---|---|
| EndpointID | `set_cookbook_unlocked` |
| Kind | Mutation |
| Domain | `world` |
| Supported resource types | `ItemDocument: Cookbook` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/cookbooks/unlock` of the local explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`; with an external bind the route does not exist and answers 404. |
| Implementation source | [../../../backend/endpoints/world/set_cookbook_unlocked.go](../../../backend/endpoints/world/set_cookbook_unlocked.go) |
| Test source | [../../../backend/endpoints/world/set_cookbook_unlocked_test.go](../../../backend/endpoints/world/set_cookbook_unlocked_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_cookbook_unlocked.go](../../../backend/saveengine/set_cookbook_unlocked.go) |
| SaveEngine tests | [../../../backend/saveengine/set_cookbook_unlocked_test.go](../../../backend/saveengine/set_cookbook_unlocked_test.go) |
| Save access | read/write — mutates the session's private in-memory snapshot; no file on disk is opened |
| Mutation | atomic bit mutation in event flag section; advances saveRevision by 1 and marks session dirty |

## Input

```go
func SetCookbookUnlocked(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	cookbookKind string,
	cookbookKey string,
	unlocked bool,
	expectedRevision string,
) (SetCookbookUnlockedResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; a `nil` engine is rejected. |
| `gameCatalog` | `*gamecatalog.Catalog` | The catalog the cookbook definitions come from. A `nil` catalog is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. |
| `characterID` | `int` | The physical slot index, `0` to `9`. |
| `cookbookKind` | `string` | The GameCatalog resource kind. Must be `"item"`. |
| `cookbookKey` | `string` | The 8-digit hex GameCatalog resource key of the cookbook (e.g. `"40002454"`). |
| `unlocked` | `bool` | `true` to set the event flag bit, `false` to clear it. |
| `expectedRevision` | `string` | The canonical decimal save revision string expected by the caller. Must match the session's current revision. |

## Output

```go
type SetCookbookUnlockedResult struct {
	SaveSessionID string              `json:"saveSessionID"`
	SaveRevision  string              `json:"saveRevision"`
	CharacterID   int                 `json:"characterID"`
	CookbookKind  schema.ResourceKind `json:"cookbookKind"`
	CookbookKey   string              `json:"cookbookKey"`
	Unlocked      bool                `json:"unlocked"`
}
```

```json
{
  "saveSessionID": "9f1c…",
  "saveRevision": "1",
  "characterID": 0,
  "cookbookKind": "item",
  "cookbookKey": "40002454",
  "unlocked": true
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the mutated session. |
| `saveRevision` | `string` | The new decimal save revision string, which is the previous revision plus 1. |
| `characterID` | `int` | The physical slot index, `0` to `9`. |
| `cookbookKind` | `string` | The GameCatalog resource kind (`"item"`). |
| `cookbookKey` | `string` | The GameCatalog resource key. |
| `unlocked` | `bool` | The new unlock state of the cookbook flag. |

## Cookbook resolution

The endpoint uses the same catalog rule as `GetCookbooks`. A resource is a
cookbook only when it is an `ItemDocument` of family `goods` with exactly one
`item.unlocks` entry whose known kind is `cookbook`. That entry is the only
source of its `eventFlagID`, name and category. The endpoint never accepts the
raw flag identifier from its caller and never derives it from the key, name,
icon or acquisition data.

The shared resolver validates the complete cookbook catalog before SaveEngine
is called. An unknown field, a duplicate event flag or two cookbook unlocks in
one resource therefore fails before the slot is touched. The event flag ID stays
internal and is absent from the receipt.

## Atomic mutation

SaveEngine resolves the flag before touching the session and accepts only the
confirmed cookbook blocks `67` and `68`. Under its single mutation lock it then
validates `characterID`, `expectedRevision`, slot activity and the complete
dynamic offset chain. It changes exactly one byte with a bitwise set or clear,
preserving the other seven bits, reads that byte back and rolls it back if
verification fails.

A successful call advances `saveRevision` by exactly one and marks the private
snapshot dirty. Repeating a request that asks for the state already stored is a
successful idempotent state assignment and still advances the revision, like
the other explicit SaveEngine mutations. Every rejection leaves the snapshot,
revision and dirty state unchanged.

## Validation and Errors

Every failure fails closed without modifying the snapshot, advancing the revision, or marking the session dirty.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` |
| `gameCatalog` is `nil` | `game catalog is not available` |
| `saveSessionID` is empty | `saveSessionID is required` |
| `saveSessionID` is unknown | `unknown save session "<id>"` |
| `characterID` outside `0..9` | `characterID <id> is outside the range 0..9` |
| `expectedRevision` not canonical decimal | `expectedRevision must be a canonical decimal saveRevision; got "<rev>"` |
| `expectedRevision` mismatch | `expectedRevision "<exp>" does not match the current saveRevision "<curr>"` |
| Inactive character slot | `character <id> is not active` |
| Resource not found in GameCatalog | `unknown resource key "<key>" in kind "<kind>"` |
| Item family not `goods` | `cookbook "<key>" has item family "<family>", want "goods"` |
| Resource has 0 or >1 cookbook unlocks | `resource kind "<kind>" key "<key>" declares no cookbook unlock` / `cookbook "<key>" declares <n> cookbook unlocks, want exactly one` |
| Catalog event flag duplicate conflict | `cookbooks "<k1>" and "<k2>" both declare event flag <id>` |
| Event flag outside blocks 67 and 68 | `event flag <id> lies in block <block>, which this reader does not support` |
| Missing anchor, corrupt declared length or out-of-slot bitfield | A field-specific fail-closed error; no fallback offset is used. |
| Write verification mismatch | `event flag mutation could not be verified; the save is unchanged` (restores original byte) |

## Swagger Route

```
PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/cookbooks/unlock
```

JSON Body:
```json
{
  "cookbookKind": "item",
  "cookbookKey": "40002454",
  "unlocked": true,
  "expectedRevision": "0"
}
```

The body is strict JSON. Unknown properties, a missing `unlocked` value and a
non-JSON media type are rejected before the endpoint is called.

## Legacy comparison

SaveForge 1.5.8 and 1.6.10 have the same cookbook mutation: both change the
event flag and also add or remove the corresponding physical Key Items record.
SaveForge 2.0 intentionally does not reproduce that coupling. `GetCookbooks`
already defines the event flag as the single source of the reported unlock
state, while the current item writer has no confirmed public contract for the
Inventory key section. This endpoint therefore changes the flag only and never
calls an inventory endpoint.

The bit layout and the dynamic section walk are shared by the PC and PS4
containers after their platform-specific slot bases. Both paths are covered by
synthetic fixtures. Semantic PS4 validation against a controlled console
before/after pair remains deferred in `TODO.md` and is not claimed here.

## Command-line Verification

```bash
go test ./backend/saveengine -run '^TestSetCookbookUnlocked' -count=1 -v
go test -race ./backend/saveengine -run '^TestSetCookbookUnlocked' -count=1
go test ./backend/endpoints/world -run '^TestSetCookbookUnlocked' -count=1 -v
go test -race ./backend/endpoints/world -run '^TestSetCookbookUnlocked' -count=1
go test ./tools/swagger -run 'SetCookbookUnlocked|OpenAPIDocumentDescribesEveryRoute' -count=1 -v
make test
npm --prefix frontend run build
git diff --check
```
