# SetWhetbladeUnlocked

## Purpose

`SetWhetbladeUnlocked` sets the complete Whetblade state of one catalog resource
for one active character. It synchronises the target Inventory item, its main
event flag, its related affinity flags and the shared Ashes of War menu flag.

## Contract

| Property | Value |
| --- | --- |
| Endpoint ID | `set_whetblade_unlocked` |
| Kind | Mutation |
| Transport | `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/whetblades/unlock` |
| Resource identity | Exact GameCatalog pair `whetbladeKind`, `whetbladeKey` |
| Concurrency | Exact canonical `expectedRevision` |

The JSON body contains:

```json
{
  "whetbladeKind": "item",
  "whetbladeKey": "4000230C",
  "unlocked": true,
  "expectedRevision": "0"
}
```

The result contains `saveSessionID`, the committed `saveRevision`,
`characterID`, the resolved Whetblade kind and key, and `unlocked`.

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
  `set_whetblade_unlocked`.
- `changedScopes` are exactly `save.session`, `inventory`, `world.flags`, `diagnostics.report`, in that canonical order.
  This mutation keeps the Whetblade goods record in step with its flags, so Inventory is invalidated beside the World flags. Storage is never touched, and the shared removal planner refuses a referenced record, so the loadout never changes.

A committed request identical to the current state still advances `saveRevision`
and still returns a complete receipt with a fresh `operationID`: the central
commit path runs even when no byte changes.

## Catalog validation

Before SaveEngine touches the slot, the endpoint validates the complete set of
catalog-declared Whetblades. Each resource must:

- be a goods item with exactly one `whetblade` unlock;
- have a unique goods game ID and main event flag;
- declare at least one `whetblade_related` event flag;
- declare exactly one `aow_menu_unlock` flag;
- use the same shared AoW menu flag as every other Whetblade;
- own no main or related event flag also owned by another Whetblade.

Unknown, incomplete or conflicting data fails closed. Raw game IDs and event
flags never enter the public request or result.

## Mutation

When `unlocked` is true, SaveEngine creates the target goods record in common
Inventory if no positive copy already exists, then sets the main flag, every
related affinity flag and the shared AoW menu flag.

When `unlocked` is false, SaveEngine removes the one target record from common
or key Inventory, clears the target flags, and clears the menu flag only when no
other Whetblade remains represented by its main flag or positive Inventory
record. Duplicate target records and referenced common records are rejected.

The Whetstone Knife relation to Storm Stomp is a `bundled_acquisition`, not part
of Whetblade state. This endpoint does not create, remove or repack the Storm
Stomp GaItem record and does not change its separate unlock flag.

All ranges are validated before the first write and applied as one verified byte
plan. Failure restores the previous bytes and does not advance the revision or
mark the session dirty. Persistence remains the responsibility of `WriteSave`.

## Compatibility evidence

SaveForge `v1.5.8` and `v1.6.10` use the same Whetblade main flags, affinity
flags, Inventory representations and shared menu rule. They also place the
Whetstone Knife system-affinity flag `1042378601` at byte `0xA0D0C`, bit 6 of
the Event Flags section. SaveForge 2.0 reimplements those confirmed save rules
through its existing PC/PS4 locators and does not import legacy code.

## Verification

Focused coverage checks PC and PS4 mutation, save-write and reload, the distant
Whetstone Knife flag, target item synchronisation, first/last Whetblade menu
semantics, invalid catalog data, strict transport JSON and OpenAPI conformance.
