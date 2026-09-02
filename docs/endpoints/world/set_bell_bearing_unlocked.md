# SetBellBearingUnlocked

## Overview

`SetBellBearingUnlocked` sets the handed-in state of one Bell Bearing in one
character slot. The public identity is the exact GameCatalog pair
`(bellBearingKind, bellBearingKey)`; raw game IDs, handles and event flags remain
private.

`unlocked: true` means the Bell Bearing was offered to the Twin Maiden Husks.
The mutation sets its acquisition flag and consumes every matching record from
the confirmed writable sections: Inventory common, Inventory key and Storage
common. `unlocked: false` clears only the flag. It never creates a Bell Bearing.

| | |
|---|---|
| EndpointID | `set_bell_bearing_unlocked` |
| Kind | Mutation |
| Domain | `world` |
| Supported resource types | `ItemDocument: BellBearing` |
| Implementation status | implemented |
| Transport | `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/bell-bearings/unlock`, available only in loopback explorer mode |
| Endpoint source | [../../../backend/endpoints/world/set_bell_bearing_unlocked.go](../../../backend/endpoints/world/set_bell_bearing_unlocked.go) |
| SaveEngine source | [../../../backend/saveengine/set_bell_bearing_unlocked.go](../../../backend/saveengine/set_bell_bearing_unlocked.go) |
| Save access | the private session snapshot only; disk changes require a later `WriteSave` |

## Input

```go
func SetBellBearingUnlocked(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	bellBearingKind string,
	bellBearingKey string,
	unlocked bool,
	expectedRevision string,
) (SetBellBearingUnlockedResult, error)
```

| Parameter | Meaning |
|---|---|
| `saveSessionID` | Exact identifier of an existing SaveEngine session. |
| `characterID` | Physical character slot `0..9`. |
| `bellBearingKind` | Exact resource kind; currently `item`. |
| `bellBearingKey` | Exact eight-digit uppercase hexadecimal resource key. |
| `unlocked` | `true` to mark handed in and consume matching records; `false` to clear only the acquisition flag. |
| `expectedRevision` | Canonical decimal revision that must equal the current session revision. |

## Output

```go
type SetBellBearingUnlockedResult struct {
	saveengine.MutationReceipt
	CharacterID     int                 `json:"characterID"`
	BellBearingKind schema.ResourceKind `json:"bellBearingKind"`
	BellBearingKey  string              `json:"bellBearingKey"`
	Unlocked        bool                `json:"unlocked"`
}
```

The result reports the new revision and the public catalog identity. It does
not expose how many physical records were consumed because those records are an
internal save representation, not the identity of the Bell Bearing.

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
  `set_bell_bearing_unlocked`.
- `changedScopes` are exactly `save.session`, `inventory`, `storage`, `world.flags`, `diagnostics.report`, in that canonical order.
  Handing a Bell Bearing in consumes every matching record and searches Inventory as well as Storage, so both containers are invalidated beside the World flags. A referenced record is refused by the shared removal planner, so the loadout never changes.

A committed request identical to the current state still advances `saveRevision`
and still returns a complete receipt with a fresh `operationID`: the central
commit path runs even when no byte changes.

## Catalog resolution

The endpoint resolves exactly one resource through GameCatalog. It must be a
goods item declaring exactly one known `bell_bearing` unlock. That declaration
supplies the acquisition flag, while `item.gameID` supplies the goods ID used
only inside SaveEngine to recognize the two confirmed record forms:

- the raw goods game ID used by game-created key records;
- the derived `0xB...` goods handle used by editor-created records.

The game ID must be known and have the goods prefix. The client cannot provide
or override either internal identifier.

## Atomic mutation

SaveEngine validates the canonical revision, Bell Bearing event-flag block
`11109`, goods ID, active character slot, event-flag layout and both containers
before the first write.

For `unlocked: true`, every matching record is cleared using the existing
section-specific removal contract:

| Section | Record after removal | Count handling |
|---|---|---|
| Inventory common | handle and quantity zero; physical row retained in the third field | common count decreases by the number removed |
| Inventory key | handle and quantity zero; physical row retained in the third field | key count remains unchanged, matching confirmed legacy behaviour |
| Storage common | all twelve bytes zero | common count decreases by the number removed |
| Storage key | unsupported; a matching record rejects the entire mutation before any write |

All matching records are consumed, including duplicate raw and derived-handle
records. No matching record is also a valid state: the acquisition flag is
still set. A common Inventory record referenced by Equipment, Quick Items or
Pouch rejects the operation instead of leaving a dangling reference.

The event flag, records and participating counts form one verified write plan.
Any write or read-back failure restores every byte already changed. A success
advances `saveRevision` by one and marks the session dirty; a rejection changes
neither.

## HTTP route

```text
PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/bell-bearings/unlock
```

```json
{
  "bellBearingKind": "item",
  "bellBearingKey": "400022CE",
  "unlocked": true,
  "expectedRevision": "0"
}
```

The JSON body is strict. Unknown fields, a missing `unlocked`, an invalid media
type or a malformed `characterID` are rejected before the endpoint is called.
The route is not registered when the explorer uses `-allow-external-bind`.

## Legacy comparison

SaveForge 1.5.8 and 1.6.10 implement the same public meaning:

- unlocking sets the acquisition flag and removes both raw-ID and computed
  goods-handle copies from Inventory and Storage;
- locking clears the flag and leaves the containers unchanged;
- removal scans all matching Inventory common/key records and Storage common
  records, not Storage key.

The 1.6.10 removal path additionally contains GaItem reprojection for physical
item families. Bell Bearings are record-free goods handles, so that path is not
used here and SaveForge 2.0 neither allocates nor repacks GaItem data.

## Verification

```text
go test ./backend/saveengine -run '^TestSetBellBearingUnlocked' -count=1
go test ./backend/endpoints/world -run '^TestSetBellBearingUnlocked' -count=1
go test ./tools/swagger -run 'SetBellBearingUnlocked|OpenAPIDocumentDescribesEveryRoute' -count=1
go test -race ./backend/saveengine -run '^TestSetBellBearingUnlocked' -count=1
go test -race ./backend/endpoints/world -run '^TestSetBellBearingUnlocked' -count=1
make test
npm --prefix frontend run build
git diff --check
```
