package data

import "testing"

// Castle Sol Main Gate is the castle's entrance grace, not a boss-arena grace.
// The boss (Commander Niall) sits at Castle Sol Rooftop, which stays boss-arena.
func TestCastleSolMainGateIsNormalGrace(t *testing.T) {
	mainGate, ok := Graces[0x00012AEA]
	if !ok {
		t.Fatal("Castle Sol Main Gate grace 0x00012AEA missing")
	}
	if mainGate.BossArena {
		t.Errorf("Castle Sol Main Gate should be a normal grace, got BossArena=true")
	}

	if rooftop := Graces[0x00012AEC]; !rooftop.BossArena {
		t.Errorf("Castle Sol Rooftop should remain a boss-arena grace")
	}
}
