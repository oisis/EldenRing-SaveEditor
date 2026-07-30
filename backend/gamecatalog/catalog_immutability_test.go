package gamecatalog_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestCatalogQueriesReturnIndependentCopies(t *testing.T) {
	catalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}

	first, ok := catalog.ItemByGameID(prototype.DaggerGameID)
	if !ok {
		t.Fatal("Dagger not found")
	}
	first.Label.Value = "Mutated"
	first.Item.Capabilities.Infusion.Rules.AllowedAffinities[0] = schema.AffinityOccult
	first.Item.Variants[0].GameID.Value = 1

	second, ok := catalog.ItemByGameID(prototype.DaggerGameID)
	if !ok {
		t.Fatal("Dagger not found after mutation")
	}
	if second.Label.Value != "Dagger" {
		t.Errorf("catalog label was mutated to %q", second.Label.Value)
	}
	if second.Item.Capabilities.Infusion.Rules.AllowedAffinities[0] != schema.AffinityStandard {
		t.Error("catalog affinity slice was mutated")
	}
	if second.Item.Variants[0].GameID.Value != prototype.DaggerGameID {
		t.Error("catalog variant slice was mutated")
	}

	manifest := catalog.Manifest()
	manifest.Sources[0].Location = "mutated"
	if catalog.Manifest().Sources[0].Location == "mutated" {
		t.Error("catalog manifest was mutated")
	}
}
