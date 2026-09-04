package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
)

const importValidDocument = `{
  "schema": "saveforge.build-template",
  "version": 2,
  "createdAt": "2026-01-01T00:00:00Z",
  "metadata": {"name": "Imported build"},
  "selection": {"stats": true},
  "sections": {"stats": {"vigor": 40, "mind": 20, "endurance": 20, "strength": 40, "dexterity": 12, "intelligence": 9, "faith": 9, "arcane": 7}}
}`

// TestImportBuildTemplateStoresAValidDocumentAndRefusesTheRest covers the whole
// import contract in one flow: a valid document becomes a library entry, and
// every way the source can be wrong is refused without storing anything.
func TestImportBuildTemplateStoresAValidDocumentAndRefusesTheRest(t *testing.T) {
	directory := t.TempDir()
	store := buildtemplates.NewStore(directory)
	source := filepath.Join(t.TempDir(), "template.json")
	if err := os.WriteFile(source, []byte(importValidDocument), 0o600); err != nil {
		t.Fatalf("write the document: %v", err)
	}

	result, err := ImportBuildTemplate(store, source)
	if err != nil {
		t.Fatalf("ImportBuildTemplate: %v", err)
	}
	if result.TemplateID == "" || result.TemplateRevision == "" {
		t.Fatalf("result = %+v", result)
	}
	stored, err := store.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(stored) != 1 || stored[0].Name != "Imported build" {
		t.Fatalf("library = %+v, want the imported template", stored)
	}

	// A document that is not a Build Template is refused by the library's own
	// validation, not by a weaker check of the import's own.
	notATemplate := filepath.Join(t.TempDir(), "other.json")
	if err := os.WriteFile(notATemplate, []byte(`{"schema":"something-else"}`), 0o600); err != nil {
		t.Fatalf("write the document: %v", err)
	}
	if _, err := ImportBuildTemplate(store, notATemplate); err == nil {
		t.Fatal("ImportBuildTemplate accepted a document that is not a Build Template")
	}
	if _, err := ImportBuildTemplate(store, ""); err == nil {
		t.Fatal("ImportBuildTemplate accepted an empty source")
	}
	if _, err := ImportBuildTemplate(store, filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("ImportBuildTemplate accepted a source that does not exist")
	}
	if _, err := ImportBuildTemplate(nil, source); err == nil {
		t.Fatal("ImportBuildTemplate accepted a missing store")
	}

	// None of the refusals added anything to the library.
	after, err := store.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(after) != 1 {
		t.Fatalf("library = %+v, want only the one valid import", after)
	}
}

// TestImportBuildTemplateRefusesAnOversizedDocument keeps the read bounded, so
// a mistakenly picked large file is never loaded whole.
func TestImportBuildTemplateRefusesAnOversizedDocument(t *testing.T) {
	store := buildtemplates.NewStore(t.TempDir())
	source := filepath.Join(t.TempDir(), "big.json")
	if err := os.WriteFile(source, make([]byte, importSizeLimit+1), 0o600); err != nil {
		t.Fatalf("write the document: %v", err)
	}
	if _, err := ImportBuildTemplate(store, source); err == nil {
		t.Fatal("ImportBuildTemplate accepted a document past the size bound")
	}
}
