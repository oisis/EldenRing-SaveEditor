package core

import "testing"

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
