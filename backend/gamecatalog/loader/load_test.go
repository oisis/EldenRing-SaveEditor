package loader_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestLoadFSReadsCatalogDocumentsAndRawJSON(t *testing.T) {
	data, err := loader.LoadFS(embeddedCatalogFS(t))
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	if data.Manifest.SchemaVersion != schema.CurrentSchemaVersion {
		t.Errorf(
			"SchemaVersion = %d, want %d",
			data.Manifest.SchemaVersion,
			schema.CurrentSchemaVersion,
		)
	}
	if data.Manifest.DataVersion == "" {
		t.Error("DataVersion is empty")
	}
	if len(data.Documents) != 2 {
		t.Fatalf("documents = %d, want 2", len(data.Documents))
	}
	if data.Documents[0].Path != "items/weapon/000f4240.json" {
		t.Errorf("first document path = %q", data.Documents[0].Path)
	}
	rawJSON, exists := data.ReadDocument(data.Documents[0].Path)
	if !exists || len(rawJSON) == 0 {
		t.Fatal("first raw document was not read on demand")
	}
	originalDocumentByte := rawJSON[0]
	rawJSON[0] ^= 0xFF
	reloadedDocument, exists := data.ReadDocument(data.Documents[0].Path)
	if !exists || reloadedDocument[0] != originalDocumentByte {
		t.Fatal("ReadDocument returned mutable catalog storage")
	}
	if _, exists := data.ReadDocument("items/weapon/not-in-catalog.json"); exists {
		t.Fatal("ReadDocument exposed a path outside the catalog manifest")
	}
	if resources := data.Resources(); len(resources) != 2 || resources[0].Item == nil {
		t.Fatalf("Resources() = %+v", resources)
	}
	icon, exists := data.ReadAsset("assets/icons/items/melee_armaments/dagger.png")
	if !exists || len(icon) == 0 {
		t.Fatal("Dagger icon asset was not loaded")
	}
	originalFirstByte := icon[0]
	icon[0] ^= 0xFF
	reloadedIcon, exists := data.ReadAsset("assets/icons/items/melee_armaments/dagger.png")
	if !exists || reloadedIcon[0] != originalFirstByte {
		t.Fatal("ReadAsset returned mutable catalog storage")
	}
	_, mediaType, exists := data.ReadAssetWithMediaType("assets/icons/items/melee_armaments/dagger.png")
	if !exists || mediaType != "image/png" {
		t.Fatalf("Dagger icon media type = %q, %t", mediaType, exists)
	}
}
