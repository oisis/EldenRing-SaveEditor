package gamecatalog_test

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestCatalogRejectsDuplicateResourceRef(t *testing.T) {
	manifest, resources := prototype.Data()
	resources = append(resources, resources[0])

	_, err := gamecatalog.New(manifest, resources)
	if err == nil || !strings.Contains(err.Error(), "duplicate resource") {
		t.Fatalf("New error = %v, want duplicate (kind, key) rejection", err)
	}
}

func TestCatalogRejectsVariantOwnedByAnotherResource(t *testing.T) {
	manifest, resources := prototype.Data()
	duplicate := resources[1]
	item := *duplicate.Item
	// The key must stay a well-formed item key and match the game ID below.
	duplicate.Key = "000F42A4"
	item.GameID.Value = prototype.DaggerGameID + 100
	duplicate.Item = &item
	resources = append(resources, duplicate)

	_, err := gamecatalog.New(manifest, resources)
	if err == nil || !strings.Contains(err.Error(), "duplicate item game ID") {
		t.Fatalf("New error = %v, want duplicate item game ID", err)
	}
}

func TestCatalogLookupUnknownItem(t *testing.T) {
	catalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}
	if _, ok := catalog.ItemByGameID(0xDEADBEEF); ok {
		t.Fatal("unknown item unexpectedly resolved")
	}
	if _, ok := catalog.ItemViewByGameID(0xDEADBEEF); ok {
		t.Fatal("unknown item view unexpectedly resolved")
	}
}

func TestCatalogIndexesTechnicalAlias(t *testing.T) {
	manifest, resources := prototype.Data()
	aliasID := uint32(0x7F000001)
	resources[0].Item.Aliases = []schema.ItemAlias{{
		GameID: catalogKnownFact(manifest, aliasID),
	}}

	catalog, err := gamecatalog.New(manifest, resources)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	resource, ok := catalog.ItemByGameID(aliasID)
	if !ok || resource.Ref() != resources[0].Ref() {
		t.Fatalf(
			"ItemByGameID(alias) = (kind %q key %q, %t), want kind %q key %q",
			resource.Kind,
			resource.Key,
			ok,
			resources[0].Kind,
			resources[0].Key,
		)
	}
}

func TestCatalogRejectsAliasOwnedByAnotherResource(t *testing.T) {
	manifest, resources := prototype.Data()
	resources[1].Item.Aliases = []schema.ItemAlias{{
		GameID: catalogKnownFact(manifest, prototype.DaggerGameID),
	}}

	_, err := gamecatalog.New(manifest, resources)
	if err == nil || !strings.Contains(err.Error(), "alias game ID") {
		t.Fatalf("New error = %v, want alias collision", err)
	}
}

func TestCatalogDoesNotIndexRelatedTechnicalRecordAsItem(t *testing.T) {
	manifest, resources := prototype.Data()
	technicalID := uint32(0x400000B6)
	p := schema.Provenance{Source: manifest.Sources[0].ID, Method: "test fixture"}
	resources[0].Item.RelatedTechnicalRecords = []schema.RelatedTechnicalRecord{{
		Kind:             catalogKnownFact(manifest, schema.TechnicalRecordAppearanceState),
		GameID:           catalogKnownFact(manifest, technicalID),
		GameMaxInventory: catalogKnownFact(manifest, uint32(999)),
		GameMaxStorage:   catalogKnownFact(manifest, uint32(999)),
		SourceRecords: []schema.ParameterRecord{{
			Table: "EquipParamGoods", RowID: 182, Provenance: p,
			Fields: []schema.ParameterField{{Name: "Row ID"}},
		}},
	}}

	catalog, err := gamecatalog.New(manifest, resources)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, exists := catalog.ItemByGameID(technicalID); exists {
		t.Fatal("related technical record was indexed as a writable item")
	}
}

func catalogKnownFact[T any](manifest schema.Manifest, value T) schema.Fact[T] {
	return schema.Fact[T]{
		Known: true,
		Value: value,
		Provenance: schema.Provenance{
			Source: manifest.Sources[0].ID,
			Method: "test fixture",
		},
	}
}
