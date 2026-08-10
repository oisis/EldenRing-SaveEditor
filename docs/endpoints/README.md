# SaveForge 2.0 endpoint documentation

This directory holds the technical documentation of the public SaveForge 2.0
backend API. Each document describes one public endpoint: what it returns or
mutates, which input it accepts, how it validates that input, which errors it
produces, and how to verify it locally.

## Naming convention

```
docs/endpoints/<domain>/<endpoint_id>.md
```

The file name is the endpoint's `EndpointID` and the directory structure mirrors
`backend/endpoints`, so `backend/endpoints/catalog/get_catalog_info.go` is
documented in `docs/endpoints/catalog/get_catalog_info.md`.

Every public endpoint has its own endpoint file in `backend/endpoints`. That
file is either `contract-only`, meaning it defines the contract without a
runtime handler, or `implemented`, meaning the handler lives there too. A
technical document is written once the runtime handler is added, so a
contract-only endpoint has an endpoint file but no document yet.

Endpoints are never grouped into a shared document, and a document is never
created for a group of endpoints.

## Getters and mutations

The public API separates reads from writes:

- **Getters** only read state. They never modify a save, the GameCatalog, or any
  application data.
- **Mutations** (setters and other mutating endpoints) perform exactly one
  explicitly named operation. A mutation may return a typed result describing
  the resulting state, but it never doubles as a getter.

A single endpoint never switches between reading and writing based on a
parameter such as `action` or `mode`.

## What this documentation is

These documents describe the backend code as it exists today. They are a
technical description, not a specification: they do not replace the endpoint
contracts in `backend/endpoints` and they do not replace the implementation.
When code and documentation disagree, the code and its tests are authoritative
and the document must be corrected.

Documents describe implemented behaviour only. They do not design future
architecture, transports, or contracts.

## Endpoint statuses

Each document reports two independent statuses.

Implementation status:

| Status | Meaning |
|---|---|
| `contract-only` | The endpoint file defines the contract (`EndpointID`, `Definition`) but no runtime handler exists yet. |
| `implemented` | A runtime handler exists in the endpoint file and is covered by tests. |

Transport status:

| Status | Meaning |
|---|---|
| `not exposed` | The endpoint is callable only as a Go function. No Wails binding, HTTP route, or CLI command reaches it. |
| `transport-exposed` | The endpoint is reachable through at least one transport (Wails, HTTP, or CLI), named in the document. |

## Documented endpoints

| Name | EndpointID | Kind | Domain | Implementation status | Transport status | Document |
|---|---|---|---|---|---|---|
| `GetApplicationInfo` | `get_application_info` | Getter | `application` | implemented | transport-exposed — `GET /api/v1/application/info` of the local explorer | [application/get_application_info.md](application/get_application_info.md) |
| `GetCatalogInfo` | `get_catalog_info` | Getter | `catalog` | implemented | transport-exposed — `GET /api/v1/catalog/info` of the local explorer | [catalog/get_catalog_info.md](catalog/get_catalog_info.md) |
| `GetResource` | `get_resource` | Getter | `catalog` | implemented | transport-exposed — `GET /api/v1/catalog/resource` of the local explorer | [catalog/get_resource.md](catalog/get_resource.md) |
| `GetItemVariants` | `get_item_variants` | Getter | `catalog` | implemented | transport-exposed — `GET /api/v1/catalog/item-variants` of the local explorer | [catalog/get_item_variants.md](catalog/get_item_variants.md) |
| `GetResourceRelations` | `get_resource_relations` | Getter | `catalog` | implemented | transport-exposed — `GET /api/v1/catalog/resource-relations` of the local explorer | [catalog/get_resource_relations.md](catalog/get_resource_relations.md) |
| `GetResources` | `get_resources` | Getter | `catalog` | implemented | transport-exposed — `GET /api/v1/catalog/resources` of the local explorer | [catalog/get_resources.md](catalog/get_resources.md) |
| `GetNetworkPresets` | `get_network_presets` | Getter | `network` | implemented | transport-exposed — `GET /api/v1/network/presets` of the local explorer | [network/get_network_presets.md](network/get_network_presets.md) |
| `GetNetworkSettings` | `get_network_settings` | Getter | `network` | implemented | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/network-settings` of the local explorer, registered only when the explorer runs without `-allow-external-bind` | [network/get_network_settings.md](network/get_network_settings.md) |
| `GetAppearancePresets` | `get_appearance_presets` | Getter | `appearance` | implemented | transport-exposed — `GET /api/v1/appearance/presets` of the local explorer | [appearance/get_appearance_presets.md](appearance/get_appearance_presets.md) |
| `LoadSave` | `load_save` | Mutation | `savesession` | implemented | transport-exposed — `POST /api/v1/save-sessions` of the local explorer, registered only when the explorer runs without `-allow-external-bind` | [savesession/load_save.md](savesession/load_save.md) |
| `GetLoadedSave` | `get_loaded_save` | Getter | `savesession` | implemented | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}` of the local explorer, registered only when the explorer runs without `-allow-external-bind` | [savesession/get_loaded_save.md](savesession/get_loaded_save.md) |
| `CloseSave` | `close_save` | Mutation | `savesession` | implemented | transport-exposed — `DELETE /api/v1/save-sessions/{saveSessionID}` of the local explorer, registered only when the explorer runs without `-allow-external-bind` | [savesession/close_save.md](savesession/close_save.md) |
| `GetSaveCharacters` | `get_save_characters` | Getter | `character` | implemented | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters` of the local explorer, registered only when the explorer runs without `-allow-external-bind` | [character/get_save_characters.md](character/get_save_characters.md) |
| `GetCharacterProfile` | `get_character_profile` | Getter | `character` | implemented | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/profile` of the local explorer, registered only when the explorer runs without `-allow-external-bind` | [character/get_character_profile.md](character/get_character_profile.md) |
| `GetCharacterStats` | `get_character_stats` | Getter | `character` | implemented | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/stats` of the local explorer, registered only when the explorer runs without `-allow-external-bind` | [character/get_character_stats.md](character/get_character_stats.md) |
| `GetCharacterAppearance` | `get_character_appearance` | Getter | `character` | implemented | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance` of the local explorer, registered only when the explorer runs without `-allow-external-bind` | [character/get_character_appearance.md](character/get_character_appearance.md) |
| `GetEquipment` | `get_equipment` | Getter | `equipment` | implemented | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipment` of the local explorer, registered only when the explorer runs without `-allow-external-bind` | [equipment/get_equipment.md](equipment/get_equipment.md) |
| `GetQuickItems` | `get_quick_items` | Getter | `equipment` | implemented | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/quick-items` of the local explorer, registered only when the explorer runs without `-allow-external-bind` | [equipment/get_quick_items.md](equipment/get_quick_items.md) |
| `GetPouchItems` | `get_pouch_items` | Getter | `equipment` | implemented | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/pouch-items` of the local explorer, registered only when the explorer runs without `-allow-external-bind` | [equipment/get_pouch_items.md](equipment/get_pouch_items.md) |
| `GetPhysickMixture` | `get_physick_mixture` | Getter | `equipment` | implemented | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/physick-mixture` of the local explorer, registered only when the explorer runs without `-allow-external-bind` | [equipment/get_physick_mixture.md](equipment/get_physick_mixture.md) |
| `GetEquippedSpells` | `get_equipped_spells` | Getter | `equipment` | implemented | transport-exposed — `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/equipped-spells` of the local explorer, registered only when the explorer runs without `-allow-external-bind` | [equipment/get_equipped_spells.md](equipment/get_equipped_spells.md) |

Only implemented endpoints are documented. The remaining contract-only endpoints
defined in `backend/endpoints` get a document when their runtime handler lands.
