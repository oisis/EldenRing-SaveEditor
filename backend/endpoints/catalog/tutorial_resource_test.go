package catalog_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const (
	tutorialKind  = "tutorial"
	tutorialKey   = "2010"
	tutorialTitle = "Item Crafting Menu"
	tutorialCount = 72
)

func TestGetResourcesReturnsTutorialsAsNonItems(t *testing.T) {
	result, err := catalog.GetResources(
		newStoredCatalog(t), tutorialKind, "", "", "", "item crafting", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(tutorial): %v", err)
	}
	if result.Total != 2 {
		// Item Crafting and Item Crafting Menu are two distinct official titles.
		t.Fatalf("tutorial search total = %d, want 2", result.Total)
	}
	all, err := catalog.GetResources(
		newStoredCatalog(t), tutorialKind, "", "", "", "", 0, 0)
	if err != nil || all.Total != tutorialCount {
		t.Fatalf("all tutorials total = %d, error %v", all.Total, err)
	}
}

func TestGetResourceReturnsIndependentTutorialDocument(t *testing.T) {
	gameCatalog := newStoredCatalog(t)
	result, err := catalog.GetResource(gameCatalog, tutorialKind, tutorialKey)
	if err != nil {
		t.Fatalf("GetResource(tutorial): %v", err)
	}
	resource := result.Resource
	if resource.Kind != schema.ResourceKindTutorial || resource.Tutorial == nil ||
		resource.Tutorial.TutorialID.Value != 2010 ||
		resource.Tutorial.Title.Value != tutorialTitle {
		t.Fatalf("tutorial resource = %+v", resource)
	}
	resource.Tutorial.Title.Value = "mutated"
	again, err := catalog.GetResource(gameCatalog, tutorialKind, tutorialKey)
	if err != nil || again.Resource.Tutorial.Title.Value != tutorialTitle {
		t.Fatalf("catalog tutorial title = %q, error %v",
			again.Resource.Tutorial.Title.Value, err)
	}
}
