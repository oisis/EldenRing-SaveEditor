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
// or chain distance fails here.
const (
	physickPCSlotDataBase  = 0x300 + 0x10 // first PC slot data, behind its MD5 prefix
	physickPCSlotStride    = 0x280010
	physickPS4SlotDataBase = 0x70 // first PS4 slot data, no MD5 prefix
	physickPS4SlotStride   = 0x280000
	physickSlotDataSize    = 0x280000

	// physickCountAt is the distance from the anchor to the projectile count, and
	// physickArmamentsAt is the size of the equipped-armaments block the Physick
	// block starts behind.
	physickCountAt     = 0x931D
	physickArmamentsAt = 0x9C
)

// physickTestAnchor is the 65-byte anchor the Physick chain is measured from,
// restated here independently of the implementation so a changed production
// pattern fails this test: one leading 0x00 byte, then four full repetitions of
// a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes.
var physickTestAnchor = []byte{
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

// physickFixture describes the synthetic slot content one test save is built
// from: which slot carries which activity flag, where its anchor sits relative
// to the start of the slot data, how many acquired-projectile records the slot
// declares, and which two raw Tear identifiers the Physick block holds. A
// residual slot is expressed as a zero flag with everything still written into
// the file.
//
// decoyTears writes a second, equally well-formed pair of identifiers behind the
// armaments block that would follow a zero projectile count — the position the
// real pair would occupy if the declared length were ignored — so a reader that
// skips the dynamic section reads the decoy and fails the assertion.
type physickFixture struct {
	platform        Platform
	slot            int
	flag            byte
	anchorAt        int64
	projectileCount uint32
	tears           [2]uint32
	decoyTears      [2]uint32
	noAnchor        bool
}

// writePhysickFixture builds a synthetic save and returns its path inside
// t.TempDir(). Only the activity flag, the anchor, the projectile count and the
// two Tear pairs are written; the rest of the container stays zeroed. A pair
// that would reach past the end of the slot data is left out, which is how the
// out-of-bounds case is expressed.
func writePhysickFixture(t *testing.T, content physickFixture) string {
	t.Helper()

	var data []byte
	var userData10Base, slotBase int64
	switch content.platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = pcUserData10DataOffset
		slotBase = physickPCSlotDataBase + int64(content.slot)*physickPCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = ps4UserData10DataOffset
		slotBase = physickPS4SlotDataBase + int64(content.slot)*physickPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", content.platform)
	}

	data[userData10Base+userData10ActiveFlagsOffset+int64(content.slot)] = content.flag

	if content.noAnchor {
		path := filepath.Join(t.TempDir(), "physick.sl2")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		return path
	}

	copy(data[slotBase+content.anchorAt:], physickTestAnchor)

	countAt := content.anchorAt + physickCountAt
	binary.LittleEndian.PutUint32(data[slotBase+countAt:], content.projectileCount)

	writeTears := func(at int64, tears [2]uint32) {
		if at+8 > physickSlotDataSize {
			return
		}
		binary.LittleEndian.PutUint32(data[slotBase+at:], tears[0])
		binary.LittleEndian.PutUint32(data[slotBase+at+4:], tears[1])
	}

	writeTears(countAt+4+physickArmamentsAt, content.decoyTears)
	writeTears(countAt+4+int64(content.projectileCount)*8+physickArmamentsAt, content.tears)

	path := filepath.Join(t.TempDir(), "physick.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGetPhysickMixtureReadsTheActiveSlotOfBothPlatforms(t *testing.T) {
	// The two fixtures put the anchor at different positions and declare a
	// different, non-zero number of projectile records, so neither a fixed offset
	// nor a fixed skip can pass both cases. Between them the expected Tears cover
	// 0, 0xFFFFFFFF and values whose high bit is set, so a reader that masks,
	// normalises or drops type bits cannot pass either.
	cases := []physickFixture{
		{
			platform:        PlatformPC,
			slot:            0,
			flag:            1,
			anchorAt:        0x01A7,
			projectileCount: 11,
			tears:           [2]uint32{0, 0x80000A01},
			decoyTears:      [2]uint32{0x40002AF9, 0x40002AFA},
		},
		{
			platform:        PlatformPS4,
			slot:            7,
			flag:            1,
			anchorAt:        0x1F4C2,
			projectileCount: 37,
			tears:           [2]uint32{0xFFFFFFFF, 0x90000B02},
			decoyTears:      [2]uint32{0x40002B0A, 0x40002B0B},
		},
	}

	for _, testCase := range cases {
		t.Run(string(testCase.platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writePhysickFixture(t, testCase), string(testCase.platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.GetPhysickMixture(loaded.SaveSessionID, testCase.slot)
			if err != nil {
				t.Fatalf("GetPhysickMixture: %v", err)
			}

			want := CharacterPhysickMixture{
				SaveSessionID: loaded.SaveSessionID,
				CharacterID:   testCase.slot,
				Active:        true,
				Tears:         testCase.tears,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}
		})
	}
}

func TestGetPhysickMixtureReportsAResidualSlotAsInactive(t *testing.T) {
	content := physickFixture{
		platform:        PlatformPC,
		slot:            4,
		flag:            0,
		anchorAt:        0x0800,
		projectileCount: 5,
		tears:           [2]uint32{0x40002AF9, 0x40002B0C},
		decoyTears:      [2]uint32{0x40002B0D, 0x40002B0E},
	}

	engine := New()
	loaded, err := engine.LoadSave(writePhysickFixture(t, content), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.GetPhysickMixture(loaded.SaveSessionID, content.slot)
	if err != nil {
		t.Fatalf("GetPhysickMixture: %v", err)
	}

	want := CharacterPhysickMixture{SaveSessionID: loaded.SaveSessionID, CharacterID: content.slot}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetPhysickMixtureRejectsInvalidRequests(t *testing.T) {
	engine := New()

	loadSlot := func(content physickFixture) string {
		t.Helper()
		loaded, err := engine.LoadSave(writePhysickFixture(t, content), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		return loaded.SaveSessionID
	}

	present := loadSlot(physickFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x0640,
		projectileCount: 11, tears: [2]uint32{0x40002AF9, 0}, decoyTears: [2]uint32{0x40002B0F, 1},
	})
	missingAnchor := loadSlot(physickFixture{
		platform: PlatformPC, slot: 2, flag: 1, noAnchor: true,
	})
	invalidCount := loadSlot(physickFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x0640,
		projectileCount: 200001, tears: [2]uint32{0x40002AF9, 0},
	})
	// The anchor sits so close to the end of the slot that the projectile count
	// still fits but the two Tears behind the armaments block do not.
	truncated := loadSlot(physickFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: physickSlotDataSize - 0x93C0,
		projectileCount: 0, tears: [2]uint32{0x40002AF9, 0},
	})

	cases := map[string]struct {
		saveSessionID string
		characterID   int
		want          string
	}{
		"empty session":   {"", 0, "saveSessionID is required"},
		"unknown session": {"missing", 0, `unknown save session "missing"`},
		"characterID -1":  {present, -1, "characterID -1 is outside the range 0..9"},
		"characterID 10":  {present, 10, "characterID 10 is outside the range 0..9"},
		"missing anchor":  {missingAnchor, 2, "character 2 carries no physick anchor"},
		"invalid projectile count": {invalidCount, 2,
			"character 2 declares 200001 projectile records, want at most 200000"},
		"truncated block": {truncated, 2, "physick mixture of character 2 does not fit into its slot"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetPhysickMixture(testCase.saveSessionID, testCase.characterID)
			if err == nil {
				t.Fatalf("GetPhysickMixture accepted %s", strconv.Quote(name))
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, CharacterPhysickMixture{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}
