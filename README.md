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
├── gamecatalog/  # Catalog schema, loader, and data
└── saveengine/   # Save formats, codecs, sessions, and operations
docs/endpoints/   # Endpoint documentation
tools/            # Local developer tools and their lifecycle scripts
├── swagger/      # Local API host, OpenAPI document, and Scalar portal
└── viewer/       # Read-only GameCatalog DB Viewer
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

Build the desktop application. `VERSION` in the `Makefile` stays the single
source of the release version and reaches the binary only through the linker:

```bash
make app-build                          # darwin/arm64 (default)
make app-build PLATFORM=windows/amd64
make app-build PLATFORM=linux/amd64
```

`PLATFORM` only selects the Wails target. Each target still needs its own CGO
toolchain and native GUI dependencies, so a foreign target is normally built on
its own runner: macOS ARM64 on macOS, Windows AMD64 on Windows, and Linux AMD64
on Linux with `libgtk-3-dev` and `libwebkit2gtk-4.1-dev`. The Linux target adds
the `webkit2_41` build tag, which is Linux-only.

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
