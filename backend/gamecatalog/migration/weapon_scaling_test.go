package migration

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestGeneratePreservesFractionalRegulationWeaponScaling(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	want := map[uint32]float64{
		0x00FC6548: 52.5,
		0x00FC65AC: 52.5,
	}
	found := make(map[uint32]float64, len(want))
	for _, resource := range catalog.Resources {
		if resource.Item == nil ||
			resource.Item.Family.Value != schema.ItemFamilyWeapon {
			continue
		}
		collectFractionalArcaneScaling(
			t,
			resource.Item.GameID.Value,
			resource.Item.Weapon,
			want,
			found,
		)
		for _, variant := range resource.Item.Variants {
			collectFractionalArcaneScaling(
				t,
				variant.GameID.Value,
				resolvedVariantWeapon(t, resource.Item, variant),
				want,
				found,
			)
		}
	}
	if len(found) != len(want) {
		t.Fatalf("fractional scaling records = %#v, want %#v", found, want)
	}
	assertArcaneScaling(t, catalog, 0x00FC6610, 81)
}

func collectFractionalArcaneScaling(
	t *testing.T,
	itemID uint32,
	weapon *schema.WeaponData,
	want map[uint32]float64,
	found map[uint32]float64,
) {
	t.Helper()
	expected, exists := want[itemID]
	if !exists {
		return
	}
	if weapon == nil ||
		!weapon.ScalingArcaneRaw.Known ||
		weapon.ScalingArcaneRaw.Value != expected ||
		weapon.ScalingArcaneRaw.Provenance.Source !=
			sourceIDByRegulationTable[RegulationTableWeapon] {
		t.Fatalf(
			"weapon 0x%08X arcane scaling = %#v, want %v from Regulation",
			itemID,
			weapon,
			expected,
		)
	}
	found[itemID] = weapon.ScalingArcaneRaw.Value
}

func assertArcaneScaling(
	t *testing.T,
	catalog GeneratedCatalog,
	itemID uint32,
	expected float64,
) {
	t.Helper()
	for _, resource := range catalog.Resources {
		if resource.Item == nil {
			continue
		}
		if resource.Item.GameID.Value == itemID {
			assertWeaponArcaneScaling(t, itemID, resource.Item.Weapon, expected)
			return
		}
		for _, variant := range resource.Item.Variants {
			if variant.GameID.Value == itemID {
				assertWeaponArcaneScaling(
					t,
					itemID,
					resolvedVariantWeapon(t, resource.Item, variant),
					expected,
				)
				return
			}
		}
	}
	t.Fatalf("weapon 0x%08X not found", itemID)
}

func assertWeaponArcaneScaling(
	t *testing.T,
	itemID uint32,
	weapon *schema.WeaponData,
	expected float64,
) {
	t.Helper()
	if weapon == nil ||
		!weapon.ScalingArcaneRaw.Known ||
		weapon.ScalingArcaneRaw.Value != expected {
		t.Fatalf(
			"weapon 0x%08X arcane scaling = %#v, want %v",
			itemID,
			weapon,
			expected,
		)
	}
}
