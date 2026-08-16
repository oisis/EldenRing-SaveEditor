# SetFogOfWarRemoved

## Purpose

`SetFogOfWarRemoved` removes the global, cosmetic Fog of War overlay of one
character slot. The overlay is the grey "you have not been here yet" layer drawn
over the world map. Removing it is a single in-place fill of one confirmed
bitfield.

The endpoint is not a map-region operation. It names no region, resolves no
GameCatalog resource, reveals no map-region visibility flag, adds no item and
does not change `UnlockedRegions`.

The session must already exist through [`LoadSave`](../savesession/load_save.md).
The mutation changes only its private in-memory snapshot; [`WriteSave`](../savesession/write_save.md)
is still required to persist it.

| | |
|---|---|
| EndpointID | `set_fog_of_war_removed` |
| Kind | Mutation |
| Domain | `world` |
| Supported resource types | — |
| Implementation status | implemented (`removed: true` only) |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/fog-of-war` in the local explorer; the route is absent when it runs with `-allow-external-bind` |
| Implementation source | [../../../backend/endpoints/world/set_fog_of_war_removed.go](../../../backend/endpoints/world/set_fog_of_war_removed.go) |
| Endpoint tests | [../../../backend/endpoints/world/set_fog_of_war_removed_test.go](../../../backend/endpoints/world/set_fog_of_war_removed_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_fog_of_war_removed.go](../../../backend/saveengine/set_fog_of_war_removed.go) |
| SaveEngine tests | [../../../backend/saveengine/set_fog_of_war_removed_test.go](../../../backend/saveengine/set_fog_of_war_removed_test.go) |

## Input

```go
func SetFogOfWarRemoved(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	removed bool,
	expectedRevision string,
) (SetFogOfWarRemovedResult, error)
```

| Parameter | Meaning |
|---|---|
| `saveSessionID` | Existing save session identifier. |
| `characterID` | Physical character slot, `0` to `9`. |
| `removed` | Must be `true`. See [Only `true` is supported](#only-true-is-supported). |
| `expectedRevision` | Canonical decimal revision that must equal the current session revision. |

## Output

```go
type SetFogOfWarRemovedResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	Removed       bool   `json:"removed"`
}
```

## Only `true` is supported

`removed: false` is rejected by SaveEngine before the session is opened or read
and before any byte is written.

The bitfield is a flat per-tile exploration mask whose bit-to-tile mapping is
unknown. Zeroing it would not restore an earlier exploration state — it would
destroy the state the save still carries, and it is not what the game writes when
a tile is unexplored in a partially explored slot. Neither SaveForge 1.5.8 nor
1.6.8 implemented an inverse operation.

The endpoint therefore has no "restore Fog of War" behavior, and none is
advertised in OpenAPI or in the explorer.

## Mutation semantics

The field is located dynamically, because the `UnlockedRegions` list in front of
it is variable-length:

```text
afterRegs = <first byte behind the UnlockedRegions list>
start     = afterRegs + 0x087E
end       = afterRegs + 0x10B0   (inclusive)
size      = 0x833 = 2099 bytes
```

`afterRegs` is not resolved by a second layout parser. SaveEngine reuses the one
locator [`GetRegions`](get_regions.md) already owns: the confirmed gesture anchor
chain resolves `GestureGameData`, the region count stands directly behind that
fixed block, and the declared IDs follow it. Every declared length is widened to
`int64` before it is multiplied or added, so a corrupt count cannot wrap into a
small, seemingly valid offset. A count above the accepted maximum, a list
reaching past the slot, and a field reaching past the slot are hard errors raised
before the first write.

The field is not validated separately against the file, because that bound is
already guaranteed: the mandatory undo point captured for this operation reads
the complete physical character slot and fails closed if it cannot, the locator
checks the field against the end of that slot, and the atomic byte plan performs
its own bounds check with rollback before it writes.

Under one revision-controlled operation SaveEngine validates the character index,
`expectedRevision`, slot activity and the complete resolved range, then applies
one atomic byte plan. The prefix in front of `+0x087E` holds structured horse and
bloodstain data and the byte behind `+0x10B0` belongs to `MenuProfile`; neither
is ever written. No data is shifted, no slot is rebuilt and no repacking occurs.

Any validation or verification error leaves the snapshot, dirty state, undo point
and revision unchanged. Success advances `saveRevision` by exactly one and
creates an undo point under `set_fog_of_war_removed`. Repeating the call is a
no-op on the bytes and a normal commit on the session.

PC and PS4 differ only in the container base of the slot; the dynamic layout
inside it is decoded identically.

## HTTP route

```text
PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/fog-of-war
```

```json
{
  "removed": true,
  "expectedRevision": "0"
}
```

The body must use `application/json`, rejects unknown fields, and requires an
explicit boolean `removed` value.

## Validation and errors

Every rejection is fail-closed.

| Condition | Behaviour |
|---|---|
| nil SaveEngine | rejected before any save read |
| `removed: false` | rejected by SaveEngine before the session is opened; never written as zeros |
| non-canonical or stale `expectedRevision` | rejected without advancing revision |
| out-of-range `characterID` | rejected before the slot is read |
| inactive character slot | rejected without reading residual slot data |
| missing gesture anchor | rejected before mutation |
| region count above the accepted maximum | rejected before mutation |
| region list or Fog of War field outside the slot | rejected before mutation |
| failed write verification | the complete plan is rolled back |

## Legacy comparison

SaveForge 1.5.8 (`app_world.go::RemoveFogOfWar`) and SaveForge 1.6.8
(`internal/application/app_world.go::RemoveFogOfWar`) are byte-identical for this
operation, including `resolveAfterRegs` and the `FoWBlobStart` / `FoWBlobEnd`
constants `0x087E` and `0x10B0`. Their `spec/27-map-reveal.md` §9 is identical in
both tags as well and documents the same 2099-byte range, the same in-place fill
with `0xFF`, the same idempotence and the explicit absence of a selective or
inverse operation.

SaveForge 2.0 reimplements that confirmed result on its own architecture. It adds
what 1.x lacked: a canonical `expectedRevision`, an explicit activity check, a
range validated against the slot before the first write, one atomic plan with
rollback, and a dedicated undo operation. It does not import
legacy code and does not restore retired allocation, repacking or slot-rebuild
behavior.

## Verification

```bash
go test ./backend/saveengine -run 'SetFogOfWarRemoved|GetRegions' -count=1
go test -race ./backend/saveengine -run 'SetFogOfWarRemoved|GetRegions' -count=1
go test ./backend/endpoints/world -run 'SetFogOfWarRemoved|GetRegions' -count=1
go test ./tools/swagger -run 'SetFogOfWarRemoved|OpenAPIDocumentDescribesEveryRoute' -count=1
make test
npm --prefix frontend run build
git diff --check
```
