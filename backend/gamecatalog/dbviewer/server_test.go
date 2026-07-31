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

func TestServerServesOnlyKnownCatalogIconAssets(t *testing.T) {
	handler := testServer(t).Handler()
	iconResponse := request(t, handler, http.MethodGet, "/catalog-assets/icons/items/melee_armaments/dagger.png")
	if iconResponse.Code != http.StatusOK {
		t.Fatalf("known icon status = %d, want 200", iconResponse.Code)
	}
	if contentType := iconResponse.Header().Get("Content-Type"); contentType != "image/png" {
		t.Errorf("known icon Content-Type = %q, want image/png", contentType)
	}
	if iconResponse.Body.Len() == 0 {
		t.Error("known icon response is empty")
	}

	unknownResponse := request(t, handler, http.MethodGet, "/catalog-assets/icons/items/weapon/deadbeef.png")
	if unknownResponse.Code != http.StatusNotFound {
		t.Errorf("unknown icon status = %d, want 404", unknownResponse.Code)
	}
}

func TestServerUsesDetectedWebPMediaTypeForLegacyPNGPath(t *testing.T) {
	catalogFS := embeddedCatalogMapFS(t)
	iconPath := "assets/icons/items/melee_armaments/dagger.png"
	catalogFS[iconPath].Data = []byte{'R', 'I', 'F', 'F', 4, 0, 0, 0, 'W', 'E', 'B', 'P'}
	data, err := loader.LoadFS(catalogFS)
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	server, err := New(data)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	response := request(t, server.Handler(), http.MethodGet, "/catalog-assets/icons/items/melee_armaments/dagger.png")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "image/webp" {
		t.Fatalf("Content-Type = %q, want image/webp", contentType)
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
