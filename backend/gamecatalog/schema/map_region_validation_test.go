package schema_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func mapRegionFixture(t *testing.T) (schema.Resource, map[schema.SourceID]struct{}) {
	t.Helper()

	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	provenance := resources[0].Item.Presentation.Name.Provenance
	return schema.Resource{
		Key:  "limgrave_limgrave_west",
		Kind: schema.ResourceKindMapRegion,
		MapRegion: &schema.MapRegionDocument{
			Name: schema.Fact[string]{
				Known: true, Value: "Limgrave, West", Provenance: provenance,
			},
			AreaLabel:          schema.Fact[string]{Known: true, Value: "Limgrave", Provenance: provenance},
			VisibleEventFlagID: schema.Fact[uint32]{Known: true, Value: 62010, Provenance: provenance},
		},
	}, sources
}

func TestValidateMapRegionResourceFailsClosed(t *testing.T) {
	complete, sources := mapRegionFixture(t)
	if err := schema.ValidateResource(complete, sources); err != nil {
		t.Fatalf("ValidateResource on a complete map region: %v", err)
	}

	for name, break_ := range map[string]func(*schema.Resource){
		"missing document": func(resource *schema.Resource) { resource.MapRegion = nil },
		"foreign document": func(resource *schema.Resource) {
			resource.Boss = &schema.BossDocument{}
		},
		"uppercase key": func(resource *schema.Resource) { resource.Key = "Limgrave" },
		"unknown name": func(resource *schema.Resource) {
			resource.MapRegion.Name.Known = false
		},
		"empty name": func(resource *schema.Resource) { resource.MapRegion.Name.Value = "" },
		"unknown area": func(resource *schema.Resource) {
			resource.MapRegion.AreaLabel.Known = false
		},
		"empty area": func(resource *schema.Resource) { resource.MapRegion.AreaLabel.Value = "" },
		"unknown flag": func(resource *schema.Resource) {
			resource.MapRegion.VisibleEventFlagID.Known = false
		},
		"zero flag": func(resource *schema.Resource) {
			resource.MapRegion.VisibleEventFlagID.Value = 0
		},
		"block below": func(resource *schema.Resource) {
			resource.MapRegion.VisibleEventFlagID.Value = 61999
		},
		"acquired block": func(resource *schema.Resource) {
			resource.MapRegion.VisibleEventFlagID.Value = 63010
		},
		"system block": func(resource *schema.Resource) {
			resource.MapRegion.VisibleEventFlagID.Value = 82001
		},
	} {
		resource, _ := mapRegionFixture(t)
		break_(&resource)
		if err := schema.ValidateResource(resource, sources); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}
