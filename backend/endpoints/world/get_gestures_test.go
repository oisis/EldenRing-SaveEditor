package world

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Synthetic PC container layout used only by this test. The endpoint owns none
// of these values; they are restated here so the fixture is accepted by
// SaveEngine without sharing anything with another test file.
const (
	getGesturesHeaderSize       = 0x300
	getGesturesEntryCountOffset = 0x0C
	getGesturesEntryCount       = 12
	getGesturesSlotBlockSize    = 0x280010
	getGesturesFixtureSize      = int64(getGesturesHeaderSize) +
		10*getGesturesSlotBlockSize + 0x60010

	getGesturesUserData10Offset = int64(getGesturesHeaderSize) +
		10*getGesturesSlotBlockSize + 0x10
	getGesturesFlagsOffset = 0x1954

	getGesturesSlot            = 3
	getGesturesAnchorAt        = 0x0640
	getGesturesProjectileCount = 17

	// Distance from the anchor to the GestureGameData block, restated literally:
	// the projectile count, its four-byte header, the projectile records, the
	// equipped-armaments block, EquipPhysicsData, the face data and the whole
	// Storage Box.
	getGesturesSectionAt = 0x931D + 4 + getGesturesProjectileCount*8 +
		0x9C + 0x0C + 0x12F + 0x6010

	getGesturesSlotCount  = 64
	getGesturesRecordSize = 4

	// The native empty sentinel of one GestureGameData record.
	getGesturesEmptySentinel = uint32(0xFFFFFFFE)

	// The number of gesture slots the stored catalog declares, and the number of
	// documents they come from. They differ because one resource declares two.
	getGesturesCatalogSlots     = 57
	getGesturesCatalogDocuments = 56

	// The resource that declares two gesture slots, and the two canonical slot
	// IDs it carries.
	getGesturesMultiSlotKey    = "401EA7A8"
	getGesturesMultiSlotFirst  = uint32(227)
	getGesturesMultiSlotSecond = uint32(233)

	// Canonical slot IDs the fixture unlocks, and raw values that must unlock
	// nothing: a plain zero, an even value no canonical gesture carries and an
	// odd value no catalog slot declares.
	getGesturesRawBow     = uint32(1)
	getGesturesRawRest    = uint32(185)
	getGesturesRawUnknown = uint32(4243)
	getGesturesRawEven    = uint32(44)
)

// getGesturesAnchor is the 65-byte anchor the chain is measured from, restated
// here independently of the implementation: one leading 0x00 byte, then four
// full repetitions of a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed by
// twelve 0x00 bytes.
var getGesturesAnchor = []byte{
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

// getGesturesRecords is the 64-record block the fixture writes. It unlocks three
// canonical slot IDs, stores one of them twice, and fills the rest with values
// that must unlock nothing: an explicit zero, an even value, an unknown odd
// value and the native empty sentinel.
func getGesturesRecords() []uint32 {
	records := make([]uint32, getGesturesSlotCount)
	for index := range records {
		records[index] = getGesturesEmptySentinel
	}
	records[0] = getGesturesRawBow
	records[1] = getGesturesRawRest
	records[2] = getGesturesRawRest // the same canonical ID a second time
	records[3] = 0
	records[4] = getGesturesRawEven
	records[5] = getGesturesRawUnknown
	records[6] = getGesturesMultiSlotFirst // only one slot of the two-slot resource
	return records
}

// getGesturesUnlocked is the set of canonical slot IDs the fixture unlocks.
var getGesturesUnlocked = map[uint32]bool{
	getGesturesRawBow:         true,
	getGesturesRawRest:        true,
	getGesturesMultiSlotFirst: true,
}

// writeGetGesturesFixture writes a minimal synthetic PC save into t.TempDir()
// with one active character whose GestureGameData holds the given records, and
// returns its path. A zero flag expresses the residual slot: the block is still
// written, so an empty unlock set proves it was never read.
func writeGetGesturesFixture(t *testing.T, records []uint32, active bool) string {
	t.Helper()

	data := make([]byte, getGesturesFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[getGesturesEntryCountOffset:], getGesturesEntryCount)

	if active {
		data[getGesturesUserData10Offset+getGesturesFlagsOffset+getGesturesSlot] = 1
	}

	slotBase := int64(getGesturesHeaderSize) + 0x10 +
		getGesturesSlot*getGesturesSlotBlockSize
	anchorBase := slotBase + getGesturesAnchorAt
	copy(data[anchorBase:], getGesturesAnchor)
	binary.LittleEndian.PutUint32(
		data[anchorBase+0x931D:], getGesturesProjectileCount)

	for index, record := range records {
		at := anchorBase + getGesturesSectionAt + int64(index)*getGesturesRecordSize
		binary.LittleEndian.PutUint32(data[at:], record)
	}

	path := filepath.Join(t.TempDir(), "get-gestures.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadGesturesSession(t *testing.T, records []uint32, active bool) (*saveengine.Engine, string) {
	t.Helper()

	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetGesturesFixture(t, records, active), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

// newGesturesCatalog builds a catalog from the stored catalog data, so the
// gestures the test resolves are the real documents and not a local invention.
func newGesturesCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	return gesturesCatalogOf(t, storedGestureResources(t))
}

func storedGestureResources(t *testing.T) []schema.Resource {
	t.Helper()

	// Resources() returns a fresh slice per call, so one test may patch a
	// document without disturbing another.
	return storedGestureCatalogData(t).Resources()
}

func gesturesCatalogOf(t *testing.T, resources []schema.Resource) *gamecatalog.Catalog {
	t.Helper()

	gameCatalog, err := gamecatalog.New(storedGestureCatalogData(t).Manifest, resources)
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	return gameCatalog
}

// storedGestureCatalogData reads the stored catalog files once for the whole
// test binary; parsing all of them per subtest would dominate the runtime.
func storedGestureCatalogData(t *testing.T) loader.Data {
	t.Helper()

	gesturesCatalogOnce.Do(func() {
		gesturesCatalogData, gesturesCatalogErr = loader.LoadFS(catalogdata.Files())
	})
	if gesturesCatalogErr != nil {
		t.Fatalf("loader.LoadFS: %v", gesturesCatalogErr)
	}
	return gesturesCatalogData
}

var (
	gesturesCatalogOnce sync.Once
	gesturesCatalogData loader.Data
	gesturesCatalogErr  error
)

func TestGetGesturesRejectsMissingBackends(t *testing.T) {
	engine, sessionID := loadGesturesSession(t, getGesturesRecords(), true)

	cases := map[string]struct {
		engine      *saveengine.Engine
		gameCatalog *gamecatalog.Catalog
		want        string
	}{
		"nil engine":  {nil, newGesturesCatalog(t), "save engine is not available"},
		"nil catalog": {engine, nil, "game catalog is not available"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := GetGestures(testCase.engine, testCase.gameCatalog, sessionID, getGesturesSlot, "")
			if err == nil {
				t.Fatal("GetGestures accepted a missing backend")
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if len(result.Gestures) != 0 || result.Active {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

func TestGetGesturesReturnsEveryCatalogGestureWithItsUnlockState(t *testing.T) {
	engine, sessionID := loadGesturesSession(t, getGesturesRecords(), true)

	result, err := GetGestures(engine, newGesturesCatalog(t), sessionID, getGesturesSlot, "")
	if err != nil {
		t.Fatalf("GetGestures: %v", err)
	}

	if result.SaveSessionID != sessionID || result.CharacterID != getGesturesSlot || !result.Active {
		t.Fatalf("result header = %q/%d/%t, want %q/%d/true",
			result.SaveSessionID, result.CharacterID, result.Active, sessionID, getGesturesSlot)
	}
	if len(result.Gestures) != getGesturesCatalogSlots {
		t.Fatalf("gesture count = %d, want %d", len(result.Gestures), getGesturesCatalogSlots)
	}

	// One resource declares two gesture slots, so the entry count is higher than
	// the document count and both slots are present as separate entries.
	documents := make(map[string]struct{}, len(result.Gestures))
	multiSlot := make([]uint32, 0, 2)
	for _, entry := range result.Gestures {
		if entry.Kind != schema.ResourceKindItem {
			t.Errorf("gesture %q kind = %q, want %q", entry.Key, entry.Kind, schema.ResourceKindItem)
		}
		if entry.Key == "" || entry.Name == "" || entry.Category == "" {
			t.Errorf("gesture %+v carries an empty catalog value", entry)
		}
		documents[entry.Key] = struct{}{}
		if entry.Key == getGesturesMultiSlotKey {
			multiSlot = append(multiSlot, entry.SlotID)
		}
		if entry.Unlocked != getGesturesUnlocked[entry.SlotID] {
			t.Errorf("gesture %q slot %d unlocked = %t, want %t",
				entry.Name, entry.SlotID, entry.Unlocked, getGesturesUnlocked[entry.SlotID])
		}
	}
	if len(documents) != getGesturesCatalogDocuments {
		t.Errorf("document count = %d, want %d", len(documents), getGesturesCatalogDocuments)
	}

	sort.Slice(multiSlot, func(i, j int) bool { return multiSlot[i] < multiSlot[j] })
	want := []uint32{getGesturesMultiSlotFirst, getGesturesMultiSlotSecond}
	if len(multiSlot) != len(want) || multiSlot[0] != want[0] || multiSlot[1] != want[1] {
		t.Errorf("slots of %q = %v, want %v", getGesturesMultiSlotKey, multiSlot, want)
	}
}

func TestGetGesturesUnlocksOnlyExactCanonicalMatches(t *testing.T) {
	engine, sessionID := loadGesturesSession(t, getGesturesRecords(), true)

	result, err := GetGestures(engine, newGesturesCatalog(t), sessionID, getGesturesSlot, "")
	if err != nil {
		t.Fatalf("GetGestures: %v", err)
	}

	bySlot := make(map[uint32]GestureEntry, len(result.Gestures))
	occurrences := make(map[uint32]int, len(result.Gestures))
	for _, entry := range result.Gestures {
		bySlot[entry.SlotID] = entry
		occurrences[entry.SlotID]++
	}

	// The three canonical IDs the block carries are unlocked, and the one stored
	// twice still produces exactly one catalog entry.
	for _, slotID := range []uint32{getGesturesRawBow, getGesturesRawRest, getGesturesMultiSlotFirst} {
		entry, exists := bySlot[slotID]
		if !exists {
			t.Fatalf("slot %d is missing from the result", slotID)
		}
		if !entry.Unlocked {
			t.Errorf("slot %d unlocked = false, want true", slotID)
		}
		if occurrences[slotID] != 1 {
			t.Errorf("slot %d appears %d times, want 1", slotID, occurrences[slotID])
		}
	}

	// The second slot of the two-slot resource was never stored, so it stays
	// locked even though its sibling is unlocked.
	if bySlot[getGesturesMultiSlotSecond].Unlocked {
		t.Errorf("slot %d unlocked = true, want false", getGesturesMultiSlotSecond)
	}

	// A zero, the empty sentinel, an even value and an unknown odd value never
	// resolve to a canonical slot ID, so exactly three gestures are unlocked and
	// no even value was converted into the odd one next to it.
	unlocked := 0
	for _, entry := range result.Gestures {
		if entry.Unlocked {
			unlocked++
		}
	}
	if unlocked != len(getGesturesUnlocked) {
		t.Errorf("unlocked count = %d, want %d", unlocked, len(getGesturesUnlocked))
	}
	for _, raw := range []uint32{getGesturesRawEven, getGesturesRawUnknown, 0, getGesturesEmptySentinel} {
		if entry, exists := bySlot[raw]; exists && entry.Unlocked {
			t.Errorf("raw value %d unlocked gesture %q", raw, entry.Name)
		}
	}
	// The even value stored in the block sits one below a canonical ID; the
	// gesture that canonical ID names must stay locked.
	if bySlot[getGesturesRawEven+1].Unlocked {
		t.Errorf("even raw value %d unlocked the neighbouring canonical slot %d",
			getGesturesRawEven, getGesturesRawEven+1)
	}
}

func TestGetGesturesReportsAResidualSlotAsLocked(t *testing.T) {
	engine, sessionID := loadGesturesSession(t, getGesturesRecords(), false)

	result, err := GetGestures(engine, newGesturesCatalog(t), sessionID, getGesturesSlot, "")
	if err != nil {
		t.Fatalf("GetGestures: %v", err)
	}

	if result.Active {
		t.Error("active = true, want false")
	}
	// The gesture block of the deleted character is still written into the
	// fixture, so a fully locked result proves it was never read.
	if len(result.Gestures) != getGesturesCatalogSlots {
		t.Fatalf("gesture count = %d, want %d", len(result.Gestures), getGesturesCatalogSlots)
	}
	for _, entry := range result.Gestures {
		if entry.Unlocked {
			t.Fatalf("gesture %q of an inactive slot is unlocked", entry.Name)
		}
	}
}

func TestGetGesturesOrdersByCategoryThenNameThenSlotID(t *testing.T) {
	engine, sessionID := loadGesturesSession(t, getGesturesRecords(), true)

	// Two resources are given the same category and name, so the canonical slot
	// ID is the only remaining tie-breaker. The stored documents never collide
	// on both values, so the tie is created here on purpose.
	resources := storedGestureResources(t)
	renamed := 0
	for index := range resources {
		item := resources[index].Item
		if item == nil || item.Gesture == nil {
			continue
		}
		if resources[index].Key != "40002340" { // Fire Spur Me, Battle, slot 109
			continue
		}
		item.Gesture.Slots[0].Name.Value = "By My Sword" // Battle, slot 105
		renamed++
	}
	if renamed != 1 {
		t.Fatalf("patched %d documents, want 1", renamed)
	}

	result, err := GetGestures(engine, gesturesCatalogOf(t, resources), sessionID, getGesturesSlot, "")
	if err != nil {
		t.Fatalf("GetGestures: %v", err)
	}

	for index := 1; index < len(result.Gestures); index++ {
		previous, current := result.Gestures[index-1], result.Gestures[index]
		switch {
		case previous.Category != current.Category:
			if previous.Category > current.Category {
				t.Fatalf("entry %d category %q follows %q", index, current.Category, previous.Category)
			}
		case previous.Name != current.Name:
			if previous.Name > current.Name {
				t.Fatalf("entry %d name %q follows %q", index, current.Name, previous.Name)
			}
		case previous.SlotID >= current.SlotID:
			t.Fatalf("entry %d slot %d follows %d", index, current.SlotID, previous.SlotID)
		}
	}

	// The two colliding entries must be adjacent and ordered by slot ID.
	tied := make([]uint32, 0, 2)
	for _, entry := range result.Gestures {
		if entry.Category == "Battle" && entry.Name == "By My Sword" {
			tied = append(tied, entry.SlotID)
		}
	}
	if len(tied) != 2 || tied[0] != 105 || tied[1] != 109 {
		t.Errorf("tied slot IDs = %v, want [105 109]", tied)
	}
}

func TestGetGesturesFiltersByAvailability(t *testing.T) {
	engine, sessionID := loadGesturesSession(t, getGesturesRecords(), true)
	gameCatalog := newGesturesCatalog(t)

	full, err := GetGestures(engine, gameCatalog, sessionID, getGesturesSlot, "")
	if err != nil {
		t.Fatalf("GetGestures: %v", err)
	}

	cases := map[string]struct {
		filter string
		want   int
	}{
		"every gesture": {"", getGesturesCatalogSlots},
		"unlocked only": {GestureAvailabilityUnlocked, len(getGesturesUnlocked)},
		"locked only":   {GestureAvailabilityLocked, getGesturesCatalogSlots - len(getGesturesUnlocked)},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := GetGestures(engine, gameCatalog, sessionID, getGesturesSlot, testCase.filter)
			if err != nil {
				t.Fatalf("GetGestures: %v", err)
			}
			if len(result.Gestures) != testCase.want {
				t.Fatalf("gesture count = %d, want %d", len(result.Gestures), testCase.want)
			}
			if result.Gestures == nil {
				t.Fatal("gestures is nil, want an empty list")
			}

			// A filter may only remove entries: the ones it keeps stay in the
			// order and with the state the unfiltered result gave them.
			position := 0
			for _, entry := range full.Gestures {
				if testCase.filter == GestureAvailabilityUnlocked && !entry.Unlocked {
					continue
				}
				if testCase.filter == GestureAvailabilityLocked && entry.Unlocked {
					continue
				}
				if result.Gestures[position] != entry {
					t.Fatalf("entry %d = %+v, want %+v", position, result.Gestures[position], entry)
				}
				position++
			}
			if position != len(result.Gestures) {
				t.Errorf("matched %d entries, want %d", position, len(result.Gestures))
			}
		})
	}
}

func TestGetGesturesRejectsAnUnknownAvailabilityFilter(t *testing.T) {
	engine, sessionID := loadGesturesSession(t, getGesturesRecords(), true)
	gameCatalog := newGesturesCatalog(t)

	// The value is matched exactly and case-sensitively and is never trimmed, so
	// a padded, case-shifted or aliased value is an error, not the filter it
	// resembles.
	for _, filter := range []string{"Unlocked", " unlocked", "unlocked ", "LOCKED", "all", "true"} {
		t.Run(strconv.Quote(filter), func(t *testing.T) {
			result, err := GetGestures(engine, gameCatalog, sessionID, getGesturesSlot, filter)
			if err == nil {
				t.Fatalf("GetGestures accepted %q", filter)
			}
			want := `availabilityFilter must be "unlocked", "locked" or empty; got ` + strconv.Quote(filter)
			if err.Error() != want {
				t.Errorf("error = %q, want %q", err, want)
			}
			if len(result.Gestures) != 0 || result.Active {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

func TestGetGesturesRejectsInvalidSessionAndCharacter(t *testing.T) {
	engine, sessionID := loadGesturesSession(t, getGesturesRecords(), true)
	gameCatalog := newGesturesCatalog(t)

	cases := map[string]struct {
		saveSessionID string
		characterID   int
		want          string
	}{
		"empty session":   {"", getGesturesSlot, "saveSessionID is required"},
		"unknown session": {"missing", getGesturesSlot, `unknown save session "missing"`},
		"characterID -1":  {sessionID, -1, "characterID -1 is outside the range 0..9"},
		"characterID 10":  {sessionID, 10, "characterID 10 is outside the range 0..9"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := GetGestures(engine, gameCatalog, testCase.saveSessionID, testCase.characterID, "")
			if err == nil {
				t.Fatalf("GetGestures accepted %s", name)
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
		})
	}
}
