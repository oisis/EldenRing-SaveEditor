package templates_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/templates"
)

func newTestStoreWithIndex(t *testing.T, indexJSON string) *buildtemplates.Store {
	t.Helper()
	dir := t.TempDir()
	indexPath := filepath.Join(dir, buildtemplates.IndexFileName)
	if err := os.WriteFile(indexPath, []byte(indexJSON), 0644); err != nil {
		t.Fatalf("WriteFile _index.json: %v", err)
	}
	return buildtemplates.NewStore(dir)
}

func TestGetBuildTemplates_RejectsNilStore(t *testing.T) {
	_, err := templates.GetBuildTemplates(nil, "", nil, 1, 50)
	if err == nil || err.Error() != "templates store is not available" {
		t.Fatalf("expected 'templates store is not available' error, got %v", err)
	}
}

func TestGetBuildTemplates_RejectsNegativePaging(t *testing.T) {
	store := buildtemplates.NewStore(t.TempDir())

	if _, err := templates.GetBuildTemplates(store, "", nil, -1, 50); err == nil {
		t.Fatal("expected error for page < 0, got nil")
	}
	if _, err := templates.GetBuildTemplates(store, "", nil, 1, -1); err == nil {
		t.Fatal("expected error for pageSize < 0, got nil")
	}
}

func TestGetBuildTemplates_RejectsEmptyTag(t *testing.T) {
	store := buildtemplates.NewStore(t.TempDir())

	if _, err := templates.GetBuildTemplates(store, "", []string{"valid", ""}, 1, 50); err == nil {
		t.Fatal("expected error for empty tag element, got nil")
	}
}

func TestGetBuildTemplates_EmptyLibraryReturnsEmptyResult(t *testing.T) {
	store := buildtemplates.NewStore(filepath.Join(t.TempDir(), "nonexistent"))

	result, err := templates.GetBuildTemplates(store, "", nil, 0, 0)
	if err != nil {
		t.Fatalf("GetBuildTemplates: %v", err)
	}

	if result.Total != 0 || len(result.Templates) != 0 || result.Page != 1 || result.PageSize != templates.GetBuildTemplatesDefaultPageSize {
		t.Fatalf("unexpected empty result: %+v", result)
	}
	if result.Templates == nil {
		t.Fatal("Templates slice must not be nil")
	}
}

func TestGetBuildTemplates_SearchFiltering(t *testing.T) {
	indexJSON := `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-1",
      "name": "Bleed Bandit",
      "description": "Rivers of Blood focus",
      "tags": ["pvp"],
      "filename": "t1.json",
      "createdAt": "2026-05-17T10:00:00Z",
      "updatedAt": "2026-05-17T12:00:00Z"
    },
    {
      "id": "tpl-2",
      "name": "Sorcerer Supreme",
      "description": "High INT bleed backup",
      "tags": ["pve"],
      "filename": "t2.json",
      "createdAt": "2026-05-17T10:00:00Z",
      "updatedAt": "2026-05-17T11:00:00Z"
    },
    {
      "id": "tpl-3",
      "name": "Pure Strength",
      "description": "Colossal swords only",
      "tags": ["pve"],
      "filename": "t3.json",
      "createdAt": "2026-05-17T10:00:00Z",
      "updatedAt": "2026-05-17T10:00:00Z"
    }
  ]
}`
	store := newTestStoreWithIndex(t, indexJSON)

	// Search matching name and description case-insensitively
	res, err := templates.GetBuildTemplates(store, "BLEED", nil, 1, 10)
	if err != nil {
		t.Fatalf("GetBuildTemplates: %v", err)
	}
	if res.Total != 2 || len(res.Templates) != 2 {
		t.Fatalf("expected 2 matches for BLEED, got total=%d len=%d", res.Total, len(res.Templates))
	}
	if res.Templates[0].TemplateID != "tpl-1" || res.Templates[1].TemplateID != "tpl-2" {
		t.Fatalf("unexpected search results order: %v, %v", res.Templates[0].TemplateID, res.Templates[1].TemplateID)
	}

	// Search matching description only
	resDesc, err := templates.GetBuildTemplates(store, "colossal", nil, 1, 10)
	if err != nil {
		t.Fatalf("GetBuildTemplates: %v", err)
	}
	if resDesc.Total != 1 || len(resDesc.Templates) != 1 || resDesc.Templates[0].TemplateID != "tpl-3" {
		t.Fatalf("expected tpl-3 match for colossal, got %+v", resDesc)
	}

	// Search with no match
	resNone, err := templates.GetBuildTemplates(store, "nonexistent", nil, 1, 10)
	if err != nil {
		t.Fatalf("GetBuildTemplates: %v", err)
	}
	if resNone.Total != 0 || len(resNone.Templates) != 0 {
		t.Fatalf("expected 0 matches, got %+v", resNone)
	}
}

func TestGetBuildTemplates_TagsFiltering(t *testing.T) {
	indexJSON := `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-1",
      "name": "PvP Bleed",
      "tags": ["pvp", "bleed", "meta"],
      "filename": "t1.json",
      "createdAt": "2026-05-17T10:00:00Z",
      "updatedAt": "2026-05-17T12:00:00Z"
    },
    {
      "id": "tpl-2",
      "name": "PvE Bleed",
      "tags": ["pve", "bleed"],
      "filename": "t2.json",
      "createdAt": "2026-05-17T10:00:00Z",
      "updatedAt": "2026-05-17T11:00:00Z"
    },
    {
      "id": "tpl-3",
      "name": "PvP Mage",
      "tags": ["pvp", "sorcery"],
      "filename": "t3.json",
      "createdAt": "2026-05-17T10:00:00Z",
      "updatedAt": "2026-05-17T10:00:00Z"
    }
  ]
}`
	store := newTestStoreWithIndex(t, indexJSON)

	// Single tag "bleed"
	resBleed, err := templates.GetBuildTemplates(store, "", []string{"bleed"}, 1, 10)
	if err != nil {
		t.Fatalf("GetBuildTemplates: %v", err)
	}
	if resBleed.Total != 2 {
		t.Fatalf("expected 2 matches for tag 'bleed', got %d", resBleed.Total)
	}

	// Multiple tags "pvp" AND "bleed"
	resBoth, err := templates.GetBuildTemplates(store, "", []string{"pvp", "bleed"}, 1, 10)
	if err != nil {
		t.Fatalf("GetBuildTemplates: %v", err)
	}
	if resBoth.Total != 1 || resBoth.Templates[0].TemplateID != "tpl-1" {
		t.Fatalf("expected 1 match for tags 'pvp' + 'bleed', got %+v", resBoth)
	}

	// Case sensitivity: "PVP" should not match lowercase "pvp"
	resCase, err := templates.GetBuildTemplates(store, "", []string{"PVP"}, 1, 10)
	if err != nil {
		t.Fatalf("GetBuildTemplates: %v", err)
	}
	if resCase.Total != 0 {
		t.Fatalf("expected 0 matches for case-sensitive 'PVP', got %d", resCase.Total)
	}
}

func TestGetBuildTemplates_Pagination(t *testing.T) {
	// Create index with 5 entries
	entriesJSON := ""
	for i := 1; i <= 5; i++ {
		if i > 1 {
			entriesJSON += ","
		}
		entriesJSON += fmt.Sprintf(`{
      "id": "tpl-%d",
      "name": "Build %d",
      "filename": "t%d.json",
      "createdAt": "2026-05-17T10:00:00Z",
      "updatedAt": "2026-05-17T10:%02d:00Z"
    }`, i, i, i, 10-i)
	}
	indexJSON := fmt.Sprintf(`{"version": 1, "entries": [%s]}`, entriesJSON)
	store := newTestStoreWithIndex(t, indexJSON)

	// Page 1 with pageSize 2 (entries tpl-1, tpl-2)
	p1, err := templates.GetBuildTemplates(store, "", nil, 1, 2)
	if err != nil {
		t.Fatalf("GetBuildTemplates page 1: %v", err)
	}
	if p1.Total != 5 || len(p1.Templates) != 2 || p1.Page != 1 || p1.PageSize != 2 {
		t.Fatalf("unexpected p1: %+v", p1)
	}
	if p1.Templates[0].TemplateID != "tpl-1" || p1.Templates[1].TemplateID != "tpl-2" {
		t.Fatalf("unexpected p1 templates: %v, %v", p1.Templates[0].TemplateID, p1.Templates[1].TemplateID)
	}

	// Page 2 with pageSize 2 (entries tpl-3, tpl-4)
	p2, err := templates.GetBuildTemplates(store, "", nil, 2, 2)
	if err != nil {
		t.Fatalf("GetBuildTemplates page 2: %v", err)
	}
	if p2.Total != 5 || len(p2.Templates) != 2 || p2.Page != 2 || p2.PageSize != 2 {
		t.Fatalf("unexpected p2: %+v", p2)
	}
	if p2.Templates[0].TemplateID != "tpl-3" || p2.Templates[1].TemplateID != "tpl-4" {
		t.Fatalf("unexpected p2 templates: %v, %v", p2.Templates[0].TemplateID, p2.Templates[1].TemplateID)
	}

	// Page 3 with pageSize 2 (entry tpl-5)
	p3, err := templates.GetBuildTemplates(store, "", nil, 3, 2)
	if err != nil {
		t.Fatalf("GetBuildTemplates page 3: %v", err)
	}
	if p3.Total != 5 || len(p3.Templates) != 1 || p3.Page != 3 || p3.PageSize != 2 {
		t.Fatalf("unexpected p3: %+v", p3)
	}
	if p3.Templates[0].TemplateID != "tpl-5" {
		t.Fatalf("unexpected p3 template: %v", p3.Templates[0].TemplateID)
	}

	// Page 4 (out of range) -> empty slice, correct total/page/pageSize
	p4, err := templates.GetBuildTemplates(store, "", nil, 4, 2)
	if err != nil {
		t.Fatalf("GetBuildTemplates page 4: %v", err)
	}
	if p4.Total != 5 || len(p4.Templates) != 0 || p4.Page != 4 || p4.PageSize != 2 {
		t.Fatalf("unexpected p4: %+v", p4)
	}
	if p4.Templates == nil {
		t.Fatal("Templates slice must not be nil when page is out of range")
	}
}

func TestGetBuildTemplates_PropagatesStoreErrors(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, buildtemplates.IndexFileName)
	if err := os.WriteFile(indexPath, []byte(`{bad json`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	store := buildtemplates.NewStore(dir)

	if _, err := templates.GetBuildTemplates(store, "", nil, 1, 50); err == nil {
		t.Fatal("expected error for corrupted index file, got nil")
	}
}

func TestGetBuildTemplates_ExposesTemplateRevision(t *testing.T) {
	store := newTestStoreWithIndex(t, `{
  "version": 1,
  "entries": [
    {
      "id": "tpl-rev",
      "name": "Revised",
      "filename": "rev.json",
      "createdAt": "2026-08-17T10:00:00Z",
      "updatedAt": "2026-08-17T11:00:00Z",
      "revision": 7
    },
    {
      "id": "tpl-legacy",
      "name": "Legacy",
      "filename": "legacy.json",
      "createdAt": "2026-08-17T09:00:00Z",
      "updatedAt": "2026-08-17T10:00:00Z"
    }
  ]
}`)

	result, err := templates.GetBuildTemplates(store, "", nil, 0, 0)
	if err != nil {
		t.Fatalf("GetBuildTemplates: %v", err)
	}
	if len(result.Templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(result.Templates))
	}
	if result.Templates[0].TemplateRevision != "7" {
		t.Errorf("tpl-rev templateRevision = %q, want \"7\"", result.Templates[0].TemplateRevision)
	}
	if result.Templates[1].TemplateRevision != "0" {
		t.Errorf("tpl-legacy templateRevision = %q, want \"0\"", result.Templates[1].TemplateRevision)
	}
}
