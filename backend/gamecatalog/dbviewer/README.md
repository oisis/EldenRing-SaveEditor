# GameCatalog DB Viewer

Small, read-only web browser for `GameCatalog` documents. It is built independently from the Wails application and uses only the Go standard library.

## Run

Use the current repository data files:

```bash
go run ./backend/gamecatalog/dbviewer/cmd/gamecatalog-viewer
```

Use another catalog directory:

```bash
gamecatalog-viewer -data /path/to/gamecatalog/data
```

Open `http://127.0.0.1:8787`.

## Build

```bash
go build -o gamecatalog-viewer ./backend/gamecatalog/dbviewer/cmd/gamecatalog-viewer
```

The resulting binary contains only the viewer. Catalog documents and icons remain external files, so rebuilding the viewer is not required after regenerating the data.

Known item icons are stored with the catalog under `assets/icons/items/`, preserving their legacy relative paths. The loader rejects missing icons and paths outside that directory.
Canonical items and variants are independently searchable. Item pages expose aliases, all family-specific facts, and lightweight references to the Regulation rows used during generation. Raw Regulation field values are not stored in the catalog.

The viewer exposes no catalog or save mutation endpoints.
