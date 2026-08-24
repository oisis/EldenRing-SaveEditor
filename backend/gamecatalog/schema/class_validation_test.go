package schema_test

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func classFixture(t *testing.T) (schema.Resource, map[schema.SourceID]struct{}) {
	t.Helper()

	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	provenance := resources[0].Item.Presentation.Name.Provenance
	return schema.Resource{
		Key:  "0",
		Kind: schema.ResourceKindClass,
		Class: &schema.ClassDocument{
			StartingClassID: schema.Fact[uint32]{Known: true, Value: 0, Provenance: provenance},
			Name:            schema.Fact[string]{Known: true, Value: "Vagabond", Provenance: provenance},
			Level:           schema.Fact[uint32]{Known: true, Value: 9, Provenance: provenance},
			Vigor:           schema.Fact[uint32]{Known: true, Value: 15, Provenance: provenance},
			Mind:            schema.Fact[uint32]{Known: true, Value: 10, Provenance: provenance},
			Endurance:       schema.Fact[uint32]{Known: true, Value: 11, Provenance: provenance},
			Strength:        schema.Fact[uint32]{Known: true, Value: 14, Provenance: provenance},
			Dexterity:       schema.Fact[uint32]{Known: true, Value: 13, Provenance: provenance},
			Intelligence:    schema.Fact[uint32]{Known: true, Value: 9, Provenance: provenance},
			Faith:           schema.Fact[uint32]{Known: true, Value: 9, Provenance: provenance},
			Arcane:          schema.Fact[uint32]{Known: true, Value: 7, Provenance: provenance},
		},
	}, sources
}

func TestValidateClassResourceFailsClosed(t *testing.T) {
	complete, sources := classFixture(t)
	if err := schema.ValidateResource(complete, sources); err != nil {
		t.Fatalf("ValidateResource on a complete class: %v", err)
	}

	for name, break_ := range map[string]func(*schema.Resource){
		"missing document": func(resource *schema.Resource) { resource.Class = nil },
		"foreign document": func(resource *schema.Resource) {
			resource.Tutorial = &schema.TutorialDocument{}
		},
		"non-digit key": func(resource *schema.Resource) { resource.Key = "a" },
		"empty key":     func(resource *schema.Resource) { resource.Key = "" },
		"multi-digit key": func(resource *schema.Resource) {
			resource.Key = "10"
			resource.Class.StartingClassID.Value = 10
		},
		"unknown id": func(resource *schema.Resource) {
			resource.Class.StartingClassID.Known = false
		},
		"key mismatch": func(resource *schema.Resource) {
			resource.Class.StartingClassID.Value = 1
		},
		"unknown name": func(resource *schema.Resource) {
			resource.Class.Name.Known = false
		},
		"empty name": func(resource *schema.Resource) { resource.Class.Name.Value = "" },
		"missing level fact provenance": func(resource *schema.Resource) {
			resource.Class.Level = schema.Fact[uint32]{Known: true, Value: 9}
		},
		"unknown level": func(resource *schema.Resource) {
			resource.Class.Level.Known = false
		},
		"zero level": func(resource *schema.Resource) {
			resource.Class.Level.Value = 0
		},
		"missing vigor fact provenance": func(resource *schema.Resource) {
			resource.Class.Vigor = schema.Fact[uint32]{Known: true, Value: 15}
		},
		"unknown vigor": func(resource *schema.Resource) {
			resource.Class.Vigor.Known = false
		},
		"zero vigor": func(resource *schema.Resource) {
			resource.Class.Vigor.Value = 0
		},
		"unknown mind": func(resource *schema.Resource) {
			resource.Class.Mind.Known = false
		},
		"zero mind": func(resource *schema.Resource) {
			resource.Class.Mind.Value = 0
		},
		"unknown endurance": func(resource *schema.Resource) {
			resource.Class.Endurance.Known = false
		},
		"zero endurance": func(resource *schema.Resource) {
			resource.Class.Endurance.Value = 0
		},
		"unknown strength": func(resource *schema.Resource) {
			resource.Class.Strength.Known = false
		},
		"zero strength": func(resource *schema.Resource) {
			resource.Class.Strength.Value = 0
		},
		"unknown dexterity": func(resource *schema.Resource) {
			resource.Class.Dexterity.Known = false
		},
		"zero dexterity": func(resource *schema.Resource) {
			resource.Class.Dexterity.Value = 0
		},
		"unknown intelligence": func(resource *schema.Resource) {
			resource.Class.Intelligence.Known = false
		},
		"zero intelligence": func(resource *schema.Resource) {
			resource.Class.Intelligence.Value = 0
		},
		"unknown faith": func(resource *schema.Resource) {
			resource.Class.Faith.Known = false
		},
		"zero faith": func(resource *schema.Resource) {
			resource.Class.Faith.Value = 0
		},
		"unknown arcane": func(resource *schema.Resource) {
			resource.Class.Arcane.Known = false
		},
		"zero arcane": func(resource *schema.Resource) {
			resource.Class.Arcane.Value = 0
		},
	} {
		resource, _ := classFixture(t)
		break_(&resource)
		if err := schema.ValidateResource(resource, sources); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

// TestValidateClassResourceEnforcesTheWritableRanges pins the ranges that make a
// class document safe to copy verbatim into a save: the level must stay inside
// 1..713 and every base attribute inside 1..99. The rejections are checked by
// message, so a class document can never be refused for the wrong field.
func TestValidateClassResourceEnforcesTheWritableRanges(t *testing.T) {
	_, sources := classFixture(t)

	for name, accepted := range map[string]func(*schema.Resource){
		"lowest legal level":     func(resource *schema.Resource) { resource.Class.Level.Value = 1 },
		"highest legal level":    func(resource *schema.Resource) { resource.Class.Level.Value = 713 },
		"lowest legal attribute": func(resource *schema.Resource) { resource.Class.Vigor.Value = 1 },
		"highest legal attribute": func(resource *schema.Resource) {
			resource.Class.Arcane.Value = 99
		},
	} {
		resource, _ := classFixture(t)
		accepted(&resource)
		if err := schema.ValidateResource(resource, sources); err != nil {
			t.Errorf("%s was rejected: %v", name, err)
		}
	}

	for name, rejected := range map[string]struct {
		break_ func(*schema.Resource)
		want   string
	}{
		"level 0": {
			func(resource *schema.Resource) { resource.Class.Level.Value = 0 },
			"class.level 0 lies outside the range 1..713",
		},
		"level 714": {
			func(resource *schema.Resource) { resource.Class.Level.Value = 714 },
			"class.level 714 lies outside the range 1..713",
		},
		"vigor 0": {
			func(resource *schema.Resource) { resource.Class.Vigor.Value = 0 },
			"class.vigor 0 lies outside the range 1..99",
		},
		"vigor 100": {
			func(resource *schema.Resource) { resource.Class.Vigor.Value = 100 },
			"class.vigor 100 lies outside the range 1..99",
		},
		"arcane 100": {
			func(resource *schema.Resource) { resource.Class.Arcane.Value = 100 },
			"class.arcane 100 lies outside the range 1..99",
		},
	} {
		resource, _ := classFixture(t)
		rejected.break_(&resource)
		err := schema.ValidateResource(resource, sources)
		if err == nil {
			t.Errorf("%s was accepted", name)
			continue
		}
		if !strings.Contains(err.Error(), rejected.want) {
			t.Errorf("%s reported %q, want it to name %q", name, err, rejected.want)
		}
	}
}
