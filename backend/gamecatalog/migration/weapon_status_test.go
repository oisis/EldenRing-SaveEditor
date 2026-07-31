package migration

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestWeaponStatusFactFailsClosedForDeferredLegacyValues(t *testing.T) {
	fact := weaponStatusFact(0, "StatusBleed", []string{"status-deferred"})
	if fact.Known || fact.Value != 0 {
		t.Fatalf("deferred status fact = %#v, want unknown zero", fact)
	}
	if fact.Provenance.Source != sourceLegacyData ||
		!strings.Contains(fact.Provenance.Method, "status derivation is deferred") {
		t.Fatalf("deferred status provenance = %#v", fact.Provenance)
	}
}

func TestWeaponStatusFactPreservesResolvedLegacyValue(t *testing.T) {
	fact := weaponStatusFact(55, "StatusBleed", nil)
	if !fact.Known || fact.Value != 55 {
		t.Fatalf("resolved status fact = %#v, want known 55", fact)
	}
}

func TestGenerateDoesNotExposeDeferredWeaponStatusesAsKnownZero(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	weaponRecords := 0
	for _, resource := range catalog.Resources {
		if resource.Item == nil ||
			resource.Item.Family.Value != schema.ItemFamilyWeapon {
			continue
		}
		assertDeferredStatusesUnknown(t, resource.Item.GameID.Value, resource.Item.Weapon)
		weaponRecords++
		for _, variant := range resource.Item.Variants {
			assertDeferredStatusesUnknown(
				t,
				variant.GameID.Value,
				resolvedVariantWeapon(t, resource.Item, variant),
			)
			weaponRecords++
		}
	}
	if weaponRecords != 3332 {
		t.Fatalf("weapon records = %d, want 3332", weaponRecords)
	}
}

func assertDeferredStatusesUnknown(
	t *testing.T,
	itemID uint32,
	weapon *schema.WeaponData,
) {
	t.Helper()
	if weapon == nil {
		t.Fatalf("weapon 0x%08X has no family data", itemID)
	}
	if !weapon.Warnings.Known ||
		!containsString(weapon.Warnings.Value, "status-deferred") {
		t.Fatalf("weapon 0x%08X warnings = %#v", itemID, weapon.Warnings)
	}
	for name, fact := range map[string]schema.Fact[int32]{
		"poison":      weapon.StatusPoison,
		"bleed":       weapon.StatusBleed,
		"frost":       weapon.StatusFrost,
		"sleep":       weapon.StatusSleep,
		"madness":     weapon.StatusMadness,
		"scarlet rot": weapon.StatusScarletRot,
	} {
		if fact.Known || fact.Value != 0 {
			t.Fatalf(
				"weapon 0x%08X %s status = %#v, want unknown zero",
				itemID,
				name,
				fact,
			)
		}
	}
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
