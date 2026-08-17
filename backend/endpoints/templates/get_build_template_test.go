package templates_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/templates"
)

func TestGetBuildTemplateDefinition(t *testing.T) {
	def := templates.GetBuildTemplateDefinition
	if def.Name != "GetBuildTemplate" {
		t.Errorf("def.Name = %q, want GetBuildTemplate", def.Name)
	}
	if def.ID != templates.GetBuildTemplateEndpointID {
		t.Errorf("def.ID = %q, want %q", def.ID, templates.GetBuildTemplateEndpointID)
	}
	if def.Kind != contract.Getter {
		t.Errorf("def.Kind = %v, want Getter", def.Kind)
	}
	if len(def.SupportedResourceVariables) != 1 || def.SupportedResourceVariables[0] != "templateID" {
		t.Errorf("def.SupportedResourceVariables = %v, want [templateID]", def.SupportedResourceVariables)
	}
}

func TestGetBuildTemplate_NilStore(t *testing.T) {
	_, err := templates.GetBuildTemplate(nil, "tpl-1")
	if err == nil {
		t.Fatal("expected error for nil store, got nil")
	}
	if err.Error() != "templates store is not available" {
		t.Errorf("error = %q, want %q", err.Error(), "templates store is not available")
	}
}

func TestGetBuildTemplate_EmptyTemplateID(t *testing.T) {
	store := buildtemplates.NewStore(t.TempDir())
	_, err := templates.GetBuildTemplate(store, "")
	if err == nil {
		t.Fatal("expected error for empty templateID, got nil")
	}
	if err.Error() != "templateID must not be empty" {
		t.Errorf("error = %q, want %q", err.Error(), "templateID must not be empty")
	}
}

func TestGetBuildTemplate_Success(t *testing.T) {
	dir := t.TempDir()
	indexJSON := `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-test",
      "name": "Test Template",
      "filename": "test.json",
      "createdAt": "2026-08-17T12:00:00Z",
      "updatedAt": "2026-08-17T12:00:00Z",
      "version": 2
    }
  ]
}`
	payloadJSON := `{
  "schema": "saveforge.build-template",
  "version": 2,
  "createdAt": "2026-08-17T12:00:00Z",
  "metadata": {
    "name": "Test Template"
  },
  "selection": {
    "items": true
  },
  "sections": {
    "items": {
      "entries": [
        {
          "entryID": "entry-1",
          "itemID": 1000,
          "category": "melee_armaments",
          "quantity": 1,
          "location": "inventory"
        }
      ]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, buildtemplates.IndexFileName), []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test.json"), []byte(payloadJSON), 0644); err != nil {
		t.Fatalf("WriteFile payload: %v", err)
	}

	store := buildtemplates.NewStore(dir)
	tpl, err := templates.GetBuildTemplate(store, "tpl-test")
	if err != nil {
		t.Fatalf("GetBuildTemplate failed: %v", err)
	}
	if tpl.Version != 2 {
		t.Errorf("tpl.Version = %d, want 2", tpl.Version)
	}
	if tpl.Sections.Items == nil || len(tpl.Sections.Items.Entries) != 1 {
		t.Fatalf("unexpected tpl items: %+v", tpl.Sections.Items)
	}
}

func TestGetBuildTemplate_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := buildtemplates.NewStore(dir)

	_, err := templates.GetBuildTemplate(store, "tpl-nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent template, got nil")
	}
	if !errors.Is(err, buildtemplates.ErrNotFound) {
		t.Fatalf("expected ErrTemplateNotFound, got: %v", err)
	}
}
