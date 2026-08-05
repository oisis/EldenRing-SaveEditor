package catalog_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func testManifest() schema.Manifest {
	return schema.Manifest{
		SchemaVersion: schema.CurrentSchemaVersion,
		DataVersion:   "2026.05.10",
		GameVersion:   "1.16",
		Sources: []schema.DataSource{{
			ID:       "regulation",
			Kind:     "game_data",
			Location: "backend/gamecatalog/data",
			Version:  "1.16",
			Evidence: schema.EvidenceRegulation,
			Reviewed: true,
		}},
	}
}

func newTestCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	gameCatalog, err := gamecatalog.New(testManifest(), nil)
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	return gameCatalog
}

func TestGetCatalogInfoReturnsManifest(t *testing.T) {
	t.Parallel()

	manifest := testManifest()
	result, err := catalog.GetCatalogInfo(newTestCatalog(t))
	if err != nil {
		t.Fatalf("GetCatalogInfo: %v", err)
	}

	if result.SchemaVersion != manifest.SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", result.SchemaVersion, manifest.SchemaVersion)
	}
	if result.DataVersion != manifest.DataVersion {
		t.Errorf("DataVersion = %q, want %q", result.DataVersion, manifest.DataVersion)
	}
	if result.GameVersion != manifest.GameVersion {
		t.Errorf("GameVersion = %q, want %q", result.GameVersion, manifest.GameVersion)
	}
	if !result.Valid {
		t.Error("Valid = false, want true for a catalog built by gamecatalog.New")
	}
	if len(result.Sources) != len(manifest.Sources) {
		t.Fatalf("len(Sources) = %d, want %d", len(result.Sources), len(manifest.Sources))
	}
	if result.Sources[0] != manifest.Sources[0] {
		t.Errorf("Sources[0] = %+v, want %+v", result.Sources[0], manifest.Sources[0])
	}
}

func TestGetCatalogInfoRejectsMissingCatalog(t *testing.T) {
	t.Parallel()

	if _, err := catalog.GetCatalogInfo(nil); err == nil {
		t.Fatal("GetCatalogInfo(nil) = nil error, want error")
	}
}

func TestGetCatalogInfoRejectsCatalogWithInvalidManifest(t *testing.T) {
	t.Parallel()

	// A zero-value catalog bypasses gamecatalog.New, so its manifest never
	// passed schema.ValidateManifest and must not be reported as valid.
	result, err := catalog.GetCatalogInfo(&gamecatalog.Catalog{})
	if err == nil {
		t.Fatalf("GetCatalogInfo(&gamecatalog.Catalog{}) = %+v, nil error; want error", result)
	}
	if result.Valid {
		t.Error("Valid = true for a manifest rejected by schema.ValidateManifest")
	}
	if result.SchemaVersion != 0 || result.DataVersion != "" || result.GameVersion != "" || result.Sources != nil {
		t.Errorf("result = %+v, want empty result on validation failure", result)
	}
}

func TestGetCatalogInfoDoesNotMutateCatalog(t *testing.T) {
	t.Parallel()

	gameCatalog := newTestCatalog(t)
	result, err := catalog.GetCatalogInfo(gameCatalog)
	if err != nil {
		t.Fatalf("GetCatalogInfo: %v", err)
	}

	result.SchemaVersion = 0
	result.DataVersion = "mutated"
	result.GameVersion = "mutated"
	result.Sources[0].Location = "mutated"

	after := gameCatalog.Manifest()
	if want := testManifest(); after.SchemaVersion != want.SchemaVersion ||
		after.DataVersion != want.DataVersion ||
		after.GameVersion != want.GameVersion ||
		after.Sources[0] != want.Sources[0] {
		t.Errorf("catalog manifest after GetCatalogInfo = %+v, want %+v", after, want)
	}
}
