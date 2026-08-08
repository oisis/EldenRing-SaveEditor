package migration

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestDaggerInfusionCapabilityUsesFullRegulationAffinityMatrix(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	item := findSeed(t, collectLegacySnapshot().Items, 0x000F4240)
	identity, err := primaryRegulationForLegacyItem(*item)
	if err != nil {
		t.Fatalf("primaryRegulationForLegacyItem: %v", err)
	}
	lookup, exists, err := regulation.LookupFamilyRow(
		identity.Family,
		RegulationTableRolePrimary,
		identity.RowID,
	)
	if err != nil || !exists {
		t.Fatalf("Dagger lookup = %+v, %t, %v", lookup, exists, err)
	}
	context := generationContext{regulation: regulation}
	capabilities, err := context.buildCapabilities(
		*item,
		schema.ItemFamilyWeapon,
		lookup.Row,
		true,
	)
	if err != nil {
		t.Fatalf("buildCapabilities: %v", err)
	}
	if !capabilities.Infusion.Known || !capabilities.Infusion.Enabled ||
		capabilities.Infusion.Rules == nil {
		t.Fatalf("Dagger infusion = %#v", capabilities.Infusion)
	}
	want := []schema.Affinity{
		schema.AffinityStandard,
		schema.AffinityHeavy,
		schema.AffinityKeen,
		schema.AffinityQuality,
		schema.AffinityFire,
		schema.AffinityFlameArt,
		schema.AffinityLightning,
		schema.AffinitySacred,
		schema.AffinityMagic,
		schema.AffinityCold,
		schema.AffinityPoison,
		schema.AffinityBlood,
		schema.AffinityOccult,
	}
	if !reflect.DeepEqual(capabilities.Infusion.Rules.AllowedAffinities, want) {
		t.Fatalf(
			"Dagger affinities = %#v, want %#v",
			capabilities.Infusion.Rules.AllowedAffinities,
			want,
		)
	}
	if capabilities.Infusion.Provenance.Source !=
		sourceIDByRegulationTable[RegulationTableWeapon] {
		t.Fatalf("Dagger infusion source = %q", capabilities.Infusion.Provenance.Source)
	}
}

func TestWeaponWithoutLegacyInfusionGateStaysDisabled(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	item := findSeed(t, collectLegacySnapshot().Items, 0x0001ADB0)
	identity, err := primaryRegulationForLegacyItem(*item)
	if err != nil {
		t.Fatalf("primaryRegulationForLegacyItem: %v", err)
	}
	lookup, exists, err := regulation.LookupFamilyRow(
		identity.Family,
		RegulationTableRolePrimary,
		identity.RowID,
	)
	if err != nil || !exists {
		t.Fatalf("Unarmed lookup = %+v, %t, %v", lookup, exists, err)
	}
	context := generationContext{regulation: regulation}
	capabilities, err := context.buildCapabilities(
		*item,
		schema.ItemFamilyWeapon,
		lookup.Row,
		true,
	)
	if err != nil {
		t.Fatalf("buildCapabilities: %v", err)
	}
	if !capabilities.Infusion.Known || capabilities.Infusion.Enabled ||
		capabilities.Infusion.Rules != nil {
		t.Fatalf("Unarmed infusion = %#v, want known disabled", capabilities.Infusion)
	}
}

func TestWeaponInfusionGateMatchesRegulationAndLegacyForEveryCanonicalWeapon(
	t *testing.T,
) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	legacyItems := collectLegacySnapshot().Items
	legacyByID := make(map[uint32]seed, len(legacyItems))
	for _, item := range legacyItems {
		legacyByID[item.ID] = item
	}

	const (
		wantWeaponCount    = 548
		wantInfusableCount = 228
	)
	weaponCount := 0
	infusableCount := 0
	for _, resource := range catalog.Resources {
		item := resource.Item
		if item == nil || item.Family.Value != schema.ItemFamilyWeapon {
			continue
		}
		weaponCount++
		legacy, exists := legacyByID[item.GameID.Value]
		if !exists || legacy.WeaponEdit == nil {
			t.Fatalf(
				"canonical weapon 0x%08X has no legacy affinity metadata",
				item.GameID.Value,
			)
		}
		wantEnabled := legacy.WeaponEdit.CanChangeAffinity
		if !item.Capabilities.Infusion.Known ||
			item.Capabilities.Infusion.Enabled != wantEnabled {
			t.Fatalf(
				"weapon 0x%08X infusion known/enabled = %t/%t, want true/%t",
				item.GameID.Value,
				item.Capabilities.Infusion.Known,
				item.Capabilities.Infusion.Enabled,
				wantEnabled,
			)
		}
		if item.Weapon == nil || !item.Weapon.IsInfusable.Known ||
			item.Weapon.IsInfusable.Value != wantEnabled {
			t.Fatalf(
				"weapon 0x%08X family isInfusable = %#v, want known %t",
				item.GameID.Value,
				item.Weapon,
				wantEnabled,
			)
		}
		if wantEnabled {
			infusableCount++
		}
	}
	if weaponCount != wantWeaponCount {
		t.Fatalf("canonical weapon count = %d, want %d", weaponCount, wantWeaponCount)
	}
	if infusableCount != wantInfusableCount {
		t.Fatalf(
			"infusable canonical weapon count = %d, want %d",
			infusableCount,
			wantInfusableCount,
		)
	}
}

func TestWeaponInfusionGateRejectsFixedAffinityWeapons(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	resourcesByItemID := make(map[uint32]*schema.ItemDocument, len(catalog.Resources))
	for index := range catalog.Resources {
		if catalog.Resources[index].Item != nil {
			item := catalog.Resources[index].Item
			resourcesByItemID[item.GameID.Value] = item
		}
	}

	for name, itemID := range map[string]uint32{
		"Black Knife":              0x000F6950,
		"Reduvia":                  0x000FDE80,
		"Sword of Night and Flame": 0x0020A760,
		"Serpentbone Blade":        0x008A8CC0,
		"Treespear":                0x010477B0,
		"Ripple Crescent Halberd":  0x011392E0,
		"Great Club":               0x015F41E0,
		"Troll's Hammer":           0x0160EF90,
	} {
		t.Run(name, func(t *testing.T) {
			item := resourcesByItemID[itemID]
			if item == nil {
				t.Fatalf("item 0x%08X is missing", itemID)
			}
			if !item.Capabilities.Infusion.Known ||
				item.Capabilities.Infusion.Enabled ||
				item.Capabilities.Infusion.Rules != nil {
				t.Fatalf(
					"item 0x%08X infusion = %#v, want known disabled",
					itemID,
					item.Capabilities.Infusion,
				)
			}
			if item.Weapon == nil || !item.Weapon.IsInfusable.Known ||
				item.Weapon.IsInfusable.Value {
				t.Fatalf(
					"item 0x%08X family isInfusable = %#v, want known false",
					itemID,
					item.Weapon,
				)
			}
			if !item.Capabilities.AshOfWarMount.Known ||
				item.Capabilities.AshOfWarMount.Enabled ||
				item.Capabilities.AshOfWarMount.Rules != nil {
				t.Fatalf(
					"item 0x%08X Ash of War mount = %#v, want known disabled",
					itemID,
					item.Capabilities.AshOfWarMount,
				)
			}
		})
	}
}
