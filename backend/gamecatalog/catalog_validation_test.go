package gamecatalog_test

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
)

func TestCatalogRejectsDuplicateResourceID(t *testing.T) {
	manifest, resources := prototype.Data()
	resources = append(resources, resources[0])

	_, err := gamecatalog.New(manifest, resources)
	if err == nil || !strings.Contains(err.Error(), "duplicate resource ID") {
		t.Fatalf("New error = %v, want duplicate resource ID", err)
	}
}

func TestCatalogRejectsVariantOwnedByAnotherResource(t *testing.T) {
	manifest, resources := prototype.Data()
	duplicate := resources[1]
	item := *duplicate.Item
	duplicate.ID = 3
	duplicate.Key = "item:variant-collision"
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
