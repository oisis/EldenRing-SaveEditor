package templates_test

import (
	"errors"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/templates"
)

const deleteTestIndex = `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-seven",
      "name": "Seven",
      "filename": "seven.json",
      "createdAt": "2026-08-17T11:00:00Z",
      "updatedAt": "2026-08-17T11:00:00Z",
      "revision": 7
    }
  ]
}`

func TestDeleteBuildTemplateDefinition(t *testing.T) {
	definition := templates.DeleteBuildTemplateDefinition
	if definition.ID != templates.DeleteBuildTemplateEndpointID {
		t.Errorf("definition.ID = %q, want %q", definition.ID, templates.DeleteBuildTemplateEndpointID)
	}
	if definition.Name != "DeleteBuildTemplate" {
		t.Errorf("definition.Name = %q, want %q", definition.Name, "DeleteBuildTemplate")
	}
}

// Every error path returns the zero receipt, so no caller can read a templateID
// out of a failed delete.
func assertZeroDeleteBuildTemplateResult(t *testing.T, result templates.DeleteBuildTemplateResult) {
	t.Helper()
	if result != (templates.DeleteBuildTemplateResult{}) {
		t.Errorf("result = %+v, want the zero DeleteBuildTemplateResult", result)
	}
}

// The endpoint is a thin adapter: on success it delegates to the store and
// echoes the public templateID back, exposing no filename and no counter.
func TestDeleteBuildTemplate_DelegatesAndReturnsReceipt(t *testing.T) {
	store := newTestStoreWithIndex(t, deleteTestIndex)

	result, err := templates.DeleteBuildTemplate(store, "tpl-seven", "7")
	if err != nil {
		t.Fatalf("DeleteBuildTemplate: %v", err)
	}
	if result.TemplateID != "tpl-seven" {
		t.Errorf("result.TemplateID = %q, want %q", result.TemplateID, "tpl-seven")
	}

	entries, err := store.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("library still holds %d entries, want 0", len(entries))
	}
}

func TestDeleteBuildTemplate_NilStore(t *testing.T) {
	result, err := templates.DeleteBuildTemplate(nil, "tpl-seven", "7")
	if err == nil {
		t.Fatal("expected error for nil store, got nil")
	}
	if err.Error() != "templates store is not available" {
		t.Errorf("error = %q, want %q", err.Error(), "templates store is not available")
	}
	assertZeroDeleteBuildTemplateResult(t, result)
}

func TestDeleteBuildTemplate_EmptyTemplateID(t *testing.T) {
	store := newTestStoreWithIndex(t, deleteTestIndex)

	result, err := templates.DeleteBuildTemplate(store, "", "7")
	if err == nil {
		t.Fatal("expected error for empty templateID, got nil")
	}
	if err.Error() != "templateID must not be empty" {
		t.Errorf("error = %q, want %q", err.Error(), "templateID must not be empty")
	}
	assertZeroDeleteBuildTemplateResult(t, result)
}

// The store owns the revision and lookup rules; the endpoint must propagate its
// sentinels unwrapped enough for the transport to classify them.
func TestDeleteBuildTemplate_PropagatesStoreErrors(t *testing.T) {
	store := newTestStoreWithIndex(t, deleteTestIndex)

	result, err := templates.DeleteBuildTemplate(store, "tpl-unknown", "7")
	if !errors.Is(err, buildtemplates.ErrNotFound) {
		t.Fatalf("unknown templateID error = %v, want ErrNotFound", err)
	}
	assertZeroDeleteBuildTemplateResult(t, result)

	result, err = templates.DeleteBuildTemplate(store, "tpl-seven", "6")
	if !errors.Is(err, buildtemplates.ErrStaleRevision) {
		t.Fatalf("stale revision error = %v, want ErrStaleRevision", err)
	}
	assertZeroDeleteBuildTemplateResult(t, result)

	result, err = templates.DeleteBuildTemplate(store, "tpl-seven", "07")
	if err == nil {
		t.Fatal("expected error for a non-canonical templateRevision, got nil")
	}
	assertZeroDeleteBuildTemplateResult(t, result)

	// None of the refusals above touched the library.
	entries, err := store.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("library holds %d entries, want the single entry to survive", len(entries))
	}
}
