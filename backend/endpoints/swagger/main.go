// Command swagger serves a local, read-only OpenAPI explorer for the currently
// implemented GameCatalog getters. It is a standalone developer tool: the Wails
// application neither imports nor starts it, and every route only calls an
// existing getter and serialises its result.
package main

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/appearance"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/application"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/network"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
)

//go:embed openapi.json docs.html
var assets embed.FS

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	address := flag.String("addr", "127.0.0.1:8788", "local address used by the explorer")
	dataDirectory := flag.String("data", "./backend/gamecatalog/data", "catalog data directory")
	allowExternal := flag.Bool("allow-external-bind", false, "allow binding a non-loopback address")
	// The explorer is a developer tool with no build-time version of its own, so
	// it supplies the application version GetApplicationInfo requires.
	applicationVersion := flag.String("app-version", "dev", "application version reported by /api/v1/application/info")
	flag.Parse()

	if err := validateAddress(*address, *allowExternal); err != nil {
		return err
	}

	gameCatalog, err := loadCatalog(*dataDirectory)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              *address,
		Handler:           newHandler(gameCatalog, *applicationVersion),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("SaveForge catalog getters explorer: http://%s/docs", *address)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve explorer: %w", err)
	}
	return nil
}

func loadCatalog(directory string) (*gamecatalog.Catalog, error) {
	data, err := loader.LoadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("load catalog directory: %w", err)
	}
	// Missing or invalid network parameters stop the explorer here, instead of
	// serving a catalog whose network route could not answer.
	networkPresets, err := gamecatalog.LoadNetworkParams(os.DirFS(directory))
	if err != nil {
		return nil, fmt.Errorf("load network parameters: %w", err)
	}
	// The same rule applies to the appearance presets and their assets.
	appearancePresets, err := gamecatalog.LoadAppearancePresets(os.DirFS(directory))
	if err != nil {
		return nil, fmt.Errorf("load appearance presets: %w", err)
	}
	gameCatalog, err := gamecatalog.NewWithData(gamecatalog.CatalogData{
		Manifest:          data.Manifest,
		Resources:         data.Resources(),
		NetworkPresets:    networkPresets,
		AppearancePresets: appearancePresets,
	})
	if err != nil {
		return nil, fmt.Errorf("build catalog: %w", err)
	}
	return gameCatalog, nil
}

// validateAddress keeps the explorer on loopback unless the operator asks for
// an external bind explicitly. The catalog is development data, but the server
// is unauthenticated, so an accidental LAN bind must not be possible by default.
func validateAddress(address string, allowExternal bool) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid address %q: %w", address, err)
	}
	if allowExternal {
		return nil
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf(
			"address %q is not a loopback address; pass -allow-external-bind to serve it anyway",
			address,
		)
	}
	return nil
}

func newHandler(gameCatalog *gamecatalog.Catalog, applicationVersion string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /openapi.json", func(writer http.ResponseWriter, _ *http.Request) {
		serveAsset(writer, "openapi.json", "application/json")
	})

	mux.HandleFunc("GET /docs", func(writer http.ResponseWriter, _ *http.Request) {
		serveAsset(writer, "docs.html", "text/html; charset=utf-8")
	})

	mux.HandleFunc("GET /api/v1/application/info", func(writer http.ResponseWriter, _ *http.Request) {
		result, err := application.GetApplicationInfo(applicationVersion)
		if err != nil {
			// The version comes from the backend, not from the client, so a
			// rejected version is a server configuration error.
			writeError(writer, http.StatusInternalServerError, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	mux.HandleFunc("GET /api/v1/catalog/info", func(writer http.ResponseWriter, _ *http.Request) {
		result, err := catalog.GetCatalogInfo(gameCatalog)
		if err != nil {
			// GetCatalogInfo takes no input, so a failure is always server side.
			writeError(writer, http.StatusInternalServerError, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	mux.HandleFunc("GET /api/v1/catalog/resource", func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		result, err := catalog.GetResource(gameCatalog, query.Get("kind"), query.Get("key"))
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	mux.HandleFunc("GET /api/v1/catalog/resource-relations", func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		result, err := catalog.GetResourceRelations(
			gameCatalog,
			query.Get("kind"),
			query.Get("key"),
			query.Get("relationType"),
			query.Get("direction"),
		)
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	mux.HandleFunc("GET /api/v1/catalog/resources", func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		page, err := parsePagingValue(query.Get("page"), "page")
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		pageSize, err := parsePagingValue(query.Get("pageSize"), "pageSize")
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		result, err := catalog.GetResources(
			gameCatalog,
			query.Get("resourceType"),
			query.Get("family"),
			query.Get("capability"),
			query.Get("endpointId"),
			query.Get("search"),
			page,
			pageSize,
		)
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	mux.HandleFunc("GET /api/v1/catalog/item-variants", func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		result, err := catalog.GetItemVariants(gameCatalog, query.Get("kind"), query.Get("key"))
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	// The presets live in the catalog data: the route needs the catalog, but not
	// the application version.
	mux.HandleFunc("GET /api/v1/network/presets", func(writer http.ResponseWriter, request *http.Request) {
		result, err := network.GetNetworkPresets(gameCatalog, request.URL.Query().Get("presetID"))
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	// The presets and their assets live in the catalog data, so the route needs
	// the catalog only. It serves preset metadata; the images themselves are not
	// exposed by any route yet.
	mux.HandleFunc("GET /api/v1/appearance/presets", func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		result, err := appearance.GetAppearancePresets(gameCatalog, query.Get("search"), parseTags(query.Get("tags")))
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})

	return mux
}

// parseTags splits the single comma-separated tags parameter into the list the
// getter expects. An absent or empty parameter means "do not filter" and becomes
// nil; the individual tags are never trimmed, so an empty element stays empty
// and the getter rejects it.
func parseTags(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}

// parsePagingValue turns a query string into the integer GetResources expects.
// An absent parameter stays 0, which is the getter's "use the default" value, so
// the HTTP layer never invents a page or a page size of its own. Anything that
// is not an integer is rejected here, because the getter only sees numbers and
// could not report the malformed text.
func parsePagingValue(raw string, name string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer; got %q", name, raw)
	}
	return value, nil
}

func serveAsset(writer http.ResponseWriter, name string, contentType string) {
	content, err := assets.ReadFile(name)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err)
		return
	}
	writer.Header().Set("Content-Type", contentType)
	if _, err := writer.Write(content); err != nil {
		log.Printf("write %s: %v", name, err)
	}
}

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// writeError keeps the message the getter returned, because the getter owns the
// wording of its own validation rules.
// ponytail: no shared EndpointError type yet; endpoints.md defers that model
// until several endpoint contracts exist.
func writeError(writer http.ResponseWriter, status int, err error) {
	writeJSON(writer, status, map[string]string{"error": err.Error()})
}
