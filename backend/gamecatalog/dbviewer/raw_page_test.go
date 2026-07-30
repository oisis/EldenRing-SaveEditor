package dbviewer

import (
	"net/http"
	"strings"
	"testing"
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
