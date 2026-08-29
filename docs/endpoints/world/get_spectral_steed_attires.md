# GetSpectralSteedAttires

## Purpose

`GetSpectralSteedAttires` returns the four Spectral Steed Attire appearances of
Torrent for one character: the default appearance and the three Regulation 1.17
attires. It reports which appearance the save has selected, or why that cannot be
decided, and which appearances the character owns.

## Contract

| Property | Value |
| --- | --- |
| Endpoint ID | `get_spectral_steed_attires` |
| Kind | Getter |
| Transport | `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/spectral-steed-attires` |
| Resource identity | Public appearance keys; event flags stay private |
| Concurrency | None; the getter never mutates |

The result contains `saveSessionID`, `characterID`, `active`, `status`,
`activeAttireKey` and the ordered `attires` array. Each entry carries
`attireKey`, `name`, `owned`, `requiredResourceKind`, `requiredResourceKey` and
`iconPath`.

## Appearance table

The four appearances are declared once, in `get_spectral_steed_attires.go`, and
that table is the single source of truth shared with `SetSpectralSteedAttire` and
`LockAllSpectralSteedAttires`:

| `attireKey` | Name | Required item |
| --- | --- | --- |
| `default` | Default Appearance | none |
| `tree_sentinel` | Tree Sentinel Spectral Steed Attire | item `401EAA00` |
| `silver_of_caria` | Silver of Caria Spectral Steed Attire | item `401EAA0A` |
| `funereal_night` | Funereal Night Spectral Steed Attire | item `401EAA14` |

The three items are resolved by their exact GameCatalog resource key. They carry
`safety.noDatabase`, so they are deliberately absent from the general
`GetResources` list and search while staying fully resolvable by that exact key —
which is also the key `AddItemToInventory` accepts.

The default appearance has no item, so its resource reference and its icon path
stay empty. Nothing is invented for it.

## Resolution state

The four appearance event flags are mutually exclusive, and the game sets exactly
one of them once the Spectral Steed Attire menu has been opened.

| `status` | Meaning | `activeAttireKey` |
| --- | --- | --- |
| `resolved` | Exactly one appearance flag is set | The active appearance |
| `legacy` | All four flags are cleared | Empty |
| `conflict` | Two or more flags are set | Empty |

`legacy` is the normal state of a save written before Regulation 1.17. A cleared
flag set is deliberately **not** read as "the default appearance is active": the
getter reports what it found and never guesses. An unreadable event flag region is
an error, never a legacy state. An inactive or residual slot reports `active`
false without its slot data being read.

## Ownership

The default appearance is always owned. An attire is owned only while its item has
a positive-quantity record in common or key `InventoryHeld`. Storage does not
count, and a set event flag is never accepted as proof of ownership, so an
appearance whose item was dropped is reported as not owned even while it is worn.

## Verification

Focused coverage checks the resolved state of each of the four appearances, the
legacy and conflict states, ownership from both native Inventory representations,
the residual-slot rule, that a repeated read reports the same state and advances
neither the revision nor the dirty flag, the catalog projection including icon
paths, transport equality with the getter and OpenAPI conformance.
