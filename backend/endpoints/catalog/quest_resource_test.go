package catalog_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const (
	questKind  = "quest"
	questKey   = "brother_corhyn"
	questName  = "Brother Corhyn"
	questCount = 36
)

func TestGetResourcesReturnsQuestsAsNonItems(t *testing.T) {
	result, err := catalog.GetResources(
		newStoredCatalog(t), questKind, "", "", "", "corhyn", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(quest): %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("quest search total = %d, want 1", result.Total)
	}
	if len(result.Resources) != 1 || result.Resources[0].Name != questName {
		t.Fatalf("quest resource = %+v, want name %q", result.Resources, questName)
	}
	all, err := catalog.GetResources(
		newStoredCatalog(t), questKind, "", "", "", "", 0, 0)
	if err != nil || all.Total != questCount {
		t.Fatalf("all quests total = %d, error %v", all.Total, err)
	}
}

func TestGetResourceReturnsIndependentQuestDocument(t *testing.T) {
	gameCatalog := newStoredCatalog(t)
	result, err := catalog.GetResource(gameCatalog, questKind, questKey)
	if err != nil {
		t.Fatalf("GetResource(quest): %v", err)
	}
	resource := result.Resource
	if resource.Kind != schema.ResourceKindQuest || resource.Quest == nil ||
		resource.Quest.Name.Value != questName || len(resource.Quest.Steps) == 0 {
		t.Fatalf("quest resource = %+v", resource)
	}

	// Mutate top-level, step, and flag fields to verify deep clone immutability.
	resource.Quest.Name.Value = "mutated"
	resource.Quest.Steps[0].Description.Value = "mutated description"
	resource.Quest.Steps[0].Flags[0].ID = 999999

	again, err := catalog.GetResource(gameCatalog, questKind, questKey)
	if err != nil {
		t.Fatalf("GetResource again: %v", err)
	}
	if again.Resource.Quest.Name.Value != questName {
		t.Fatalf("catalog quest name = %q, want %q", again.Resource.Quest.Name.Value, questName)
	}
	if again.Resource.Quest.Steps[0].Description.Value == "mutated description" {
		t.Fatal("catalog quest step description was mutated")
	}
	if again.Resource.Quest.Steps[0].Flags[0].ID == 999999 {
		t.Fatal("catalog quest step flag was mutated")
	}
}
