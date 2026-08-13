package equipment

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Synthetic PC container layout used only by this test. The endpoint owns none
// of these values; they are duplicated here so the fixture is accepted by
// SaveEngine without sharing anything with another test file.
const (
	getEquippedSpellsHeaderSize       = 0x300
	getEquippedSpellsEntryCountOffset = 0x0C
	getEquippedSpellsEntryCount       = 12
	getEquippedSpellsSlotBlockSize    = 0x280010
	getEquippedSpellsFixtureSize      = int64(getEquippedSpellsHeaderSize) +
		10*getEquippedSpellsSlotBlockSize + 0x60010

	getEquippedSpellsUserData10Offset = int64(getEquippedSpellsHeaderSize) +
		10*getEquippedSpellsSlotBlockSize + 0x10
	getEquippedSpellsFlagsOffset = 0x1954

	getEquippedSpellsSlot                = 3
	getEquippedSpellsAnchorAt            = 0x0640
	getEquippedSpellsSectionAt           = 0x9205
	getEquippedSpellsCountAt             = 0x931D
	getEquippedSpellsInventoryAt         = 505
	getEquippedSpellsTalismansAt         = -241
	getEquippedSpellsProjectileCount     = 17
	getEquippedSpellsPublicRecordCount   = 12
	getEquippedSpellsPhysicalRecordCount = 14
	getEquippedSpellsRecordSize          = 8

	// The fixture gives the character three Memory Stones, so the base capacity
	// of two plus three stones is the expected available capacity.
	getEquippedSpellsMemoryStones      = 3
	getEquippedSpellsWantAvailable     = 5
	getEquippedSpellsMemoryStoneHandle = 0xB000272E

	// Raw MagicParam identifiers the fixture equips, and the catalog facts they
	// must resolve to.
	rawGlintstonePebble = 0x0FA0
	rawCometAzur        = 0x1068
	rawRennalasFullMoon = 0x1108
	rawMemoryStoneGoods = 0x272E // a known item that is not a spell
	rawUnknownSpell     = 0x0FFF // no catalog item carries game ID 0x40000FFF
)

// getEquippedSpellsAnchor is the 65-byte anchor the chain is measured from,
// restated here independently of the implementation: one leading 0x00 byte, then
// four full repetitions of a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed
// by twelve 0x00 bytes.
var getEquippedSpellsAnchor = []byte{
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

// writeGetEquippedSpellsFixture writes a minimal synthetic PC save into
// t.TempDir() with one active character whose EquippedSpells section holds the
// given raw identifiers followed by native empty records, and returns its path.
func writeGetEquippedSpellsFixture(t *testing.T, occupied []uint32) string {
	t.Helper()

	data := make([]byte, getEquippedSpellsFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[getEquippedSpellsEntryCountOffset:], getEquippedSpellsEntryCount)

	data[getEquippedSpellsUserData10Offset+getEquippedSpellsFlagsOffset+getEquippedSpellsSlot] = 1

	slotBase := int64(getEquippedSpellsHeaderSize) + 0x10 +
		getEquippedSpellsSlot*getEquippedSpellsSlotBlockSize
	anchorBase := slotBase + getEquippedSpellsAnchorAt
	copy(data[anchorBase:], getEquippedSpellsAnchor)

	for index := 0; index < getEquippedSpellsPhysicalRecordCount; index++ {
		at := anchorBase + getEquippedSpellsSectionAt + int64(index)*getEquippedSpellsRecordSize
		spellID, follower := uint32(0xFFFFFFFF), uint32(0x00000000)
		if index < len(occupied) {
			spellID, follower = occupied[index], 0xFFFFFFFF
		}
		binary.LittleEndian.PutUint32(data[at:], spellID)
		binary.LittleEndian.PutUint32(data[at+4:], follower)
	}

	// One Memory Stone stack, and no talisman beyond the single unlocked field.
	stoneAt := anchorBase + getEquippedSpellsInventoryAt + 97*12
	binary.LittleEndian.PutUint32(data[stoneAt:], getEquippedSpellsMemoryStoneHandle)
	binary.LittleEndian.PutUint32(data[stoneAt+4:], getEquippedSpellsMemoryStones)
	data[anchorBase+getEquippedSpellsTalismansAt] = 0
	binary.LittleEndian.PutUint32(
		data[anchorBase+getEquippedSpellsCountAt:], getEquippedSpellsProjectileCount)

	path := filepath.Join(t.TempDir(), "get-equipped-spells.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// newEquippedSpellsCatalog builds a catalog from the stored catalog data, so the
// spells the test resolves are the real documents and not a local invention.
func newEquippedSpellsCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	return catalogOf(t, storedCatalogResources(t))
}

func storedCatalogResources(t *testing.T) []schema.Resource {
	t.Helper()

	// Resources() returns a fresh slice per call, so one test may patch a
	// document without disturbing another.
	return storedCatalogData(t).Resources()
}

func catalogOf(t *testing.T, resources []schema.Resource) *gamecatalog.Catalog {
	t.Helper()

	gameCatalog, err := gamecatalog.New(storedCatalogData(t).Manifest, resources)
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	return gameCatalog
}

// storedCatalogData reads the stored catalog files once for the whole test
// binary; parsing all of them per subtest would dominate the runtime.
func storedCatalogData(t *testing.T) loader.Data {
	t.Helper()

	equippedSpellsCatalogOnce.Do(func() {
		equippedSpellsCatalogData, equippedSpellsCatalogErr = loader.LoadFS(catalogdata.Files())
	})
	if equippedSpellsCatalogErr != nil {
		t.Fatalf("loader.LoadFS: %v", equippedSpellsCatalogErr)
	}
	return equippedSpellsCatalogData
}

var (
	equippedSpellsCatalogOnce sync.Once
	equippedSpellsCatalogData loader.Data
	equippedSpellsCatalogErr  error
)

func loadEquippedSpellsSession(t *testing.T, occupied []uint32) (*saveengine.Engine, string) {
	t.Helper()

	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetEquippedSpellsFixture(t, occupied), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func TestGetEquippedSpellsResolvesEveryOccupiedRecord(t *testing.T) {
	engine, sessionID := loadEquippedSpellsSession(t,
		[]uint32{rawGlintstonePebble, rawCometAzur, rawRennalasFullMoon})

	result, err := GetEquippedSpells(engine, newEquippedSpellsCatalog(t), sessionID, getEquippedSpellsSlot)
	if err != nil {
		t.Fatalf("GetEquippedSpells: %v", err)
	}

	if result.SaveSessionID != sessionID || result.CharacterID != getEquippedSpellsSlot || !result.Active {
		t.Fatalf("result header = %q/%d/%t, want %q/%d/true",
			result.SaveSessionID, result.CharacterID, result.Active, sessionID, getEquippedSpellsSlot)
	}
	if len(result.Spells) != getEquippedSpellsPublicRecordCount {
		t.Fatalf("record count = %d, want %d", len(result.Spells), getEquippedSpellsPublicRecordCount)
	}

	wantOccupied := []EquippedSpellSlot{
		{RawMagicParamID: rawGlintstonePebble, ResourceKey: "40000FA0", Name: "Glintstone Pebble", MemorySlots: 1},
		{RawMagicParamID: rawCometAzur, ResourceKey: "40001068", Name: "Comet Azur", MemorySlots: 3},
		{RawMagicParamID: rawRennalasFullMoon, ResourceKey: "40001108", Name: "Rennala's Full Moon", MemorySlots: 2},
	}
	for index, want := range wantOccupied {
		if result.Spells[index] != want {
			t.Errorf("record %d = %+v, want %+v", index, result.Spells[index], want)
		}
	}
	// Every remaining record keeps the native sentinel and stays unresolved.
	for index := len(wantOccupied); index < getEquippedSpellsPublicRecordCount; index++ {
		want := EquippedSpellSlot{RawMagicParamID: 0xFFFFFFFF}
		if result.Spells[index] != want {
			t.Errorf("empty record %d = %+v, want %+v", index, result.Spells[index], want)
		}
	}

	if result.UsedMemorySlots != 6 {
		t.Errorf("usedMemorySlots = %d, want 6", result.UsedMemorySlots)
	}
	if result.AvailableMemorySlots != getEquippedSpellsWantAvailable {
		t.Errorf("availableMemorySlots = %d, want %d",
			result.AvailableMemorySlots, getEquippedSpellsWantAvailable)
	}
}

func TestGetEquippedSpellsRejectsUnresolvableRecords(t *testing.T) {
	cases := map[string]struct {
		occupied []uint32
		want     string
	}{
		"unknown raw spell ID": {
			[]uint32{rawGlintstonePebble, rawUnknownSpell},
			"spell slot 1: game ID 0x40000FFF is not a known item",
		},
		"known item that is not a spell": {
			[]uint32{rawMemoryStoneGoods},
			"spell slot 0: game ID 0x4000272E is not a spell",
		},
		"raw ID that already carries family bits": {
			[]uint32{0x40000FA0},
			"spell slot 0: 0x40000FA0 is not a raw MagicParam ID",
		},
		"zero raw ID": {
			[]uint32{0x00000000},
			"spell slot 0: 0x00000000 is not a raw MagicParam ID",
		},
	}

	gameCatalog := newEquippedSpellsCatalog(t)
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			engine, sessionID := loadEquippedSpellsSession(t, testCase.occupied)

			result, err := GetEquippedSpells(engine, gameCatalog, sessionID, getEquippedSpellsSlot)
			if err == nil {
				t.Fatal("GetEquippedSpells accepted an unresolvable record")
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, GetEquippedSpellsResult{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

// A spell whose Memory Slots cost is not known cannot be costed, so it fails
// closed instead of contributing a guessed zero.
func TestGetEquippedSpellsRejectsUnknownMemorySlots(t *testing.T) {
	resources := storedCatalogResources(t)
	patched := false
	for index := range resources {
		item := resources[index].Item
		if item == nil || item.GameID.Value != 0x40000000|rawGlintstonePebble || item.Spell == nil {
			continue
		}
		spell := *item.Spell
		spell.MemorySlots.Known = false
		spell.MemorySlots.Value = 0
		document := *item
		document.Spell = &spell
		resources[index].Item = &document
		patched = true
		break
	}
	if !patched {
		t.Fatal("the catalog carries no Glintstone Pebble document to patch")
	}

	engine, sessionID := loadEquippedSpellsSession(t, []uint32{rawGlintstonePebble})

	result, err := GetEquippedSpells(engine, catalogOf(t, resources), sessionID, getEquippedSpellsSlot)
	if err == nil {
		t.Fatal("GetEquippedSpells accepted a spell without a known memory-slot cost")
	}
	if err.Error() != "spell slot 0: spell 0x40000FA0 has no known memory slots" {
		t.Errorf("error = %q, want the unknown-memory-slots error", err)
	}
	if !reflect.DeepEqual(result, GetEquippedSpellsResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}

func TestGetEquippedSpellsRejectsMissingDependencies(t *testing.T) {
	engine, sessionID := loadEquippedSpellsSession(t, []uint32{rawGlintstonePebble})

	t.Run("nil engine", func(t *testing.T) {
		result, err := GetEquippedSpells(nil, newEquippedSpellsCatalog(t), sessionID, getEquippedSpellsSlot)
		if err == nil {
			t.Fatal("GetEquippedSpells accepted a nil engine")
		}
		if err.Error() != "save engine is not available" {
			t.Errorf("error = %q, want %q", err, "save engine is not available")
		}
		if !reflect.DeepEqual(result, GetEquippedSpellsResult{}) {
			t.Errorf("result = %+v, want the zero value", result)
		}
	})

	t.Run("nil catalog", func(t *testing.T) {
		result, err := GetEquippedSpells(engine, nil, sessionID, getEquippedSpellsSlot)
		if err == nil {
			t.Fatal("GetEquippedSpells accepted a nil catalog")
		}
		if err.Error() != "game catalog is not available" {
			t.Errorf("error = %q, want %q", err, "game catalog is not available")
		}
		if !reflect.DeepEqual(result, GetEquippedSpellsResult{}) {
			t.Errorf("result = %+v, want the zero value", result)
		}
	})
}

// The session and slot rules stay in SaveEngine; the endpoint only passes them
// on, so its errors are the engine's wording.
func TestGetEquippedSpellsRejectsInvalidRequests(t *testing.T) {
	engine, sessionID := loadEquippedSpellsSession(t, []uint32{rawGlintstonePebble})
	gameCatalog := newEquippedSpellsCatalog(t)

	cases := map[string]struct {
		saveSessionID string
		characterID   int
		want          string
	}{
		"missing session": {"", getEquippedSpellsSlot, "saveSessionID is required"},
		"unknown session": {"missing", getEquippedSpellsSlot, `unknown save session "missing"`},
		"characterID -1":  {sessionID, -1, "characterID -1 is outside the range 0..9"},
		"characterID 10":  {sessionID, 10, "characterID 10 is outside the range 0..9"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := GetEquippedSpells(engine, gameCatalog, testCase.saveSessionID, testCase.characterID)
			if err == nil {
				t.Fatalf("GetEquippedSpells accepted %s", name)
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, GetEquippedSpellsResult{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

// An inactive slot is a normal result: fourteen zero-value records, both counts
// zero, and no catalog lookup at all.
func TestGetEquippedSpellsReportsAnInactiveSlot(t *testing.T) {
	engine, sessionID := loadEquippedSpellsSession(t, []uint32{rawGlintstonePebble})

	const inactiveSlot = 5
	result, err := GetEquippedSpells(engine, newEquippedSpellsCatalog(t), sessionID, inactiveSlot)
	if err != nil {
		t.Fatalf("GetEquippedSpells: %v", err)
	}
	if result.Active || result.UsedMemorySlots != 0 || result.AvailableMemorySlots != 0 {
		t.Fatalf("result = %+v, want an inactive slot with both counts zero", result)
	}
	if len(result.Spells) != getEquippedSpellsPublicRecordCount {
		t.Fatalf("record count = %d, want %d", len(result.Spells), getEquippedSpellsPublicRecordCount)
	}
	for index, spell := range result.Spells {
		if spell != (EquippedSpellSlot{}) {
			t.Errorf("record %d = %+v, want the zero value", index, spell)
		}
	}
	if strings.TrimSpace(result.SaveSessionID) != sessionID {
		t.Errorf("saveSessionID = %q, want %q", result.SaveSessionID, sessionID)
	}
}
