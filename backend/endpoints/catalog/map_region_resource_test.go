package catalog_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const (
	mapRegionKind     = "map_region"
	mapRegionWestKey  = "limgrave_limgrave_west"
	mapRegionWestName = "Limgrave, West"
	mapRegionWestArea = "Limgrave"
	mapRegionWestFlag = 62010
	mapRegionCount    = 263
)

func TestGetResourcesReturnsMapRegionsAsNonItems(t *testing.T) {
	result, err := catalog.GetResources(
		newStoredCatalog(t), mapRegionKind, "", "", "", "limgrave_west", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(map_region): %v", err)
	}
	if result.Total != 1 || len(result.Resources) != 1 {
		t.Fatalf("total/resources = %d/%d, want 1/1", result.Total, len(result.Resources))
	}
	entry := result.Resources[0]
	if entry.Kind != schema.ResourceKindMapRegion || entry.Key != mapRegionWestKey ||
		entry.Name != mapRegionWestName || entry.Family != "" {
		t.Fatalf("entry = %+v, want Limgrave West without an item family", entry)
	}

	all, err := catalog.GetResources(
		newStoredCatalog(t), mapRegionKind, "", "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(all map regions): %v", err)
	}
	if all.Total != mapRegionCount {
		t.Fatalf("map region total = %d, want %d", all.Total, mapRegionCount)
	}
}

func TestGetResourceReturnsIndependentMapRegionDocument(t *testing.T) {
	result, err := catalog.GetResource(newStoredCatalog(t), mapRegionKind, mapRegionWestKey)
	if err != nil {
		t.Fatalf("GetResource(map_region): %v", err)
	}
	resource := result.Resource
	if resource.Item != nil || resource.Colosseum != nil || resource.Region != nil ||
		resource.SummoningPool != nil || resource.Grace != nil || resource.Boss != nil ||
		resource.MapRegion == nil {
		t.Fatalf("map region union = %+v, want only a map region document", resource)
	}
	document := resource.MapRegion
	if document.Name.Value != mapRegionWestName ||
		document.AreaLabel.Value != mapRegionWestArea ||
		document.VisibleEventFlagID.Value != mapRegionWestFlag {
		t.Fatalf("map region document = %+v", document)
	}
	document.Name.Value = "mutated"
	again, err := catalog.GetResource(newStoredCatalog(t), mapRegionKind, mapRegionWestKey)
	if err != nil {
		t.Fatalf("GetResource(map_region) again: %v", err)
	}
	if again.Resource.MapRegion.Name.Value != mapRegionWestName {
		t.Fatalf("catalog name = %q, want %q",
			again.Resource.MapRegion.Name.Value, mapRegionWestName)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(encoded), `"item"`) ||
		strings.Contains(string(encoded), `"colosseum"`) ||
		strings.Contains(string(encoded), `"summoningPool"`) ||
		strings.Contains(string(encoded), `"grace"`) ||
		strings.Contains(string(encoded), `"boss"`) ||
		!strings.Contains(string(encoded), `"mapRegion"`) {
		t.Fatalf("map region JSON carries the wrong union fields: %s", encoded)
	}
}
