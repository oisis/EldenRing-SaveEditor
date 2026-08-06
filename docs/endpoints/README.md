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
| `GetCatalogInfo` | `get_catalog_info` | Getter | `catalog` | implemented | not exposed through Wails/HTTP/CLI | [catalog/get_catalog_info.md](catalog/get_catalog_info.md) |
| `GetResource` | `get_resource` | Getter | `catalog` | implemented | not exposed through Wails/HTTP/CLI | [catalog/get_resource.md](catalog/get_resource.md) |

Only implemented endpoints are documented. The remaining contract-only endpoints
defined in `backend/endpoints` get a document when their runtime handler lands.
