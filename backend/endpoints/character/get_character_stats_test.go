package character

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Synthetic PC container layout used only by this test. The endpoint owns none
// of these values; they are duplicated here so the fixture is accepted by
// SaveEngine without sharing anything with another test file.
const (
	getCharacterStatsHeaderSize       = 0x300
	getCharacterStatsEntryCountOffset = 0x0C
	getCharacterStatsEntryCount       = 12
	getCharacterStatsSlotBlockSize    = 0x280010
	getCharacterStatsFixtureSize      = int64(getCharacterStatsHeaderSize) + 10*getCharacterStatsSlotBlockSize + 0x60010

	getCharacterStatsUserData10Offset = int64(getCharacterStatsHeaderSize) + 10*getCharacterStatsSlotBlockSize + 0x10
	getCharacterStatsFlagsOffset      = 0x1954

	getCharacterStatsSlot     = 3
	getCharacterStatsAnchorAt = 0x0640
)

// getCharacterStatsAnchor is the 65-byte statistics anchor, restated here
// independently of the implementation: one leading 0x00 byte, then four full
// repetitions of a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed by twelve
// 0x00 bytes.
var getCharacterStatsAnchor = []byte{
	0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// writeGetCharacterStatsFixture writes a minimal synthetic PC save into
// t.TempDir() with one active character and returns its path.
func writeGetCharacterStatsFixture(t *testing.T, values GetCharacterStatsResult) string {
	t.Helper()

	data := make([]byte, getCharacterStatsFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[getCharacterStatsEntryCountOffset:], getCharacterStatsEntryCount)

	data[getCharacterStatsUserData10Offset+getCharacterStatsFlagsOffset+getCharacterStatsSlot] = 1

	anchor := int64(getCharacterStatsHeaderSize) + 0x10 +
		getCharacterStatsSlot*getCharacterStatsSlotBlockSize + getCharacterStatsAnchorAt
	copy(data[anchor:], getCharacterStatsAnchor)
	for offset, value := range map[int64]uint32{
		-423: values.HP,
		-419: values.MaxHP,
		-415: values.BaseMaxHP,
		-411: values.FP,
		-407: values.MaxFP,
		-403: values.BaseMaxFP,
		-395: values.SP,
		-391: values.MaxSP,
		-387: values.BaseMaxSP,
		-379: values.Vigor,
		-375: values.Mind,
		-371: values.Endurance,
		-367: values.Strength,
		-363: values.Dexterity,
		-359: values.Intelligence,
		-355: values.Faith,
		-351: values.Arcane,
		-335: values.Level,
	} {
		binary.LittleEndian.PutUint32(data[anchor+offset:], value)
	}

	path := filepath.Join(t.TempDir(), "get-character-stats.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGetCharacterStatsReturnsTheActiveStatsOfALoadedSession(t *testing.T) {
	values := GetCharacterStatsResult{
		HP: 1320, MaxHP: 1320, BaseMaxHP: 1300,
		FP: 180, MaxFP: 200, BaseMaxFP: 195,
		SP: 121, MaxSP: 121, BaseMaxSP: 120,
		Vigor: 38, Mind: 19, Endurance: 22, Strength: 45,
		Dexterity: 17, Intelligence: 11, Faith: 13, Arcane: 8,
		Level: 94,
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetCharacterStatsFixture(t, values), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := GetCharacterStats(engine, loaded.SaveSessionID, getCharacterStatsSlot)
	if err != nil {
		t.Fatalf("GetCharacterStats: %v", err)
	}

	want := values
	want.SaveSessionID = loaded.SaveSessionID
	want.SaveRevision = "0"
	want.CharacterID = getCharacterStatsSlot
	want.Active = true
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetCharacterStatsRejectsMissingEngine(t *testing.T) {
	result, err := GetCharacterStats(nil, "any-session", 0)
	if err == nil {
		t.Fatal("GetCharacterStats accepted a nil engine")
	}
	if err.Error() != "save engine is not available" {
		t.Errorf("error = %q, want %q", err, "save engine is not available")
	}
	if !reflect.DeepEqual(result, GetCharacterStatsResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
