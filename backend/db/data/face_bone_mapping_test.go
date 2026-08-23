package data

import "testing"

// TestLookupFaceBonePartsID pins the shared Face / Bone Structure table used by
// both Type A and Type B: the exact six UI→PartsId pairs and the no-fallback
// contract for anything outside 1-6.
func TestLookupFaceBonePartsID(t *testing.T) {
	for ui, want := range map[uint8]uint8{1: 0, 2: 10, 3: 20, 4: 30, 5: 40, 6: 50} {
		got, ok := LookupFaceBonePartsID(ui)
		if !ok || got != want {
			t.Errorf("LookupFaceBonePartsID(%d) = (%d, %v), want (%d, true)", ui, got, ok, want)
		}
	}
	for _, ui := range []uint8{0, 7, 99, 255} {
		if got, ok := LookupFaceBonePartsID(ui); ok || got != 0 {
			t.Errorf("LookupFaceBonePartsID(%d) = (%d, %v), want (0, false)", ui, got, ok)
		}
	}
}
