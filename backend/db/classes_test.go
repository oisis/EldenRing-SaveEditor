package db

import (
	"testing"
)

func TestGetClassStats_AllClasses(t *testing.T) {
	for id := uint8(0); id <= 9; id++ {
		cs := GetClassStats(id)
		if cs == nil {
			t.Errorf("GetClassStats(%d) returned nil", id)
			continue
		}
		if cs.ID != id {
			t.Errorf("class %d: ID mismatch, got %d", id, cs.ID)
		}
		if cs.Name == "" {
			t.Errorf("class %d: empty name", id)
		}
		if cs.Level < 1 || cs.Level > 10 {
			t.Errorf("class %d: level %d outside [1, 10]", id, cs.Level)
		}

		// Verify level formula: Level = sum(attrs) - 79
		sum := cs.Vigor + cs.Mind + cs.Endurance + cs.Strength +
			cs.Dexterity + cs.Intelligence + cs.Faith + cs.Arcane
		var expectedLevel uint32
		if sum > 79 {
			expectedLevel = sum - 79
		} else {
			expectedLevel = 1
		}
		if cs.Level != expectedLevel {
			t.Errorf("class %d (%s): level %d != sum(%d)-79 = %d",
				id, cs.Name, cs.Level, sum, expectedLevel)
		}
	}
}

func TestGetClassStats_Unknown(t *testing.T) {
	cs := GetClassStats(10)
	if cs != nil {
		t.Errorf("expected nil for unknown class 10, got %v", cs)
	}
	cs = GetClassStats(255)
	if cs != nil {
		t.Errorf("expected nil for unknown class 255, got %v", cs)
	}
}
