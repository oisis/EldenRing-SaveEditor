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
	self := schema.ResourceRef{Kind: schema.ResourceKindItem, Key: "000F4240"}
	relation := schema.Relation{
		From: self,
		To:   self,
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

// A relation endpoint is a complete (kind, key) pair; either half missing is a
// rejection, and the same key under a different kind is not a self reference.
func TestValidateRelationRequiresCompleteEndpoints(t *testing.T) {
	manifest, _ := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	provenance := schema.Provenance{Source: manifest.Sources[0].ID, Method: "test"}
	complete := schema.ResourceRef{Kind: schema.ResourceKindItem, Key: "000F4240"}

	cases := map[string]schema.Relation{
		"missing from kind": {From: schema.ResourceRef{Key: "000F4240"}, To: complete},
		"missing from key":  {From: schema.ResourceRef{Kind: schema.ResourceKindItem}, To: complete},
		"missing to kind":   {From: complete, To: schema.ResourceRef{Key: "8000EA60"}},
		"missing to key":    {From: complete, To: schema.ResourceRef{Kind: schema.ResourceKindItem}},
		"unsupported kind":  {From: complete, To: schema.ResourceRef{Kind: "gesture", Key: "8000EA60"}},
	}
	for name, relation := range cases {
		t.Run(name, func(t *testing.T) {
			relation.Kind = schema.RelationCompatibleWithAshOfWar
			relation.Provenance = provenance
			if err := schema.ValidateRelation(relation, sources); err == nil {
				t.Fatal("ValidateRelation error = nil, want an incomplete-endpoint rejection")
			}
		})
	}

	valid := schema.Relation{
		From:       complete,
		To:         schema.ResourceRef{Kind: schema.ResourceKindItem, Key: "8000EA60"},
		Kind:       schema.RelationCompatibleWithAshOfWar,
		Provenance: provenance,
	}
	if err := schema.ValidateRelation(valid, sources); err != nil {
		t.Fatalf("ValidateRelation(complete endpoints) = %v, want nil", err)
	}
}
