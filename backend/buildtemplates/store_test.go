package buildtemplates

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestStore_MissingDirectoryReturnsEmpty(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent_subdir")
	store := NewStore(dir)

	entries, err := store.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates on missing directory returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
	// Verify directory was not created.
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("Store.ListTemplates must not create directory: %v", err)
	}
}

func TestStore_MissingIndexReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	entries, err := store.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates on missing index returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
	// Verify _index.json was not created.
	indexPath := filepath.Join(dir, IndexFileName)
	if _, err := os.Stat(indexPath); !os.IsNotExist(err) {
		t.Fatalf("Store.ListTemplates must not create _index.json: %v", err)
	}
}

func TestStore_ValidIndexParsesAndSortsCorrectly(t *testing.T) {
	dir := t.TempDir()
	indexJSON := `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-2",
      "name": "Mid Date Build",
      "description": "Mid date description",
      "tags": ["pvp", "meta"],
      "filename": "mid-date.json",
      "createdAt": "2026-05-17T10:00:00Z",
      "updatedAt": "2026-05-17T12:00:00Z",
      "inventoryItems": 5,
      "storageItems": 2,
      "warnings": 0,
      "version": 2,
      "selectedSections": ["inventory.workspace", "equipment"]
    },
    {
      "id": "tpl-1",
      "name": "Oldest Build",
      "description": "Oldest description",
      "tags": ["pve"],
      "filename": "oldest.json",
      "createdAt": "2026-05-17T09:00:00Z",
      "updatedAt": "2026-05-17T10:00:00Z",
      "inventoryItems": 10,
      "storageItems": 0,
      "warnings": 1,
      "version": 1,
      "selectedSections": ["inventory.workspace"]
    },
    {
      "id": "tpl-4",
      "name": "Tie Break B",
      "description": "Tie description B",
      "tags": ["bleed"],
      "filename": "tie-b.json",
      "createdAt": "2026-05-17T13:00:00Z",
      "updatedAt": "2026-05-17T14:00:00Z",
      "inventoryItems": 3,
      "storageItems": 1,
      "warnings": 0,
      "version": 2,
      "selectedSections": ["items"]
    },
    {
      "id": "tpl-3",
      "name": "Tie Break A",
      "description": "Tie description A",
      "tags": ["bleed"],
      "filename": "tie-a.json",
      "createdAt": "2026-05-17T13:00:00Z",
      "updatedAt": "2026-05-17T14:00:00Z",
      "inventoryItems": 3,
      "storageItems": 1,
      "warnings": 0,
      "version": 2,
      "selectedSections": ["items"]
    }
  ]
}`
	indexPath := filepath.Join(dir, IndexFileName)
	if err := os.WriteFile(indexPath, []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store := NewStore(dir)
	entries, err := store.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	// Expected order:
	// 1. tpl-3 (updatedAt: 14:00:00, id: tpl-3 tie-break before tpl-4)
	// 2. tpl-4 (updatedAt: 14:00:00, id: tpl-4)
	// 3. tpl-2 (updatedAt: 12:00:00)
	// 4. tpl-1 (updatedAt: 10:00:00)
	expectedIDs := []string{"tpl-3", "tpl-4", "tpl-2", "tpl-1"}
	for i, wantID := range expectedIDs {
		if entries[i].TemplateID != wantID {
			t.Errorf("entry[%d].TemplateID = %q, want %q", i, entries[i].TemplateID, wantID)
		}
	}

	// Verify field mapping for entry tpl-2
	e2 := entries[2]
	if e2.Name != "Mid Date Build" || e2.Description != "Mid date description" ||
		!reflect.DeepEqual(e2.Tags, []string{"pvp", "meta"}) ||
		e2.CreatedAt != "2026-05-17T10:00:00Z" || e2.UpdatedAt != "2026-05-17T12:00:00Z" ||
		e2.SchemaVersion != 2 ||
		!reflect.DeepEqual(e2.SelectedSections, []string{"inventory.workspace", "equipment"}) ||
		e2.InventoryItems != 5 || e2.StorageItems != 2 || e2.Warnings != 0 {
		t.Errorf("unexpected mapped entry for tpl-2: %+v", e2)
	}
}

func TestStore_MalformedJSONFailsClosed(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, IndexFileName)
	if err := os.WriteFile(indexPath, []byte(`{invalid json`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store := NewStore(dir)
	_, err := store.ListTemplates()
	if err == nil {
		t.Fatal("expected error on malformed JSON, got nil")
	}
}

func TestStore_UnsupportedVersionFailsClosed(t *testing.T) {
	for _, ver := range []int{0, 2, 99} {
		dir := t.TempDir()
		indexPath := filepath.Join(dir, IndexFileName)
		content := fmt.Sprintf(`{"version": %d, "entries": []}`, ver)
		if err := os.WriteFile(indexPath, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		store := NewStore(dir)
		_, err := store.ListTemplates()
		if err == nil {
			t.Fatalf("expected error for unsupported version %d, got nil", ver)
		}
		expectedErrMsg := fmt.Sprintf("unsupported index version %d; expected %d", ver, IndexVersion)
		if err.Error() != expectedErrMsg {
			t.Fatalf("error = %q, want %q", err.Error(), expectedErrMsg)
		}
	}
}

func TestStore_NeverReadsTemplatePayloadFiles(t *testing.T) {
	dir := t.TempDir()
	// Write valid index pointing to a completely broken / unparseable payload file.
	indexJSON := `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-1",
      "name": "Test Build",
      "filename": "corrupted-payload.json",
      "createdAt": "2026-05-17T10:00:00Z",
      "updatedAt": "2026-05-17T10:00:00Z",
      "inventoryItems": 1
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, IndexFileName), []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}
	// Write invalid payload file. If ListTemplates tried to open and parse it, it would fail.
	if err := os.WriteFile(filepath.Join(dir, "corrupted-payload.json"), []byte(`THIS IS NOT JSON AND WOULD FAIL`), 0644); err != nil {
		t.Fatalf("WriteFile payload: %v", err)
	}

	store := NewStore(dir)
	entries, err := store.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates failed: %v (it should not read template payload files)", err)
	}
	if len(entries) != 1 || entries[0].TemplateID != "tpl-1" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestStore_ReadOnlyGuaranteesNoMutation(t *testing.T) {
	dir := t.TempDir()
	indexJSON := `{"version": 1, "entries": []}`
	indexPath := filepath.Join(dir, IndexFileName)
	if err := os.WriteFile(indexPath, []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	infoBefore, err := os.Stat(indexPath)
	if err != nil {
		t.Fatalf("Stat before: %v", err)
	}

	store := NewStore(dir)
	time.Sleep(10 * time.Millisecond)
	if _, err := store.ListTemplates(); err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}

	infoAfter, err := os.Stat(indexPath)
	if err != nil {
		t.Fatalf("Stat after: %v", err)
	}

	if infoBefore.ModTime() != infoAfter.ModTime() {
		t.Fatalf("modtime changed: before %v, after %v", infoBefore.ModTime(), infoAfter.ModTime())
	}
}
