# GameCatalog DB Viewer

Small, read-only web browser for `GameCatalog` documents. It is built independently from the Wails application and uses only the Go standard library.

The viewer is read-only: it modifies neither `GameCatalog` data nor save files.

## Run

Use `tools/run_viewer.sh` from any working directory. It builds the
viewer into its own state directory outside the repository, starts it against the
repository catalog data on `127.0.0.1:8787`, and stops only the process it
started itself.

```bash
tools/run_viewer.sh start
tools/run_viewer.sh stop
tools/run_viewer.sh restart
```

Viewer flags are forwarded unchanged, and an explicit `-data` or `-addr`
overrides the script default:

```bash
tools/run_viewer.sh start -addr 127.0.0.1:9000 -data /path/to/gamecatalog/data
```

`restart` without flags reuses the flags of the previous `start`; with flags it
replaces and persists them.

Running the command directly is an optional developer-level alternative:

```bash
go run ./tools/viewer/cmd/gamecatalog-viewer
```

Open `http://127.0.0.1:8787`.

## Build

```bash
go build -o gamecatalog-viewer ./tools/viewer/cmd/gamecatalog-viewer
```

The resulting binary contains only the viewer. Catalog documents and icons remain external files, so rebuilding the viewer is not required after regenerating the data.

Known item icons are stored with the catalog under `assets/icons/items/`, preserving their legacy relative paths. The loader rejects missing icons and paths outside that directory.
Canonical items and variants are independently searchable. Item pages expose aliases, all family-specific facts, and lightweight references to the Regulation rows used during generation. Raw Regulation field values are not stored in the catalog.

The viewer exposes no catalog or save mutation endpoints.
