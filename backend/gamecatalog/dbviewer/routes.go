package dbviewer

import (
	"io/fs"
	"net/http"
)

func (server *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetFiles()))))
	mux.HandleFunc("GET /catalog-assets/{assetPath...}", server.iconAssetHandler)
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /items/{gameID}/raw", server.rawItemHandler)
	mux.HandleFunc("GET /items/{gameID}", server.itemHandler)
	mux.HandleFunc("GET /{$}", server.catalogHandler)
	return securityHeaders(mux)
}

func assetFiles() fs.FS {
	assets, err := fs.Sub(webFiles, "web/assets")
	if err != nil {
		panic(err)
	}
	return assets
}

func healthHandler(response http.ResponseWriter, _ *http.Request) {
	response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write([]byte("ok\n"))
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; base-uri 'none'; frame-ancestors 'none'")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(response, request)
	})
}
