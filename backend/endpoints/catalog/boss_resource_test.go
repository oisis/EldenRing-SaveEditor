package catalog_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const (
	bossKind        = "boss"
	bossGodrickKey  = "stormveil_castle_godrick_the_grafted"
	bossGodrickName = "Godrick the Grafted"
	bossGodrickRegi = "Stormveil Castle"
	bossGodrickFlag = 9101
	bossCount       = 110
)

func TestGetResourcesReturnsBossesAsNonItems(t *testing.T) {
	result, err := catalog.GetResources(
		newStoredCatalog(t), bossKind, "", "", "", "godrick_the_grafted", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(boss): %v", err)
	}
	if result.Total != 1 || len(result.Resources) != 1 {
		t.Fatalf("total/resources = %d/%d, want 1/1", result.Total, len(result.Resources))
	}
	entry := result.Resources[0]
	if entry.Kind != schema.ResourceKindBoss || entry.Key != bossGodrickKey ||
		entry.Name != bossGodrickName || entry.Family != "" {
		t.Fatalf("entry = %+v, want the Godrick boss without an item family", entry)
	}

	all, err := catalog.GetResources(newStoredCatalog(t), bossKind, "", "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(all bosses): %v", err)
	}
	if all.Total != bossCount {
		t.Fatalf("boss total = %d, want %d", all.Total, bossCount)
	}
}

func TestGetResourceReturnsIndependentBossDocument(t *testing.T) {
	result, err := catalog.GetResource(newStoredCatalog(t), bossKind, bossGodrickKey)
	if err != nil {
		t.Fatalf("GetResource(boss): %v", err)
	}
	resource := result.Resource
	if resource.Item != nil || resource.Colosseum != nil || resource.Region != nil ||
		resource.SummoningPool != nil || resource.Grace != nil || resource.Boss == nil {
		t.Fatalf("boss union = %+v, want only a boss document", resource)
	}
	document := resource.Boss
	if document.Name.Value != bossGodrickName ||
		document.RegionLabel.Value != bossGodrickRegi ||
		document.EncounterType.Value != schema.BossEncounterTypeMain ||
		!document.Remembrance.Value ||
		document.DefeatEventFlagID.Value != bossGodrickFlag {
		t.Fatalf("boss document = %+v", document)
	}
	document.Name.Value = "mutated"
	again, err := catalog.GetResource(newStoredCatalog(t), bossKind, bossGodrickKey)
	if err != nil {
		t.Fatalf("GetResource(boss) again: %v", err)
	}
	if again.Resource.Boss.Name.Value != bossGodrickName {
		t.Fatalf("catalog name = %q, want %q", again.Resource.Boss.Name.Value, bossGodrickName)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(encoded), `"item"`) ||
		strings.Contains(string(encoded), `"colosseum"`) ||
		strings.Contains(string(encoded), `"region"`) ||
		strings.Contains(string(encoded), `"summoningPool"`) ||
		strings.Contains(string(encoded), `"grace"`) ||
		!strings.Contains(string(encoded), `"boss"`) {
		t.Fatalf("boss JSON carries the wrong union fields: %s", encoded)
	}
}
