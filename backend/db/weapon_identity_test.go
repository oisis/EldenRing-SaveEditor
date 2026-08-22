package db

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

// Weapon identity references (EquipParamWeapon row = item ID):
//
//	Dueling Shield        0x03B9ACA0  wepType 90, gemMountType 2, MaxUpgrade 25
//	  Cold anchor         0x03B9B024  = base + 900   (its own DB entry)
//	  Occult anchor       0x03B9B150  = base + 1200  (its own DB entry)
//	Carian Thrusting Sh.  0x03B9D3B0  full affinity family
//	Lance                 0x010450A0  wepType 28, gemMountType 2, MaxUpgrade 25
//	  Bloody Lance        0x010454EC  = base + 1100  (the family's only variant)
//	Dagger                0x000F4240  no affinity variants in the app DB
//	Bolt of Gransax       0x00F58390  somber, MaxUpgrade 10, no affinity anchors
//	Unarmed               0x0001ADB0  technical placeholder, originEquipWep = -1

const (
	duelingShield         = uint32(0x03B9ACA0)
	carianThrustingShield = uint32(0x03B9D3B0)
	lance                 = uint32(0x010450A0)
	dagger                = uint32(0x000F4240)
	boltOfGransax         = uint32(0x00F58390)
)

func TestResolveWeaponIdentity(t *testing.T) {
	cases := []struct {
		name     string
		itemID   uint32
		wantBase uint32
		wantInf  string
		wantLvl  int
	}{
		// The reported bug: every Dueling Shield row must resolve to the family
		// base, with the affinity carried separately instead of being folded
		// into the identity.
		{"Dueling Shield Standard +0", duelingShield, duelingShield, "", 0},
		{"Dueling Shield Standard +25", duelingShield + 25, duelingShield, "", 25},
		{"Dueling Shield Cold +0", duelingShield + 900, duelingShield, "Cold", 0},
		{"Dueling Shield Cold +5", duelingShield + 905, duelingShield, "Cold", 5},
		{"Dueling Shield Occult +25", duelingShield + 1225, duelingShield, "Occult", 25},
		{"Dueling Shield Heavy +12", duelingShield + 112, duelingShield, "Heavy", 12},

		// Second full-affinity Thrusting Shield family.
		{"Carian Thrusting Shield Standard +0", carianThrustingShield, carianThrustingShield, "", 0},
		{"Carian Thrusting Shield Cold +12", carianThrustingShield + 912, carianThrustingShield, "Cold", 12},
		{"Carian Thrusting Shield Occult +25", carianThrustingShield + 1225, carianThrustingShield, "Occult", 25},

		// Sparse family: only the Blood variant exists as its own DB entry, and
		// the affinity rows that have no DB entry must resolve just as well.
		{"Lance Standard +0", lance, lance, "", 0},
		{"Bloody Lance +0", lance + 1100, lance, "Blood", 0},
		{"Bloody Lance +25", lance + 1125, lance, "Blood", 25},
		{"Lance Cold +7 (no DB entry for the anchor)", lance + 907, lance, "Cold", 7},

		// Plain family with no affinity variants in the app DB at all. +1225 is
		// the far edge of the old range scan.
		{"Dagger Standard +0", dagger, dagger, "", 0},
		{"Dagger Heavy +25", dagger + 125, dagger, "Heavy", 25},
		{"Dagger Occult +25", dagger + 1225, dagger, "Occult", 25},

		// Somber weapon: no affinity anchors exist, upgrades still resolve.
		{"Bolt of Gransax +0", boltOfGransax, boltOfGransax, "", 0},
		{"Bolt of Gransax +10", boltOfGransax + 10, boltOfGransax, "", 10},
		// Level validity belongs to editor.Validate (upgrade_out_of_range), not
		// to identity: an over-upgraded weapon stays a repairable known item.
		{"Bolt of Gransax +11 stays identified", boltOfGransax + 11, boltOfGransax, "", 11},

		// Technical placeholder with no family relation in regulation.bin.
		{"Unarmed", ItemUnarmed, ItemUnarmed, "", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := ResolveWeaponIdentity(c.itemID)
			if !ok {
				t.Fatalf("ResolveWeaponIdentity(0x%08X) = not resolved, want resolved", c.itemID)
			}
			if got.BaseID != c.wantBase {
				t.Errorf("BaseID = 0x%08X, want 0x%08X", got.BaseID, c.wantBase)
			}
			if got.AffinityName != c.wantInf {
				t.Errorf("AffinityName = %q, want %q", got.AffinityName, c.wantInf)
			}
			if got.Level != c.wantLvl {
				t.Errorf("Level = %d, want %d", got.Level, c.wantLvl)
			}
			if want := c.itemID - uint32(c.wantLvl) - c.wantBase; got.AffinityOffset != want {
				t.Errorf("AffinityOffset = %d, want %d", got.AffinityOffset, want)
			}
		})
	}
}

// TestResolveWeaponIdentity_FailsClosed covers the identity failures. None of
// them may resolve to a different, valid weapon — an unresolvable ID must stay
// unresolvable so the scanner reports it instead of silently renaming it.
func TestResolveWeaponIdentity_FailsClosed(t *testing.T) {
	cases := []struct {
		name   string
		itemID uint32
	}{
		{"unknown weapon ID", 0x00FFFFFF},
		{"affinity offset above Occult", duelingShield + 1300},
		{"somber weapon with an affinity offset", boltOfGransax + 100},
		{"armor ID", 0x100249F0},
		{"talisman ID", 0x200003E8},
		{"goods ID", 0x400003E9},
		{"Ash of War ID", 0x80003070},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got, ok := ResolveWeaponIdentity(c.itemID); ok {
				t.Errorf("ResolveWeaponIdentity(0x%08X) resolved to base 0x%08X (%s +%d), want unresolved",
					c.itemID, got.BaseID, got.AffinityName, got.Level)
			}
		})
	}
}

// TestResolveWeaponIdentity_Deterministic guards the actual defect: the old
// implementation scanned three Go maps for the first base within +1225, so a
// family with both a base entry and affinity entries resolved differently
// between runs. Identity now comes from a single keyed lookup.
func TestResolveWeaponIdentity_Deterministic(t *testing.T) {
	const coldPlus5 = duelingShield + 905
	first, ok := ResolveWeaponIdentity(coldPlus5)
	if !ok {
		t.Fatal("Cold Dueling Shield +5 did not resolve")
	}
	for i := 0; i < 1000; i++ {
		got, ok := ResolveWeaponIdentity(coldPlus5)
		if !ok || got != first {
			t.Fatalf("iteration %d resolved to %+v (ok=%v), want %+v", i, got, ok, first)
		}
	}
}

// TestResolveWeaponIdentity_OriginEquipWepIsTheSource pins the exact regulation
// relation identity is derived from, independently of any lookup helper: the
// affinity anchor's generated OriginEquipWep is the canonical base, and the
// distance between them is the affinity offset.
func TestResolveWeaponIdentity_OriginEquipWepIsTheSource(t *testing.T) {
	const coldAnchor = duelingShield + 900
	mount, ok := data.WeaponGemMounts[coldAnchor]
	if !ok {
		t.Fatalf("EquipParamWeapon row 0x%08X missing from WeaponGemMounts", coldAnchor)
	}
	if mount.OriginEquipWep != duelingShield {
		t.Fatalf("OriginEquipWep = 0x%08X, want the family base 0x%08X", mount.OriginEquipWep, duelingShield)
	}
	got, ok := ResolveWeaponIdentity(coldAnchor + 5)
	if !ok {
		t.Fatal("Cold Dueling Shield +5 did not resolve")
	}
	if got.BaseID != mount.OriginEquipWep {
		t.Errorf("BaseID = 0x%08X, want OriginEquipWep 0x%08X", got.BaseID, mount.OriginEquipWep)
	}
	if got.AffinityOffset != coldAnchor-mount.OriginEquipWep {
		t.Errorf("AffinityOffset = %d, want %d", got.AffinityOffset, coldAnchor-mount.OriginEquipWep)
	}
}

// TestResolveWeaponIdentity_EveryAppWeaponResolves is the general invariant: no
// app weapon entry may be left unidentified, every resolved base must itself be
// an app weapon entry, and every affinity offset must be one of the 13. Unarmed
// is the single documented row without a family relation.
func TestResolveWeaponIdentity_EveryAppWeaponResolves(t *testing.T) {
	maps := []map[uint32]data.ItemData{data.Weapons, data.RangedAndCatalysts, data.Shields}
	checked := 0
	for _, m := range maps {
		for id, item := range m {
			if item.Name == "" {
				continue
			}
			checked++
			got, ok := ResolveWeaponIdentity(id)
			if !ok {
				t.Errorf("app weapon %s (0x%08X) does not resolve", item.Name, id)
				continue
			}
			if !isWeaponCategoryItem(got.BaseID) {
				t.Errorf("%s (0x%08X) resolved to base 0x%08X, which is not an app weapon",
					item.Name, id, got.BaseID)
			}
			if got.AffinityOffset%100 != 0 || got.AffinityOffset > 1200 {
				t.Errorf("%s (0x%08X) resolved to affinity offset %d, outside {0,100..1200}",
					item.Name, id, got.AffinityOffset)
			}
			if got.Level != 0 {
				t.Errorf("%s (0x%08X) is a +0 row but resolved to level %d", item.Name, id, got.Level)
			}
			if base := GetItemData(got.BaseID); base.MaxUpgrade != item.MaxUpgrade {
				t.Errorf("%s (0x%08X) MaxUpgrade %d differs from its base %s MaxUpgrade %d",
					item.Name, id, item.MaxUpgrade, base.Name, base.MaxUpgrade)
			}
		}
	}
	if checked < 600 {
		t.Fatalf("only %d app weapons checked — the category maps look truncated", checked)
	}
}

// TestGetItemDataFuzzy_CanonicalWeaponBase is the public-path regression: the
// caller that decodes upgrade and infusion gets the family base, so Cold +5
// stops decoding as Standard +5.
func TestGetItemDataFuzzy_CanonicalWeaponBase(t *testing.T) {
	cases := []struct {
		name     string
		itemID   uint32
		wantName string
		wantBase uint32
	}{
		{"Cold Dueling Shield +5", duelingShield + 905, "Dueling Shield", duelingShield},
		{"Cold Dueling Shield +0", duelingShield + 900, "Dueling Shield", duelingShield},
		{"Occult Dueling Shield +25", duelingShield + 1225, "Dueling Shield", duelingShield},
		{"Bloody Lance +0", lance + 1100, "Lance", lance},
		{"Dagger +0", dagger, "Dagger", dagger},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			item, base := GetItemDataFuzzy(c.itemID)
			if item.Name != c.wantName {
				t.Errorf("name = %q, want %q", item.Name, c.wantName)
			}
			if base != c.wantBase {
				t.Errorf("baseID = 0x%08X, want 0x%08X", base, c.wantBase)
			}
		})
	}
}

// TestGetItemDataFuzzy_NonWeaponPathsUnchanged proves routing weapons through
// the resolver left every other resolution path alone.
func TestGetItemDataFuzzy_NonWeaponPathsUnchanged(t *testing.T) {
	cases := []struct {
		name     string
		itemID   uint32
		wantName string
		wantBase uint32
	}{
		{"talisman", 0x200003E8, "Crimson Amber Medallion", 0x200003E8},
		{"armor", 0x100249F0, "Iron Kasa", 0x100249F0},
		{"goods handle", 0xB00003E9, "Flask of Crimson Tears", 0x400003E9},
		{"Ash of War", 0x80003070, "Sword Dance", 0x80003070},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			item, base := GetItemDataFuzzy(c.itemID)
			if item.Name != c.wantName {
				t.Errorf("name = %q, want %q", item.Name, c.wantName)
			}
			if base != c.wantBase {
				t.Errorf("baseID = 0x%08X, want 0x%08X", base, c.wantBase)
			}
		})
	}
}

// TestGetItemsByCategory_HidesInfuseVariants protects the Add-flow gate: the
// picker exposes only canonical family bases, which is what keeps the Add path
// from ever combining a pre-infused row with a second affinity offset.
func TestGetItemsByCategory_HidesInfuseVariants(t *testing.T) {
	byID := func(cat string) map[uint32]string {
		out := make(map[uint32]string)
		for _, e := range GetItemsByCategory(cat, "pc") {
			out[e.ID] = e.Name
		}
		return out
	}

	shields := byID("shields")
	if _, ok := shields[duelingShield]; !ok {
		t.Error("picker must still expose the canonical Dueling Shield")
	}
	for _, offset := range []uint32{100, 200, 900, 1200} {
		if name, ok := shields[duelingShield+offset]; ok {
			t.Errorf("picker exposes Dueling Shield affinity variant 0x%08X (%q)", duelingShield+offset, name)
		}
	}
	// A neighbouring plain shield must not be filtered out as collateral.
	if _, ok := shields[carianThrustingShield]; !ok {
		t.Error("picker must still expose Carian Thrusting Shield")
	}

	melee := byID("melee_armaments")
	if _, ok := melee[lance]; !ok {
		t.Error("picker must still expose the canonical Lance")
	}
	if name, ok := melee[lance+1100]; ok {
		t.Errorf("picker exposes Bloody Lance 0x%08X (%q) as a separate choice", lance+1100, name)
	}
	if _, ok := melee[dagger]; !ok {
		t.Error("picker must still expose Dagger")
	}
	if _, ok := melee[boltOfGransax]; !ok {
		t.Error("picker must still expose Bolt of Gransax")
	}
}
