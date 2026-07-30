package dbviewer

import (
	"net/http"
	"strings"
	"testing"
)

func TestCatalogPageListsBothDocuments(t *testing.T) {
	response := request(t, testServer(t).Handler(), http.MethodGet, "/")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{
		"Dagger",
		"Ash of War: Determination",
		"items/weapon/000f4240.json",
		"items/ash_of_war/8000ea60.json",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("catalog page does not contain %q", expected)
		}
	}
}

func TestCatalogPageFiltersByFamily(t *testing.T) {
	response := request(t, testServer(t).Handler(), http.MethodGet, "/?family=ash_of_war")
	body := response.Body.String()
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if !strings.Contains(body, "Ash of War: Determination") {
		t.Error("filtered catalog does not contain Determination")
	}
	if strings.Contains(body, ">Dagger</a>") {
		t.Error("filtered catalog unexpectedly contains Dagger row")
	}
}

func TestCatalogPageSearchesHexadecimalGameID(t *testing.T) {
	response := request(t, testServer(t).Handler(), http.MethodGet, "/?q=0x000F4240")
	body := response.Body.String()
	if !strings.Contains(body, ">Dagger</a>") {
		t.Error("hexadecimal search did not find Dagger")
	}
	if strings.Contains(body, ">Ash of War: Determination</a>") {
		t.Error("hexadecimal search unexpectedly found Determination")
	}
}
