# GetItemCapacity

## Overview

`GetItemCapacity` is a read-only preflight for adding one catalog resource or
variant to the common section of Inventory or Storage. It reports the current
item quantity, the quantity after the planned delta, the applicable catalog
limit, free and required physical rows, and free and required active
`GaItemData` entries.

The result is **not a reservation**. It carries the `saveRevision` read with the
snapshot, but a later mutation must still verify its own `expectedRevision` and
repeat every check under the mutation lock.

| | |
|---|---|
| EndpointID | `get_item_capacity` |
| Kind | Getter |
| Domain | `inventory` |
| Implementation status | implemented |
| Transport | `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/item-capacity` in the local explorer |
| Save access | private session snapshot, read-only |
| Implementation | [../../../backend/endpoints/inventory/get_item_capacity.go](../../../backend/endpoints/inventory/get_item_capacity.go) |
| Tests | [../../../backend/endpoints/inventory/get_item_capacity_test.go](../../../backend/endpoints/inventory/get_item_capacity_test.go) and [../../../backend/saveengine/item_capacity_test.go](../../../backend/saveengine/item_capacity_test.go) |

## Supported resources

The endpoint accepts only `ItemDocument` resources of family `goods` or
`talisman`. These families use handles derived from the game ID and need no
record in the variable-length `GaItem` table. Weapon, armour, Ash of War, spell
and spirit-ash resources are rejected; the getter does not revive or predict the
retired GaItem repacker.

The following resources are also rejected fail-closed:

- category `key_items`, because the catalog does not distinguish its common and
  key routing subsets;
- subcategory `Flasks`, because the catalog's per-row `maxInventory=20` does not
  express the game's shared maximum of Sacred Flask charges;
- goods targeting Storage when `item.goods.isDepositable` is unknown or false;
- resources whose family, game-ID prefix, record mode or relevant limits are
  unknown or inconsistent.

The family, prefix, category and record-mode checks are shared with the common
item-add endpoints. Inventory uses
`min(item.capabilities.stack.rules.maxPerStack, item.storage.maxInventory)` as
the per-record stack limit. Storage uses `item.storage.maxStorage`, matching the
repository quantity rule in SaveForge 1.5.8 and 1.6.8. A separate-instance item
accepts quantity `1` and consumes one new physical row.

## Request

```http
GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/item-capacity?destination=inventory&kind=item&key=400006A4&quantity=5
```

| Parameter | Location | Type | Rule |
|---|---|---|---|
| `saveSessionID` | path | string | Existing session, matched exactly. |
| `characterID` | path | integer | Physical character slot `0..9`. |
| `destination` | query | string | Exactly `inventory` or `storage`. |
| `kind` | query | string | Resource kind, matched exactly. |
| `key` | query | string | Resource key inside `kind`, matched exactly. |
| `variantID` | query | optional uint32 | Decimal game ID of one stored variant of the same resource. Absence selects the base document. |
| `quantity` | query | uint32 | Planned positive delta. Separate instances accept `1` only. |

## Response

```json
{
  "saveSessionID": "6bd5...",
  "saveRevision": "3",
  "characterID": 0,
  "active": true,
  "destination": "inventory",
  "kind": "item",
  "key": "400006A4",
  "gameID": 1073743524,
  "quantity": 5,
  "canFit": true,
  "limitingFactor": "",
  "currentQuantity": 10,
  "quantityAfter": 15,
  "maxContainerQuantity": 40,
  "freePhysicalRecords": 2670,
  "physicalRecordsRequired": 0,
  "freeGaItemDataEntries": 6812,
  "gaItemDataEntriesRequired": 0
}
```

`currentQuantity` is the sum of the resolved game ID in the selected container.
For a stack already present in common, `physicalRecordsRequired` is zero and the
first physical stack is checked against its per-record limit. A new stack or a
new separate instance requires one row.

`gaItemDataEntriesRequired` is one only when the character owns no physical
record of the game ID in either container and the active `GaItemData` prefix
does not already contain it. Goods and talismans never require a serialized
`GaItem` record, so no GaItem-table budget is exposed.

`limitingFactor` is the first failed budget in deterministic order:

| Value | Meaning |
|---|---|
| empty string | Every checked budget fits; `canFit` is true. |
| `per_record_quantity` | The first existing stack would exceed its row limit. |
| `container_quantity` | The selected container total would exceed its catalog limit. |
| `physical_records` | No safe common row is available. |
| `acquisition_allocator` | A new row cannot receive safe acquisition metadata. |
| `gaitemdata_entries` | A new active `GaItemData` entry is required but the prefix is full. |

An inactive or residual slot is a normal `active:false`, `canFit:false` result.
Its slot data is not searched. Structural corruption, an unresolvable existing
handle or an existing matching item in either key section is an error, not a
capacity result.

## Read-only guarantee

SaveEngine performs the complete read while holding its session lock. It reads
the private snapshot directly and deliberately does not call the public
Inventory or Storage getters, because those getters mint revision-scoped
`ownedItemID` values. `GetItemCapacity` changes no byte, revision, dirty flag,
identity registry or sequence counter.

PC and PS4 differ only in their slot and container bases. Existing
platform-specific locators select those bases; the Inventory, Storage and
GaItemData structures inside the slot are interpreted identically.
