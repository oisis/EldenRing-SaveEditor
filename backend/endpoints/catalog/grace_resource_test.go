package catalog_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const (
	graceKind        = "grace"
	graceTombswardKe = "weeping_peninsula_tombsward_catacombs"
	graceTombswardNa = "Tombsward Catacombs"
	graceTombswardRe = "Weeping Peninsula"
	graceTombswardFl = 73000
	graceTombswardDo = 1043338600
	graceCount       = 419
)

func TestGetResourcesReturnsGracesAsNonItems(t *testing.T) {
	result, err := catalog.GetResources(
		newStoredCatalog(t), graceKind, "", "", "", "tombsward_catacombs", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(grace): %v", err)
	}
	if result.Total != 1 || len(result.Resources) != 1 {
		t.Fatalf("total/resources = %d/%d, want 1/1", result.Total, len(result.Resources))
	}
	entry := result.Resources[0]
	if entry.Kind != schema.ResourceKindGrace || entry.Key != graceTombswardKe ||
		entry.Name != graceTombswardNa || entry.Family != "" {
		t.Fatalf("entry = %+v, want the Tombsward Catacombs grace without an item family", entry)
	}

	all, err := catalog.GetResources(newStoredCatalog(t), graceKind, "", "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(all graces): %v", err)
	}
	if all.Total != graceCount {
		t.Fatalf("grace total = %d, want %d", all.Total, graceCount)
	}
}

func TestGetResourceReturnsIndependentGraceDocument(t *testing.T) {
	result, err := catalog.GetResource(newStoredCatalog(t), graceKind, graceTombswardKe)
	if err != nil {
		t.Fatalf("GetResource(grace): %v", err)
	}
	resource := result.Resource
	if resource.Item != nil || resource.Colosseum != nil || resource.Region != nil ||
		resource.SummoningPool != nil || resource.Grace == nil {
		t.Fatalf("grace union = %+v, want only a grace document", resource)
	}
	document := resource.Grace
	if document.Name.Value != graceTombswardNa ||
		document.RegionLabel.Value != graceTombswardRe ||
		document.VisitEventFlagID.Value != graceTombswardFl ||
		document.BossArena.Value ||
		document.DungeonType.Value != schema.GraceDungeonTypeCatacomb ||
		document.DoorEventFlagID.Value != graceTombswardDo {
		t.Fatalf("grace document = %+v", document)
	}
	document.Name.Value = "mutated"
	again, err := catalog.GetResource(newStoredCatalog(t), graceKind, graceTombswardKe)
	if err != nil {
		t.Fatalf("GetResource(grace) again: %v", err)
	}
	if again.Resource.Grace.Name.Value != graceTombswardNa {
		t.Fatalf("catalog name = %q, want %q", again.Resource.Grace.Name.Value, graceTombswardNa)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(encoded), `"item"`) ||
		strings.Contains(string(encoded), `"colosseum"`) ||
		strings.Contains(string(encoded), `"region"`) ||
		strings.Contains(string(encoded), `"summoningPool"`) ||
		!strings.Contains(string(encoded), `"grace"`) {
		t.Fatalf("grace JSON carries the wrong union fields: %s", encoded)
	}
}
