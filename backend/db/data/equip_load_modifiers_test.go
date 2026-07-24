package data

import "testing"

func TestEquipLoadModifiers(t *testing.T) {
	for _, tc := range []struct {
		id             uint32
		enduranceBonus int
		equipLoadRate  float64
	}{
		{0x20000406, 0, 0.15},
		{0x20000407, 0, 0.17},
		{0x20000408, 0, 0.19},
		{0x20000410, 0, 0.05},
		{0x20000411, 0, 0.065},
		{0x20000412, 0, 0.08},
		{0x2000041A, 3, 0},
		{0x2000041B, 5, 0},
		{0x100F1B30, 2, 0},
		{0x10108A60, 2, 0},
		{0x104F0A60, 0, 0.045},
	} {
		got, ok := EquipLoadModifiers[tc.id]
		if !ok {
			t.Errorf("modifier 0x%08X is missing", tc.id)
			continue
		}
		if got.EnduranceBonus != tc.enduranceBonus || got.EquipLoadRate != tc.equipLoadRate {
			t.Errorf("modifier 0x%08X = %+v, want EnduranceBonus=%d EquipLoadRate=%g", tc.id, got, tc.enduranceBonus, tc.equipLoadRate)
		}
	}
}
