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
	quickItemsPCSlotDataBase  = 0x300 + 0x10 // first PC slot data, behind its MD5 prefix
	quickItemsPCSlotStride    = 0x280010
	quickItemsPS4SlotDataBase = 0x70 // first PS4 slot data, no MD5 prefix
	quickItemsPS4SlotStride   = 0x280000
	quickItemsSlotDataSize    = 0x280000

	// quickItemsSectionAt is the distance from the anchor to the start of
	// EquipItemData, and quickItemsActiveAt the distance from that section to the
	// raw active-slot int32 behind the ten records.
	quickItemsSectionAt = 0x9279
	quickItemsActiveAt  = 0x50

	quickItemsTestCount = 10
)

// quickItemsTestAnchor is the 65-byte anchor the quick-items chain is measured
// from, restated here independently of the implementation so a changed
// production pattern fails this test: one leading 0x00 byte, then four full
// repetitions of a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed by twelve
// 0x00 bytes.
var quickItemsTestAnchor = []byte{
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

// quickItemsFixture describes the synthetic slot content one test save is built
// from: which slot carries which activity flag, where its anchor sits relative
// to the start of the slot data, which raw record pairs the EquipItemData
// section holds and which raw active-slot value follows them. A residual slot is
// expressed as a zero flag with everything still written into the file.
type quickItemsFixture struct {
	platform    Platform
	slot        int
	flag        byte
	anchorAt    int64
	items       [quickItemsTestCount]QuickItemSlot
	activeQuick uint32
	noAnchor    bool
}

// writeQuickItemsFixture builds a synthetic save and returns its path inside
// t.TempDir(). Only the activity flag, the anchor, the ten records and the
// active-slot value are written; the rest of the container stays zeroed. A value
// that would reach past the end of the slot data is left out, which is how the
// out-of-bounds case is expressed.
func writeQuickItemsFixture(t *testing.T, content quickItemsFixture) string {
	t.Helper()

	var data []byte
	var userData10Base, slotBase int64
	switch content.platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = pcUserData10DataOffset
		slotBase = quickItemsPCSlotDataBase + int64(content.slot)*quickItemsPCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = ps4UserData10DataOffset
		slotBase = quickItemsPS4SlotDataBase + int64(content.slot)*quickItemsPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", content.platform)
	}

	data[userData10Base+userData10ActiveFlagsOffset+int64(content.slot)] = content.flag

	path := filepath.Join(t.TempDir(), "quick-items.sl2")
	write := func() string {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		return path
	}

	if content.noAnchor {
		return write()
	}

	copy(data[slotBase+content.anchorAt:], quickItemsTestAnchor)

	sectionAt := content.anchorAt + quickItemsSectionAt
	putUint32 := func(at int64, value uint32) {
		if at+4 > quickItemsSlotDataSize {
			return
		}
		binary.LittleEndian.PutUint32(data[slotBase+at:], value)
	}
	for index, item := range content.items {
		putUint32(sectionAt+int64(index)*8, item.ItemID)
		putUint32(sectionAt+int64(index)*8+4, item.EquipIndex)
	}
	putUint32(sectionAt+quickItemsActiveAt, content.activeQuick)

	return write()
}

// quickItemsValues fills all ten records with distinct pairs and deliberately
// includes 0, 0xFFFFFFFF and values whose high bit is set, so a reader that
// masks, normalises or drops type bits cannot pass.
func quickItemsValues(seed uint32) [quickItemsTestCount]QuickItemSlot {
	var items [quickItemsTestCount]QuickItemSlot
	for index := range items {
		items[index] = QuickItemSlot{
			ItemID:     0x00110000*uint32(index+1) + seed,
			EquipIndex: uint32(index)*3 + seed,
		}
	}
	items[0] = QuickItemSlot{ItemID: 0, EquipIndex: 0xFFFFFFFF}
	items[3] = QuickItemSlot{ItemID: 0xFFFFFFFF, EquipIndex: 0xFFFFFFFF}
	items[4] = QuickItemSlot{ItemID: 0x80000000 | (0x0A01 + seed), EquipIndex: 0x80000004}
	items[7] = QuickItemSlot{ItemID: 0, EquipIndex: 0}
	items[9] = QuickItemSlot{ItemID: 0x90000000 | (0x0B02 + seed), EquipIndex: 0xFFFFFFF6}
	return items
}

func TestGetQuickItemsReadsTheActiveSlotOfBothPlatforms(t *testing.T) {
	// The two fixtures put the anchor at different positions, so a reader that
	// depends on a fixed position inside the slot cannot pass both cases. The PS4
	// fixture stores a negative active-slot value, which only survives if the
	// field is reported as a signed int32.
	cases := []struct {
		content quickItemsFixture
		want    int32
	}{
		{
			content: quickItemsFixture{
				platform: PlatformPC, slot: 0, flag: 1, anchorAt: 0x01A7,
				items: quickItemsValues(0x11), activeQuick: 4,
			},
			want: 4,
		},
		{
			content: quickItemsFixture{
				platform: PlatformPS4, slot: 7, flag: 1, anchorAt: 0x1F4C2,
				items: quickItemsValues(0x27), activeQuick: 0xFFFFFFF6,
			},
			want: -10,
		},
	}

	for _, testCase := range cases {
		t.Run(string(testCase.content.platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(
				writeQuickItemsFixture(t, testCase.content), string(testCase.content.platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.GetQuickItems(loaded.SaveSessionID, testCase.content.slot)
			if err != nil {
				t.Fatalf("GetQuickItems: %v", err)
			}

			want := CharacterQuickItems{
				SaveSessionID: loaded.SaveSessionID,
				CharacterID:   testCase.content.slot,
				Active:        true,
				Items:         testCase.content.items,
				ActiveQuick:   testCase.want,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}
		})
	}
}

func TestGetQuickItemsReportsAResidualSlotAsInactive(t *testing.T) {
	content := quickItemsFixture{
		platform: PlatformPC, slot: 4, flag: 0, anchorAt: 0x0800,
		items: quickItemsValues(0x31), activeQuick: 0xFFFFFFF6,
	}

	engine := New()
	loaded, err := engine.LoadSave(writeQuickItemsFixture(t, content), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.GetQuickItems(loaded.SaveSessionID, content.slot)
	if err != nil {
		t.Fatalf("GetQuickItems: %v", err)
	}

	want := CharacterQuickItems{SaveSessionID: loaded.SaveSessionID, CharacterID: content.slot}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetQuickItemsRejectsInvalidRequests(t *testing.T) {
	engine := New()

	loadSlot := func(content quickItemsFixture) string {
		t.Helper()
		loaded, err := engine.LoadSave(writeQuickItemsFixture(t, content), "", "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		return loaded.SaveSessionID
	}

	present := loadSlot(quickItemsFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x0640,
		items: quickItemsValues(0x05), activeQuick: 2,
	})
	missingAnchor := loadSlot(quickItemsFixture{
		platform: PlatformPC, slot: 2, flag: 1, noAnchor: true,
	})
	// The anchor sits so close to the end of the slot that the ten records still
	// fit but the active-slot value behind them no longer does.
	truncated := loadSlot(quickItemsFixture{
		platform: PlatformPC, slot: 2, flag: 1,
		anchorAt: quickItemsSlotDataSize - (quickItemsSectionAt + quickItemsActiveAt + 3),
		items:    quickItemsValues(0x05), activeQuick: 2,
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
		"missing anchor":  {missingAnchor, 2, "character 2 carries no quick-items anchor"},
		"truncated section": {truncated, 2,
			"quick items of character 2 do not fit into its slot"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetQuickItems(testCase.saveSessionID, testCase.characterID)
			if err == nil {
				t.Fatalf("GetQuickItems accepted %s", strconv.Quote(name))
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, CharacterQuickItems{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}
