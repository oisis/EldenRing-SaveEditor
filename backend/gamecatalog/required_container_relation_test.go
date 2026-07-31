package gamecatalog_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestCatalogDerivesRequiredContainerRelation(t *testing.T) {
	manifest, resources := prototype.Data()
	resources[0].Item.Acquisition.RequiredContainerID = catalogKnownFact(
		manifest,
		prototype.DeterminationGameID,
	)

	catalog, err := gamecatalog.New(manifest, resources)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	view, ok := catalog.ItemViewByGameID(prototype.DaggerGameID)
	if !ok {
		t.Fatal("Dagger not found")
	}
	found := false
	for _, relation := range view.OutgoingRelations {
		if relation.Kind == schema.RelationRequiresContainer &&
			relation.To == resources[1].ID {
			found = true
		}
	}
	if !found {
		t.Fatal("required-container relation not derived")
	}
}

func TestRequiredContainerRelationsAreDeterministic(t *testing.T) {
	manifest, prototypeResources := prototype.Data()
	first := prototypeResources[0]
	firstItem := *first.Item
	first.Item = &firstItem
	first.Item.Acquisition.RequiredContainerID = catalogKnownFact(
		manifest,
		prototype.DeterminationGameID,
	)

	second := first
	second.ID = 3
	second.Key = "item:000F4241"
	secondItem := *first.Item
	second.Item = &secondItem
	second.Item.GameID.Value = 0x000F4241
	second.Item.Variants = nil
	second.Item.Aliases = nil

	target := prototypeResources[1]
	orders := [][]schema.Resource{
		{first, target, second},
		{second, first, target},
		{target, second, first},
	}
	var expected []schema.ResourceID
	for index, resources := range orders {
		catalog, err := gamecatalog.New(manifest, resources)
		if err != nil {
			t.Fatalf("order %d New: %v", index, err)
		}
		view, ok := catalog.ItemViewByGameID(prototype.DeterminationGameID)
		if !ok {
			t.Fatalf("order %d target not found", index)
		}
		var from []schema.ResourceID
		for _, relation := range view.IncomingRelations {
			if relation.Kind == schema.RelationRequiresContainer {
				from = append(from, relation.From)
			}
		}
		if index == 0 {
			expected = from
			continue
		}
		if !reflect.DeepEqual(from, expected) {
			t.Fatalf(
				"order %d required-container sources = %v, want %v",
				index,
				from,
				expected,
			)
		}
	}
	if !reflect.DeepEqual(expected, []schema.ResourceID{1, 3}) {
		t.Fatalf("required-container sources = %v, want [1 3]", expected)
	}
}

func TestCatalogRejectsMissingRequiredContainer(t *testing.T) {
	manifest, resources := prototype.Data()
	resources[0].Item.Acquisition.RequiredContainerID = catalogKnownFact(
		manifest,
		uint32(0x7F000001),
	)

	_, err := gamecatalog.New(manifest, resources)
	if err == nil || !strings.Contains(err.Error(), "required container item") {
		t.Fatalf("New error = %v, want missing-container rejection", err)
	}
}
