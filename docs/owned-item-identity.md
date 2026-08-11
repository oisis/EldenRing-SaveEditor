# OwnedItemID and saveRevision — shared Inventory contract

> **Status: Proposal — requires explicit approval before implementation.**
>
> This document designs a contract that does not exist in the code yet. It is
> not a description of implemented behaviour, so it lives outside
> [`docs/endpoints/`](endpoints/README.md), which documents implemented
> endpoints only, and outside `tools/swagger/openapi.json` and the Scalar
> portal. Nothing here is a runtime fact until a separate, explicitly approved
> task implements it.

| | |
|---|---|
| Scope | `OwnedItemID` and `saveRevision`, shared by the Inventory and Equipment surfaces |
| Proposed owner | `backend/saveengine` (one component), never an endpoint |
| Affected contracts today | exactly 12 contract-only endpoint files declaring `ownedItemID`, `weaponOwnedItemID` or `orderedOwnedItemIDs` |
| Affects implemented code | no — `GetInventory` and `GetStorage` stay phase 1 until a separate approved task |
| Transport | none proposed; no handler, no route, no OpenAPI entry, no Scalar entry |

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
| `GetOwnedItem` | declares `characterID`, `ownedItemID` — must gain `saveSessionID` |
| `SetOwnedItemQuantity`, `RemoveOwnedItem`, `MoveOwnedItemToInventory`, `MoveOwnedItemToStorage`, `SetWeaponUpgradeLevel`, `SetWeaponInfusion`, `SetSpiritAshUpgradeLevel` | declare `ownedItemID` without `saveSessionID` — must gain it |
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
   (`backend/saveengine/inventory.go:138-140`). The engine holds sessions in a
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

**No Go contract file changes in this task.** The twelve contract-only files
keep their current `SupportedResourceVariables` until a separate, explicitly
approved task amends the public contracts. That task also updates their contract
headers, the endpoint-structure expectations, and — for any endpoint that is
transport-exposed by then — the route, `openapi.json`, the document and the
Scalar navigation, per the synchronisation rules in `AGENTS.md`. This document
only records the decision those changes must follow.

### 1.5 Explicit non-goals

- No general `EndpointError` model. `tmp/app-se/endpoints-2.0.md` line 21
  defers that deliberately; this document only names the *information* an error
  must carry.
- No GaItem parser design. That is separate work and is a prerequisite for
  phase 2, not part of this contract.
- No change to the implemented phase-1 contract of `GetInventory` and
  `GetStorage` (see §3 of this document and §6.2).
- No persistence of identities across application restarts.
- No global registry of open save paths, no per-file lock and no new
  infrastructure of any kind (see the multi-session row of §7).

---

## 2. Evidence and its confidence

### 2.1 Confirmed facts (code in this repository, current branch)

| # | Fact | Source |
|---|---|---|
| C1 | `GetInventory` and `GetStorage` are implemented and return raw records with `containerSection` and `physicalIndex`, and both explicitly document that this pair is **not** a stable item identity. | `backend/saveengine/inventory.go:86-104`, `backend/saveengine/storage.go:125-140` |
| C2 | Neither getter returns an `OwnedItemID`. Both state so in the type comment and in their documents. | `backend/saveengine/inventory.go:106-121`, `docs/endpoints/inventory/get_inventory.md` |
| C3 | The phase-1 contract is locked by an assertion on the exact `SupportedResourceTypes` value and the exact ordered variable list. Changing either is a protected-test change. | `backend/endpoints/inventory/get_inventory_test.go:326`, `get_storage_test.go:395` |
| C4 | `saveengine.Session` holds only `id`, `platform`, `format`. There is no revision, no dirty flag, no per-character state. `SessionInfo.UnsavedChanges` is hard-coded `false`. | `backend/saveengine/session.go:21-63` |
| C5 | The engine is read-only today: `LoadSave`, `GetSessionInfo`, `CloseSession` and the getters. There is no mutation path, no rollback, no `WriteSave`, no `UndoCharacterChanges`. | `backend/saveengine/engine.go` |
| C6 | Two native sentinels mark an absent record: handle `0x00000000` and `0xFFFFFFFF`. Both getters skip them; every other handle is reported as stored. | `backend/saveengine/inventory.go:56-58` |
| C7 | The stored quantity carries a high bit that is not part of the count; it is masked with `0x7FFFFFFF` in exactly one place. | `backend/saveengine/inventory.go:52` |
| C8 | The 2.0 API spec forbids public setters that take raw bytes, offsets, handles, indices or event flags, and forbids accepting GaItem handles as a public identity. | `tmp/app-se/endpoints-2.0.md` lines 9 and 200 |
| C9 | Almost every 2.0 mutation contract declares `expectedRevision`; `GetRepairPlan` declares `saveRevision` as an input. The two names differ by direction of use, not by rule (§5.6). | `tmp/app-se/endpoints-2.0.md` lines 87, 97–178 |
| C10 | `GetOwnedItem`, every inventory mutation contract and `SetEquippedTalismans` declare `characterID` but **not** `saveSessionID`, while the implemented `GetInventory`/`GetStorage` require `saveSessionID`. | `backend/endpoints/inventory/*.go`, `backend/endpoints/equipment/set_equipped_talismans.go:25` vs `get_inventory.go:32` |
| C11 | `containerSection` is an input *filter* on both implemented getters: the empty string means both physical sections, `"common"` and `"key"` select one. It is not part of any record identity. | `backend/saveengine/inventory.go:14-19, 145-148` |
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
  (L3) but that is tolerance, not proof. **Open — decide before phase 2.**
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

**There is currently no native-save evidence in this repository for the
persistence of an item instance.**

- No `.sl2` or native `.dat` file is tracked by git (`git ls-files` returns
  none).
- Every SaveEngine test builds a **synthetic** fixture in a temp directory
  (`writeInventoryFixture`, `backend/saveengine/inventory_test.go:81`). A
  synthetic fixture proves that the reader handles a byte layout; it does not
  prove that the layout is what the game writes.
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
phase 1: raw native records, no catalog resolution, no identity (C1–C3). The
tests `TestGetInventoryContractIsRawPhaseOne` and
`TestGetStorageContractIsRawPhaseOne` assert the exact `SupportedResourceTypes`
value and the exact ordered variable list.

Phase 2b (§6.4) changes that contract and therefore requires explicit user
approval under the regression-test rules of `AGENTS.md`, with replacement
coverage at least as strong. Until then:

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

The pair already returned by the phase-1 getters.

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
| Unknown / malformed record data | The record still gets an ID and is still listed, exactly as phase 1 lists an unresolvable handle (C6). What must never happen is an unknown record being dropped, silently repaired, or turned into a different valid item. |

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
IDs and its slot data is never searched, exactly as phase 1 already behaves. The
read is still a valid, non-error result and still returns the current
`saveRevision`; an empty container is not an excuse to omit the revision.

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

Because the engine is read-only today (C5), the revision starts and stays at 0
in this task; the increment path is written but has no caller until the first
mutation lands. The tests must nevertheless prove the increment, the rollback
non-increment and the invalidation, using an internal test hook rather than a
public mutation.

### 6.2 Task 2a — `GetInventory` and `GetStorage` return the identity

The two getters mint and return `OwnedItemID` per record and `saveRevision` per
result. **Nothing else changes:** no catalog resolution, no `kind`, no `key`, no
name, no family, no variant — those stay in Task 2b.

**What this task does to the contract definitions: nothing.** It does not change
`SupportedResourceTypes` and does not change the input-variable lists of either
endpoint. Therefore `TestGetInventoryContractIsRawPhaseOne`
(`backend/endpoints/inventory/get_inventory_test.go:326`) and
`TestGetStorageContractIsRawPhaseOne`
(`backend/endpoints/inventory/get_storage_test.go:395`) stay exactly as they are
and must keep passing untouched.

**What this task does require approval for: the result shape.** Adding two
fields to the public result widens what the getters return, so every test that
asserts a *complete* result or a *complete* record list with
`reflect.DeepEqual` will fail until its expected value gains the new fields.
Naming this change plainly: it is **an extension of the public result plus
correspondingly stronger assertions** — each listed assertion continues to
compare the full value exactly, over a larger value. It is not a weakening, not
a relaxation, not a switch to partial matching, and no case, boundary or fixture
is removed.

The exact assertions that need explicit user approval before Task 2a starts are
these **15 full-result assertions**:

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
minting no IDs, so its result is no longer the zero value. They must become an
explicit expected value carrying the revision, not a looser check.

**Four further assertions are *not* on the list and must not be touched:**

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
value — widening the result shape does not change them, and they must keep
passing untouched. An earlier revision of this document mistook them for the
inactive-slot assertions and counted 19; the approved count is 15.

None of these assertions is weakened, removed, skipped or altered by *this*
documentation task.

This task also updates `docs/endpoints/inventory/get_inventory.md`,
`docs/endpoints/inventory/get_storage.md`, `tools/swagger/openapi.json` and
`tools/swagger/main_test.go` for the widened result, per the synchronisation
rules in `AGENTS.md`. The route, the endpoint index and the Scalar navigation do
not change, because no endpoint is added or removed.

### 6.3 Gate — controlled native-save research for the GaItem parser

Not a code task, and it produces no commit in `backend/`. It is a blocking
prerequisite of Task 2b, stated as its own step because a prerequisite buried in
a paragraph is a prerequisite that gets skipped.

§2.6 records that this repository holds **no** native-save evidence for the
GaItem record: every fixture is synthetic, and the legacy record model (L9) is
legacy testimony, not a read of a real save. Resolving a record to `kind`, `key`
and a variant requires parsing that record, so Task 2b cannot start on the
current evidence without turning a hypothesis into production behaviour, which
`AGENTS.md` forbids.

This gate closes when the record layout is established from controlled evidence
on read-only copies, across the applicable platform and slot-version variants,
and the findings are written down with their confidence separated as §2 does.
Until then Task 2b, and therefore everything downstream of it, stays blocked.
Task 1 and Task 2a are unaffected: neither parses a record.

### 6.4 Task 2b — catalog resolution in the two getters

Only after Task 2a ships and the §6.3 gate is closed. This is the task that
resolves records against GameCatalog and ends the raw phase-1 contract. It
changes `SupportedResourceTypes` and the documented contract, so it — and only
it — requires explicit approval to modify
`TestGetInventoryContractIsRawPhaseOne` and
`TestGetStorageContractIsRawPhaseOne` (C3), reporting the exact assertions, the
old contract, the conflict, and replacement coverage that is at least as strong.

It carries the full synchronisation set in the same task:
`docs/endpoints/inventory/*.md`, `docs/endpoints/README.md`,
`tools/swagger/main.go`, `tools/swagger/openapi.json`,
`tools/swagger/docs-portal/scalar.config.json` and `tools/swagger/main_test.go`.

### 6.5 Task 3 — `GetOwnedItem` and the setters as consumers

**`GetOwnedItem` comes last of the getters, not first.** It may only start after
Task 1, Task 2a, the §6.3 research gate and Task 2b, in that order.

The reason is its own contract, not a scheduling preference.
`backend/endpoints/inventory/get_owned_item.go:24` declares
`SupportedResourceTypes: "ItemDocument"`, so its result must resolve one owned
instance to `kind`, `key` and its variant. Task 2a deliberately produces none of
those: it returns raw records carrying an `OwnedItemID` and a `saveRevision`,
with no GaItem parser, no GameCatalog read and no catalog identity (§6.2).
Task 2b is where records become semantically resolved, and it in turn waits on
the §6.3 gate. `GetOwnedItem` therefore needs both, and an ordering that put it
after Task 2a alone would ask it to answer a question its inputs cannot answer.

Three things this explicitly rules out:

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
first meet in a single-instance result. It still goes before the mutations,
because it is a getter and can be verified without any mutation path existing.
The mutations follow, each in its own task, each depending additionally on the
mutation/rollback machinery that this document does not design. The
`saveSessionID` amendment of the twelve contract files (§1.4) is part of this
stage, one endpoint at a time.

### 6.6 Required test coverage

Every task above must cover, at the layer where the behaviour is observable:

| Dimension | Required cases |
|---|---|
| Platform | PC and PS4, both producing IDs and both rejecting a foreign one. |
| Slot state | Active, inactive, and residual (a deleted character whose data is still in the file) — a residual slot mints no IDs and its data is not searched, but the result still carries the current revision. |
| Sentinels | Handles `0x00000000` and `0xFFFFFFFF` get no ID and stay invisible (C6). |
| Unknown / malformed data | An unresolvable handle still gets an ID and is still listed; it is never dropped, repaired or reinterpreted. |
| Stackable goods | The same `kind`/`key` present in Inventory and in Storage yields two distinct IDs; two rows of the same stackable item in one container yield two distinct IDs. |
| Idempotent minting | Re-reading the same revision returns identical IDs; page 1 then page 2, both sections then one section, and Inventory read before Storage or after it, all yield the same ID for the same physical record (§4.2). |
| Lazy materialisation | Reading only Inventory does not mint Storage entries, and the later first Storage read of the same revision succeeds and mints them. |
| Stale IDs after a mutation | An ID minted at revision *n* is rejected at revision *n+1*, with an error distinguishable from "unknown". |
| Revision | Initial `"0"`; +1 on commit; unchanged on validation failure; unchanged on rollback; +1 on undo; +1 on every `WriteSave`, asserted before and after the write. Every assertion compares the decimal string exactly. |
| Concurrency | `-race` over concurrent getter/mutation access to one session — mandatory, because the registry and the counter are shared mutable state behind `Engine.mutex`. |
| Cross-session | An ID from session A is rejected by session B, and by a reopened session on the same file. |

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

## 8. Decisions requested

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
   research gate, Task 2b, then Task 3 — acknowledging that Task 2a needs
   approval to extend the 15 listed full-result assertions, that Task 2b — not
   Task 2a — is the one that changes the two phase-1 contract tests, and that
   `GetOwnedItem` waits for Task 2b rather than shipping a raw interim variant
   (§6.5).

Nothing in this document is implemented, and nothing may be implemented until
these decisions are recorded.
