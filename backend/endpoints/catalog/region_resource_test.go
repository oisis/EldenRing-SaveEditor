package catalog_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const (
	regionKind          = "region"
	regionFirstStepKey  = "limgrave_the_first_step"
	regionFirstStepName = "The First Step"
	regionFirstStepArea = "Limgrave"
	regionFirstStepID   = 6100000
	regionCount         = 274
)

func TestGetResourcesReturnsRegionsAsNonItems(t *testing.T) {
	result, err := catalog.GetResources(
		newStoredCatalog(t), regionKind, "", "", "", "first step", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(region): %v", err)
	}
	if result.Total != 1 || len(result.Resources) != 1 {
		t.Fatalf("total/resources = %d/%d, want 1/1", result.Total, len(result.Resources))
	}
	entry := result.Resources[0]
	if entry.Kind != schema.ResourceKindRegion || entry.Key != regionFirstStepKey ||
		entry.Name != regionFirstStepName || entry.Family != "" {
		t.Fatalf("entry = %+v, want The First Step region without an item family", entry)
	}

	all, err := catalog.GetResources(newStoredCatalog(t), regionKind, "", "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(all regions): %v", err)
	}
	if all.Total != regionCount {
		t.Fatalf("region total = %d, want %d", all.Total, regionCount)
	}
}

func TestGetResourceReturnsIndependentRegionDocument(t *testing.T) {
	result, err := catalog.GetResource(newStoredCatalog(t), regionKind, regionFirstStepKey)
	if err != nil {
		t.Fatalf("GetResource(region): %v", err)
	}
	resource := result.Resource
	if resource.Item != nil || resource.Colosseum != nil || resource.Region == nil {
		t.Fatalf("region union = %+v, want only a region document", resource)
	}
	if resource.Region.RegionID.Value != regionFirstStepID ||
		resource.Region.Name.Value != regionFirstStepName ||
		resource.Region.Area.Value != regionFirstStepArea {
		t.Fatalf("region document = %+v", resource.Region)
	}
	resource.Region.Name.Value = "mutated"
	again, err := catalog.GetResource(newStoredCatalog(t), regionKind, regionFirstStepKey)
	if err != nil {
		t.Fatalf("GetResource(region) again: %v", err)
	}
	if again.Resource.Region.Name.Value != regionFirstStepName {
		t.Fatalf("catalog name = %q, want %q",
			again.Resource.Region.Name.Value, regionFirstStepName)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(encoded), `"item"`) ||
		strings.Contains(string(encoded), `"colosseum"`) ||
		!strings.Contains(string(encoded), `"region"`) {
		t.Fatalf("region JSON carries the wrong union fields: %s", encoded)
	}
}
