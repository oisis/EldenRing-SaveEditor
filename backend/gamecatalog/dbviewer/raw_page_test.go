package dbviewer

import (
	"net/http"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
)

func TestRawPageShowsSourceDocument(t *testing.T) {
	response := request(t, testServer(t).Handler(), http.MethodGet, "/items/000F4240/raw")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{
		"Dagger — raw document",
		"items/weapon/000f4240.json",
		"item:000F4240",
		"allowedAffinities",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("raw page does not contain %q", expected)
		}
	}
}

func TestRawPagePreservesRequestedVariantIdentity(t *testing.T) {
	response := request(t, testServer(t).Handler(), http.MethodGet, "/items/000F436C/raw")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{
		"Dagger (quality) — raw document",
		`href="/items/000F436C"`,
		"items/weapon/000f4240.json",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("variant raw page does not contain %q", expected)
		}
	}
}

func TestRawPageReportsUnavailableLazyDocument(t *testing.T) {
	catalogFS := embeddedCatalogMapFS(t)
	data, err := loader.LoadFS(catalogFS)
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	server, err := New(data)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	delete(catalogFS, "items/weapon/000f4240.json")

	response := request(t, server.Handler(), http.MethodGet, "/items/000F4240/raw")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	if !strings.Contains(response.Body.String(), "Raw catalog document is unavailable") {
		t.Fatalf("body = %q", response.Body.String())
	}
}
