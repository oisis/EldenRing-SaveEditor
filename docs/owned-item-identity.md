# OwnedItemID and saveRevision — shared Inventory contract

> **Status: approved. The identity tasks, the first mutation, and `WriteSave`
> are implemented. `WriteSave` and `inventory.SetOwnedItemQuantity` are exposed
> by the local explorer. Every other public setter endpoint is still
> contract-only.**
>
> This document is the design record of the contract, not endpoint
> documentation. The implemented endpoints are described in
> [`docs/endpoints/`](endpoints/README.md), `tools/swagger/openapi.json` and the
> Scalar portal, which stay the single sources of truth for them. Everything
> still marked here as proposed — the remaining setter *endpoints*, and every
> rule that depends on a mutation SaveEngine does not yet perform — is not a
> runtime fact until a separate, explicitly approved task implements it.
>
> **What the first mutation does and does not mean.** It changes the four
> quantity bytes of one record inside the session's private in-memory snapshot
> and opens no file itself. A separate successful `WriteSave` persists that
> snapshot, advances the revision again, retires previous identities, and clears
> `SessionInfo.UnsavedChanges`.

| | |
|---|---|
| Scope | `OwnedItemID` and `saveRevision`, shared by the Inventory and Equipment surfaces |
| Proposed owner | `backend/saveengine` (one component), never an endpoint |
| Affected contracts today | originally 12 contract-only endpoint files declaring `ownedItemID`, `weaponOwnedItemID` or `orderedOwnedItemIDs`; `get_owned_item.go` is implemented since Task 3 and `set_owned_item_quantity.go` since Task 5, so 10 remain contract-only |
| Affects implemented code | yes since Tasks 1, 2a, 2b, 3, 4 and 5 — `GetInventory`, `GetStorage` and `GetOwnedItem` are implemented, `saveengine.Engine.SetOwnedItemQuantity` is the first implemented mutation, and `inventory.SetOwnedItemQuantity` is the first implemented mutation *endpoint*; every other setter endpoint is still a separate later task |
| Transport | `GetOwnedItem`, `SetOwnedItemQuantity` and `WriteSave` are transport-exposed by the local explorer |

---

## 1. Problem and boundaries

### 1.1 Why `OwnedItemID` is needed

Twelve endpoint contract files already declare an owned-item identity as an
input variable, and none of them can be implemented without a definition of what
that value is:

| Contract file | Endpoint | Declared variable |
|---|---|---|
| `backend/endpoints/inventory/get_owned_item.go` | `GetOwnedItem` | `ownedItemID` |
| `backend/endpoints/inventory/set_owned_item_quantity.go` | `SetOwnedItemQuantity` | `ownedItemID` |
| `backend/endpoints/inventory/remove_owned_item.go` | `RemoveOwnedItem` | `ownedItemID` |
| `backend/endpoints/inventory/move_owned_item_to_inventory.go` | `MoveOwnedItemToInventory` | `ownedItemID` |
| `backend/endpoints/inventory/move_owned_item_to_storage.go` | `MoveOwnedItemToStorage` | `ownedItemID` |
| `backend/endpoints/inventory/set_inventory_order.go` | `SetInventoryOrder` | `orderedOwnedItemIDs` |
| `backend/endpoints/inventory/set_storage_order.go` | `SetStorageOrder` | `orderedOwnedItemIDs` |
| `backend/endpoints/inventory/set_weapon_upgrade_level.go` | `SetWeaponUpgradeLevel` | `ownedItemID` |
| `backend/endpoints/inventory/set_weapon_infusion.go` | `SetWeaponInfusion` | `ownedItemID` |
| `backend/endpoints/inventory/set_weapon_ash_of_war.go` | `SetWeaponAshOfWar` | `weaponOwnedItemID` |
| `backend/endpoints/inventory/set_spirit_ash_upgrade_level.go` | `SetSpiritAshUpgradeLevel` | `ownedItemID` |
| `backend/endpoints/equipment/set_equipped_talismans.go` | `SetEquippedTalismans` | `orderedOwnedItemIDs` |

`SetEquippedTalismans` is an existing contract file in
`backend/endpoints/equipment/`, not merely a row in the `tmp/app-se` spec. It is
the only one of the twelve outside `backend/endpoints/inventory/`, which is why
this contract is shared by the Inventory *and* Equipment surfaces.

The need is therefore not speculative. It is already declared in those twelve
files and locked by `backend/endpoints/contract/endpoint_structure_test.go`,
which reads the contract header of each file.

The catalog identity pair `kind` + `key` cannot replace it. `kind`/`key`
identifies *what an item is*, not *which owned instance* is meant. A character
can hold several distinct records of the same `kind`/`key` — different upgrade
levels, different infusions, different Ash of War, or simply two rows of the
same talisman in the same container. Every endpoint in the table above operates
on one instance, not on an item type.

### 1.2 Why `GetOwnedItem` must not define it locally

`GetOwnedItem` is a consumer of the identity, not its producer. If it defined
the rule locally:

- the two producers (`GetInventory`, `GetStorage`) and the nine mutations would
  each need their own copy of the same decoding rule, which is exactly the
  parallel-implementation pattern `AGENTS.md` forbids;
- `SetInventoryOrder` and `SetStorageOrder` accept a *complete ordered list* of
  IDs produced by a getter, so producer and consumer must agree byte for byte;
- `tmp/app-se/endpoints.md` §"Niezależność endpointów" states that an endpoint
  never calls another endpoint and that a shared need is served by the owner of
  the data. An identity defined inside `GetOwnedItem` would force either
  duplication or a forbidden endpoint-to-endpoint call.

### 1.3 Rule owner

The proposed owner is **one component inside `backend/saveengine`**, the same
component that already owns session lifecycle and raw record reading. The
argument is the one already recorded in `tmp/app-se/architecture.md` §3:
`SaveEngine` is the single source of truth for reading and mutating a save, and
endpoints are thin. Concretely:

- SaveEngine mints every `OwnedItemID`;
- SaveEngine is the only component that can resolve one back to a physical
  record;
- SaveEngine owns `saveRevision` and rejects a stale one;
- endpoints validate presence and type of the string and pass it through. They
  never parse it, never construct one, never compare two of them for physical
  meaning.

Frontend, tests and automation consume the same opaque string through the same
endpoints.

### 1.4 The twelve consumers take `saveSessionID`

**Decided here, not deferred.** The rule covers exactly the twelve contract
files listed in §1.1 — the files that produce or consume `ownedItemID`,
`weaponOwnedItemID` or `orderedOwnedItemIDs` — plus the two implemented
producers, which already satisfy it. It is **not** a general rule for every
endpoint that touches save state; no endpoint outside this list is in scope.

| Contract | Today |
|---|---|
| `GetInventory`, `GetStorage` | already declare `saveSessionID` — the producers need no change |
| `GetOwnedItem` | satisfied since Task 3: declares `saveSessionID`, `characterID`, `ownedItemID`, in that exact order (`backend/endpoints/inventory/get_owned_item.go:33`) |
| `SetOwnedItemQuantity` | satisfied since Task 5: declares `saveSessionID`, `characterID`, `ownedItemID`, `quantity`, `expectedRevision`, in that exact order (`backend/endpoints/inventory/set_owned_item_quantity.go:33`) |
| `RemoveOwnedItem`, `MoveOwnedItemToInventory`, `MoveOwnedItemToStorage`, `SetWeaponUpgradeLevel`, `SetWeaponInfusion`, `SetSpiritAshUpgradeLevel` | declare `ownedItemID` without `saveSessionID` — must gain it |
| `SetWeaponAshOfWar` | declares `weaponOwnedItemID` without `saveSessionID` — must gain it |
| `SetInventoryOrder`, `SetStorageOrder` | declare `orderedOwnedItemIDs` without `saveSessionID` — must gain it |
| `SetEquippedTalismans` | declares `characterID`, `orderedOwnedItemIDs`, `expectedRevision` without `saveSessionID` — must gain it |

Two reasons, both already established rather than assumed:

1. **The identity is session-scoped by construction.** §4.1 makes an
   `OwnedItemID` valid only inside the session that minted it. An endpoint that
   accepts an `OwnedItemID` without a `saveSessionID` cannot state *which*
   session it belongs to, so it could only guess — resolve against "the"
   session, or against whichever session happens to hold that character. Both
   are fail-open. With `saveSessionID` present, an identity from another session
   is rejected as unknown (§4.1) instead of silently resolving.
2. **It is the pattern the implemented getters already use.** `GetInventory`
   and `GetStorage` take `saveSessionID` explicitly and match it exactly,
   without trimming, normalisation or fallback
   (`backend/saveengine/inventory.go:177-186`). The engine holds sessions in a
   map keyed by that identifier and has no concept of a current or default
   session. A contract without `saveSessionID` would require inventing one.

**Explicitly out of scope.** `SetEquippedArmaments` and `SetEquippedArmor` may
later need instance identity inside their `slotAssignments`, because an equipped
armament is one owned instance rather than an item type. They declare none of
the three variables today, so they are outside this contract and outside this
rule. Revisit them when their `slotAssignments` shape is specified; do not
pre-emptively widen the rule to reach them.

**Contradiction on record.** `tmp/app-se/endpoints-2.0.md` line 43 describes
`GetOwnedItem` as taking a *stable* `OwnedItemID`. That wording conflicts with
§4.1 of this document, where an ID is valid only inside one session and one
`saveRevision` and is invalidated by every commit. The 2.0 contract proposed
here is the session- and revision-scoped one, because the evidence in §2.6 does
not support any persistence claim. The `tmp/` files are research material and are
not edited by this document; the conflict is recorded, not resolved by rewriting
the source.

**Amendment of the Go contract files, one task at a time.** The original
documentation task that produced this document changed no Go contract file; that
boundary applied to it alone and is recorded here as history. Task 3 has since
amended `get_owned_item.go`, which now declares `saveSessionID`, `characterID`,
`ownedItemID` and is implemented, and Task 5 has amended
`set_owned_item_quantity.go`, which now declares `saveSessionID`, `characterID`,
`ownedItemID`, `quantity`, `expectedRevision` and is implemented. The ten
remaining contract-only files keep
their current `SupportedResourceVariables` until their own separate, explicitly
approved tasks amend them. Each such task also updates its contract header, the
endpoint-structure expectations, and — for any endpoint that is
transport-exposed by then — the route, `openapi.json`, the document and the
Scalar navigation, per the synchronisation rules in `AGENTS.md`. This document
only records the decision those changes must follow.

### 1.5 Explicit non-goals

- No general `EndpointError` model. `tmp/app-se/endpoints-2.0.md` line 21
  defers that deliberately; this document only names the *information* an error
  must carry.
- No GaItem parser design. That was separate work and a prerequisite for
  catalog resolution, not part of this contract.
- No further change to the contract of `GetInventory` and `GetStorage` beyond
  the one Task 2b already made with explicit approval (see §3 and §6.4).
- No persistence of identities across application restarts.
- No global registry of open save paths, no per-file lock and no new
  infrastructure of any kind (see the multi-session row of §7).

---

## 2. Evidence and its confidence

### 2.1 Confirmed facts (code in this repository, current branch)

| # | Fact | Source |
|---|---|---|
| C1 | SaveEngine's `GetInventory` and `GetStorage` return the raw physical fields — `gaItemHandle`, `quantity`, `acquisitionIndex`, `containerSection`, `physicalIndex` — together with the record's `OwnedItemID` and the result's `saveRevision`, and both explicitly document that `containerSection` + `physicalIndex` is **not** a stable item identity. The public endpoints of the same names additionally resolve each record to an `ItemDocument` (`kind`, `key`, `gameID`) through GameCatalog. | `backend/saveengine/inventory.go:86-112`, `backend/saveengine/storage.go:125-150`, `backend/endpoints/inventory/get_inventory.go:28-35` |
| C2 | Both getters return an `OwnedItemID` per record and a `saveRevision` per result, minted by the session registry. | `backend/saveengine/inventory.go:103-112`, `backend/saveengine/storage.go:143-150`, `docs/endpoints/inventory/get_inventory.md` |
| C3 | The public getter contract is locked by an assertion on the exact `SupportedResourceTypes` value (`ItemDocument`) and the exact ordered variable list. Changing either is a protected-test change. The raw phase-one contract tests were replaced under Task 2b by `TestGetInventoryContractResolvesItemDocuments` and `TestGetStorageContractResolvesItemDocuments`. | `backend/endpoints/inventory/get_inventory_test.go:453`, `get_storage_test.go:487`, `get_owned_item_test.go:238` |
| C4 | `saveengine.Session` holds `id`, `platform`, `format`, the private `revision` counter, the private `dirty` flag, the two-directional identity registry `ownedByLocator` / `ownedByID` and the mint counter `ownedSeq`. None of them is part of `SessionInfo` except through `SessionInfo.UnsavedChanges`, which reports `dirty`. There is still no per-character state. | `backend/saveengine/session.go` |
| C5 | The engine implements one content mutation, `SetOwnedItemQuantity`, plus `WriteSave`. The setter changes only the private snapshot and marks it dirty; `WriteSave` serializes, reload-validates and atomically persists that snapshot, then advances the revision and clears the dirty state. `UndoCharacterChanges` remains unimplemented. | `backend/saveengine/set_owned_item_quantity.go`, `backend/saveengine/write_save.go`, `backend/saveengine/owned_item_id.go` |
| C6 | Two native sentinels mark an absent record: handle `0x00000000` and `0xFFFFFFFF`. Both getters skip them; every other handle is reported as stored. | `backend/saveengine/inventory.go:54-58, 312` |
| C7 | The stored quantity carries a high bit that is not part of the count. Each container reader states and applies that mask once for its own section — `inventoryHeldQuantityMask` for InventoryHeld, `storageQuantityMask` for the Storage Box — and no other reader masks a quantity. The writer never uses a masked value: it reads the raw four bytes, keeps their high bit exactly as the game left it, and replaces only the 31-bit count (`newRaw = (oldRaw & 0x80000000) \| quantity`). | `backend/saveengine/inventory.go`, `backend/saveengine/storage.go`, `backend/saveengine/set_owned_item_quantity.go` |
| C8 | The 2.0 API spec forbids public setters that take raw bytes, offsets, handles, indices or event flags, and forbids accepting GaItem handles as a public identity. | `tmp/app-se/endpoints-2.0.md` lines 9 and 200 |
| C9 | Almost every 2.0 mutation contract declares `expectedRevision`; `GetRepairPlan` declares `saveRevision` as an input. The two names differ by direction of use, not by rule (§5.6). | `tmp/app-se/endpoints-2.0.md` lines 87, 97–178 |
| C10 | `GetInventory`, `GetStorage`, `GetOwnedItem` and `SetOwnedItemQuantity` all declare `saveSessionID`. The gap is now limited to the still contract-only files: every remaining inventory mutation contract and `SetEquippedTalismans` declare `characterID` but **not** `saveSessionID`. | `backend/endpoints/inventory/get_owned_item.go:33`, `get_inventory.go:33`, `set_owned_item_quantity.go:33` vs `backend/endpoints/equipment/set_equipped_talismans.go:25` and the remaining `backend/endpoints/inventory/set_*.go` |
| C11 | `containerSection` is an input *filter* on both implemented getters: the empty string means both physical sections, `"common"` and `"key"` select one. It is not part of any record identity. | `backend/saveengine/inventory.go:15-19, 191-196` |
| C12 | The engine serialises every session operation on one process-wide `Engine.mutex`. There is no per-session lock. | `backend/saveengine/engine.go:20-21, 97, 117, 147` |

### 2.2 Confirmed facts (legacy 1.5.8 / 1.6.8, behaviour evidence only)

| # | Fact | Source |
|---|---|---|
| L1 | Legacy already concluded that a handle alone does not identify a record, and said so explicitly: talisman handles (and other item-derived handles) are legitimately shared by several records in one container or split across inventory and storage. | `tmp/er-sf-1.6.8/backend/editor/session.go:26-31` |
| L2 | Legacy therefore minted a string UID: `hnd:0x%08X`, upgraded to `hnd:0x%08X:<container>:<slotIdx>` whenever the handle occurred more than once in the snapshot. | `tmp/er-sf-1.6.8/backend/editor/workspace.go:410-422` |
| L3 | A duplicate UID was a *known, non-blocking* validation error (`CodeDuplicateUID`), i.e. the legacy identity was not guaranteed unique and the save path proceeded anyway. | `tmp/er-sf-1.6.8/backend/editor/save.go:94-100` |
| L4 | The UID was session-scoped: it lived on an in-memory `InventoryEditSession` that was never persisted, and the baseline map keyed by UID was regenerated after every successful save. | `tmp/er-sf-1.6.8/backend/editor/session.go:20-50, 137-171` |
| L5 | A transfer between containers can mint a **fresh handle** for the moved instance when the destination already holds the same handle. The handle is therefore not stable across a mutation. | `tmp/er-sf-1.6.8/backend/core/transfer_rehandle_integration_test.go` |
| L6 | A transfer also **reassigns** the acquisition index of the moved record. That value is not stable across a mutation either. | `tmp/er-sf-1.6.8/backend/core/transfer_acquisition_index_test.go` |
| L7 | Legacy had a revision-shaped field, `InventoryEditSession.BaseRevision`, a truncated SHA-256 over slot version, GaItem count, magic offset and the first 1024 slot bytes. | `tmp/er-sf-1.6.8/backend/editor/session.go:122-136` |
| L8 | `BaseRevision` was **never compared anywhere** — in either tag, in Go or in the frontend. It was set at `StartSession`, refreshed after save, and read by nothing. The concurrency guard it was meant to provide did not exist. | grep over `tmp/er-sf-1.6.8` and `git show v1.5.8:backend/editor/session.go` |
| L9 | The GaItem record itself is small and its size depends on the item-ID family: 8 bytes for a plain item, 16 for armor, 21 for a weapon (weapon records additionally carry `AoWGaItemHandle`). | `tmp/er-sf-1.6.8/backend/core/structures.go:58-90` |

### 2.3 Differences between v1.5.8 and v1.6.8

| Area | v1.5.8 | v1.6.8 | Consequence for 2.0 |
|---|---|---|---|
| UID construction | identical (`hnd:0x%08X`, qualified on collision) | identical | The rule is stable across both tags → the *problem* it solves is real, but the rule itself is still not a 2.0 contract. |
| `ComputeBaseRevision` | present, identical body | present, identical body | Two releases shipped an unenforced revision. Do not repeat: a revision without an enforcement point is a fail-open. |
| `BaseRevision` enforcement | none | none | Same as above. |
| `EditableItem` shape | same fields plus formatting | adds `CanChangeAffinity`; whitespace realignment | Irrelevant to identity. |
| Transfer handle collision | present in `backend/core/transfer.go` | 1.6.8 adds the persistent integration test `transfer_rehandle_integration_test.go` proving the rehandle path end to end | 1.6.8 is the stronger evidence for L5, but only because it added a test — not because behaviour changed. |
| Acquisition index on transfer | reassignment logic present | 1.6.8 adds `transfer_acquisition_index_test.go` | Same: stronger documentation of unchanged behaviour. |

The versions do not conflict on any point relevant to this contract. Where 1.6.8
is preferred it is because it carries the test, not because it is newer.

### 2.4 Inferences (reasoned, not directly observed)

- **I1.** Because both the handle (L5) and the acquisition index (L6) can change
  during a mutation, any identity derived from them must be invalidated by that
  mutation rather than assumed to survive it.
- **I2.** Because the physical row of a record moves when a section is rewritten
  (C1 states this for the row identity), `physicalIndex` cannot survive a
  mutation either.
- **I3.** Because C8 forbids exposing handles and indices publicly, and L1–L2
  show that a *combination* of them is what actually disambiguates a record, the
  combination has to be hidden behind an opaque value rather than published.
- **I4.** Because L8 shows a revision field that nobody checked, the minimal
  useful revision contract must define the *rejection point*, not only the
  field.

### 2.5 Hypotheses and open questions

- **H1.** Whether a quantity-stacked goods row (`recordMode = quantity_stack`,
  see `tmp/app-se/stackable_items.md`) can legitimately appear as two separate
  rows of the same item in the same container. Legacy tolerated duplicates
  (L3) but that is tolerance, not proof. **Still open as a format question, and
  deliberately not answered by the quantity setter.** No available evidence shows
  two `quantity_stack` rows of one item in one container, and this document does
  **not** claim that the layout is natively impossible. `SetOwnedItemQuantity` is
  built so the answer does not matter: it validates the requested quantity plus
  the summed quantities of every *other* record of the same resolved game ID in
  the same physical container — both sections — against the container limit it
  was given. If such a duplicate exists the limit still holds; if it does not,
  the sum is simply the one record. The setter never merges, deduplicates, moves
  or reindexes rows to make the layout fit its expectations.
- **H2.** Whether the same instance can be present in Inventory and Storage at
  the same time, or whether the containers are strictly disjoint. L1 says a
  *handle* can appear in both; it does not say whether that is one instance or
  two. **Open.** The registry rule in §4.2 does not depend on the answer:
  Inventory and Storage are different physical containers and get different
  entries either way.
- **H3.** Whether `saveRevision` should be per-session or per-character.
  Section 5 proposes per-session and explains why; the contracts declare
  `expectedRevision` on both save-scoped and character-scoped mutations, which
  is consistent with per-session but does not prove it. **Open — decision
  requested.**

The `saveSessionID` asymmetry recorded in C10 was previously listed here as an
open question. It is **resolved** in §1.4: the twelve consumers take
`saveSessionID`. Only the amendment of the Go contract files is deferred, and it
is deferred to a named task, not to an open decision.

### 2.6 Absence of native evidence — stated separately

**There is currently no controlled native-save diff evidence in this repository
for the persistence of an item instance.**

- No `.sl2` or native `.dat` file is tracked by git (`git ls-files` returns
  none).
- Every SaveEngine test builds a **synthetic** fixture in a temp directory
  (`writeInventoryFixture`, `backend/saveengine/inventory_test.go:81`). A
  synthetic fixture proves that the reader handles a byte layout; it does not
  prove item-instance persistence after a game reload.
- The read-only PC and PS4 artifacts kept under `tmp/` establish that the
  current parser can locate the relevant slot data, but are not controlled
  before/after item-mutation evidence.
- The legacy findings L5 and L6 come from legacy Go tests over legacy
  in-memory fixtures, not from a diff of two native saves.

Therefore: nothing in §2.2 may be presented as confirmed in-game format
evidence. The design below is deliberately built so that it does **not** depend
on any unproven persistence property of a native record — that is the main
reason the recommendation in §4 does not encode physical data into the
identity.

---

## 3. What must not be weakened

`GetInventory` and `GetStorage` are implemented, documented and locked as
resolved ItemDocument getters: they expose raw physical fields together with
`ownedItemID`, `saveRevision`, `kind`, `key` and the exact resolved `gameID`.
The tests `TestGetInventoryContractResolvesItemDocuments` and
`TestGetStorageContractResolvesItemDocuments` assert the exact
`SupportedResourceTypes` value and the exact ordered variable list.

Task 2b (§6.4) changed that contract, which required explicit user approval
under the regression-test rules of `AGENTS.md` and shipped replacement coverage
at least as strong. No further change to it is in scope, and these invariants
hold:

- no field may be removed from `InventoryRecord` / `StorageRecord`;
- `gaItemHandle` and `acquisitionIndex` stay raw and unmasked;
- the `0x7FFFFFFF` quantity mask stays in exactly one place;
- an inactive or residual slot keeps returning `active: false`, an empty list
  and total `0`, without its slot data being searched.

---

## 4. `OwnedItemID` — variants

Each variant is judged on: stability, scope, behaviour for stackable goods
visible in both containers, exposure of physical data, maintenance cost,
behaviour after a mutation, and SSOT / fail-closed compliance.

### Variant A — `containerSection` + `physicalIndex`

The pair the implemented getters already return alongside the identity.

| Dimension | Assessment |
|---|---|
| Stability | **Poor.** The physical row moves whenever the game or the writer rewrites the section (C1, I2). |
| Scope | Character + container + physical layout. |
| Stackable goods in both containers | Works — the two rows live in different containers, so they never collide. |
| Physical exposure | **Fails.** `physicalIndex` is a raw index; C8 forbids raw indices in the public API. |
| Maintenance cost | Lowest — nothing to mint, nothing to store. |
| After mutation | Silently wrong. A stale pair still resolves, to the wrong record. This is the dangerous failure mode: it does not error, it edits the wrong item. |
| SSOT / fail-closed | **Fails fail-closed.** A stale ID cannot be distinguished from a fresh one. |

**Rejected.** The silent-misresolution failure mode alone disqualifies it.

### Variant B — `GaItemHandle` alone

| Dimension | Assessment |
|---|---|
| Stability | **Poor.** A transfer can mint a fresh handle for the same instance (L5). |
| Scope | Character-wide, in principle. |
| Stackable goods in both containers | **Fails.** L1 states directly that one handle is legitimately shared by several records, in one container or split across containers. Legacy needed a container+slot qualifier precisely for this (L2), and still tolerated duplicates (L3). |
| Physical exposure | **Fails.** C8 names GaItem handles explicitly as a value the public API must not accept. |
| Maintenance cost | Low. |
| After mutation | May resolve to a different instance, or to none. |
| SSOT / fail-closed | **Fails** — ambiguity is resolved arbitrarily rather than rejected. |

**Rejected**, and it is the variant with the most direct evidence against it.

### Variant C — explicit or encoded container + handle combination

E.g. a public struct `{container, gaItemHandle}` or the legacy-style string
`hnd:0x…:<container>:<index>`.

| Dimension | Assessment |
|---|---|
| Stability | Better than A or B, still tied to two values that a mutation can change (L5, I2). |
| Scope | Character + container. |
| Stackable goods in both containers | Handled, exactly as legacy handled it — that is what the qualifier is for (L2). |
| Physical exposure | **Fails.** Explicit form publishes the handle; the encoded form publishes it in hex, which is the same disclosure with extra steps. Any consumer will decode it, and then the format becomes a de-facto contract. |
| Maintenance cost | Medium — the encoding becomes a parsing surface on both sides. |
| After mutation | Same silent-misresolution risk as A once handles are reallocated. |
| SSOT / fail-closed | Partially. Ambiguity is at least detectable, but nothing forces detection. |

**Rejected** on C8, and on the observation that an encoded format is a published
format whether or not the document says so.

### Variant D — opaque identifier registered in the session

SaveEngine mints an opaque string per record while producing a getter result,
records the mapping `OwnedItemID → physical location` in the session, and is the
only component that can resolve it. The mapping is discarded whenever the
underlying data changes.

| Dimension | Assessment |
|---|---|
| Stability | Stable for exactly as long as it is meaningful: within one session, one character and one `saveRevision`. It never claims stability it cannot deliver. |
| Scope | Session + character + revision. Explicitly not persisted, not shareable between sessions, not stored in templates or presets. |
| Stackable goods in both containers | Handled by construction — two rows are two registry entries, whatever their handle or quantity. Also unaffected by H1/H2 remaining open, because the registry does not need to know whether two rows are "the same item". |
| Physical exposure | **None.** No offset, no index, no handle leaves the package; the registry holds them privately, exactly as the snapshot is already held privately (C5). |
| Maintenance cost | One map on the session plus one mint/resolve pair. No new package, no new public type beyond `string`. |
| After mutation | The revision changes, the whole map is cleared, and every previously issued ID is rejected with a typed "stale identity" error rather than silently resolving. |
| SSOT / fail-closed | **Complies.** One owner, one registry, unknown or stale input rejected before any mutation. |

**Recommended — see §4.1.**

### Variant E — a stable native instance key read from the save

Rejected without further analysis: it would require a persistence property of a
native record that §2.6 says nobody has demonstrated. Proposing it would mean
turning a hypothesis into production behaviour, which `AGENTS.md` forbids.

### 4.1 Recommendation

> **Recommended: Variant D — an opaque, SaveEngine-minted identifier registered
> in the session. Requires user approval.**

**Format: deliberately deferred.** The contract is *"an opaque, non-empty
string, compared byte for byte, never parsed by any consumer"*. The proposal
deliberately does **not** fix the internal shape. Two reasons:

1. Publishing a shape invites consumers to depend on it, and then it stops being
   opaque — that is exactly how Variant C degrades.
2. The GaItem parser does not exist yet. Fixing an encoding now would fix it
   against a record model that has not been read from a real save.

A random or counter-derived token is sufficient. This is a `ponytail:` style
deliberate simplification: the ceiling is that IDs cannot survive a restart, and
the upgrade path is a native instance key *if and only if* §2.6 is ever
satisfied with real evidence.

**Validity scope.** An `OwnedItemID` is valid only for the exact tuple:

| Dimension | Rule |
|---|---|
| Session | The `saveSessionID` it was minted under. `CloseSave`/`CloseSession` destroys it. |
| Character | The `characterID` it was minted for. Presenting it with another `characterID` is an error, never a lookup in the other slot. |
| Container | The physical container it was read from — Inventory or Storage — is recorded in the registry entry. A move updates the registry under a new revision; the caller receives fresh IDs from the next getter call. |
| Revision | The `saveRevision` at mint time. Any increment invalidates it. |

**Expiry.** An ID expires when any of these happens: `saveRevision` increments
(any committed mutation, including undo, and every `WriteSave` — §5.3); the
character slot is deleted, cloned over or deactivated; the session is closed;
the application restarts. It never expires on a read.

**Behaviour for the difficult cases:**

| Case | Required behaviour |
|---|---|
| Missing / empty ID | Reject with "ownedItemID is required". No default, no "first record". |
| Unknown ID (never minted, or minted by another session) | Reject as unknown. Never fall back to a physical lookup. |
| Stale ID (minted under an earlier revision) | Reject as stale, distinguishable in the error from "unknown", because the caller's remedy differs: re-read the container. |
| Wrong character | Reject. Never resolve into the correct character's slot. |
| Duplicate — the same physical record reachable by two IDs | Must be impossible by construction: one registry entry per physical record per revision (§4.2). If the mint step ever detects it, that is a hard internal error, not a tolerated warning. This is the explicit point where 2.0 departs from legacy `CodeDuplicateUID` (L3). |
| Ambiguous — the ID resolves but the record no longer matches what was registered | Reject and do not mutate. Fail-closed. |
| Unknown / malformed record data | Two layers, deliberately different. SaveEngine mints an identity for **every** non-sentinel physical record (C6) and never drops one, so nothing becomes unaddressable. The public endpoints `GetInventory`, `GetStorage` and `GetOwnedItem` resolve each record against GameCatalog and **reject the whole request** when a handle or a save-side game ID does not yield a valid `ItemDocument` (`backend/endpoints/inventory/get_inventory.go:115`). What must never happen at either layer is an unknown record being silently dropped, repaired, or turned into a different valid item. |

**What the ID never carries:** no offset, no `physicalIndex`, no `GaItemHandle`,
no `acquisitionIndex`, no slot address. Those stay inside `saveengine`, as they
do today.

### 4.2 The registry — one map, and idempotent minting

This section states the registry rules that the rest of the document depends on.

**One registry, keyed by `(saveSessionID, characterID, saveRevision)`.** There is
exactly one identity registry per that triple, and it covers Inventory and
Storage together. There is no separate Inventory registry and no separate
Storage registry, because an `OwnedItemID` handed to
`MoveOwnedItemToInventory` or `MoveOwnedItemToStorage` must be resolvable
without the caller first declaring which container it came from.

**Inventory and Storage are different physical containers.** Each registry entry
records which of the two its record lives in. Two rows of the same item, one in
Inventory and one in Storage, are two entries and two IDs — this is true
regardless of how H2 is eventually answered.

**`containerSection` is a view filter, not identity.** `common` and `key` are
the two physical sections of InventoryHeld and are an input filter on the getter
(C11). A record's section is a property of the record, never a component of its
identity, and passing a different `containerSection` never changes which ID a
record gets.

**Lazy materialisation, per container.** The registry is not built at
`LoadSave`. It materialises on first read: the first Inventory read of a given
revision mints entries for that character's Inventory records; the first Storage
read of the same revision mints entries for that character's Storage records.
Either may happen without the other, and neither triggers the other. Entries
already present for the current revision are reused, never re-minted.

**Paging and filtering never affect the minted ID.** `page`, `pageSize` and
`containerSection` select *which* records are returned; they never take part in
minting. Reading page 2, then page 1, then both sections unfiltered, all within
one revision, must yield the same `OwnedItemID` for the same physical record.
The first read of a container mints for the whole container, not for the
returned page — otherwise the ID would depend on how the caller paged.

**A revision increment clears the whole map.** On increment, every entry for
that session is dropped. The registry is **not** rebuilt eagerly: no automatic
rebuild under the global lock, no accumulation of old and new entries side by
side. It re-materialises lazily on the next read of each container, per the rule
above. This keeps a commit cheap and keeps exactly one generation of identities
alive at any moment.

**Inactive and residual slots mint nothing.** An inactive slot — including a
residual one, whose deleted character's data is still in the file — mints no
IDs and its slot data is never searched, exactly as the implemented getters
already behave. The read is still a valid, non-error result and still returns the
current `saveRevision`; an empty container is not an excuse to omit the revision.

---

## 5. `saveRevision` contract

### 5.1 Owner and type

- **Owner:** `saveengine.Session` — the struct that already exists (C4). One
  new unexported field, no new type and no new package.
- **Internal type:** `uint64`, monotonically increasing, never reused, never
  decreasing. This stays inside `saveengine`.
- **Public representation:** `saveRevision` and `expectedRevision` are
  **non-empty decimal strings** — the base-10 rendering of the internal
  `uint64`, with no sign, no prefix, no padding, no separator. They are compared
  **exactly**, byte for byte: never trimmed, never parsed, never normalised.
  They are never a JSON number, never a timestamp and never a hash.

The string is not a stylistic choice. A JSON number is an IEEE-754 double in
JavaScript and TypeScript, which represents integers exactly only up to
2^53 − 1. A full `uint64` above that bound is silently rounded on parse, so two
different revisions can compare equal in the frontend — a fail-open that this
contract exists to prevent (I4, L8). Rendering the counter as a decimal string
keeps the comparison exact in every consumer without forcing `BigInt` handling
into the frontend, and it makes the "compare, never compute" rule structural
rather than a convention: nothing downstream can increment, subtract or order
the value by accident.

Not a content hash. L7/L8 show a content hash that nobody compared; a counter is
smaller, is trivially comparable, and cannot be accidentally equal after a real
change.

**Open question H3:** this proposal is per-session, not per-character. Rationale:
`CloseSave`, `WriteSave` and `SetSaveAccountID` all declare `expectedRevision`
and are save-scoped, so a per-character counter would need a second, save-scoped
counter beside it. One counter is the smaller contract. The cost is that a
mutation on character 0 invalidates the IDs held for character 3. Given that the
UI edits one character at a time, that cost looks acceptable — **but this is a
decision for the user, not an established fact.**

### 5.2 Initial value

After a successful `LoadSave`, `saveRevision` is **`"0"`** — the internal
counter is 0, rendered as the one-character decimal string. A freshly loaded,
unmodified session is revision 0 by definition, so `"0"` also means "no mutation
has been applied in this session".

### 5.3 Increment condition

`saveRevision` increments by exactly 1 **when, and only when, a mutation
commits successfully**, and `WriteSave` always counts as such a commit:

| Event | Increments? | Reason |
|---|---|---|
| Any getter | No | Getters are non-mutating (`docs/endpoints/README.md`, "Getters and mutations"). |
| Mutation that fails validation before the first byte changes | No | Nothing changed. |
| Mutation that fails mid-way and rolls back | No | The rollback restores the pre-mutation state, so the revision must also be the pre-mutation one. Incrementing here would invalidate identities for a change that did not happen. |
| Mutation that commits | **Yes, +1** | |
| `UndoCharacterChanges` that commits | **Yes, +1** | An undo is a mutation. It changes the snapshot, so every ID minted against the undone state must expire. The revision never goes backwards — undo moves forward to a new revision that happens to hold older content. |
| `WriteSave` | **Yes, +1, always** | See below. |
| `CloseSave` | N/A | The session and its whole registry cease to exist. |

**`WriteSave` always increments `saveRevision`.** There is no conditional
variant, no "only when the snapshot is replaced" clause and no per-implementation
answer.

This is a deliberately conservative, fail-closed invalidation. `WriteSave`
serialises, reloads and validates the result before writing it
(`tmp/app-se/endpoints-2.0.md` line 98), and a reload re-derives handles and
acquisition indices (L5, L6, I1, I2). Deciding case by case whether the reloaded
result becomes the active snapshot would make identity validity depend on an
implementation detail of the writer — exactly the kind of implicit contract that
produced the unenforced legacy revision (L8). Unconditional invalidation costs
one re-read after each save and removes the whole question.

The cost is bounded, because the consumer never has to re-read to learn the new
revision: `WriteSave` returns it in its typed result (§5.6). The re-read after a
save is needed only to obtain fresh `OwnedItemID`s, and only for containers the
consumer still intends to address — which is the same lazy rule as §4.2.

The increment happens **inside the same critical section as the commit**, under
the existing process-wide `Engine.mutex` (C12) — there is no per-session lock
today and this contract does not introduce one — so no caller can observe a
committed change with a stale revision or vice versa. This is the enforcement
point whose absence made legacy `BaseRevision` useless (L8, I4), and it is the
reason a `-race` test is mandatory for this work.

### 5.4 Relation to atomic mutation and rollback

The proposed order inside one mutation:

1. take the existing `Engine.mutex` (C12);
2. resolve `expectedRevision` against the current `saveRevision`; on mismatch,
   return the stale-revision error and **stop before reading the plan**;
3. resolve every `ownedItemID` through the registry; on unknown/stale/ambiguous,
   stop;
4. build and validate the complete plan (`tmp/app-se/architecture.md` §5);
5. apply atomically;
6. on failure, roll back and leave `saveRevision` and the registry untouched;
7. on success, increment `saveRevision`, clear the identity registry for the
   session (§4.2 — it re-materialises lazily on the next read), and return the
   new revision in the typed result;
8. release the mutex.

Steps 2 and 3 are both fail-closed and both happen before any mutation.

### 5.5 Error semantics for a stale revision

No general `EndpointError` is designed here. The required information is:

- the operation was **rejected**, not partially applied — the caller must be
  able to tell that the save is untouched;
- the reason is **a stale revision**, distinct from "unknown session",
  "unknown ownedItemID" and "stale ownedItemID", because the remedies differ;
- the **current** `saveRevision`, so the caller can re-read and retry without a
  second round trip;
- a user-facing message naming the rejected field and the reason
  (`AGENTS.md`, implementation rules), with the session identifier kept to the
  internal log rather than the message.

### 5.6 What getters return and setters accept

- **Getters** return `saveRevision` — the value at the moment the snapshot was
  read, under the same lock that produced the result. Any getter that also
  returns `OwnedItemID`s **must** return the revision those IDs were minted
  under; otherwise the caller cannot construct a valid mutation.
- **Setters** accept `expectedRevision`, matching the name already declared in
  every contract file (C9). An empty, absent or non-decimal `expectedRevision`
  is **rejected**, not treated as "any revision" — a fail-open default here
  would reintroduce exactly the legacy failure of L8.
- Setters return the **new** `saveRevision` in their typed result, so a caller
  performing a sequence of mutations does not have to re-read between them.

**The parameter name carries no semantics.** `GetRepairPlan` declares its input
as `saveRevision` rather than `expectedRevision` (C9); that is not an exception
and does not need one. The rule is semantic, not lexical: **every endpoint that
accepts a revision validates it identically and fail-closed** — exact
byte-for-byte comparison against the session's current value, no trimming, no
parsing, no normalisation, and rejection on empty, absent, malformed or
mismatched input. `expectedRevision` is simply the name used where the value
expresses a precondition for a mutation, and `saveRevision` where it binds a
non-mutating result to a revision. Both are validated by the same rule.

---

## 6. Implementation plan — separate, sequential tasks

Each step is a separate approved task with its own commit. No step may be merged
into another.

### 6.1 Task 1 — identity and revision owner in SaveEngine

Add `saveRevision` and the identity registry to `saveengine`, with mint and
resolve unexported. No endpoint changes, no public getter change. Deliverable:
the owner exists and is unit-tested, and nothing outside the package can see it.

Because the engine was read-only when this task ran, the revision started and
stayed at 0 within it; the increment path was written but had no caller. The
tests nevertheless had to prove the increment, the rollback non-increment and the
invalidation, using an internal test hook rather than a public mutation. The
first real caller arrived later, with `SetOwnedItemQuantity` (§6.7).

### 6.2 Task 2a — `GetInventory` and `GetStorage` return the identity

Completed. The two getters mint and return `OwnedItemID` per record and
`saveRevision` per result. **Nothing else changed in this task:** no catalog
resolution, no `kind`, no `key`, no name, no family, no variant — those were
left to Task 2b.

**What this task did to the contract definitions: nothing.** It changed neither
`SupportedResourceTypes` nor the input-variable lists of either endpoint, so the
raw-phase contract tests that existed at the time,
`TestGetInventoryContractIsRawPhaseOne` and
`TestGetStorageContractIsRawPhaseOne`, stayed untouched and kept passing.

That statement applied to Task 2a only. Task 2b subsequently replaced those
raw-phase contract tests with the ItemDocument contract tests recorded in §6.4,
so neither name exists in the tree any more (C3).

**What this task required approval for: the result shape.** Adding two fields to
the public result widened what the getters return, so every test that asserted a
*complete* result or a *complete* record list with `reflect.DeepEqual` failed
until its expected value gained the new fields. Naming that change plainly: it
was **an extension of the public result plus correspondingly stronger
assertions** — each listed assertion kept comparing the full value exactly, over
a larger value. It was not a weakening, not a relaxation, not a switch to
partial matching, and no case, boundary or fixture was removed.

The assertions approved before Task 2a started were these **15 full-result
assertions**. The line numbers are the historical record of the pre-Task-2a
tree; the files have moved on since Task 2a and Task 2b:

| File | Line | Assertion |
|---|---|---|
| `backend/endpoints/inventory/get_inventory_test.go` | 202 | full `GetInventoryResult` |
| `backend/endpoints/inventory/get_inventory_test.go` | 244 | full `result.Records` |
| `backend/endpoints/inventory/get_inventory_test.go` | 266 | residual slot vs a full `GetInventoryResult` |
| `backend/endpoints/inventory/get_storage_test.go` | 223 | full `GetStorageResult` |
| `backend/endpoints/inventory/get_storage_test.go` | 243 | full `GetStorageResult` |
| `backend/endpoints/inventory/get_storage_test.go` | 284 | full `result.Records` |
| `backend/endpoints/inventory/get_storage_test.go` | 306 | residual slot vs a full `GetStorageResult` |
| `backend/saveengine/inventory_test.go` | 219 | full `CharacterInventory` |
| `backend/saveengine/inventory_test.go` | 250 | full `result.Records` |
| `backend/saveengine/inventory_test.go` | 291 | full `result.Records` |
| `backend/saveengine/inventory_test.go` | 323 | residual slot vs a full `CharacterInventory` |
| `backend/saveengine/storage_test.go` | 246 | full `CharacterStorage` |
| `backend/saveengine/storage_test.go` | 278 | full `result.Records` |
| `backend/saveengine/storage_test.go` | 319 | full `result.Records` |
| `backend/saveengine/storage_test.go` | 351 | residual slot vs a full `CharacterStorage` |

The four residual-slot assertions in that table — the ones at
`get_inventory_test.go:266`, `get_storage_test.go:306`,
`inventory_test.go:323` and `storage_test.go:351` — are on the list because §4.2
requires an inactive or residual slot to return the current `saveRevision` while
minting no IDs, so its result is no longer the zero value. They became an
explicit expected value carrying the revision, not a looser check.

**Four further assertions were *not* on the list and were not touched:**

| File | Line | Assertion |
|---|---|---|
| `backend/endpoints/inventory/get_inventory_test.go` | 319 | rejected request vs `GetInventoryResult{}` |
| `backend/endpoints/inventory/get_storage_test.go` | 361 | rejected request vs `GetStorageResult{}` |
| `backend/saveengine/inventory_test.go` | 390 | rejected request vs `CharacterInventory{}` |
| `backend/saveengine/storage_test.go` | 435 | rejected request vs `CharacterStorage{}` |

These four sit inside the `RejectsInvalidRequests` tests, not inside the
residual-slot tests. They assert the **fail-closed error path**: a rejected
request returns the zero value and nothing else. A rejected request has no
result to identify and no revision to report, so the zero value stays the zero
value — widening the result shape did not change them, and they kept passing
untouched. An earlier revision of this document mistook them for the
inactive-slot assertions and counted 19; the approved count is 15.

None of these assertions was weakened, removed, skipped or altered by the
documentation task that recorded this plan.

Task 2a also updated `docs/endpoints/inventory/get_inventory.md`,
`docs/endpoints/inventory/get_storage.md`, `tools/swagger/openapi.json` and
`tools/swagger/main_test.go` for the widened result, per the synchronisation
rules in `AGENTS.md`. The route, the endpoint index and the Scalar navigation did
not change, because no endpoint was added or removed.

### 6.3 Gate — GaItem parser evidence

**Closed for the read-only Task 2b getter.** The accepted basis is the matching
1.5.8/1.6.8 GaItem reader, the static read-only PC and PS4 artifacts in `tmp/`,
and synthetic coverage of both platform entry points and both legacy record
counts. This is sufficient to resolve an existing loaded snapshot fail-closed;
it does not establish a mutation, serialization or game-reload guarantee.

No new PS4 test artifact is required for this task. If setters, WriteSave or a
platform-specific discrepancy later need stronger evidence, this gate must be
reopened for that narrower operation.

### 6.4 Task 2b — catalog resolution in the two getters

Completed after Task 2a and the §6.3 gate. This task resolved records against
GameCatalog and ended the raw phase-one contract. It changed
`SupportedResourceTypes` and the documented contract; the approved replacement
tests are `TestGetInventoryContractResolvesItemDocuments` and
`TestGetStorageContractResolvesItemDocuments`, which preserve the exact ordered
variable-list assertions while requiring `ItemDocument`.

It carried the full synchronisation set in the same task:
`docs/endpoints/inventory/*.md`, `docs/endpoints/README.md`,
`tools/swagger/main.go`, `tools/swagger/openapi.json`,
`tools/swagger/docs-portal/scalar.config.json` and `tools/swagger/main_test.go`.

### 6.5 Task 3 — `GetOwnedItem` and the setters as consumers

**`GetOwnedItem` is implemented.** It was completed after Task 2b, with the full
contract below and none of the three exclusions it rules out. Its
`SupportedResourceVariables` are `saveSessionID`, `characterID`, `ownedItemID` in
that order, and it is transport-exposed as
`GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/owned-items/{ownedItemID}`
of the local explorer. `GetInventory` and `GetStorage` kept their behaviour: the
complete physical read of each container was extracted into one private reader
per container, which both the paging getter and `GetOwnedItem` now share, so the
anchors, bounds, section sizes, sentinels, quantity mask, physical indexes and
minting still have exactly one owner each. The documented contract lives in
[`docs/endpoints/inventory/get_owned_item.md`](endpoints/inventory/get_owned_item.md).

**The setter endpoints remain unimplemented, except the first one.** They are
separate later tasks. The `saveSessionID` amendment of the remaining contract
files (§1.4) travels with them, one endpoint at a time. The SaveEngine side of
the first of them exists since §6.7, and the endpoint itself since §6.8.

**Why it came last of the getters, not first (historical ordering).** The
sequence it had to follow was Task 1, Task 2a, the §6.3 research gate and Task
2b, in that order; it was implemented only once all four were done.

The reason was its own contract, not a scheduling preference.
`backend/endpoints/inventory/get_owned_item.go:32` declares
`SupportedResourceTypes: "ItemDocument"`, so its result must resolve one owned
instance to `kind`, `key` and its variant. Task 2a deliberately produced none of
those: it returned raw records carrying an `OwnedItemID` and a `saveRevision`,
with no GaItem parser, no GameCatalog read and no catalog identity (§6.2).
Task 2b is where records became semantically resolved, and it in turn waited on
the §6.3 gate. `GetOwnedItem` therefore needed both, and an ordering that put it
after Task 2a alone would have asked it to answer a question its inputs could
not answer.

Three things that ordering explicitly ruled out, and the implementation honours:

- **No temporary "raw `GetOwnedItem`".** An interim handler returning a record
  without `kind`/`key` is not this endpoint; it is a different one wearing its
  name, and it would ship a contract the file does not declare.
- **No split of the public contract into a raw and a semantic version.** One
  endpoint, one `SupportedResourceTypes`, one documented result. Two variants
  would be two sources of truth for one identity, which is exactly the pattern
  `AGENTS.md` forbids, and the second one would then have to be retired.
- **No partial `ItemDocument`.** Shipping the field with unresolved or
  placeholder values would publish a resolution guarantee the backend cannot
  keep, and unknown data must fail safely rather than be dressed up as valid.

`GetOwnedItem` is the first consumer of the finished, semantically resolved
identity — it is where the producers of Task 2a and the resolution of Task 2b
meet in a single-instance result. It went before the mutations, because it is a
getter and could be verified without any mutation path existing. The mutations
follow, each in its own task, each depending additionally on the
mutation/rollback machinery that this document does not design. The
`saveSessionID` amendment of the ten remaining contract files (§1.4) is part
of that stage, one endpoint at a time.

### 6.6 Required test coverage

Every task above must cover, at the layer where the behaviour is observable:

| Dimension | Required cases |
|---|---|
| Platform | PC and PS4, both producing IDs and both rejecting a foreign one. |
| Slot state | Active, inactive, and residual (a deleted character whose data is still in the file) — a residual slot mints no IDs and its data is not searched, but the result still carries the current revision. |
| Sentinels | Handles `0x00000000` and `0xFFFFFFFF` get no ID and stay invisible (C6). |
| Unknown / malformed data | An unresolvable handle, malformed GaItem table or unknown catalog game ID rejects the whole getter result; it is never dropped, repaired, reinterpreted or replaced. |
| Stackable goods | The same `kind`/`key` present in Inventory and in Storage yields two distinct IDs; two rows of the same stackable item in one container yield two distinct IDs. |
| Idempotent minting | Re-reading the same revision returns identical IDs; page 1 then page 2, both sections then one section, and Inventory read before Storage or after it, all yield the same ID for the same physical record (§4.2). |
| Lazy materialisation | Reading only Inventory does not mint Storage entries, and the later first Storage read of the same revision succeeds and mints them. |
| Stale IDs after a mutation | An ID minted at revision *n* is rejected at revision *n+1*, with an error distinguishable from "unknown". |
| Revision | Initial `"0"`; +1 on commit; unchanged on validation failure; unchanged on rollback; +1 on undo; +1 on every `WriteSave`, asserted before and after the write. Every assertion compares the decimal string exactly. |
| Concurrency | `-race` over concurrent getter/mutation access to one session — mandatory, because the registry and the counter are shared mutable state behind `Engine.mutex`. |
| Cross-session | An ID from session A is rejected by session B, and by a reopened session on the same file. |

### 6.7 Task 4 — the first SaveEngine mutation

Implemented: `saveengine.Engine.SetOwnedItemQuantity` sets the quantity of one
existing record addressed by its `OwnedItemID`. It is an internal SaveEngine
boundary for the future endpoint, and it changed no endpoint, no route, no
`openapi.json` entry, no endpoint document and no Scalar navigation.

What it establishes, and what it deliberately does not:

- **In memory only.** This mutation lands in the session's private snapshot and
  opens no file. `SessionInfo.UnsavedChanges` becomes `true`; the separately
  implemented `WriteSave` is now the operation that persists and clears it.
- **One critical section.** `commitRevision` takes the process-wide
  `Engine.mutex` exactly once and hands the locked session to the mutation, which
  therefore uses only the helpers that already require the lock
  (`resolveOwnedItemID`, `readInventoryRecords`, `readStorageRecords`,
  `readGaItemMap`, `resolveGaItemHandle`) and calls no public engine method.
- **Every fallible check precedes the write.** `expectedRevision`,
  `ownedItemID`, the record match, `expectedGameID`, the per-record limit and
  the container total are all settled first; the write is then four bytes wide,
  is read back and verified, and a failed verification restores the exact
  previous four bytes. Nothing is copied and no snapshot-sized buffer exists.
- **The limits come from above.** `maxPerRecord` and `maxContainerTotal` are
  supplied by the caller and enforced verbatim. SaveEngine never invents,
  defaults, widens or clamps them, and it holds no Safe Mode or Chaos Mode rule.
  Deciding which GameCatalog value each one is — `maxPerStack`, `maxInventory`,
  `maxStorage` or a minimum of them — belongs to the endpoint task and is
  deliberately not decided here.
- **`expectedGameID` is an anti-TOCTOU guard, not a selector.** The endpoint will
  resolve the item outside the lock in order to read those limits; the engine
  re-resolves the addressed record's handle under the lock and refuses unless it
  is still the same game ID.
- **It never removes anything.** `quantity` of `0` is an error; removal belongs
  to a later `RemoveOwnedItem`. No record is created, merged, moved or reordered,
  and no handle or acquisition index changes.
- **The identity contract is unchanged.** A commit advances the revision by
  exactly 1 and clears the registry with no eager rebuild (§4.2), so the
  `ownedItemID` the mutation was performed with is stale as soon as it returns.
  The result echoes it as the identifier that was used, not as one that can be
  used again.

### 6.8 Task 5 — the first public mutation endpoint

Implemented: `inventory.SetOwnedItemQuantity`. It is the endpoint task §6.7 left
open, and it is the first public mutation of SaveForge 2.0. The local explorer
exposes it through its loopback-only HTTP route, OpenAPI operation and Scalar
entry; there is still no Wails binding, CLI command or frontend. The
documented contract lives in
[`docs/endpoints/inventory/set_owned_item_quantity.md`](endpoints/inventory/set_owned_item_quantity.md).

**The limit rule §6.7 deferred, now approved and implemented:**

| Limit | Source |
|---|---|
| `maxContainerTotal` | `item.storage.maxInventory` for a record in Inventory; `item.storage.maxStorage` for a record in Storage |
| `maxPerRecord` | `min(item.capabilities.stack.rules.maxPerStack, maxContainerTotal)` |

The rule is deliberately fail-closed: in Storage a single record still never
exceeds `maxPerStack`, even when `maxStorage` is larger. The endpoint accepts no
mode, so `safeModeMaxInventory`, `safeModeMaxStorage` and the `-sfv` fields are
never read; deciding a mode-dependent limit stays a later, separate decision.
Unknown catalog data — an unknown or disabled stack capability, missing stack
rules, a `recordMode` that is unknown or `separate_instances`, or an unknown or
zero limit of the record's own container — rejects the whole request. No limit is
defaulted, invented, widened or clamped, and `quantity` is never clamped to fit.

This differs from SaveForge 1.5.8 and 1.6.8 on purpose. Both older versions
capped a quantity by the container limit alone (`MaxInventory` / `MaxStorage`,
or their `GameMax*` variants) and **clamped** the requested value to it
(`resolveQty` in 1.5.8, the clamp repair primitive in 1.6.8). 2.0 rejects
instead of clamping and additionally bounds one physical record by `maxPerStack`.
The older behaviour was not adopted; it is recorded here as the compared
alternative.

The endpoint calls no other endpoint: it uses `engine.GetOwnedItem`,
`engine.ResolveGaItemIDs`, `gameCatalog.ItemByGameID` and
`engine.SetOwnedItemQuantity` directly, and passes `saveSessionID`,
`ownedItemID`, `expectedRevision` and `quantity` through byte for byte. The
identity contract of §6.7 is unchanged: a successful call increments
`saveRevision`, invalidates every outstanding `ownedItemID` of the session, sets
`UnsavedChanges`, and writes no file. The `ownedItemID` in its result is already
stale and identifies the performed operation only.

---

## 7. Impact matrix

| Dimension | Impact | Note |
|---|---|---|
| **PC** | Yes | Must mint and resolve identically. |
| **PS4** | Yes | Must mint and resolve identically. The identity is derived from the record model, which is already shared; only the container differs (`inventory_pc.go` / `inventory_ps4.go`). No PC finding may be generalised to PS4 without a PS4 test. |
| **Slot versions** | Yes | The revision is version-independent, but the registry is materialised from a version-specific read. Every supported slot version must be covered by Task 2a. |
| **Inventory** | Yes | Primary producer; a distinct physical container with its own registry entries. |
| **Storage** | Yes | Primary producer, same contract, a distinct physical container. The cross-container case (H2) is exactly why the identity must not be handle-derived. |
| **`containerSection`** | No | A view filter only (C11). It never influences a minted ID and is not part of any identity. |
| **Safe Mode** | Yes | The identity and the revision are mode-independent by design: minting is a read, and the revision counts commits regardless of mode. A mode-dependent identity would be a second source of truth. |
| **Chaos Mode** | Yes | Same as Safe Mode. Chaos Mode may allow more mutations, so it will produce more revision increments — but the rule is unchanged. |
| **Backend** | Yes | `saveengine` owns everything; `backend/endpoints/inventory` and `backend/endpoints/equipment` pass values through. |
| **Frontend** | Yes | Must treat `ownedItemID` as an opaque string, must carry `saveRevision` back as `expectedRevision` **as a string** — never through `number`, `parseInt` or JSON number parsing (§5.1) — and must handle the stale-revision error by re-reading rather than retrying. No frontend work is in scope until Task 2a. |
| **Preview** | Yes | A preview is a read: it may return IDs and a revision, and must not increment. |
| **Apply** | Yes | Validates `expectedRevision` and every `ownedItemID` before the first byte changes; increments exactly once on commit. |
| **Save (`WriteSave`)** | Yes | Always increments (§5.3). Every ID issued before the write is stale afterwards; the new revision comes back in the write's own result. |
| **Reload** | Yes | A reload is a new session: every previously issued ID is invalid, revision restarts at 0. |
| **Scan** | Yes | Non-mutating, so no increment. A scan that wants to reference a record must use an `OwnedItemID`, not a physical index, so its findings survive being displayed. |
| **Multiple sessions on one file** | **Limitation, stated not solved** | The revision is per session, so it does not protect two independent snapshots of the same file from each other: session A and session B each load the file at revision 0, and A committing does not make B's revision stale. Nothing here claims otherwise. The identity contract is still sound in each session, because an ID from A is rejected by B (§4.1). Guarding the *write target* against a competing writer is a separate concern and belongs to the `WriteSave` contract — this proposal deliberately adds no global registry of open paths, no per-file lock and no new infrastructure to reach for it. |
| **Normal data** | Yes | Baseline case. |
| **Empty data** | Yes | An empty container yields an empty ID list and a valid revision — not an error. |
| **Boundary data** | Yes | First and last physical row of each section; a full container; the last usable revision value is `uint64`, i.e. not reachable in practice. |
| **Malformed data** | Yes | Gets an ID, stays listed, is never repaired implicitly. |
| **Unknown data** | Yes | Same as malformed: visible, identified, never reinterpreted. |
| **Placeholder data** | Yes | Native absent sentinels (C6) are not records and get no ID. |
| **Existing save data** | Yes | Identities are minted for records that already exist in the user's file. |
| **New application data** | Yes | A record created by `AddItemToInventory` gets an ID from the next read after the commit, under the new revision. |
| **GameCatalog data / generated files** | **Not relevant** | The identity is an instance identity produced by SaveEngine. It carries no catalog data, is not stored in the catalog, and no generator output changes. `kind`/`key` remain the catalog identity and are unaffected. |
| **Diagnostics / repair subsystem** | **Not relevant to this proposal** | It will consume the identity later, but nothing in `backend/diagnostics` changes as part of the contract itself. Revisit when `GetRepairPlan` / `ApplyRepairs` are specified — both already declare a revision variable (C9) and are covered by the uniform validation rule in §5.6. |
| **Templates / Favorites** | **Not relevant** | Both store item *types* and configurations, not instances. They must never persist an `OwnedItemID`, because it is session-scoped by construction (§4.1). |

---

## 8. Decisions requested — all recorded as approved

These six decisions were requested by this document and have since been
approved. They are kept verbatim as the record of what was approved; Tasks 1,
2a, 2b and 3 were implemented under them, and the setter tasks that remain are
bound by them.

1. Approve **Variant D** as the `OwnedItemID` contract, with the format
   deliberately deferred (§4.1).
2. Approve the registry rules of §4.2: one registry per
   `(saveSessionID, characterID, saveRevision)` covering both containers, lazy
   per-container materialisation, minting independent of `page`, `pageSize` and
   `containerSection`, and a full clear — without eager rebuild — on increment.
3. Approve **per-session** `saveRevision`: internal `uint64` starting at 0,
   exposed publicly as a non-empty decimal string compared exactly (§5.1), or
   direct a per-character counter instead (H3).
4. Approve the unconditional `WriteSave` rule in §5.3: every write increments
   the revision and invalidates every outstanding identity.
5. Approve the rule in §1.4 that the twelve listed consumers take
   `saveSessionID`, that `SetEquippedArmaments` and `SetEquippedArmor` stay out
   of scope, and that amending the contract-only Go files is a separate,
   explicitly approved task.
6. Approve the task sequence in §6 — Task 1, Task 2a, the §6.3 native-save
   research gate, Task 2b, then Task 3 — acknowledging that Task 2a needed
   approval to extend the 15 listed full-result assertions, that Task 2b — not
   Task 2a — was the one that changed the two raw phase-one contract tests, and
   that `GetOwnedItem` waited for Task 2b rather than shipping a raw interim
   variant (§6.5).

The identity, the registry and the three getters are implemented, and so are the
revision increment together with the first mutation that drives it (§6.7) and the
first public mutation endpoint, `SetOwnedItemQuantity` (§6.8). `WriteSave` now
implements the §5.3 rule and is exposed by the local explorer. What remains
unimplemented is every other public setter endpoint and every other mutation.
