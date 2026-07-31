package migration

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestGenerateBuildsAllWeaponCoreStatsFromRegulation(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	count := 0
	for _, resource := range catalog.Resources {
		if resource.Item == nil ||
			resource.Item.Family.Value != schema.ItemFamilyWeapon {
			continue
		}
		assertWeaponCoreStatsFromRegulation(t, resource.Item.Weapon)
		count++
		for _, variant := range resource.Item.Variants {
			assertWeaponCoreStatsFromRegulation(
				t,
				resolvedVariantWeapon(t, resource.Item, variant),
			)
			count++
		}
	}
	if count != 3332 {
		t.Fatalf("weapon records = %d, want 3332", count)
	}
}

func assertWeaponCoreStatsFromRegulation(
	t *testing.T,
	weapon *schema.WeaponData,
) {
	t.Helper()
	if weapon == nil {
		t.Fatal("weapon family data is missing")
	}
	weaponSource := sourceIDByRegulationTable[RegulationTableWeapon]
	facts := []struct {
		name   string
		known  bool
		source schema.SourceID
	}{
		{"weaponTypeID", weapon.WeaponTypeID.Known, weapon.WeaponTypeID.Provenance.Source},
		{"sortGroupID", weapon.SortGroupID.Known, weapon.SortGroupID.Provenance.Source},
		{"reinforceTypeID", weapon.ReinforceTypeID.Known, weapon.ReinforceTypeID.Provenance.Source},
		{"gemMountType", weapon.GemMountType.Known, weapon.GemMountType.Provenance.Source},
		{"weight", weapon.Weight.Known, weapon.Weight.Provenance.Source},
		{"attackPhysical", weapon.AttackPhysical.Known, weapon.AttackPhysical.Provenance.Source},
		{"attackMagic", weapon.AttackMagic.Known, weapon.AttackMagic.Provenance.Source},
		{"attackFire", weapon.AttackFire.Known, weapon.AttackFire.Provenance.Source},
		{"attackLightning", weapon.AttackLightning.Known, weapon.AttackLightning.Provenance.Source},
		{"attackHoly", weapon.AttackHoly.Known, weapon.AttackHoly.Provenance.Source},
		{"attackStamina", weapon.AttackStamina.Known, weapon.AttackStamina.Provenance.Source},
		{"guardPhysical", weapon.GuardPhysical.Known, weapon.GuardPhysical.Provenance.Source},
		{"guardMagic", weapon.GuardMagic.Known, weapon.GuardMagic.Provenance.Source},
		{"guardFire", weapon.GuardFire.Known, weapon.GuardFire.Provenance.Source},
		{"guardLightning", weapon.GuardLightning.Known, weapon.GuardLightning.Provenance.Source},
		{"guardHoly", weapon.GuardHoly.Known, weapon.GuardHoly.Provenance.Source},
		{"guardBoost", weapon.GuardBoost.Known, weapon.GuardBoost.Provenance.Source},
		{"requiredStrength", weapon.RequiredStrength.Known, weapon.RequiredStrength.Provenance.Source},
		{"requiredDexterity", weapon.RequiredDexterity.Known, weapon.RequiredDexterity.Provenance.Source},
		{"requiredIntelligence", weapon.RequiredIntelligence.Known, weapon.RequiredIntelligence.Provenance.Source},
		{"requiredFaith", weapon.RequiredFaith.Known, weapon.RequiredFaith.Provenance.Source},
		{"requiredArcane", weapon.RequiredArcane.Known, weapon.RequiredArcane.Provenance.Source},
		{"critical", weapon.Critical.Known, weapon.Critical.Provenance.Source},
		{"defaultAshOfWarID", weapon.DefaultAshOfWarID.Known, weapon.DefaultAshOfWarID.Provenance.Source},
	}
	for _, fact := range facts {
		if !fact.known || fact.source != weaponSource {
			t.Fatalf(
				"weapon %s known/source = %t/%q, want true/%q",
				fact.name,
				fact.known,
				fact.source,
				weaponSource,
			)
		}
	}
	if weapon.IsInfusable.Provenance.Source != weaponSource {
		t.Fatalf(
			"weapon isInfusable source = %q, want %q",
			weapon.IsInfusable.Provenance.Source,
			weaponSource,
		)
	}
	reinforceSource := sourceIDByRegulationTable[RegulationTableReinforceWeapon]
	for name, fact := range map[string]schema.Provenance{
		"isSomber":   weapon.IsSomber.Provenance,
		"maxUpgrade": weapon.MaxUpgrade.Provenance,
	} {
		if fact.Source != reinforceSource {
			t.Fatalf(
				"weapon %s source = %q, want %q",
				name,
				fact.Source,
				reinforceSource,
			)
		}
	}
}
