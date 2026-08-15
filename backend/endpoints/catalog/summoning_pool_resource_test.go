package catalog_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const (
	summoningPoolKind        = "summoning_pool"
	summoningPoolGatesideKey = "stormveil_castle_gateside_chamber"
	summoningPoolGatesideNam = "Gateside Chamber"
	summoningPoolGatesideReg = "Stormveil Castle"
	summoningPoolGatesideFla = 670130
	summoningPoolCount       = 213
)

func TestGetResourcesReturnsSummoningPoolsAsNonItems(t *testing.T) {
	result, err := catalog.GetResources(
		newStoredCatalog(t), summoningPoolKind, "", "", "", "gateside", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(summoning_pool): %v", err)
	}
	if result.Total != 1 || len(result.Resources) != 1 {
		t.Fatalf("total/resources = %d/%d, want 1/1", result.Total, len(result.Resources))
	}
	entry := result.Resources[0]
	if entry.Kind != schema.ResourceKindSummoningPool || entry.Key != summoningPoolGatesideKey ||
		entry.Name != summoningPoolGatesideNam || entry.Family != "" {
		t.Fatalf("entry = %+v, want the Gateside Chamber pool without an item family", entry)
	}

	all, err := catalog.GetResources(newStoredCatalog(t), summoningPoolKind, "", "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(all summoning pools): %v", err)
	}
	if all.Total != summoningPoolCount {
		t.Fatalf("summoning pool total = %d, want %d", all.Total, summoningPoolCount)
	}
}

func TestGetResourceReturnsIndependentSummoningPoolDocument(t *testing.T) {
	result, err := catalog.GetResource(
		newStoredCatalog(t), summoningPoolKind, summoningPoolGatesideKey)
	if err != nil {
		t.Fatalf("GetResource(summoning_pool): %v", err)
	}
	resource := result.Resource
	if resource.Item != nil || resource.Colosseum != nil || resource.Region != nil ||
		resource.SummoningPool == nil {
		t.Fatalf("summoning pool union = %+v, want only a summoning pool document", resource)
	}
	document := resource.SummoningPool
	if document.Name.Value != summoningPoolGatesideNam ||
		document.RegionLabel.Value != summoningPoolGatesideReg ||
		document.ActivationEventFlagID.Value != summoningPoolGatesideFla {
		t.Fatalf("summoning pool document = %+v", document)
	}
	document.Name.Value = "mutated"
	again, err := catalog.GetResource(
		newStoredCatalog(t), summoningPoolKind, summoningPoolGatesideKey)
	if err != nil {
		t.Fatalf("GetResource(summoning_pool) again: %v", err)
	}
	if again.Resource.SummoningPool.Name.Value != summoningPoolGatesideNam {
		t.Fatalf("catalog name = %q, want %q",
			again.Resource.SummoningPool.Name.Value, summoningPoolGatesideNam)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(encoded), `"item"`) ||
		strings.Contains(string(encoded), `"colosseum"`) ||
		strings.Contains(string(encoded), `"region"`) ||
		!strings.Contains(string(encoded), `"summoningPool"`) {
		t.Fatalf("summoning pool JSON carries the wrong union fields: %s", encoded)
	}
}
