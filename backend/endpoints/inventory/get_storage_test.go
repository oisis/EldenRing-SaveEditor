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
	getStorageHeaderSize       = 0x300
	getStorageEntryCountOffset = 0x0C
	getStorageEntryCount       = 12
	getStorageSlotBlockSize    = 0x280010
	getStorageSlotDataSize     = 0x280000
	getStorageFixtureSize      = int64(getStorageHeaderSize) + 10*getStorageSlotBlockSize + 0x60010

	getStorageUserData10Offset = int64(getStorageHeaderSize) + 10*getStorageSlotBlockSize + 0x10
	getStorageFlagsOffset      = 0x1954

	getStoragePS4HeaderSize      = 0x70
	getStoragePS4SlotSize        = 0x280000
	getStoragePS4FixtureSize     = int64(getStoragePS4HeaderSize) + 10*getStoragePS4SlotSize + 0x60000
	getStoragePS4UserData10Start = int64(getStoragePS4HeaderSize) + 10*getStoragePS4SlotSize
	getStoragePS4EntryTable      = 0x10
	getStoragePS4FirstEntry      = 7
	getStoragePS4EntryMarker     = 0x7F7F7F7F

	getStorageSlot     = 3
	getStorageAnchorAt = 0xA028

	// Distance from the anchor to the acquired-projectile count, the stride of
	// one projectile record and the three fixed blocks between the projectile
	// records and the Storage Box, restated literally.
	getStorageProjectileCountAt = 0xD0 + 0x58 + 0x1C + 0x58 + 0x58 + 0x9011 + 0x74 + 0x8C + 0x18
	getStorageProjectileStride  = 8
	getStorageBlocksBefore      = 0x9C + 0x0C + 0x12F

	// The section itself: the four-byte non-empty count, the 0x780 common
	// records, the four-byte key count, the 0x80 key records and the two
	// trailing counters.
	getStorageRecordSize  = 12
	getStorageCommonAt    = 4
	getStorageKeyAt       = getStorageCommonAt + 0x780*getStorageRecordSize + 4
	getStorageSectionSize = getStorageKeyAt + 0x80*getStorageRecordSize + 8

	// getStorageProjectiles is the number of acquired projectiles the fixture
	// declares, so the section never sits at a constant distance from the anchor.
	getStorageProjectiles = 11
)

// getStorageAnchor is the 65-byte anchor the storage chain is measured from,
// restated here independently of the implementation: one leading 0x00 byte, then
// four full repetitions of a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed
// by twelve 0x00 bytes.
var getStorageAnchor = []byte{
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

// getStorageRow is one raw record written into a synthetic section, with the
// stored quantity exactly as the game keeps it, including the high bit.
type getStorageRow struct {
	index       int
	handle      uint32
	rawQuantity uint32
	acquisition uint32
}

// The two sections deliberately mix occupied rows with both native absent
// sentinels and leave gaps between them, so a reader that renumbers rows, drops
// the wrong sentinel or reorders the sections cannot pass.
func getStorageCommonRows() []getStorageRow {
	return []getStorageRow{
		{index: 0, handle: 0x00000000, rawQuantity: 6, acquisition: 1},
		{index: 1, handle: 0xB000272E, rawQuantity: 0x80000003, acquisition: 7},
		{index: 4, handle: 0xFFFFFFFF, rawQuantity: 8, acquisition: 2},
		{index: 6, handle: 0x90001111, rawQuantity: 1, acquisition: 9},
	}
}

func getStorageKeyRows() []getStorageRow {
	return []getStorageRow{
		{index: 1, handle: 0xC0000001, rawQuantity: 0x80000001, acquisition: 12},
	}
}

func getStorageWantCommon() []StorageRecord {
	return []StorageRecord{
		{Kind: "item", Key: "4000272E", GameID: 0x4000272E, ContainerSection: "common", PhysicalIndex: 1, GaItemHandle: 0xB000272E, Quantity: 3, AcquisitionIndex: 7},
		{Kind: "item", Key: "100704E0", GameID: 0x100704E0, ContainerSection: "common", PhysicalIndex: 6, GaItemHandle: 0x90001111, Quantity: 1, AcquisitionIndex: 9},
	}
}

func getStorageWantKey() []StorageRecord {
	return []StorageRecord{
		{Kind: "item", Key: "8000EA60", GameID: 0x8000EA60, ContainerSection: "key", PhysicalIndex: 1, GaItemHandle: 0xC0000001, Quantity: 1, AcquisitionIndex: 12},
	}
}

func getStorageWantAll() []StorageRecord {
	return append(getStorageWantCommon(), getStorageWantKey()...)
}

// getStorageIdentities keys the opaque identifier of every returned record by
// its physical coordinates, and proves on the way that each one is present and
// unique. A token is opaque by contract, so a test outside SaveEngine may not
// spell it out or parse it; the coordinates are the only handle it has on one.
func getStorageIdentities(t *testing.T, records []StorageRecord) map[string]string {
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

// getStorageWithIDs stamps identifiers taken from one canonical read onto the
// records another read is expected to return. Everything else in the expected
// value stays exact, so a filtered or paged read that reordered, re-numbered or
// re-identified a record still fails.
func getStorageWithIDs(
	t *testing.T, want []StorageRecord, identities map[string]string,
) []StorageRecord {
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

// writeGetStorageFixture writes a minimal synthetic save of the given platform
// into t.TempDir() with one active character carrying the raw records above, and
// returns its path. anchorAt places the anchor inside the slot data, so a
// position that leaves no room for the whole section expresses the truncated
// case.
func writeGetStorageFixture(t *testing.T, platform string, active bool, anchorAt int64) string {
	t.Helper()

	var data []byte
	var slotBase int64
	switch platform {
	case "ps4":
		data = make([]byte, getStoragePS4FixtureSize)
		data[0], data[1], data[2], data[3] = 0xCB, 0x01, 0x9C, 0x2C
		for entry := 0; entry < getStorageEntryCount; entry++ {
			at := getStoragePS4EntryTable + entry*8
			binary.LittleEndian.PutUint32(data[at:], uint32(getStoragePS4FirstEntry+entry))
			binary.LittleEndian.PutUint32(data[at+4:], getStoragePS4EntryMarker)
		}
		if active {
			data[getStoragePS4UserData10Start+getStorageFlagsOffset+getStorageSlot] = 1
		}
		slotBase = int64(getStoragePS4HeaderSize) + getStorageSlot*getStoragePS4SlotSize
	default:
		data = make([]byte, getStorageFixtureSize)
		copy(data, []byte("BND4"))
		binary.LittleEndian.PutUint32(data[getStorageEntryCountOffset:], getStorageEntryCount)
		if active {
			data[getStorageUserData10Offset+getStorageFlagsOffset+getStorageSlot] = 1
		}
		slotBase = int64(getStorageHeaderSize) + 0x10 + getStorageSlot*getStorageSlotBlockSize
	}

	copy(data[slotBase+anchorAt:], getStorageAnchor)
	if anchorAt == getStorageAnchorAt {
		binary.LittleEndian.PutUint32(data[slotBase:], 82)
		binary.LittleEndian.PutUint32(data[slotBase+0x20:], 0x90001111)
		binary.LittleEndian.PutUint32(data[slotBase+0x24:], 0x100704E0)
		binary.LittleEndian.PutUint32(data[slotBase+0x30:], 0xC0000001)
		binary.LittleEndian.PutUint32(data[slotBase+0x34:], 0x8000EA60)
	}

	countAt := anchorAt + getStorageProjectileCountAt
	if countAt+4 <= getStorageSlotDataSize {
		binary.LittleEndian.PutUint32(data[slotBase+countAt:], getStorageProjectiles)
	}
	sectionAt := countAt + 4 + getStorageProjectiles*getStorageProjectileStride + getStorageBlocksBefore

	putRow := func(sectionOffset int64, row getStorageRow) {
		at := sectionAt + sectionOffset + int64(row.index)*getStorageRecordSize
		if at+getStorageRecordSize > getStorageSlotDataSize {
			return
		}
		binary.LittleEndian.PutUint32(data[slotBase+at:], row.handle)
		binary.LittleEndian.PutUint32(data[slotBase+at+4:], row.rawQuantity)
		binary.LittleEndian.PutUint32(data[slotBase+at+8:], row.acquisition)
	}
	for _, row := range getStorageCommonRows() {
		putRow(getStorageCommonAt, row)
	}
	for _, row := range getStorageKeyRows() {
		putRow(getStorageKeyAt, row)
	}

	path := filepath.Join(t.TempDir(), "get-storage.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// loadGetStorageSession loads one synthetic save and returns the engine and the
// session identifier the endpoint is called with.
func loadGetStorageSession(t *testing.T, platform string, active bool, anchorAt int64) (*saveengine.Engine, string) {
	t.Helper()

	engine := saveengine.New()
	session, err := engine.LoadSave(writeGetStorageFixture(t, platform, active, anchorAt), platform, "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, session.SaveSessionID
}

func TestGetStorageReturnsTheRawRecordsOfBothPlatforms(t *testing.T) {
	for _, platform := range []string{"pc", "ps4"} {
		t.Run(platform, func(t *testing.T) {
			engine, sessionID := loadGetStorageSession(t, platform, true, getStorageAnchorAt)
			gameCatalog := inventoryCatalog(t)

			result, err := GetStorage(engine, gameCatalog, sessionID, getStorageSlot, "", 0, 0)
			if err != nil {
				t.Fatalf("GetStorage: %v", err)
			}

			want := GetStorageResult{
				SaveSessionID: sessionID,
				SaveRevision:  "0",
				CharacterID:   getStorageSlot,
				Active:        true,
				Records: getStorageWithIDs(
					t, getStorageWantAll(), getStorageIdentities(t, result.Records)),
				Total:    3,
				Page:     1,
				PageSize: 50,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			// Nothing was committed in between, so the second read of the same
			// revision has to be identical down to every identifier.
			again, err := GetStorage(engine, gameCatalog, sessionID, getStorageSlot, "", 0, 0)
			if err != nil {
				t.Fatalf("second GetStorage: %v", err)
			}
			if !reflect.DeepEqual(again, result) {
				t.Errorf("a repeated read = %+v, want the identical first result %+v", again, result)
			}
		})
	}
}

// The endpoint must delegate: its result has to be the SaveEngine result itself,
// not a reshaped, re-ordered or re-paged copy of it.
func TestGetStorageResolvesSaveEngineRecords(t *testing.T) {
	engine, sessionID := loadGetStorageSession(t, "pc", true, getStorageAnchorAt)
	gameCatalog := inventoryCatalog(t)

	want, err := engine.GetStorage(sessionID, getStorageSlot, "key", 1, 1)
	if err != nil {
		t.Fatalf("engine.GetStorage: %v", err)
	}
	result, err := GetStorage(engine, gameCatalog, sessionID, getStorageSlot, "key", 1, 1)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	if len(result.Records) != 1 || result.Records[0].OwnedItemID != want.Records[0].OwnedItemID ||
		result.Records[0].GaItemHandle != want.Records[0].GaItemHandle {
		t.Errorf("endpoint record = %+v, want SaveEngine row %+v", result.Records, want.Records)
	}
}

func TestGetStorageFiltersAndPages(t *testing.T) {
	engine, sessionID := loadGetStorageSession(t, "pc", true, getStorageAnchorAt)
	gameCatalog := inventoryCatalog(t)

	// The canonical unfiltered, unpaged read is what every case below is compared
	// against, so a section filter or a page that changed an identifier fails.
	canonical, err := GetStorage(engine, gameCatalog, sessionID, getStorageSlot, "", 0, 0)
	if err != nil {
		t.Fatalf("canonical GetStorage: %v", err)
	}
	identities := getStorageIdentities(t, canonical.Records)
	all := getStorageWithIDs(t, getStorageWantAll(), identities)

	cases := map[string]struct {
		section        string
		page, pageSize int
		wantTotal      int
		wantPage       int
		wantPageSize   int
		want           []StorageRecord
	}{
		"common only":  {"common", 0, 0, 2, 1, 50, getStorageWithIDs(t, getStorageWantCommon(), identities)},
		"key only":     {"key", 0, 0, 1, 1, 50, getStorageWithIDs(t, getStorageWantKey(), identities)},
		"both":         {"", 0, 0, 3, 1, 50, all},
		"first page":   {"", 1, 2, 3, 1, 2, all[:2]},
		"last page":    {"", 2, 2, 3, 2, 2, all[2:]},
		"beyond total": {"", 7, 2, 3, 7, 2, []StorageRecord{}},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := GetStorage(
				engine, gameCatalog, sessionID, getStorageSlot, testCase.section, testCase.page, testCase.pageSize)
			if err != nil {
				t.Fatalf("GetStorage: %v", err)
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

func TestGetStorageReportsAResidualSlotAsInactive(t *testing.T) {
	engine, sessionID := loadGetStorageSession(t, "pc", false, getStorageAnchorAt)
	gameCatalog := inventoryCatalog(t)

	result, err := GetStorage(engine, gameCatalog, sessionID, getStorageSlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}

	want := GetStorageResult{
		SaveSessionID: sessionID,
		SaveRevision:  "0",
		CharacterID:   getStorageSlot,
		Records:       []StorageRecord{},
		Page:          1,
		PageSize:      50,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetStorageRejectsInvalidRequests(t *testing.T) {
	engine, sessionID := loadGetStorageSession(t, "pc", true, getStorageAnchorAt)
	gameCatalog := inventoryCatalog(t)
	// The anchor sits so close to the end of the slot that the section no longer
	// fits inside the slot data.
	truncatedEngine, truncatedID := loadGetStorageSession(t, "pc", true,
		getStorageSlotDataSize-
			(getStorageProjectileCountAt+4+getStorageProjectiles*getStorageProjectileStride+
				getStorageBlocksBefore+getStorageSectionSize-1))

	closedEngine, closedID := loadGetStorageSession(t, "pc", true, getStorageAnchorAt)
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
		"nil engine":      {nil, sessionID, getStorageSlot, "", 0, 0, "save engine is not available"},
		"empty session":   {engine, "", getStorageSlot, "", 0, 0, "saveSessionID is required"},
		"unknown session": {engine, "missing", getStorageSlot, "", 0, 0, `unknown save session "missing"`},
		"closed session": {closedEngine, closedID, getStorageSlot, "", 0, 0,
			`unknown save session ` + strconv.Quote(closedID)},
		"characterID -1": {engine, sessionID, -1, "", 0, 0, "characterID -1 is outside the range 0..9"},
		"characterID 10": {engine, sessionID, 10, "", 0, 0, "characterID 10 is outside the range 0..9"},
		"unknown section": {engine, sessionID, getStorageSlot, "storage", 0, 0,
			`containerSection must be "common", "key" or empty; got "storage"`},
		"case-shifted section": {engine, sessionID, getStorageSlot, "Key", 0, 0,
			`containerSection must be "common", "key" or empty; got "Key"`},
		"negative page":     {engine, sessionID, getStorageSlot, "", -2, 0, "page must not be negative; got -2"},
		"negative pageSize": {engine, sessionID, getStorageSlot, "", 0, -1, "pageSize must not be negative; got -1"},
		"truncated section": {truncatedEngine, truncatedID, getStorageSlot, "", 0, 0,
			"storage of character 3 does not fit into its slot"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := GetStorage(
				testCase.engine, gameCatalog, testCase.saveSessionID, testCase.characterID,
				testCase.section, testCase.page, testCase.pageSize)
			if err == nil {
				t.Fatalf("GetStorage accepted %s", strconv.Quote(name))
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, GetStorageResult{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

func TestGetStorageRejectsANilCatalog(t *testing.T) {
	engine, sessionID := loadGetStorageSession(t, "pc", true, getStorageAnchorAt)

	result, err := GetStorage(engine, nil, sessionID, getStorageSlot, "", 0, 0)
	if err == nil || err.Error() != "game catalog is not available" {
		t.Errorf("nil catalog error = %v, want game catalog is not available", err)
	}
	if !reflect.DeepEqual(result, GetStorageResult{}) {
		t.Errorf("nil catalog result = %+v, want the zero value", result)
	}
}

// A save whose slot carries no storage anchor at all is a hard error, not an
// empty result: the getter never falls back to a guessed position.
func TestGetStorageRejectsASlotWithoutTheStorageMarker(t *testing.T) {
	engine := saveengine.New()
	data := make([]byte, getStorageFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[getStorageEntryCountOffset:], getStorageEntryCount)
	data[getStorageUserData10Offset+getStorageFlagsOffset+getStorageSlot] = 1

	path := filepath.Join(t.TempDir(), "no-marker.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	session, err := engine.LoadSave(path, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := GetStorage(engine, inventoryCatalog(t), session.SaveSessionID, getStorageSlot, "", 0, 0)
	if err == nil {
		t.Fatalf("GetStorage accepted a slot without a storage marker: %+v", result)
	}
	if err.Error() != "character 3 carries no storage anchor" {
		t.Errorf("error = %q, want the missing-anchor error", err)
	}
}

func TestGetStorageContractResolvesItemDocuments(t *testing.T) {
	if GetStorageDefinition.SupportedResourceTypes != "ItemDocument" {
		t.Errorf("supported resource types = %q, want ItemDocument",
			GetStorageDefinition.SupportedResourceTypes)
	}
	want := []string{"saveSessionID", "characterID", "containerSection", "page", "pageSize"}
	if !reflect.DeepEqual(GetStorageDefinition.SupportedResourceVariables, want) {
		t.Errorf("supported resource variables = %v, want %v",
			GetStorageDefinition.SupportedResourceVariables, want)
	}
}
