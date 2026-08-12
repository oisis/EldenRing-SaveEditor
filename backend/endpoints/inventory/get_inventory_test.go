package inventory

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
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
	getInventoryAnchorAt   = 0xA028
	getInventoryRecordSize = 12
	getInventoryCommonAt   = 505
	getInventoryKeyAt      = getInventoryCommonAt + 0xA80*getInventoryRecordSize + 4
	getInventorySectionEnd = getInventoryKeyAt + 0x180*getInventoryRecordSize + 8
)

func inventoryCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()
	inventoryCatalogOnce.Do(func() {
		inventoryCatalogData, inventoryCatalogErr = loader.LoadFS(catalogdata.Files())
	})
	if inventoryCatalogErr != nil {
		t.Fatalf("loader.LoadFS: %v", inventoryCatalogErr)
	}
	catalog, err := gamecatalog.New(inventoryCatalogData.Manifest, inventoryCatalogData.Resources())
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	return catalog
}

var (
	inventoryCatalogOnce sync.Once
	inventoryCatalogData loader.Data
	inventoryCatalogErr  error
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

func getInventoryWantCommon() []InventoryRecord {
	return []InventoryRecord{
		{Kind: "item", Key: "4000272E", GameID: 0x4000272E, ContainerSection: "common", PhysicalIndex: 1, GaItemHandle: 0xB000272E, Quantity: 3, AcquisitionIndex: 7},
		{Kind: "item", Key: "100704E0", GameID: 0x100704E0, ContainerSection: "common", PhysicalIndex: 6, GaItemHandle: 0x90001111, Quantity: 1, AcquisitionIndex: 9},
	}
}

func getInventoryWantKey() []InventoryRecord {
	return []InventoryRecord{
		{Kind: "item", Key: "8000EA60", GameID: 0x8000EA60, ContainerSection: "key", PhysicalIndex: 1, GaItemHandle: 0xC0000001, Quantity: 1, AcquisitionIndex: 12},
	}
}

func getInventoryWantAll() []InventoryRecord {
	return append(getInventoryWantCommon(), getInventoryWantKey()...)
}

// getInventoryIdentities keys the opaque identifier of every returned record by
// its physical coordinates, and proves on the way that each one is present and
// unique. A token is opaque by contract, so a test outside SaveEngine may not
// spell it out or parse it; the coordinates are the only handle it has on one.
func getInventoryIdentities(t *testing.T, records []InventoryRecord) map[string]string {
	t.Helper()

	byRow := make(map[string]string, len(records))
	owners := make(map[string]string, len(records))
	for _, record := range records {
		row := record.ContainerSection + "#" + strconv.Itoa(record.PhysicalIndex)
		if record.OwnedItemID == "" {
			t.Fatalf("record %s carries no ownedItemID", row)
		}
		if owner, taken := owners[record.OwnedItemID]; taken {
			t.Fatalf("record %s reuses the ownedItemID of %s", row, owner)
		}
		owners[record.OwnedItemID] = row
		byRow[row] = record.OwnedItemID
	}
	return byRow
}

// getInventoryWithIDs stamps identifiers taken from one canonical read onto the
// records another read is expected to return. Everything else in the expected
// value stays exact, so a filtered or paged read that reordered, re-numbered or
// re-identified a record still fails.
func getInventoryWithIDs(
	t *testing.T, want []InventoryRecord, identities map[string]string,
) []InventoryRecord {
	t.Helper()

	for index := range want {
		row := want[index].ContainerSection + "#" + strconv.Itoa(want[index].PhysicalIndex)
		id, known := identities[row]
		if !known {
			t.Fatalf("the canonical read never identified %s", row)
		}
		want[index].OwnedItemID = id
	}
	return want
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
	if anchorAt == getInventoryAnchorAt {
		binary.LittleEndian.PutUint32(data[slotBase:], 82)
		binary.LittleEndian.PutUint32(data[slotBase+0x20:], 0x90001111)
		binary.LittleEndian.PutUint32(data[slotBase+0x24:], 0x100704E0)
		binary.LittleEndian.PutUint32(data[slotBase+0x30:], 0xC0000001)
		binary.LittleEndian.PutUint32(data[slotBase+0x34:], 0x8000EA60)
	}

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
			gameCatalog := inventoryCatalog(t)

			result, err := GetInventory(engine, gameCatalog, sessionID, getInventorySlot, "", 0, 0)
			if err != nil {
				t.Fatalf("GetInventory: %v", err)
			}

			want := GetInventoryResult{
				SaveSessionID: sessionID,
				SaveRevision:  "0",
				CharacterID:   getInventorySlot,
				Active:        true,
				Records: getInventoryWithIDs(
					t, getInventoryWantAll(), getInventoryIdentities(t, result.Records)),
				Total:    3,
				Page:     1,
				PageSize: 50,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			// Nothing was committed in between, so the second read of the same
			// revision has to be identical down to every identifier.
			again, err := GetInventory(engine, gameCatalog, sessionID, getInventorySlot, "", 0, 0)
			if err != nil {
				t.Fatalf("second GetInventory: %v", err)
			}
			if !reflect.DeepEqual(again, result) {
				t.Errorf("a repeated read = %+v, want the identical first result %+v", again, result)
			}
		})
	}
}

func TestGetInventoryFiltersAndPages(t *testing.T) {
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	gameCatalog := inventoryCatalog(t)

	// The canonical unfiltered, unpaged read is what every case below is compared
	// against, so a section filter or a page that changed an identifier fails.
	canonical, err := GetInventory(engine, gameCatalog, sessionID, getInventorySlot, "", 0, 0)
	if err != nil {
		t.Fatalf("canonical GetInventory: %v", err)
	}
	identities := getInventoryIdentities(t, canonical.Records)
	all := getInventoryWithIDs(t, getInventoryWantAll(), identities)

	cases := map[string]struct {
		section        string
		page, pageSize int
		wantTotal      int
		wantPage       int
		wantPageSize   int
		want           []InventoryRecord
	}{
		"common only":  {"common", 0, 0, 2, 1, 50, getInventoryWithIDs(t, getInventoryWantCommon(), identities)},
		"key only":     {"key", 0, 0, 1, 1, 50, getInventoryWithIDs(t, getInventoryWantKey(), identities)},
		"first page":   {"", 1, 2, 3, 1, 2, all[:2]},
		"last page":    {"", 2, 2, 3, 2, 2, all[2:]},
		"beyond total": {"", 7, 2, 3, 7, 2, []InventoryRecord{}},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := GetInventory(
				engine, gameCatalog, sessionID, getInventorySlot, testCase.section, testCase.page, testCase.pageSize)
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
			if result.SaveRevision != canonical.SaveRevision {
				t.Errorf("saveRevision = %q, want %q", result.SaveRevision, canonical.SaveRevision)
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
	gameCatalog := inventoryCatalog(t)

	result, err := GetInventory(engine, gameCatalog, sessionID, getInventorySlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}

	want := GetInventoryResult{
		SaveSessionID: sessionID,
		SaveRevision:  "0",
		CharacterID:   getInventorySlot,
		Records:       []InventoryRecord{},
		Page:          1,
		PageSize:      50,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetInventoryRejectsInvalidRequests(t *testing.T) {
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	gameCatalog := inventoryCatalog(t)
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
				testCase.engine, gameCatalog, testCase.saveSessionID, testCase.characterID,
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

func TestGetInventoryRejectsAnUnknownCatalogItem(t *testing.T) {
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	prototype, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("gamecatalog.NewPrototype: %v", err)
	}

	result, err := GetInventory(engine, prototype, sessionID, getInventorySlot, "", 0, 0)
	if err == nil {
		t.Fatalf("GetInventory accepted an item absent from the prototype catalog: %+v", result)
	}
	if !reflect.DeepEqual(result, GetInventoryResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}

	result, err = GetInventory(engine, nil, sessionID, getInventorySlot, "", 0, 0)
	if err == nil || err.Error() != "game catalog is not available" {
		t.Errorf("nil catalog error = %v, want game catalog is not available", err)
	}
	if !reflect.DeepEqual(result, GetInventoryResult{}) {
		t.Errorf("nil catalog result = %+v, want the zero value", result)
	}
}

func TestGetInventoryContractResolvesItemDocuments(t *testing.T) {
	if GetInventoryDefinition.SupportedResourceTypes != "ItemDocument" {
		t.Errorf("supported resource types = %q, want ItemDocument",
			GetInventoryDefinition.SupportedResourceTypes)
	}
	want := []string{"saveSessionID", "characterID", "containerSection", "page", "pageSize"}
	if !reflect.DeepEqual(GetInventoryDefinition.SupportedResourceVariables, want) {
		t.Errorf("supported resource variables = %v, want %v",
			GetInventoryDefinition.SupportedResourceVariables, want)
	}
}
