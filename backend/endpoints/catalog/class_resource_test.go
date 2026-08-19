package catalog_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const (
	classKind  = "class"
	classCount = 10
)

func TestGetResourcesReturnsClassesAsNonItems(t *testing.T) {
	cat := newStoredCatalog(t)

	result, err := catalog.GetResources(cat, classKind, "", "", "", "samurai", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(class, samurai): %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("samurai search total = %d, want 1", result.Total)
	}
	if len(result.Resources) != 1 || result.Resources[0].Name != "Samurai" || result.Resources[0].Key != "7" {
		t.Fatalf("class resource = %+v, want Samurai (key 7)", result.Resources)
	}

	all, err := catalog.GetResources(cat, classKind, "", "", "", "", 0, 0)
	if err != nil || all.Total != classCount {
		t.Fatalf("all classes total = %d, error %v", all.Total, err)
	}
}

func TestGetResourceReturnsIndependentClassDocument(t *testing.T) {
	cat := newStoredCatalog(t)

	result, err := catalog.GetResource(cat, classKind, "6")
	if err != nil {
		t.Fatalf("GetResource(class, 6): %v", err)
	}
	resource := result.Resource
	if resource.Kind != schema.ResourceKindClass || resource.Class == nil ||
		resource.Class.StartingClassID.Value != 6 ||
		resource.Class.Name.Value != "Confessor" {
		t.Fatalf("class resource = %+v", resource)
	}

	// Deep copy immutability check
	resource.Class.Name.Value = "mutated"
	again, err := catalog.GetResource(cat, classKind, "6")
	if err != nil || again.Resource.Class.Name.Value != "Confessor" {
		t.Fatalf("catalog class name = %q, error %v",
			again.Resource.Class.Name.Value, err)
	}
}
