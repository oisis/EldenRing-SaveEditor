# GetEquipmentCandidates

## Overview

`GetEquipmentCandidates` returns one paged, backend-filtered list of the
resources one Equipment slot type currently accepts. It exists so the Equipment
slot pickers never decide compatibility, visibility or ordering themselves: the
backend answers with exactly the page it will also accept from the matching
setter.

| | |
|---|---|
| EndpointID | `get_equipment_candidates` |
| Kind | Getter |
| Domain | `equipment` |
| Implementation status | implemented |
| Transport | Wails only — `Bridge.GetEquipmentCandidates`. The endpoint has no HTTP route, no OpenAPI operation and no page in the Scalar portal, because no explorer surface consumes it. |
| Wails | `Bridge.GetEquipmentCandidates(saveSessionID, characterID, slotType, search, page, pageSize)` |
| Save access | read-only session snapshot; no source file is opened |
| Mutation | none |

## Input

```go
func GetEquipmentCandidates(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	safetyProfile string,
	saveSessionID string,
	characterID int,
	slotType string,
	search string,
	page int,
	pageSize int,
) (GetEquipmentCandidatesResult, error)
```

- `safetyProfile` is the active host setting. The Wails bridge reads it through
  the same `activeSafetyProfile` path every other profile-aware method uses, so
  the interface never sends a profile and a call that bypasses the interface
  cannot reveal a resource the setting hides.
- `saveSessionID` must identify an already loaded session.
- `characterID` is the physical zero-based slot index `0..9`.
- `slotType` is one value of the closed dictionary below. Every other value,
  including the empty string, is rejected.
- `search` is case-insensitive on the candidate name and on the canonical
  resource key. An empty search never filters.
- `page` is one-based; `0` resolves to the first page. `pageSize` `0` resolves
  to `GetEquipmentCandidatesDefaultPageSize` (30). A negative value is rejected.
- A missing SaveEngine or GameCatalog dependency is rejected.

## Supported slot types

| `slotType` | Source | Item family | Setter it feeds |
|---|---|---|---|
| `left_hand`, `right_hand` | Inventory common | `weapon` | `set_equipped_armaments` |
| `head`, `chest`, `arms`, `legs` | Inventory common | `armor` | `set_equipped_armor` |
| `talisman` | Inventory common | `talisman` | `set_equipped_talismans` |
| `quick_item` | Inventory common | `goods` | `set_quick_items` |
| `pouch` | Inventory common | `goods` | `set_pouch_items` |
| `physick` | Inventory common **and** Inventory key | `goods` | `set_physick_mixture` |
| `spell_memory` | GameCatalog | `spell` | `set_equipped_spells` |

Storage is never a source. No Equipment setter addresses a Storage record, so
a Storage item offered as a candidate would advertise a mutation that the
setter rejects.

`arrow` and `bolt` are deliberately absent. No confirmed writer addresses the
ammunition fields, so offering candidates for them would advertise a mutation
that does not exist.

## Compatibility rule

The backend is the only owner of the rule, and it is the rule the owning setter
already enforces:

- an owned slot type requires an existing owned record with a positive
  quantity, the item family of the table above, and a confirmed
  `item.capabilities.equipment` whose `allowedSlots` contains the requested
  slot. That family-and-capability rule is one shared validator, called by this
  getter and by the five setters it feeds, so the two can never disagree;
- `physick` reads both Inventory sections, because `set_physick_mixture` accepts
  a Crystal Tear owned in Inventory common or in Inventory key, and Crystal
  Tears are key items. It additionally requires the exact rule of
  `set_physick_mixture`, asked for through that endpoint's own validator, so
  only owned Crystal Tears qualify;
- `spell_memory` requires the shared spell validator: a confirmed spell game ID
  and a confirmed positive `item.spell.memorySlots`. Ownership is not part of the
  spell contract and is not checked here either;
- an unknown item, an item with an unconfirmed capability and an item of the
  wrong family are never candidates.

Being offered is a necessary condition, never a promise. The setter validates
the complete plan again and may still reject it — for instance when the same
record would be assigned to two positions of one group.

## Visibility

Visibility is the shared safety-profile policy and nothing else:

- an item marked `safety.noDatabase` is never a candidate, under any profile;
- an item marked `banRisk` or `cutContent` is a candidate only under `chaos`;
- `dlc` and `preOrder` are presentation facts and hide nothing.

## Output

```go
type EquipmentCandidate struct {
	Resource    schema.ResourceRef
	OwnedItemID string
	Name        string
	IconPath    string
	Quantity    uint32
	MemorySlots int
	BanRisk     bool
	CutContent  bool
}
```

- `OwnedItemID` is present exactly for the slot types whose setter addresses an
  owned record: the hand, armor, talisman, Quick Item and Pouch slots. It is
  scoped to the returned `saveRevision` and must not be reused after the session
  revision changes. The `physick` and `spell_memory` setters take a catalog
  reference instead, so those candidates carry none.
- `Quantity` is the stored quantity of the owned record and is absent for a
  candidate without an `OwnedItemID`.
- `MemorySlots` is the confirmed capacity cost of a spell and is absent for
  every other slot type. A spell whose cost the catalog does not confirm is not
  offered at all, so a client never has to assume a cost.
- `Name` and `IconPath` are the catalog's own presentation values. `IconPath` is
  embedded catalog metadata, not a filesystem path or a ready URL.
- A duplicate owned record is one candidate per record, because the setter
  addresses records. `physick` is the exception: it commits a catalog reference,
  so two owned copies of the same Crystal Tear — in either Inventory section —
  are one candidate, deduplicated by canonical `ResourceRef`.

The result also carries `saveSessionID`, `saveRevision`, `characterID`,
`active`, the resolved `safetyProfile`, the echoed `slotType`, and `total`,
`page` and `pageSize`. `total` counts every candidate that passed the filters
before paging. An inactive character slot is a valid result with `active: false`
and an empty candidate list.

Candidates are ordered by name, then by canonical key, then by owned identity,
so the order is total and two calls never disagree. A candidate whose name the
catalog does not know sorts last rather than first.

## Errors

- unknown or empty `saveSessionID`, and a `characterID` outside `0..9`;
- an unsupported `slotType`, including `arrow`, `bolt` and the empty string;
- an unknown safety profile;
- a negative `page` or `pageSize`;
- a missing SaveEngine or GameCatalog;
- any failure of the underlying container read, which is reported unchanged.

There is no partial result: a rejected request returns the zero value.

## Local verification

```
go test ./backend/endpoints/equipment/ -run TestGetEquipmentCandidates -count=1
```
