package loader_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
)

func TestLoadFSReadsCatalogDocumentsAndRawJSON(t *testing.T) {
	data, err := loader.LoadFS(embeddedCatalogFS(t))
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	if data.Manifest.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", data.Manifest.SchemaVersion)
	}
	if len(data.Documents) != 2 {
		t.Fatalf("documents = %d, want 2", len(data.Documents))
	}
	if data.Documents[0].Path != "items/weapon/000f4240.json" {
		t.Errorf("first document path = %q", data.Documents[0].Path)
	}
	if len(data.Documents[0].RawJSON) == 0 {
		t.Error("first document raw JSON is empty")
	}
	if resources := data.Resources(); len(resources) != 2 || resources[0].Item == nil {
		t.Fatalf("Resources() = %+v", resources)
	}
}
