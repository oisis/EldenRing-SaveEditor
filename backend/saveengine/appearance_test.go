package saveengine

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

// Synthetic container layout used only by this test. The offsets are restated
// literally instead of reused from the implementation, so a changed base, stride
// or field offset fails here.
const (
	appearancePCSlotDataBase  = 0x300 + 0x10 // first PC slot data, behind its MD5 prefix
	appearancePCSlotStride    = 0x280010
	appearancePS4SlotDataBase = 0x70 // first PS4 slot data, no MD5 prefix
	appearancePS4SlotStride   = 0x280000
	appearanceSlotDataSize    = 0x280000

	appearanceBlockSize = 0x12F
	appearanceGenderAt  = -249
	appearanceVoiceAt   = -245
)

// appearanceTestAnchor is the 65-byte player-data anchor, restated here
// independently of the implementation so a changed production pattern fails
// this test: one leading 0x00 byte, then four full repetitions of a 16-byte
// block made of 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes.
var appearanceTestAnchor = []byte{
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

// appearanceFixture describes the synthetic slot content one test save is built
// from: which slot carries which activity flag, where its player-data anchor and
// its appearance block sit relative to the start of the slot data, and which raw
// values are written into them. A residual slot is expressed as a zero flag with
// everything still written into the file. laterBlockAt writes a second, equally
// well-formed appearance block behind the first one, carrying laterValues.
type appearanceFixture struct {
	platform     Platform
	slot         int
	flag         byte
	anchorAt     int64
	blockAt      int64
	values       CharacterAppearance
	laterBlockAt int64
	laterValues  CharacterAppearance
	noBlock      bool
	alignment    uint32
	innerSize    uint32
}

// writeAppearanceFixture builds a synthetic save and returns its path inside
// t.TempDir(). Only the activity flag, the player-data anchor, the gender and
// voice type in front of it and the appearance blocks are written; the rest of
// the container stays zeroed.
func writeAppearanceFixture(t *testing.T, content appearanceFixture) string {
	t.Helper()

	var data []byte
	var userData10Base, slotBase int64
	switch content.platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = pcUserData10DataOffset
		slotBase = appearancePCSlotDataBase + int64(content.slot)*appearancePCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = ps4UserData10DataOffset
		slotBase = appearancePS4SlotDataBase + int64(content.slot)*appearancePS4SlotStride
	default:
		t.Fatalf("unknown platform %q", content.platform)
	}

	data[userData10Base+userData10ActiveFlagsOffset+int64(content.slot)] = content.flag

	anchor := slotBase + content.anchorAt
	copy(data[anchor:], appearanceTestAnchor)
	data[anchor+appearanceGenderAt] = content.values.Gender
	data[anchor+appearanceVoiceAt] = content.values.VoiceType

	writeBlock := func(at int64, values CharacterAppearance, alignment, innerSize uint32) {
		block := make([]byte, appearanceBlockSize)
		copy(block, []byte{0xFF, 0xFF, 0xFF, 0xFF, 'F', 'A', 'C', 'E'})
		binary.LittleEndian.PutUint32(block[0x08:], alignment)
		binary.LittleEndian.PutUint32(block[0x0C:], innerSize)
		for index, id := range values.ModelIDs {
			binary.LittleEndian.PutUint32(block[0x10+index*4:], id)
		}
		copy(block[0x30:], values.FaceShape[:])
		copy(block[0xB0:], values.Body[:])
		copy(block[0xB7:], values.Skin[:])
		copy(data[slotBase+at:], block)
	}

	if !content.noBlock {
		alignment, innerSize := content.alignment, content.innerSize
		if alignment == 0 && innerSize == 0 {
			alignment, innerSize = 4, 0x120
		}
		writeBlock(content.blockAt, content.values, alignment, innerSize)
	}
	if content.laterBlockAt != 0 {
		writeBlock(content.laterBlockAt, content.laterValues, 4, 0x120)
	}

	path := filepath.Join(t.TempDir(), "appearance.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// appearanceValues fills every parameter block completely, so a decoder that
// stops short of 64, 7 or 91 bytes cannot pass. The model IDs include a value
// far above 255, so a decoder narrowing them to a byte cannot pass either.
func appearanceValues(seed uint8) CharacterAppearance {
	values := CharacterAppearance{
		Gender:    1,
		VoiceType: 3,
		ModelIDs:  [8]uint32{0, 9, 0x0134 + uint32(seed), 3, 1, 0xFFFF0001, 0, 2},
	}
	for index := range values.FaceShape {
		values.FaceShape[index] = seed + uint8(index)
	}
	for index := range values.Body {
		values.Body[index] = seed + 0x40 + uint8(index)
	}
	for index := range values.Skin {
		values.Skin[index] = seed + 0x80 + uint8(index)
	}
	return values
}

func TestGetCharacterAppearanceReadsTheActiveSlotOfBothPlatforms(t *testing.T) {
	// The two fixtures put the anchor and the appearance block at different
	// positions inside the slot, so a fixed offset instead of a search cannot
	// pass both cases.
	cases := []appearanceFixture{
		{
			// The PC slot also carries a second, equally well-formed appearance
			// block behind the first one, as a healthy slot does. Its values
			// differ in every field, so the assertion below fails if anything but
			// the first block is decoded.
			platform:     PlatformPC,
			slot:         0,
			flag:         1,
			anchorAt:     0x01A7,
			blockAt:      0x2000,
			values:       appearanceValues(0x11),
			laterBlockAt: 0x40000,
			laterValues:  appearanceValues(0x9A),
		},
		{
			platform: PlatformPS4,
			slot:     7,
			flag:     1,
			anchorAt: 0x1F4C2,
			blockAt:  0x0800,
			values:   appearanceValues(0x53),
		},
	}

	for _, testCase := range cases {
		t.Run(string(testCase.platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeAppearanceFixture(t, testCase), string(testCase.platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.GetCharacterAppearance(loaded.SaveSessionID, testCase.slot)
			if err != nil {
				t.Fatalf("GetCharacterAppearance: %v", err)
			}

			want := testCase.values
			want.SaveSessionID = loaded.SaveSessionID
			want.CharacterID = testCase.slot
			want.Active = true
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}
		})
	}
}

func TestGetCharacterAppearanceReportsAResidualSlotAsInactive(t *testing.T) {
	content := appearanceFixture{
		platform: PlatformPC,
		slot:     4,
		flag:     0,
		anchorAt: 0x0800,
		blockAt:  0x4000,
		values:   appearanceValues(0x27),
	}

	engine := New()
	loaded, err := engine.LoadSave(writeAppearanceFixture(t, content), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.GetCharacterAppearance(loaded.SaveSessionID, content.slot)
	if err != nil {
		t.Fatalf("GetCharacterAppearance: %v", err)
	}

	want := CharacterAppearance{SaveSessionID: loaded.SaveSessionID, CharacterID: content.slot}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetCharacterAppearanceRejectsInvalidRequests(t *testing.T) {
	engine := New()

	loadSlot := func(content appearanceFixture) string {
		t.Helper()
		loaded, err := engine.LoadSave(writeAppearanceFixture(t, content), "", "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		return loaded.SaveSessionID
	}

	present := loadSlot(appearanceFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x0640, blockAt: 0x1000,
		values: appearanceValues(0x05),
	})
	missing := loadSlot(appearanceFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x0640, noBlock: true,
	})
	wrongInnerSize := loadSlot(appearanceFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x0640, blockAt: 0x1000,
		values: appearanceValues(0x05), alignment: 4, innerSize: 0x110,
	})
	truncated := loadSlot(appearanceFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x0640,
		blockAt: appearanceSlotDataSize - 8, values: appearanceValues(0x05),
	})

	cases := map[string]struct {
		saveSessionID string
		characterID   int
		want          string
	}{
		"empty session":    {"", 0, "saveSessionID is required"},
		"unknown session":  {"missing", 0, `unknown save session "missing"`},
		"characterID -1":   {present, -1, "characterID -1 is outside the range 0..9"},
		"characterID 10":   {present, 10, "characterID 10 is outside the range 0..9"},
		"missing block":    {missing, 2, "character 2 carries no appearance block"},
		"wrong inner size": {wrongInnerSize, 2, "appearance block of character 2 declares alignment 4 and inner size 0x110, want 4 and 0x120"},
		"truncated block":  {truncated, 2, "appearance block of character 2 does not fit into its slot"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetCharacterAppearance(testCase.saveSessionID, testCase.characterID)
			if err == nil {
				t.Fatalf("GetCharacterAppearance accepted %s", strconv.Quote(name))
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, CharacterAppearance{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}
