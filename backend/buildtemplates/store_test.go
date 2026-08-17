package buildtemplates

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestStore_GetTemplate_Success(t *testing.T) {
	dir := t.TempDir()
	indexJSON := `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-v1",
      "name": "V1 Loadout",
      "filename": "v1..json",
      "createdAt": "2026-05-17T10:00:00Z",
      "updatedAt": "2026-05-17T10:00:00Z",
      "version": 1
    },
    {
      "id": "tpl-v2",
      "name": "V2 Loadout",
      "filename": "v2.json",
      "createdAt": "2026-08-17T10:00:00Z",
      "updatedAt": "2026-08-17T10:00:00Z",
      "version": 2
    }
  ]
}`
	payloadV1 := `{
  "schema": "saveforge.build-template",
  "version": 1,
  "createdAt": "2026-05-17T10:00:00Z",
  "metadata": {
    "name": "V1 Loadout"
  },
  "sections": {
    "inventory.workspace": {
      "inventoryItems": [
        {"baseItemID": 100, "quantity": 1, "container": "inventory", "position": 0}
      ],
      "storageItems": []
    }
  }
}`
	payloadV2 := `{
  "schema": "saveforge.build-template",
  "version": 2,
  "createdAt": "2026-08-17T10:00:00Z",
  "metadata": {
    "name": "V2 Loadout"
  },
  "selection": {
    "items": true
  },
  "sections": {
    "items": {
      "entries": [
        {"entryID": "e1", "itemID": 100, "category": "melee_armaments", "quantity": 1, "location": "inventory"}
      ]
    }
  }
}`

	if err := os.WriteFile(filepath.Join(dir, IndexFileName), []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "v1..json"), []byte(payloadV1), 0644); err != nil {
		t.Fatalf("WriteFile v1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "v2.json"), []byte(payloadV2), 0644); err != nil {
		t.Fatalf("WriteFile v2: %v", err)
	}

	store := NewStore(dir)

	// Retrieve v1
	tpl1, revision1, err := store.GetTemplate("tpl-v1")
	if err != nil {
		t.Fatalf("GetTemplate(tpl-v1) failed: %v", err)
	}
	if tpl1.Version != 1 || tpl1.Sections.InventoryWorkspace == nil {
		t.Fatalf("unexpected tpl1: %+v", tpl1)
	}
	// Legacy index entries carry no revision field.
	if revision1 != "0" {
		t.Errorf("GetTemplate(tpl-v1) revision = %q, want \"0\"", revision1)
	}

	// Retrieve v2
	tpl2, revision2, err := store.GetTemplate("tpl-v2")
	if err != nil {
		t.Fatalf("GetTemplate(tpl-v2) failed: %v", err)
	}
	if tpl2.Version != 2 || tpl2.Sections.Items == nil {
		t.Fatalf("unexpected tpl2: %+v", tpl2)
	}
	if revision2 != "0" {
		t.Errorf("GetTemplate(tpl-v2) revision = %q, want \"0\"", revision2)
	}
}

func TestStore_GetTemplate_NotFoundErrors(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	// Missing index -> ErrNotFound
	_, _, err := store.GetTemplate("tpl-1")
	if err == nil || !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing index, got: %v", err)
	}

	indexJSON := `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-1",
      "name": "Missing File",
      "filename": "does-not-exist.json",
      "createdAt": "2026-05-17T10:00:00Z",
      "updatedAt": "2026-05-17T10:00:00Z"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, IndexFileName), []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Unknown ID -> ErrNotFound
	_, _, err = store.GetTemplate("tpl-unknown")
	if err == nil || !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown ID, got: %v", err)
	}

	// Missing target file -> ErrNotFound
	_, _, err = store.GetTemplate("tpl-1")
	if err == nil || !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing target payload, got: %v", err)
	}
}

func TestStore_GetTemplate_FailClosedIndexAndPayload(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	// Empty templateID
	_, _, err := store.GetTemplate("")
	if err == nil {
		t.Fatal("expected error for empty templateID, got nil")
	}

	// Duplicate template ID in index
	dupIndex := `{
  "version": 1,
  "entries": [
    {"id": "tpl-dup", "name": "A", "filename": "a.json"},
    {"id": "tpl-dup", "name": "B", "filename": "b.json"}
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, IndexFileName), []byte(dupIndex), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, _, err = store.GetTemplate("tpl-dup")
	if err == nil || !strings.Contains(err.Error(), "duplicate template ID") {
		t.Fatalf("expected duplicate ID error, got: %v", err)
	}

	// Empty filename in index
	emptyFilenameIndex := `{
  "version": 1,
  "entries": [
    {"id": "tpl-empty", "name": "A", "filename": ""}
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, IndexFileName), []byte(emptyFilenameIndex), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, _, err = store.GetTemplate("tpl-empty")
	if err == nil || !strings.Contains(err.Error(), "empty filename") {
		t.Fatalf("expected empty filename error, got: %v", err)
	}

	// Path traversal filename in index
	traversalIndex := `{
  "version": 1,
  "entries": [
    {"id": "tpl-trav", "name": "A", "filename": "../outside.json"}
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, IndexFileName), []byte(traversalIndex), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, _, err = store.GetTemplate("tpl-trav")
	if err == nil || !strings.Contains(err.Error(), "invalid filename") {
		t.Fatalf("expected invalid filename error, got: %v", err)
	}

	// Version mismatch between index and payload
	mismatchIndex := `{
  "version": 1,
  "entries": [
    {"id": "tpl-mismatch", "name": "V2 in index", "filename": "mismatch.json", "version": 2}
  ]
}`
	mismatchPayload := `{
  "schema": "saveforge.build-template",
  "version": 1,
  "createdAt": "2026-05-17T10:00:00Z",
  "metadata": {"name": "V2 in index"},
  "sections": {
    "inventory.workspace": {
      "inventoryItems": [{"baseItemID": 1, "quantity": 1, "container": "inventory", "position": 0}],
      "storageItems": []
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, IndexFileName), []byte(mismatchIndex), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mismatch.json"), []byte(mismatchPayload), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, _, err = store.GetTemplate("tpl-mismatch")
	if err == nil || !strings.Contains(err.Error(), "version mismatch") {
		t.Fatalf("expected version mismatch error, got: %v", err)
	}

	// Name mismatch between index and payload
	nameMismatchIndex := `{
  "version": 1,
  "entries": [
    {"id": "tpl-name-mismatch", "name": "Index Name", "filename": "name-mismatch.json", "version": 1}
  ]
}`
	nameMismatchPayload := `{
  "schema": "saveforge.build-template",
  "version": 1,
  "createdAt": "2026-05-17T10:00:00Z",
  "metadata": {"name": "Payload Name"},
  "sections": {
    "inventory.workspace": {
      "inventoryItems": [{"baseItemID": 1, "quantity": 1, "container": "inventory", "position": 0}],
      "storageItems": []
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, IndexFileName), []byte(nameMismatchIndex), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "name-mismatch.json"), []byte(nameMismatchPayload), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, _, err = store.GetTemplate("tpl-name-mismatch")
	if err == nil || !strings.Contains(err.Error(), "metadata mismatch") {
		t.Fatalf("expected metadata mismatch error, got: %v", err)
	}
}

func TestStore_GetTemplate_SymlinkEscapeRejected(t *testing.T) {
	tempParent := t.TempDir()
	storeDir := filepath.Join(tempParent, "store")
	outsideDir := filepath.Join(tempParent, "outside")
	if err := os.Mkdir(storeDir, 0755); err != nil {
		t.Fatalf("Mkdir store: %v", err)
	}
	if err := os.Mkdir(outsideDir, 0755); err != nil {
		t.Fatalf("Mkdir outside: %v", err)
	}

	outsideSecret := filepath.Join(outsideDir, "secret-template.json")
	validPayload := `{
  "schema": "saveforge.build-template",
  "version": 1,
  "createdAt": "2026-05-17T10:00:00Z",
  "sections": {
    "inventory.workspace": {
      "inventoryItems": [{"baseItemID": 1, "quantity": 1, "container": "inventory", "position": 0}],
      "storageItems": []
    }
  }
}`
	if err := os.WriteFile(outsideSecret, []byte(validPayload), 0644); err != nil {
		t.Fatalf("WriteFile outside: %v", err)
	}

	symlinkPath := filepath.Join(storeDir, "escaped.json")
	if err := os.Symlink(outsideSecret, symlinkPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	indexJSON := `{
  "version": 1,
  "entries": [
    {"id": "tpl-escape", "name": "Escape", "filename": "escaped.json", "version": 1}
  ]
}`
	if err := os.WriteFile(filepath.Join(storeDir, IndexFileName), []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}

	store := NewStore(storeDir)
	_, _, err := store.GetTemplate("tpl-escape")
	if err == nil || !strings.Contains(err.Error(), "escapes store directory") {
		t.Fatalf("expected symlink escape error, got: %v", err)
	}
}

// The revision counter is a persistent per-template value in _index.json that
// both getters must surface as the same canonical decimal templateRevision
// token. An entry written before the counter existed reports "0".
func TestStore_TemplateRevisionToken(t *testing.T) {
	dir := t.TempDir()
	indexJSON := fmt.Sprintf(`{
  "version": 1,
  "entries": [
    {
      "id": "tpl-legacy",
      "name": "Legacy",
      "filename": "legacy.json",
      "createdAt": "2026-08-17T10:00:00Z",
      "updatedAt": "2026-08-17T10:00:00Z"
    },
    {
      "id": "tpl-seven",
      "name": "Seven",
      "filename": "seven.json",
      "createdAt": "2026-08-17T11:00:00Z",
      "updatedAt": "2026-08-17T11:00:00Z",
      "revision": 7
    },
    {
      "id": "tpl-max",
      "name": "Max",
      "filename": "max.json",
      "createdAt": "2026-08-17T12:00:00Z",
      "updatedAt": "2026-08-17T12:00:00Z",
      "revision": %d
    }
  ]
}`, uint64(math.MaxUint64))
	if err := os.WriteFile(filepath.Join(dir, IndexFileName), []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}

	payload := `{
  "schema": "saveforge.build-template",
  "version": 1,
  "createdAt": "2026-08-17T10:00:00Z",
  "metadata": {"name": %q},
  "sections": {
    "inventory.workspace": {
      "inventoryItems": [{"baseItemID": 1, "quantity": 1, "container": "inventory", "position": 0}],
      "storageItems": []
    }
  }
}`
	for filename, name := range map[string]string{
		"legacy.json": "Legacy",
		"seven.json":  "Seven",
		"max.json":    "Max",
	} {
		if err := os.WriteFile(filepath.Join(dir, filename), []byte(fmt.Sprintf(payload, name)), 0644); err != nil {
			t.Fatalf("WriteFile %s: %v", filename, err)
		}
	}

	store := NewStore(dir)
	entries, err := store.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	listed := make(map[string]string, len(entries))
	for _, entry := range entries {
		listed[entry.TemplateID] = entry.TemplateRevision
	}

	for templateID, want := range map[string]string{
		"tpl-legacy": "0",
		"tpl-seven":  "7",
		"tpl-max":    "18446744073709551615",
	} {
		if listed[templateID] != want {
			t.Errorf("ListTemplates %s templateRevision = %q, want %q", templateID, listed[templateID], want)
		}
		_, revision, err := store.GetTemplate(templateID)
		if err != nil {
			t.Fatalf("GetTemplate(%s): %v", templateID, err)
		}
		if revision != want {
			t.Errorf("GetTemplate %s templateRevision = %q, want %q", templateID, revision, want)
		}
		if revision != listed[templateID] {
			t.Errorf("%s templateRevision differs: GetTemplate %q, ListTemplates %q",
				templateID, revision, listed[templateID])
		}
	}
}

// newDeleteFixture writes a three-entry library: a legacy entry with no
// revision field, an entry at revision 7, and an entry at math.MaxUint64.
// Payload contents are opaque here, because DeleteTemplate never decodes them.
func newDeleteFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	indexJSON := fmt.Sprintf(`{
  "version": 1,
  "entries": [
    {
      "id": "tpl-legacy",
      "name": "Legacy",
      "filename": "legacy.json",
      "createdAt": "2026-08-17T10:00:00Z",
      "updatedAt": "2026-08-17T10:00:00Z"
    },
    {
      "id": "tpl-seven",
      "name": "Seven",
      "filename": "seven.json",
      "createdAt": "2026-08-17T11:00:00Z",
      "updatedAt": "2026-08-17T11:00:00Z",
      "revision": 7
    },
    {
      "id": "tpl-max",
      "name": "Max",
      "filename": "max.json",
      "createdAt": "2026-08-17T12:00:00Z",
      "updatedAt": "2026-08-17T12:00:00Z",
      "revision": %d
    }
  ]
}`, uint64(math.MaxUint64))
	if err := os.WriteFile(filepath.Join(dir, IndexFileName), []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}
	for _, filename := range []string{"legacy.json", "seven.json", "max.json"} {
		if err := os.WriteFile(filepath.Join(dir, filename), []byte("payload of "+filename), 0644); err != nil {
			t.Fatalf("WriteFile %s: %v", filename, err)
		}
	}
	return dir
}

func readIndexEntryIDs(t *testing.T, dir string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, IndexFileName))
	if err != nil {
		t.Fatalf("ReadFile index: %v", err)
	}
	var idx indexFile
	if err := json.Unmarshal(data, &idx); err != nil {
		t.Fatalf("unmarshal index: %v", err)
	}
	if idx.Version != IndexVersion {
		t.Fatalf("rewritten index version = %d, want %d", idx.Version, IndexVersion)
	}
	ids := make([]string, 0, len(idx.Entries))
	for _, e := range idx.Entries {
		ids = append(ids, e.ID)
	}
	return ids
}

func snapshotDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	names, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	snapshot := make(map[string]string, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name.Name()))
		if err != nil {
			t.Fatalf("ReadFile %s: %v", name.Name(), err)
		}
		snapshot[name.Name()] = string(data)
	}
	return snapshot
}

// A delete against an explicit revision removes exactly one entry and exactly
// one payload; the surviving entries keep their order, and a neighbouring
// payload is left untouched.
func TestStore_DeleteTemplate_Success(t *testing.T) {
	dir := newDeleteFixture(t)
	store := NewStore(dir)

	if err := store.DeleteTemplate("tpl-seven", "7"); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}

	if got := readIndexEntryIDs(t, dir); !reflect.DeepEqual(got, []string{"tpl-legacy", "tpl-max"}) {
		t.Errorf("index entries = %v, want [tpl-legacy tpl-max]", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "seven.json")); !os.IsNotExist(err) {
		t.Errorf("payload seven.json still present: %v", err)
	}
	neighbour, err := os.ReadFile(filepath.Join(dir, "legacy.json"))
	if err != nil {
		t.Fatalf("ReadFile legacy.json: %v", err)
	}
	if string(neighbour) != "payload of legacy.json" {
		t.Errorf("neighbouring payload = %q, want it untouched", neighbour)
	}

	// The entry is gone, so a repeat delete is a not-found.
	err = store.DeleteTemplate("tpl-seven", "7")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("repeat DeleteTemplate error = %v, want ErrNotFound", err)
	}
}

// An entry written before the revision counter existed has the canonical token
// "0", which must be enough to delete it.
func TestStore_DeleteTemplate_LegacyRevisionZero(t *testing.T) {
	dir := newDeleteFixture(t)
	store := NewStore(dir)

	if err := store.DeleteTemplate("tpl-legacy", "0"); err != nil {
		t.Fatalf("DeleteTemplate legacy entry: %v", err)
	}
	if got := readIndexEntryIDs(t, dir); !reflect.DeepEqual(got, []string{"tpl-seven", "tpl-max"}) {
		t.Errorf("index entries = %v, want [tpl-seven tpl-max]", got)
	}
}

// math.MaxUint64 is an ordinary, valid revision for Delete.
func TestStore_DeleteTemplate_MaxUint64Revision(t *testing.T) {
	dir := newDeleteFixture(t)
	store := NewStore(dir)

	if err := store.DeleteTemplate("tpl-max", "18446744073709551615"); err != nil {
		t.Fatalf("DeleteTemplate max revision: %v", err)
	}
	if got := readIndexEntryIDs(t, dir); !reflect.DeepEqual(got, []string{"tpl-legacy", "tpl-seven"}) {
		t.Errorf("index entries = %v, want [tpl-legacy tpl-seven]", got)
	}
}

// A revision that no longer matches the entry is a stale-revision refusal, and
// it must not change a single byte in the library.
func TestStore_DeleteTemplate_StaleRevisionMutatesNothing(t *testing.T) {
	dir := newDeleteFixture(t)
	store := NewStore(dir)
	before := snapshotDir(t, dir)

	err := store.DeleteTemplate("tpl-seven", "6")
	if !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("DeleteTemplate error = %v, want ErrStaleRevision", err)
	}
	// A legacy entry must not accept a non-zero token either.
	if err := store.DeleteTemplate("tpl-legacy", "1"); !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("legacy entry with revision \"1\": error = %v, want ErrStaleRevision", err)
	}
	if after := snapshotDir(t, dir); !reflect.DeepEqual(before, after) {
		t.Error("a stale revision changed the library on disk")
	}
}

// Non-canonical tokens are rejected before the first file is touched.
func TestStore_DeleteTemplate_NonCanonicalRevisionRejected(t *testing.T) {
	dir := newDeleteFixture(t)
	store := NewStore(dir)
	before := snapshotDir(t, dir)

	for _, token := range []string{"", "01", "+1", "-1", " 1", "1 ", "1.0", "0x7", "seven", "18446744073709551616"} {
		err := store.DeleteTemplate("tpl-seven", token)
		if err == nil {
			t.Fatalf("DeleteTemplate(%q) succeeded, want a canonical-token rejection", token)
		}
		if errors.Is(err, ErrStaleRevision) || errors.Is(err, ErrNotFound) {
			t.Errorf("DeleteTemplate(%q) error = %v, want a canonical-token rejection", token, err)
		}
		if !strings.Contains(err.Error(), "canonical decimal revision token") {
			t.Errorf("DeleteTemplate(%q) error = %v, want the canonical-token message", token, err)
		}
	}
	if after := snapshotDir(t, dir); !reflect.DeepEqual(before, after) {
		t.Error("a non-canonical revision changed the library on disk")
	}
}

func TestStore_DeleteTemplate_RejectsNilStoreAndEmptyID(t *testing.T) {
	var nilStore *Store
	if err := nilStore.DeleteTemplate("tpl-seven", "7"); err == nil {
		t.Error("nil store: expected an error, got nil")
	}

	store := NewStore(newDeleteFixture(t))
	if err := store.DeleteTemplate("", "7"); err == nil || err.Error() != "templateID must not be empty" {
		t.Errorf("empty templateID error = %v, want \"templateID must not be empty\"", err)
	}
}

func TestStore_DeleteTemplate_UnknownTemplateID(t *testing.T) {
	dir := newDeleteFixture(t)
	store := NewStore(dir)
	before := snapshotDir(t, dir)

	if err := store.DeleteTemplate("tpl-unknown", "0"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteTemplate error = %v, want ErrNotFound", err)
	}
	if after := snapshotDir(t, dir); !reflect.DeepEqual(before, after) {
		t.Error("an unknown templateID changed the library on disk")
	}
}

// A payload the user already removed by hand does not block the entry removal:
// the index is the authority, and it must not keep pointing at a missing file.
func TestStore_DeleteTemplate_MissingPayloadStillRemovesEntry(t *testing.T) {
	dir := newDeleteFixture(t)
	if err := os.Remove(filepath.Join(dir, "seven.json")); err != nil {
		t.Fatalf("Remove payload: %v", err)
	}
	store := NewStore(dir)

	if err := store.DeleteTemplate("tpl-seven", "7"); err != nil {
		t.Fatalf("DeleteTemplate with missing payload: %v", err)
	}
	if got := readIndexEntryIDs(t, dir); !reflect.DeepEqual(got, []string{"tpl-legacy", "tpl-max"}) {
		t.Errorf("index entries = %v, want [tpl-legacy tpl-max]", got)
	}
}

// The writer refuses to rewrite an index it cannot fully account for, even when
// the defect sits on an entry other than the target.
func TestStore_DeleteTemplate_FailClosedIndexValidation(t *testing.T) {
	for name, spec := range map[string]struct {
		index    string
		wantText string
	}{
		"duplicate template ID": {
			index: `{"version":1,"entries":[
			  {"id":"tpl-dup","name":"A","filename":"a.json"},
			  {"id":"tpl-dup","name":"B","filename":"b.json"},
			  {"id":"tpl-target","name":"T","filename":"t.json"}
			]}`,
			wantText: "duplicate template ID",
		},
		"shared filename": {
			index: `{"version":1,"entries":[
			  {"id":"tpl-a","name":"A","filename":"shared.json"},
			  {"id":"tpl-b","name":"B","filename":"shared.json"},
			  {"id":"tpl-target","name":"T","filename":"t.json"}
			]}`,
			wantText: "shares filename",
		},
		"target filename shared with another entry": {
			index: `{"version":1,"entries":[
			  {"id":"tpl-other","name":"O","filename":"t.json"},
			  {"id":"tpl-target","name":"T","filename":"t.json"}
			]}`,
			wantText: "shares filename",
		},
		"empty filename": {
			index: `{"version":1,"entries":[
			  {"id":"tpl-empty","name":"E","filename":""},
			  {"id":"tpl-target","name":"T","filename":"t.json"}
			]}`,
			wantText: "empty filename",
		},
		"unsafe filename": {
			index: `{"version":1,"entries":[
			  {"id":"tpl-trav","name":"X","filename":"../escape.json"},
			  {"id":"tpl-target","name":"T","filename":"t.json"}
			]}`,
			wantText: "invalid filename",
		},
		"filename dot": {
			index: `{"version":1,"entries":[
			  {"id":"tpl-dot","name":"Dot","filename":"."},
			  {"id":"tpl-target","name":"T","filename":"t.json"}
			]}`,
			wantText: "invalid filename",
		},
		"filename dot-dot": {
			index: `{"version":1,"entries":[
			  {"id":"tpl-dotdot","name":"DotDot","filename":".."},
			  {"id":"tpl-target","name":"T","filename":"t.json"}
			]}`,
			wantText: "invalid filename",
		},
		"unsupported index version": {
			index:    `{"version":99,"entries":[{"id":"tpl-target","name":"T","filename":"t.json"}]}`,
			wantText: "unsupported index version",
		},
		"unknown top-level field": {
			index:    `{"version":1,"unknownField":"val","entries":[{"id":"tpl-target","name":"T","filename":"t.json"}]}`,
			wantText: "unknown field",
		},
		"unknown entry field": {
			index: `{"version":1,"entries":[
			  {"id":"tpl-target","name":"T","filename":"t.json","unknownField":"val"}
			]}`,
			wantText: "unknown field",
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, IndexFileName), []byte(spec.index), 0644); err != nil {
				t.Fatalf("WriteFile index: %v", err)
			}
			if err := os.WriteFile(filepath.Join(dir, "t.json"), []byte("target payload"), 0644); err != nil {
				t.Fatalf("WriteFile payload: %v", err)
			}
			store := NewStore(dir)
			before := snapshotDir(t, dir)

			err := store.DeleteTemplate("tpl-target", "0")
			if err == nil {
				t.Fatal("DeleteTemplate succeeded on an invalid index")
			}
			if !strings.Contains(err.Error(), spec.wantText) {
				t.Errorf("error = %v, want it to mention %q", err, spec.wantText)
			}
			if after := snapshotDir(t, dir); !reflect.DeepEqual(before, after) {
				t.Error("an invalid index was partially rewritten")
			}
		})
	}
}

func newUpdateFixture(t *testing.T, warnings int) (string, *Store) {
	t.Helper()
	dir := t.TempDir()
	indexJSON := fmt.Sprintf(`{
  "version": 1,
  "entries": [
    {
      "id": "tpl-123",
      "name": "Old Name",
      "description": "Old Desc",
      "tags": ["old-tag"],
      "filename": "tpl-123.json",
      "createdAt": "2026-05-17T10:00:00Z",
      "updatedAt": "2026-05-17T10:00:00Z",
      "inventoryItems": 1,
      "storageItems": 0,
      "warnings": %d,
      "version": 1,
      "selectedSections": ["inventory.workspace"],
      "revision": 5
    }
  ]
}`, warnings)
	if err := os.WriteFile(filepath.Join(dir, IndexFileName), []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}
	payloadJSON := `{
  "schema": "saveforge.build-template",
  "version": 1,
  "createdAt": "2026-05-17T10:00:00Z",
  "metadata": {
    "name": "Old Name",
    "description": "Old Desc",
    "author": "Original Author",
    "tags": ["old-tag"],
    "sourceCharacterIndex": 2,
    "sourceCharacterName": "Tarnished"
  },
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
	return dir, NewStore(dir)
}

func TestStore_UpdateTemplate_MetadataOnly(t *testing.T) {
	dir, store := newUpdateFixture(t, 2)

	newRev, err := store.UpdateTemplate("tpl-123", "5", &TemplateMetadataUpdate{
		Name:        "Updated Name",
		Description: "Updated Desc",
		Tags:        []string{"new-tag1", "new-tag2"},
	}, nil)
	if err != nil {
		t.Fatalf("UpdateTemplate failed: %v", err)
	}
	if newRev != "6" {
		t.Errorf("newRev = %q, want %q", newRev, "6")
	}

	tpl, rev, err := store.GetTemplate("tpl-123")
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if rev != "6" || tpl.Metadata.Name != "Updated Name" || tpl.Metadata.Description != "Updated Desc" {
		t.Errorf("GetTemplate mismatch: rev=%q tpl=%+v", rev, tpl.Metadata)
	}
	if tpl.Metadata.Author != "Original Author" || tpl.Metadata.SourceCharacterIndex != 2 || tpl.Metadata.SourceCharacterName != "Tarnished" {
		t.Errorf("author/source character corrupted: %+v", tpl.Metadata)
	}

	list, err := store.ListTemplates()
	if err != nil || len(list) != 1 {
		t.Fatalf("ListTemplates err=%v len=%d", err, len(list))
	}
	e := list[0]
	if e.Name != "Updated Name" || e.Warnings != 2 || e.TemplateRevision != "6" {
		t.Errorf("index entry mismatch: %+v", e)
	}
	_ = dir
}

func TestStore_UpdateTemplate_ContentOnly(t *testing.T) {
	_, store := newUpdateFixture(t, 3)

	var newDoc BuildTemplate
	v2JSON := `{
  "schema": "saveforge.build-template",
  "version": 2,
  "createdAt": "2026-08-17T12:00:00Z",
  "metadata": {
    "name": "New V2 Build",
    "description": "V2 Description",
    "tags": ["v2", "items"]
  },
  "selection": {"items": true},
  "sections": {
    "items": {
      "entries": [
        {"entryID": "entry-1", "itemID": 1000, "category": "melee_armaments", "quantity": 1, "location": "inventory"},
        {"entryID": "entry-2", "itemID": 2000, "category": "talismans", "quantity": 2, "location": "storage"}
      ]
    }
  }
}`
	if err := json.Unmarshal([]byte(v2JSON), &newDoc); err != nil {
		t.Fatalf("unmarshal v2Doc: %v", err)
	}

	newRev, err := store.UpdateTemplate("tpl-123", "5", nil, &newDoc)
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	if newRev != "6" {
		t.Errorf("newRev = %q, want %q", newRev, "6")
	}

	loaded, rev, err := store.GetTemplate("tpl-123")
	if err != nil || rev != "6" || loaded.Version != 2 || loaded.Metadata.Name != "New V2 Build" {
		t.Errorf("loaded template mismatch: rev=%q err=%v tpl=%+v", rev, err, loaded)
	}

	list, err := store.ListTemplates()
	if err != nil || len(list) != 1 {
		t.Fatalf("ListTemplates err=%v len=%d", err, len(list))
	}
	e := list[0]
	if e.InventoryItems != 1 || e.StorageItems != 1 || e.SchemaVersion != 2 || e.Warnings != 3 {
		t.Errorf("index entry mismatch: %+v", e)
	}
}

func TestStore_UpdateTemplate_BothFields_MetadataPrecedence(t *testing.T) {
	_, store := newUpdateFixture(t, 0)

	var doc BuildTemplate
	docJSON := `{
  "schema": "saveforge.build-template",
  "version": 1,
  "createdAt": "2026-08-17T12:00:00Z",
  "metadata": {
    "name": "Content Name",
    "description": "Content Desc",
    "author": "Content Author",
    "tags": ["content-tag"]
  },
  "sections": {
    "inventory.workspace": {
      "inventoryItems": [{"baseItemID": 100, "quantity": 1, "container": "inventory", "position": 0}],
      "storageItems": []
    }
  }
}`
	if err := json.Unmarshal([]byte(docJSON), &doc); err != nil {
		t.Fatalf("unmarshal doc: %v", err)
	}

	newRev, err := store.UpdateTemplate("tpl-123", "5", &TemplateMetadataUpdate{
		Name:        "Override Name",
		Description: "Override Desc",
		Tags:        []string{"override-tag"},
	}, &doc)
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	if newRev != "6" {
		t.Errorf("newRev = %q, want %q", newRev, "6")
	}

	loaded, _, err := store.GetTemplate("tpl-123")
	if err != nil || loaded.Metadata.Name != "Override Name" || loaded.Metadata.Author != "Content Author" {
		t.Errorf("metadata override failed: %+v", loaded.Metadata)
	}
}

func TestStore_UpdateTemplate_Rejections(t *testing.T) {
	for name, setup := range map[string]func(t *testing.T) (store *Store, id, rev string, meta *TemplateMetadataUpdate, content *BuildTemplate, dir string){
		"nil store": func(t *testing.T) (*Store, string, string, *TemplateMetadataUpdate, *BuildTemplate, string) {
			return nil, "tpl-123", "5", &TemplateMetadataUpdate{Name: "X"}, nil, ""
		},
		"empty templateID": func(t *testing.T) (*Store, string, string, *TemplateMetadataUpdate, *BuildTemplate, string) {
			dir, store := newUpdateFixture(t, 0)
			return store, "", "5", &TemplateMetadataUpdate{Name: "X"}, nil, dir
		},
		"neither metadata nor content": func(t *testing.T) (*Store, string, string, *TemplateMetadataUpdate, *BuildTemplate, string) {
			dir, store := newUpdateFixture(t, 0)
			return store, "tpl-123", "5", nil, nil, dir
		},
		"stale revision": func(t *testing.T) (*Store, string, string, *TemplateMetadataUpdate, *BuildTemplate, string) {
			dir, store := newUpdateFixture(t, 0)
			return store, "tpl-123", "4", &TemplateMetadataUpdate{Name: "X"}, nil, dir
		},
		"non-canonical revision": func(t *testing.T) (*Store, string, string, *TemplateMetadataUpdate, *BuildTemplate, string) {
			dir, store := newUpdateFixture(t, 0)
			return store, "tpl-123", "05", &TemplateMetadataUpdate{Name: "X"}, nil, dir
		},
		"max uint64 overflow": func(t *testing.T) (*Store, string, string, *TemplateMetadataUpdate, *BuildTemplate, string) {
			dir := t.TempDir()
			idx := fmt.Sprintf(`{"version":1,"entries":[{"id":"tpl-max","name":"M","filename":"m.json","revision":%d}]}`, uint64(math.MaxUint64))
			_ = os.WriteFile(filepath.Join(dir, IndexFileName), []byte(idx), 0644)
			_ = os.WriteFile(filepath.Join(dir, "m.json"), []byte(`{"schema":"saveforge.build-template","version":1,"metadata":{"name":"M"},"sections":{"inventory.workspace":{"inventoryItems":[{"baseItemID":100,"quantity":1,"container":"inventory","position":0}],"storageItems":[]}}}`), 0644)
			return NewStore(dir), "tpl-max", "18446744073709551615", &TemplateMetadataUpdate{Name: "X"}, nil, dir
		},
		"not found": func(t *testing.T) (*Store, string, string, *TemplateMetadataUpdate, *BuildTemplate, string) {
			dir, store := newUpdateFixture(t, 0)
			return store, "tpl-missing", "0", &TemplateMetadataUpdate{Name: "X"}, nil, dir
		},
		"invalid candidate template content": func(t *testing.T) (*Store, string, string, *TemplateMetadataUpdate, *BuildTemplate, string) {
			dir, store := newUpdateFixture(t, 0)
			invalidDoc := &BuildTemplate{Schema: SchemaKey, Version: 1}
			return store, "tpl-123", "5", nil, invalidDoc, dir
		},
		"corrupted existing payload": func(t *testing.T) (*Store, string, string, *TemplateMetadataUpdate, *BuildTemplate, string) {
			dir, store := newUpdateFixture(t, 0)
			_ = os.WriteFile(filepath.Join(dir, "tpl-123.json"), []byte(`{invalid`), 0644)
			return store, "tpl-123", "5", &TemplateMetadataUpdate{Name: "X"}, nil, dir
		},
		"existing payload metadata mismatch with index": func(t *testing.T) (*Store, string, string, *TemplateMetadataUpdate, *BuildTemplate, string) {
			dir, store := newUpdateFixture(t, 0)
			mismatchedPayload := `{"schema":"saveforge.build-template","version":1,"metadata":{"name":"Mismatched Name"},"sections":{"inventory.workspace":{"inventoryItems":[{"baseItemID":100,"quantity":1,"container":"inventory","position":0}],"storageItems":[]}}}`
			_ = os.WriteFile(filepath.Join(dir, "tpl-123.json"), []byte(mismatchedPayload), 0644)
			return store, "tpl-123", "5", &TemplateMetadataUpdate{Name: "X"}, nil, dir
		},
		"symlink payload rejected": func(t *testing.T) (*Store, string, string, *TemplateMetadataUpdate, *BuildTemplate, string) {
			dir, store := newUpdateFixture(t, 0)
			realFile := filepath.Join(dir, "real.json")
			_ = os.Rename(filepath.Join(dir, "tpl-123.json"), realFile)
			_ = os.Symlink(realFile, filepath.Join(dir, "tpl-123.json"))
			return store, "tpl-123", "5", &TemplateMetadataUpdate{Name: "X"}, nil, dir
		},
	} {
		t.Run(name, func(t *testing.T) {
			store, id, rev, meta, content, dir := setup(t)
			var before map[string]string
			if dir != "" {
				before = snapshotDir(t, dir)
			}
			_, err := store.UpdateTemplate(id, rev, meta, content)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if dir != "" {
				if after := snapshotDir(t, dir); !reflect.DeepEqual(before, after) {
					t.Error("files were mutated on error")
				}
			}
		})
	}
}
