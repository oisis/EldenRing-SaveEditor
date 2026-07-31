package migration

import "testing"

func TestPrimaryRegulationForLegacyItemMapsExplicitFamilies(t *testing.T) {
	tests := []struct {
		name     string
		item     seed
		family   RegulationFamily
		rawRowID uint32
	}{
		{"weapon", seed{ID: 0x000F4240, Category: "melee_armaments"}, RegulationFamilyWeapon, 1000000},
		{"armor", seed{ID: 0x100F4240, Category: "head"}, RegulationFamilyProtector, 1000000},
		{"talisman", seed{ID: 0x20000406, Category: "talismans"}, RegulationFamilyAccessory, 1030},
		{"ash of war", seed{ID: 0x8000EA60, Category: "ashes_of_war"}, RegulationFamilyAshOfWar, 60000},
		{"spell", seed{ID: 0x40000FA0, Category: "sorceries"}, RegulationFamilySpell, 4000},
		{"goods", seed{ID: 0x4000012C, Category: "tools"}, RegulationFamilyGoods, 300},
		{"gesture goods", seed{ID: 0x40002328, Category: "gestures"}, RegulationFamilyGoods, 9000},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := primaryRegulationForLegacyItem(test.item)
			if err != nil {
				t.Fatalf("primaryRegulationForLegacyItem: %v", err)
			}
			if got.Family != test.family || got.RowID != test.rawRowID {
				t.Fatalf("identity = %+v, want %q/%d", got, test.family, test.rawRowID)
			}
		})
	}
}

func TestPrimaryRegulationForLegacyItemRejectsUnknownCategory(t *testing.T) {
	_, err := primaryRegulationForLegacyItem(seed{ID: 1, Category: "unknown"})
	if err == nil {
		t.Fatal("unknown category was accepted")
	}
}
