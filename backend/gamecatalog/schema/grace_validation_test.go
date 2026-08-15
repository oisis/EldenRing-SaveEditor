package schema_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func graceFixture(t *testing.T) (schema.Resource, map[schema.SourceID]struct{}) {
	t.Helper()

	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	provenance := resources[0].Item.Presentation.Name.Provenance
	return schema.Resource{
		Key:  "weeping_peninsula_tombsward_catacombs",
		Kind: schema.ResourceKindGrace,
		Grace: &schema.GraceDocument{
			Name:             schema.Fact[string]{Known: true, Value: "Tombsward Catacombs", Provenance: provenance},
			RegionLabel:      schema.Fact[string]{Known: true, Value: "Weeping Peninsula", Provenance: provenance},
			VisitEventFlagID: schema.Fact[uint32]{Known: true, Value: 73000, Provenance: provenance},
			BossArena:        schema.Fact[bool]{Known: true, Value: false, Provenance: provenance},
			DungeonType:      schema.Fact[string]{Known: true, Value: schema.GraceDungeonTypeCatacomb, Provenance: provenance},
			DoorEventFlagID:  schema.Fact[uint32]{Known: true, Value: 1043338600, Provenance: provenance},
		},
	}, sources
}

func TestValidateGraceResourceFailsClosed(t *testing.T) {
	complete, sources := graceFixture(t)
	if err := schema.ValidateResource(complete, sources); err != nil {
		t.Fatalf("ValidateResource on a complete grace: %v", err)
	}

	// A regular grace is the other confirmed shape: no dungeon type and no door.
	regular, _ := graceFixture(t)
	regular.Grace.DungeonType.Value = schema.GraceDungeonTypeNone
	regular.Grace.DoorEventFlagID.Value = 0
	if err := schema.ValidateResource(regular, sources); err != nil {
		t.Fatalf("ValidateResource on a regular grace: %v", err)
	}

	for name, break_ := range map[string]func(*schema.Resource){
		"missing document": func(resource *schema.Resource) { resource.Grace = nil },
		"foreign document in the union": func(resource *schema.Resource) {
			resource.SummoningPool = &schema.SummoningPoolDocument{}
		},
		"unknown name": func(resource *schema.Resource) {
			resource.Grace.Name = schema.Fact[string]{Provenance: resource.Grace.Name.Provenance}
		},
		"empty region label": func(resource *schema.Resource) {
			resource.Grace.RegionLabel.Value = ""
		},
		"zero visit flag": func(resource *schema.Resource) {
			resource.Grace.VisitEventFlagID.Value = 0
		},
		// Block 75 has a bitfield position but carries no grace of the curated
		// table, so it must be rejected exactly like a far-away block.
		"visit flag in the unused block 75": func(resource *schema.Resource) {
			resource.Grace.VisitEventFlagID.Value = 75000
		},
		"visit flag outside every grace block": func(resource *schema.Resource) {
			resource.Grace.VisitEventFlagID.Value = 670130
		},
		"unknown boss arena": func(resource *schema.Resource) {
			resource.Grace.BossArena.Known = false
		},
		"unknown dungeon type": func(resource *schema.Resource) {
			resource.Grace.DungeonType.Known = false
		},
		"unsupported dungeon type": func(resource *schema.Resource) {
			resource.Grace.DungeonType.Value = "cave"
		},
		"unknown door flag": func(resource *schema.Resource) {
			resource.Grace.DoorEventFlagID.Known = false
		},
		"door flag without a dungeon type": func(resource *schema.Resource) {
			resource.Grace.DungeonType.Value = schema.GraceDungeonTypeNone
		},
	} {
		resource, _ := graceFixture(t)
		break_(&resource)
		if err := schema.ValidateResource(resource, sources); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}
