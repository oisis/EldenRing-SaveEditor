package saveengine

import (
	"bytes"
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
	spellsPCSlotDataBase  = 0x300 + 0x10 // first PC slot data, behind its MD5 prefix
	spellsPCSlotStride    = 0x280010
	spellsPS4SlotDataBase = 0x70 // first PS4 slot data, no MD5 prefix
	spellsPS4SlotStride   = 0x280000
	spellsSlotDataSize    = 0x280000

	// Distances from the anchor, restated independently of the implementation.
	spellsSectionAt   = 0x9205 // start of EquippedSpells
	spellsCountAt     = 0x931D // the acquired-projectiles count
	spellsInventoryAt = 505    // first common inventory record
	spellsTalismansAt = -241   // the additional-talisman-slots byte

	spellsRecordSize        = 8
	spellsRecordCount       = 14
	spellsInventoryRecord   = 12
	spellsInventoryCommon   = 0xA80
	spellsInventoryKeyStart = spellsInventoryCommon*spellsInventoryRecord + 4
	spellsMemoryStoneHandle = 0xB000272E
	spellsMoonOfNokstella   = 0x20000474
	spellsFirstTalisman     = 17
)

// spellsTestAnchor is the 65-byte anchor the whole chain is measured from,
// restated here independently of the implementation so a changed production
// pattern fails this test: one leading 0x00 byte, then four full repetitions of
// a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes.
var spellsTestAnchor = []byte{
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

// spellsFixture describes the synthetic slot content one test save is built
// from: which slot carries which activity flag, where its anchor sits relative
// to the start of the slot data, which fourteen records the EquippedSpells
// section holds, how many Memory Stones the inventory carries and where, how
// many talisman fields are unlocked, and which talisman field holds Moon of
// Nokstella.
//
// records holds the raw pairs verbatim, so a malformed pair is expressed by
// writing one; occupiedIDs is the shorthand that builds the native pairs for a
// compact loadout.
type spellsFixture struct {
	platform Platform
	slot     int
	flag     byte
	anchorAt int64
	noAnchor bool

	occupiedIDs []uint32
	records     *[spellsRecordCount][2]uint32

	projectileCount uint32

	memoryStones      uint32
	memoryStonesInKey bool
	noMemoryStones    bool

	additionalTalismans byte
	moonAtField         int // -1 when Moon of Nokstella is not equipped at all
}

// writeSpellsFixture builds a synthetic save and returns its path inside
// t.TempDir(). Only the activity flag, the anchor, the spell records, the
// Memory Stone record, the talisman-slot byte, the projectile count and the
// talisman fields are written; the rest of the container stays zeroed. A range
// that would reach past the end of the slot data is left out, which is how the
// out-of-bounds cases are expressed.
func writeSpellsFixture(t *testing.T, content spellsFixture) string {
	t.Helper()

	var data []byte
	var userData10Base, slotBase int64
	switch content.platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = pcUserData10DataOffset
		slotBase = spellsPCSlotDataBase + int64(content.slot)*spellsPCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = ps4UserData10DataOffset
		slotBase = spellsPS4SlotDataBase + int64(content.slot)*spellsPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", content.platform)
	}

	data[userData10Base+userData10ActiveFlagsOffset+int64(content.slot)] = content.flag

	write := func(path string) string {
		t.Helper()
		full := filepath.Join(t.TempDir(), path)
		if err := os.WriteFile(full, data, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		return full
	}

	if content.noAnchor {
		return write("spells.sl2")
	}

	copy(data[slotBase+content.anchorAt:], spellsTestAnchor)

	putU32 := func(at int64, value uint32) {
		if at < 0 || at+4 > spellsSlotDataSize {
			return
		}
		binary.LittleEndian.PutUint32(data[slotBase+at:], value)
	}

	// The fourteen physical records.
	records := spellRecordsOf(content)
	for index, record := range records {
		at := content.anchorAt + spellsSectionAt + int64(index)*spellsRecordSize
		putU32(at, record[0])
		putU32(at+4, record[1])
	}

	// The Memory Stone stack. It is written into a record far inside its range,
	// so a reader that only looks at the first record fails.
	if !content.noMemoryStones {
		recordIndex := int64(97)
		at := content.anchorAt + spellsInventoryAt + recordIndex*spellsInventoryRecord
		if content.memoryStonesInKey {
			at = content.anchorAt + spellsInventoryAt + spellsInventoryKeyStart +
				recordIndex*spellsInventoryRecord
		}
		putU32(at, spellsMemoryStoneHandle)
		// The high bit belongs to the stored quantity, not to the count.
		putU32(at+4, content.memoryStones|0x80000000)
	}

	// The unlocked talisman fields and the talisman block behind the declared
	// acquired-projectile records.
	if at := content.anchorAt + spellsTalismansAt; at >= 0 {
		data[slotBase+at] = content.additionalTalismans
	}
	countAt := content.anchorAt + spellsCountAt
	putU32(countAt, content.projectileCount)
	if content.moonAtField >= 0 {
		blockAt := countAt + 4 + int64(content.projectileCount)*8
		putU32(blockAt+int64(spellsFirstTalisman+content.moonAtField)*4, spellsMoonOfNokstella)
	}

	return write("spells.sl2")
}

// spellRecordsOf turns a fixture into the fourteen raw pairs it stores: either
// the verbatim records it declares, or the native pairs of its compact loadout
// followed by the native empty pair.
func spellRecordsOf(content spellsFixture) [spellsRecordCount][2]uint32 {
	if content.records != nil {
		return *content.records
	}
	var records [spellsRecordCount][2]uint32
	for index := range records {
		if index < len(content.occupiedIDs) {
			records[index] = [2]uint32{content.occupiedIDs[index], 0xFFFFFFFF}
			continue
		}
		records[index] = [2]uint32{0xFFFFFFFF, 0x00000000}
	}
	return records
}

// wantSpellsOf is the raw identifier list a fixture must produce.
func wantSpellsOf(content spellsFixture) [spellsRecordCount]uint32 {
	records := spellRecordsOf(content)
	var want [spellsRecordCount]uint32
	for index, record := range records {
		want[index] = record[0]
	}
	return want
}

func TestGetEquippedSpellsReadsTheActiveSlotOfBothPlatforms(t *testing.T) {
	// The two fixtures put the anchor at different positions and declare a
	// different, non-zero number of projectile records, so neither a fixed offset
	// nor a fixed skip can pass both cases. Between them the records cover a full
	// fourteen-slot loadout, a partially filled one, both native pairs, a
	// duplicate spell and identifiers whose high bits are set.
	fourteen := make([]uint32, 0, spellsRecordCount)
	for index := 0; index < spellsRecordCount; index++ {
		fourteen = append(fourteen, uint32(0x1770+index))
	}

	cases := []struct {
		name    string
		content spellsFixture
		want    int
	}{
		{
			name: "pc full loadout",
			content: spellsFixture{
				platform: PlatformPC, slot: 0, flag: 1, anchorAt: 0x01A7,
				occupiedIDs: fourteen, projectileCount: 11,
				memoryStones: 3, additionalTalismans: 3, moonAtField: -1,
			},
			want: 5,
		},
		{
			name: "ps4 partial loadout with duplicates",
			content: spellsFixture{
				platform: PlatformPS4, slot: 7, flag: 1, anchorAt: 0x1F4C2,
				occupiedIDs:     []uint32{0x0FA0, 0x0FA0, 0x1770, 0x0FFFFFFF},
				projectileCount: 37,
				memoryStones:    2, additionalTalismans: 0, moonAtField: -1,
			},
			want: 4,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeSpellsFixture(t, testCase.content), string(testCase.content.platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.GetEquippedSpells(loaded.SaveSessionID, testCase.content.slot)
			if err != nil {
				t.Fatalf("GetEquippedSpells: %v", err)
			}

			want := CharacterEquippedSpells{
				SaveSessionID:        loaded.SaveSessionID,
				SaveRevision:         "0",
				CharacterID:          testCase.content.slot,
				Active:               true,
				Spells:               wantSpellsOf(testCase.content),
				AvailableMemorySlots: testCase.want,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}
		})
	}
}

// The empty sentinel is a stored value, not an absence: every one of the
// fourteen records keeps it.
func TestGetEquippedSpellsPreservesTheEmptySentinelOfEveryRecord(t *testing.T) {
	content := spellsFixture{
		platform: PlatformPC, slot: 1, flag: 1, anchorAt: 0x0640,
		projectileCount: 3, memoryStones: 0, moonAtField: -1,
	}

	engine := New()
	loaded, err := engine.LoadSave(writeSpellsFixture(t, content), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.GetEquippedSpells(loaded.SaveSessionID, content.slot)
	if err != nil {
		t.Fatalf("GetEquippedSpells: %v", err)
	}
	if len(result.Spells) != spellsRecordCount {
		t.Fatalf("record count = %d, want %d", len(result.Spells), spellsRecordCount)
	}
	for index, spell := range result.Spells {
		if spell != 0xFFFFFFFF {
			t.Errorf("record %d = 0x%08X, want the empty sentinel 0xFFFFFFFF", index, spell)
		}
	}
	// Two base slots and no Memory Stone at all is a valid, minimal capacity.
	if result.AvailableMemorySlots != 2 {
		t.Errorf("availableMemorySlots = %d, want 2", result.AvailableMemorySlots)
	}
}

func TestGetEquippedSpellsReportsAResidualSlotAsInactive(t *testing.T) {
	content := spellsFixture{
		platform: PlatformPC, slot: 4, flag: 0, anchorAt: 0x0800,
		occupiedIDs: []uint32{0x1770, 0x0FA0}, projectileCount: 5,
		memoryStones: 8, additionalTalismans: 3, moonAtField: 0,
	}

	engine := New()
	loaded, err := engine.LoadSave(writeSpellsFixture(t, content), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.GetEquippedSpells(loaded.SaveSessionID, content.slot)
	if err != nil {
		t.Fatalf("GetEquippedSpells: %v", err)
	}

	want := CharacterEquippedSpells{SaveSessionID: loaded.SaveSessionID, SaveRevision: "0", CharacterID: content.slot}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

// The capacity rule is base plus effective Memory Stones, capped by the game
// maximum, plus Moon of Nokstella only in an unlocked talisman field.
func TestGetEquippedSpellsComputesTheAvailableMemorySlots(t *testing.T) {
	cases := []struct {
		name    string
		content spellsFixture
		want    int
	}{
		{
			name: "no memory stone record at all",
			content: spellsFixture{
				noMemoryStones: true, additionalTalismans: 3, moonAtField: -1,
			},
			want: 2,
		},
		{
			name: "memory stones from the key items fall back",
			content: spellsFixture{
				memoryStones: 6, memoryStonesInKey: true, additionalTalismans: 3, moonAtField: -1,
			},
			want: 8,
		},
		{
			name: "eight memory stones reach the standard limit",
			content: spellsFixture{
				memoryStones: 8, additionalTalismans: 3, moonAtField: -1,
			},
			want: 10,
		},
		{
			name: "more memory stones than the game grants stay capped",
			content: spellsFixture{
				memoryStones: 40, additionalTalismans: 3, moonAtField: -1,
			},
			want: 10,
		},
		{
			name: "moon of nokstella in an unlocked field adds two",
			content: spellsFixture{
				memoryStones: 4, additionalTalismans: 3, moonAtField: 3,
			},
			want: 8,
		},
		{
			name: "moon of nokstella in a locked field adds nothing",
			content: spellsFixture{
				memoryStones: 4, additionalTalismans: 0, moonAtField: 3,
			},
			want: 6,
		},
		{
			name: "moon of nokstella in the only unlocked field adds two",
			content: spellsFixture{
				memoryStones: 4, additionalTalismans: 0, moonAtField: 0,
			},
			want: 8,
		},
		{
			name: "the maximum is never exceeded",
			content: spellsFixture{
				memoryStones: 8, additionalTalismans: 3, moonAtField: 2,
			},
			want: 12,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			content := testCase.content
			content.platform = PlatformPC
			content.slot = 2
			content.flag = 1
			content.anchorAt = 0x0640
			content.projectileCount = 7

			engine := New()
			loaded, err := engine.LoadSave(writeSpellsFixture(t, content), "", "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.GetEquippedSpells(loaded.SaveSessionID, content.slot)
			if err != nil {
				t.Fatalf("GetEquippedSpells: %v", err)
			}
			if result.AvailableMemorySlots != testCase.want {
				t.Errorf("availableMemorySlots = %d, want %d", result.AvailableMemorySlots, testCase.want)
			}
		})
	}
}

func TestGetEquippedSpellsRejectsInvalidRequests(t *testing.T) {
	engine := New()

	loadSlot := func(content spellsFixture) string {
		t.Helper()
		loaded, err := engine.LoadSave(writeSpellsFixture(t, content), "", "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		return loaded.SaveSessionID
	}

	valid := spellsFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x0640,
		occupiedIDs: []uint32{0x1770}, projectileCount: 11,
		memoryStones: 3, additionalTalismans: 1, moonAtField: -1,
	}
	present := loadSlot(valid)

	missingAnchor := loadSlot(spellsFixture{
		platform: PlatformPC, slot: 2, flag: 1, noAnchor: true, moonAtField: -1,
	})

	// An occupied identifier with the empty follower is a pair the game never
	// writes, and neither is the empty identifier with the occupied follower.
	occupiedRecords := spellRecordsOf(valid)
	occupiedRecords[3] = [2]uint32{0x1770, 0x00000000}
	malformedOccupied := valid
	malformedOccupied.records = &occupiedRecords
	malformedOccupiedID := loadSlot(malformedOccupied)

	emptyRecords := spellRecordsOf(valid)
	emptyRecords[13] = [2]uint32{0xFFFFFFFF, 0xFFFFFFFF}
	malformedEmpty := valid
	malformedEmpty.records = &emptyRecords
	malformedEmptyID := loadSlot(malformedEmpty)

	invalidCount := valid
	invalidCount.projectileCount = 200001
	invalidCountID := loadSlot(invalidCount)

	// The anchor sits so close to the end of the slot that the spell records no
	// longer fit behind it.
	truncated := valid
	truncated.anchorAt = spellsSlotDataSize - 0x9270
	truncatedID := loadSlot(truncated)

	// The anchor sits so close to the start of the slot that the talisman-slot
	// byte in front of it would lie in the previous slot.
	noRoomInFront := valid
	noRoomInFront.anchorAt = 0x10
	noRoomInFrontID := loadSlot(noRoomInFront)

	cases := map[string]struct {
		saveSessionID string
		characterID   int
		want          string
	}{
		"empty session":   {"", 0, "saveSessionID is required"},
		"unknown session": {"missing", 0, `unknown save session "missing"`},
		"characterID -1":  {present, -1, "characterID -1 is outside the range 0..9"},
		"characterID 10":  {present, 10, "characterID 10 is outside the range 0..9"},
		"missing anchor":  {missingAnchor, 2, "character 2 carries no equipped-spells anchor"},
		"occupied record with the empty follower": {malformedOccupiedID, 2,
			"spell record 3 of character 2 stores the pair (0x00001770, 0x00000000), which is neither empty nor occupied"},
		"empty record with the occupied follower": {malformedEmptyID, 2,
			"spell record 13 of character 2 stores the pair (0xFFFFFFFF, 0xFFFFFFFF), which is neither empty nor occupied"},
		"invalid projectile count": {invalidCountID, 2,
			"character 2 declares 200001 projectile records, want at most 200000"},
		"truncated section": {truncatedID, 2,
			"equipped spells of character 2 do not fit into its slot"},
		"no room in front of the anchor": {noRoomInFrontID, 2,
			"talisman slot count of character 2 lies outside its slot"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetEquippedSpells(testCase.saveSessionID, testCase.characterID)
			if err == nil {
				t.Fatalf("GetEquippedSpells accepted %s", strconv.Quote(name))
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, CharacterEquippedSpells{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

// The getter is a reader: neither the source file nor the private snapshot may
// change, and repeated calls must keep returning the same values.
func TestGetEquippedSpellsMutatesNothing(t *testing.T) {
	content := spellsFixture{
		platform: PlatformPS4, slot: 3, flag: 1, anchorAt: 0x2000,
		occupiedIDs: []uint32{0x1770, 0x0FA0}, projectileCount: 9,
		memoryStones: 5, additionalTalismans: 2, moonAtField: 1,
	}
	path := writeSpellsFixture(t, content)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	engine := New()
	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	first, err := engine.GetEquippedSpells(loaded.SaveSessionID, content.slot)
	if err != nil {
		t.Fatalf("GetEquippedSpells: %v", err)
	}
	second, err := engine.GetEquippedSpells(loaded.SaveSessionID, content.slot)
	if err != nil {
		t.Fatalf("GetEquippedSpells (second call): %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("second result = %+v, want the first one %+v", second, first)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("re-read fixture: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Error("the source save changed while it was only read")
	}
}

func TestSetEquippedSpellsHappyPath(t *testing.T) {
	cases := map[string]struct {
		platform Platform
		slot     int
	}{
		"PC":  {PlatformPC, 3},
		"PS4": {PlatformPS4, 1},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			content := spellsFixture{
				platform: tc.platform, slot: tc.slot, flag: 1, anchorAt: 0x2000,
				occupiedIDs: []uint32{0x1770}, projectileCount: 5,
				memoryStones: 3, additionalTalismans: 1, moonAtField: -1,
			}
			path := writeSpellsFixture(t, content)
			engine := New()
			loaded, err := engine.LoadSave(path, "", "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			newSpells := []uint32{0x0FA0, 0x1068}
			res, err := engine.SetEquippedSpells(loaded.SaveSessionID, tc.slot, newSpells, 4, "0")
			if err != nil {
				t.Fatalf("SetEquippedSpells: %v", err)
			}
			if res.SaveRevision != "1" {
				t.Errorf("revision = %q, want 1", res.SaveRevision)
			}
			if !reflect.DeepEqual(res.RawMagicParamIDs, newSpells) {
				t.Errorf("spells = %v, want %v", res.RawMagicParamIDs, newSpells)
			}
			if res.UsedMemorySlots != 4 || res.AvailableMemorySlots != 5 {
				t.Errorf("used/available = %d/%d, want 4/5", res.UsedMemorySlots, res.AvailableMemorySlots)
			}

			stored, err := engine.GetEquippedSpells(loaded.SaveSessionID, tc.slot)
			if err != nil {
				t.Fatalf("GetEquippedSpells after mutation: %v", err)
			}
			if stored.Spells[0] != 0x0FA0 || stored.Spells[1] != 0x1068 || stored.Spells[2] != 0xFFFFFFFF {
				t.Errorf("stored spells = %v, want [0x0FA0 0x1068 0xFFFFFFFF...]", stored.Spells[:3])
			}
		})
	}
}

func TestSetEquippedSpellsCapacityWithAndWithoutMoon(t *testing.T) {
	// 8 memory stones -> 2 + 8 = 10 capacity without Moon
	contentNoMoon := spellsFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x1500,
		memoryStones: 8, additionalTalismans: 3, moonAtField: -1,
	}
	pathNoMoon := writeSpellsFixture(t, contentNoMoon)
	engineNoMoon := New()
	loadedNoMoon, err := engineNoMoon.LoadSave(pathNoMoon, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// 11 slots cost > 10 available -> fail
	_, err = engineNoMoon.SetEquippedSpells(loadedNoMoon.SaveSessionID, 2, []uint32{0x1068, 0x1108}, 11, "0")
	if err == nil {
		t.Fatal("SetEquippedSpells accepted used slots exceeding capacity")
	}

	// With Moon of Nokstella equipped in unlocked field -> 2 + 8 + 2 = 12 capacity
	contentMoon := spellsFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x1500,
		memoryStones: 8, additionalTalismans: 3, moonAtField: 1,
	}
	pathMoon := writeSpellsFixture(t, contentMoon)
	engineMoon := New()
	loadedMoon, err := engineMoon.LoadSave(pathMoon, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	resMoon, err := engineMoon.SetEquippedSpells(loadedMoon.SaveSessionID, 2, []uint32{0x1068, 0x1108}, 11, "0")
	if err != nil {
		t.Fatalf("SetEquippedSpells with Moon failed: %v", err)
	}
	if resMoon.AvailableMemorySlots != 12 {
		t.Errorf("availableMemorySlots = %d, want 12", resMoon.AvailableMemorySlots)
	}
}

func TestSetEquippedSpellsRejectsNonEmptyTailPositions(t *testing.T) {
	var records [14][2]uint32
	for i := range records {
		records[i] = [2]uint32{0xFFFFFFFF, 0x00000000}
	}
	records[12] = [2]uint32{0x1770, 0xFFFFFFFF} // Non-empty position 13

	content := spellsFixture{
		platform: PlatformPC, slot: 0, flag: 1, anchorAt: 0x1000,
		records: &records,
	}
	path := writeSpellsFixture(t, content)
	engine := New()
	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	_, err = engine.SetEquippedSpells(loaded.SaveSessionID, 0, []uint32{0x0FA0}, 1, "0")
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("position 13 or 14")) {
		t.Fatalf("expected non-empty tail position error, got %v", err)
	}
}

func TestSetEquippedSpellsPreservesPositions13And14Bytes(t *testing.T) {
	content := spellsFixture{
		platform: PlatformPC, slot: 0, flag: 1, anchorAt: 0x1000,
		occupiedIDs: []uint32{0x1770},
	}
	path := writeSpellsFixture(t, content)
	engine := New()
	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	slotBase := int64(spellsPCSlotDataBase)
	anchorBase := slotBase + content.anchorAt
	tailOffset := anchorBase + spellsSectionAt + 12*spellsRecordSize

	engine.mutex.Lock()
	sess := engine.sessions[loaded.SaveSessionID]
	engine.mutex.Unlock()

	beforeTail, err := sess.snapshot.readAt(tailOffset, 16)
	if err != nil {
		t.Fatalf("read tail before: %v", err)
	}

	_, err = engine.SetEquippedSpells(loaded.SaveSessionID, 0, []uint32{0x0FA0, 0x1068}, 4, "0")
	if err != nil {
		t.Fatalf("SetEquippedSpells: %v", err)
	}

	afterTail, err := sess.snapshot.readAt(tailOffset, 16)
	if err != nil {
		t.Fatalf("read tail after: %v", err)
	}

	if !bytes.Equal(beforeTail, afterTail) {
		t.Errorf("tail bytes changed: before %X, after %X", beforeTail, afterTail)
	}
}

func TestSetEquippedSpellsActiveIndexBehavior(t *testing.T) {
	content := spellsFixture{
		platform: PlatformPC, slot: 0, flag: 1, anchorAt: 0x1000,
		occupiedIDs: []uint32{0x1770, 0x0FA0, 0x1068},
	}
	path := writeSpellsFixture(t, content)
	engine := New()
	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	slotBase := int64(spellsPCSlotDataBase)
	anchorBase := slotBase + content.anchorAt
	activeIdxOffset := anchorBase + spellsSectionAt + 112

	engine.mutex.Lock()
	sess := engine.sessions[loaded.SaveSessionID]
	engine.mutex.Unlock()

	// Set active index to 2 manually in snapshot
	binary.LittleEndian.PutUint32(sess.snapshot.data[activeIdxOffset:], 2)

	// Mutate to 2 spells -> previous index 2 is out of bounds [0..1] -> reset to 0
	_, err = engine.SetEquippedSpells(loaded.SaveSessionID, 0, []uint32{0x0FA0, 0x1068}, 4, "0")
	if err != nil {
		t.Fatalf("SetEquippedSpells: %v", err)
	}
	idxAfter := binary.LittleEndian.Uint32(sess.snapshot.data[activeIdxOffset:])
	if idxAfter != 0 {
		t.Errorf("active index = %d, want 0 (out of bounds reset)", idxAfter)
	}

	// Mutate to empty list -> active index set to 0xFFFFFFFF
	_, err = engine.SetEquippedSpells(loaded.SaveSessionID, 0, nil, 0, "1")
	if err != nil {
		t.Fatalf("SetEquippedSpells empty: %v", err)
	}
	idxEmpty := binary.LittleEndian.Uint32(sess.snapshot.data[activeIdxOffset:])
	if idxEmpty != 0xFFFFFFFF {
		t.Errorf("active index empty = 0x%08X, want 0xFFFFFFFF", idxEmpty)
	}
}

func TestSetEquippedSpellsRejectsInvalidRecordPairInPlayableSlots(t *testing.T) {
	var records [14][2]uint32
	for i := range records {
		records[i] = [2]uint32{0xFFFFFFFF, 0x00000000}
	}
	records[2] = [2]uint32{0x1770, 0x00000000} // Invalid pair in record index 2 (record 3)

	content := spellsFixture{
		platform: PlatformPC, slot: 0, flag: 1, anchorAt: 0x1000,
		records: &records,
	}
	path := writeSpellsFixture(t, content)
	engine := New()
	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	engine.mutex.Lock()
	sess := engine.sessions[loaded.SaveSessionID]
	engine.mutex.Unlock()
	snapshotBefore := append([]byte(nil), sess.snapshot.data...)

	wantErr := "spell record 2 of character 0 stores the pair (0x00001770, 0x00000000), which is neither empty nor occupied"
	_, err = engine.SetEquippedSpells(loaded.SaveSessionID, 0, []uint32{0x0FA0}, 1, "0")
	if err == nil || err.Error() != wantErr {
		t.Fatalf("error = %v, want %q", err, wantErr)
	}

	if !bytes.Equal(snapshotBefore, sess.snapshot.data) {
		t.Error("snapshot bytes changed after rejected mutation")
	}
	if rev := sess.session.revisionString(); rev != "0" {
		t.Errorf("revision = %q, want 0", rev)
	}
	if sess.session.dirty {
		t.Error("session is dirty, want clean")
	}
}
