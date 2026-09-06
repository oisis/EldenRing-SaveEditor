package saveengine

import (
	"encoding/binary"
	"os"
	"reflect"
	"testing"
)

func TestGetSaveCharactersReportsEachDeclaredSlotVersion(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := charactersFixture{}
			content.flags[0], content.names[0] = 1, "First"
			content.flags[1], content.names[1] = 1, "Second"
			content.flags[3] = 2 // Unknown classification still has a declaration.
			path := writeCharactersFixture(t, platform, content)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			versions := []uint32{0x4C, 0xE6, 0, 0x1234}
			for slot, version := range versions {
				binary.LittleEndian.PutUint32(data[slotDataBase(platform, slot):], version)
			}
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			engine := New()
			loaded, err := engine.LoadSave(path, string(platform), "local")
			if err != nil {
				t.Fatal(err)
			}
			result, err := engine.GetSaveCharacters(loaded.SaveSessionID)
			if err != nil {
				t.Fatal(err)
			}
			for slot, version := range versions {
				got := result.Slots[slot]
				if got.SlotVersion != version || got.SlotVersionKnown != (version != 0) {
					t.Errorf("slot %d version = %d, known = %v; want %d, %v", slot,
						got.SlotVersion, got.SlotVersionKnown, version, version != 0)
				}
			}
			if result.SaveRevision != "0" {
				t.Errorf("read changed revision: %s", result.SaveRevision)
			}
		})
	}
}

// The slot-management projection must classify the four states from the same
// evidence the writers evaluate, and it must never offer a capability the
// matching writer would refuse.
func TestGetSaveCharactersProjectsSlotStatesAndCapabilities(t *testing.T) {
	content := charactersFixture{}
	// Slot 0: an ordinary active character with a starting class other than 0.
	content.flags[0], content.names[0], content.level[0], content.class[0] = 1, "Tarnished", 150, 5
	// Slot 2: a residual slot — the flag is cleared while the deleted
	// character's summary values are still in the file.
	content.flags[2], content.names[2], content.level[2], content.class[2] = 0, "Deleted", 99, 3
	// Slot 9: an activity flag that is neither 0 nor 1.
	content.flags[9], content.names[9] = 2, "Ghost"

	want := make([]CharacterSlot, characterSlotCount)
	for slot := range want {
		want[slot] = CharacterSlot{
			CharacterID:  slot,
			State:        CharacterSlotStateEmpty,
			Capabilities: CharacterSlotCapabilities{CloneInto: true},
		}
	}
	want[0] = CharacterSlot{
		CharacterID:        0,
		State:              CharacterSlotStateActive,
		StartingClassID:    5,
		StartingClassKnown: true,
		Capabilities: CharacterSlotCapabilities{
			Deactivate: true, CloneFrom: true, Delete: true,
		},
	}
	// Residual data can be cleared, but this fixture carries no statistics
	// anchor, so SetCharacterActive would refuse the reactivation and the
	// capability must say so. The residual starting class stays undisclosed.
	want[2] = CharacterSlot{
		CharacterID:  2,
		State:        CharacterSlotStateResidual,
		Capabilities: CharacterSlotCapabilities{Delete: true},
	}
	// An unsupported flag is the fail-safe case: no state, no operation.
	want[9] = CharacterSlot{CharacterID: 9, State: CharacterSlotStateUnknown}

	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine := New()
			info, err := engine.LoadSave(
				writeCharactersFixture(t, platform, content), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.GetSaveCharacters(info.SaveSessionID)
			if err != nil {
				t.Fatalf("GetSaveCharacters: %v", err)
			}
			if !reflect.DeepEqual(result.Slots, want) {
				t.Errorf("slots =\n%+v\nwant\n%+v", result.Slots, want)
			}
		})
	}
}

// The residual slot of an unsupported target must never be offered as a clone
// target, and an empty slot must never be offered as a deletion target: those
// are exactly the two rejections CloneCharacter and DeleteCharacter enforce.
func TestCharacterSlotCapabilitiesMatchTheWriterRejections(t *testing.T) {
	content := charactersFixture{}
	content.flags[0], content.names[0], content.level[0] = 1, "Tarnished", 150
	content.flags[2], content.names[2] = 0, "Deleted"

	engine := New()
	info, err := engine.LoadSave(
		writeCharactersFixture(t, PlatformPC, content), string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	result, err := engine.GetSaveCharacters(info.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSaveCharacters: %v", err)
	}

	residual := result.Slots[2]
	if residual.Capabilities.CloneInto {
		t.Error("a residual slot must not be offered as a clone target")
	}
	if _, err := engine.CloneCharacter(info.SaveSessionID, 0, 2, result.SaveRevision); err == nil {
		t.Error("CloneCharacter accepted a residual target slot")
	}

	empty := result.Slots[6]
	if empty.Capabilities.Delete {
		t.Error("an empty slot must not be offered as a deletion target")
	}
	if _, err := engine.DeleteCharacter(info.SaveSessionID, 6, result.SaveRevision); err == nil {
		t.Error("DeleteCharacter accepted an empty slot")
	}
}
