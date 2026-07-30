package schema_test

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestValidateRelationRejectsSelfReference(t *testing.T) {
	manifest, _ := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	relation := schema.Relation{
		From: 1,
		To:   1,
		Kind: schema.RelationCompatibleWithAshOfWar,
		Provenance: schema.Provenance{
			Source: manifest.Sources[0].ID,
			Method: "test",
		},
	}

	err := schema.ValidateRelation(relation, sources)
	if err == nil || !strings.Contains(err.Error(), "same resource") {
		t.Fatalf("ValidateRelation error = %v, want self-reference rejection", err)
	}
}
