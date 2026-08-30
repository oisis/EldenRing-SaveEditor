# GetCharacterLoadout

## Overview

`GetCharacterLoadout` returns one coherent, read-only view of the player-facing
loadout of a character in an existing save session. SaveEngine reads all native
groups while holding the session lock once. The endpoint then resolves occupied
positions through GameCatalog and returns names, icons and canonical resource
references without exposing save-layout offsets or requiring the frontend to
interpret game IDs.

| | |
|---|---|
| EndpointID | `get_character_loadout` |
| Kind | Getter |
| Domain | `equipment` |
| Implementation status | implemented |
| Transport | `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/loadout`, available only in the local loopback explorer |
| Wails | `Bridge.GetCharacterLoadout(saveSessionID, characterID)` |
| Save access | read-only session snapshot; no source file is opened |
| Mutation | none |

## Input

```go
func GetCharacterLoadout(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
) (GetCharacterLoadoutResult, error)
```

- `saveSessionID` must identify an already loaded session.
- `characterID` is the physical zero-based slot index `0..9`.
- A missing SaveEngine or GameCatalog dependency is rejected.

## Output

The result always identifies `saveSessionID`, `saveRevision`, `characterID` and
whether the character is active. An active slot has these positional groups:

| Field | Count | Meaning |
|---|---:|---|
| `rightHand`, `leftHand` | 3 each | Armament positions 1–3. Native Unarmed records are `empty`. |
| `arrows`, `bolts` | 2 each | Ammunition positions. |
| `armor` | 4 | Head, chest, arms and legs. Native Bare records are `empty`. |
| `talismans` | 4 | Positions beyond `unlockedTalismanSlots` are `locked`. |
| `quickItems` | 10 | Inventory-backed positions with `ownedItemID` and quantity when occupied. |
| `pouch` | 6 | Inventory-backed positions with `ownedItemID` and quantity when occupied. |
| `physick` | 2 | Crystal Tear positions. |
| `spells` | 12 | Public spell positions with per-spell Memory Slots cost. |

The result also carries `activeQuickItem`, `activeSpellIndex`,
`usedMemorySlots`, `availableMemorySlots` and `unlockedTalismanSlots`.
`saveRevision` scopes every returned `ownedItemID`; clients must not reuse an
owned-item token after the session revision changes.

Each positional record has a `state`:

- `empty` — a native empty sentinel or a confirmed technical empty record;
- `occupied` — a known GameCatalog resource;
- `locked` — an unavailable talisman position.

An inactive or residual character is successful but returns `active: false`,
the current revision, `activeSpellIndex: -1`, and empty group arrays. Its
residual slot data is not read or exposed.

## Validation and fail-closed behaviour

The getter returns no partial loadout when an active slot is malformed. It
rejects, among other cases:

- a missing confirmed native section or a range outside the character slot;
- a Quick Items or Pouch handle, Inventory row, equip index and tail game ID
  that do not all identify the same positive-quantity owned record;
- an occupied game ID absent from GameCatalog or belonging to the wrong family;
- an invalid active spell index or an unknown spell Memory Slots cost;
- equipped spells whose total cost exceeds the available capacity.

There is no fallback lookup, guessed presentation, automatic repair or
mutation. Spell positions are not marked `locked` based on their array index:
Memory Slots are a capacity cost, not a count of unlocked positional cells.

## Ownership and compatibility

- SaveEngine owns native layout, session locking, revision capture and
  Quick/Pouch cross-structure validation.
- GameCatalog owns item identity, canonical aliases, family, presentation and
  spell costs.
- The endpoint owns only the public grouping and projection.
- The frontend consumes this result directly and must not reconstruct the
  loadout from the five older narrow getters.
- The existing raw getters remain available and unchanged for diagnostics and
  lower-level consumers.

PC and PS4 use different slot bases but the same confirmed loadout structures.
Both platform paths are covered by SaveEngine regression tests.

## Verification

```bash
go test ./backend/saveengine -run '^TestGetCharacterLoadoutSnapshot' -count=1
go test ./backend/endpoints/equipment -run '^TestGetCharacterLoadout' -count=1
go test ./tools/swagger -run 'TestSaveCharacterRoutesMatchGetters|TestOpenAPIDocumentMatchesPublicContracts' -count=1
```
