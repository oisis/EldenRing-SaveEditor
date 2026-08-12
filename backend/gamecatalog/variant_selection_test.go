package gamecatalog_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const (
	// daggerKey and determinationKey are the catalog keys of the two prototype
	// resources: the Dagger carries variants, Determination carries none.
	daggerKey        = "000F4240"
	determinationKey = "8000EA60"
	// heavyDaggerVariantID is the first stored Dagger variant (affinity heavy).
	heavyDaggerVariantID uint32 = 0x000F42A4
	// daggerAliasID is a technical alias of the Dagger, never a variant.
	daggerAliasID uint32 = 0x7F000001
)

// variantSelectionCatalog builds the prototype catalog with one Dagger alias
// and with the first Dagger variant carrying its own storage, capability and
// safety values, so a materialised variant is distinguishable from the base.
func variantSelectionCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()
	manifest, resources := prototype.Data()
	dagger := resources[0]
	if dagger.Key != daggerKey || dagger.Item == nil || len(dagger.Item.Variants) == 0 {
		t.Fatalf("Dagger fixture = kind %q key %q, want a key %q resource with variants",
			dagger.Kind, dagger.Key, daggerKey)
	}
	dagger.Item.Aliases = []schema.ItemAlias{{
		GameID: catalogKnownFact(manifest, daggerAliasID),
	}}
	variant := &dagger.Item.Variants[0]
	if variant.GameID.Value != heavyDaggerVariantID {
		t.Fatalf("first Dagger variant = 0x%08X, want 0x%08X",
			variant.GameID.Value, heavyDaggerVariantID)
	}
	variant.Data.Storage.MaxInventory = catalogKnownFact(manifest, uint32(7))
	variant.Data.Safety.CutContent = catalogKnownFact(manifest, true)
	variant.Data.Capabilities.Upgrade.Rules.MaxLevel = 3

	catalog, err := gamecatalog.New(manifest, resources)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return catalog
}

func TestResourceByKindKeyAndVariantSelectsBaseDocument(t *testing.T) {
	catalog := variantSelectionCatalog(t)

	resource, err := catalog.ResourceByKindKeyAndVariant(schema.ResourceKindItem, daggerKey, nil)
	if err != nil {
		t.Fatalf("ResourceByKindKeyAndVariant(nil variant): %v", err)
	}
	if resource.Kind != schema.ResourceKindItem || resource.Key != daggerKey {
		t.Fatalf("identity = (kind %q, key %q), want (%q, %q)",
			resource.Kind, resource.Key, schema.ResourceKindItem, daggerKey)
	}
	if resource.Item.GameID.Value != prototype.DaggerGameID {
		t.Errorf("game ID = 0x%08X, want base 0x%08X",
			resource.Item.GameID.Value, prototype.DaggerGameID)
	}
	if resource.Item.Storage.MaxInventory.Value != 1 {
		t.Errorf("base max inventory = %d, want 1", resource.Item.Storage.MaxInventory.Value)
	}
	if resource.Item.Safety.CutContent.Value {
		t.Error("base cut content flag was taken from a variant")
	}
	if got := resource.Item.Capabilities.Upgrade.Rules.MaxLevel; got != 25 {
		t.Errorf("base max upgrade level = %d, want 25", got)
	}
	if resource.Item.Weapon == nil || resource.Item.Weapon.AttackPhysical.Value != 74 {
		t.Errorf("base weapon data = %+v, want physical attack 74", resource.Item.Weapon)
	}
}

func TestResourceByKindKeyAndVariantMaterializesTheExactVariant(t *testing.T) {
	catalog := variantSelectionCatalog(t)
	variantID := heavyDaggerVariantID

	resource, err := catalog.ResourceByKindKeyAndVariant(schema.ResourceKindItem, daggerKey, &variantID)
	if err != nil {
		t.Fatalf("ResourceByKindKeyAndVariant(0x%08X): %v", variantID, err)
	}
	if resource.Kind != schema.ResourceKindItem || resource.Key != daggerKey {
		t.Fatalf("identity = (kind %q, key %q), want the resolved pair (%q, %q)",
			resource.Kind, resource.Key, schema.ResourceKindItem, daggerKey)
	}
	if resource.Item.GameID.Value != variantID {
		t.Errorf("game ID = 0x%08X, want the variant 0x%08X", resource.Item.GameID.Value, variantID)
	}
	// The variant document data must replace the base data field by field.
	if got := resource.Item.Storage.MaxInventory.Value; got != 7 {
		t.Errorf("max inventory = %d, want the variant value 7", got)
	}
	if !resource.Item.Safety.CutContent.Value {
		t.Error("cut content flag = false, want the variant value true")
	}
	if got := resource.Item.Capabilities.Upgrade.Rules.MaxLevel; got != 3 {
		t.Errorf("max upgrade level = %d, want the variant value 3", got)
	}
	if resource.Item.Weapon == nil || resource.Item.Weapon.AttackPhysical.Value != 71 {
		t.Errorf("weapon data = %+v, want the variant physical attack 71", resource.Item.Weapon)
	}
	if resource.Item.Family.Value != schema.ItemFamilyWeapon {
		t.Errorf("family = %q, want %q", resource.Item.Family.Value, schema.ItemFamilyWeapon)
	}
}

func TestResourceByKindKeyAndVariantRejectsIDsThatAreNotVariantsOfThePair(t *testing.T) {
	catalog := variantSelectionCatalog(t)
	cases := []struct {
		name      string
		key       string
		variantID uint32
	}{
		{"unknown ID", daggerKey, 0xDEADBEEF},
		{"base game ID", daggerKey, prototype.DaggerGameID},
		{"alias game ID", daggerKey, daggerAliasID},
		{"variant of another resource", determinationKey, heavyDaggerVariantID},
		{"zero", daggerKey, 0},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			variantID := testCase.variantID
			_, err := catalog.ResourceByKindKeyAndVariant(schema.ResourceKindItem, testCase.key, &variantID)
			if err == nil {
				t.Fatalf("variant 0x%08X on key %q resolved, want an unknown variant error",
					variantID, testCase.key)
			}
			message := err.Error()
			formattedVariantID := fmt.Sprintf("0x%08X", variantID)
			if !strings.Contains(message, "unknown variant ID") ||
				!strings.Contains(message, formattedVariantID) ||
				!strings.Contains(message, testCase.key) ||
				!strings.Contains(message, string(schema.ResourceKindItem)) {
				t.Fatalf("error = %q, want the unknown variant ID %q, kind and key",
					message, formattedVariantID)
			}
		})
	}
}

// TestResourceByKindKeyAndVariantMatchesKindAndKeyExactly also pins the two
// rejections apart: an unsupported kind and an unknown key stay distinct
// errors, so a normalised spelling never degrades into the other one.
func TestResourceByKindKeyAndVariantMatchesKindAndKeyExactly(t *testing.T) {
	catalog := variantSelectionCatalog(t)
	cases := []struct {
		name    string
		kind    schema.ResourceKind
		key     string
		wantErr string
	}{
		{"padded key", schema.ResourceKindItem, " 000F4240", "unknown resource key"},
		{"lowercase key", schema.ResourceKindItem, "000f4240", "unknown resource key"},
		{"pre-migration key", schema.ResourceKindItem, "item:000F4240", "unknown resource key"},
		{"unknown key", schema.ResourceKindItem, "0000DEAD", "unknown resource key"},
		{"uppercase kind", "Item", daggerKey, "only kind"},
		{"padded kind", " item", daggerKey, "only kind"},
		{"unsupported kind", "relation", daggerKey, "only kind"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := catalog.ResourceByKindKeyAndVariant(testCase.kind, testCase.key, nil)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("kind %q key %q error = %v, want %q",
					testCase.kind, testCase.key, err, testCase.wantErr)
			}
		})
	}
}

func TestResourceByKindKeyAndVariantRequiresCatalogAndArguments(t *testing.T) {
	var missing *gamecatalog.Catalog
	if _, err := missing.ResourceByKindKeyAndVariant(schema.ResourceKindItem, daggerKey, nil); err == nil ||
		!strings.Contains(err.Error(), "game catalog is not loaded") {
		t.Fatalf("missing catalog error = %v, want a not loaded error", err)
	}

	catalog := variantSelectionCatalog(t)
	if _, err := catalog.ResourceByKindKeyAndVariant("", daggerKey, nil); err == nil ||
		!strings.Contains(err.Error(), "resource kind is required") {
		t.Fatalf("missing kind error = %v, want a required kind error", err)
	}
	if _, err := catalog.ResourceByKindKeyAndVariant(schema.ResourceKindItem, "", nil); err == nil ||
		!strings.Contains(err.Error(), "resource key is required") {
		t.Fatalf("missing key error = %v, want a required key error", err)
	}
}

func TestResourceByKindKeyAndVariantReturnsIndependentCopies(t *testing.T) {
	catalog := variantSelectionCatalog(t)
	variantID := heavyDaggerVariantID

	selected, err := catalog.ResourceByKindKeyAndVariant(schema.ResourceKindItem, daggerKey, &variantID)
	if err != nil {
		t.Fatalf("ResourceByKindKeyAndVariant(0x%08X): %v", variantID, err)
	}
	selected.Item.Presentation.Name.Value = "Mutated"
	selected.Item.Storage.MaxInventory.Value = 99
	selected.Item.Capabilities.Upgrade.Rules.MaxLevel = 99
	selected.Item.Weapon.AttackPhysical.Value = 99
	selected.Item.Variants[0].Data.Storage.MaxInventory.Value = 99

	again, err := catalog.ResourceByKindKeyAndVariant(schema.ResourceKindItem, daggerKey, &variantID)
	if err != nil {
		t.Fatalf("ResourceByKindKeyAndVariant(0x%08X) after mutation: %v", variantID, err)
	}
	if again.Item.Presentation.Name.Value == "Mutated" ||
		again.Item.Storage.MaxInventory.Value != 7 ||
		again.Item.Capabilities.Upgrade.Rules.MaxLevel != 3 ||
		again.Item.Weapon.AttackPhysical.Value != 71 ||
		again.Item.Variants[0].Data.Storage.MaxInventory.Value != 7 {
		t.Fatal("mutating the selected variant changed the catalog")
	}

	base, err := catalog.ResourceByKindKeyAndVariant(schema.ResourceKindItem, daggerKey, nil)
	if err != nil {
		t.Fatalf("ResourceByKindKeyAndVariant(nil variant) after mutation: %v", err)
	}
	if base.Item.Storage.MaxInventory.Value != 1 ||
		base.Item.Capabilities.Upgrade.Rules.MaxLevel != 25 ||
		base.Item.Weapon.AttackPhysical.Value != 74 {
		t.Fatal("mutating the selected variant changed the stored base document")
	}
}
