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
	equipmentPCSlotDataBase  = 0x300 + 0x10 // first PC slot data, behind its MD5 prefix
	equipmentPCSlotStride    = 0x280010
	equipmentPS4SlotDataBase = 0x70 // first PS4 slot data, no MD5 prefix
	equipmentPS4SlotStride   = 0x280000
	equipmentSlotDataSize    = 0x280000

	// equipmentCountAt is the distance from the anchor to the projectile count.
	equipmentCountAt   = 0x931D
	equipmentTestSlots = 22
)

// equipmentTestAnchor is the 65-byte anchor the equipment chain is measured
// from, restated here independently of the implementation so a changed
// production pattern fails this test: one leading 0x00 byte, then four full
// repetitions of a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed by twelve
// 0x00 bytes.
var equipmentTestAnchor = []byte{
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

// equipmentFixture describes the synthetic slot content one test save is built
// from: which slot carries which activity flag, where its anchor sits relative
// to the start of the slot data, how many acquired-projectile records the slot
// declares, and which raw values the equipment block holds. A residual slot is
// expressed as a zero flag with everything still written into the file.
//
// decoyValues writes a second, equally well-formed 22-slot block immediately
// behind the projectile count — the position the real block would occupy if the
// declared count were zero — so a reader that ignores the dynamic length reads
// the decoy and fails the assertion.
type equipmentFixture struct {
	platform        Platform
	slot            int
	flag            byte
	anchorAt        int64
	projectileCount uint32
	values          [equipmentTestSlots]uint32
	decoyValues     [equipmentTestSlots]uint32
	noAnchor        bool
}

// writeEquipmentFixture builds a synthetic save and returns its path inside
// t.TempDir(). Only the activity flag, the anchor, the projectile count and the
// two equipment blocks are written; the rest of the container stays zeroed. A
// block that would reach past the end of the slot data is left out, which is how
// the out-of-bounds cases are expressed.
func writeEquipmentFixture(t *testing.T, content equipmentFixture) string {
	t.Helper()

	var data []byte
	var userData10Base, slotBase int64
	switch content.platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = pcUserData10DataOffset
		slotBase = equipmentPCSlotDataBase + int64(content.slot)*equipmentPCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = ps4UserData10DataOffset
		slotBase = equipmentPS4SlotDataBase + int64(content.slot)*equipmentPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", content.platform)
	}

	data[userData10Base+userData10ActiveFlagsOffset+int64(content.slot)] = content.flag

	if content.noAnchor {
		path := filepath.Join(t.TempDir(), "equipment.sl2")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		return path
	}

	copy(data[slotBase+content.anchorAt:], equipmentTestAnchor)

	countAt := content.anchorAt + equipmentCountAt
	binary.LittleEndian.PutUint32(data[slotBase+countAt:], content.projectileCount)

	writeBlock := func(at int64, values [equipmentTestSlots]uint32) {
		if at+equipmentTestSlots*4 > equipmentSlotDataSize {
			return
		}
		for index, value := range values {
			binary.LittleEndian.PutUint32(data[slotBase+at+int64(index)*4:], value)
		}
	}

	writeBlock(countAt+4, content.decoyValues)
	writeBlock(countAt+4+int64(content.projectileCount)*8, content.values)

	path := filepath.Join(t.TempDir(), "equipment.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// equipmentValues fills all 22 fields with distinct values and deliberately
// includes 0, 0xFFFFFFFF and values whose high bit is set, so a reader that
// masks, normalises or drops type bits cannot pass.
func equipmentValues(seed uint32) [equipmentTestSlots]uint32 {
	var values [equipmentTestSlots]uint32
	for index := range values {
		values[index] = 0x00110000*uint32(index+1) + seed
	}
	values[1] = 0x80000000 | (0x0A01 + seed)
	values[6] = 0xFFFFFFFF
	values[10] = 0
	values[13] = 0x90000000 | (0x0B02 + seed)
	values[21] = 0x8FFFFFFE - seed
	return values
}

func TestGetEquipmentReadsTheActiveSlotOfBothPlatforms(t *testing.T) {
	// The two fixtures put the anchor at different positions and declare a
	// different, non-zero number of projectile records, so neither a fixed offset
	// nor a fixed skip can pass both cases. The PC count of 11 records makes the
	// decoy block fill the projectile section exactly; the PS4 count of 37 leaves
	// zeroed records between the decoy and the real block.
	cases := []equipmentFixture{
		{
			platform:        PlatformPC,
			slot:            0,
			flag:            1,
			anchorAt:        0x01A7,
			projectileCount: 11,
			values:          equipmentValues(0x11),
			decoyValues:     equipmentValues(0x53),
		},
		{
			platform:        PlatformPS4,
			slot:            7,
			flag:            1,
			anchorAt:        0x1F4C2,
			projectileCount: 37,
			values:          equipmentValues(0x27),
			decoyValues:     equipmentValues(0x64),
		},
	}

	for _, testCase := range cases {
		t.Run(string(testCase.platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeEquipmentFixture(t, testCase), string(testCase.platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.GetEquipment(loaded.SaveSessionID, testCase.slot)
			if err != nil {
				t.Fatalf("GetEquipment: %v", err)
			}

			want := CharacterEquipment{
				SaveSessionID: loaded.SaveSessionID,
				SaveRevision:  "0",
				CharacterID:   testCase.slot,
				Active:        true,
				Slots:         testCase.values,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}
		})
	}
}

func TestGetEquipmentReportsAResidualSlotAsInactive(t *testing.T) {
	content := equipmentFixture{
		platform:        PlatformPC,
		slot:            4,
		flag:            0,
		anchorAt:        0x0800,
		projectileCount: 5,
		values:          equipmentValues(0x31),
		decoyValues:     equipmentValues(0x72),
	}

	engine := New()
	loaded, err := engine.LoadSave(writeEquipmentFixture(t, content), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.GetEquipment(loaded.SaveSessionID, content.slot)
	if err != nil {
		t.Fatalf("GetEquipment: %v", err)
	}

	want := CharacterEquipment{SaveSessionID: loaded.SaveSessionID, SaveRevision: "0", CharacterID: content.slot}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetEquipmentRejectsInvalidRequests(t *testing.T) {
	engine := New()

	loadSlot := func(content equipmentFixture) string {
		t.Helper()
		loaded, err := engine.LoadSave(writeEquipmentFixture(t, content), "", "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		return loaded.SaveSessionID
	}

	present := loadSlot(equipmentFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x0640,
		projectileCount: 11, values: equipmentValues(0x05), decoyValues: equipmentValues(0x44),
	})
	missingAnchor := loadSlot(equipmentFixture{
		platform: PlatformPC, slot: 2, flag: 1, noAnchor: true,
	})
	invalidCount := loadSlot(equipmentFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x0640,
		projectileCount: 200001, values: equipmentValues(0x05),
	})
	// The anchor sits so close to the end of the slot that the projectile count
	// still fits but the 22-field block behind it does not.
	truncated := loadSlot(equipmentFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: equipmentSlotDataSize - 0x9349,
		projectileCount: 0, values: equipmentValues(0x05),
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
		"missing anchor":  {missingAnchor, 2, "character 2 carries no equipment anchor"},
		"invalid projectile count": {invalidCount, 2,
			"character 2 declares 200001 projectile records, want at most 200000"},
		"truncated block": {truncated, 2, "equipment block of character 2 does not fit into its slot"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetEquipment(testCase.saveSessionID, testCase.characterID)
			if err == nil {
				t.Fatalf("GetEquipment accepted %s", strconv.Quote(name))
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, CharacterEquipment{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}
