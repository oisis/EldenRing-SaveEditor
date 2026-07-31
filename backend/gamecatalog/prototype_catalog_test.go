package gamecatalog_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestPrototypeCatalogContainsTwoItems(t *testing.T) {
	catalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}
	if got := catalog.ResourceCount(); got != 2 {
		t.Fatalf("ResourceCount = %d, want 2", got)
	}

	dagger, ok := catalog.ItemByGameID(prototype.DaggerGameID)
	if !ok {
		t.Fatal("Dagger not found")
	}
	if dagger.Label.Value != "Dagger" {
		t.Errorf("Dagger label = %q", dagger.Label.Value)
	}
	if dagger.Item.Family.Value != schema.ItemFamilyWeapon {
		t.Errorf("Dagger family = %q", dagger.Item.Family.Value)
	}
	if !dagger.Item.Capabilities.Upgrade.Known || !dagger.Item.Capabilities.Upgrade.Enabled {
		t.Fatal("Dagger upgrade capability is not known and enabled")
	}
	if got := dagger.Item.Capabilities.Upgrade.Rules.MaxLevel; got != 25 {
		t.Errorf("Dagger max upgrade = %d, want 25", got)
	}
	if !dagger.Item.Capabilities.Infusion.Enabled {
		t.Fatal("Dagger infusion capability is disabled")
	}
	if got := len(dagger.Item.Capabilities.Infusion.Rules.AllowedAffinities); got != 13 {
		t.Errorf("Dagger affinities = %d, want 13", got)
	}
	if !dagger.Item.Storage.MaxInventory.Known ||
		dagger.Item.Storage.MaxInventory.Value != 1 ||
		!dagger.Item.Storage.MaxStorage.Known ||
		dagger.Item.Storage.MaxStorage.Value != 1 {
		t.Errorf("Dagger effective storage limits = %+v", dagger.Item.Storage)
	}
	if dagger.Item.Storage.GameMaxInventory.Known ||
		dagger.Item.Storage.GameMaxStorage.Known {
		t.Error("Regulation zero sentinels must not become known game limits")
	}

	determination, ok := catalog.ItemByGameID(prototype.DeterminationGameID)
	if !ok {
		t.Fatal("Determination not found")
	}
	if determination.Item.Family.Value != schema.ItemFamilyAshOfWar {
		t.Errorf("Determination family = %q", determination.Item.Family.Value)
	}
	if got := determination.Item.AshOfWar.CompatibilityMask.Value; got != 0xF4000FEFFFF {
		t.Errorf("Determination compatibility mask = 0x%X", got)
	}
	if determination.Item.Capabilities.Upgrade.Enabled {
		t.Error("Determination upgrade capability must be disabled")
	}
}

func TestPrototypeCatalogIndexesWeaponVariants(t *testing.T) {
	catalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}

	qualityDaggerID := prototype.DaggerGameID + 300
	resource, ok := catalog.ItemByGameID(qualityDaggerID)
	if !ok {
		t.Fatalf("Quality Dagger 0x%08X not found", qualityDaggerID)
	}
	if resource.Item.GameID.Value != qualityDaggerID {
		t.Errorf("variant resolved to game ID 0x%08X", resource.Item.GameID.Value)
	}
}

func TestPrototypeCatalogDerivesCompatibilityRelation(t *testing.T) {
	catalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}

	dagger, ok := catalog.ItemViewByGameID(prototype.DaggerGameID)
	if !ok {
		t.Fatal("Dagger view not found")
	}
	if len(dagger.OutgoingRelations) != 1 {
		t.Fatalf("Dagger outgoing relations = %d, want 1", len(dagger.OutgoingRelations))
	}
	if dagger.OutgoingRelations[0].Kind != schema.RelationCompatibleWithAshOfWar {
		t.Errorf("relation kind = %q", dagger.OutgoingRelations[0].Kind)
	}
	if len(dagger.RelatedResources) != 1 ||
		dagger.RelatedResources[0].Item.GameID.Value != prototype.DeterminationGameID {
		t.Fatalf("Dagger related resources = %+v", dagger.RelatedResources)
	}

	determination, ok := catalog.ItemViewByGameID(prototype.DeterminationGameID)
	if !ok {
		t.Fatal("Determination view not found")
	}
	if len(determination.IncomingRelations) != 1 {
		t.Fatalf("Determination incoming relations = %d, want 1", len(determination.IncomingRelations))
	}
	if len(determination.RelatedResources) != 1 ||
		determination.RelatedResources[0].Item.GameID.Value != prototype.DaggerGameID {
		t.Fatalf("Determination related resources = %+v", determination.RelatedResources)
	}
}

func TestUnknownCompatibilityFailsClosedWithoutRejectingCatalog(t *testing.T) {
	manifest, resources := prototype.Data()
	resources[1].Item.AshOfWar.CompatibilityMask.Known = false
	resources[1].Item.AshOfWar.CompatibilityMask.Value = 0

	catalog, err := gamecatalog.New(manifest, resources)
	if err != nil {
		t.Fatalf("New with unknown compatibility: %v", err)
	}
	view, ok := catalog.ItemViewByGameID(prototype.DaggerGameID)
	if !ok {
		t.Fatal("Dagger view not found")
	}
	if len(view.OutgoingRelations) != 0 {
		t.Fatalf("unknown compatibility produced %d relation(s)", len(view.OutgoingRelations))
	}
}
