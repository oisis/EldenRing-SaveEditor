package data

import (
	"os"
	"path/filepath"
	"testing"
)

// Regulation 1.17 added eight armament families. These tests pin the whole
// shape of that addition: the base records, the affinity fan-out for the
// standard-smithing families, the somber families that must stay single
// records, the two technical duplicate rows that must never become items, and
// the generated tables every one of the 80 public IDs depends on.

// affinityVariants is the canonical +0x64 affinity ladder used by every
// standard-smithing armament family in the DB.
var affinityVariants = []string{
	"", "Heavy ", "Keen ", "Quality ", "Fire ", "Flame Art ", "Lightning ",
	"Sacred ", "Magic ", "Cold ", "Poison ", "Blood ", "Occult ",
}

var regulation117Armaments = []struct {
	base        uint32
	name        string
	category    string
	subCategory string
	wepType     uint16
	maxUpgrade  uint32
	infusable   bool // true → 13 affinity records; false → somber, base record only
	items       map[uint32]ItemData
	iconPath    string
}{
	{0x00365240, "Leontiel's Greatsword", "melee_armaments", SubcatMeleeGreatswords, 5, 10, false, Weapons, "items/melee_armaments/leontiels_greatsword.png"},
	{0x00822850, "Hefty Scimitar", "melee_armaments", SubcatMeleeCurvedGreatswords, 11, 25, true, Weapons, "items/melee_armaments/hefty_scimitar.png"},
	{0x00CE2570, "Golden Order Flail", "melee_armaments", SubcatMeleeFlails, 24, 10, false, Weapons, "items/melee_armaments/golden_order_flail.png"},
	{0x01E14320, "Silver Grooved Shield", "shields", SubcatShieldsMedium, 67, 25, true, Shields, "items/shields/silver_grooved_shield.png"},
	{0x03B9FAC0, "Ritual Thrusting Shield", "shields", SubcatShieldsThrusting, 90, 25, true, Shields, "items/shields/ritual_thrusting_shield.png"},
	{0x03D8A650, "Reverse-Bladed Sword", "melee_armaments", SubcatMeleeBackhandBlades, 92, 25, true, Weapons, "items/melee_armaments/reverse_bladed_sword.png"},
	{0x03F72AD0, "Reed Great Katana", "melee_armaments", SubcatMeleeGreatKatanas, 94, 25, true, Weapons, "items/melee_armaments/reed_great_katana.png"},
	{0x04066D10, "Idus Sword", "melee_armaments", SubcatMeleeLightGreatswords, 93, 25, true, Weapons, "items/melee_armaments/idus_sword.png"},
}

// regulation117IDs returns every public item ID the 1.17 addition introduced.
func regulation117IDs() []uint32 {
	var ids []uint32
	for _, f := range regulation117Armaments {
		if !f.infusable {
			ids = append(ids, f.base)
			continue
		}
		for i := range affinityVariants {
			ids = append(ids, f.base+uint32(i)*0x64)
		}
	}
	return ids
}

func TestRegulation117Armaments_BaseRecords(t *testing.T) {
	for _, f := range regulation117Armaments {
		t.Run(f.name, func(t *testing.T) {
			item, ok := f.items[f.base]
			if !ok {
				t.Fatalf("0x%08X (%s) missing from the item DB", f.base, f.name)
			}
			if item.Name != f.name {
				t.Errorf("Name = %q, want %q", item.Name, f.name)
			}
			if item.Category != f.category {
				t.Errorf("Category = %q, want %q", item.Category, f.category)
			}
			if item.SubCategory != f.subCategory {
				t.Errorf("SubCategory = %q, want %q", item.SubCategory, f.subCategory)
			}
			if item.MaxUpgrade != f.maxUpgrade {
				t.Errorf("MaxUpgrade = %d, want %d", item.MaxUpgrade, f.maxUpgrade)
			}
			if item.MaxInventory != 1 || item.MaxStorage != 1 {
				t.Errorf("MaxInventory/MaxStorage = %d/%d, want 1/1", item.MaxInventory, item.MaxStorage)
			}
			if got := weaponWepType[f.base]; got != f.wepType {
				t.Errorf("weaponWepType = %d, want %d", got, f.wepType)
			}
			var dlc bool
			for _, flag := range item.Flags {
				if flag == "dlc" {
					dlc = true
				}
			}
			if !dlc {
				t.Errorf("Flags = %v, want the base record to carry \"dlc\"", item.Flags)
			}
		})
	}
}

func TestRegulation117Armaments_AffinityFanOut(t *testing.T) {
	for _, f := range regulation117Armaments {
		if !f.infusable {
			continue
		}
		t.Run(f.name, func(t *testing.T) {
			for i, prefix := range affinityVariants {
				id := f.base + uint32(i)*0x64
				item, ok := f.items[id]
				if !ok {
					t.Errorf("0x%08X (%s%s) missing from the item DB", id, prefix, f.name)
					continue
				}
				if want := prefix + f.name; item.Name != want {
					t.Errorf("0x%08X: Name = %q, want %q", id, item.Name, want)
				}
				if item.MaxUpgrade != f.maxUpgrade {
					t.Errorf("0x%08X: MaxUpgrade = %d, want %d", id, item.MaxUpgrade, f.maxUpgrade)
				}
				if item.SubCategory != f.subCategory {
					t.Errorf("0x%08X: SubCategory = %q, want %q", id, item.SubCategory, f.subCategory)
				}
			}
			// One past the ladder must not exist: the family is exactly 13 wide.
			if extra := f.base + uint32(len(affinityVariants))*0x64; f.items[extra].Name != "" {
				t.Errorf("0x%08X: %q exists beyond the 13-variant affinity ladder", extra, f.items[extra].Name)
			}
		})
	}
}

func TestRegulation117Armaments_SomberFamiliesHaveNoVariants(t *testing.T) {
	for _, f := range regulation117Armaments {
		if f.infusable {
			continue
		}
		t.Run(f.name, func(t *testing.T) {
			if got := f.items[f.base].MaxUpgrade; got != 10 {
				t.Errorf("MaxUpgrade = %d, want 10 (somber path)", got)
			}
			if mount, ok := WeaponGemMounts[f.base]; !ok {
				t.Errorf("0x%08X missing from WeaponGemMounts", f.base)
			} else if mount.CanChangeAffinity {
				t.Errorf("0x%08X: CanChangeAffinity = true, want false (somber weapons block affinity)", f.base)
			}
			for i := 1; i < len(affinityVariants); i++ {
				id := f.base + uint32(i)*0x64
				if f.items[id].Name != "" {
					t.Errorf("0x%08X: %q exists, but a somber family has only its base record", id, f.items[id].Name)
				}
			}
		})
	}
}

// The two technical duplicate rows behind Leontiel's Greatsword and the Golden
// Order Flail carry no separate item identity and must stay out of the DB.
func TestRegulation117Armaments_TechnicalDuplicatesAreNotItems(t *testing.T) {
	for _, id := range []uint32{0x003BA970, 0x00D418E0} {
		for name, m := range map[string]map[uint32]ItemData{
			"Weapons": Weapons, "Shields": Shields, "RangedAndCatalysts": RangedAndCatalysts,
		} {
			if item, ok := m[id]; ok {
				t.Errorf("technical duplicate 0x%08X is exposed as %q in %s", id, item.Name, name)
			}
		}
	}
}

func TestRegulation117Armaments_GeneratedTablesCoverEveryID(t *testing.T) {
	ids := regulation117IDs()
	if len(ids) != 80 {
		t.Fatalf("regulation117IDs() = %d IDs, want 80 (6 × 13 + 2)", len(ids))
	}
	for _, id := range ids {
		if _, ok := weaponWepType[id]; !ok {
			t.Errorf("0x%08X missing from weaponWepType", id)
		}
		if _, ok := WeaponStatsV1ByID[id]; !ok {
			t.Errorf("0x%08X missing from WeaponStatsV1ByID", id)
		}
		if _, ok := ItemTexts[id]; !ok {
			t.Errorf("0x%08X missing from ItemTexts", id)
		}
	}
}

func TestRegulation117Armaments_IconsExist(t *testing.T) {
	const publicRoot = "../../../frontend/public"
	for _, f := range regulation117Armaments {
		if _, err := os.Stat(filepath.Join(publicRoot, filepath.FromSlash(f.iconPath))); err != nil {
			t.Errorf("%s: icon missing: %s (%v)", f.name, f.iconPath, err)
		}
		ids := []uint32{f.base}
		if f.infusable {
			ids = nil
			for i := range affinityVariants {
				ids = append(ids, f.base+uint32(i)*0x64)
			}
		}
		for _, id := range ids {
			if got := f.items[id].IconPath; got != f.iconPath {
				t.Errorf("0x%08X: IconPath = %q, want %q", id, got, f.iconPath)
			}
		}
	}
}
