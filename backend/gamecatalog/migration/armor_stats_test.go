package migration

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestGenerateBuildsAllArmorStatsFromRegulation(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	count := 0
	for _, resource := range catalog.Resources {
		if resource.Item == nil ||
			resource.Item.Family.Value != schema.ItemFamilyArmor {
			continue
		}
		count++
		assertArmorStatsKnownFromRegulation(t, resource.Item.Armor)
	}
	if count != 723 {
		t.Fatalf("armor documents = %d, want 723", count)
	}

	danesHat := findGeneratedItem(t, catalog, 0x102DC6C0)
	assertArmorValues(
		t,
		danesHat.Armor,
		2.2,
		1.8,
		16,
		3,
	)
	bullGoatArmor := findGeneratedItem(t, catalog, 0x10022344)
	assertArmorValues(
		t,
		bullGoatArmor.Armor,
		26.5,
		20.4,
		71,
		47,
	)
}

func TestGenerateDoesNotPreserveUnpopulatedLegacyPoise(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, resource := range catalog.Resources {
		if resource.Item == nil ||
			resource.Item.Family.Value != schema.ItemFamilyArmor {
			continue
		}
		if resource.Item.Armor.PoiseSFV != nil {
			t.Fatalf(
				"armor %s poise-sfv = %#v, want nil",
				resource.Key,
				resource.Item.Armor.PoiseSFV,
			)
		}
	}
}

func assertArmorStatsKnownFromRegulation(
	t *testing.T,
	armor *schema.ArmorData,
) {
	t.Helper()
	if armor == nil {
		t.Fatal("armor family data is missing")
	}
	facts := []struct {
		name   string
		known  bool
		source schema.SourceID
	}{
		{"weight", armor.Weight.Known, armor.Weight.Provenance.Source},
		{"physical", armor.Physical.Known, armor.Physical.Provenance.Source},
		{"strike", armor.Strike.Known, armor.Strike.Provenance.Source},
		{"slash", armor.Slash.Known, armor.Slash.Provenance.Source},
		{"pierce", armor.Pierce.Known, armor.Pierce.Provenance.Source},
		{"magic", armor.Magic.Known, armor.Magic.Provenance.Source},
		{"fire", armor.Fire.Known, armor.Fire.Provenance.Source},
		{"lightning", armor.Lightning.Known, armor.Lightning.Provenance.Source},
		{"holy", armor.Holy.Known, armor.Holy.Provenance.Source},
		{"immunity", armor.Immunity.Known, armor.Immunity.Provenance.Source},
		{"robustness", armor.Robustness.Known, armor.Robustness.Provenance.Source},
		{"focus", armor.Focus.Known, armor.Focus.Provenance.Source},
		{"vitality", armor.Vitality.Known, armor.Vitality.Provenance.Source},
		{"poise", armor.Poise.Known, armor.Poise.Provenance.Source},
	}
	expectedSource := sourceIDByRegulationTable[RegulationTableProtector]
	for _, fact := range facts {
		if !fact.known || fact.source != expectedSource {
			t.Fatalf(
				"armor %s known/source = %t/%q, want true/%q",
				fact.name,
				fact.known,
				fact.source,
				expectedSource,
			)
		}
	}
}

func assertArmorValues(
	t *testing.T,
	armor *schema.ArmorData,
	weight float64,
	physical float64,
	immunity uint32,
	poise float64,
) {
	t.Helper()
	if armor == nil ||
		armor.Weight.Value != weight ||
		armor.Physical.Value != physical ||
		armor.Immunity.Value != immunity ||
		armor.Poise.Value != poise {
		t.Fatalf(
			"armor values = %#v, want weight %.1f physical %.1f immunity %d poise %.0f",
			armor,
			weight,
			physical,
			immunity,
			poise,
		)
	}
}

func findGeneratedItem(
	t *testing.T,
	catalog GeneratedCatalog,
	itemID uint32,
) *schema.ItemDocument {
	t.Helper()
	for _, resource := range catalog.Resources {
		if resource.Item != nil && resource.Item.GameID.Value == itemID {
			return resource.Item
		}
	}
	t.Fatalf("item 0x%08X not found", itemID)
	return nil
}
