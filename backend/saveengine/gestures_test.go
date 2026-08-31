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
// or section distance fails here.
const (
	gestureTestPCSlotDataBase  = 0x300 + 0x10 // first PC slot data, behind its MD5 prefix
	gestureTestPCSlotStride    = 0x280010
	gestureTestPS4SlotDataBase = 0x70 // first PS4 slot data, no MD5 prefix
	gestureTestPS4SlotStride   = 0x280000
	gestureTestSlotDataSize    = 0x280000

	// Distance from the anchor to the uint32 that declares how many
	// acquired-projectile records follow it, restated as the literal sum of the
	// fixed structures in between: SpEffect, EquipedItemIndex,
	// ActiveEquipedItems, EquipedItemsID, ActiveEquipedItemsGa, InventoryHeld,
	// EquippedSpells, EquipItemData and EquippedGestures.
	gestureTestProjectileCountAt = 0xD0 + 0x58 + 0x1C + 0x58 + 0x58 + 0x9011 + 0x74 + 0x8C + 0x18
	gestureTestProjectileStride  = 8

	// Distance from the end of the projectile records to the first byte of the
	// Storage Box: the equipped-armaments block, EquipPhysicsData and the face
	// data, restated literally.
	gestureTestBlocksBefore = 0x9C + 0x0C + 0x12F

	// The Storage Box stands between the face data and GestureGameData and is
	// restated literally as its confirmed size: the four-byte non-empty count,
	// the 0x780 common records, the four-byte key count, the 0x80 key records
	// and the two trailing counters.
	gestureTestStorageBoxSize = 4 + 0x780*12 + 4 + 0x80*12 + 8

	// GestureGameData itself: 64 little-endian uint32 records, 0x100 bytes.
	gestureTestSlotCount   = 64
	gestureTestRecordSize  = 4
	gestureTestSectionSize = gestureTestSlotCount * gestureTestRecordSize

	// The native empty sentinel of one gesture record.
	gestureTestEmptySentinel = uint32(0xFFFFFFFE)
)

// gestureTestAnchor is the 65-byte anchor the gesture chain is measured from,
// restated here independently of the implementation so a changed production
// pattern fails this test: one leading 0x00 byte, then four full repetitions of
// a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes.
var gestureTestAnchor = []byte{
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

// gestureTestFixture describes the synthetic slot content one test save is built
// from. A residual slot is expressed as a zero flag with everything still
// written into the file, and projectiles shifts the block the way an acquired
// projectile list does in a native save.
type gestureTestFixture struct {
	platform    Platform
	slot        int
	flag        byte
	anchorAt    int64
	projectiles uint32
	records     []uint32
	noAnchor    bool
}

// writeGestureFixture builds a synthetic save and returns its path inside
// t.TempDir(). Only the activity flag, the anchor, the projectile count and the
// requested records are written; the rest of the container stays zeroed. A
// record that would reach past the end of the slot data is left out, which is
// how the truncated case is expressed.
func writeGestureFixture(t *testing.T, content gestureTestFixture) string {
	t.Helper()

	var data []byte
	var userData10Base, slotBase int64
	switch content.platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = pcUserData10DataOffset
		slotBase = gestureTestPCSlotDataBase + int64(content.slot)*gestureTestPCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = ps4UserData10DataOffset
		slotBase = gestureTestPS4SlotDataBase + int64(content.slot)*gestureTestPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", content.platform)
	}

	data[userData10Base+userData10ActiveFlagsOffset+int64(content.slot)] = content.flag

	path := filepath.Join(t.TempDir(), "gestures.sl2")
	write := func() string {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		return path
	}

	if content.noAnchor {
		return write()
	}
	copy(data[slotBase+content.anchorAt:], gestureTestAnchor)

	countAt := content.anchorAt + gestureTestProjectileCountAt
	if countAt+4 <= gestureTestSlotDataSize {
		binary.LittleEndian.PutUint32(data[slotBase+countAt:], content.projectiles)
	}
	sectionAt := countAt + 4 +
		int64(content.projectiles)*gestureTestProjectileStride +
		gestureTestBlocksBefore + gestureTestStorageBoxSize

	for index, record := range content.records {
		at := sectionAt + int64(index)*gestureTestRecordSize
		if at+gestureTestRecordSize > gestureTestSlotDataSize {
			continue
		}
		binary.LittleEndian.PutUint32(data[slotBase+at:], record)
	}
	return write()
}

// gestureTestRecords is the full 64-record block the fixtures write. It mixes
// canonical odd slot IDs with a repeated one, an even value no canonical gesture
// carries, a plain zero and the native empty sentinel, so a reader that sorts,
// deduplicates, drops sentinels, masks a value or converts an even value into an
// odd one cannot pass.
func gestureTestRecords() []uint32 {
	records := make([]uint32, gestureTestSlotCount)
	for index := range records {
		records[index] = gestureTestEmptySentinel
	}
	records[0] = 1     // Bow
	records[1] = 185   // Rest
	records[2] = 0     // an explicit zero, not the sentinel
	records[3] = 44    // an even value; no canonical gesture carries it
	records[4] = 185   // the same canonical ID a second time
	records[5] = 229   // Let Us Go Together, a DLC gesture
	records[63] = 3    // the last physical record is occupied
	records[62] = 4242 // an unknown odd value
	return records
}

func gestureTestActiveFixture(
	platform Platform, slot int, anchorAt int64, projectiles uint32,
) gestureTestFixture {
	return gestureTestFixture{
		platform: platform, slot: slot, flag: 1, anchorAt: anchorAt,
		projectiles: projectiles, records: gestureTestRecords(),
	}
}

func TestGetGesturesReadsTheActiveSlotOfBothPlatforms(t *testing.T) {
	// The two fixtures put the anchor at different positions and declare a
	// different number of acquired projectiles, so a reader that depends on a
	// fixed position inside the slot, or that ignores the declared length in
	// front of the block, cannot pass both cases.
	cases := []gestureTestFixture{
		gestureTestActiveFixture(PlatformPC, 0, 0x01A7, 0),
		gestureTestActiveFixture(PlatformPS4, 7, 0x1F4C2, 37),
	}

	for _, content := range cases {
		t.Run(string(content.platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(
				writeGestureFixture(t, content), string(content.platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.GetGestures(loaded.SaveSessionID, content.slot)
			if err != nil {
				t.Fatalf("GetGestures: %v", err)
			}

			want := CharacterGestures{
				SaveSessionID: loaded.SaveSessionID,
				CharacterID:   content.slot,
				Active:        true,
				Slots:         gestureTestRecords(),
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}
		})
	}
}

func TestGetGesturesKeepsEveryRawRecordAsStored(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeGestureFixture(t, gestureTestActiveFixture(PlatformPC, 2, 0x0640, 11)), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.GetGestures(loaded.SaveSessionID, 2)
	if err != nil {
		t.Fatalf("GetGestures: %v", err)
	}

	if len(result.Slots) != gestureTestSlotCount {
		t.Fatalf("slot count = %d, want %d", len(result.Slots), gestureTestSlotCount)
	}

	// Every rule the raw reader must not apply, stated per physical position.
	cases := map[string]struct {
		index int
		want  uint32
	}{
		"canonical odd ID":                {0, 1},
		"explicit zero is not a sentinel": {2, 0},
		"even value stays even":           {3, 44},
		"duplicate ID stays duplicated":   {4, 185},
		"DLC canonical ID":                {5, 229},
		"unknown odd value survives":      {62, 4242},
		"last physical record":            {63, 3},
		"empty sentinel is preserved":     {6, gestureTestEmptySentinel},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			if result.Slots[testCase.index] != testCase.want {
				t.Errorf("slot %d = %d, want %d",
					testCase.index, result.Slots[testCase.index], testCase.want)
			}
		})
	}

	// The duplicate is a property of the raw block, not of one position: both
	// records carry the same canonical ID and neither was collapsed.
	if result.Slots[1] != result.Slots[4] {
		t.Errorf("slots 1 and 4 = %d and %d, want the same duplicated value",
			result.Slots[1], result.Slots[4])
	}
}

func TestGetGesturesReportsAResidualSlotAsInactive(t *testing.T) {
	content := gestureTestActiveFixture(PlatformPC, 4, 0x0800, 0)
	content.flag = 0

	engine := New()
	loaded, err := engine.LoadSave(writeGestureFixture(t, content), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.GetGestures(loaded.SaveSessionID, content.slot)
	if err != nil {
		t.Fatalf("GetGestures: %v", err)
	}

	// The gesture block of the deleted character is still written into the
	// fixture, so an empty result proves it was never located or decoded.
	want := CharacterGestures{
		SaveSessionID: loaded.SaveSessionID,
		CharacterID:   content.slot,
		Slots:         []uint32{},
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
	if result.Slots == nil {
		t.Error("slots is nil, want an empty list")
	}
}

func TestGetGesturesRejectsInvalidRequests(t *testing.T) {
	engine := New()

	loadSlot := func(content gestureTestFixture) string {
		t.Helper()
		loaded, err := engine.LoadSave(writeGestureFixture(t, content), "", "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		return loaded.SaveSessionID
	}

	present := loadSlot(gestureTestActiveFixture(PlatformPC, 2, 0x0640, 0))
	missingAnchor := loadSlot(gestureTestFixture{
		platform: PlatformPC, slot: 2, flag: 1, noAnchor: true,
	})
	// The anchor sits so close to the end of the slot that the last record of
	// the gesture block no longer fits inside the slot data.
	truncated := loadSlot(gestureTestFixture{
		platform: PlatformPC, slot: 2, flag: 1,
		anchorAt: gestureTestSlotDataSize -
			(gestureTestProjectileCountAt + 4 + gestureTestBlocksBefore +
				gestureTestStorageBoxSize + gestureTestSectionSize - 1),
	})
	// A declared projectile count far beyond anything a native save carries is
	// corrupt, not a position to follow.
	corruptCount := loadSlot(gestureTestFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x0640, projectiles: 200001,
	})
	// An accepted count that still pushes the block past the end of the slot is
	// rejected before anything is read.
	shiftedOutOfSlot := loadSlot(gestureTestFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x200000, projectiles: 100000,
	})

	closed := loadSlot(gestureTestActiveFixture(PlatformPC, 2, 0x0640, 0))
	if err := engine.CloseSession(closed); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}

	cases := map[string]struct {
		saveSessionID string
		characterID   int
		want          string
	}{
		"empty session":   {"", 0, "saveSessionID is required"},
		"unknown session": {"missing", 0, `unknown save session "missing"`},
		"closed session":  {closed, 2, `unknown save session ` + strconv.Quote(closed)},
		"characterID -1":  {present, -1, "characterID -1 is outside the range 0..9"},
		"characterID 10":  {present, 10, "characterID 10 is outside the range 0..9"},
		"missing anchor":  {missingAnchor, 2, "character 2 carries no gesture anchor"},
		"truncated block": {truncated, 2, "gestures of character 2 do not fit into their slot"},
		"corrupt projectile count": {corruptCount, 2,
			"character 2 declares 200001 projectile records, want at most 200000"},
		"block pushed out of the slot": {shiftedOutOfSlot, 2,
			"gestures of character 2 do not fit into their slot"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetGestures(testCase.saveSessionID, testCase.characterID)
			if err == nil {
				t.Fatalf("GetGestures accepted %s", strconv.Quote(name))
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, CharacterGestures{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}
