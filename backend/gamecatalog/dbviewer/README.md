# GameCatalog DB Viewer

Small, read-only web browser for `GameCatalog` documents. It is built independently from the Wails application and uses only the Go standard library.

## Run

Use the catalog files embedded at build time:

```bash
go run ./backend/gamecatalog/dbviewer/cmd/gamecatalog-viewer
```

Read the current repository data files without rebuilding:

```bash
go run ./backend/gamecatalog/dbviewer/cmd/gamecatalog-viewer -data ./backend/gamecatalog/data
```

Open `http://127.0.0.1:8787`.

## Build

```bash
go build -o gamecatalog-viewer ./backend/gamecatalog/dbviewer/cmd/gamecatalog-viewer
```

The resulting binary contains the HTML, CSS, JavaScript, and the default catalog snapshot. Pass `-data` to inspect another directory that follows the same file contract.

The viewer exposes no catalog or save mutation endpoints.
