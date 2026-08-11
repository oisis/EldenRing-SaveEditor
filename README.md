# Elden Ring SaveForge 2.0

SaveForge 2.0 is a greenfield rewrite of the Elden Ring save editor. It is a
breaking change and does not preserve the architecture or public API of the
1.x application.

The project is currently a backend foundation under active development. There
is no installable desktop application or release pipeline for 2.0 yet.

## Current scope

- `GameCatalog` is the read-only source of truth for game resources, their
  capabilities, relations, provenance, and bundled assets.
- `SaveEngine` owns save decoding and the currently implemented read paths for
  PC and PS4 saves.
- Public backend endpoints expose the implemented catalog and save operations
  through explicit getter and setter contracts.
- Scalar Docs presents the OpenAPI reference and endpoint documentation.
- GameCatalog DB Viewer provides read-only inspection of the catalog artifact.

The committed catalog under `backend/gamecatalog/data` is self-contained at
runtime. Historical source paths stored in its manifest are provenance
metadata, not filesystem dependencies.

## Project layout

```text
backend/
├── endpoints/    # Public SaveForge 2.0 endpoint contracts
├── gamecatalog/  # Catalog schema, loader, data, and read-only viewer
├── saveengine/   # Save formats, codecs, sessions, and operations
└── swagger/      # Local API host, OpenAPI document, and Scalar portal
docs/endpoints/   # Endpoint documentation
scripts/          # Local Scalar and GameCatalog Viewer lifecycle scripts
```

## Development

Requirements:

- Go 1.25 or newer;
- Node.js with `npx` to run the Scalar documentation portal.

Run the active SaveForge 2.0 checks:

```bash
make test
make test-race
make vet
```

Open Scalar Docs and its local API host:

```bash
make swagger-start
# http://localhost:7970/

make swagger-stop
```

Open the read-only GameCatalog DB Viewer:

```bash
make viewer-start
# http://127.0.0.1:8787/

make viewer-stop
```

Both lifecycle scripts resolve the repository from their own location, so they
can also be invoked directly from any working directory.

## Documentation

- [Endpoint documentation](docs/endpoints/README.md)
- [SaveForge 2.0 endpoint status](docs/endpoints/SF-2.0.md)
- [SL2 binary format specification](docs/sl2-binary-format-spec.md)

SaveForge 1.6.8 is retained in Git history and the `v1.6.8` tag. It is not the
architecture or implementation baseline for 2.0.

## License

See [LICENSE](LICENSE) and
[THIRD-PARTY-NOTICES.md](docs/THIRD-PARTY-NOTICES.md). The final 2.0 code and
data attribution set will be audited after the migration boundary is complete.
