package dbviewer

import (
	"net/http"
	"strings"
	"testing"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
)

func TestServerIsReadOnlyAndSetsSecurityHeaders(t *testing.T) {
	handler := testServer(t).Handler()
	getResponse := request(t, handler, http.MethodGet, "/")
	if !strings.Contains(getResponse.Header().Get("Content-Security-Policy"), "default-src 'self'") {
		t.Error("missing restrictive Content-Security-Policy")
	}

	postResponse := request(t, handler, http.MethodPost, "/")
	if postResponse.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST / status = %d, want 405", postResponse.Code)
	}
}

func TestServerServesEmbeddedAssetsAndHealth(t *testing.T) {
	handler := testServer(t).Handler()
	for _, target := range []string{"/assets/styles.css", "/assets/app.js", "/healthz"} {
		response := request(t, handler, http.MethodGet, target)
		if response.Code != http.StatusOK {
			t.Errorf("%s status = %d, want 200", target, response.Code)
		}
	}
}

func TestNewRejectsDuplicateCatalogDocument(t *testing.T) {
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	data.Documents = append(data.Documents, data.Documents[0])

	_, err = New(data)
	if err == nil || !strings.Contains(err.Error(), "duplicate resource ID") {
		t.Fatalf("New error = %v, want duplicate resource rejection", err)
	}
}
