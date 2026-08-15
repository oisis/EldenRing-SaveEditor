package schema_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func tutorialFixture(t *testing.T) (schema.Resource, map[schema.SourceID]struct{}) {
	t.Helper()

	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	provenance := resources[0].Item.Presentation.Name.Provenance
	return schema.Resource{
		Key:  "2010",
		Kind: schema.ResourceKindTutorial,
		Tutorial: &schema.TutorialDocument{
			TutorialID: schema.Fact[uint32]{Known: true, Value: 2010, Provenance: provenance},
			Title:      schema.Fact[string]{Known: true, Value: "Item Crafting Menu", Provenance: provenance},
		},
	}, sources
}

func TestValidateTutorialResourceFailsClosed(t *testing.T) {
	complete, sources := tutorialFixture(t)
	if err := schema.ValidateResource(complete, sources); err != nil {
		t.Fatalf("ValidateResource on a complete tutorial: %v", err)
	}

	for name, break_ := range map[string]func(*schema.Resource){
		"missing document": func(resource *schema.Resource) { resource.Tutorial = nil },
		"foreign document": func(resource *schema.Resource) {
			resource.MapRegion = &schema.MapRegionDocument{}
		},
		"uppercase key": func(resource *schema.Resource) { resource.Key = "Tutorial" },
		"unknown id": func(resource *schema.Resource) {
			resource.Tutorial.TutorialID.Known = false
		},
		"zero id": func(resource *schema.Resource) { resource.Tutorial.TutorialID.Value = 0 },
		"key mismatch": func(resource *schema.Resource) {
			resource.Tutorial.TutorialID.Value = 2020
		},
		"unknown title": func(resource *schema.Resource) {
			resource.Tutorial.Title.Known = false
		},
		"empty title": func(resource *schema.Resource) { resource.Tutorial.Title.Value = "" },
	} {
		resource, _ := tutorialFixture(t)
		break_(&resource)
		if err := schema.ValidateResource(resource, sources); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}
