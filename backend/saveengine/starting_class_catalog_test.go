package saveengine

import (
	"testing"
)

// TestEveryGeneratedClassDrivesSetCharacterStartingClass proves the regenerated
// GameCatalog still resolves a complete definition — eight base attributes and a
// base level — for all ten starting classes, and that SetCharacterStartingClass
// applies each of them verbatim. The definition is loaded from the class
// documents, so a class lost, renamed or stripped of its level during a catalog
// regeneration breaks this endpoint rather than a catalog test alone.
func TestEveryGeneratedClassDrivesSetCharacterStartingClass(t *testing.T) {
	// A developed build that sits above every confirmed class minimum, so each
	// switch has to lower the attributes rather than leave them alone.
	attributes := CharacterAttributes{
		Vigor: 20, Mind: 20, Endurance: 20, Strength: 20,
		Dexterity: 20, Intelligence: 20, Faith: 20, Arcane: 20,
	}
	const soulMemory = uint32(2_000_000)
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
		soulMemory:     soulMemory,
	})

	revision := "0"
	for classID := uint8(0); classID < 10; classID++ {
		definition, err := startingClass(classID)
		if err != nil {
			t.Fatalf("startingClass(%d): %v", classID, err)
		}
		for index, minimum := range definition.attributes {
			if minimum == 0 || minimum > 99 {
				t.Fatalf("class %d attribute %d minimum = %d", classID, index, minimum)
			}
		}
		if definition.level == 0 || definition.level > 10 {
			t.Fatalf("class %d base level = %d, want the confirmed 1..10", classID, definition.level)
		}

		want := setStartingClassConfirmedBases[classID]
		if definition.attributes != want.attributes.ordered() || definition.level != want.level {
			t.Fatalf("class %d definition = %+v, want %+v at level %d",
				classID, definition, want.attributes, want.level)
		}

		result, err := engine.SetCharacterStartingClass(
			sessionID, setStartingClassTestSlot, classID, true, revision)
		if err != nil {
			t.Fatalf("SetCharacterStartingClass(%d): %v", classID, err)
		}
		if result.StartingClassID != classID {
			t.Fatalf("class %d result = %+v", classID, result)
		}
		if result.Attributes != want.attributes || result.Level != want.level {
			t.Fatalf("class %d result = %+v, want the class base %+v at level %d",
				classID, result, want.attributes, want.level)
		}
		if result.SoulMemory != soulMemory {
			t.Fatalf("class %d result.SoulMemory = %d, want it preserved at %d",
				classID, result.SoulMemory, soulMemory)
		}
		revision = result.SaveRevision
	}

	if _, err := startingClass(10); err == nil {
		t.Fatal("startingClass(10): expected an error for an unknown class")
	}
}
