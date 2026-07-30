package schema_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func mustValidateManifest(t *testing.T, manifest schema.Manifest) map[schema.SourceID]struct{} {
	t.Helper()
	sources, err := schema.ValidateManifest(manifest)
	if err != nil {
		t.Fatalf("ValidateManifest: %v", err)
	}
	return sources
}
