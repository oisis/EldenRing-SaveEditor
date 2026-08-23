package saveengine

import (
	"testing"
)

// TestEveryGeneratedClassDrivesSetCharacterStartingClass proves the regenerated
// GameCatalog still resolves attribute minima for all ten starting classes and
// that SetCharacterStartingClass accepts each of them. The minima table is
// loaded from the class documents, so a class lost or renamed during a catalog
// regeneration would break this endpoint rather than a catalog test alone.
func TestEveryGeneratedClassDrivesSetCharacterStartingClass(t *testing.T) {
	// 20 is at or above every confirmed class minimum, so no class collides and
	// each switch is a pure class change.
	attributes := CharacterAttributes{
		Vigor: 20, Mind: 20, Endurance: 20, Strength: 20,
		Dexterity: 20, Intelligence: 20, Faith: 20, Arcane: 20,
	}
	engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
		platform:       PlatformPC,
		active:         true,
		withAnchor:     true,
		anchorAt:       setStartingClassTestAnchorAt,
		classID:        byte(setStartingClassTestVagabond),
		summaryClassID: byte(setStartingClassTestVagabond),
		attributes:     attributes,
		level:          81,
		runes:          5000,
		soulMemory:     2_000_000,
	})

	revision := "0"
	for classID := uint8(0); classID < 10; classID++ {
		minima, err := startingClassMinima(classID)
		if err != nil {
			t.Fatalf("startingClassMinima(%d): %v", classID, err)
		}
		for index, minimum := range minima {
			if minimum == 0 || minimum > 99 {
				t.Fatalf("class %d attribute %d minimum = %d", classID, index, minimum)
			}
		}

		result, err := engine.SetCharacterStartingClass(
			sessionID, setStartingClassTestSlot, classID, revision)
		if err != nil {
			t.Fatalf("SetCharacterStartingClass(%d): %v", classID, err)
		}
		if result.StartingClassID != classID {
			t.Fatalf("class %d result = %+v", classID, result)
		}
		if result.Attributes != attributes || result.AttributesRaised {
			t.Fatalf("class %d changed attributes: %+v", classID, result)
		}
		revision = result.SaveRevision
	}

	if _, err := startingClassMinima(10); err == nil {
		t.Fatal("startingClassMinima(10): expected an error for an unknown class")
	}
}
