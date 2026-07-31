package schema_test

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestValidateManifestRejectsDuplicateSource(t *testing.T) {
	manifest, _ := prototype.Data()
	manifest.Sources = append(manifest.Sources, manifest.Sources[0])

	_, err := schema.ValidateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "duplicate source ID") {
		t.Fatalf("ValidateManifest error = %v, want duplicate source rejection", err)
	}
}

func TestValidateManifestRejectsTemporarySourceLocation(t *testing.T) {
	manifest, _ := prototype.Data()
	manifest.Sources[0].Location = "tmp/regulation-bin-dump/csv/EquipParamWeapon.csv"

	_, err := schema.ValidateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "temporary working directory") {
		t.Fatalf("ValidateManifest error = %v, want temporary location rejection", err)
	}
}

func TestValidateManifestRejectsAbsoluteSourceLocation(t *testing.T) {
	manifest, _ := prototype.Data()
	manifest.Sources[0].Location = "/Users/developer/regulation.bin"

	_, err := schema.ValidateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "absolute local path") {
		t.Fatalf("ValidateManifest error = %v, want absolute location rejection", err)
	}
}

func TestValidateManifestAcceptsGameDataEvidence(t *testing.T) {
	manifest, _ := prototype.Data()
	manifest.Sources[0].Evidence = schema.EvidenceGameData

	if _, err := schema.ValidateManifest(manifest); err != nil {
		t.Fatalf("ValidateManifest: %v", err)
	}
}

func TestValidateManifestRejectsFutureSchemaVersion(t *testing.T) {
	manifest, _ := prototype.Data()
	manifest.SchemaVersion = schema.CurrentSchemaVersion + 1

	_, err := schema.ValidateManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("ValidateManifest error = %v, want unsupported-version rejection", err)
	}
}
