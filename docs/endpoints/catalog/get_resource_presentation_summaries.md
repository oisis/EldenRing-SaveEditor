# GetResourcePresentationSummaries

## Overview

`GetResourcePresentationSummaries` returns lightweight presentation metadata for
an ordered batch of exact GameCatalog identities. It exists for screens that
already know the resources stored by a save and need names and icons without
loading a full resource document once per record.

| | |
|---|---|
| EndpointID | `get_resource_presentation_summaries` |
| Kind | Getter |
| Domain | `catalog` |
| Implementation status | implemented |
| Transport status | transport-exposed — Wails bridge method `GetResourcePresentationSummaries` and `POST /api/v1/catalog/resource-presentation-summaries` of the local OpenAPI explorer |
| Implementation source | [../../../backend/endpoints/catalog/get_resource_presentation_summaries.go](../../../backend/endpoints/catalog/get_resource_presentation_summaries.go) |
| Test source | [../../../backend/endpoints/catalog/get_resource_presentation_summaries_test.go](../../../backend/endpoints/catalog/get_resource_presentation_summaries_test.go) |
| Save access | none |
| Mutation | none |

## Input

The Go signature is:

```go
func GetResourcePresentationSummaries(
    gameCatalog *gamecatalog.Catalog,
    identities []ResourcePresentationIdentity,
) (GetResourcePresentationSummariesResult, error)
```

Each identity has exactly two public fields:

```go
type ResourcePresentationIdentity struct {
    Kind string `json:"kind"`
    Key  string `json:"key"`
}
```

`kind` and `key` are matched exactly and case-sensitively. They are not trimmed,
normalised, recased, parsed as a `GameID`, retried under another kind, or resolved
through an alias. The input order is significant and duplicate identities are
valid. A nil or empty list is a valid request.

## Output

```go
type ResourcePresentationSummary struct {
    Kind     string `json:"kind"`
    Key      string `json:"key"`
    Name     string `json:"name"`
    IconPath string `json:"iconPath"`
}

type GetResourcePresentationSummariesResult struct {
    Resources []ResourcePresentationSummary `json:"resources"`
}
```

The result preserves the exact input order and duplicates. It returns only four
scalar fields per identity. It never includes an item document, variants,
relations, capabilities, provenance, safety facts, descriptions, statistics, or
save state.

`name` and `iconPath` are empty when their GameCatalog facts are unknown. The
endpoint never falls back to a key, category, placeholder, or derived asset
name. Non-item resources may have a name but do not have an item icon path.

An empty input returns:

```json
{"resources":[]}
```

The array is never `null` on success.

## Resolution and safety

Every pair is resolved through `Catalog.ResourceSummaryByKindAndKey`, which uses
the same exact two-level catalog index as `ResourceByKindAndKey` but projects a
scalar-only summary instead of cloning the full resource document. The same
internal summary projection is also the source used by `GetResources`, so name
and item-icon facts are not reimplemented by this endpoint.

The batch is atomic. If any identity is invalid, the endpoint returns an empty
result and an error prefixed with the zero-based input index, for example:

```text
identity 1: unknown resource key "UNKNOWN" in kind "item"
```

No earlier presentation rows are returned as a partial success.

Unlike the general `GetResources` list, exact identities whose item has
`safety.noDatabase=true` remain reachable. Those resources are reserved for the
feature that owns them; a feature that already has the exact identity must still
be able to present it. This endpoint has no search, browsing, filtering, or
paging surface that could expose them as general catalog choices.

## Icon transport

`iconPath` is catalog metadata, not a host filesystem path and not an HTTP URL.
The desktop application exposes validated embedded item icons through the Wails
AssetServer under:

```text
/catalog-assets/<iconPath>
```

For example:

```text
/catalog-assets/assets/icons/items/melee_armaments/dagger.png
```

The handler serves only paths registered by the validated embedded
`loader.Data`, only below `assets/icons/items/`, and never reads an arbitrary
path from disk. The local OpenAPI explorer returns `iconPath` metadata but does
not expose the Wails asset route.

## HTTP transport

The local explorer exposes the getter as:

```http
POST /api/v1/catalog/resource-presentation-summaries
Content-Type: application/json

{
  "identities": [
    {"kind": "item", "key": "000F4240"},
    {"kind": "item", "key": "8000EA60"},
    {"kind": "item", "key": "000F4240"}
  ]
}
```

`POST` is used because the ordered list is a JSON request body; the endpoint is
still a read-only getter. The decoder rejects malformed JSON and unknown fields.
An omitted `identities` field has the same meaning as an empty list.

## Validation and errors

| Condition | Result |
|---|---|
| `gameCatalog` is `nil` | `game catalog is not loaded` |
| a kind is unknown, empty, recased, or untrimmed | the exact catalog unknown-kind error, prefixed with `identity N:` |
| a key is unknown, empty, recased, or untrimmed inside a known kind | the exact catalog unknown-key error, prefixed with `identity N:` |
| the HTTP JSON is malformed or has an unknown field | HTTP `400` from the transport decoder |

The endpoint reads no save, owns no session or revision, performs no mutation,
and has no PC/PS4 or Safe/Chaos Mode branch.

## Local verification

```bash
go test ./backend/gamecatalog ./backend/endpoints/catalog -run 'ResourcePresentation|ResourceSummar' -count=1
go test ./internal/desktop ./internal/catalogassets -run 'ResourcePresentation|Handler' -count=1
go test ./tools/swagger -run 'ResourcePresentation|OpenAPI' -count=1
```
