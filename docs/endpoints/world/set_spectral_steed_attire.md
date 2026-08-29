# SetSpectralSteedAttire

## Purpose

`SetSpectralSteedAttire` activates exactly one Spectral Steed Attire appearance of
Torrent for one active character.

## Contract

| Property | Value |
| --- | --- |
| Endpoint ID | `set_spectral_steed_attire` |
| Kind | Mutation |
| Transport | `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/spectral-steed-attires/select` |
| Resource identity | Public appearance key `attireKey` |
| Concurrency | Exact canonical `expectedRevision` |

The JSON body contains:

```json
{
  "attireKey": "tree_sentinel",
  "expectedRevision": "0"
}
```

The result contains `saveSessionID`, the committed `saveRevision`, `characterID`
and the selected `attireKey`.

## Accepted appearances

`attireKey` accepts only `default`, `tree_sentinel`, `silver_of_caria` and
`funereal_night`. They come from the shared appearance table documented in
[get_spectral_steed_attires.md](get_spectral_steed_attires.md), which also owns
the event flags and required items behind them. Nothing else in this endpoint
restates that mapping.

## Item requirement

The default appearance needs no item. Each of the other three requires its attire
item as a positive-quantity record in Inventory; Storage does not count and a set
event flag is not proof of ownership.

A missing item is rejected before the first byte moves, so the session, the
snapshot and the revision stay exactly as they were. This endpoint never adds the
item: it is added beforehand through `AddItemToInventory` by the item's exact
GameCatalog resource key.

## Mutation

SaveEngine clears all four appearance flags and sets the selected one as a single
verified byte plan, so a successful call always leaves exactly one flag set. It
therefore also resolves a `legacy` save whose flags are all cleared and a
`conflict` save with more than one set.

Selecting the appearance that is already active moves no byte and still commits
one revision, which is the revision contract every mutation of this engine
follows. A mutation that changed no byte records no undo point.

A failure restores the previous bytes and advances neither the revision nor the
dirty flag. Persistence remains the responsibility of `WriteSave`.

## Compatibility evidence

SaveForge `v1.7.1` confirms the same contract on PC native Regulation 1.17 saves:
event flags 6700-6703 are mutually exclusive, a cleared set means the save predates
1.17 and must not be read as "default", and an attire requires its key item while
the endpoint never adds it. Its later PS4 fix removed the platform gate entirely:
the appearance flags carry no platform-specific behavior. SaveForge 2.0
reimplements those confirmed save rules through its own shared PC/PS4 event-flag
locator and does not import legacy code.

## Verification

Focused coverage checks every appearance key, a missing required item, an unknown
key, a stale and a malformed revision, the resolution of legacy and conflict
states, the repeated-selection revision contract, PC and PS4 mutation with
save-write and reload, strict transport JSON and OpenAPI conformance.
