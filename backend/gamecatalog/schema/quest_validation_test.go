package schema_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func questFixture(t *testing.T) (schema.Resource, map[schema.SourceID]struct{}) {
	t.Helper()
	manifest, _ := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	return schema.Resource{
		Key:  "brother_corhyn",
		Kind: schema.ResourceKindQuest,
		Quest: &schema.QuestDocument{
			Name: schema.Fact[string]{
				Known: true,
				Value: "Brother Corhyn",
				Provenance: schema.Provenance{
					Source: "legacy_db_data",
					Method: "curated NPC name of quest \"brother_corhyn\" from legacy QuestData",
					Table:  "QuestData",
					Row:    "Brother Corhyn",
					Field:  "map key",
				},
			},
			Steps: []schema.QuestStepDocument{
				{
					Key: "legacy_000",
					Description: schema.Fact[string]{
						Known: true,
						Value: "Initial dialogue with Brother Corhyn",
						Provenance: schema.Provenance{
							Source: "legacy_db_data",
							Method: "curated description",
							Table:  "QuestData",
							Row:    "Brother Corhyn",
							Field:  "Description",
						},
					},
					Location: schema.Fact[string]{
						Known: true,
						Value: "Roundtable Hold",
						Provenance: schema.Provenance{
							Source: "legacy_db_data",
							Method: "curated location",
							Table:  "QuestData",
							Row:    "Brother Corhyn",
							Field:  "Location",
						},
					},
					Flags: []schema.QuestFlag{
						{ID: 60841, Value: true},
						{ID: 11009456, Value: false},
					},
				},
			},
		},
	}, sources
}

func TestValidateQuestResourceFailsClosed(t *testing.T) {
	fixture, sources := questFixture(t)
	if err := schema.ValidateResource(fixture, sources); err != nil {
		t.Fatalf("ValidateResource: %v", err)
	}

	mutate := func(fn func(resource *schema.Resource)) schema.Resource {
		cloned := fixture
		quest := *fixture.Quest
		steps := append([]schema.QuestStepDocument(nil), fixture.Quest.Steps...)
		for i := range steps {
			steps[i].Flags = append([]schema.QuestFlag(nil), fixture.Quest.Steps[i].Flags...)
		}
		quest.Steps = steps
		cloned.Quest = &quest
		fn(&cloned)
		return cloned
	}

	cases := map[string]schema.Resource{
		"uppercase_key": mutate(func(r *schema.Resource) {
			r.Key = "Brother_Corhyn"
		}),
		"nil_quest_document": mutate(func(r *schema.Resource) {
			r.Quest = nil
		}),
		"unknown_name": mutate(func(r *schema.Resource) {
			r.Quest.Name.Known = false
		}),
		"empty_name": mutate(func(r *schema.Resource) {
			r.Quest.Name.Value = ""
		}),
		"empty_steps": mutate(func(r *schema.Resource) {
			r.Quest.Steps = nil
		}),
		"invalid_step_key": mutate(func(r *schema.Resource) {
			r.Quest.Steps[0].Key = "Legacy 000"
		}),
		"duplicate_step_key": mutate(func(r *schema.Resource) {
			r.Quest.Steps = append(r.Quest.Steps, r.Quest.Steps[0])
		}),
		"unknown_step_description": mutate(func(r *schema.Resource) {
			r.Quest.Steps[0].Description.Known = false
		}),
		"empty_step_description": mutate(func(r *schema.Resource) {
			r.Quest.Steps[0].Description.Value = ""
		}),
		"empty_flags": mutate(func(r *schema.Resource) {
			r.Quest.Steps[0].Flags = nil
		}),
		"zero_flag_id": mutate(func(r *schema.Resource) {
			r.Quest.Steps[0].Flags[0].ID = 0
		}),
		"duplicate_flag_id": mutate(func(r *schema.Resource) {
			r.Quest.Steps[0].Flags = append(r.Quest.Steps[0].Flags, r.Quest.Steps[0].Flags[0])
		}),
		"carries_foreign_item_document": mutate(func(r *schema.Resource) {
			_, resources := prototype.Data()
			r.Item = resources[0].Item
		}),
	}

	for name, invalid := range cases {
		t.Run(name, func(t *testing.T) {
			if err := schema.ValidateResource(invalid, sources); err == nil {
				t.Fatalf("ValidateResource accepted %q", name)
			}
		})
	}
}
