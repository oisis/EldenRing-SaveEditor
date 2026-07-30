package dbviewer

import (
	"html"
	"net/http"
	"strings"
	"testing"
)

func TestItemPageShowsHumanReadableDataAndRelations(t *testing.T) {
	response := request(t, testServer(t).Handler(), http.MethodGet, "/items/000F4240")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := html.UnescapeString(response.Body.String())
	for _, expected := range []string{
		"Maximum level: +25",
		"Allowed: standard, heavy, keen",
		"Physical attack",
		"Ash of War: Determination",
		"regulation.bin/csv/EquipParamWeapon.csv",
		`href="/items/000F4240/raw"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("item page does not contain %q", expected)
		}
	}
}

func TestItemPageResolvesVariantToCanonicalDocument(t *testing.T) {
	response := request(t, testServer(t).Handler(), http.MethodGet, "/items/000F436C")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), "items/weapon/000f4240.json") {
		t.Error("variant did not resolve to canonical Dagger document")
	}
}

func TestItemPageReturnsNotFoundForInvalidOrUnknownID(t *testing.T) {
	handler := testServer(t).Handler()
	for _, target := range []string{"/items/not-a-number", "/items/DEADBEEF"} {
		response := request(t, handler, http.MethodGet, target)
		if response.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", target, response.Code)
		}
	}
}
