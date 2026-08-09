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
	getCharacterProfileHeaderSize       = 0x300
	getCharacterProfileEntryCountOffset = 0x0C
	getCharacterProfileEntryCount       = 12
	getCharacterProfileFixtureSize      = int64(getCharacterProfileHeaderSize) + 10*0x280010 + 0x60010

	getCharacterProfileUserData10Offset = int64(getCharacterProfileHeaderSize) + 10*0x280010 + 0x10
	getCharacterProfileFlagsOffset      = 0x1954
	getCharacterProfileSummaryOffset    = 0x195E
	getCharacterProfileSummaryStride    = 0x24C
	getCharacterProfileLevelOffset      = 0x22
	getCharacterProfileSecondsOffset    = 0x26
	getCharacterProfileGenderOffset     = 0x242
	getCharacterProfileClassOffset      = 0x243

	getCharacterProfileSlot = 6
)

// writeGetCharacterProfileFixture writes a minimal synthetic PC save into
// t.TempDir() with one active character and returns its path.
func writeGetCharacterProfileFixture(t *testing.T) string {
	t.Helper()

	data := make([]byte, getCharacterProfileFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[getCharacterProfileEntryCountOffset:], getCharacterProfileEntryCount)

	data[getCharacterProfileUserData10Offset+getCharacterProfileFlagsOffset+getCharacterProfileSlot] = 1
	summary := getCharacterProfileUserData10Offset +
		getCharacterProfileSummaryOffset +
		getCharacterProfileSlot*getCharacterProfileSummaryStride
	for index, unit := range utf16.Encode([]rune("Blaidd")) {
		binary.LittleEndian.PutUint16(data[summary+int64(index)*2:], unit)
	}
	binary.LittleEndian.PutUint32(data[summary+getCharacterProfileLevelOffset:], 138)
	binary.LittleEndian.PutUint32(data[summary+getCharacterProfileSecondsOffset:], 7_384)
	data[summary+getCharacterProfileGenderOffset] = 1
	data[summary+getCharacterProfileClassOffset] = 2

	path := filepath.Join(t.TempDir(), "get-character-profile.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGetCharacterProfileReturnsTheActiveProfileOfALoadedSession(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetCharacterProfileFixture(t), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := GetCharacterProfile(engine, loaded.SaveSessionID, getCharacterProfileSlot)
	if err != nil {
		t.Fatalf("GetCharacterProfile: %v", err)
	}

	want := GetCharacterProfileResult{
		SaveSessionID:   loaded.SaveSessionID,
		CharacterID:     getCharacterProfileSlot,
		Active:          true,
		Name:            "Blaidd",
		Level:           138,
		StartingClassID: 2,
		Gender:          1,
		SecondsPlayed:   7_384,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetCharacterProfileRejectsMissingEngine(t *testing.T) {
	result, err := GetCharacterProfile(nil, "any-session", 0)
	if err == nil {
		t.Fatal("GetCharacterProfile accepted a nil engine")
	}
	if err.Error() != "save engine is not available" {
		t.Errorf("error = %q, want %q", err, "save engine is not available")
	}
	if !reflect.DeepEqual(result, GetCharacterProfileResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
