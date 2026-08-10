package inventory

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Synthetic container layout used only by this test. The endpoint owns none of
// these values; they are duplicated here so the fixture is accepted by
// SaveEngine without sharing anything with another test file.
const (
	getInventoryHeaderSize       = 0x300
	getInventoryEntryCountOffset = 0x0C
	getInventoryEntryCount       = 12
	getInventorySlotBlockSize    = 0x280010
	getInventorySlotDataSize     = 0x280000
	getInventoryFixtureSize      = int64(getInventoryHeaderSize) + 10*getInventorySlotBlockSize + 0x60010

	getInventoryUserData10Offset = int64(getInventoryHeaderSize) + 10*getInventorySlotBlockSize + 0x10
	getInventoryFlagsOffset      = 0x1954

	getInventoryPS4HeaderSize      = 0x70
	getInventoryPS4SlotSize        = 0x280000
	getInventoryPS4FixtureSize     = int64(getInventoryPS4HeaderSize) + 10*getInventoryPS4SlotSize + 0x60000
	getInventoryPS4UserData10Start = int64(getInventoryPS4HeaderSize) + 10*getInventoryPS4SlotSize
	getInventoryPS4EntryTable      = 0x10
	getInventoryPS4FirstEntry      = 7
	getInventoryPS4EntryMarker     = 0x7F7F7F7F

	getInventorySlot       = 3
	getInventoryAnchorAt   = 0x0640
	getInventoryRecordSize = 12
	getInventoryCommonAt   = 505
	getInventoryKeyAt      = getInventoryCommonAt + 0xA80*getInventoryRecordSize + 4
	getInventorySectionEnd = getInventoryKeyAt + 0x180*getInventoryRecordSize + 8
)

// getInventoryAnchor is the 65-byte anchor the inventory section is measured
// from, restated here independently of the implementation: one leading 0x00
// byte, then four full repetitions of a 16-byte block made of 0xFF 0xFF 0xFF
// 0xFF followed by twelve 0x00 bytes.
var getInventoryAnchor = []byte{
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

// getInventoryRow is one raw record written into a synthetic section, with the
// stored quantity exactly as the game keeps it, including the high bit.
type getInventoryRow struct {
	index       int
	handle      uint32
	rawQuantity uint32
	acquisition uint32
}

// The two sections deliberately mix occupied rows with both native absent
// sentinels and leave gaps between them, so a reader that renumbers rows, drops
// the wrong sentinel or reorders the sections cannot pass.
func getInventoryCommonRows() []getInventoryRow {
	return []getInventoryRow{
		{index: 0, handle: 0x00000000, rawQuantity: 6, acquisition: 1},
		{index: 1, handle: 0xB000272E, rawQuantity: 0x80000003, acquisition: 7},
		{index: 4, handle: 0xFFFFFFFF, rawQuantity: 8, acquisition: 2},
		{index: 6, handle: 0x90001111, rawQuantity: 1, acquisition: 9},
	}
}

func getInventoryKeyRows() []getInventoryRow {
	return []getInventoryRow{
		{index: 1, handle: 0xC0000001, rawQuantity: 0x80000001, acquisition: 12},
	}
}

func getInventoryWantCommon() []saveengine.InventoryRecord {
	return []saveengine.InventoryRecord{
		{ContainerSection: "common", PhysicalIndex: 1, GaItemHandle: 0xB000272E, Quantity: 3, AcquisitionIndex: 7},
		{ContainerSection: "common", PhysicalIndex: 6, GaItemHandle: 0x90001111, Quantity: 1, AcquisitionIndex: 9},
	}
}

func getInventoryWantKey() []saveengine.InventoryRecord {
	return []saveengine.InventoryRecord{
		{ContainerSection: "key", PhysicalIndex: 1, GaItemHandle: 0xC0000001, Quantity: 1, AcquisitionIndex: 12},
	}
}

func getInventoryWantAll() []saveengine.InventoryRecord {
	return append(getInventoryWantCommon(), getInventoryWantKey()...)
}

// writeGetInventoryFixture writes a minimal synthetic save of the given platform
// into t.TempDir() with one active character carrying the raw records above, and
// returns its path. anchorAt places the anchor inside the slot data, so a
// position that leaves no room for the whole section expresses the truncated
// case.
func writeGetInventoryFixture(t *testing.T, platform string, active bool, anchorAt int64) string {
	t.Helper()

	var data []byte
	var slotBase int64
	switch platform {
	case "ps4":
		data = make([]byte, getInventoryPS4FixtureSize)
		data[0], data[1], data[2], data[3] = 0xCB, 0x01, 0x9C, 0x2C
		for entry := 0; entry < getInventoryEntryCount; entry++ {
			at := getInventoryPS4EntryTable + entry*8
			binary.LittleEndian.PutUint32(data[at:], uint32(getInventoryPS4FirstEntry+entry))
			binary.LittleEndian.PutUint32(data[at+4:], getInventoryPS4EntryMarker)
		}
		if active {
			data[getInventoryPS4UserData10Start+getInventoryFlagsOffset+getInventorySlot] = 1
		}
		slotBase = int64(getInventoryPS4HeaderSize) + getInventorySlot*getInventoryPS4SlotSize
	default:
		data = make([]byte, getInventoryFixtureSize)
		copy(data, []byte("BND4"))
		binary.LittleEndian.PutUint32(data[getInventoryEntryCountOffset:], getInventoryEntryCount)
		if active {
			data[getInventoryUserData10Offset+getInventoryFlagsOffset+getInventorySlot] = 1
		}
		slotBase = int64(getInventoryHeaderSize) + 0x10 + getInventorySlot*getInventorySlotBlockSize
	}

	copy(data[slotBase+anchorAt:], getInventoryAnchor)

	putRow := func(sectionAt int64, row getInventoryRow) {
		at := anchorAt + sectionAt + int64(row.index)*getInventoryRecordSize
		if at+getInventoryRecordSize > getInventorySlotDataSize {
			return
		}
		binary.LittleEndian.PutUint32(data[slotBase+at:], row.handle)
		binary.LittleEndian.PutUint32(data[slotBase+at+4:], row.rawQuantity)
		binary.LittleEndian.PutUint32(data[slotBase+at+8:], row.acquisition)
	}
	for _, row := range getInventoryCommonRows() {
		putRow(getInventoryCommonAt, row)
	}
	for _, row := range getInventoryKeyRows() {
		putRow(getInventoryKeyAt, row)
	}

	path := filepath.Join(t.TempDir(), "get-inventory.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// loadGetInventorySession loads one synthetic save and returns the engine and the
// session identifier the endpoint is called with.
func loadGetInventorySession(t *testing.T, platform string, active bool, anchorAt int64) (*saveengine.Engine, string) {
	t.Helper()

	engine := saveengine.New()
	session, err := engine.LoadSave(writeGetInventoryFixture(t, platform, active, anchorAt), platform)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, session.SaveSessionID
}

func TestGetInventoryReturnsTheRawRecordsOfBothPlatforms(t *testing.T) {
	for _, platform := range []string{"pc", "ps4"} {
		t.Run(platform, func(t *testing.T) {
			engine, sessionID := loadGetInventorySession(t, platform, true, getInventoryAnchorAt)

			result, err := GetInventory(engine, sessionID, getInventorySlot, "", 0, 0)
			if err != nil {
				t.Fatalf("GetInventory: %v", err)
			}

			want := GetInventoryResult{
				SaveSessionID: sessionID,
				CharacterID:   getInventorySlot,
				Active:        true,
				Records:       getInventoryWantAll(),
				Total:         3,
				Page:          1,
				PageSize:      50,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}
		})
	}
}

func TestGetInventoryFiltersAndPages(t *testing.T) {
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	all := getInventoryWantAll()

	cases := map[string]struct {
		section        string
		page, pageSize int
		wantTotal      int
		wantPage       int
		wantPageSize   int
		want           []saveengine.InventoryRecord
	}{
		"common only":  {"common", 0, 0, 2, 1, 50, getInventoryWantCommon()},
		"key only":     {"key", 0, 0, 1, 1, 50, getInventoryWantKey()},
		"first page":   {"", 1, 2, 3, 1, 2, all[:2]},
		"last page":    {"", 2, 2, 3, 2, 2, all[2:]},
		"beyond total": {"", 7, 2, 3, 7, 2, []saveengine.InventoryRecord{}},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := GetInventory(
				engine, sessionID, getInventorySlot, testCase.section, testCase.page, testCase.pageSize)
			if err != nil {
				t.Fatalf("GetInventory: %v", err)
			}
			if result.Total != testCase.wantTotal {
				t.Errorf("total = %d, want %d", result.Total, testCase.wantTotal)
			}
			if result.Page != testCase.wantPage || result.PageSize != testCase.wantPageSize {
				t.Errorf("page/pageSize = %d/%d, want %d/%d",
					result.Page, result.PageSize, testCase.wantPage, testCase.wantPageSize)
			}
			if result.Records == nil {
				t.Fatal("records is nil, want an empty list")
			}
			if !reflect.DeepEqual(result.Records, testCase.want) {
				t.Errorf("records = %+v, want %+v", result.Records, testCase.want)
			}
		})
	}
}

func TestGetInventoryReportsAResidualSlotAsInactive(t *testing.T) {
	engine, sessionID := loadGetInventorySession(t, "pc", false, getInventoryAnchorAt)

	result, err := GetInventory(engine, sessionID, getInventorySlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}

	want := GetInventoryResult{
		SaveSessionID: sessionID,
		CharacterID:   getInventorySlot,
		Records:       []saveengine.InventoryRecord{},
		Page:          1,
		PageSize:      50,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetInventoryRejectsInvalidRequests(t *testing.T) {
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	// The anchor sits so close to the end of the slot that the section no longer
	// fits inside the slot data.
	truncatedEngine, truncatedID := loadGetInventorySession(
		t, "pc", true, getInventorySlotDataSize-(getInventorySectionEnd-1))

	closedEngine, closedID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	if err := closedEngine.CloseSession(closedID); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}

	cases := map[string]struct {
		engine         *saveengine.Engine
		saveSessionID  string
		characterID    int
		section        string
		page, pageSize int
		want           string
	}{
		"nil engine":      {nil, sessionID, getInventorySlot, "", 0, 0, "save engine is not available"},
		"empty session":   {engine, "", getInventorySlot, "", 0, 0, "saveSessionID is required"},
		"unknown session": {engine, "missing", getInventorySlot, "", 0, 0, `unknown save session "missing"`},
		"closed session": {closedEngine, closedID, getInventorySlot, "", 0, 0,
			`unknown save session ` + strconv.Quote(closedID)},
		"characterID -1": {engine, sessionID, -1, "", 0, 0, "characterID -1 is outside the range 0..9"},
		"characterID 10": {engine, sessionID, 10, "", 0, 0, "characterID 10 is outside the range 0..9"},
		"unknown section": {engine, sessionID, getInventorySlot, "storage", 0, 0,
			`containerSection must be "common", "key" or empty; got "storage"`},
		"case-shifted section": {engine, sessionID, getInventorySlot, "Key", 0, 0,
			`containerSection must be "common", "key" or empty; got "Key"`},
		"negative page":     {engine, sessionID, getInventorySlot, "", -2, 0, "page must not be negative; got -2"},
		"negative pageSize": {engine, sessionID, getInventorySlot, "", 0, -1, "pageSize must not be negative; got -1"},
		"truncated section": {truncatedEngine, truncatedID, getInventorySlot, "", 0, 0,
			"inventory of character 3 does not fit into its slot"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := GetInventory(
				testCase.engine, testCase.saveSessionID, testCase.characterID,
				testCase.section, testCase.page, testCase.pageSize)
			if err == nil {
				t.Fatalf("GetInventory accepted %s", strconv.Quote(name))
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, GetInventoryResult{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

func TestGetInventoryContractIsRawPhaseOne(t *testing.T) {
	if GetInventoryDefinition.SupportedResourceTypes != "—" {
		t.Errorf("supported resource types = %q, want the raw phase-1 contract without a resource type",
			GetInventoryDefinition.SupportedResourceTypes)
	}
	want := []string{"saveSessionID", "characterID", "containerSection", "page", "pageSize"}
	if !reflect.DeepEqual(GetInventoryDefinition.SupportedResourceVariables, want) {
		t.Errorf("supported resource variables = %v, want %v",
			GetInventoryDefinition.SupportedResourceVariables, want)
	}
}
