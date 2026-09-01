# SetSaveAccountID

## Overview

`SetSaveAccountID` sets the owner identifier of one SaveEngine session. On PC a
save stores that identity twice: once globally in `UserData10` and once inside
every active character slot. Both representations are game-checked and must
agree, so the endpoint writes all of them together as a single atomic plan.

The endpoint does not read the current identifier back, convert between
platforms, repair unknown slot data, or persist anything. The change lives in
the session's private snapshot until a separate `WriteSave` serializes it and
recalculates the PC MD5 prefixes.

| | |
|---|---|
| EndpointID | `set_save_account_id` |
| Kind | Mutation |
| Domain | `savesession` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PATCH /api/v1/save-sessions/{saveSessionID}/account-id` of the local OpenAPI explorer. The route exists only without `-allow-external-bind`; no Wails binding, CLI command, or frontend reaches it. |
| Implementation source | [../../../backend/endpoints/savesession/set_save_account_id.go](../../../backend/endpoints/savesession/set_save_account_id.go) |
| Test source | [../../../backend/endpoints/savesession/set_save_account_id_test.go](../../../backend/endpoints/savesession/set_save_account_id_test.go) |
| Save access | writes the global and every active slot copy in the private session snapshot |
| GameCatalog access | none |

## Input

```go
func SetSaveAccountID(
	engine *saveengine.Engine,
	saveSessionID string,
	accountID string,
	expectedRevision string,
) (SetSaveAccountIDResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | Existing SaveEngine. A `nil` engine is rejected. |
| `saveSessionID` | `string` | Existing session identifier, matched exactly. |
| `accountID` | `string` | Canonical decimal representation of a `uint64`. |
| `expectedRevision` | `string` | Current canonical decimal revision, compared byte for byte. |

`accountID` is a string, not a number, because JSON and JavaScript lose the
precision of large identifiers. Accepted values are `0` and any unsigned decimal
number without leading zeros that fits into `uint64`. A sign, a prefix, padding,
a separator, surrounding whitespace, an empty value, and a value above the
`uint64` maximum are all rejected.

The local HTTP request uses the session identifier from the path:

```http
PATCH /api/v1/save-sessions/{saveSessionID}/account-id
Content-Type: application/json
```

```json
{
  "accountID": "0",
  "expectedRevision": "0"
}
```

The JSON body is strict. Both fields are required, and unknown fields are
rejected. Neither value is trimmed, normalised, or given a default by the
transport. The `Content-Type` header is required and must be `application/json`;
a media-type parameter such as `; charset=utf-8` is accepted.

The example above deliberately uses `0`. No document, log, error, or test in
this repository states a real account identifier.

## Output

```go
type SetSaveAccountIDResult = saveengine.SetSaveAccountIDResult

type SetSaveAccountIDResult struct {
	saveengine.MutationReceipt
}

type MutationReceipt struct {
	OperationID   string   `json:"operationID"`
	OperationKind string   `json:"operationKind"`
	SaveSessionID string   `json:"saveSessionID"`
	SaveRevision  string   `json:"saveRevision"`
	ChangedScopes []string `json:"changedScopes"`
}
```

The receipt is embedded anonymously, so the JSON result is flat and carries the
five receipt members at its top level. There is no nested `receipt` object and
no domain field of its own:

```json
{
  "operationID": "op-0f1e2d3c4b5a69788796a5b4c3d2e1f0",
  "operationKind": "set_save_account_id",
  "saveSessionID": "save-session-1",
  "saveRevision": "1",
  "changedScopes": ["save.session", "diagnostics.report"]
}
```

The embedded `saveengine.MutationReceipt` is exactly the receipt the central
SaveEngine commit path produced for this execution. Nothing here is
reassembled from the EndpointID, the session, the revision or a scope lookup.

- `operationID` names this one execution. It is opaque and unpredictable.
  Identifiers do not repeat among the receipts issued by one running SaveEngine
  instance. That guarantee does not currently cover application restarts:
  uniqueness across restarts requires a persistent operation journal and stays
  outside stage 3b.1. A rejected call returns the zero result and no
  `operationID` at all.
- `operationKind` is the stable kind of the mutation and is always exactly
  `set_save_account_id`.
- `changedScopes` are exactly `save.session` and `diagnostics.report`,
  in that canonical order.

The identifier itself remains private account data: it never appears in a
result, an error message, or a log, so a rejected value is never echoed back to
the caller. The account identifier reaches no getter, so the mutation
invalidates the session scope and the pinned validation report only.

There is no matching getter. The endpoint sets the identifier; it never reports
one.

## Processing flow

1. Reject a malformed `expectedRevision` and a non-canonical `accountID` before
   any session is resolved.
2. Under the SaveEngine mutex, reject an unknown session, a PS4 session, and a
   stale revision.
3. Read the ten `UserData10` slot activity flags.
4. Resolve the target offset of every active slot along the confirmed locator
   chain, and range-check each one.
5. Apply the global write and all slot writes as one plan: previous bytes are
   captured first, every range is verified after the last write, and any failure
   restores everything already changed.
6. Advance `saveRevision`, mark the session dirty, and retire every
   `ownedItemID` minted under the previous revision.

### Slot locator chain

The global copy sits at a fixed position: bytes `[0x04, 0x0C)` counted from the
start of the `UserData10` data. The four bytes in front of it are metadata and
are never part of the identifier.

The slot copy has no fixed position. It differs between slots of the same save
and between saves, so it is resolved by continuing the confirmed chain that the
event-flag reader already walks from the slot anchor:

1. the event-flag bitfield and its one-byte terminator;
2. five size-prefixed world blocks — field area, world area, world geometry, its
   second block, and the renderer block — each with its own confirmed ceiling;
3. the real 61-byte `PlayerCoordinates` block;
4. the spawn fields, whose two trailing members exist only from the slot
   versions the slot itself declares;
5. `NetMan`, one `uint32` in front of a 128 KB opaque payload;
6. inside the trailing fixed block, `WorldAreaWeather`, `WorldAreaTime`, and
   `BaseVersion` precede the identifier.

The offset is never taken as `SlotSize-8`, never assumed constant, and never
found by scanning the slot for the bytes of an account identifier.

## Atomicity and errors

Every offset is resolved and range-checked before the first byte is written. If
one active slot declares an unusable slot version, an out-of-range world block,
or a field that does not fit inside its slot or the snapshot, the whole
operation is rejected. In that case:

- the snapshot is unchanged;
- `saveRevision` is unchanged;
- `unsavedChanges` is unchanged;
- the OwnedItemID registry is unchanged.

A partially synced save — a new global identifier with stale slot copies — is
the failure mode SaveForge 1.6.10 confirmed the game treats as corrupt. It cannot
be produced here, because either all copies change or none does.

Inactive and residual slots are never read past their activity flag and never
written.

## Platform scope

PC only. A PS4 session is rejected with an explicit error before any
account-identifier field is read or written, because PS4 carries no confirmed
Steam identity. No PSN identifier and no hypothetical PS4 equivalent is invented
or written, and the endpoint performs no platform conversion.

## Difference from SaveForge 1.x

- `1.5.8` updated the global `UserData10` copy alone. That behaviour is not
  reproduced.
- `1.6.10` added propagation into every active slot and confirmed that a
  disagreement between the global and the slot copies makes the game treat the
  save as corrupt.

The 2.0 implementation follows the confirmed `1.6.10` behaviour but shares no
code, type, helper, or package structure with either version.

## Dependencies and transport safety

- The endpoint delegates directly to `backend/saveengine` and calls no other
  endpoint.
- It reads no GameCatalog data and accepts no `GameResource` reference.
- The HTTP route is registered only for the loopback explorer. Starting the
  explorer with `-allow-external-bind` withholds SaveEngine and the route answers
  404.

## Verification

```bash
go test ./backend/saveengine -run '^TestSetSaveAccountID' -count=1
go test -race ./backend/saveengine -run '^TestSetSaveAccountID' -count=1
go test ./backend/endpoints/savesession -run '^TestSetSaveAccountID' -count=1
go test ./tools/swagger -run 'TestSetSaveAccountIDRoute|TestOpenAPIDocumentDescribesEveryRoute' -count=1
```

The tests use only synthetic files created under `t.TempDir()`. No real save is
opened for writing, and no real account identifier appears in any fixture.
