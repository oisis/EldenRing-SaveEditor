package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// The positions this fixture writes, restated independently of the
// implementation so a changed production layout fails these tests instead of
// silently moving with them.
const (
	validationTestSlot                = 0
	validationTestPCSlotBase          = int64(0x310)
	validationTestPCSlotStride        = int64(0x280010)
	validationTestPS4SlotBase         = int64(0x70)
	validationTestPS4SlotStride       = int64(0x280000)
	validationTestPCUserData10Base    = int64(0x19003B0)
	validationTestPS4UserData10Base   = int64(0x1900070)
	validationTestActiveFlagsOffset   = int64(0x1954)
	validationTestSlotVersion         = uint32(0x6E)
	validationTestAnchorAt            = int64(0xB000)
	validationTestClassOffset         = int64(-248)
	validationTestVigorOffset         = int64(-379)
	validationTestLevelOffset         = int64(-335)
	validationTestSoulMemoryOffset    = int64(-327)
	validationTestTalismanSlotsOffset = int64(-241)
	validationTestEquipRowsOffset     = int64(209)
	validationTestEquipHandlesOffset  = int64(413)
	validationTestInventoryCommonAt   = int64(505)
	validationTestInventoryKeyAt      = int64(505 + 0xA80*12 + 4)
	validationTestRecordSize          = int64(12)
	validationTestSpellsAt            = int64(0x9205)
	validationTestQuickItemsAt        = int64(0x9279)
	validationTestPouchAt             = int64(0x9279 + 0x54)
	validationTestStorageCountAt      = int64(0x931D)
	validationTestStorageCommonAt     = int64(0x94FC)
	validationTestSpellEmptyID        = uint32(0xFFFFFFFF)
	validationTestSpellEmptyFollower  = uint32(0x00000000)
	validationTestSpellOccFollower    = uint32(0xFFFFFFFF)
	validationTestRowBase             = uint32(0x180)
	validationTestInvalidRow          = uint32(0xFFFFFFFF)
)

var validationTestAnchor = []byte{
	0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// validationTestVagabond is the starting attribute set of class 0. Its sum is
// 88, so the level the confirmed formula derives from it is 9.
var validationTestVagabond = [8]uint32{15, 10, 11, 14, 13, 9, 9, 7}

const validationTestVagabondLevel = uint32(9)

type validationTestRow struct {
	index       int
	handle      uint32
	rawQuantity uint32
}

type validationTestRef struct {
	slot   int
	handle uint32
	row    uint32
}

// validationTestFixture describes one synthetic save. Its zero value plus a
// platform is a known-good slot: a Vagabond at the level its attributes
// produce, with enough lifetime runes, empty containers, no reference and no
// equipped spell.
type validationTestFixture struct {
	platform        Platform
	inactive        bool
	attributes      *[8]uint32
	level           *uint32
	soulMemory      *uint32
	class           *byte
	inventoryCommon []validationTestRow
	inventoryKey    []validationTestRow
	storageCommon   []validationTestRow
	spells          map[int]uint32
	corruptSpell    bool
	quickRefs       []validationTestRef
	equipmentRefs   []validationTestRef
}

func writeValidationFixture(t *testing.T, content validationTestFixture) string {
	t.Helper()

	var data []byte
	var userData10Base, slotBase int64
	switch content.platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = validationTestPCUserData10Base
		slotBase = validationTestPCSlotBase + int64(validationTestSlot)*validationTestPCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = validationTestPS4UserData10Base
		slotBase = validationTestPS4SlotBase + int64(validationTestSlot)*validationTestPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", content.platform)
	}

	if !content.inactive {
		data[userData10Base+validationTestActiveFlagsOffset+int64(validationTestSlot)] = 1
	}
	binary.LittleEndian.PutUint32(data[slotBase:], validationTestSlotVersion)
	copy(data[slotBase+validationTestAnchorAt:], validationTestAnchor)

	anchor := slotBase + validationTestAnchorAt
	put := func(at int64, value uint32) {
		binary.LittleEndian.PutUint32(data[anchor+at:], value)
	}
	putRow := func(sectionAt int64, row validationTestRow) {
		record := sectionAt + int64(row.index)*validationTestRecordSize
		put(record, row.handle)
		put(record+4, row.rawQuantity)
	}

	attributes := validationTestVagabond
	if content.attributes != nil {
		attributes = *content.attributes
	}
	for index, value := range attributes {
		put(validationTestVigorOffset+int64(index)*4, value)
	}

	level := validationTestVagabondLevel
	if content.level != nil {
		level = *content.level
	}
	put(validationTestLevelOffset, level)

	// The default is far above what level 9 requires, so a known-good fixture
	// never trips the lifetime-runes rule by accident.
	soulMemory := uint32(100_000)
	if content.soulMemory != nil {
		soulMemory = *content.soulMemory
	}
	put(validationTestSoulMemoryOffset, soulMemory)

	if content.class != nil {
		data[anchor+validationTestClassOffset] = *content.class
	}
	data[anchor+validationTestTalismanSlotsOffset] = 0

	for _, row := range content.inventoryCommon {
		putRow(validationTestInventoryCommonAt, row)
	}
	for _, row := range content.inventoryKey {
		putRow(validationTestInventoryKeyAt, row)
	}
	for _, row := range content.storageCommon {
		putRow(validationTestStorageCommonAt, row)
	}
	put(validationTestStorageCountAt, 0)

	for index := 0; index < 14; index++ {
		put(validationTestSpellsAt+int64(index*8), validationTestSpellEmptyID)
		put(validationTestSpellsAt+int64(index*8+4), validationTestSpellEmptyFollower)
	}
	for index, raw := range content.spells {
		put(validationTestSpellsAt+int64(index*8), raw)
		put(validationTestSpellsAt+int64(index*8+4), validationTestSpellOccFollower)
	}
	if content.corruptSpell {
		// A stored identifier with the empty follower is neither of the two
		// pairs the game writes.
		put(validationTestSpellsAt, 0x00000FA0)
		put(validationTestSpellsAt+4, validationTestSpellEmptyFollower)
	}

	// Every reference slot starts as the native "references nothing" pair.
	for slot := 0; slot < 22; slot++ {
		put(validationTestEquipHandlesOffset+int64(slot)*4, 0)
		put(validationTestEquipRowsOffset+int64(slot)*4, validationTestInvalidRow)
	}
	for slot := 0; slot < 10; slot++ {
		put(validationTestQuickItemsAt+int64(slot)*8, 0)
		put(validationTestQuickItemsAt+int64(slot)*8+4, validationTestInvalidRow)
	}
	for slot := 0; slot < 6; slot++ {
		put(validationTestPouchAt+int64(slot)*8, 0)
		put(validationTestPouchAt+int64(slot)*8+4, validationTestInvalidRow)
	}
	for _, reference := range content.equipmentRefs {
		put(validationTestEquipHandlesOffset+int64(reference.slot)*4, reference.handle)
		put(validationTestEquipRowsOffset+int64(reference.slot)*4, reference.row)
	}
	for _, reference := range content.quickRefs {
		put(validationTestQuickItemsAt+int64(reference.slot)*8, reference.handle)
		put(validationTestQuickItemsAt+int64(reference.slot)*8+4, reference.row)
	}

	path := filepath.Join(t.TempDir(), "validation.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadValidationFixture(t *testing.T, content validationTestFixture) (*Engine, string) {
	t.Helper()

	path := writeValidationFixture(t, content)
	engine := New()
	session, err := engine.LoadSave(path, string(content.platform))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, session.SaveSessionID
}

// TestGetSaveValidationFacts_KnownGoodSlotReportsNoDefect is the regression that
// protects known-good native data: a slot that satisfies every confirmed rule
// must produce no failure, no dangling reference and no rule violation on both
// platforms.
func TestGetSaveValidationFacts_KnownGoodSlotReportsNoDefect(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine, session := loadValidationFixture(t, validationTestFixture{platform: platform})

			facts, err := engine.GetSaveValidationFacts(session, validationTestSlot)
			if err != nil {
				t.Fatalf("GetSaveValidationFacts: %v", err)
			}
			if !facts.Active {
				t.Fatal("Active = false, want true")
			}
			for name, failure := range map[string]string{
				"inventory": facts.InventoryFailure,
				"storage":   facts.StorageFailure,
				"stats":     facts.StatsFailure,
				"equipment": facts.EquipmentFailure,
				"spells":    facts.SpellsFailure,
			} {
				if failure != "" {
					t.Errorf("%s failure = %q, want none", name, failure)
				}
			}
			if len(facts.Items) != 0 {
				t.Errorf("Items = %d, want 0", len(facts.Items))
			}
			if len(facts.DanglingReferences) != 0 {
				t.Errorf("DanglingReferences = %v, want none", facts.DanglingReferences)
			}
			if facts.Stats.LevelError != "" || facts.Stats.ClassMinimumError != "" {
				t.Errorf("stats errors = %q / %q, want none",
					facts.Stats.LevelError, facts.Stats.ClassMinimumError)
			}
			if facts.Stats.StoredLevel != facts.Stats.ExpectedLevel {
				t.Errorf("StoredLevel = %d, ExpectedLevel = %d, want equal",
					facts.Stats.StoredLevel, facts.Stats.ExpectedLevel)
			}
			if facts.Stats.StoredSoulMemory < facts.Stats.MinimumSoulMemory {
				t.Errorf("StoredSoulMemory = %d, MinimumSoulMemory = %d, want at least equal",
					facts.Stats.StoredSoulMemory, facts.Stats.MinimumSoulMemory)
			}
			if facts.AvailableMemorySlots != spellBaseMemorySlots {
				t.Errorf("AvailableMemorySlots = %d, want %d",
					facts.AvailableMemorySlots, spellBaseMemorySlots)
			}
			for index, raw := range facts.Spells {
				if raw != validationTestSpellEmptyID {
					t.Errorf("spell %d = 0x%08X, want the empty sentinel", index, raw)
				}
			}
		})
	}
}

func TestGetSaveValidationFacts_InactiveSlotIsNeverJudged(t *testing.T) {
	engine, session := loadValidationFixture(t, validationTestFixture{
		platform: PlatformPC,
		inactive: true,
	})

	facts, err := engine.GetSaveValidationFacts(session, validationTestSlot)
	if err != nil {
		t.Fatalf("GetSaveValidationFacts: %v", err)
	}
	if facts.Active {
		t.Fatal("Active = true, want false")
	}
	if len(facts.Items) != 0 || len(facts.DanglingReferences) != 0 {
		t.Errorf("inactive slot produced facts: %d items, %d references",
			len(facts.Items), len(facts.DanglingReferences))
	}
	if facts.Stats != (ValidationStats{}) {
		t.Errorf("Stats = %+v, want the zero value", facts.Stats)
	}
	if facts.InventoryFailure != "" || facts.SpellsFailure != "" {
		t.Errorf("inactive slot reported failures %q / %q, want none",
			facts.InventoryFailure, facts.SpellsFailure)
	}
}

// TestGetSaveValidationFacts_UnresolvedHandleStaysVisible protects the one
// difference to ResolveGaItemIDs: a handle without a GaItem record must be
// reported as one unresolved record instead of failing the whole pass.
func TestGetSaveValidationFacts_UnresolvedHandleStaysVisible(t *testing.T) {
	engine, session := loadValidationFixture(t, validationTestFixture{
		platform: PlatformPC,
		inventoryCommon: []validationTestRow{
			{index: 0, handle: 0xB0000FA0, rawQuantity: 0x80000003},
			{index: 1, handle: 0x80000001, rawQuantity: 1},
		},
		storageCommon: []validationTestRow{
			{index: 0, handle: 0xA0000042, rawQuantity: 1},
		},
	})

	facts, err := engine.GetSaveValidationFacts(session, validationTestSlot)
	if err != nil {
		t.Fatalf("GetSaveValidationFacts: %v", err)
	}
	if facts.InventoryFailure != "" || facts.StorageFailure != "" {
		t.Fatalf("failures = %q / %q, want none", facts.InventoryFailure, facts.StorageFailure)
	}
	if len(facts.Items) != 3 {
		t.Fatalf("Items = %d, want 3", len(facts.Items))
	}

	goods := facts.Items[0]
	if !goods.Resolved || goods.GameID != 0x40000FA0 {
		t.Errorf("goods record = %+v, want resolved game ID 0x40000FA0", goods)
	}
	if goods.Quantity != 3 {
		t.Errorf("goods quantity = %d, want 3 with the high bit masked off", goods.Quantity)
	}
	if goods.Container != "inventory" || goods.OwnedItemID == "" {
		t.Errorf("goods record = %+v, want the inventory container and a minted identity", goods)
	}

	weapon := facts.Items[1]
	if weapon.Resolved || weapon.ResolutionError == "" {
		t.Errorf("weapon record = %+v, want unresolved with a reason", weapon)
	}
	if weapon.GameID != 0 {
		t.Errorf("weapon GameID = 0x%08X, want nothing invented for an unresolved handle", weapon.GameID)
	}

	accessory := facts.Items[2]
	if accessory.Container != "storage" || !accessory.Resolved || accessory.GameID != 0x20000042 {
		t.Errorf("storage record = %+v, want resolved game ID 0x20000042", accessory)
	}
}

func TestGetSaveValidationFacts_DanglingReferences(t *testing.T) {
	tests := []struct {
		name       string
		fixture    validationTestFixture
		wantCount  int
		wantReason string
	}{
		{
			name: "reference matching its row is consistent",
			fixture: validationTestFixture{
				platform:        PlatformPC,
				inventoryCommon: []validationTestRow{{index: 5, handle: 0xB0001111, rawQuantity: 1}},
				equipmentRefs: []validationTestRef{
					{slot: 1, handle: 0xB0001111, row: validationTestRowBase + 5},
				},
			},
		},
		{
			name: "reference to an empty row is dangling",
			fixture: validationTestFixture{
				platform: PlatformPC,
				equipmentRefs: []validationTestRef{
					{slot: 1, handle: 0xB0001111, row: validationTestRowBase + 5},
				},
			},
			wantCount:  1,
			wantReason: "inventory common row 5 is empty",
		},
		{
			name: "reference to a row carrying another handle is dangling",
			fixture: validationTestFixture{
				platform:        PlatformPC,
				inventoryCommon: []validationTestRow{{index: 5, handle: 0xB0002222, rawQuantity: 1}},
				quickRefs: []validationTestRef{
					{slot: 3, handle: 0xB0001111, row: validationTestRowBase + 5},
				},
			},
			wantCount:  1,
			wantReason: "inventory common row 5 carries the different handle 0xB0002222",
		},
		{
			name: "key-section record never satisfies a common-row reference",
			fixture: validationTestFixture{
				platform:     PlatformPC,
				inventoryKey: []validationTestRow{{index: 5, handle: 0xB0001111, rawQuantity: 1}},
				equipmentRefs: []validationTestRef{
					{slot: 1, handle: 0xB0001111, row: validationTestRowBase + 5},
				},
			},
			wantCount:  1,
			wantReason: "inventory common row 5 is empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine, session := loadValidationFixture(t, tc.fixture)

			facts, err := engine.GetSaveValidationFacts(session, validationTestSlot)
			if err != nil {
				t.Fatalf("GetSaveValidationFacts: %v", err)
			}
			if facts.EquipmentFailure != "" {
				t.Fatalf("EquipmentFailure = %q, want none", facts.EquipmentFailure)
			}
			if len(facts.DanglingReferences) != tc.wantCount {
				t.Fatalf("DanglingReferences = %+v, want %d", facts.DanglingReferences, tc.wantCount)
			}
			if tc.wantCount == 0 {
				return
			}
			if facts.DanglingReferences[0].Reason != tc.wantReason {
				t.Errorf("reason = %q, want %q", facts.DanglingReferences[0].Reason, tc.wantReason)
			}
		})
	}
}

func TestGetSaveValidationFacts_StatsRules(t *testing.T) {
	belowClassMinimum := validationTestVagabond
	belowClassMinimum[0] = 14 // Vagabond requires 15 Vigor.
	belowClassMinimum[1] = 11 // Keep the sum, so only the class rule breaks.

	outOfRange := validationTestVagabond
	outOfRange[0] = 0

	wrongLevel := uint32(40)
	lowSoulMemory := uint32(0)
	unknownClass := byte(200)

	// One level above the Vagabond base of 9, bought with one extra point of
	// Vigor. Unlike the untouched base build, this character did pay for a
	// level, so a stored zero is genuinely below the minimum.
	oneLevelAboveBase := validationTestVagabond
	oneLevelAboveBase[0] = 16
	oneLevelAboveBaseLevel := validationTestVagabondLevel + 1

	tests := []struct {
		name                  string
		fixture               validationTestFixture
		wantLevelError        bool
		wantClassError        bool
		wantLevelMismatch     bool
		wantBelowSoulMemory   bool
		wantExpectedLevelZero bool
	}{
		{
			name:              "stored level disagreeing with the attributes",
			fixture:           validationTestFixture{platform: PlatformPC, level: &wrongLevel},
			wantLevelMismatch: true,
		},
		{
			name:           "attribute below the starting-class minimum",
			fixture:        validationTestFixture{platform: PlatformPC, attributes: &belowClassMinimum},
			wantClassError: true,
		},
		{
			name:                  "attribute outside the absolute range",
			fixture:               validationTestFixture{platform: PlatformPC, attributes: &outOfRange},
			wantLevelError:        true,
			wantClassError:        true,
			wantExpectedLevelZero: true,
		},
		{
			// The native vanilla save stores exactly this: a freshly created
			// class sitting at its own base level with TotalGetSoul zero. The
			// minimum is counted from that base level, so nothing is owed and
			// the untouched character must not be reported.
			name:                "a freshly created class at its base level owes nothing",
			fixture:             validationTestFixture{platform: PlatformPC, soulMemory: &lowSoulMemory},
			wantBelowSoulMemory: false,
		},
		{
			name: "lifetime runes below the minimum the levels above the class base require",
			fixture: validationTestFixture{
				platform:   PlatformPC,
				attributes: &oneLevelAboveBase,
				level:      &oneLevelAboveBaseLevel,
				soulMemory: &lowSoulMemory,
			},
			wantBelowSoulMemory: true,
		},
		{
			name:           "unknown starting class carries no minima",
			fixture:        validationTestFixture{platform: PlatformPC, class: &unknownClass},
			wantClassError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine, session := loadValidationFixture(t, tc.fixture)

			facts, err := engine.GetSaveValidationFacts(session, validationTestSlot)
			if err != nil {
				t.Fatalf("GetSaveValidationFacts: %v", err)
			}
			if facts.StatsFailure != "" {
				t.Fatalf("StatsFailure = %q, want none", facts.StatsFailure)
			}
			if got := facts.Stats.LevelError != ""; got != tc.wantLevelError {
				t.Errorf("LevelError = %q, want present = %v", facts.Stats.LevelError, tc.wantLevelError)
			}
			if got := facts.Stats.ClassMinimumError != ""; got != tc.wantClassError {
				t.Errorf("ClassMinimumError = %q, want present = %v",
					facts.Stats.ClassMinimumError, tc.wantClassError)
			}
			if got := facts.Stats.StoredLevel != facts.Stats.ExpectedLevel; got != tc.wantLevelMismatch &&
				!tc.wantExpectedLevelZero {
				t.Errorf("StoredLevel = %d, ExpectedLevel = %d, want mismatch = %v",
					facts.Stats.StoredLevel, facts.Stats.ExpectedLevel, tc.wantLevelMismatch)
			}
			if got := facts.Stats.StoredSoulMemory < facts.Stats.MinimumSoulMemory; got != tc.wantBelowSoulMemory {
				t.Errorf("StoredSoulMemory = %d, MinimumSoulMemory = %d, want below = %v",
					facts.Stats.StoredSoulMemory, facts.Stats.MinimumSoulMemory, tc.wantBelowSoulMemory)
			}
			if tc.wantExpectedLevelZero && facts.Stats.ExpectedLevel != 0 {
				t.Errorf("ExpectedLevel = %d, want 0 for an illegal attribute set", facts.Stats.ExpectedLevel)
			}
		})
	}
}

func TestGetSaveValidationFacts_SpellRecords(t *testing.T) {
	engine, session := loadValidationFixture(t, validationTestFixture{
		platform: PlatformPC,
		spells:   map[int]uint32{0: 0x00000FA0, 12: 0x00000FA1},
	})

	facts, err := engine.GetSaveValidationFacts(session, validationTestSlot)
	if err != nil {
		t.Fatalf("GetSaveValidationFacts: %v", err)
	}
	if facts.SpellsFailure != "" {
		t.Fatalf("SpellsFailure = %q, want none", facts.SpellsFailure)
	}
	if facts.Spells[0] != 0x00000FA0 {
		t.Errorf("spell 0 = 0x%08X, want 0x00000FA0", facts.Spells[0])
	}
	if facts.Spells[12] != 0x00000FA1 {
		t.Errorf("spell 12 = 0x%08X, want the reserved position reported as stored", facts.Spells[12])
	}
	if facts.AvailableMemorySlots != spellBaseMemorySlots {
		t.Errorf("AvailableMemorySlots = %d, want %d", facts.AvailableMemorySlots, spellBaseMemorySlots)
	}
}

// TestGetSaveValidationFacts_CorruptScopeLeavesOthersCovered protects the
// coverage contract: a scope that cannot be decoded reports its reason and the
// remaining scopes are still judged, instead of the whole report failing.
func TestGetSaveValidationFacts_CorruptScopeLeavesOthersCovered(t *testing.T) {
	engine, session := loadValidationFixture(t, validationTestFixture{
		platform:     PlatformPC,
		corruptSpell: true,
	})

	facts, err := engine.GetSaveValidationFacts(session, validationTestSlot)
	if err != nil {
		t.Fatalf("GetSaveValidationFacts: %v", err)
	}
	if facts.SpellsFailure == "" {
		t.Fatal("SpellsFailure = empty, want the reason the spell records could not be decoded")
	}
	if facts.InventoryFailure != "" || facts.StatsFailure != "" || facts.EquipmentFailure != "" {
		t.Errorf("unrelated scopes failed: %q / %q / %q",
			facts.InventoryFailure, facts.StatsFailure, facts.EquipmentFailure)
	}
	if facts.Stats.StoredLevel != validationTestVagabondLevel {
		t.Errorf("StoredLevel = %d, want the stats scope still judged", facts.Stats.StoredLevel)
	}
}

// TestGetSaveValidationFacts_ChangesNothing proves the getter is non-mutating:
// no snapshot byte, no revision and no unsaved-changes flag may move.
func TestGetSaveValidationFacts_ChangesNothing(t *testing.T) {
	engine, session := loadValidationFixture(t, validationTestFixture{
		platform:        PlatformPC,
		inventoryCommon: []validationTestRow{{index: 0, handle: 0xB0000FA0, rawQuantity: 5}},
		spells:          map[int]uint32{0: 0x00000FA0},
		equipmentRefs: []validationTestRef{
			{slot: 1, handle: 0xB0009999, row: validationTestRowBase + 7},
		},
	})

	engine.mutex.Lock()
	loaded := engine.sessions[session]
	before := bytes.Clone(loaded.snapshot.data)
	revisionBefore := loaded.session.revisionString()
	dirtyBefore := loaded.session.dirty
	engine.mutex.Unlock()

	if _, err := engine.GetSaveValidationFacts(session, validationTestSlot); err != nil {
		t.Fatalf("GetSaveValidationFacts: %v", err)
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if !bytes.Equal(before, loaded.snapshot.data) {
		t.Error("the snapshot changed during a non-mutating validation pass")
	}
	if got := loaded.session.revisionString(); got != revisionBefore {
		t.Errorf("revision = %q, want the unchanged %q", got, revisionBefore)
	}
	if loaded.session.dirty != dirtyBefore {
		t.Errorf("hasUnsavedChanges = %v, want the unchanged %v",
			loaded.session.dirty, dirtyBefore)
	}
}

func TestGetSaveValidationFacts_RejectsUnknownInput(t *testing.T) {
	engine, session := loadValidationFixture(t, validationTestFixture{platform: PlatformPC})

	if _, err := engine.GetSaveValidationFacts("", 0); err == nil {
		t.Error("an empty saveSessionID was accepted")
	}
	if _, err := engine.GetSaveValidationFacts("unknown", 0); err == nil {
		t.Error("an unknown saveSessionID was accepted")
	}
	for _, characterID := range []int{-1, characterSlotCount} {
		if _, err := engine.GetSaveValidationFacts(session, characterID); err == nil {
			t.Errorf("characterID %d was accepted", characterID)
		}
	}
}
