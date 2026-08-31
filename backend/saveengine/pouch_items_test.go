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
	pouchItemsPCSlotDataBase  = 0x300 + 0x10 // first PC slot data, behind its MD5 prefix
	pouchItemsPCSlotStride    = 0x280010
	pouchItemsPS4SlotDataBase = 0x70 // first PS4 slot data, no MD5 prefix
	pouchItemsPS4SlotStride   = 0x280000
	pouchItemsSlotDataSize    = 0x280000

	// pouchItemsSectionAt is the distance from the anchor to the first pouch
	// record: 0x9279 to the start of EquipItemData, then the ten quick-item
	// records (0x50) and the four-byte active-quick value behind them.
	pouchItemsSectionAt = 0x92CD

	pouchItemsTestCount = 6
)

// pouchItemsTestAnchor is the 65-byte anchor the pouch chain is measured from,
// restated here independently of the implementation so a changed production
// pattern fails this test: one leading 0x00 byte, then four full repetitions of
// a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes.
var pouchItemsTestAnchor = []byte{
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

// pouchItemsFixture describes the synthetic slot content one test save is built
// from: which slot carries which activity flag, where its anchor sits relative
// to the start of the slot data and which raw record pairs the pouch section
// holds. A residual slot is expressed as a zero flag with everything still
// written into the file.
type pouchItemsFixture struct {
	platform Platform
	slot     int
	flag     byte
	anchorAt int64
	items    [pouchItemsTestCount]PouchItemSlot
	noAnchor bool
}

// writePouchItemsFixture builds a synthetic save and returns its path inside
// t.TempDir(). Only the activity flag, the anchor and the six records are
// written; the rest of the container stays zeroed. A value that would reach past
// the end of the slot data is left out, which is how the out-of-bounds case is
// expressed.
func writePouchItemsFixture(t *testing.T, content pouchItemsFixture) string {
	t.Helper()

	var data []byte
	var userData10Base, slotBase int64
	switch content.platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = pcUserData10DataOffset
		slotBase = pouchItemsPCSlotDataBase + int64(content.slot)*pouchItemsPCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = ps4UserData10DataOffset
		slotBase = pouchItemsPS4SlotDataBase + int64(content.slot)*pouchItemsPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", content.platform)
	}

	data[userData10Base+userData10ActiveFlagsOffset+int64(content.slot)] = content.flag

	path := filepath.Join(t.TempDir(), "pouch-items.sl2")
	write := func() string {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		return path
	}

	if content.noAnchor {
		return write()
	}

	copy(data[slotBase+content.anchorAt:], pouchItemsTestAnchor)

	sectionAt := content.anchorAt + pouchItemsSectionAt
	putUint32 := func(at int64, value uint32) {
		if at+4 > pouchItemsSlotDataSize {
			return
		}
		binary.LittleEndian.PutUint32(data[slotBase+at:], value)
	}
	for index, item := range content.items {
		putUint32(sectionAt+int64(index)*8, item.ItemID)
		putUint32(sectionAt+int64(index)*8+4, item.EquipIndex)
	}

	return write()
}

// pouchItemsValues fills all six records with distinct pairs and deliberately
// includes 0, 0xFFFFFFFF and values whose high bit is set, so a reader that
// masks, normalises or drops type bits cannot pass.
func pouchItemsValues(seed uint32) [pouchItemsTestCount]PouchItemSlot {
	var items [pouchItemsTestCount]PouchItemSlot
	for index := range items {
		items[index] = PouchItemSlot{
			ItemID:     0x00110000*uint32(index+1) + seed,
			EquipIndex: uint32(index)*3 + seed,
		}
	}
	items[0] = PouchItemSlot{ItemID: 0, EquipIndex: 0xFFFFFFFF}
	items[2] = PouchItemSlot{ItemID: 0xFFFFFFFF, EquipIndex: 0xFFFFFFFF}
	items[3] = PouchItemSlot{ItemID: 0x80000000 | (0x0A01 + seed), EquipIndex: 0x80000004}
	items[4] = PouchItemSlot{ItemID: 0, EquipIndex: 0}
	items[5] = PouchItemSlot{ItemID: 0x90000000 | (0x0B02 + seed), EquipIndex: 0xFFFFFFF6}
	return items
}

func TestGetPouchItemsReadsTheActiveSlotOfBothPlatforms(t *testing.T) {
	// The two fixtures put the anchor at different positions, so a reader that
	// depends on a fixed position inside the slot cannot pass both cases.
	cases := []pouchItemsFixture{
		{platform: PlatformPC, slot: 0, flag: 1, anchorAt: 0x01A7, items: pouchItemsValues(0x11)},
		{platform: PlatformPS4, slot: 7, flag: 1, anchorAt: 0x1F4C2, items: pouchItemsValues(0x27)},
	}

	for _, content := range cases {
		t.Run(string(content.platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(
				writePouchItemsFixture(t, content), string(content.platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.GetPouchItems(loaded.SaveSessionID, content.slot)
			if err != nil {
				t.Fatalf("GetPouchItems: %v", err)
			}

			want := CharacterPouchItems{
				SaveSessionID: loaded.SaveSessionID,
				CharacterID:   content.slot,
				Active:        true,
				Items:         content.items,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}
		})
	}
}

func TestGetPouchItemsReportsAResidualSlotAsInactive(t *testing.T) {
	content := pouchItemsFixture{
		platform: PlatformPC, slot: 4, flag: 0, anchorAt: 0x0800,
		items: pouchItemsValues(0x31),
	}

	engine := New()
	loaded, err := engine.LoadSave(writePouchItemsFixture(t, content), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.GetPouchItems(loaded.SaveSessionID, content.slot)
	if err != nil {
		t.Fatalf("GetPouchItems: %v", err)
	}

	want := CharacterPouchItems{SaveSessionID: loaded.SaveSessionID, CharacterID: content.slot}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetPouchItemsRejectsInvalidRequests(t *testing.T) {
	engine := New()

	loadSlot := func(content pouchItemsFixture) string {
		t.Helper()
		loaded, err := engine.LoadSave(writePouchItemsFixture(t, content), "", "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		return loaded.SaveSessionID
	}

	present := loadSlot(pouchItemsFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x0640,
		items: pouchItemsValues(0x05),
	})
	missingAnchor := loadSlot(pouchItemsFixture{
		platform: PlatformPC, slot: 2, flag: 1, noAnchor: true,
	})
	// The anchor sits so close to the end of the slot that the last record no
	// longer fits inside the slot data.
	truncated := loadSlot(pouchItemsFixture{
		platform: PlatformPC, slot: 2, flag: 1,
		anchorAt: pouchItemsSlotDataSize - (pouchItemsSectionAt + 6*8 - 1),
		items:    pouchItemsValues(0x05),
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
		"missing anchor":  {missingAnchor, 2, "character 2 carries no pouch-items anchor"},
		"truncated section": {truncated, 2,
			"pouch items of character 2 do not fit into its slot"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetPouchItems(testCase.saveSessionID, testCase.characterID)
			if err == nil {
				t.Fatalf("GetPouchItems accepted %s", strconv.Quote(name))
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, CharacterPouchItems{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}
