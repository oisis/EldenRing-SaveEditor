package core

import (
	"math"
	"testing"
)

func TestBaseEquipLoad(t *testing.T) {
	for _, tc := range []struct {
		endurance uint32
		want      float64
	}{
		{0, 45.0},
		{8, 45.0},
		{9, 46.6},
		{20, 64.1},
		{25, 72.0},
		{60, 120.0},
		{99, 160.0},
		{100, 160.0},
	} {
		if got := BaseEquipLoad(tc.endurance); got != tc.want {
			t.Errorf("BaseEquipLoad(%d) = %.1f, want %.1f", tc.endurance, got, tc.want)
		}
	}
}

func TestClassifyEquipLoad(t *testing.T) {
	for _, tc := range []struct {
		name    string
		current float64
		maximum float64
		want    EquipLoadClass
	}{
		{"light", 29.9, 100, EquipLoadLight},
		{"medium at threshold", 30, 100, EquipLoadMedium},
		{"medium", 69.9, 100, EquipLoadMedium},
		{"heavy at threshold", 70, 100, EquipLoadHeavy},
		{"heavy", 99.9, 100, EquipLoadHeavy},
		{"overloaded at threshold", 100, 100, EquipLoadOverloaded},
		{"overloaded above threshold", 101, 100, EquipLoadOverloaded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyEquipLoad(tc.current, tc.maximum); got != tc.want {
				t.Errorf("ClassifyEquipLoad(%g, %g) = %q, want %q", tc.current, tc.maximum, got, tc.want)
			}
		})
	}
}

func TestMaxEquipLoad(t *testing.T) {
	for _, tc := range []struct {
		name           string
		endurance      uint32
		enduranceBonus int
		equipLoadRate  float64
		want           float64
	}{
		{"base only", 20, 0, 0, 64.1},
		{"endurance bonus uses curve", 20, 2, 0, 67.2},
		{"endurance bonus caps at 99", 99, 5, 0, 160},
		{"great jars arsenal", 20, 0, 0.19, 76.279},
		{"arsenal plus erdtree favor additively", 20, 0, 0.27, 81.407},
		{"head and talisman rates", 20, 0, 0.235, 79.1635},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := MaxEquipLoad(tc.endurance, tc.enduranceBonus, tc.equipLoadRate); math.Abs(got-tc.want) > 0.000001 {
				t.Errorf("MaxEquipLoad(%d, %d, %.3f) = %g, want %g", tc.endurance, tc.enduranceBonus, tc.equipLoadRate, got, tc.want)
			}
		})
	}
}
