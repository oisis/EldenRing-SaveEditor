package dbviewer

import (
	"net/http"
	"net/http/httptest"
	"testing"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	server, err := New(data)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return server
}

func request(t *testing.T, handler http.Handler, method string, target string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(method, target, nil))
	return response
}
