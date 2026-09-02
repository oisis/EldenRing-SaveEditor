# LockAllSpectralSteedAttires

## Purpose

`LockAllSpectralSteedAttires` removes every Spectral Steed Attire item from the
Inventory of one active character and restores the default appearance of Torrent,
as one atomic mutation.

## Contract

| Property | Value |
| --- | --- |
| Endpoint ID | `lock_all_spectral_steed_attires` |
| Kind | Mutation |
| Transport | `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/spectral-steed-attires/lock-all` |
| Resource identity | The complete shared appearance table |
| Concurrency | Exact canonical `expectedRevision` |

The JSON body contains:

```json
{
  "expectedRevision": "0"
}
```

The result contains `saveSessionID`, the committed `saveRevision`, `characterID`
and `attireKey`, which is always `default` — the only state this operation can
leave behind.

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
  `lock_all_spectral_steed_attires`.
- `changedScopes` are exactly `save.session`, `inventory`, `world.flags`, `diagnostics.report`, in that canonical order.
  This mutation removes the attire records from Inventory and writes the appearance flags, so Inventory is invalidated beside the World flags. Storage holds no attire record and is not invalidated.

A committed request identical to the current state still advances `saveRevision`
and still returns a complete receipt with a fresh `operationID`: the central
commit path runs even when no byte changes.

## Why it is a dedicated endpoint

The same effect could be described as a sequence of `RemoveOwnedItem` calls
followed by `SetSpectralSteedAttire`. It is deliberately not built that way: a
failure between those calls would leave the save wearing an appearance whose item
is gone, or holding items for an appearance it can no longer select. This endpoint
therefore composes no other endpoint and calls SaveEngine once.

## Mutation

SaveEngine plans the removal of the three attire records and the four appearance
flags together and applies them as one verified byte plan, so the slot either
reaches the complete locked state or does not move at all. The three attire items
are the exact items of the shared appearance table documented in
[get_spectral_steed_attires.md](get_spectral_steed_attires.md); every other
Inventory record is preserved.

The removals share one section counter, which is written once per section rather
than once per record. An attire that is not held contributes nothing, so locking an
already locked character only restores the default appearance.

A duplicate or referenced attire record is refused before any byte moves. A
failure restores the previous bytes and advances neither the revision nor the
dirty flag. Persistence remains the responsibility of `WriteSave`.

## Compatibility evidence

SaveForge `v1.7.1` implements the same rule and states the same reason: composing
the generic remove operation with the appearance selection could leave the save
half-mutated. SaveForge 2.0 reimplements the confirmed behavior through its own
Inventory and event-flag plans and does not import legacy code.

## Verification

Focused coverage checks that exactly the three attire items are removed, that
unrelated records and the common section counter survive correctly, that the
default appearance becomes the resolved one, PC and PS4 mutation with save-write
and reload, a refused Lock All leaving neither Inventory nor flags changed, strict
transport JSON and OpenAPI conformance.
