package world

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
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
	getCookbooksHeaderSize       = 0x300
	getCookbooksEntryCountOffset = 0x0C
	getCookbooksEntryCount       = 12
	getCookbooksSlotBlockSize    = 0x280010
	getCookbooksFixtureSize      = int64(getCookbooksHeaderSize) +
		10*getCookbooksSlotBlockSize + 0x60010

	getCookbooksUserData10Offset = int64(getCookbooksHeaderSize) +
		10*getCookbooksSlotBlockSize + 0x10
	getCookbooksFlagsOffset = 0x1954

	getCookbooksSlot     = 3
	getCookbooksAnchorAt = 0x01A7

	// The declared lengths the fixture writes into the chain. None of them is a
	// legacy default: the tutorial payload is deliberately not 0x400 bytes long,
	// so a reader that assumes the legacy size lands on a shifted bitfield.
	getCookbooksProjectiles  = 37
	getCookbooksRegions      = 91
	getCookbooksMenuSize     = 0x800
	getCookbooksTutorialSize = 0x321

	// Distance from the anchor to the uint32 that declares the acquired
	// projectiles: SpEffect, EquipedItemIndex, ActiveEquipedItems,
	// EquipedItemsID, ActiveEquipedItemsGa, InventoryHeld, EquippedSpells,
	// EquipItemData and EquippedGestures.
	getCookbooksProjectileCountAt = 0xD0 + 0x58 + 0x1C + 0x58 + 0x58 + 0x9011 + 0x74 + 0x8C + 0x18

	// The fixed blocks behind the projectile records: the equipped armaments,
	// EquipPhysicsData, the face data, the Storage Box and GestureGameData.
	getCookbooksBlocksBeforeStorage = 0x9C + 0x0C + 0x12F
	getCookbooksStorageBoxSize      = 0x6010
	getCookbooksGestureSectionSize  = 64 * 4

	// Torrent plus its control byte, and the blood stain plus its padding.
	getCookbooksHorseSize      = 0x28 + 1
	getCookbooksBloodStainSize = 0x44 + 8

	// A variable block is stored as u16 + u16 + u32 size, then its payload.
	getCookbooksDynamicHeader = 2 + 2 + 4

	// TrophyEquipData, and GaItemGameData with its int64 count in front of 7000
	// sixteen-byte entries.
	getCookbooksTrophySize = 0x34
	getCookbooksGaItemSize = 8 + 7000*16

	// The scalar block between the end of the tutorial data and the bitfield.
	getCookbooksScalarsSize = 3 + 4 + 4 + 1 + 4 + 4 + 1 + 4 + 4

	getCookbooksBlockSize = 125

	// Distance from the anchor to the first byte of the event flag bitfield, for
	// the lengths this fixture declares.
	getCookbooksSectionAt = getCookbooksProjectileCountAt + 4 +
		getCookbooksProjectiles*8 +
		getCookbooksBlocksBeforeStorage + getCookbooksStorageBoxSize +
		getCookbooksGestureSectionSize + 4 +
		getCookbooksRegions*4 + getCookbooksHorseSize + getCookbooksBloodStainSize +
		getCookbooksDynamicHeader + getCookbooksMenuSize +
		getCookbooksTrophySize + getCookbooksGaItemSize +
		getCookbooksDynamicHeader + getCookbooksTutorialSize +
		getCookbooksScalarsSize

	// The number of cookbooks the stored catalog declares.
	getCookbooksCatalogCount = 104

	// Two resources whose cookbook unlocks the patched-catalog tests operate on.
	getCookbooksFirstKey  = "40002454" // Nomadic Warrior's Cookbook [1], flag 67000
	getCookbooksSecondKey = "40002455" // Nomadic Warrior's Cookbook [3], flag 67010

	// A stored weapon resource, used to place a cookbook unlock outside the goods
	// family without inventing a document the catalog would reject.
	getCookbooksWeaponKey = "0001ADB0"
)

// getCookbooksAnchor is the 65-byte anchor the chain is measured from, restated
// here independently of the implementation: one leading 0x00 byte, then four
// full repetitions of a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed by
// twelve 0x00 bytes.
var getCookbooksAnchor = []byte{
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

// getCookbooksSetFlags are the event flags the fixture sets, with the cookbook
// each one belongs to according to the stored catalog. They cover both supported
// blocks and, inside them, the first flag of block 67, the first flag of block 68
// and a late flag of block 68, so a shifted byte or an inverted bit direction
// cannot pass.
var getCookbooksSetFlags = map[uint32]string{
	67000: "Nomadic Warrior's Cookbook [1]",
	68000: "Ancient Dragon Apostle's Cookbook [1]",
	68870: "Tibia's Cookbook",
}

// getCookbooksClearFlags are cookbooks of both blocks whose flag the fixture
// leaves at zero. They sit next to the set ones in the same bitfield byte or in
// the neighbouring one, so an off-by-one in either direction unlocks one of them.
var getCookbooksClearFlags = map[uint32]string{
	67010: "Nomadic Warrior's Cookbook [3]",
	68010: "Ancient Dragon Apostle's Cookbook [2]",
	68950: "St. Trina Disciple's Cookbook [2]",
}

// getCookbooksPosition places one event flag inside the bitfield, restating the
// confirmed formula independently of the implementation.
func getCookbooksPosition(t *testing.T, id uint32) (int64, uint8) {
	t.Helper()

	position, supported := map[uint32]int64{67: 17, 68: 18}[id/1000]
	if !supported {
		t.Fatalf("test fixture cannot place event flag %d", id)
	}
	index := int64(id % 1000)
	return position*getCookbooksBlockSize + index/8, uint8(7 - index%8)
}

// writeGetCookbooksFixture writes a minimal synthetic PC save into t.TempDir()
// with one character slot whose event flag bitfield carries the given flags, and
// returns its path. A zero activity flag expresses the residual slot: the
// bitfield is still written, so a fully locked result proves it was never read.
func writeGetCookbooksFixture(t *testing.T, set map[uint32]string, active bool) string {
	t.Helper()

	data := make([]byte, getCookbooksFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[getCookbooksEntryCountOffset:], getCookbooksEntryCount)

	if active {
		data[getCookbooksUserData10Offset+getCookbooksFlagsOffset+getCookbooksSlot] = 1
	}

	slotBase := int64(getCookbooksHeaderSize) + 0x10 +
		getCookbooksSlot*getCookbooksSlotBlockSize
	anchorBase := slotBase + getCookbooksAnchorAt
	copy(data[anchorBase:], getCookbooksAnchor)

	at := anchorBase + getCookbooksProjectileCountAt
	binary.LittleEndian.PutUint32(data[at:], getCookbooksProjectiles)
	at += 4 + getCookbooksProjectiles*8 +
		getCookbooksBlocksBeforeStorage + getCookbooksStorageBoxSize +
		getCookbooksGestureSectionSize
	binary.LittleEndian.PutUint32(data[at:], getCookbooksRegions)
	at += 4 + getCookbooksRegions*4 + getCookbooksHorseSize + getCookbooksBloodStainSize
	binary.LittleEndian.PutUint32(data[at+4:], getCookbooksMenuSize)
	at += getCookbooksDynamicHeader + getCookbooksMenuSize +
		getCookbooksTrophySize + getCookbooksGaItemSize
	binary.LittleEndian.PutUint32(data[at+4:], getCookbooksTutorialSize)

	for id := range set {
		offset, bit := getCookbooksPosition(t, id)
		data[anchorBase+getCookbooksSectionAt+offset] |= 1 << bit
	}

	path := filepath.Join(t.TempDir(), "get-cookbooks.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadCookbooksSession(t *testing.T, active bool) (*saveengine.Engine, string) {
	t.Helper()

	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetCookbooksFixture(t, getCookbooksSetFlags, active), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

// newCookbooksCatalog builds a catalog from the stored catalog data, so the
// cookbooks the test resolves are the real documents and not a local invention.
func newCookbooksCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	return cookbooksCatalogOf(t, storedCookbookResources(t))
}

func storedCookbookResources(t *testing.T) []schema.Resource {
	t.Helper()

	// Resources() returns a fresh slice per call, so one test may patch a
	// document without disturbing another.
	return storedCookbookCatalogData(t).Resources()
}

func cookbooksCatalogOf(t *testing.T, resources []schema.Resource) *gamecatalog.Catalog {
	t.Helper()

	gameCatalog, err := gamecatalog.New(storedCookbookCatalogData(t).Manifest, resources)
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	return gameCatalog
}

// storedCookbookCatalogData reads the stored catalog files once for the whole
// test binary; parsing all of them per subtest would dominate the runtime.
func storedCookbookCatalogData(t *testing.T) loader.Data {
	t.Helper()

	cookbooksCatalogOnce.Do(func() {
		cookbooksCatalogData, cookbooksCatalogErr = loader.LoadFS(catalogdata.Files())
	})
	if cookbooksCatalogErr != nil {
		t.Fatalf("loader.LoadFS: %v", cookbooksCatalogErr)
	}
	return cookbooksCatalogData
}

var (
	cookbooksCatalogOnce sync.Once
	cookbooksCatalogData loader.Data
	cookbooksCatalogErr  error
)

// patchCookbookDocument applies change to the ItemDocument of one stored
// resource and fails when the key does not select exactly one. Resources() hands
// out a fresh slice but the documents behind it are shared, so the patched
// document and its unlocks are copied first and no other test sees the change.
func patchCookbookDocument(
	t *testing.T, resources []schema.Resource, key string, change func(*schema.ItemDocument),
) []schema.Resource {
	t.Helper()

	patched := 0
	for index := range resources {
		if resources[index].Key != key || resources[index].Item == nil {
			continue
		}
		document := *resources[index].Item
		document.Unlocks = append([]schema.ItemUnlock(nil), document.Unlocks...)
		change(&document)
		resources[index].Item = &document
		patched++
	}
	if patched != 1 {
		t.Fatalf("patched %d resources of %q, want 1", patched, key)
	}
	return resources
}

// storedCookbookUnlock returns the single cookbook unlock of one stored resource,
// so a test can reuse a real unlock — including the provenance the catalog
// requires — instead of inventing one the catalog would reject.
func storedCookbookUnlock(
	t *testing.T, resources []schema.Resource, key string,
) schema.ItemUnlock {
	t.Helper()

	for _, resource := range resources {
		if resource.Key != key || resource.Item == nil {
			continue
		}
		for _, unlock := range resource.Item.Unlocks {
			if unlock.Kind.Known && unlock.Kind.Value == "cookbook" {
				return unlock
			}
		}
	}
	t.Fatalf("resource %q declares no cookbook unlock", key)
	return schema.ItemUnlock{}
}

// patchCookbookUnlock applies change to the single cookbook unlock of one stored
// resource and fails when the document does not carry exactly one.
func patchCookbookUnlock(
	t *testing.T, resources []schema.Resource, key string, change func(*schema.ItemUnlock),
) []schema.Resource {
	t.Helper()

	return patchCookbookDocument(t, resources, key, func(document *schema.ItemDocument) {
		patched := 0
		for unlockIndex := range document.Unlocks {
			unlock := &document.Unlocks[unlockIndex]
			if !unlock.Kind.Known || unlock.Kind.Value != "cookbook" {
				continue
			}
			change(unlock)
			patched++
		}
		if patched != 1 {
			t.Fatalf("patched %d cookbook unlocks of %q, want 1", patched, key)
		}
	})
}

func TestGetCookbooksRejectsMissingBackends(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)

	cases := map[string]struct {
		engine      *saveengine.Engine
		gameCatalog *gamecatalog.Catalog
		want        string
	}{
		"nil engine":  {nil, newCookbooksCatalog(t), "save engine is not available"},
		"nil catalog": {engine, nil, "game catalog is not available"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := GetCookbooks(
				testCase.engine, testCase.gameCatalog, sessionID, getCookbooksSlot, "")
			if err == nil {
				t.Fatal("GetCookbooks accepted a missing backend")
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if len(result.Cookbooks) != 0 || result.Active {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

func TestGetCookbooksReturnsEveryCatalogCookbookWithItsUnlockState(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)

	result, err := GetCookbooks(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot, "")
	if err != nil {
		t.Fatalf("GetCookbooks: %v", err)
	}

	if result.SaveSessionID != sessionID || result.CharacterID != getCookbooksSlot || !result.Active {
		t.Fatalf("result header = %q/%d/%t, want %q/%d/true",
			result.SaveSessionID, result.CharacterID, result.Active, sessionID, getCookbooksSlot)
	}
	if len(result.Cookbooks) != getCookbooksCatalogCount {
		t.Fatalf("cookbook count = %d, want %d", len(result.Cookbooks), getCookbooksCatalogCount)
	}

	byName := make(map[string]CookbookEntry, len(result.Cookbooks))
	for _, entry := range result.Cookbooks {
		if entry.Kind != schema.ResourceKindItem {
			t.Errorf("cookbook %q kind = %q, want %q", entry.Name, entry.Kind, schema.ResourceKindItem)
		}
		if entry.Key == "" || entry.Name == "" || entry.Category == "" {
			t.Errorf("cookbook %+v carries an empty catalog value", entry)
		}
		if _, duplicate := byName[entry.Name]; duplicate {
			t.Errorf("cookbook %q appears twice", entry.Name)
		}
		byName[entry.Name] = entry
	}

	// Exactly the cookbooks whose flag the fixture set are unlocked, and every
	// one of them is the cookbook the stored catalog maps that flag to.
	unlocked := 0
	for _, entry := range result.Cookbooks {
		if entry.Unlocked {
			unlocked++
		}
	}
	if unlocked != len(getCookbooksSetFlags) {
		t.Errorf("unlocked count = %d, want %d", unlocked, len(getCookbooksSetFlags))
	}
	for id, name := range getCookbooksSetFlags {
		entry, exists := byName[name]
		if !exists {
			t.Fatalf("cookbook %q of flag %d is missing from the result", name, id)
		}
		if !entry.Unlocked {
			t.Errorf("cookbook %q of set flag %d is locked", name, id)
		}
	}
	for id, name := range getCookbooksClearFlags {
		entry, exists := byName[name]
		if !exists {
			t.Fatalf("cookbook %q of flag %d is missing from the result", name, id)
		}
		if entry.Unlocked {
			t.Errorf("cookbook %q of clear flag %d is unlocked", name, id)
		}
	}
}

func TestGetCookbooksReportsAResidualSlotAsLocked(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, false)

	result, err := GetCookbooks(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot, "")
	if err != nil {
		t.Fatalf("GetCookbooks: %v", err)
	}

	if result.Active {
		t.Error("active = true, want false")
	}
	// The bitfield of the deleted character is still written into the fixture, so
	// a fully locked result proves it was never located or read.
	if len(result.Cookbooks) != getCookbooksCatalogCount {
		t.Fatalf("cookbook count = %d, want %d", len(result.Cookbooks), getCookbooksCatalogCount)
	}
	for _, entry := range result.Cookbooks {
		if entry.Unlocked {
			t.Fatalf("cookbook %q of an inactive slot is unlocked", entry.Name)
		}
	}
}

func TestGetCookbooksOrdersByCategoryThenNameThenKey(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)

	// Two resources are given the same category and name, so the resource key is
	// the only remaining tie-breaker. The stored documents never collide on both
	// values, so the tie is created here on purpose.
	resources := patchCookbookUnlock(t, storedCookbookResources(t), getCookbooksSecondKey,
		func(unlock *schema.ItemUnlock) {
			unlock.Name.Value = getCookbooksSetFlags[67000]
		})

	result, err := GetCookbooks(engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot, "")
	if err != nil {
		t.Fatalf("GetCookbooks: %v", err)
	}

	for index := 1; index < len(result.Cookbooks); index++ {
		previous, current := result.Cookbooks[index-1], result.Cookbooks[index]
		switch {
		case previous.Category != current.Category:
			if previous.Category > current.Category {
				t.Fatalf("entry %d category %q follows %q", index, current.Category, previous.Category)
			}
		case previous.Name != current.Name:
			if previous.Name > current.Name {
				t.Fatalf("entry %d name %q follows %q", index, current.Name, previous.Name)
			}
		case previous.Key >= current.Key:
			t.Fatalf("entry %d key %q follows %q", index, current.Key, previous.Key)
		}
	}

	// The two colliding entries must be adjacent and ordered by resource key.
	tied := make([]string, 0, 2)
	for _, entry := range result.Cookbooks {
		if entry.Name == getCookbooksSetFlags[67000] {
			tied = append(tied, entry.Key)
		}
	}
	if len(tied) != 2 || tied[0] != getCookbooksFirstKey || tied[1] != getCookbooksSecondKey {
		t.Errorf("tied keys = %v, want [%s %s]", tied, getCookbooksFirstKey, getCookbooksSecondKey)
	}
}

func TestGetCookbooksFiltersByAvailability(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	full, err := GetCookbooks(engine, gameCatalog, sessionID, getCookbooksSlot, "")
	if err != nil {
		t.Fatalf("GetCookbooks: %v", err)
	}

	cases := map[string]struct {
		filter string
		want   int
	}{
		"every cookbook": {"", getCookbooksCatalogCount},
		"unlocked only":  {CookbookAvailabilityUnlocked, len(getCookbooksSetFlags)},
		"locked only":    {CookbookAvailabilityLocked, getCookbooksCatalogCount - len(getCookbooksSetFlags)},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := GetCookbooks(engine, gameCatalog, sessionID, getCookbooksSlot, testCase.filter)
			if err != nil {
				t.Fatalf("GetCookbooks: %v", err)
			}
			if len(result.Cookbooks) != testCase.want {
				t.Fatalf("cookbook count = %d, want %d", len(result.Cookbooks), testCase.want)
			}
			if result.Cookbooks == nil {
				t.Fatal("cookbooks is nil, want an empty list")
			}

			// A filter may only remove entries: the ones it keeps stay in the order
			// and with the state the unfiltered result gave them.
			position := 0
			for _, entry := range full.Cookbooks {
				if testCase.filter == CookbookAvailabilityUnlocked && !entry.Unlocked {
					continue
				}
				if testCase.filter == CookbookAvailabilityLocked && entry.Unlocked {
					continue
				}
				if result.Cookbooks[position] != entry {
					t.Fatalf("entry %d = %+v, want %+v", position, result.Cookbooks[position], entry)
				}
				position++
			}
			if position != len(result.Cookbooks) {
				t.Errorf("matched %d entries, want %d", position, len(result.Cookbooks))
			}
		})
	}
}

func TestGetCookbooksRejectsAnUnknownAvailabilityFilter(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	// The value is matched exactly and case-sensitively and is never trimmed, so
	// a padded, case-shifted or aliased value is an error, not the filter it
	// resembles.
	for _, filter := range []string{"Unlocked", " unlocked", "unlocked ", "LOCKED", "all", "true"} {
		t.Run(strconv.Quote(filter), func(t *testing.T) {
			result, err := GetCookbooks(engine, gameCatalog, sessionID, getCookbooksSlot, filter)
			if err == nil {
				t.Fatalf("GetCookbooks accepted %q", filter)
			}
			want := `availabilityFilter must be "unlocked", "locked" or empty; got ` + strconv.Quote(filter)
			if err.Error() != want {
				t.Errorf("error = %q, want %q", err, want)
			}
			if len(result.Cookbooks) != 0 || result.Active {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

func TestGetCookbooksRejectsInvalidSessionAndCharacter(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	closed, err := engine.LoadSave(writeGetCookbooksFixture(t, getCookbooksSetFlags, true), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if err := engine.CloseSession(closed.SaveSessionID); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}

	cases := map[string]struct {
		saveSessionID string
		characterID   int
		want          string
	}{
		"empty session":   {"", getCookbooksSlot, "saveSessionID is required"},
		"unknown session": {"missing", getCookbooksSlot, `unknown save session "missing"`},
		"closed session": {closed.SaveSessionID, getCookbooksSlot,
			"unknown save session " + strconv.Quote(closed.SaveSessionID)},
		"characterID -1": {sessionID, -1, "characterID -1 is outside the range 0..9"},
		"characterID 10": {sessionID, 10, "characterID 10 is outside the range 0..9"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := GetCookbooks(
				engine, gameCatalog, testCase.saveSessionID, testCase.characterID, "")
			if err == nil {
				t.Fatalf("GetCookbooks accepted %s", name)
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if len(result.Cookbooks) != 0 || result.Active {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

func TestGetCookbooksRejectsIncompleteCatalogData(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)

	cases := map[string]struct {
		change func(*schema.ItemUnlock)
		want   string
	}{
		"missing name": {
			func(unlock *schema.ItemUnlock) { unlock.Name = schema.Fact[string]{} },
			`cookbook "40002454" unlock 0 has no known name`,
		},
		"missing category": {
			func(unlock *schema.ItemUnlock) { unlock.Category = schema.Fact[string]{} },
			`cookbook "40002454" unlock 0 has no known category`,
		},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			resources := patchCookbookUnlock(
				t, storedCookbookResources(t), getCookbooksFirstKey, testCase.change)

			result, err := GetCookbooks(
				engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot, "")
			if err == nil {
				t.Fatalf("GetCookbooks accepted %s", name)
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if len(result.Cookbooks) != 0 || result.Active {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

func TestGetCookbooksRejectsADuplicateEventFlagID(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)

	// Two cookbooks that claim one event flag are a conflict, not a state one of
	// them may win: neither is dropped, renamed or preferred.
	resources := patchCookbookUnlock(t, storedCookbookResources(t), getCookbooksSecondKey,
		func(unlock *schema.ItemUnlock) { unlock.EventFlagID.Value = 67000 })

	result, err := GetCookbooks(
		engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot, "")
	if err == nil {
		t.Fatal("GetCookbooks accepted a duplicate event flag")
	}
	want := `cookbooks "40002454" and "40002455" both declare event flag 67000`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
	if len(result.Cookbooks) != 0 || result.Active {
		t.Errorf("result = %+v, want the zero value", result)
	}
}

func TestGetCookbooksRejectsAFlagFromAnotherSupportedDomain(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)

	resources := patchCookbookUnlock(t, storedCookbookResources(t), getCookbooksFirstKey,
		func(unlock *schema.ItemUnlock) { unlock.EventFlagID.Value = 60130 })

	result, err := GetCookbooks(
		engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot, "")
	if err == nil {
		t.Fatal("GetCookbooks accepted a whetblade event flag for a cookbook")
	}
	want := "event flag 60130 lies in block 60, which this reader does not support"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
	if len(result.Cookbooks) != 0 || result.Active {
		t.Errorf("result = %+v, want the zero value", result)
	}
}

func TestGetCookbooksRejectsACookbookUnlockOutsideTheGoodsFamily(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)

	// A cookbook unlock in a resource of another family is a catalog conflict, not
	// a cookbook to report and not an entry to drop silently. The unlock is a copy
	// of a stored one, so it carries the provenance the catalog requires and its
	// own event flag: only the family of the declaring resource can reject it.
	resources := storedCookbookResources(t)
	unlock := storedCookbookUnlock(t, resources, getCookbooksFirstKey)
	unlock.EventFlagID.Value = 67999
	resources = patchCookbookDocument(t, resources, getCookbooksWeaponKey,
		func(document *schema.ItemDocument) {
			document.Unlocks = append(document.Unlocks, unlock)
		})

	result, err := GetCookbooks(
		engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot, "")
	if err == nil {
		t.Fatal("GetCookbooks accepted a cookbook unlock outside the goods family")
	}
	want := `cookbook "0001ADB0" has item family "weapon", want "goods"`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
	if len(result.Cookbooks) != 0 || result.Active {
		t.Errorf("result = %+v, want the zero value", result)
	}
}

func TestGetCookbooksRejectsASecondCookbookUnlockInOneResource(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)

	// The second unlock is a copy of the stored one, so it keeps the provenance the
	// catalog requires and is free of every other conflict — its own event flag,
	// name and category. Only the count can reject it. Two entries for one resource
	// would repeat the public kind and key identity.
	resources := storedCookbookResources(t)
	second := storedCookbookUnlock(t, resources, getCookbooksFirstKey)
	second.EventFlagID.Value = 67999
	second.Name.Value = "Second Cookbook"
	second.Category.Value = "Second Cookbook Series"
	resources = patchCookbookDocument(t, resources, getCookbooksFirstKey,
		func(document *schema.ItemDocument) {
			document.Unlocks = append(document.Unlocks, second)
		})

	result, err := GetCookbooks(
		engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot, "")
	if err == nil {
		t.Fatal("GetCookbooks accepted two cookbook unlocks in one resource")
	}
	want := `cookbook "40002454" declares 2 cookbook unlocks, want exactly one`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
	if len(result.Cookbooks) != 0 || result.Active {
		t.Errorf("result = %+v, want the zero value", result)
	}
}

func TestGetCookbooksReportsOneEntryPerResourceWithAUniqueIdentity(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)

	result, err := GetCookbooks(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot, "")
	if err != nil {
		t.Fatalf("GetCookbooks: %v", err)
	}
	if len(result.Cookbooks) != getCookbooksCatalogCount {
		t.Fatalf("cookbook count = %d, want %d", len(result.Cookbooks), getCookbooksCatalogCount)
	}

	// One resource is one entry, so the public kind and key identity never repeats
	// and the entry count equals the number of declaring resources.
	identities := make(map[string]bool, len(result.Cookbooks))
	for _, entry := range result.Cookbooks {
		identity := string(entry.Kind) + "/" + entry.Key
		if identities[identity] {
			t.Errorf("identity %q appears twice", identity)
		}
		identities[identity] = true
	}
}

func TestGetCookbooksLeavesTheSaveFileUntouched(t *testing.T) {
	path := writeGetCookbooksFixture(t, getCookbooksSetFlags, true)

	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	gameCatalog := newCookbooksCatalog(t)
	// Both an active and an inactive slot, so neither the located read nor the
	// activity-only path can touch the source file.
	for _, characterID := range []int{getCookbooksSlot, getCookbooksSlot + 1} {
		if _, err := GetCookbooks(
			engine, gameCatalog, loaded.SaveSessionID, characterID, ""); err != nil {
			t.Fatalf("GetCookbooks(%d): %v", characterID, err)
		}
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reread fixture: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Error("the save file changed, want it untouched")
	}
	if info, err := engine.GetSessionInfo(loaded.SaveSessionID); err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	} else if info.UnsavedChanges {
		t.Error("unsavedChanges = true, want a read to leave the session clean")
	}
}
