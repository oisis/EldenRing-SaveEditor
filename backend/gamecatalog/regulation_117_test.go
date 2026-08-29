package gamecatalog_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/png"
	"math"
	"testing"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// The Regulation 1.17 ingest is protected here at the layer a user reaches it:
// the loaded catalog. The package it was generated from is evidence, not a
// contract, so nothing below reads it; every expected value is stated inline
// and a regeneration that silently drops back to 1.16 fails.

// regulation117IconSource is the manifest source the 24 product icons of the
// 1.17 content must carry. Anything else means the icons were re-derived
// instead of taken from the stable release they were verified in.
const regulation117IconSource = schema.SourceID("product_item_icons_1_7_1")

const regulation117IconRelease = "053afaf2bbdaa8eac2e329447d9af21dbffa29cf"

// affinityOrder is the affinity a variant carries at family base + index*100.
var affinityOrder = []schema.Affinity{
	schema.AffinityStandard, schema.AffinityHeavy, schema.AffinityKeen,
	schema.AffinityQuality, schema.AffinityFire, schema.AffinityFlameArt,
	schema.AffinityLightning, schema.AffinitySacred, schema.AffinityMagic,
	schema.AffinityCold, schema.AffinityPoison, schema.AffinityBlood,
	schema.AffinityOccult,
}

type regulation117Armament struct {
	key            string
	gameID         uint32
	name           string
	category       string
	subcategory    string
	weaponType     uint16
	weight         float64
	attackPhysical int32
	guardBoost     int32
	sortID         uint32
	iconID         uint32
	maxUpgrade     int32
	somber         bool
	infusable      bool
	variants       int
	ashOfWar       int32
	iconPath       string
}

// regulation117Armaments are the eight families 1.17 added. Six carry the full
// thirteen-row affinity set, two are somber single rows.
var regulation117Armaments = []regulation117Armament{
	{key: "00365240", gameID: 3560000, name: "Leontiel's Greatsword", category: "melee_armaments", subcategory: "Greatswords", weaponType: 5, weight: 9.5, attackPhysical: 130, guardBoost: 44, sortID: 1212900, iconID: 12401, maxUpgrade: 10, somber: true, infusable: false, variants: 0, ashOfWar: 1200, iconPath: "assets/icons/items/melee_armaments/leontiels_greatsword.png"},
	{key: "00822850", gameID: 8530000, name: "Hefty Scimitar", category: "melee_armaments", subcategory: "Curved Greatswords", weaponType: 11, weight: 9.5, attackPhysical: 135, guardBoost: 46, sortID: 1700000, iconID: 12400, maxUpgrade: 25, somber: false, infusable: true, variants: 12, ashOfWar: 103, iconPath: "assets/icons/items/melee_armaments/hefty_scimitar.png"},
	{key: "00CE2570", gameID: 13510000, name: "Golden Order Flail", category: "melee_armaments", subcategory: "Flails", weaponType: 24, weight: 5.0, attackPhysical: 75, guardBoost: 22, sortID: 2303900, iconID: 12402, maxUpgrade: 10, somber: true, infusable: false, variants: 0, ashOfWar: 1201, iconPath: "assets/icons/items/melee_armaments/golden_order_flail.png"},
	{key: "01E14320", gameID: 31540000, name: "Silver Grooved Shield", category: "shields", subcategory: "Medium Shields", weaponType: 67, weight: 3.0, attackPhysical: 61, guardBoost: 50, sortID: 7209500, iconID: 12407, maxUpgrade: 25, somber: false, infusable: true, variants: 12, ashOfWar: 10, iconPath: "assets/icons/items/shields/silver_grooved_shield.png"},
	{key: "03B9FAC0", gameID: 62520000, name: "Ritual Thrusting Shield", category: "shields", subcategory: "Thrusting Shields", weaponType: 90, weight: 10.0, attackPhysical: 135, guardBoost: 55, sortID: 7400000, iconID: 12404, maxUpgrade: 25, somber: false, infusable: true, variants: 12, ashOfWar: 8000, iconPath: "assets/icons/items/shields/ritual_thrusting_shield.png"},
	{key: "03D8A650", gameID: 64530000, name: "Reverse-Bladed Sword", category: "melee_armaments", subcategory: "Backhand Blades", weaponType: 92, weight: 3.0, attackPhysical: 115, guardBoost: 13, sortID: 1750000, iconID: 12403, maxUpgrade: 25, somber: false, infusable: true, variants: 12, ashOfWar: 4090, iconPath: "assets/icons/items/melee_armaments/reverse_bladed_sword.png"},
	{key: "03F72AD0", gameID: 66530000, name: "Reed Great Katana", category: "melee_armaments", subcategory: "Great Katanas", weaponType: 94, weight: 10.0, attackPhysical: 158, guardBoost: 33, sortID: 1850000, iconID: 12405, maxUpgrade: 25, somber: false, infusable: true, variants: 12, ashOfWar: 4110, iconPath: "assets/icons/items/melee_armaments/reed_great_katana.png"},
	{key: "04066D10", gameID: 67530000, name: "Idus Sword", category: "melee_armaments", subcategory: "Light Greatswords", weaponType: 93, weight: 7.0, attackPhysical: 124, guardBoost: 38, sortID: 1150000, iconID: 12406, maxUpgrade: 25, somber: false, infusable: true, variants: 12, ashOfWar: 101, iconPath: "assets/icons/items/melee_armaments/idus_sword.png"},
}

// regulation117HeftyScimitarVariantNames pins the official variant names of the
// one family whose affinity names do not follow the "<Affinity> <Base>" shape.
// Reading them from the base name would silently invent "Heavy Hefty Scimitar".
var regulation117HeftyScimitarVariantNames = map[uint32]string{
	8530100: "Hefty Heavy Scimitar",
	8530200: "Hefty Keen Scimitar",
	8530300: "Hefty Quality Scimitar",
	8530400: "Hefty Fire Scimitar",
	8530500: "Hefty Flame Art Scimitar",
	8530600: "Hefty Lightning Scimitar",
	8530700: "Hefty Sacred Scimitar",
	8530800: "Hefty Magic Scimitar",
	8530900: "Hefty Cold Scimitar",
	8531000: "Hefty Poison Scimitar",
	8531100: "Hefty Blood Scimitar",
	8531200: "Hefty Occult Scimitar",
}

type regulation117Armour struct {
	key        string
	gameID     uint32
	name       string
	category   string
	weight     float64
	poise      float64
	physical   float64
	sortID     uint32
	iconMale   uint32
	iconFemale uint32
	iconPath   string
}

// regulation117Armour are the sixteen supported_public pieces of the four new
// sets. The two altered variants are deliberately absent; see
// TestRegulation117UnsupportedContentNeverReachesTheCatalog.
var regulation117ArmourPieces = []regulation117Armour{
	{key: "10517B60", gameID: 0x10517B60, name: "Broken Gold Mask", category: "head", weight: 2.0, poise: 2.0, physical: 2.0, sortID: 501245, iconMale: 15700, iconFemale: 15700, iconPath: "assets/icons/items/head/broken_gold_mask.png"},
	{key: "10517BC4", gameID: 0x10517BC4, name: "Gold Tattoo (Chest)", category: "chest", weight: 2.2, poise: 5.0, physical: 3.8, sortID: 601305, iconMale: 15701, iconFemale: 15718, iconPath: "assets/icons/items/chest/gold_tattoo_chest.png"},
	{key: "10517C28", gameID: 0x10517C28, name: "Gold Tattoo (Arm)", category: "arms", weight: 0.6, poise: 1.0, physical: 0.8, sortID: 701115, iconMale: 15702, iconFemale: 15702, iconPath: "assets/icons/items/arms/gold_tattoo_arm.png"},
	{key: "10517C8C", gameID: 0x10517C8C, name: "Gold Tattoo (Leg)", category: "legs", weight: 3.5, poise: 4.0, physical: 3.2, sortID: 801145, iconMale: 15703, iconFemale: 15703, iconPath: "assets/icons/items/legs/gold_tattoo_leg.png"},
	{key: "1051A270", gameID: 0x1051A270, name: "Silver Grooved Helm", category: "head", weight: 3.0, poise: 6.0, physical: 3.7, sortID: 507015, iconMale: 15704, iconFemale: 15704, iconPath: "assets/icons/items/head/silver_grooved_helm.png"},
	{key: "1051A2D4", gameID: 0x1051A2D4, name: "Silver Grooved Armor", category: "chest", weight: 7.1, poise: 16.0, physical: 10.2, sortID: 607025, iconMale: 15705, iconFemale: 15705, iconPath: "assets/icons/items/chest/silver_grooved_armor.png"},
	{key: "1051A338", gameID: 0x1051A338, name: "Silver Grooved Gauntlets", category: "arms", weight: 2.4, poise: 3.0, physical: 3.1, sortID: 706015, iconMale: 15706, iconFemale: 15706, iconPath: "assets/icons/items/arms/silver_grooved_gauntlets.png"},
	{key: "1051A39C", gameID: 0x1051A39C, name: "Silver Grooved Greaves", category: "legs", weight: 4.4, poise: 8.0, physical: 6.4, sortID: 806015, iconMale: 15707, iconFemale: 15707, iconPath: "assets/icons/items/legs/silver_grooved_greaves.png"},
	{key: "1051C980", gameID: 0x1051C980, name: "Leontiel's Hat", category: "head", weight: 3.2, poise: 3.0, physical: 3.5, sortID: 506131, iconMale: 15709, iconFemale: 15709, iconPath: "assets/icons/items/head/leontiel_s_hat.png"},
	{key: "1051C9E4", gameID: 0x1051C9E4, name: "Leontiel's Armor", category: "chest", weight: 8.3, poise: 8.0, physical: 11.9, sortID: 606081, iconMale: 15710, iconFemale: 15710, iconPath: "assets/icons/items/chest/leontiel_s_armor.png"},
	{key: "1051CA48", gameID: 0x1051CA48, name: "Leontiel's Leather Gloves", category: "arms", weight: 1.9, poise: 2.0, physical: 3.0, sortID: 705131, iconMale: 15711, iconFemale: 15711, iconPath: "assets/icons/items/arms/leontiel_s_leather_gloves.png"},
	{key: "1051CAAC", gameID: 0x1051CAAC, name: "Leontiel's Boots", category: "legs", weight: 4.7, poise: 5.0, physical: 6.7, sortID: 805121, iconMale: 15712, iconFemale: 15712, iconPath: "assets/icons/items/legs/leontiel_s_boots.png"},
	{key: "1051F090", gameID: 0x1051F090, name: "Steel Helm", category: "head", weight: 7.9, poise: 13.0, physical: 7.2, sortID: 509000, iconMale: 15714, iconFemale: 15714, iconPath: "assets/icons/items/head/steel_helm.png"},
	{key: "1051F0F4", gameID: 0x1051F0F4, name: "Steel Armor", category: "chest", weight: 18.5, poise: 35.0, physical: 18.5, sortID: 609000, iconMale: 15715, iconFemale: 15715, iconPath: "assets/icons/items/chest/steel_armor.png"},
	{key: "1051F158", gameID: 0x1051F158, name: "Steel Gauntlets", category: "arms", weight: 6.1, poise: 10.0, physical: 5.6, sortID: 708000, iconMale: 15716, iconFemale: 15716, iconPath: "assets/icons/items/arms/steel_gauntlets.png"},
	{key: "1051F1BC", gameID: 0x1051F1BC, name: "Steel Greaves", category: "legs", weight: 11.3, poise: 19.0, physical: 10.8, sortID: 808000, iconMale: 15717, iconFemale: 15717, iconPath: "assets/icons/items/legs/steel_greaves.png"},
}

type regulation117AttireItem struct {
	key         string
	gameID      uint32
	rowID       uint32
	name        string
	caption     string
	description string
	iconID      uint32
	sortID      uint32
	sortGroupID uint8
	iconPath    string
	iconSHA256  string
}

// regulation117Attire is the Spectral Steed Attire goods 1.17 added: three DLC
// key items that unlock a Torrent appearance. Every text is the official
// item_dlc02 FMG entry, never the application-authored name the stable v1.7.1
// release displayed, and every limit is the one that release settled on.
var regulation117Attire = []regulation117AttireItem{
	{
		key: "401EAA00", gameID: 0x401EAA00, rowID: 2009600,
		name:        "Spectral Steed Regalia: Tree Sentinel",
		caption:     "A smaller version of the solemn gold armor that adorns the warhorses of the Tree Sentinels who serve the Erdtree.\n\nOne of several liveries once prepared for spectral steeds, which allows Torrent, through grace, to change his appearance.\n\nThough a spectral steed's form will change when vested in this garb, their heart will remain the same.",
		description: "Allows grace to alter Torrent's appearance",
		iconID:      3900, sortID: 206160, sortGroupID: 60,
		iconPath:   "assets/icons/items/key_items/tree_sentinel_spectral_steed_attire.png",
		iconSHA256: "8026042d0462e91949bb8045bb875c8469a41d5558e6188293a184a78e367aa5",
	},
	{
		key: "401EAA0A", gameID: 0x401EAA0A, rowID: 2009610,
		name:        "Spectral Steed Regalia: Carian Silver",
		caption:     "The richly ornamented white silver armor of the Carian Royal line is a one-of-a-kind item made for a certain spectral steed, once forged in the lands of the Three Sisters.\n\nIt is through grace that the spectral steed Torrent's appearance may be changed.\n\nThough a spectral steed's form will change when vested in this garb, their heart will remain the same.",
		description: "Allows grace to alter Torrent's appearance",
		iconID:      3901, sortID: 206170, sortGroupID: 60,
		iconPath:   "assets/icons/items/key_items/silver_of_caria_spectral_steed_attire.png",
		iconSHA256: "7e85e8a85008d38430ccc94c5441f96a58bb5878e36ec0134108f1bedbf23431",
	},
	{
		key: "401EAA14", gameID: 0x401EAA14, rowID: 2009620,
		name:        "Spectral Steed Regalia: Funereal Night",
		caption:     "A smaller version of the sable attire used to adorn and conceal the warhorses of the Night's Cavalry.\n\nOne of several liveries once prepared for spectral steeds, which allows Torrent, through grace, to change his appearance.\n\nThough a spectral steed's form will change when vested in this garb, their heart will remain the same.",
		description: "Allows grace to alter Torrent's appearance",
		iconID:      3902, sortID: 206180, sortGroupID: 60,
		iconPath:   "assets/icons/items/key_items/funereal_night_spectral_steed_attire.png",
		iconSHA256: "3bc04cfa3aa861a4c24f66e7697d3fb339bbbf2d226798dabbcad87d9edce4da",
	},
}

// TestRegulation117ArmamentFamiliesAreComplete is the public-layer contract of
// the eight new families: identity, tab placement, balance values, upgrade and
// infusion capability, and the icon the frontend renders.
func TestRegulation117ArmamentFamiliesAreComplete(t *testing.T) {
	catalog := newRealCatalog(t)
	for _, want := range regulation117Armaments {
		resource, err := catalog.ResourceByKindAndKey(schema.ResourceKindItem, want.key)
		if err != nil {
			t.Errorf("%s: resolve key %q: %v", want.name, want.key, err)
			continue
		}
		item := resource.Item
		if item == nil {
			t.Errorf("%s: resource carries no item document", want.name)
			continue
		}
		if item.GameID.Value != want.gameID || item.Family.Value != schema.ItemFamilyWeapon {
			t.Errorf("%s identity = 0x%08X / %q", want.name, item.GameID.Value, item.Family.Value)
		}
		if item.Presentation.Name.Value != want.name {
			t.Errorf("%s name = %q", want.key, item.Presentation.Name.Value)
		}
		if item.Category.Value != want.category || item.Subcategory.Value != want.subcategory {
			t.Errorf("%s placement = %q/%q, want %q/%q",
				want.name, item.Category.Value, item.Subcategory.Value,
				want.category, want.subcategory)
		}
		weapon := item.Weapon
		if weapon == nil {
			t.Errorf("%s: weapon section is missing", want.name)
			continue
		}
		assertEqual(t, want.name, "sourceRowID", weapon.SourceRowID.Value, want.gameID)
		assertEqual(t, want.name, "weaponTypeID", weapon.WeaponTypeID.Value, want.weaponType)
		assertEqual(t, want.name, "sortID", weapon.SortID.Value, want.sortID)
		assertEqual(t, want.name, "iconID", weapon.IconID.Value, want.iconID)
		assertEqual(t, want.name, "attackPhysical", weapon.AttackPhysical.Value, want.attackPhysical)
		assertEqual(t, want.name, "guardBoost", weapon.GuardBoost.Value, want.guardBoost)
		assertEqual(t, want.name, "maxUpgrade", weapon.MaxUpgrade.Value, want.maxUpgrade)
		assertEqual(t, want.name, "isSomber", weapon.IsSomber.Value, want.somber)
		assertEqual(t, want.name, "isInfusable", weapon.IsInfusable.Value, want.infusable)
		assertEqual(t, want.name, "defaultAshOfWarID", weapon.DefaultAshOfWarID.Value, want.ashOfWar)
		assertClose(t, want.name, "weight", weapon.Weight.Value, want.weight)
		if weapon.SwordArtsName.Value == "" || !weapon.SwordArtsName.Known {
			t.Errorf("%s: sword arts name = %#v", want.name, weapon.SwordArtsName)
		}

		upgrade := item.Capabilities.Upgrade
		if !upgrade.Known || !upgrade.Enabled || upgrade.Rules == nil {
			t.Errorf("%s: upgrade capability = %#v", want.name, upgrade)
		} else {
			wantModel := schema.UpgradeModelStandard
			if want.somber {
				wantModel = schema.UpgradeModelSomber
			}
			if upgrade.Rules.Model != wantModel ||
				int32(upgrade.Rules.MaxLevel) != want.maxUpgrade {
				t.Errorf("%s: upgrade rules = %#v, want %q up to %d",
					want.name, *upgrade.Rules, wantModel, want.maxUpgrade)
			}
		}

		infusion := item.Capabilities.Infusion
		if infusion.Enabled != want.infusable {
			t.Errorf("%s: infusion enabled = %t, want %t",
				want.name, infusion.Enabled, want.infusable)
		}
		if want.infusable && (infusion.Rules == nil ||
			len(infusion.Rules.AllowedAffinities) != len(affinityOrder)) {
			t.Errorf("%s: infusion rules = %#v, want all %d affinities",
				want.name, infusion.Rules, len(affinityOrder))
		}
		// A somber armament must not offer an Ash of War mount either; both
		// come from the same gemMountType and drifting apart is the failure
		// that reaches the user as a mountable somber weapon.
		if mount := item.Capabilities.AshOfWarMount; mount.Enabled != want.infusable {
			t.Errorf("%s: Ash of War mount enabled = %t, want %t",
				want.name, mount.Enabled, want.infusable)
		}

		if len(item.Variants) != want.variants {
			t.Errorf("%s: %d variants, want %d", want.name, len(item.Variants), want.variants)
		}
		assertRegulation117Variants(t, want, item)
		assertRegulation117Presentation(t, want.name, item.Presentation, want.iconPath)
	}
}

// assertRegulation117Variants proves each affinity variant sits at its own
// regulation row, carries its own official name and shares the family's weight
// and guard boost, so a half-updated family cannot pass.
func assertRegulation117Variants(
	t *testing.T,
	want regulation117Armament,
	item *schema.ItemDocument,
) {
	t.Helper()
	seen := make(map[schema.Affinity]uint32, len(item.Variants))
	for _, variant := range item.Variants {
		offset := variant.GameID.Value - want.gameID
		if offset%100 != 0 || offset/100 == 0 || offset/100 > 12 {
			t.Errorf("%s: variant 0x%08X is not a family affinity row", want.name, variant.GameID.Value)
			continue
		}
		index := offset / 100
		wantAffinity := affinityOrder[index]
		if variant.Kind.Value != schema.ItemVariantAffinity ||
			variant.Affinity.Value != wantAffinity ||
			variant.UpgradeLevel.Value != 0 {
			t.Errorf("%s: variant %d = %q/%q/level %d, want affinity %q at level 0",
				want.name, index, variant.Kind.Value, variant.Affinity.Value,
				variant.UpgradeLevel.Value, wantAffinity)
		}
		if _, duplicate := seen[variant.Affinity.Value]; duplicate {
			t.Errorf("%s: affinity %q appears twice", want.name, variant.Affinity.Value)
		}
		seen[variant.Affinity.Value] = variant.GameID.Value

		data := variant.Data.Weapon
		if data == nil {
			t.Errorf("%s: variant %d carries no weapon section", want.name, index)
			continue
		}
		assertClose(t, want.name, fmt.Sprintf("variant %d weight", index), data.Weight.Value, want.weight)
		assertEqual(t, want.name, fmt.Sprintf("variant %d guardBoost", index), data.GuardBoost.Value, want.guardBoost)
		assertEqual(t, want.name, fmt.Sprintf("variant %d maxUpgrade", index), data.MaxUpgrade.Value, want.maxUpgrade)
		if variant.Data.Presentation.Name.Value == "" {
			t.Errorf("%s: variant %d has no official name", want.name, index)
		}
		if variant.Data.Presentation.IconPath.Value != want.iconPath {
			t.Errorf("%s: variant %d icon = %q, want the family icon %q",
				want.name, index, variant.Data.Presentation.IconPath.Value, want.iconPath)
		}
		if want.gameID == 8530000 {
			if got := variant.Data.Presentation.Name.Value; got != regulation117HeftyScimitarVariantNames[variant.GameID.Value] {
				t.Errorf("Hefty Scimitar variant 0x%08X name = %q, want %q",
					variant.GameID.Value, got,
					regulation117HeftyScimitarVariantNames[variant.GameID.Value])
			}
		}
	}
	if want.variants > 0 && len(seen) != want.variants {
		t.Errorf("%s: %d distinct affinities, want %d", want.name, len(seen), want.variants)
	}
}

// TestRegulation117ArmourSetsAreComplete is the same contract for the sixteen
// supported armour pieces, including the one row whose female icon differs.
func TestRegulation117ArmourSetsAreComplete(t *testing.T) {
	catalog := newRealCatalog(t)
	slots := map[string]schema.EquipmentSlot{
		"head":  schema.EquipmentSlotHead,
		"chest": schema.EquipmentSlotChest,
		"arms":  schema.EquipmentSlotArms,
		"legs":  schema.EquipmentSlotLegs,
	}
	for _, want := range regulation117ArmourPieces {
		resource, err := catalog.ResourceByKindAndKey(schema.ResourceKindItem, want.key)
		if err != nil {
			t.Errorf("%s: resolve key %q: %v", want.name, want.key, err)
			continue
		}
		item := resource.Item
		if item == nil || item.Armor == nil {
			t.Errorf("%s: armour document is missing", want.name)
			continue
		}
		if item.GameID.Value != want.gameID || item.Family.Value != schema.ItemFamilyArmor {
			t.Errorf("%s identity = 0x%08X / %q", want.name, item.GameID.Value, item.Family.Value)
		}
		if item.Presentation.Name.Value != want.name {
			t.Errorf("%s name = %q", want.key, item.Presentation.Name.Value)
		}
		if item.Category.Value != want.category {
			t.Errorf("%s category = %q, want %q", want.name, item.Category.Value, want.category)
		}
		armour := item.Armor
		assertEqual(t, want.name, "sortID", armour.SortID.Value, want.sortID)
		assertEqual(t, want.name, "iconIDMale", armour.IconIDMale.Value, want.iconMale)
		assertEqual(t, want.name, "iconIDFemale", armour.IconIDFemale.Value, want.iconFemale)
		assertClose(t, want.name, "weight", armour.Weight.Value, want.weight)
		assertClose(t, want.name, "poise", armour.Poise.Value, want.poise)
		assertClose(t, want.name, "physical", armour.Physical.Value, want.physical)

		equipment := item.Capabilities.Equipment
		if !equipment.Enabled || equipment.Rules == nil ||
			len(equipment.Rules.AllowedSlots) != 1 ||
			equipment.Rules.AllowedSlots[0] != slots[want.category] {
			t.Errorf("%s: equipment rules = %#v, want the single %q slot",
				want.name, equipment.Rules, want.category)
		}
		if item.Capabilities.Upgrade.Enabled ||
			item.Capabilities.Infusion.Enabled ||
			item.Capabilities.AshOfWarMount.Enabled {
			t.Errorf("%s: armour must not be upgradeable, infusable or mountable", want.name)
		}
		if len(item.Variants) != 0 {
			t.Errorf("%s: armour carries %d variants", want.name, len(item.Variants))
		}
		assertRegulation117Presentation(t, want.name, item.Presentation, want.iconPath)
	}
}

// assertRegulation117Presentation holds the text and icon contract agreed for
// 1.17 content: an official FMG name and caption, no invented description or
// location, and an icon taken from the released v1.7.1 product assets rather
// than re-derived.
func assertRegulation117Presentation(
	t *testing.T,
	label string,
	presentation schema.ItemPresentation,
	wantIcon string,
) {
	t.Helper()
	if !presentation.Caption.Known || presentation.Caption.Value == "" {
		t.Errorf("%s: caption = %#v, want the official FMG caption", label, presentation.Caption)
	}
	if presentation.Description.Known || presentation.Description.Value != "" {
		t.Errorf("%s: description = %#v, want an unknown fact", label, presentation.Description)
	}
	if presentation.Location.Known || presentation.Location.Value != "" {
		t.Errorf("%s: location = %#v, want an unknown fact", label, presentation.Location)
	}
	if presentation.Description.Provenance.Source == "" ||
		presentation.Location.Provenance.Source == "" {
		t.Errorf("%s: missing text must still carry provenance: %#v / %#v",
			label, presentation.Description.Provenance, presentation.Location.Provenance)
	}
	if presentation.IconPath.Value != wantIcon {
		t.Errorf("%s: icon = %q, want %q", label, presentation.IconPath.Value, wantIcon)
	}
	if presentation.IconPath.Provenance.Source != regulation117IconSource {
		t.Errorf("%s: icon source = %q, want %q",
			label, presentation.IconPath.Provenance.Source, regulation117IconSource)
	}
}

// TestRegulation117IconsAreTheReleasedProductAssets proves all 27 icons ship,
// decode as 256x256 images and that none of them fell back to the placeholder.
// The three Spectral Steed Attire icons additionally have to match the byte
// content of the v1.7.1 release, so a re-encode or a rescale is a failure and
// not a cosmetic difference.
func TestRegulation117IconsAreTheReleasedProductAssets(t *testing.T) {
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	paths := make([]string, 0, 27)
	for _, want := range regulation117Armaments {
		paths = append(paths, want.iconPath)
	}
	for _, want := range regulation117ArmourPieces {
		paths = append(paths, want.iconPath)
	}
	for _, want := range regulation117Attire {
		paths = append(paths, want.iconPath)
	}
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if _, duplicate := seen[path]; duplicate {
			t.Errorf("icon %q is claimed by two different 1.17 items", path)
		}
		seen[path] = struct{}{}
		content, exists := data.ReadAsset(path)
		if !exists {
			t.Errorf("icon asset %q does not ship", path)
			continue
		}
		config, err := png.DecodeConfig(bytes.NewReader(content))
		if err != nil {
			t.Errorf("icon asset %q is not a decodable PNG: %v", path, err)
			continue
		}
		if config.Width != 256 || config.Height != 256 {
			t.Errorf("icon asset %q is %dx%d, want 256x256", path, config.Width, config.Height)
		}
	}
	if len(seen) != 27 {
		t.Errorf("%d distinct 1.17 icons, want 27", len(seen))
	}
	for _, want := range regulation117Attire {
		content, exists := data.ReadAsset(want.iconPath)
		if !exists {
			continue
		}
		digest := sha256.Sum256(content)
		if got := hex.EncodeToString(digest[:]); got != want.iconSHA256 {
			t.Errorf("icon asset %q sha256 = %s, want the v1.7.1 asset %s",
				want.iconPath, got, want.iconSHA256)
		}
	}
	source, found := manifestSource(data.Manifest, regulation117IconSource)
	if !found {
		t.Fatalf("manifest has no %q source", regulation117IconSource)
	}
	if source.Version != regulation117IconRelease {
		t.Errorf("icon source version = %q, want the v1.7.1 release commit %q",
			source.Version, regulation117IconRelease)
	}
}

// TestRegulation117SpectralSteedAttireGoodsAreComplete is the public-layer
// contract of the three Spectral Steed Attire key items. The stable v1.7.1
// release settled what they are: DLC key items that live in the inventory only,
// never reach storage and stay out of the general item database until the World
// feature consumes them. Everything below is stated inline, so an ingest that
// silently renames them to the v1.7.1 application names, opens storage, drops
// the noDatabase flag or invents a location fails here.
func TestRegulation117SpectralSteedAttireGoodsAreComplete(t *testing.T) {
	catalog := newRealCatalog(t)
	for _, want := range regulation117Attire {
		resource, err := catalog.ResourceByKindAndKey(schema.ResourceKindItem, want.key)
		if err != nil {
			t.Errorf("%s: resolve key %q: %v", want.name, want.key, err)
			continue
		}
		if resource.Key != want.key {
			t.Errorf("%s: key = %q, want %q", want.name, resource.Key, want.key)
		}
		item := resource.Item
		if item == nil {
			t.Errorf("%s: resource carries no item document", want.name)
			continue
		}

		assertEqual(t, want.name, "gameID", item.GameID.Value, want.gameID)
		assertEqual(t, want.name, "family", item.Family.Value, schema.ItemFamilyGoods)
		assertEqual(t, want.name, "category", item.Category.Value, "key_items")
		if item.Subcategory.Known {
			t.Errorf("%s: subcategory = %q, want an unknown fact: no reviewed subcategory exists",
				want.name, item.Subcategory.Value)
		}
		if _, found := catalog.ItemByGameID(want.gameID); !found {
			t.Errorf("%s: 0x%08X does not resolve by game ID", want.name, want.gameID)
		}

		presentation := item.Presentation
		assertEqual(t, want.name, "name", presentation.Name.Value, want.name)
		assertEqual(t, want.name, "name source", presentation.Name.Provenance.Source,
			schema.SourceID("game_text_goods_name_dlc02"))
		assertEqual(t, want.name, "caption", presentation.Caption.Value, want.caption)
		assertEqual(t, want.name, "caption source", presentation.Caption.Provenance.Source,
			schema.SourceID("game_text_goods_caption_dlc02"))
		assertEqual(t, want.name, "description", presentation.Description.Value, want.description)
		assertEqual(t, want.name, "description source", presentation.Description.Provenance.Source,
			schema.SourceID("game_text_goods_info_dlc02"))
		if presentation.Location.Known || presentation.Location.Value != "" {
			t.Errorf("%s: location = %#v, want an unknown fact: no location was ever confirmed",
				want.name, presentation.Location)
		}

		metadata := presentation.TextMetadata
		assertEqual(t, want.name, "captionSource", metadata.CaptionSource.Value, "fmg")
		assertEqual(t, want.name, "descriptionSource", metadata.DescriptionSource.Value, "fmg")
		if metadata.LocationSource.Known {
			t.Errorf("%s: locationSource = %q, want an unknown fact",
				want.name, metadata.LocationSource.Value)
		}
		assertEqual(t, want.name, "dlcSource", metadata.DLCSource.Value, "dlc02")

		assertEqual(t, want.name, "iconPath", presentation.IconPath.Value, want.iconPath)
		assertEqual(t, want.name, "icon source", presentation.IconPath.Provenance.Source,
			regulation117IconSource)

		goods := item.Goods
		if goods == nil {
			t.Errorf("%s: goods section is missing", want.name)
			continue
		}
		assertEqual(t, want.name, "sourceRowID", goods.SourceRowID.Value, want.rowID)
		assertEqual(t, want.name, "iconID", goods.IconID.Value, want.iconID)
		assertEqual(t, want.name, "sortID", goods.SortID.Value, want.sortID)
		assertEqual(t, want.name, "sortGroupID", goods.SortGroupID.Value, want.sortGroupID)
		assertEqual(t, want.name, "goodsType", goods.GoodsType.Value, uint16(1))
		assertClose(t, want.name, "weight", goods.Weight.Value, 0)
		assertEqual(t, want.name, "maxQuantity", goods.MaxQuantity.Value, uint32(1))
		// The raw parameter fact stays 1 even though the item can never be
		// deposited; the effective storage limit below is what the writer reads.
		assertEqual(t, want.name, "maxRepository", goods.MaxRepository.Value, uint32(1))
		assertEqual(t, want.name, "isDepositable", goods.IsDepositable.Value, false)
		assertEqual(t, want.name, "isConsumable", goods.IsConsumable.Value, false)
		assertEqual(t, want.name, "isDiscardable", goods.IsDiscardable.Value, false)
		assertEqual(t, want.name, "isDroppable", goods.IsDroppable.Value, false)
		assertEqual(t, want.name, "isEquipable", goods.IsEquipable.Value, false)

		storage := item.Storage
		assertEqual(t, want.name, "recordMode", storage.RecordMode.Value,
			schema.RecordModeQuantityStack)
		assertEqual(t, want.name, "maxInventory", storage.MaxInventory.Value, uint32(1))
		// isDeposit=0 means storage is closed, and a non-zero effective limit is
		// exactly what would open the Move to Storage path for these items.
		assertEqual(t, want.name, "maxStorage", storage.MaxStorage.Value, uint32(0))

		capabilities := item.Capabilities
		assertCapabilityDisabled(t, want.name, "stack", capabilities.Stack)
		assertCapabilityDisabled(t, want.name, "equipment", capabilities.Equipment)
		assertCapabilityDisabled(t, want.name, "upgrade", capabilities.Upgrade)
		assertCapabilityDisabled(t, want.name, "infusion", capabilities.Infusion)
		assertCapabilityDisabled(t, want.name, "ashOfWarMount", capabilities.AshOfWarMount)

		safety := item.Safety
		for _, flag := range []struct {
			label string
			got   bool
			want  bool
		}{
			{"cutContent", safety.CutContent.Value, false},
			{"banRisk", safety.BanRisk.Value, false},
			{"dlc", safety.DLC.Value, true},
			{"noDatabase", safety.NoDatabase.Value, true},
			{"scalesWithNG", safety.ScalesWithNG.Value, false},
			{"preOrder", safety.PreOrder.Value, false},
		} {
			assertEqual(t, want.name, "safety."+flag.label, flag.got, flag.want)
		}

		if len(item.Variants) != 0 || len(item.Aliases) != 0 || len(item.Unlocks) != 0 {
			t.Errorf("%s: variants/aliases/unlocks = %d/%d/%d, want none: the attire has no "+
				"variant, no confirmed alias contract and no unlock record",
				want.name, len(item.Variants), len(item.Aliases), len(item.Unlocks))
		}
		// The 6700..6703 attire event flags belong to a later World contract. A
		// document that already claims one of them would make this ingest decide
		// a contract it has no evidence for.
		for _, link := range item.Links.RelatedEventFlags {
			if link.EventFlagID.Value >= 6700 && link.EventFlagID.Value <= 6703 {
				t.Errorf("%s: relates to attire event flag %d, which no ingested contract owns yet",
					want.name, link.EventFlagID.Value)
			}
		}
		for _, unlock := range item.Unlocks {
			if unlock.EventFlagID.Value >= 6700 && unlock.EventFlagID.Value <= 6703 {
				t.Errorf("%s: unlocks attire event flag %d, which no ingested contract owns yet",
					want.name, unlock.EventFlagID.Value)
			}
		}
	}
}

// TestRegulation117UnsupportedContentNeverReachesTheCatalog fails closed on the
// scope decision: the two altered protector variants and the two unnamed
// technical armament rows are evidence, not catalog items, and none of their
// icons may ship. The three Spectral Steed Attire goods left this list when they
// were ingested; TestRegulation117SpectralSteedAttireGoodsAreComplete is what
// protects them now.
func TestRegulation117UnsupportedContentNeverReachesTheCatalog(t *testing.T) {
	catalog := newRealCatalog(t)
	for _, excluded := range []struct {
		gameID uint32
		reason string
	}{
		{0x10000000 | 5351100, "Silver Grooved Armor (Altered) is an unsupported altered variant"},
		{0x10000000 | 5361000, "Leontiel's Hat (Altered) is an unsupported altered variant"},
		{3910000, "unnamed technical armament row"},
		{13900000, "unnamed technical armament row"},
	} {
		if resource, found := catalog.ItemByGameID(excluded.gameID); found {
			t.Errorf("0x%08X resolved to %q but %s",
				excluded.gameID, resource.Key, excluded.reason)
		}
	}

	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	for _, path := range []string{
		"assets/icons/items/chest/silver_grooved_armor_altered.png",
		"assets/icons/items/head/leontiel_s_hat_altered.png",
	} {
		if _, exists := data.ReadAsset(path); exists {
			t.Errorf("altered-variant icon %q ships", path)
		}
	}
}

type regulation117Change struct {
	gameID  uint32
	name    string
	field   string
	stale   float64
	current float64
}

// regulation117WeaponChanges is every pre-existing armament family whose values
// 1.17 changed, with both the value it must no longer carry and the one it must
// carry now. Stating the stale value makes a silent revert to the 1.16 dump a
// failure rather than a no-op.
var regulation117WeaponChanges = []regulation117Change{
	{gameID: 2530000, name: "Carian Sorcery Sword", field: "attackPhysical", stale: 69, current: 38},
	{gameID: 8020000, name: "Dismounter", field: "sortID", stale: 1700000, current: 1700500},
	{gameID: 32000000, name: "Dragon Towershield", field: "guardBoost", stale: 69, current: 71},
	{gameID: 32000000, name: "Dragon Towershield", field: "weight", stale: 17.5, current: 16.5},
	{gameID: 32020000, name: "Distinguished Greatshield", field: "guardBoost", stale: 68, current: 71},
	{gameID: 32020000, name: "Distinguished Greatshield", field: "weight", stale: 17.0, current: 16.0},
	{gameID: 32030000, name: "Crucible Hornshield", field: "guardBoost", stale: 60, current: 70},
	{gameID: 32040000, name: "Dragonclaw Shield", field: "guardBoost", stale: 61, current: 70},
	{gameID: 32050000, name: "Briar Greatshield", field: "reinforceTypeID", stale: 8200, current: 8600},
	{gameID: 32050000, name: "Briar Greatshield", field: "guardBoost", stale: 58, current: 62},
	{gameID: 32050000, name: "Briar Greatshield", field: "weight", stale: 9.5, current: 8.5},
	{gameID: 32080000, name: "Erdtree Greatshield", field: "guardBoost", stale: 60, current: 69},
	{gameID: 32080000, name: "Erdtree Greatshield", field: "weight", stale: 13.5, current: 11.5},
	{gameID: 32090000, name: "Golden Beast Crest Shield", field: "reinforceTypeID", stale: 8200, current: 8600},
	{gameID: 32090000, name: "Golden Beast Crest Shield", field: "guardBoost", stale: 60, current: 63},
	{gameID: 32090000, name: "Golden Beast Crest Shield", field: "weight", stale: 12.5, current: 10.5},
	{gameID: 32120000, name: "Jellyfish Shield", field: "guardBoost", stale: 52, current: 60},
	{gameID: 32140000, name: "Icon Shield", field: "guardBoost", stale: 59, current: 61},
	{gameID: 32140000, name: "Icon Shield", field: "weight", stale: 11.5, current: 9.0},
	{gameID: 32150000, name: "One-Eyed Shield", field: "guardBoost", stale: 67, current: 69},
	{gameID: 32170000, name: "Spiked Palisade Shield", field: "reinforceTypeID", stale: 8200, current: 8600},
	{gameID: 32170000, name: "Spiked Palisade Shield", field: "guardBoost", stale: 59, current: 63},
	{gameID: 32170000, name: "Spiked Palisade Shield", field: "weight", stale: 11.5, current: 9.5},
	{gameID: 32190000, name: "Manor Towershield", field: "guardBoost", stale: 67, current: 71},
	{gameID: 32190000, name: "Manor Towershield", field: "weight", stale: 16.0, current: 15.5},
	{gameID: 32200000, name: "Crossed-Tree Towershield", field: "guardBoost", stale: 67, current: 71},
	{gameID: 32200000, name: "Crossed-Tree Towershield", field: "weight", stale: 16.0, current: 15.5},
	{gameID: 32210000, name: "Inverted Hawk Towershield", field: "guardBoost", stale: 65, current: 70},
	{gameID: 32210000, name: "Inverted Hawk Towershield", field: "weight", stale: 16.0, current: 15.5},
	{gameID: 32220000, name: "Ant's Skull Plate", field: "guardBoost", stale: 63, current: 70},
	{gameID: 32220000, name: "Ant's Skull Plate", field: "weight", stale: 13.5, current: 12.5},
	{gameID: 32230000, name: "Redmane Greatshield", field: "guardBoost", stale: 65, current: 70},
	{gameID: 32240000, name: "Eclipse Crest Greatshield", field: "guardBoost", stale: 67, current: 71},
	{gameID: 32250000, name: "Cuckoo Greatshield", field: "guardBoost", stale: 64, current: 70},
	{gameID: 32260000, name: "Golden Greatshield", field: "guardBoost", stale: 70, current: 72},
	{gameID: 32270000, name: "Gilded Greatshield", field: "guardBoost", stale: 69, current: 71},
	{gameID: 32270000, name: "Gilded Greatshield", field: "weight", stale: 17.5, current: 16.5},
	{gameID: 32280000, name: "Haligtree Crest Greatshield", field: "weight", stale: 18.5, current: 17.0},
	{gameID: 32290000, name: "Wooden Greatshield", field: "reinforceTypeID", stale: 8200, current: 8600},
	{gameID: 32290000, name: "Wooden Greatshield", field: "guardBoost", stale: 56, current: 62},
	{gameID: 32300000, name: "Lordsworn's Shield", field: "reinforceTypeID", stale: 8200, current: 8600},
	{gameID: 32300000, name: "Lordsworn's Shield", field: "guardBoost", stale: 60, current: 63},
	{gameID: 32300000, name: "Lordsworn's Shield", field: "weight", stale: 10.0, current: 9.0},
	{gameID: 32500000, name: "Black Steel Greatshield", field: "guardBoost", stale: 69, current: 71},
	{gameID: 32500000, name: "Black Steel Greatshield", field: "weight", stale: 19.5, current: 18.0},
	{gameID: 61500000, name: "Firespark Perfume Bottle", field: "attackFire", stale: 110, current: 118},
	{gameID: 61510000, name: "Chilling Perfume Bottle", field: "attackMagic", stale: 105, current: 112},
	{gameID: 61520000, name: "Frenzyflame Perfume Bottle", field: "attackFire", stale: 105, current: 112},
	{gameID: 61530000, name: "Lightning Perfume Bottle", field: "attackLightning", stale: 110, current: 118},
	{gameID: 61540000, name: "Deadly Poison Perfume Bottle", field: "attackPhysical", stale: 92, current: 98},
	{gameID: 62500000, name: "Dueling Shield", field: "sortID", stale: 7400000, current: 7400500},
	{gameID: 64500000, name: "Backhand Blade", field: "sortID", stale: 1750000, current: 1750500},
	{gameID: 66500000, name: "Great Katana", field: "sortID", stale: 1850000, current: 1850500},
	{gameID: 67500000, name: "Milady", field: "sortID", stale: 1150000, current: 1150500},
}

// regulation117ArmourChanges is the sortId renumbering 1.17 applied to the nine
// pre-existing protector rows that collided with the new sets.
var regulation117ArmourChanges = []regulation117Change{
	{gameID: 0x10030D40, name: "Banished Knight Helm", field: "sortID", stale: 509000, current: 509005},
	{gameID: 0x10030DA4, name: "Banished Knight Armor", field: "sortID", stale: 609000, current: 609005},
	{gameID: 0x10030E08, name: "Banished Knight Gauntlets", field: "sortID", stale: 708000, current: 708005},
	{gameID: 0x10030E6C, name: "Banished Knight Greaves", field: "sortID", stale: 808000, current: 808005},
	{gameID: 0x104E2000, name: "Freyja's Helm", field: "sortID", stale: 506131, current: 506135},
	{gameID: 0x104E2064, name: "Freyja's Armor", field: "sortID", stale: 606081, current: 606085},
	{gameID: 0x104E20C8, name: "Freyja's Gauntlets", field: "sortID", stale: 705131, current: 705135},
	{gameID: 0x104E212C, name: "Freyja's Greaves", field: "sortID", stale: 805121, current: 805125},
	{gameID: 0x104E244C, name: "Freyja's Armor (Altered)", field: "sortID", stale: 606082, current: 606086},
}

// TestRegulation117RefreshesPreExistingItems is the regression for the data
// refresh: each changed family carries the 1.17 value and no longer the 1.16
// one, on the base row and on every affinity variant alike.
func TestRegulation117RefreshesPreExistingItems(t *testing.T) {
	catalog := newRealCatalog(t)
	for _, change := range regulation117WeaponChanges {
		resource, found := catalog.ItemByGameID(change.gameID)
		if !found || resource.Item == nil || resource.Item.Weapon == nil {
			t.Errorf("%s: 0x%08X does not resolve to an armament", change.name, change.gameID)
			continue
		}
		if got := weaponFieldValue(t, resource.Item.Weapon, change.field); got != change.current {
			t.Errorf("%s %s = %v, want the 1.17 value %v (1.16 was %v)",
				change.name, change.field, got, change.current, change.stale)
		}
		assertVariantsFollowTheFamily(t, change, resource.Item)
	}
	for _, change := range regulation117ArmourChanges {
		resource, found := catalog.ItemByGameID(change.gameID)
		if !found || resource.Item == nil || resource.Item.Armor == nil {
			t.Errorf("%s: 0x%08X does not resolve to an armour piece", change.name, change.gameID)
			continue
		}
		if got := float64(resource.Item.Armor.SortID.Value); got != change.current {
			t.Errorf("%s sortID = %v, want the 1.17 value %v (1.16 was %v)",
				change.name, got, change.current, change.stale)
		}
	}
}

// assertVariantsFollowTheFamily is the invariant behind the reported failure
// mode: refreshing only the base row leaves every infused copy of the shield on
// the old guard boost and weight. sortId is renumbered per variant, so it is
// checked as a band around the family base instead of an exact value.
func assertVariantsFollowTheFamily(
	t *testing.T,
	change regulation117Change,
	item *schema.ItemDocument,
) {
	t.Helper()
	for _, variant := range item.Variants {
		data := variant.Data.Weapon
		if data == nil {
			t.Errorf("%s: variant 0x%08X carries no weapon section", change.name, variant.GameID.Value)
			continue
		}
		switch change.field {
		case "guardBoost", "weight":
			if got := weaponFieldValue(t, data, change.field); got != change.current {
				t.Errorf("%s variant 0x%08X %s = %v, want %v",
					change.name, variant.GameID.Value, change.field, got, change.current)
			}
		case "sortID":
			base := float64(item.Weapon.SortID.Value)
			got := float64(data.SortID.Value)
			if got < base || got > base+99 {
				t.Errorf("%s variant 0x%08X sortID = %v, outside the family band [%v, %v]",
					change.name, variant.GameID.Value, got, base, base+99)
			}
		}
	}
}

func weaponFieldValue(t *testing.T, weapon *schema.WeaponData, field string) float64 {
	t.Helper()
	switch field {
	case "guardBoost":
		return float64(weapon.GuardBoost.Value)
	case "weight":
		return weapon.Weight.Value
	case "sortID":
		return float64(weapon.SortID.Value)
	case "reinforceTypeID":
		return float64(weapon.ReinforceTypeID.Value)
	case "attackPhysical":
		return float64(weapon.AttackPhysical.Value)
	case "attackMagic":
		return float64(weapon.AttackMagic.Value)
	case "attackFire":
		return float64(weapon.AttackFire.Value)
	case "attackLightning":
		return float64(weapon.AttackLightning.Value)
	default:
		t.Fatalf("unsupported field %q", field)
		return 0
	}
}

// TestRegulation117ManifestDeclaresItsSources proves the catalog states which
// regulation it now carries and where the new text and assets came from. A
// document set refreshed to 1.17 under a manifest still claiming 1.16 is a
// provenance lie, not a cosmetic mismatch.
func TestRegulation117ManifestDeclaresItsSources(t *testing.T) {
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	if data.Manifest.GameVersion != "1.17-class" {
		t.Errorf("manifest gameVersion = %q, want %q",
			data.Manifest.GameVersion, "1.17-class")
	}
	for _, want := range []struct {
		id       schema.SourceID
		kind     string
		location string
		evidence schema.EvidenceLevel
	}{
		{"game_text_weapon_caption_base", "game_text_fmg_extract",
			"regulation.bin/msg/engus/item.msgbnd/WeaponCaption.fmg", schema.EvidenceGameData},
		{"game_text_protector_caption_base", "game_text_fmg_extract",
			"regulation.bin/msg/engus/item.msgbnd/ProtectorCaption.fmg", schema.EvidenceGameData},
		// The names of the two 1.17 starting classes are captions 297140 and
		// 297141, which live only in the menu_dlc02 GR_MenuText, never in the
		// base menu.msgbnd one the ten original classes are named from.
		{"game_text_gr_menu_text_dlc02", "game_text_fmg_extract",
			"regulation.bin/msg/engus/menu_dlc02.msgbnd/GR_MenuText.fmg", schema.EvidenceGameData},
		// CharMakeMenuListItemParam is what ties each of those captions to a
		// starting class ID, so the manifest has to declare it as a source of its
		// own instead of leaving the link unattributed.
		{"regulation_char_make_menu_list_item_param", "regulation_parameter_csv",
			"regulation.bin/csv/CharMakeMenuListItemParam.csv", schema.EvidenceRegulation},
		{regulation117IconSource, "product_item_assets",
			"saveforge-1.7.1/frontend/public/items", schema.EvidenceVerifiedResearch},
		{"curated_regulation_117", "curated_regulation_117_ingest",
			"saveforge-2.0/gamecatalog/curation/regulation_117", schema.EvidenceCurated},
	} {
		source, found := manifestSource(data.Manifest, want.id)
		if !found {
			t.Errorf("manifest has no source %q", want.id)
			continue
		}
		if source.Kind != want.kind || source.Location != want.location ||
			source.Evidence != want.evidence {
			t.Errorf("source %q = %q/%q/%q, want %q/%q/%q", want.id,
				source.Kind, source.Location, source.Evidence,
				want.kind, want.location, want.evidence)
		}
		if source.Version == "" {
			t.Errorf("source %q has no version", want.id)
		}
	}
}

func manifestSource(manifest schema.Manifest, id schema.SourceID) (schema.DataSource, bool) {
	for _, source := range manifest.Sources {
		if source.ID == id {
			return source, true
		}
	}
	return schema.DataSource{}, false
}

// assertCapabilityDisabled is the "this item cannot do that" contract: the fact
// has to be known, so it is a decision rather than a gap, disabled, and carry no
// rules that a later reader could act on anyway.
func assertCapabilityDisabled[T any](t *testing.T, label, name string, capability schema.Capability[T]) {
	t.Helper()
	if !capability.Known || capability.Enabled || capability.Rules != nil {
		t.Errorf("%s: %s capability = %#v, want a known, disabled capability with no rules",
			label, name, capability)
	}
}

func assertEqual[T comparable](t *testing.T, label, field string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s %s = %v, want %v", label, field, got, want)
	}
}

func assertClose(t *testing.T, label, field string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s %s = %v, want %v", label, field, got, want)
	}
}
