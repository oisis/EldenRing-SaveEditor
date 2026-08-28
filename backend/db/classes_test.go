package db

import (
	"testing"
)

func TestGetClassStats_AllClasses(t *testing.T) {
	for id := uint8(0); id <= 11; id++ {
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

func TestGetClassStats_DLCClasses(t *testing.T) {
	// Regulation 1.17 classes, confirmed against menu_dlc01.msgbnd GR_MenuText.fmg
	// entries 288110 and 288111.
	want := []ClassStats{
		{ID: 10, Name: "Idus Knight", Level: 7, Vigor: 10, Mind: 12, Endurance: 11, Strength: 13, Dexterity: 15, Intelligence: 8, Faith: 11, Arcane: 6},
		{ID: 11, Name: "Heavy Knight", Level: 10, Vigor: 14, Mind: 8, Endurance: 17, Strength: 15, Dexterity: 11, Intelligence: 7, Faith: 8, Arcane: 9},
	}
	for _, w := range want {
		got := GetClassStats(w.ID)
		if got == nil {
			t.Fatalf("GetClassStats(%d) returned nil", w.ID)
		}
		if *got != w {
			t.Errorf("GetClassStats(%d) = %+v, want %+v", w.ID, *got, w)
		}
	}
}

func TestGetClassStats_Unknown(t *testing.T) {
	cs := GetClassStats(12)
	if cs != nil {
		t.Errorf("expected nil for unknown class 12, got %v", cs)
	}
	cs = GetClassStats(255)
	if cs != nil {
		t.Errorf("expected nil for unknown class 255, got %v", cs)
	}
}
