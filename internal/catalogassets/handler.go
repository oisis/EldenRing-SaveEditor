// Package catalogassets exposes validated, embedded GameCatalog item icons to
// the Wails webview. It never reads paths from the host filesystem.
package catalogassets

import (
	"net/http"
	"strconv"
	"strings"
)

const (
	URLPrefix       = "/catalog-assets/"
	itemIconsPrefix = "assets/icons/items/"
)

// Reader is the narrow catalog-data capability the handler needs. loader.Data
// implements it by reading only assets registered during validated catalog
// loading.
type Reader interface {
	ReadAssetWithMediaType(assetPath string) ([]byte, string, bool)
}

// New builds the Wails AssetServer fallback handler for embedded item icons.
func New(reader Reader) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			writer.Header().Set("Allow", "GET, HEAD")
			http.Error(writer, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		assetPath, valid := itemIconPath(request.URL.Path)
		if !valid {
			http.NotFound(writer, request)
			return
		}
		if reader == nil {
			http.NotFound(writer, request)
			return
		}
		content, mediaType, exists := reader.ReadAssetWithMediaType(assetPath)
		if !exists {
			http.NotFound(writer, request)
			return
		}

		// Paths are stable catalog identities rather than content hashes. Require
		// revalidation so an application update can replace an icon at the same
		// path without a year-long stale webview cache.
		writer.Header().Set("Cache-Control", "no-cache")
		writer.Header().Set("Content-Length", strconv.Itoa(len(content)))
		writer.Header().Set("Content-Type", mediaType)
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.WriteHeader(http.StatusOK)
		if request.Method == http.MethodGet {
			_, _ = writer.Write(content)
		}
	})
}

func itemIconPath(requestPath string) (string, bool) {
	if !strings.HasPrefix(requestPath, URLPrefix) {
		return "", false
	}
	assetPath := strings.TrimPrefix(requestPath, URLPrefix)
	if !strings.HasPrefix(assetPath, itemIconsPrefix) ||
		!strings.HasSuffix(assetPath, ".png") ||
		strings.Contains(assetPath, "\\") {
		return "", false
	}
	for _, segment := range strings.Split(assetPath, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", false
		}
	}
	return assetPath, true
}
