package gamecatalog_test

import (
	"testing"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const (
	lanceKey         = "010450A0"
	messmerSpearKey  = "010B2E70"
	chillingMistKey  = "800058AC"
	duelingShieldKey = "03B9ACA0"
	rottenStaffKey   = "016116A0"
	rainOfFireKey    = "401EA302"

	// greatSpearMountBit is canMountWep_SpearHeavy. staleGreatSpearMountBit is
	// canMountWep_SpearLarge, which EquipParamGem never sets for a Great Spear.
	greatSpearMountBit      = uint8(17)
	staleGreatSpearMountBit = uint8(16)
)

func requireItem(t *testing.T, key string) schema.Resource {
	t.Helper()
	resource, err := newRealCatalog(t).ResourceByKindAndKey(schema.ResourceKindItem, key)
	if err != nil {
		t.Fatalf("resolve item %q: %v", key, err)
	}
	if resource.Item == nil {
		t.Fatalf("resource %q carries no item document", key)
	}
	return resource
}

// TestGreatSpearsMountOnTheHeavySpearBit is the catalog-level regression for the
// v1.6.12 fix. wepType 28 checks canMountWep_SpearHeavy; the SpearLarge bit is
// clear in every Ash of War mask that accepts a Great Spear, so the wrong bit
// silently removes all Great Spear compatibility.
func TestGreatSpearsMountOnTheHeavySpearBit(t *testing.T) {
	for _, key := range []string{lanceKey, messmerSpearKey} {
		item := requireItem(t, key).Item
		mount := item.Capabilities.AshOfWarMount
		if !mount.Known || !mount.Enabled || mount.Rules == nil {
			t.Fatalf("weapon %q Ash of War mount = %#v", key, mount)
		}
		if mount.Rules.CompatibilityBit != greatSpearMountBit {
			t.Fatalf("weapon %q compatibility bit = %d, want %d",
				key, mount.Rules.CompatibilityBit, greatSpearMountBit)
		}
		if mount.Rules.CompatibilityBit == staleGreatSpearMountBit {
			t.Fatalf("weapon %q still uses the stale SpearLarge bit %d",
				key, staleGreatSpearMountBit)
		}
		if mount.Rules.WeaponType != "Great Spears" {
			t.Fatalf("weapon %q mount weapon type = %q", key, mount.Rules.WeaponType)
		}
	}
}

// TestGreatSpearsAreCompatibleWithChillingMist proves the corrected bit reaches
// the derived relation both endpoints and the frontend read.
func TestGreatSpearsAreCompatibleWithChillingMist(t *testing.T) {
	catalog := newRealCatalog(t)

	ash := requireItem(t, chillingMistKey)
	mask := ash.Item.AshOfWar.CompatibilityMask
	if !mask.Known {
		t.Fatalf("Chilling Mist compatibility mask = %#v", mask)
	}
	if mask.Value&(uint64(1)<<greatSpearMountBit) == 0 {
		t.Fatalf("Chilling Mist mask %d does not set bit %d", mask.Value, greatSpearMountBit)
	}
	if mask.Value&(uint64(1)<<staleGreatSpearMountBit) != 0 {
		t.Fatalf("Chilling Mist mask %d unexpectedly sets bit %d",
			mask.Value, staleGreatSpearMountBit)
	}

	ashRef := ash.Ref()
	for _, key := range []string{lanceKey, messmerSpearKey} {
		outgoing, _, err := catalog.RelationsByKindAndKey(schema.ResourceKindItem, key)
		if err != nil {
			t.Fatalf("relations for %q: %v", key, err)
		}
		found := 0
		for _, relation := range outgoing {
			if relation.Kind == schema.RelationCompatibleWithAshOfWar && relation.To == ashRef {
				found++
			}
		}
		if found != 1 {
			t.Fatalf("weapon %q has %d compatible_with_aow relations to Chilling Mist, want 1",
				key, found)
		}
	}
}

// TestDuelingShieldAffinitySwitchesStayInOneFamily walks an upgraded Cold
// Dueling Shield through Heavy and back to Standard. Every hop must resolve
// inside the same regulation-derived family and keep the exact upgrade level;
// nothing may fall back to a nearest-lower base or to map iteration order.
func TestDuelingShieldAffinitySwitchesStayInOneFamily(t *testing.T) {
	catalog := newRealCatalog(t)

	base := requireItem(t, duelingShieldKey).Item
	anchors := map[schema.Affinity]uint32{schema.AffinityStandard: base.GameID.Value}
	for _, variant := range base.Variants {
		if variant.Kind.Value != schema.ItemVariantAffinity ||
			!variant.Affinity.Known || !variant.UpgradeLevel.Known ||
			variant.UpgradeLevel.Value != 0 {
			continue
		}
		if _, duplicate := anchors[variant.Affinity.Value]; duplicate {
			t.Fatalf("Dueling Shield has two anchors for affinity %q", variant.Affinity.Value)
		}
		anchors[variant.Affinity.Value] = variant.GameID.Value
	}
	for _, affinity := range []schema.Affinity{
		schema.AffinityCold, schema.AffinityHeavy, schema.AffinityStandard,
	} {
		if _, exists := anchors[affinity]; !exists {
			t.Fatalf("Dueling Shield has no %q anchor", affinity)
		}
	}

	const level = uint8(7)
	current := anchors[schema.AffinityCold] + uint32(level)
	for _, step := range []schema.Affinity{
		schema.AffinityHeavy, schema.AffinityStandard, schema.AffinityCold,
	} {
		next, gotLevel, err := catalog.WeaponInfusionTarget(current, step)
		if err != nil {
			t.Fatalf("WeaponInfusionTarget(0x%08X, %q): %v", current, step, err)
		}
		if gotLevel != level {
			t.Fatalf("switching to %q changed the upgrade level to %d, want %d",
				step, gotLevel, level)
		}
		if want := anchors[step] + uint32(level); next != want {
			t.Fatalf("switching to %q resolved 0x%08X, want 0x%08X", step, next, want)
		}
		// Every hop must stay the same catalog resource, never a look-alike.
		resource, found := catalog.ItemByGameID(next - uint32(level))
		if !found || resource.Key != duelingShieldKey {
			t.Fatalf("affinity %q left the Dueling Shield family: found=%t key=%q",
				step, found, resource.Key)
		}
		current = next
	}
	if want := anchors[schema.AffinityCold] + uint32(level); current != want {
		t.Fatalf("round trip ended at 0x%08X, want 0x%08X", current, want)
	}
}

// TestRelocatedItemsKeepTheirIdentity pins the v1.6.13 category corrections and
// proves the relocation moved only category, subcategory and icon.
func TestRelocatedItemsKeepTheirIdentity(t *testing.T) {
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("load embedded catalog: %v", err)
	}

	for _, want := range []struct {
		key         string
		gameID      uint32
		name        string
		family      schema.ItemFamily
		category    string
		subcategory string
		iconPath    string
		staleIcon   string
	}{
		{
			key: rottenStaffKey, gameID: 0x016116A0, name: "Rotten Staff",
			family: schema.ItemFamilyWeapon, category: "melee_armaments",
			subcategory: "Colossal Weapons",
			iconPath:    "assets/icons/items/melee_armaments/rotten_staff.png",
			staleIcon:   "assets/icons/items/ranged_and_catalysts/rotten_staff.png",
		},
		{
			key: rainOfFireKey, gameID: 0x401EA302, name: "Rain of Fire",
			family: schema.ItemFamilySpell, category: "incantations",
			iconPath:  "assets/icons/items/incantations/rain_of_fire.png",
			staleIcon: "assets/icons/items/sorceries/rain_of_fire.png",
		},
	} {
		item := requireItem(t, want.key).Item
		if item.GameID.Value != want.gameID || item.Family.Value != want.family ||
			item.Presentation.Name.Value != want.name {
			t.Fatalf("%s identity = 0x%08X / %q / %q",
				want.name, item.GameID.Value, item.Family.Value, item.Presentation.Name.Value)
		}
		if item.Category.Value != want.category {
			t.Fatalf("%s category = %q, want %q",
				want.name, item.Category.Value, want.category)
		}
		if want.subcategory != "" && item.Subcategory.Value != want.subcategory {
			t.Fatalf("%s subcategory = %q, want %q",
				want.name, item.Subcategory.Value, want.subcategory)
		}
		if item.Presentation.IconPath.Value != want.iconPath {
			t.Fatalf("%s icon = %q, want %q",
				want.name, item.Presentation.IconPath.Value, want.iconPath)
		}
		if _, exists := data.ReadAsset(want.iconPath); !exists {
			t.Fatalf("%s icon asset %q is missing", want.name, want.iconPath)
		}
		if _, exists := data.ReadAsset(want.staleIcon); exists {
			t.Fatalf("%s stale icon asset %q still ships", want.name, want.staleIcon)
		}
	}
}

// TestNoItemStillReferencesARelocatedIcon proves the vacated icon paths are
// gone from the whole catalog, not only from the two relocated documents.
func TestNoItemStillReferencesARelocatedIcon(t *testing.T) {
	stalePaths := map[string]struct{}{
		"assets/icons/items/ranged_and_catalysts/rotten_staff.png": {},
		"assets/icons/items/sorceries/rain_of_fire.png":            {},
	}
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("load embedded catalog: %v", err)
	}
	for stale := range stalePaths {
		if _, exists := data.ReadAsset(stale); exists {
			t.Fatalf("stale icon asset %q still ships", stale)
		}
	}
	for _, resource := range data.Resources() {
		if resource.Item == nil {
			continue
		}
		icon := resource.Item.Presentation.IconPath
		if _, stale := stalePaths[icon.Value]; icon.Known && stale {
			t.Fatalf("item %q still references the stale icon %q", resource.Key, icon.Value)
		}
		for _, variant := range resource.Item.Variants {
			icon := variant.Data.Presentation.IconPath
			if _, stale := stalePaths[icon.Value]; icon.Known && stale {
				t.Fatalf("item %q variant 0x%08X still references the stale icon %q",
					resource.Key, variant.GameID.Value, icon.Value)
			}
		}
	}
}
