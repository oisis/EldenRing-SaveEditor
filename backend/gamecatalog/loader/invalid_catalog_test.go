package loader_test

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
)

func TestLoadFSRejectsUnknownCatalogField(t *testing.T) {
	catalogFS := embeddedCatalogFS(t)
	content := string(catalogFS["catalog.json"].Data)
	catalogFS["catalog.json"].Data = []byte(strings.Replace(content, `"documents":`, `"unexpected": true, "documents":`, 1))

	_, err := loader.LoadFS(catalogFS)
	if err == nil || !strings.Contains(err.Error(), `unknown field "unexpected"`) {
		t.Fatalf("LoadFS error = %v, want unknown field", err)
	}
}

func TestLoadFSRejectsDuplicateDocumentPath(t *testing.T) {
	catalogFS := embeddedCatalogFS(t)
	content := string(catalogFS["catalog.json"].Data)
	content = strings.Replace(content, "items/ash_of_war/8000ea60.json", "items/weapon/000f4240.json", 1)
	catalogFS["catalog.json"].Data = []byte(content)

	_, err := loader.LoadFS(catalogFS)
	if err == nil || !strings.Contains(err.Error(), "duplicate path") {
		t.Fatalf("LoadFS error = %v, want duplicate path", err)
	}
}

func TestLoadFSRejectsEscapingDocumentPath(t *testing.T) {
	catalogFS := embeddedCatalogFS(t)
	content := string(catalogFS["catalog.json"].Data)
	content = strings.Replace(content, "items/weapon/000f4240.json", "../000f4240.json", 1)
	catalogFS["catalog.json"].Data = []byte(content)

	_, err := loader.LoadFS(catalogFS)
	if err == nil || !strings.Contains(err.Error(), "relative, slash-separated path") {
		t.Fatalf("LoadFS error = %v, want invalid relative path", err)
	}
}
