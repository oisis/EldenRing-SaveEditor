package schema_test

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func summoningPoolFixture(t *testing.T) (schema.Resource, map[schema.SourceID]struct{}) {
	t.Helper()

	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	provenance := resources[0].Item.Presentation.Name.Provenance
	return schema.Resource{
		Key:  "stormveil_castle_gateside_chamber",
		Kind: schema.ResourceKindSummoningPool,
		SummoningPool: &schema.SummoningPoolDocument{
			Name:                  schema.Fact[string]{Known: true, Value: "Gateside Chamber", Provenance: provenance},
			RegionLabel:           schema.Fact[string]{Known: true, Value: "Stormveil Castle", Provenance: provenance},
			ActivationEventFlagID: schema.Fact[uint32]{Known: true, Value: 670130, Provenance: provenance},
		},
	}, sources
}

// The single-document rule runs inside each known kind, at the position the
// pairwise checks used to hold, so a foreign document can never take precedence
// over the kind resolution or over the key rule.
func TestValidateResourceResolvesTheKindBeforeTheUnionCheck(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	document := resources[0].Item

	unknownKind := schema.Resource{Key: "stormveil_castle", Kind: "quest", Item: document}
	if err := schema.ValidateResource(unknownKind, sources); err == nil ||
		!strings.Contains(err.Error(), "unsupported kind") {
		t.Errorf("unknown kind carrying a document = %v, want an unsupported kind error", err)
	}

	malformedItemKey := schema.Resource{
		Key:       "not_hex",
		Kind:      schema.ResourceKindItem,
		Item:      document,
		Colosseum: &schema.ColosseumDocument{},
	}
	if err := schema.ValidateResource(malformedItemKey, sources); err == nil ||
		!strings.Contains(err.Error(), "item key must be exactly eight") {
		t.Errorf("malformed item key with a foreign document = %v, want the key error", err)
	}
}

func TestValidateSummoningPoolResourceFailsClosed(t *testing.T) {
	complete, sources := summoningPoolFixture(t)
	if err := schema.ValidateResource(complete, sources); err != nil {
		t.Fatalf("ValidateResource on a complete summoning pool: %v", err)
	}

	for name, break_ := range map[string]func(*schema.Resource){
		"missing document": func(resource *schema.Resource) { resource.SummoningPool = nil },
		"foreign document in the union": func(resource *schema.Resource) {
			resource.Colosseum = &schema.ColosseumDocument{}
		},
		"unknown name": func(resource *schema.Resource) {
			resource.SummoningPool.Name = schema.Fact[string]{
				Provenance: resource.SummoningPool.Name.Provenance}
		},
		"empty region label": func(resource *schema.Resource) {
			resource.SummoningPool.RegionLabel.Value = ""
		},
		"zero activation flag": func(resource *schema.Resource) {
			resource.SummoningPool.ActivationEventFlagID.Value = 0
		},
		"flag below block 670": func(resource *schema.Resource) {
			resource.SummoningPool.ActivationEventFlagID.Value = 669999
		},
		"flag above block 670": func(resource *schema.Resource) {
			resource.SummoningPool.ActivationEventFlagID.Value = 671000
		},
	} {
		resource, _ := summoningPoolFixture(t)
		break_(&resource)
		if err := schema.ValidateResource(resource, sources); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}
