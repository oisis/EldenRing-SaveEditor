# GetApplicationInfo

## Overview

`GetApplicationInfo` reports the version of the application, the schema versions
the backend supports, and the capabilities the backend currently declares.

| | |
|---|---|
| EndpointID | `get_application_info` |
| Kind | Getter |
| Domain | `application` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/application/info` of the local OpenAPI explorer (`tools/swagger`). No Wails binding and no permanent CLI command reach it. |
| Implementation source | [../../../backend/endpoints/application/get_application_info.go](../../../backend/endpoints/application/get_application_info.go) |
| Test source | [../../../backend/endpoints/application/get_application_info_test.go](../../../backend/endpoints/application/get_application_info_test.go) |
| Save access | none — the endpoint never opens, reads, or writes a save |
| Mutation | none — the endpoint builds a result and modifies nothing |

## Input

The public contract of this endpoint has no input parameters. There is no
filter, no pagination, and no identifier to resolve.

The Go signature takes the application version:

```go
func GetApplicationInfo(applicationVersion string) (GetApplicationInfoResult, error)
```

That argument is a backend dependency, not a transport parameter:

- `applicationVersion` is supplied by the backend caller, never by a client. No
  HTTP query parameter, request body, or header reaches it.
- The endpoint owns no source of the application version. It does not read the
  `Makefile`, `wails.json`, a generated version file, or an environment
  variable, and it does not create a version package or generator of its own.
- The caller that wires the endpoint into a runtime owns the single source of
  the application version and passes it in.
- The endpoint does not depend on the legacy `internal/application` package and
  does not import it.
- The frontend never passes this parameter and never influences the reported
  version.

## Output

The endpoint returns a typed result:

```go
type SupportedSchema struct {
	Name           string `json:"name"`
	MinimumVersion uint32 `json:"minimumVersion"`
	CurrentVersion uint32 `json:"currentVersion"`
}

type GetApplicationInfoResult struct {
	ApplicationVersion string            `json:"applicationVersion"`
	SupportedSchemas   []SupportedSchema `json:"supportedSchemas"`
	Capabilities       []string          `json:"capabilities"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `applicationVersion` | `string` | Exactly the version the backend caller supplied. It is never trimmed, normalised, or replaced by a fallback. |
| `supportedSchemas` | array of `SupportedSchema` | The schemas the backend can read. Today it holds exactly one entry, for the GameCatalog schema. |
| `capabilities` | array of `string` | The capabilities the backend declares. Today it holds exactly one value, `catalog_read`. |

Each `SupportedSchema` entry describes one schema and its accepted version
range:

| Field | Type | Meaning |
|---|---|---|
| `name` | `string` | Name of the schema. The only value today is `game_catalog`. |
| `minimumVersion` | `uint32` | `schema.MinimumSchemaVersion` — the oldest GameCatalog schema version the backend accepts. |
| `currentVersion` | `uint32` | `schema.CurrentSchemaVersion` — the GameCatalog schema version the backend was built against. |

Both versions are the compile-time constants of
`backend/gamecatalog/schema`. The endpoint never reads a loaded GameCatalog
instance and never reads a catalog manifest, so the reported range describes
what the backend supports, not what any particular catalog data set declares.

`catalog_read` is the only declared capability. The backend does not yet declare
save reading, save writing, or any mutating capability through this endpoint.

## Processing flow

1. The backend caller passes the application version it owns.
2. The getter rejects an empty version.
3. The getter builds the single `game_catalog` schema entry from
   `schema.MinimumSchemaVersion` and `schema.CurrentSchemaVersion`.
4. The getter builds the single `catalog_read` capability.
5. The getter returns the typed result. Both slices are non-nil and are built
   per call, so mutating one result never affects another call.
6. The getter modifies nothing.

## Validation and errors

- An empty `applicationVersion` returns the error
  `application version is required` and an empty result. This is a backend
  wiring error, not a client error, so the HTTP route maps it to `500`.
- A non-empty version is never validated further. The endpoint does not parse,
  interpret, or normalise the version string.

## Command-line verification

`GetApplicationInfo` is exposed over HTTP as `GET /api/v1/application/info` by
the local OpenAPI explorer in `tools/swagger`, a developer tool the
application neither imports nor starts. The explorer takes the version from its
own `-app-version` flag, which defaults to `dev`. That flag belongs to the
explorer only; it is not a source of the application version for the
application itself.

### Run tests

From the repository root:

```bash
go test ./backend/endpoints/application -run '^TestGetApplicationInfo' -count=1 -v
```

### Call the route

Start the explorer:

```bash
go run ./tools/swagger -app-version dev
```

Then call the route:

```bash
curl -s http://127.0.0.1:8788/api/v1/application/info
```

The output reports the version passed to the explorer, the current schema range,
and the single capability:

```json
{
  "applicationVersion": "dev",
  "supportedSchemas": [
    {
      "name": "game_catalog",
      "minimumVersion": 1,
      "currentVersion": 14
    }
  ],
  "capabilities": [
    "catalog_read"
  ]
}
```

The schema versions above are the values of the constants at the time of
writing; they change whenever the GameCatalog schema is versioned up.

No demonstration program or helper script for this endpoint is kept in the
repository.

## Current limitations

- The endpoint is not exposed through Wails.
- The only HTTP route is `GET /api/v1/application/info` of the local OpenAPI
  explorer in `tools/swagger`, a developer tool.
- There is no permanent CLI command for it.
- The getter does not determine the application version. It requires the version
  from its backend caller.
- `supportedSchemas` covers the GameCatalog schema only. No save-format schema is
  reported yet.
- `capabilities` declares `catalog_read` only. Save reading, save writing, and
  mutations are not declared.
