package db

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

func TestCanWeaponChangeAffinity_RepresentativeClasses(t *testing.T) {
	cases := []struct {
		name   string
		itemID uint32
		want   bool
	}{
		{"Dagger", 0x000F4240, true},
		{"Brass Shield", 0x01DB0190, true},
		{"Dueling Shield", 0x03B9ACA0, true},
		{"Shortbow", 0x02625A00, false},
		{"Composite Bow", 0x02631D50, false},
		{"Longbow", 0x02719C40, false},
		{"Greatbow", 0x02817AC0, false},
		{"Soldier's Crossbow", 0x029020C0, false},
		{"Glintstone Staff", 0x01F78A40, false},
		{"Finger Seal", 0x0206CC80, false},
		{"Steel-Wire Torch", 0x016E8420, false},
		{"Icerind Hatchet", 0x00D6D800, false},
		{"Firespark Perfume Bottle", 0x03AA6A60, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanWeaponChangeAffinity(tc.itemID); got != tc.want {
				t.Fatalf("CanWeaponChangeAffinity(0x%08X)=%v, want %v", tc.itemID, got, tc.want)
			}
		})
	}
}

func TestWeaponEditingMetadata_CoversAllApplicationWeapons(t *testing.T) {
	sets := []map[uint32]data.ItemData{
		data.Weapons,
		data.RangedAndCatalysts,
		data.Shields,
	}

	covered := 0
	changeable := 0
	for _, items := range sets {
		for id, source := range items {
			mount, ok := data.WeaponGemMounts[id]
			if !ok {
				t.Errorf("%s (0x%08X): missing EquipParamWeapon metadata", source.Name, id)
				continue
			}
			item := GetItemData(id)
			if item.Name == "" {
				t.Errorf("%s (0x%08X): missing from DB index", source.Name, id)
				continue
			}
			if item.WepType != mount.WepType {
				t.Errorf("%s (0x%08X): WepType=%d, want %d", source.Name, id, item.WepType, mount.WepType)
			}
			if item.GemMountType != mount.GemMountType {
				t.Errorf("%s (0x%08X): GemMountType=%d, want %d", source.Name, id, item.GemMountType, mount.GemMountType)
			}
			if item.CanChangeAffinity != mount.CanChangeAffinity {
				t.Errorf("%s (0x%08X): CanChangeAffinity=%v, want %v",
					source.Name, id, item.CanChangeAffinity, mount.CanChangeAffinity)
			}
			if item.CanChangeAffinity {
				changeable++
			}
			covered++
		}
	}

	if covered != 668 {
		t.Fatalf("covered %d application weapon records, want 668", covered)
	}
	if changeable != 416 {
		t.Fatalf("changeable affinity records=%d, want 416 (blocked=%d, want 252)",
			changeable, covered-changeable)
	}
}
