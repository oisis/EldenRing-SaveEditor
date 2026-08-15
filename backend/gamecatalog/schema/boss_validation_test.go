package schema_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func bossFixture(t *testing.T) (schema.Resource, map[schema.SourceID]struct{}) {
	t.Helper()

	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	provenance := resources[0].Item.Presentation.Name.Provenance
	return schema.Resource{
		Key:  "stormveil_castle_godrick_the_grafted",
		Kind: schema.ResourceKindBoss,
		Boss: &schema.BossDocument{
			Name:              schema.Fact[string]{Known: true, Value: "Godrick the Grafted", Provenance: provenance},
			RegionLabel:       schema.Fact[string]{Known: true, Value: "Stormveil Castle", Provenance: provenance},
			EncounterType:     schema.Fact[string]{Known: true, Value: schema.BossEncounterTypeMain, Provenance: provenance},
			Remembrance:       schema.Fact[bool]{Known: true, Value: true, Provenance: provenance},
			DefeatEventFlagID: schema.Fact[uint32]{Known: true, Value: 9101, Provenance: provenance},
		},
	}, sources
}

func TestValidateBossResourceFailsClosed(t *testing.T) {
	complete, sources := bossFixture(t)
	if err := schema.ValidateResource(complete, sources); err != nil {
		t.Fatalf("ValidateResource on a complete boss: %v", err)
	}

	// A field encounter without a remembrance is the other confirmed shape.
	field, _ := bossFixture(t)
	field.Boss.EncounterType.Value = schema.BossEncounterTypeField
	field.Boss.Remembrance.Value = false
	if err := schema.ValidateResource(field, sources); err != nil {
		t.Fatalf("ValidateResource on a field boss: %v", err)
	}

	for name, break_ := range map[string]func(*schema.Resource){
		"missing document": func(resource *schema.Resource) { resource.Boss = nil },
		"foreign document in the union": func(resource *schema.Resource) {
			resource.Grace = &schema.GraceDocument{}
		},
		"uppercase key": func(resource *schema.Resource) { resource.Key = "Godrick" },
		"unknown name": func(resource *schema.Resource) {
			resource.Boss.Name = schema.Fact[string]{Provenance: resource.Boss.Name.Provenance}
		},
		"empty region label": func(resource *schema.Resource) {
			resource.Boss.RegionLabel.Value = ""
		},
		"unknown encounter type": func(resource *schema.Resource) {
			resource.Boss.EncounterType.Known = false
		},
		"unsupported encounter type": func(resource *schema.Resource) {
			resource.Boss.EncounterType.Value = "boss"
		},
		"unknown remembrance": func(resource *schema.Resource) {
			resource.Boss.Remembrance.Known = false
		},
		"zero defeat flag": func(resource *schema.Resource) {
			resource.Boss.DefeatEventFlagID.Value = 0
		},
		// The neighbouring blocks 8 and 10 carry no curated boss, so they must be
		// rejected exactly like a far-away block.
		"defeat flag below the confirmed block": func(resource *schema.Resource) {
			resource.Boss.DefeatEventFlagID.Value = 8999
		},
		"defeat flag above the confirmed block": func(resource *schema.Resource) {
			resource.Boss.DefeatEventFlagID.Value = 10000
		},
		"defeat flag of another curated kind": func(resource *schema.Resource) {
			resource.Boss.DefeatEventFlagID.Value = 71000
		},
	} {
		resource, _ := bossFixture(t)
		break_(&resource)
		if err := schema.ValidateResource(resource, sources); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}
