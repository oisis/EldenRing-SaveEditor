# SaveForge 2.0 API documentation portal

This is a local documentation portal. It renders documentation that already
exists in the repository; it stores none of its own.

| Section | Source |
| --- | --- |
| Endpoint pages | `docs/endpoints/**/*.md`, read in place through relative paths |
| API Reference | `tools/swagger/openapi.json`, read as a local file |

Nothing under `docs/endpoints` is copied, generated or symlinked into this
directory. `docs/endpoints` stays the only source of endpoint documentation,
and `openapi.json` stays the only machine-readable contract.

## Running the portal

This portal is the only browser interface. The Go process in `tools/swagger`
is not a user interface: it is the local API host that serves the endpoints the
API Reference calls, and it has no page of its own.

Both processes are started together by the helper script, from any working
directory:

```bash
tools/run_swagger.sh start
tools/run_swagger.sh stop
tools/run_swagger.sh restart
```

`start` runs both in the background and prints the portal URLs once they
answer:

```
Scalar Docs: http://localhost:7970/
API Reference: http://localhost:7970/api
```

`stop` stops both; `restart` reuses the arguments of the previous `start`
unless new ones are given. The script resolves the repository root from its own
location and passes an absolute `-data` path, so it needs no particular working
directory, and the API host defaults to `127.0.0.1:8788` — the address
`openapi.json` names. Every argument is forwarded to that command:

```bash
tools/run_swagger.sh start -addr 127.0.0.1:9000 -app-version dev
tools/run_swagger.sh restart          # reuses those arguments
tools/run_swagger.sh restart -addr 127.0.0.1:8788   # replaces them
```

The script only ever stops the processes it started itself. A Swagger or Scalar
process started any other way, or anything else already holding one of the
addresses, is reported and left running.

The equivalent direct invocations run one process each. The API host requires
the repository root as the working directory; the portal requires this one:

```bash
go run ./tools/swagger -addr 127.0.0.1:8788 -data ./backend/gamecatalog/data
```

```bash
cd tools/swagger/docs-portal
npx @scalar/cli@2.0.1 project preview
```

The API host serves the endpoints and the OpenAPI document at
<http://127.0.0.1:8788/openapi.json>; it serves no page. The portal reads
`../openapi.json` from disk, so it renders the API Reference whether or not the
API host runs — the host is needed only to answer a request sent from it.

Validate the configuration without starting a server:

```bash
npx @scalar/cli@2.0.1 project check-config
```

## Sending requests

The API Reference has a request client. It needs no server selection:
`openapi.json` declares exactly one server, `http://127.0.0.1:8788`, so the
client sends there by default. One thing does have to match, because the portal
and the API host are different origins:

**Open the portal as `http://localhost:7970`, not `http://127.0.0.1:7970`.**
Those are different origins to a browser, and the API host permits only the
first one.

The API host answers cross-origin requests only for the exact origin
`http://localhost:7970`, only for `GET`, `POST`, `DELETE` and the `OPTIONS`
preflight, and only for the `Content-Type` request header. It sends no
wildcard `Access-Control-Allow-Origin`, allows no credentials and runs no
proxy. Serving either side on a different port or host breaks the request
client until the corresponding address is updated: the origin in
`portalOrigin` in `tools/swagger/main.go`, the server entry in
`tools/swagger/openapi.json`, or both.

## Scope

This portal documents the endpoint surface. It does not replace the
GameCatalog DB Viewer, which presents per-fact provenance the OpenAPI
document does not carry.
