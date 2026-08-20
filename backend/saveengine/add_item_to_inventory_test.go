package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Synthetic slot layout used only by this test and by gaitem_data_test.go. Every
// offset is restated literally instead of reused from the implementation, so a
// changed base, stride, section distance or block size fails here.
//
// One fixture carries everything this mutation touches at once, because the
// mutation itself touches all of it: the GaItem table in front of the anchor,
// both InventoryHeld sections with their count header and their two trailing
// allocators, the Storage Box behind the declared projectile list, and the
// GaItemGameData block behind the menu profile. It is composed from the shared
// container primitives; no existing fixture builder is changed.
const (
	addItemTestPCSlotDataBase  = 0x300 + 0x10 // first PC slot data, behind its MD5 prefix
	addItemTestPCSlotStride    = 0x280010
	addItemTestPS4SlotDataBase = 0x70 // first PS4 slot data, no MD5 prefix
	addItemTestPS4SlotStride   = 0x280000
	addItemTestSlotDataSize    = 0x280000

	// The anchor sits behind a full GaItem table: the table starts at offset 0x20
	// of the slot data and ends immediately before the anchor, and 5120 all-zero
	// records are eight bytes each.
	addItemTestAnchorAt = 0x20 + 5120*8

	// The slot version is above the legacy record-count break, so the table is
	// read with the current record count.
	addItemTestSlotVersion = 83

	// Distances from the anchor inside InventoryHeld: the common records start
	// behind the fixed structures and the four-byte common-item count, the key
	// records behind the 0xA80 common records and the four-byte key-item count,
	// and the two trailing counters behind the 0x180 key records.
	addItemTestRecordSize    = 12
	addItemTestCommonAt      = 505
	addItemTestCommonCountAt = addItemTestCommonAt - 4
	addItemTestCommonRecords = 0xA80
	addItemTestCommonSize    = addItemTestCommonRecords * addItemTestRecordSize
	addItemTestKeyAt         = addItemTestCommonAt + addItemTestCommonSize + 4
	addItemTestKeySize       = 0x180 * addItemTestRecordSize
	addItemTestNextEquipAt   = addItemTestKeyAt + addItemTestKeySize
	addItemTestNextAcqAt     = addItemTestNextEquipAt + 4

	// The chain from the anchor to the Storage Box: the fixed structures, the
	// declared projectile count this fixture leaves at zero, and the equipped
	// armaments, EquipPhysicsData and face data behind it.
	addItemTestProjectileCountAt = 0xD0 + 0x58 + 0x1C + 0x58 + 0x58 + 0x9011 + 0x74 + 0x8C + 0x18
	addItemTestBlocksBeforeStore = 0x9C + 0x0C + 0x12F
	addItemTestStorageAt         = addItemTestProjectileCountAt + 4 + addItemTestBlocksBeforeStore
	addItemTestStorageCommonAt   = 4
	addItemTestStorageKeyCountAt = addItemTestStorageCommonAt + 0x780*addItemTestRecordSize
	addItemTestStorageKeyAt      = addItemTestStorageKeyCountAt + 4
	addItemTestStorageSize       = 4 + 0x780*12 + 4 + 0x80*12 + 8

	// The chain onwards to GaItemGameData: GestureGameData, the declared region
	// count this fixture leaves at zero, Torrent with its control byte, the blood
	// stain with its padding, the menu profile header with the empty payload this
	// fixture declares, and TrophyEquipData.
	addItemTestGestureSize    = 0x100
	addItemTestHorseSize      = 0x28 + 1
	addItemTestBloodStainSize = 0x44 + 8
	addItemTestDynamicHeader  = 2 + 2 + 4
	addItemTestTrophySize     = 0x34
	addItemTestGaItemDataAt   = addItemTestStorageAt + addItemTestStorageSize +
		addItemTestGestureSize + 4 + addItemTestHorseSize + addItemTestBloodStainSize +
		addItemTestDynamicHeader + addItemTestTrophySize

	// GaItemGameData itself: the four-byte count of distinct active entries, four
	// bytes this reader does not interpret, an active prefix of eight-byte
	// entries and, behind its 7000-entry capacity, the second half of the
	// preallocated 7000 sixteen-byte area.
	addItemTestGaItemDataArrayAt = addItemTestGaItemDataAt + 8
	addItemTestGaItemEntrySize   = 8
	addItemTestGaItemMaxCount    = 7000
	addItemTestGaItemDataSize    = 8 + 7000*16

	// The item both platforms add: a goods resource whose handle is derived from
	// its game ID, so the GaItem table needs no record for it.
	addItemTestGoodsID     = 0x400006A4
	addItemTestGoodsHandle = 0xB00006A4

	// A second goods resource, used wherever a neighbouring record has to stay
	// untouched or a container has to be partly occupied.
	addItemTestOtherID     = 0x40000456
	addItemTestOtherHandle = 0xB0000456

	// A talisman, whose every copy is its own physical record.
	addItemTestTalismanID     = 0x20000064
	addItemTestTalismanHandle = 0xA0000064

	// addItemTestFillHandleBase is where the handles of a completely occupied
	// common section start.
	addItemTestFillHandleBase = 0xB0800000
)

// addItemTestAnchor is the 65-byte anchor every section of this fixture is
// measured from, restated here independently of the implementation so a changed
// production pattern fails this test.
var addItemTestAnchor = []byte{
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

// addItemTestRow is one raw twelve-byte record written into a synthetic section,
// with the stored quantity exactly as the game keeps it, including the high bit.
type addItemTestRow struct {
	index       int
	handle      uint32
	rawQuantity uint32
	acquisition uint32
}

// addItemTestFixture describes the synthetic slot content one test save is built
// from. Everything left at its zero value is the native empty state of that
// field: an empty record, a zero count and a zero allocator.
type addItemTestFixture struct {
	platform        Platform
	slot            int
	inactive        bool
	common          []addItemTestRow
	key             []addItemTestRow
	storage         []addItemTestRow
	storageKey      []addItemTestRow
	commonCount     uint32
	storageCount    uint32
	storageKeyCount uint32
	nextEquipIndex  uint32
	nextAcquisition uint32
	gaItemData      []uint32
	gaItemDataCount int // when non-zero, the declared count instead of len(gaItemData)
	fillCommon      bool
	tailMarker      bool
}

// addItemTestSlotBase reports where the data of the fixture's slot starts inside
// the container of its platform.
func addItemTestSlotBase(t *testing.T, platform Platform, slot int) int64 {
	t.Helper()

	switch platform {
	case PlatformPC:
		return addItemTestPCSlotDataBase + int64(slot)*addItemTestPCSlotStride
	case PlatformPS4:
		return addItemTestPS4SlotDataBase + int64(slot)*addItemTestPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", platform)
		return 0
	}
}

// writeAddItemFixture builds a synthetic save and returns its path inside
// t.TempDir(). It writes only the activity flag, the slot version, the anchor
// and the requested content; every other byte stays zeroed, which is the native
// empty state of each field this mutation reads.
func writeAddItemFixture(t *testing.T, content addItemTestFixture) string {
	t.Helper()

	var data []byte
	var userData10Base int64
	switch content.platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = pcUserData10DataOffset
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = ps4UserData10DataOffset
	default:
		t.Fatalf("unknown platform %q", content.platform)
	}
	slotBase := addItemTestSlotBase(t, content.platform, content.slot)

	if !content.inactive {
		data[userData10Base+userData10ActiveFlagsOffset+int64(content.slot)] = 1
	}
	binary.LittleEndian.PutUint32(data[slotBase:], addItemTestSlotVersion)
	copy(data[slotBase+addItemTestAnchorAt:], addItemTestAnchor)

	put := func(at int64, value uint32) {
		binary.LittleEndian.PutUint32(data[slotBase+addItemTestAnchorAt+at:], value)
	}
	putRow := func(at int64, row addItemTestRow) {
		record := at + int64(row.index)*addItemTestRecordSize
		put(record, row.handle)
		put(record+4, row.rawQuantity)
		put(record+8, row.acquisition)
	}

	for _, row := range content.common {
		putRow(addItemTestCommonAt, row)
	}
	if content.fillCommon {
		// The fill occupies a handle range of its own, far away from every item
		// the tests add, so a full section never accidentally becomes a top-up.
		for index := 0; index < addItemTestCommonRecords; index++ {
			putRow(addItemTestCommonAt, addItemTestRow{
				index:  index,
				handle: addItemTestFillHandleBase + uint32(index),
			})
		}
	}
	for _, row := range content.key {
		putRow(addItemTestKeyAt, row)
	}
	for _, row := range content.storage {
		putRow(addItemTestStorageAt+addItemTestStorageCommonAt, row)
	}
	for _, row := range content.storageKey {
		putRow(addItemTestStorageAt+addItemTestStorageKeyAt, row)
	}

	put(addItemTestCommonCountAt, content.commonCount)
	put(addItemTestNextEquipAt, content.nextEquipIndex)
	put(addItemTestNextAcqAt, content.nextAcquisition)
	put(addItemTestStorageAt, content.storageCount)
	put(addItemTestStorageAt+addItemTestStorageKeyCountAt, content.storageKeyCount)

	declared := uint32(len(content.gaItemData))
	if content.gaItemDataCount != 0 {
		declared = uint32(content.gaItemDataCount)
	}
	put(addItemTestGaItemDataAt, declared)
	for index, id := range content.gaItemData {
		entry := addItemTestGaItemDataArrayAt + int64(index)*addItemTestGaItemEntrySize
		put(entry, id)
		put(entry+4, 1)
	}
	if content.tailMarker {
		// The second half of the preallocated area and the very last byte of the
		// block carry a recognisable pattern, so a write that reaches past the
		// active capacity is visible as a changed range rather than as zeroes that
		// happen to match.
		for offset := int64(8 + addItemTestGaItemMaxCount*addItemTestGaItemEntrySize); offset+4 <=
			addItemTestGaItemDataSize; offset += 4 {
			put(addItemTestGaItemDataAt+offset, 0xA5A5A5A5)
		}
	}

	path := filepath.Join(t.TempDir(), "add-item.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// addItemTestSlotData copies the whole slot data of the session's snapshot, so a
// test can prove that a mutation changed exactly the ranges it claims and a
// rejected one changed nothing at all.
func addItemTestSlotData(
	t *testing.T, engine *Engine, saveSessionID string, platform Platform, slot int,
) []byte {
	t.Helper()

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		t.Fatalf("unknown save session %q", saveSessionID)
	}
	base, end := inventorySlotBounds(platform, slot)
	raw, err := loaded.snapshot.readAt(base, int(end-base))
	if err != nil {
		t.Fatalf("read slot data: %v", err)
	}
	return raw
}

// addItemTestSessionState reports the revision and the unsaved-changes flag of
// one session, so a rejected mutation can be proven not to have advanced either.
func addItemTestSessionState(t *testing.T, engine *Engine, saveSessionID string) (string, bool) {
	t.Helper()

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		t.Fatalf("unknown save session %q", saveSessionID)
	}
	return loaded.session.revisionString(), loaded.session.dirty
}

// addItemTestChangedRanges reports every maximal run of differing bytes as a
// half-open [start, end) pair, counted from the start of the slot data.
func addItemTestChangedRanges(t *testing.T, before, after []byte) [][2]int64 {
	t.Helper()

	if len(before) != len(after) {
		t.Fatalf("slot data changed length from %d to %d", len(before), len(after))
	}
	var ranges [][2]int64
	index := 0
	for index < len(before) {
		if before[index] == after[index] {
			index++
			continue
		}
		start := index
		for index < len(before) && before[index] != after[index] {
			index++
		}
		ranges = append(ranges, [2]int64{int64(start), int64(index)})
	}
	return ranges
}

// addItemTestAssertChanged proves that a mutation changed exactly the requested
// ranges. Every expected range is stated as an offset from the anchor, which is
// how the fixture states every position, and a changed run is required to lie
// completely inside one of them.
func addItemTestAssertChanged(t *testing.T, before, after []byte, expected [][2]int64) {
	t.Helper()

	changedRanges := addItemTestChangedRanges(t, before, after)
	observed := make([]bool, len(expected))
	for _, changed := range changedRanges {
		inside := false
		for index, allowed := range expected {
			from, to := addItemTestAnchorAt+allowed[0], addItemTestAnchorAt+allowed[1]
			if changed[0] >= from && changed[1] <= to {
				inside = true
				observed[index] = true
				break
			}
		}
		if !inside {
			t.Errorf("the mutation changed the unexpected range [0x%X, 0x%X) of the slot",
				changed[0], changed[1])
		}
	}
	for index, seen := range observed {
		if !seen {
			t.Errorf("the mutation did not change any byte of expected range [0x%X, 0x%X)",
				addItemTestAnchorAt+expected[index][0], addItemTestAnchorAt+expected[index][1])
		}
	}
}

// addItemTestUint32 reads one little-endian uint32 out of a slot-data copy, at a
// distance from the anchor.
func addItemTestUint32(slot []byte, at int64) uint32 {
	return binary.LittleEndian.Uint32(slot[addItemTestAnchorAt+at:])
}

func TestAddItemToInventoryCreatesARecordOnBothPlatforms(t *testing.T) {
	// The two containers carry identical slot content, so only the platform base
	// differs and a mutation that mixes the two bases cannot pass both cases.
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := addItemTestFixture{
				platform: platform, slot: 2, tailMarker: true,
				common:      []addItemTestRow{{index: 0, handle: addItemTestOtherHandle, rawQuantity: 4, acquisition: 11}},
				commonCount: 1, nextEquipIndex: 433, nextAcquisition: 968,
				gaItemData: []uint32{addItemTestOtherID},
			}
			engine := New()
			loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, content.slot)

			result, err := engine.AddItemToInventory(
				loaded.SaveSessionID, content.slot, addItemTestGoodsID, 5, "0", false, 40, 600)
			if err != nil {
				t.Fatalf("AddItemToInventory: %v", err)
			}

			want := AddItemToInventoryResult{
				SaveSessionID: loaded.SaveSessionID, SaveRevision: "1", CharacterID: content.slot,
				GameID: addItemTestGoodsID, Added: 5, Quantity: 5, CreatedRecord: true,
				ContainerSection: InventorySectionCommon, PhysicalIndex: 1,
			}
			if result != want {
				t.Errorf("AddItemToInventory = %+v, want %+v", result, want)
			}

			after := addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, content.slot)
			row := int64(addItemTestCommonAt + addItemTestRecordSize)
			addItemTestAssertChanged(t, before, after, [][2]int64{
				{row, row + addItemTestRecordSize},
				{addItemTestCommonCountAt, addItemTestCommonCountAt + 4},
				{addItemTestNextEquipAt, addItemTestNextEquipAt + 4},
				{addItemTestNextAcqAt, addItemTestNextAcqAt + 4},
				{addItemTestGaItemDataAt, addItemTestGaItemDataAt + 4},
				{addItemTestGaItemDataArrayAt, addItemTestGaItemDataArrayAt + 2*addItemTestGaItemEntrySize},
			})

			// The mark 968 is already even, so the new record takes 969 and the
			// allocator is left one past it.
			for _, field := range []struct {
				name string
				at   int64
				want uint32
			}{
				{"handle", row, addItemTestGoodsHandle},
				{"quantity", row + 4, 5},
				{"acquisition index", row + 8, 969},
				{"common item count", addItemTestCommonCountAt, 2},
				{"NextEquipIndex", addItemTestNextEquipAt, 434},
				{"NextAcquisitionSortId", addItemTestNextAcqAt, 970},
				{"GaItemData count", addItemTestGaItemDataAt, 2},
			} {
				if stored := addItemTestUint32(after, field.at); stored != field.want {
					t.Errorf("%s = %d, want %d", field.name, stored, field.want)
				}
			}
			// The ordinary entries stay ascending, so the new lower game ID is
			// placed in front of the one the fixture carried.
			if id := addItemTestUint32(after, addItemTestGaItemDataArrayAt); id != addItemTestOtherID {
				t.Errorf("first GaItemData entry = 0x%08X, want 0x%08X", id, addItemTestOtherID)
			}
			if id := addItemTestUint32(after,
				addItemTestGaItemDataArrayAt+addItemTestGaItemEntrySize); id != addItemTestGoodsID {
				t.Errorf("second GaItemData entry = 0x%08X, want 0x%08X", id, addItemTestGoodsID)
			}
		})
	}
}

func TestAddItemToInventorySurvivesAReload(t *testing.T) {
	content := addItemTestFixture{
		platform: PlatformPC, slot: 4,
		nextEquipIndex: 433, nextAcquisition: 968,
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if _, err := engine.AddItemToInventory(
		loaded.SaveSessionID, content.slot, addItemTestGoodsID, 7, "0", false, 40, 600); err != nil {
		t.Fatalf("AddItemToInventory: %v", err)
	}

	target := filepath.Join(t.TempDir(), "reloaded.sl2")
	if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
		t.Fatalf("WriteSave: %v", err)
	}

	fresh := New()
	again, err := fresh.LoadSave(target, string(PlatformPC))
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	inventory, err := fresh.GetInventory(again.SaveSessionID, content.slot, InventorySectionCommon, 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	if inventory.Total != 1 {
		t.Fatalf("the reloaded inventory holds %d records, want 1", inventory.Total)
	}
	record := inventory.Records[0]
	if record.PhysicalIndex != 0 || record.GaItemHandle != addItemTestGoodsHandle ||
		record.Quantity != 7 || record.AcquisitionIndex != 969 {
		t.Errorf("the reloaded record is %+v", record)
	}
	resolved, err := fresh.ResolveGaItemIDs(again.SaveSessionID, content.slot,
		[]uint32{record.GaItemHandle})
	if err != nil {
		t.Fatalf("ResolveGaItemIDs: %v", err)
	}
	if resolved[0] != addItemTestGoodsID {
		t.Errorf("the reloaded handle resolves to 0x%08X, want 0x%08X", resolved[0], addItemTestGoodsID)
	}
}

func TestAddItemToInventoryTopsUpExistingStack(t *testing.T) {
	content := addItemTestFixture{
		platform: PlatformPC, slot: 1, tailMarker: true,
		common: []addItemTestRow{
			{index: 0, handle: addItemTestOtherHandle, rawQuantity: 4, acquisition: 11},
			// The stored quantity carries the native high bit, which is not part of
			// the count and has to survive the write exactly as the game left it.
			{index: 2, handle: addItemTestGoodsHandle, rawQuantity: 0x80000000 | 3, acquisition: 13},
			{index: 5, handle: addItemTestTalismanHandle, rawQuantity: 1, acquisition: 17},
		},
		commonCount: 3, nextEquipIndex: 433, nextAcquisition: 968,
		gaItemData: []uint32{addItemTestGoodsID, addItemTestOtherID, addItemTestTalismanID},
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)

	result, err := engine.AddItemToInventory(
		loaded.SaveSessionID, content.slot, addItemTestGoodsID, 5, "0", false, 40, 600)
	if err != nil {
		t.Fatalf("AddItemToInventory: %v", err)
	}
	if !(result.Added == 5 && result.Quantity == 8 && !result.CreatedRecord &&
		result.PhysicalIndex == 2) {
		t.Errorf("AddItemToInventory = %+v, want a top-up of row 2 to 8", result)
	}

	// A top-up writes four bytes and nothing else: no counter, no header, no
	// GaItemData entry and no second stack of the same item.
	after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)
	quantityAt := int64(addItemTestCommonAt + 2*addItemTestRecordSize + 4)
	addItemTestAssertChanged(t, before, after, [][2]int64{{quantityAt, quantityAt + 4}})
	if stored := addItemTestUint32(after, quantityAt); stored != 0x80000000|8 {
		t.Errorf("the stored quantity is 0x%08X, want 0x%08X", stored, 0x80000000|8)
	}
}

func TestAddItemToInventoryRejectsATopUpAboveThePerRecordLimit(t *testing.T) {
	content := addItemTestFixture{
		platform: PlatformPC, slot: 1,
		common:      []addItemTestRow{{index: 0, handle: addItemTestGoodsHandle, rawQuantity: 38, acquisition: 11}},
		commonCount: 1, gaItemData: []uint32{addItemTestGoodsID},
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)

	// 38 + 5 exceeds the 40 one record holds. The request is rejected outright:
	// nothing is clamped to 40 and no second stack is opened for the remainder.
	if _, err := engine.AddItemToInventory(
		loaded.SaveSessionID, content.slot, addItemTestGoodsID, 5, "0", false, 40, 600); err == nil {
		t.Fatal("a top-up above the per-record limit was accepted")
	}
	after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)
	addItemTestAssertChanged(t, before, after, nil)
	if revision, dirty := addItemTestSessionState(t, engine, loaded.SaveSessionID); revision != "0" || dirty {
		t.Errorf("the rejected add left the session at revision %q, dirty %v", revision, dirty)
	}
}

func TestAddItemToInventoryUsesTheFirstFreeRow(t *testing.T) {
	// Row 1 carries the invalid sentinel and row 2 the empty one, so the first
	// free row is 1 and the occupied rows around it stay exactly as they are.
	content := addItemTestFixture{
		platform: PlatformPC, slot: 0,
		common: []addItemTestRow{
			{index: 0, handle: addItemTestOtherHandle, rawQuantity: 4, acquisition: 11},
			{index: 1, handle: 0xFFFFFFFF, rawQuantity: 0xDEADBEEF, acquisition: 0xDEADBEEF},
			{index: 3, handle: addItemTestOtherHandle + 1, rawQuantity: 9, acquisition: 21},
		},
		commonCount: 2, gaItemData: []uint32{addItemTestOtherID},
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)

	result, err := engine.AddItemToInventory(
		loaded.SaveSessionID, content.slot, addItemTestGoodsID, 1, "0", false, 40, 600)
	if err != nil {
		t.Fatalf("AddItemToInventory: %v", err)
	}
	if result.PhysicalIndex != 1 {
		t.Errorf("the new record landed on row %d, want 1", result.PhysicalIndex)
	}
	after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)
	row := int64(addItemTestCommonAt + addItemTestRecordSize)
	addItemTestAssertChanged(t, before, after, [][2]int64{
		{row, row + addItemTestRecordSize},
		{addItemTestCommonCountAt, addItemTestCommonCountAt + 4},
		{addItemTestNextEquipAt, addItemTestNextEquipAt + 4},
		{addItemTestNextAcqAt, addItemTestNextAcqAt + 4},
		{addItemTestGaItemDataAt, addItemTestGaItemDataAt + 4},
		{addItemTestGaItemDataArrayAt, addItemTestGaItemDataArrayAt + 2*addItemTestGaItemEntrySize},
	})
}

func TestAddItemToInventoryFillsTheLastFreeRecord(t *testing.T) {
	content := addItemTestFixture{platform: PlatformPC, slot: 3, fillCommon: true}
	// The very last common row is emptied again, so exactly one record is free.
	path := writeAddItemFixture(t, content)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	slotBase := addItemTestSlotBase(t, PlatformPC, content.slot)
	lastRow := slotBase + addItemTestAnchorAt + addItemTestCommonAt +
		(addItemTestCommonRecords-1)*addItemTestRecordSize
	copy(data[lastRow:lastRow+addItemTestRecordSize], make([]byte, addItemTestRecordSize))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}

	engine := New()
	loaded, err := engine.LoadSave(path, string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	result, err := engine.AddItemToInventory(
		loaded.SaveSessionID, content.slot, addItemTestGoodsID, 1, "0", false, 40, 600)
	if err != nil {
		t.Fatalf("AddItemToInventory: %v", err)
	}
	if result.PhysicalIndex != addItemTestCommonRecords-1 {
		t.Errorf("the last free record is row %d, want %d",
			result.PhysicalIndex, addItemTestCommonRecords-1)
	}

	// A second add now finds no free row at all and changes nothing.
	before := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)
	_, err = engine.AddItemToInventory(
		loaded.SaveSessionID, content.slot, 0x40000001, 1, "1", false, 40, 600)
	if err == nil || !strings.Contains(err.Error(), "no free record") {
		t.Fatalf("a full common section reported %v, want a no-free-record error", err)
	}
	after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)
	addItemTestAssertChanged(t, before, after, nil)
	if revision, _ := addItemTestSessionState(t, engine, loaded.SaveSessionID); revision != "1" {
		t.Errorf("the rejected add advanced the revision to %q", revision)
	}
}

func TestAddItemToInventoryTalismanAlwaysCreatesItsOwnRecord(t *testing.T) {
	content := addItemTestFixture{
		platform: PlatformPC, slot: 2, tailMarker: true,
		common: []addItemTestRow{
			{index: 0, handle: addItemTestTalismanHandle, rawQuantity: 1, acquisition: 11},
		},
		commonCount: 1, nextEquipIndex: 433, nextAcquisition: 968,
		gaItemData: []uint32{addItemTestTalismanID},
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)

	result, err := engine.AddItemToInventory(
		loaded.SaveSessionID, content.slot, addItemTestTalismanID, 1, "0", true, 1, 600)
	if err != nil {
		t.Fatalf("AddItemToInventory: %v", err)
	}
	if !result.CreatedRecord || result.PhysicalIndex != 1 || result.Quantity != 1 {
		t.Errorf("AddItemToInventory = %+v, want a new row 1 holding 1", result)
	}

	// The item is already owned physically, so the second copy adds no second
	// GaItemData entry and the whole block stays untouched.
	after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)
	row := int64(addItemTestCommonAt + addItemTestRecordSize)
	addItemTestAssertChanged(t, before, after, [][2]int64{
		{row, row + addItemTestRecordSize},
		{addItemTestCommonCountAt, addItemTestCommonCountAt + 4},
		{addItemTestNextEquipAt, addItemTestNextEquipAt + 4},
		{addItemTestNextAcqAt, addItemTestNextAcqAt + 4},
	})
	if count := addItemTestUint32(after, addItemTestGaItemDataAt); count != 1 {
		t.Errorf("the GaItemData count moved to %d, want 1", count)
	}
}

func TestAddItemToInventoryAddsNoGaItemDataEntryForAnItemHeldInStorage(t *testing.T) {
	content := addItemTestFixture{
		platform: PlatformPC, slot: 2, tailMarker: true,
		storage: []addItemTestRow{
			{index: 0, handle: addItemTestGoodsHandle, rawQuantity: 6, acquisition: 3},
		},
		storageCount: 1, gaItemData: []uint32{addItemTestGoodsID},
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)

	if _, err := engine.AddItemToInventory(
		loaded.SaveSessionID, content.slot, addItemTestGoodsID, 2, "0", false, 40, 600); err != nil {
		t.Fatalf("AddItemToInventory: %v", err)
	}
	after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)
	addItemTestAssertChanged(t, before, after, [][2]int64{
		{addItemTestCommonAt, addItemTestCommonAt + addItemTestRecordSize},
		{addItemTestCommonCountAt, addItemTestCommonCountAt + 4},
		{addItemTestNextEquipAt, addItemTestNextEquipAt + 4},
		{addItemTestNextAcqAt, addItemTestNextAcqAt + 4},
	})
	// The Storage Box itself is never written by this mutation.
	if stored := addItemTestUint32(after, addItemTestStorageAt); stored != 1 {
		t.Errorf("the storage count moved to %d, want 1", stored)
	}
}

func TestAddItemToInventoryRejectsAnItemHeldInStorageKey(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := addItemTestFixture{
				platform: platform, slot: 2, tailMarker: true,
				common: []addItemTestRow{
					{index: 0, handle: addItemTestGoodsHandle, rawQuantity: 3, acquisition: 1},
				},
				storageKey: []addItemTestRow{
					{index: 0, handle: addItemTestGoodsHandle, rawQuantity: 1, acquisition: 3},
				},
				commonCount: 1, storageKeyCount: 1, gaItemData: []uint32{addItemTestGoodsID},
			}
			engine := New()
			loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, content.slot)

			// The existing Inventory common row would normally take the top-up branch.
			// Storage key must still be preflighted first and reject the whole mutation.
			_, err = engine.AddItemToInventory(
				loaded.SaveSessionID, content.slot, addItemTestGoodsID, 1, "0", false, 40, 600)
			if err == nil || !strings.Contains(err.Error(), "storage key record") {
				t.Fatalf("an item held in Storage key reported %v, want a storage-key rejection", err)
			}
			after := addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, content.slot)
			addItemTestAssertChanged(t, before, after, nil)
			if revision, dirty := addItemTestSessionState(
				t, engine, loaded.SaveSessionID); revision != "0" || dirty {
				t.Errorf("the rejected add left the session at revision %q, dirty %v", revision, dirty)
			}
		})
	}
}

func TestAddItemToInventoryRejectsAndChangesNothing(t *testing.T) {
	base := func() addItemTestFixture {
		return addItemTestFixture{
			platform: PlatformPC, slot: 2, tailMarker: true,
			common: []addItemTestRow{
				{index: 0, handle: addItemTestOtherHandle, rawQuantity: 4, acquisition: 11},
			},
			commonCount: 1, nextEquipIndex: 433, nextAcquisition: 968,
			gaItemData: []uint32{addItemTestOtherID},
		}
	}
	withKeyRecord := base()
	withKeyRecord.key = []addItemTestRow{
		{index: 0, handle: addItemTestGoodsHandle, rawQuantity: 1, acquisition: 5},
	}
	fullGaItemData := base()
	fullGaItemData.gaItemDataCount = addItemTestGaItemMaxCount
	negativeGaItemData := base()
	negativeGaItemData.gaItemDataCount = -1
	fullCount := base()
	fullCount.commonCount = addItemTestCommonRecords
	exhaustedAllocator := base()
	// 0xFFFFFFFC is the last safe stored mark. The first value above it proves
	// that the public mutation rejects the exact overflow boundary without
	// changing the snapshot or session state.
	exhaustedAllocator.nextAcquisition = 0xFFFFFFFD

	cases := []struct {
		name              string
		content           addItemTestFixture
		characterID       int
		gameID            uint32
		quantity          uint32
		expectedRevision  string
		separateInstances bool
		maxPerRecord      uint32
		maxContainerTotal uint32
		wants             string
	}{
		{"quantity zero", base(), 2, addItemTestGoodsID, 0, "0", false, 40, 600,
			"quantity must be at least 1"},
		{"quantity above the record capacity", base(), 2, addItemTestGoodsID, 0x80000000, "0", false,
			0xFFFFFFFF, 0xFFFFFFFF, "exceeds the"},
		{"quantity above the per-record limit", base(), 2, addItemTestGoodsID, 41, "0", false, 40, 600,
			"exceeds the limit of 40 per record"},
		{"zero limits", base(), 2, addItemTestGoodsID, 1, "0", false, 0, 600,
			"must both be at least 1"},
		{"separate instances above one", base(), 2, addItemTestTalismanID, 2, "0", true, 1, 600,
			"quantity must be 1"},
		{"malformed expectedRevision", base(), 2, addItemTestGoodsID, 1, "00", false, 40, 600,
			"canonical decimal saveRevision"},
		{"stale expectedRevision", base(), 2, addItemTestGoodsID, 1, "7", false, 40, 600,
			"does not match the current saveRevision"},
		{"characterID below the range", base(), -1, addItemTestGoodsID, 1, "0", false, 40, 600,
			"outside the range"},
		{"characterID above the range", base(), 10, addItemTestGoodsID, 1, "0", false, 40, 600,
			"outside the range"},
		{"inactive slot", func() addItemTestFixture {
			content := base()
			content.inactive = true
			return content
		}(), 2, addItemTestGoodsID, 1, "0", false, 40, 600, "is not active"},
		{"weapon game ID", base(), 2, 0x00123456, 1, "0", false, 40, 600,
			"needs a record in the GaItem table"},
		{"armor game ID", base(), 2, 0x10123456, 1, "0", false, 40, 600,
			"needs a record in the GaItem table"},
		{"ash of war game ID", base(), 2, 0x80123456, 1, "0", false, 40, 600,
			"needs a record in the GaItem table"},
		{"container total above the limit", base(), 2, addItemTestGoodsID, 30, "0", false, 40, 20,
			"above the limit of 20"},
		{"item already held in the key section", withKeyRecord, 2, addItemTestGoodsID, 1, "0", false,
			40, 600, "already holds a key record"},
		{"GaItemData at capacity", fullGaItemData, 2, addItemTestGoodsID, 1, "0", false, 40, 600,
			"active GaItemData entries"},
		{"GaItemData count is negative", negativeGaItemData, 2, addItemTestGoodsID, 1, "0", false,
			40, 600, "active GaItemData entries"},
		{"common item count already full", fullCount, 2, addItemTestGoodsID, 1, "0", false, 40, 600,
			"receives no item"},
		{"acquisition allocator exhausted", exhaustedAllocator, 2, addItemTestGoodsID, 1, "0", false,
			40, 600, "cannot be advanced"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(
				writeAddItemFixture(t, testCase.content), string(testCase.content.platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := addItemTestSlotData(
				t, engine, loaded.SaveSessionID, testCase.content.platform, testCase.content.slot)

			_, err = engine.AddItemToInventory(
				loaded.SaveSessionID, testCase.characterID, testCase.gameID, testCase.quantity,
				testCase.expectedRevision, testCase.separateInstances,
				testCase.maxPerRecord, testCase.maxContainerTotal)
			if err == nil {
				t.Fatalf("the request was accepted, want a rejection mentioning %q", testCase.wants)
			}
			if !strings.Contains(err.Error(), testCase.wants) {
				t.Errorf("error %q does not mention %q", err, testCase.wants)
			}

			after := addItemTestSlotData(
				t, engine, loaded.SaveSessionID, testCase.content.platform, testCase.content.slot)
			addItemTestAssertChanged(t, before, after, nil)
			if revision, dirty := addItemTestSessionState(t, engine, loaded.SaveSessionID); revision != "0" || dirty {
				t.Errorf("the rejected add left the session at revision %q, dirty %v", revision, dirty)
			}
		})
	}
}

func TestAddItemToInventoryRejectsAnUnknownSession(t *testing.T) {
	engine := New()
	if _, err := engine.AddItemToInventory(
		"missing", 0, addItemTestGoodsID, 1, "0", false, 40, 600); err == nil {
		t.Fatal("an unknown session was accepted")
	}
}

func TestNextAcquisitionIndexStabilisesTheMark(t *testing.T) {
	// The mark never falls below the reserved equipment range, is stabilised to
	// an even value, and the new record takes the odd index one past it.
	for _, testCase := range []struct{ stored, want uint32 }{
		{0, 435}, {1, 435}, {433, 435}, {434, 435}, {435, 437}, {968, 969}, {969, 971},
		{0xFFFFFFFB, 0xFFFFFFFD}, {0xFFFFFFFC, 0xFFFFFFFD},
	} {
		got, err := nextAcquisitionIndex(testCase.stored, 0)
		if err != nil {
			t.Fatalf("nextAcquisitionIndex(%d): %v", testCase.stored, err)
		}
		if got != testCase.want {
			t.Errorf("nextAcquisitionIndex(%d) = %d, want %d", testCase.stored, got, testCase.want)
		}
	}
	for _, stored := range []uint32{0xFFFFFFFD, 0xFFFFFFFE, 0xFFFFFFFF} {
		if _, err := nextAcquisitionIndex(stored, 0); err == nil {
			t.Errorf("exhausted allocator 0x%08X was accepted", stored)
		}
	}
}

func TestAddItemToInventoryAcceptsTheLastSafeAcquisitionMark(t *testing.T) {
	content := addItemTestFixture{
		platform: PlatformPC, slot: 2, nextAcquisition: 0xFFFFFFFC,
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	if _, err := engine.AddItemToInventory(
		loaded.SaveSessionID, content.slot, addItemTestGoodsID, 1, "0", false, 40, 600); err != nil {
		t.Fatalf("AddItemToInventory: %v", err)
	}
	after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)
	if acquisition := addItemTestUint32(after, addItemTestCommonAt+8); acquisition != 0xFFFFFFFD {
		t.Errorf("acquisition index is 0x%08X, want 0xFFFFFFFD", acquisition)
	}
	if next := addItemTestUint32(after, addItemTestNextAcqAt); next != 0xFFFFFFFE {
		t.Errorf("NextAcquisitionSortId is 0x%08X, want 0xFFFFFFFE", next)
	}
}

func TestGaItemHandleForGameIDIsTheInverseOfResolution(t *testing.T) {
	empty := map[uint32]uint32{}
	for _, gameID := range []uint32{addItemTestGoodsID, addItemTestTalismanID, 0x40000000, 0x2000FFFF} {
		handle, err := gaItemHandleForGameID(gameID)
		if err != nil {
			t.Fatalf("gaItemHandleForGameID(0x%08X): %v", gameID, err)
		}
		resolved, err := resolveGaItemHandle(empty, handle)
		if err != nil {
			t.Fatalf("resolveGaItemHandle(0x%08X): %v", handle, err)
		}
		if resolved != gameID {
			t.Errorf("0x%08X became handle 0x%08X and resolved back to 0x%08X",
				gameID, handle, resolved)
		}
	}
	for _, gameID := range []uint32{0x00000001, 0x10000001, 0x80000001, 0x30000001} {
		if _, err := gaItemHandleForGameID(gameID); err == nil {
			t.Errorf("0x%08X received a handle it has no record for", gameID)
		}
	}
}

func TestAddItemToInventoryRejectsExistingDuplicateRecords(t *testing.T) {
	content := addItemTestFixture{
		platform: PlatformPC,
		slot:     2,
		common: []addItemTestRow{
			{index: 1, handle: addItemTestGoodsHandle, rawQuantity: 0x80000005, acquisition: 450},
			{index: 3, handle: addItemTestGoodsHandle, rawQuantity: 0x80000005, acquisition: 470},
		},
		commonCount: 2,
		gaItemData:  []uint32{addItemTestGoodsID},
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)

	_, err = engine.AddItemToInventory(
		loaded.SaveSessionID, content.slot, addItemTestGoodsID, 1, "0", false, 40, 600)
	if err == nil || !strings.Contains(err.Error(), "already holds 2 duplicate records in Inventory") {
		t.Fatalf("error = %v, want duplicate quantity_stack rejection", err)
	}
	if after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot); !bytes.Equal(after, before) {
		t.Error("a rejected duplicate add changed the slot")
	}
}
