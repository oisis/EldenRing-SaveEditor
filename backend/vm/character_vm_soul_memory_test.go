package vm

import "testing"

func TestMinimumSoulMemoryForLevel_ReferenceVectors(t *testing.T) {
	for _, tc := range []struct {
		level uint32
		want  uint32
	}{
		{1, 0},
		{9, 473},
		{50, 256_598},
		{150, 7_106_585},
		{713, 1_692_560_963},
	} {
		if got := MinimumSoulMemoryForLevel(tc.level); got != tc.want {
			t.Errorf("MinimumSoulMemoryForLevel(%d)=%d, want %d", tc.level, got, tc.want)
		}
	}
}
