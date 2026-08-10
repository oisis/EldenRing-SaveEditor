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
	inventoryTestPCSlotDataBase  = 0x300 + 0x10 // first PC slot data, behind its MD5 prefix
	inventoryTestPCSlotStride    = 0x280010
	inventoryTestPS4SlotDataBase = 0x70 // first PS4 slot data, no MD5 prefix
	inventoryTestPS4SlotStride   = 0x280000
	inventoryTestSlotDataSize    = 0x280000

	// Distances from the anchor: the common records start behind the fixed
	// structures and the four-byte common-item count, the key records behind the
	// 0xA80 common records and the four-byte key-item count, and the two trailing
	// counters behind the 0x180 key records.
	inventoryTestCommonAt   = 505
	inventoryTestRecordSize = 12
	inventoryTestCommonSize = 0xA80 * inventoryTestRecordSize
	inventoryTestKeyAt      = inventoryTestCommonAt + inventoryTestCommonSize + 4
	inventoryTestKeySize    = 0x180 * inventoryTestRecordSize
	inventoryTestEnd        = inventoryTestKeyAt + inventoryTestKeySize + 8
)

// inventoryTestAnchor is the 65-byte anchor the inventory section is measured
// from, restated here independently of the implementation so a changed
// production pattern fails this test: one leading 0x00 byte, then four full
// repetitions of a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed by twelve
// 0x00 bytes.
var inventoryTestAnchor = []byte{
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

// inventoryTestRow is one raw record written into a synthetic section, with the
// stored quantity exactly as the game keeps it, including the high bit.
type inventoryTestRow struct {
	index       int
	handle      uint32
	rawQuantity uint32
	acquisition uint32
}

// inventoryTestFixture describes the synthetic slot content one test save is
// built from. A residual slot is expressed as a zero flag with everything still
// written into the file.
type inventoryTestFixture struct {
	platform Platform
	slot     int
	flag     byte
	anchorAt int64
	common   []inventoryTestRow
	key      []inventoryTestRow
	noAnchor bool
}

// writeInventoryFixture builds a synthetic save and returns its path inside
// t.TempDir(). Only the activity flag, the anchor and the requested records are
// written; the rest of the container stays zeroed, which is the native empty
// sentinel of a record. A value that would reach past the end of the slot data
// is left out, which is how the truncated case is expressed.
func writeInventoryFixture(t *testing.T, content inventoryTestFixture) string {
	t.Helper()

	var data []byte
	var userData10Base, slotBase int64
	switch content.platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = pcUserData10DataOffset
		slotBase = inventoryTestPCSlotDataBase + int64(content.slot)*inventoryTestPCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = ps4UserData10DataOffset
		slotBase = inventoryTestPS4SlotDataBase + int64(content.slot)*inventoryTestPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", content.platform)
	}

	data[userData10Base+userData10ActiveFlagsOffset+int64(content.slot)] = content.flag

	path := filepath.Join(t.TempDir(), "inventory.sl2")
	write := func() string {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		return path
	}

	if content.noAnchor {
		return write()
	}
	copy(data[slotBase+content.anchorAt:], inventoryTestAnchor)

	putRow := func(sectionAt int64, row inventoryTestRow) {
		at := content.anchorAt + sectionAt + int64(row.index)*inventoryTestRecordSize
		if at+inventoryTestRecordSize > inventoryTestSlotDataSize {
			return
		}
		binary.LittleEndian.PutUint32(data[slotBase+at:], row.handle)
		binary.LittleEndian.PutUint32(data[slotBase+at+4:], row.rawQuantity)
		binary.LittleEndian.PutUint32(data[slotBase+at+8:], row.acquisition)
	}
	for _, row := range content.common {
		putRow(inventoryTestCommonAt, row)
	}
	for _, row := range content.key {
		putRow(inventoryTestKeyAt, row)
	}
	return write()
}

// inventoryTestCommonRows and inventoryTestKeyRows deliberately mix occupied
// rows with both native absent sentinels and leave gaps between them, so a
// reader that renumbers rows, drops the wrong sentinel or reorders the sections
// cannot pass.
func inventoryTestCommonRows() []inventoryTestRow {
	return []inventoryTestRow{
		{index: 0, handle: 0x00000000, rawQuantity: 0x00000005, acquisition: 1},
		{index: 1, handle: 0xB000272E, rawQuantity: 0x80000003, acquisition: 7},
		{index: 2, handle: 0xFFFFFFFF, rawQuantity: 0x00000009, acquisition: 2},
		{index: 5, handle: 0x90001111, rawQuantity: 0x00000001, acquisition: 9},
		{index: 0xA7F, handle: 0x80000002, rawQuantity: 0xFFFFFFFF, acquisition: 0xFFFFFFFF},
	}
}

func inventoryTestKeyRows() []inventoryTestRow {
	return []inventoryTestRow{
		{index: 0, handle: 0xFFFFFFFF, rawQuantity: 0x00000004, acquisition: 3},
		{index: 2, handle: 0xC0000001, rawQuantity: 0x80000001, acquisition: 12},
		{index: 0x17F, handle: 0x00000003, rawQuantity: 0x00000002, acquisition: 40},
	}
}

// inventoryTestWantCommon and inventoryTestWantKey are the records the two row
// sets must produce: the sentinels are gone, the physical index of every
// remaining row is preserved, and the high bit of the quantity is masked off
// while the handle and the acquisition index stay untouched.
func inventoryTestWantCommon() []InventoryRecord {
	return []InventoryRecord{
		{ContainerSection: "common", PhysicalIndex: 1, GaItemHandle: 0xB000272E, Quantity: 3, AcquisitionIndex: 7},
		{ContainerSection: "common", PhysicalIndex: 5, GaItemHandle: 0x90001111, Quantity: 1, AcquisitionIndex: 9},
		{ContainerSection: "common", PhysicalIndex: 0xA7F, GaItemHandle: 0x80000002,
			Quantity: 0x7FFFFFFF, AcquisitionIndex: 0xFFFFFFFF},
	}
}

func inventoryTestWantKey() []InventoryRecord {
	return []InventoryRecord{
		{ContainerSection: "key", PhysicalIndex: 2, GaItemHandle: 0xC0000001, Quantity: 1, AcquisitionIndex: 12},
		{ContainerSection: "key", PhysicalIndex: 0x17F, GaItemHandle: 0x00000003, Quantity: 2, AcquisitionIndex: 40},
	}
}

func inventoryTestActiveFixture(platform Platform, slot int, anchorAt int64) inventoryTestFixture {
	return inventoryTestFixture{
		platform: platform, slot: slot, flag: 1, anchorAt: anchorAt,
		common: inventoryTestCommonRows(), key: inventoryTestKeyRows(),
	}
}

func TestGetInventoryReadsTheActiveSlotOfBothPlatforms(t *testing.T) {
	// The two fixtures put the anchor at different positions, so a reader that
	// depends on a fixed position inside the slot cannot pass both cases.
	cases := []inventoryTestFixture{
		inventoryTestActiveFixture(PlatformPC, 0, 0x01A7),
		inventoryTestActiveFixture(PlatformPS4, 7, 0x1F4C2),
	}

	for _, content := range cases {
		t.Run(string(content.platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(
				writeInventoryFixture(t, content), string(content.platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.GetInventory(loaded.SaveSessionID, content.slot, "", 0, 0)
			if err != nil {
				t.Fatalf("GetInventory: %v", err)
			}

			want := CharacterInventory{
				SaveSessionID: loaded.SaveSessionID,
				CharacterID:   content.slot,
				Active:        true,
				Records:       append(inventoryTestWantCommon(), inventoryTestWantKey()...),
				Total:         5,
				Page:          1,
				PageSize:      50,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}
		})
	}
}

func TestGetInventoryFiltersBySection(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeInventoryFixture(t, inventoryTestActiveFixture(PlatformPC, 3, 0x0640)), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	cases := map[string]struct {
		section string
		want    []InventoryRecord
	}{
		"common": {InventorySectionCommon, inventoryTestWantCommon()},
		"key":    {InventorySectionKey, inventoryTestWantKey()},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetInventory(loaded.SaveSessionID, 3, testCase.section, 0, 0)
			if err != nil {
				t.Fatalf("GetInventory: %v", err)
			}
			if result.Total != len(testCase.want) {
				t.Fatalf("total = %d, want %d", result.Total, len(testCase.want))
			}
			if !reflect.DeepEqual(result.Records, testCase.want) {
				t.Errorf("records = %+v, want %+v", result.Records, testCase.want)
			}
		})
	}
}

func TestGetInventoryPagesWithoutLosingTheNativeOrder(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeInventoryFixture(t, inventoryTestActiveFixture(PlatformPC, 1, 0x0640)), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	all := append(inventoryTestWantCommon(), inventoryTestWantKey()...)

	cases := map[string]struct {
		page, pageSize int
		wantPage       int
		wantPageSize   int
		want           []InventoryRecord
	}{
		"first page":          {1, 2, 1, 2, all[:2]},
		"last partial page":   {3, 2, 3, 2, all[4:]},
		"page zero means one": {0, 2, 1, 2, all[:2]},
		"beyond the total":    {9, 2, 9, 2, []InventoryRecord{}},
		"one page for all":    {1, 0, 1, 50, all},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetInventory(loaded.SaveSessionID, 1, "", testCase.page, testCase.pageSize)
			if err != nil {
				t.Fatalf("GetInventory: %v", err)
			}
			if result.Total != len(all) {
				t.Errorf("total = %d, want %d", result.Total, len(all))
			}
			if result.Page != testCase.wantPage || result.PageSize != testCase.wantPageSize {
				t.Errorf("page/pageSize = %d/%d, want %d/%d",
					result.Page, result.PageSize, testCase.wantPage, testCase.wantPageSize)
			}
			if !reflect.DeepEqual(result.Records, testCase.want) {
				t.Errorf("records = %+v, want %+v", result.Records, testCase.want)
			}
			if result.Records == nil {
				t.Error("records is nil, want an empty list")
			}
		})
	}
}

func TestGetInventoryReportsAResidualSlotAsInactive(t *testing.T) {
	content := inventoryTestActiveFixture(PlatformPC, 4, 0x0800)
	content.flag = 0

	engine := New()
	loaded, err := engine.LoadSave(writeInventoryFixture(t, content), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.GetInventory(loaded.SaveSessionID, content.slot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}

	want := CharacterInventory{
		SaveSessionID: loaded.SaveSessionID,
		CharacterID:   content.slot,
		Records:       []InventoryRecord{},
		Page:          1,
		PageSize:      50,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetInventoryRejectsInvalidRequests(t *testing.T) {
	engine := New()

	loadSlot := func(content inventoryTestFixture) string {
		t.Helper()
		loaded, err := engine.LoadSave(writeInventoryFixture(t, content), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		return loaded.SaveSessionID
	}

	present := loadSlot(inventoryTestActiveFixture(PlatformPC, 2, 0x0640))
	missingAnchor := loadSlot(inventoryTestFixture{
		platform: PlatformPC, slot: 2, flag: 1, noAnchor: true,
	})
	// The anchor sits so close to the end of the slot that the last trailing
	// counter no longer fits inside the slot data.
	truncated := loadSlot(inventoryTestFixture{
		platform: PlatformPC, slot: 2, flag: 1,
		anchorAt: inventoryTestSlotDataSize - (inventoryTestEnd - 1),
	})

	closed := loadSlot(inventoryTestActiveFixture(PlatformPC, 2, 0x0640))
	if err := engine.CloseSession(closed); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}

	cases := map[string]struct {
		saveSessionID    string
		characterID      int
		containerSection string
		page, pageSize   int
		want             string
	}{
		"empty session":   {"", 0, "", 0, 0, "saveSessionID is required"},
		"unknown session": {"missing", 0, "", 0, 0, `unknown save session "missing"`},
		"closed session":  {closed, 2, "", 0, 0, `unknown save session ` + strconv.Quote(closed)},
		"characterID -1":  {present, -1, "", 0, 0, "characterID -1 is outside the range 0..9"},
		"characterID 10":  {present, 10, "", 0, 0, "characterID 10 is outside the range 0..9"},
		"unknown section": {present, 2, "Common", 0, 0,
			`containerSection must be "common", "key" or empty; got "Common"`},
		"padded section": {present, 2, " key", 0, 0,
			`containerSection must be "common", "key" or empty; got " key"`},
		"negative page":     {present, 2, "", -1, 0, "page must not be negative; got -1"},
		"negative pageSize": {present, 2, "", 0, -5, "pageSize must not be negative; got -5"},
		"missing anchor":    {missingAnchor, 2, "", 0, 0, "character 2 carries no inventory anchor"},
		"truncated section": {truncated, 2, "", 0, 0,
			"inventory of character 2 does not fit into its slot"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetInventory(
				testCase.saveSessionID, testCase.characterID,
				testCase.containerSection, testCase.page, testCase.pageSize)
			if err == nil {
				t.Fatalf("GetInventory accepted %s", strconv.Quote(name))
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, CharacterInventory{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}
