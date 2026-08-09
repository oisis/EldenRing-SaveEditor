# SaveForge 2.0 — Architecture Boundaries

## 1. Purpose

This document is the map of the physical boundaries of SaveForge 2.0 inside this
monorepo. It states which directories and files already belong to SaveForge 2.0,
which directories are planned for it, and which existing paths belong to the old
application and must never become a dependency of the new architecture.

The goal is to keep the two architectures physically separated: SaveForge 2.0 is
built greenfield, and the legacy application is research material only. Anyone
adding a file should be able to answer, from this document alone, whether that
file belongs to SaveForge 2.0 and where it goes.

## 2. Current SaveForge 2.0 paths

| Path | Status | Responsibility | Notes |
| --- | --- | --- | --- |
| `docs/endpoints/` | Exists | Technical documentation of SaveForge 2.0 endpoints. | Documentation only; no runtime effect. |
| `docs/endpoints/README.md` | Exists | Index of endpoint documentation and endpoint statuses. | Entry point for endpoint docs. |
| `docs/endpoints/SF-2.0.md` | Exists | This architecture boundary document. | Must be updated whenever a boundary changes. |
| `backend/endpoints/` | Exists | Public contracts and implementations of SaveForge 2.0 endpoints. | One public endpoint lives in one file. |
| `backend/endpoints/catalog/` | Exists | Getters reading GameCatalog. | Read-only access to catalog data. |
| `backend/endpoints/appearance/` | Exists | Appearance endpoints, including `GetAppearancePresets`. | Presets are served from GameCatalog data. |
| `backend/endpoints/application/` | Exists | Application metadata endpoints. | Application-level information only. |
| `backend/endpoints/network/` | Exists | Network preset endpoints and future save network-state endpoints. | Presets come from GameCatalog regulation data. |
| `backend/endpoints/savesession/` | Exists | Contracts and future implementations of the save session lifecycle, including `LoadSave` and `GetLoadedSave`. | Implementations depend on the planned SaveEngine. |
| `backend/endpoints/swagger/` | Exists | Local developer HTTP/OpenAPI explorer. | Not the application frontend and not part of the application runtime. |
| `backend/endpoints/contract/` | Exists | Endpoint contract definitions and their structural validation. | Single source of truth for endpoint shape rules. |
| `backend/endpoints/character/` | Contracts only | Existing character endpoint contracts. | Not implemented. Implementations will require SaveEngine and a save session. |
| `backend/endpoints/diagnostics/` | Contracts only | Existing contracts for diagnostics and future non-mutating reports and repair plans. | Not implemented. Scans and previews must stay non-mutating. |
| `backend/endpoints/equipment/` | Contracts only | Existing contracts for equipment reading and future equipment mutation. | Not implemented. Implementations will require SaveEngine. |
| `backend/endpoints/favorites/` | Contracts only | Existing Favorites contracts. | Not implemented. Their future persistent data source requires a separate architectural decision. |
| `backend/endpoints/inventory/` | Contracts only | Existing inventory and storage contracts. | Not implemented. Implementations will require SaveEngine and GameCatalog. |
| `backend/endpoints/itemrouting/` | Exists | Internal validator of the shared item routing contract for future mutations. | Not a public endpoint domain. |
| `backend/endpoints/templates/` | Contracts only | Existing Build Templates contracts. | Not implemented. Their future persistent data source requires a separate architectural decision. |
| `backend/endpoints/world/` | Contracts only | Existing world and progression contracts. | Not implemented. Implementations will require SaveEngine and GameCatalog. |
| `backend/gamecatalog/` | Exists | Independent, immutable source of truth about game assets and SaveForge 2.0 configuration data. | Never mutated at runtime. |
| `backend/gamecatalog/data/` | Exists | Embedded GameCatalog data. | Generated game asset data under `items/` changes only through its generator. SaveForge 2.0 configuration data and assets, such as `regulation/network_params.json`, `presets/appearance.json` and `assets/appearance`, are versioned catalog data and require an explicit, reviewed change together with loader validation. No part of GameCatalog may be modified at application runtime. |
| `backend/gamecatalog/data/items/` | Exists | Game asset data. | Item definitions and related asset records. |
| `backend/gamecatalog/data/regulation/` | Exists | Data derived from regulation, such as `network_params.json`. | Provenance must stay traceable to regulation. |
| `backend/gamecatalog/data/presets/` | Exists | SaveForge 2.0 preset configuration, currently `appearance.json`. | Application-owned configuration, not game data. |
| `backend/gamecatalog/data/assets/` | Exists | Assets owned by GameCatalog, currently item icons and appearance preset images. | Embedded GameCatalog assets. Endpoints may currently return their file names as metadata; no public HTTP route for fetching assets exists yet. |
| `tmp/app-se/` | Exists | Project specifications and architectural decisions for SaveForge 2.0. | Developer material, not application runtime. |

## 3. Planned SaveForge 2.0 paths

These paths do not exist yet. They are reserved locations, listed here so that
implementation work lands in the agreed place.

| Path | Status | Responsibility | Notes |
| --- | --- | --- | --- |
| `backend/saveengine/` | Planned | Independent owner of save reading and, later, save modification. | Must not import or absorb implementations from `backend/core`. |
| `backend/saveengine/engine.go` | Planned | SaveEngine facade that opens a save read-only and creates a session. | Public entry point used by endpoints. |
| `backend/saveengine/session.go` | Planned | Save session model and safe session metadata. | Session state and its exposed metadata only. |
| `backend/saveengine/pc.go` | Planned | PC format reading and recognition. | PC-specific behavior stays PC-specific. |
| `backend/saveengine/ps4.go` | Planned | PS4 format reading and recognition. | PS4-specific behavior stays PS4-specific. |
| `backend/saveengine/codec.go` | Planned | Private component performing raw save data reading. | Not exported outside SaveEngine. |

The actual internal split may be refined once PC and PS4 format differences are
confirmed by evidence. That flexibility applies to SaveEngine internals only:
endpoints must never take over SaveEngine logic, regardless of how SaveEngine is
divided into files.

## 4. Explicitly excluded legacy paths

The following existing paths belong to the old application. They are not a
source of implementation for SaveForge 2.0.

| Path | Status | Relation to SaveForge 2.0 |
| --- | --- | --- |
| `backend/core/` | Legacy | Research material only. Not importable by SaveForge 2.0. |
| `backend/db/` | Legacy | Research material only. Not importable by SaveForge 2.0. |
| `backend/editor/` | Legacy | Research material only. Not importable by SaveForge 2.0. |
| `backend/templates/` | Legacy | Research material only. Not importable by SaveForge 2.0. |
| `backend/vm/` | Legacy | Research material only. Not importable by SaveForge 2.0. |
| `internal/` | Legacy | Research material only. Not importable by SaveForge 2.0. |
| `frontend/` | Legacy | Old application UI. The SaveForge 2.0 frontend does not exist yet. |

Rules for these paths:

- Legacy code may be used only as research material and as a source of confirmed
  data and confirmed findings.
- Its structures, helpers and interfaces must not be copied, and its packages
  must not be imported into SaveForge 2.0.
- SaveForge 2.0 is built greenfield. A legacy implementation is never an
  argument for a SaveForge 2.0 design decision.
- Data may be migrated into GameCatalog only after validation, and only with its
  own recorded provenance.

Not every directory under `backend/` is legacy — `backend/endpoints/` and
`backend/gamecatalog/` are SaveForge 2.0, and only the directories listed in the
table above are excluded. This document maps the established SaveForge 2.0
boundaries and the explicitly excluded legacy paths; it is not a complete
inventory of the whole repository.

## 5. Boundary rules

- GameCatalog describes the meaning of game data and stays immutable at runtime.
- SaveEngine will be the sole owner of save reading and later save modification.
- Codec will be the only component with direct access to raw save bytes.
- Endpoints are thin and contain no GameCatalog, SaveEngine or Codec rules.
- One public endpoint lives in one file.
- `LoadSave` creates a session but never modifies the input file.
- The first SaveEngine stage supports PC and PS4 read-only.
- Swagger is a developer tool; the application frontend does not exist yet and is
  outside the current scope.
- Do not add new SaveForge 2.0 paths without a prior architectural decision.
