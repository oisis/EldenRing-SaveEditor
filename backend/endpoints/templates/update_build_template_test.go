package templates_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/templates"
)

func TestUpdateBuildTemplate_NilStore(t *testing.T) {
	_, err := templates.UpdateBuildTemplate(nil, "tpl-123", templates.UpdateBuildTemplateRequest{
		TemplateRevision: "1",
		Metadata: &templates.UpdateBuildTemplateMetadata{
			Name: "Updated Name",
		},
	})
	if err == nil {
		t.Fatal("expected error on nil store, got nil")
	}
}

func TestUpdateBuildTemplate_Success(t *testing.T) {
	dir := t.TempDir()
	indexJSON := `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-123",
      "name": "Original Name",
      "filename": "tpl-123.json",
      "createdAt": "2026-08-17T12:00:00Z",
      "updatedAt": "2026-08-17T12:00:00Z",
      "revision": 3
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, buildtemplates.IndexFileName), []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}
	payloadJSON := `{
  "schema": "saveforge.build-template",
  "version": 1,
  "createdAt": "2026-08-17T12:00:00Z",
  "metadata": {"name": "Original Name"},
  "sections": {
    "inventory.workspace": {
      "inventoryItems": [{"baseItemID": 100, "quantity": 1, "container": "inventory", "position": 0}],
      "storageItems": []
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "tpl-123.json"), []byte(payloadJSON), 0644); err != nil {
		t.Fatalf("WriteFile payload: %v", err)
	}

	store := buildtemplates.NewStore(dir)

	result, err := templates.UpdateBuildTemplate(store, "tpl-123", templates.UpdateBuildTemplateRequest{
		TemplateRevision: "3",
		Metadata: &templates.UpdateBuildTemplateMetadata{
			Name: "New Name",
		},
	})
	if err != nil {
		t.Fatalf("UpdateBuildTemplate: %v", err)
	}
	if result.TemplateID != "tpl-123" || result.TemplateRevision != "4" {
		t.Errorf("unexpected result: %+v", result)
	}
}
