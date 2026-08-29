package core

import (
	"strings"
	"testing"
)

// TestDiagnoseSaveCorruption_ClassRange verifies the Class bound follows
// Regulation 1.17, which added Idus Knight (10) and Heavy Knight (11) to the
// starting classes. Checked through the public entry point that feeds the
// application's diagnostics.
func TestDiagnoseSaveCorruption_ClassRange(t *testing.T) {
	cases := []struct {
		class    uint8
		wantWarn bool
	}{
		{9, false},  // Wretch — last pre-1.17 class
		{10, false}, // Idus Knight
		{11, false}, // Heavy Knight
		{12, true},  // beyond the known classes
	}

	for _, tc := range cases {
		slot := &SaveSlot{Version: 0x4C, Data: make([]byte, SlotSize)}
		slot.Player.Level = 89 // sum(attrs) - 79
		slot.Player.Vigor = 21
		slot.Player.Mind = 21
		slot.Player.Endurance = 21
		slot.Player.Strength = 21
		slot.Player.Dexterity = 21
		slot.Player.Intelligence = 21
		slot.Player.Faith = 21
		slot.Player.Arcane = 21
		slot.Player.Class = tc.class

		var got string
		for _, e := range DiagnoseSaveCorruption(slot, 0).Issues {
			if strings.HasPrefix(e.Description, "Class=") {
				got = e.Description
			}
		}

		switch {
		case tc.wantWarn && got != "Class=12 out of range [0, 11]":
			t.Errorf("Class=%d: got %q, want %q", tc.class, got, "Class=12 out of range [0, 11]")
		case !tc.wantWarn && got != "":
			t.Errorf("Class=%d: unexpected warning %q", tc.class, got)
		}
	}
}
