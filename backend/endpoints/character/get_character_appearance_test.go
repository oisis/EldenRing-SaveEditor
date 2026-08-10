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
	getCharacterAppearanceHeaderSize       = 0x300
	getCharacterAppearanceEntryCountOffset = 0x0C
	getCharacterAppearanceEntryCount       = 12
	getCharacterAppearanceSlotBlockSize    = 0x280010
	getCharacterAppearanceFixtureSize      = int64(getCharacterAppearanceHeaderSize) + 10*getCharacterAppearanceSlotBlockSize + 0x60010

	getCharacterAppearanceUserData10Offset = int64(getCharacterAppearanceHeaderSize) + 10*getCharacterAppearanceSlotBlockSize + 0x10
	getCharacterAppearanceFlagsOffset      = 0x1954

	getCharacterAppearanceSlot     = 3
	getCharacterAppearanceAnchorAt = 0x0640
	getCharacterAppearanceBlockAt  = 0x3000
)

// getCharacterAppearanceAnchor is the 65-byte statistics anchor the gender and
// voice type are addressed from, restated here independently of the
// implementation: one leading 0x00 byte, then four full repetitions of a 16-byte
// block made of 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes.
var getCharacterAppearanceAnchor = []byte{
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

// writeGetCharacterAppearanceFixture writes a minimal synthetic PC save into
// t.TempDir() with one active character and returns its path.
func writeGetCharacterAppearanceFixture(t *testing.T, values GetCharacterAppearanceResult) string {
	t.Helper()

	data := make([]byte, getCharacterAppearanceFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[getCharacterAppearanceEntryCountOffset:], getCharacterAppearanceEntryCount)

	data[getCharacterAppearanceUserData10Offset+getCharacterAppearanceFlagsOffset+getCharacterAppearanceSlot] = 1

	slotBase := int64(getCharacterAppearanceHeaderSize) + 0x10 +
		getCharacterAppearanceSlot*getCharacterAppearanceSlotBlockSize

	anchor := slotBase + getCharacterAppearanceAnchorAt
	copy(data[anchor:], getCharacterAppearanceAnchor)
	data[anchor-249] = values.Gender
	data[anchor-245] = values.VoiceType

	block := data[slotBase+getCharacterAppearanceBlockAt:]
	copy(block, []byte{0xFF, 0xFF, 0xFF, 0xFF, 'F', 'A', 'C', 'E'})
	binary.LittleEndian.PutUint32(block[0x08:], 4)
	binary.LittleEndian.PutUint32(block[0x0C:], 0x120)
	for index, id := range values.ModelIDs {
		binary.LittleEndian.PutUint32(block[0x10+index*4:], id)
	}
	copy(block[0x30:], values.FaceShape[:])
	copy(block[0xB0:], values.Body[:])
	copy(block[0xB7:], values.Skin[:])

	path := filepath.Join(t.TempDir(), "get-character-appearance.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGetCharacterAppearanceReturnsTheActiveAppearanceOfALoadedSession(t *testing.T) {
	values := GetCharacterAppearanceResult{
		Gender:    1,
		VoiceType: 2,
		ModelIDs:  [8]uint32{7, 0x0121, 4, 0, 1, 0, 0, 3},
	}
	for index := range values.FaceShape {
		values.FaceShape[index] = uint8(index) + 1
	}
	for index := range values.Body {
		values.Body[index] = uint8(index) + 0x41
	}
	for index := range values.Skin {
		values.Skin[index] = uint8(index) + 0x81
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetCharacterAppearanceFixture(t, values), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := GetCharacterAppearance(engine, loaded.SaveSessionID, getCharacterAppearanceSlot)
	if err != nil {
		t.Fatalf("GetCharacterAppearance: %v", err)
	}

	want := values
	want.SaveSessionID = loaded.SaveSessionID
	want.CharacterID = getCharacterAppearanceSlot
	want.Active = true
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetCharacterAppearanceRejectsMissingEngine(t *testing.T) {
	result, err := GetCharacterAppearance(nil, "any-session", 0)
	if err == nil {
		t.Fatal("GetCharacterAppearance accepted a nil engine")
	}
	if err.Error() != "save engine is not available" {
		t.Errorf("error = %q, want %q", err, "save engine is not available")
	}
	if !reflect.DeepEqual(result, GetCharacterAppearanceResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
