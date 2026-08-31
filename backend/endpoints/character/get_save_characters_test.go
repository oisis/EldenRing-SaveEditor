package character

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"unicode/utf16"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Synthetic PC container layout used only by this test. The endpoint owns none
// of these values; they are duplicated here so the fixture is accepted by
// SaveEngine without sharing anything with another test file.
const (
	getSaveCharactersHeaderSize       = 0x300
	getSaveCharactersEntryCountOffset = 0x0C
	getSaveCharactersEntryCount       = 12
	getSaveCharactersFixtureSize      = int64(getSaveCharactersHeaderSize) + 10*0x280010 + 0x60010

	getSaveCharactersUserData10Offset = int64(getSaveCharactersHeaderSize) + 10*0x280010 + 0x10
	getSaveCharactersFlagsOffset      = 0x1954
	getSaveCharactersSummaryOffset    = 0x195E
	getSaveCharactersSummaryStride    = 0x24C
	getSaveCharactersLevelOffset      = 0x22
)

// writeGetSaveCharactersFixture writes a minimal synthetic PC save into
// t.TempDir() with one active character in slot 3 and returns its path.
func writeGetSaveCharactersFixture(t *testing.T) string {
	t.Helper()

	data := make([]byte, getSaveCharactersFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[getSaveCharactersEntryCountOffset:], getSaveCharactersEntryCount)

	data[getSaveCharactersUserData10Offset+getSaveCharactersFlagsOffset+3] = 1
	summary := getSaveCharactersUserData10Offset + getSaveCharactersSummaryOffset + 3*getSaveCharactersSummaryStride
	for index, unit := range utf16.Encode([]rune("Ranni")) {
		binary.LittleEndian.PutUint16(data[summary+int64(index)*2:], unit)
	}
	binary.LittleEndian.PutUint32(data[summary+getSaveCharactersLevelOffset:], 120)

	path := filepath.Join(t.TempDir(), "get-save-characters.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGetSaveCharactersReturnsTheSlotsOfALoadedSession(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetSaveCharactersFixture(t), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := GetSaveCharacters(engine, loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSaveCharacters: %v", err)
	}

	// The endpoint delegates, so its result must equal what SaveEngine returns.
	want, err := engine.GetSaveCharacters(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("engine.GetSaveCharacters: %v", err)
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result =\n%+v\nwant\n%+v", result, want)
	}

	if len(result.Characters) != 10 {
		t.Fatalf("characters = %d, want 10", len(result.Characters))
	}
	for slot, character := range result.Characters {
		if character.CharacterID != slot {
			t.Errorf("characters[%d].characterID = %d", slot, character.CharacterID)
		}
	}
	active := result.Characters[3]
	if !active.Active || active.Name != "Ranni" || active.Level != 120 {
		t.Errorf("characters[3] = %+v, want the active Ranni at level 120", active)
	}
	if result.Characters[0].Active || result.Characters[0].Name != "" || result.Characters[0].Level != 0 {
		t.Errorf("characters[0] = %+v, want an empty inactive slot", result.Characters[0])
	}
}

func TestGetSaveCharactersRejectsMissingEngine(t *testing.T) {
	result, err := GetSaveCharacters(nil, "any-session")
	if err == nil {
		t.Fatal("GetSaveCharacters accepted a nil engine")
	}
	if err.Error() != "save engine is not available" {
		t.Errorf("error = %q, want %q", err, "save engine is not available")
	}
	if !reflect.DeepEqual(result, GetSaveCharactersResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
