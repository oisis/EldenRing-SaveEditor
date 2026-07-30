package loader_test

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
)

func TestLoadFSRejectsUnknownDocumentField(t *testing.T) {
	catalogFS := embeddedCatalogFS(t)
	path := "items/weapon/000f4240.json"
	content := string(catalogFS[path].Data)
	catalogFS[path].Data = []byte(strings.Replace(content, "{", `{"unexpected": true,`, 1))

	_, err := loader.LoadFS(catalogFS)
	if err == nil || !strings.Contains(err.Error(), `unknown field "unexpected"`) {
		t.Fatalf("LoadFS error = %v, want unknown document field", err)
	}
}

func TestLoadFSRejectsMissingDocument(t *testing.T) {
	catalogFS := embeddedCatalogFS(t)
	delete(catalogFS, "items/weapon/000f4240.json")

	_, err := loader.LoadFS(catalogFS)
	if err == nil || !strings.Contains(err.Error(), "read document") {
		t.Fatalf("LoadFS error = %v, want missing document", err)
	}
}

func TestLoadFSRejectsTrailingDocumentJSON(t *testing.T) {
	catalogFS := embeddedCatalogFS(t)
	path := "items/weapon/000f4240.json"
	catalogFS[path].Data = append(catalogFS[path].Data, []byte("\n{}")...)

	_, err := loader.LoadFS(catalogFS)
	if err == nil || !strings.Contains(err.Error(), "multiple JSON values") {
		t.Fatalf("LoadFS error = %v, want trailing JSON rejection", err)
	}
}
