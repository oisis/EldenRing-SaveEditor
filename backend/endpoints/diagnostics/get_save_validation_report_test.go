package diagnostics_test

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// The positions this fixture writes, restated independently of the
// implementation so a changed production layout fails this test instead of
// silently moving with it.
const (
	reportTestSlot              = 0
	reportTestPCSlotBase        = int64(0x310)
	reportTestPCSlotStride      = int64(0x280010)
	reportTestPS4SlotBase       = int64(0x70)
	reportTestPS4SlotStride     = int64(0x280000)
	reportTestPCUserData10Base  = int64(0x19003B0)
	reportTestPS4UserData10Base = int64(0x1900070)
	reportTestActiveFlagsOffset = int64(0x1954)
	reportTestSlotVersion       = uint32(0x6E)
	reportTestAnchorAt          = int64(0xB000)
	reportTestClassOffset       = int64(-248)
	reportTestVigorOffset       = int64(-379)
	reportTestLevelOffset       = int64(-335)
	reportTestSoulMemoryOffset  = int64(-327)
	reportTestTalismanOffset    = int64(-241)
	reportTestEquipRowsAt       = int64(209)
	reportTestEquipHandlesAt    = int64(413)
	reportTestInventoryAt       = int64(505)
	reportTestRecordSize        = int64(12)
	reportTestSpellsAt          = int64(0x9205)
	reportTestQuickItemsAt      = int64(0x9279)
	reportTestPouchAt           = int64(0x9279 + 0x54)
	reportTestStorageCountAt    = int64(0x931D)
	reportTestStorageAt         = int64(0x94FC)
	reportTestSpellEmpty        = uint32(0xFFFFFFFF)
	reportTestSpellOccFollower  = uint32(0xFFFFFFFF)
	reportTestRowBase           = uint32(0x180)
	reportTestInvalidRow        = uint32(0xFFFFFFFF)

	// The confirmed catalog documents this test resolves against. They are read
	// from the stored catalog, not invented here, so a changed limit fails the
	// test instead of being asserted against a local copy of itself.
	reportTestGoodsSmallStack = uint32(0xB000012C) // 0x4000012C: 10 per stack, 10 in the inventory.
	reportTestGoodsBigStack   = uint32(0xB00000BE) // 0x400000BE: 99 per stack, 600 in the storage.
	reportTestGoodsNoStorage  = uint32(0xB0000064) // 0x40000064: not allowed in the storage at all.
	reportTestAccessory       = uint32(0xA00003E8) // 0x200003E8: one instance per record.
	reportTestSpellRaw        = uint32(0x00000FA0) // 0x40000FA0: one memory slot.
	reportTestSpellRawB       = uint32(0x00000FA1) // 0x40000FA1: one memory slot.
	reportTestSpellRawC       = uint32(0x00000FAA) // 0x40000FAA: one memory slot.
	reportTestUnknownSpellRaw = uint32(0x00000001)
)

// The container layout of both platforms, restated here so the fixture is
// accepted by SaveEngine without sharing anything with another test file.
const (
	reportTestPCHeaderSize    = 0x300
	reportTestPCEntryCountAt  = 0x0C
	reportTestPCEntryCount    = 12
	reportTestPCFixtureSize   = int64(reportTestPCHeaderSize) + 10*reportTestPCSlotStride + 0x60010
	reportTestPS4HeaderSize   = 0x70
	reportTestPS4EntryTableAt = 0x10
	reportTestPS4EntryCount   = 12
	reportTestPS4EntryStride  = 8
	reportTestPS4FirstEntry   = 7
	reportTestPS4EntryMarker  = 0x7F7F7F7F
	reportTestPS4FixtureSize  = int64(reportTestPS4HeaderSize) + 10*reportTestPS4SlotStride + 0x60000
)

func reportTestPCHeader() []byte {
	header := make([]byte, reportTestPCHeaderSize)
	copy(header, []byte("BND4"))
	binary.LittleEndian.PutUint32(header[reportTestPCEntryCountAt:], reportTestPCEntryCount)
	return header
}

func reportTestPS4Header() []byte {
	header := make([]byte, reportTestPS4HeaderSize)
	copy(header, []byte{0xCB, 0x01, 0x9C, 0x2C})
	for entry := 0; entry < reportTestPS4EntryCount; entry++ {
		at := reportTestPS4EntryTableAt + entry*reportTestPS4EntryStride
		binary.LittleEndian.PutUint32(header[at:], uint32(reportTestPS4FirstEntry+entry))
		binary.LittleEndian.PutUint32(header[at+4:], reportTestPS4EntryMarker)
	}
	return header
}

var reportTestAnchor = []byte{
	0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// reportTestVagabond is the starting attribute set of class 0; its sum of 88
// produces level 9.
var reportTestVagabond = [8]uint32{15, 10, 11, 14, 13, 9, 9, 7}

const reportTestLevel = uint32(9)

type reportTestRow struct {
	index    int
	handle   uint32
	quantity uint32
}

type reportTestRef struct {
	slot   int
	handle uint32
	row    uint32
}

type reportTestFixture struct {
	platform saveengine.Platform
	inactive bool
	// startingClass is the raw StartingClassID written to the slot. The zero
	// value is class 0, Vagabond, whose minima reportTestVagabond restates.
	startingClass uint8
	attributes    *[8]uint32
	level         *uint32
	soulMemory    *uint32
	inventory     []reportTestRow
	storage       []reportTestRow
	spells        map[int]uint32
	references    []reportTestRef
}

func writeReportFixture(t *testing.T, content reportTestFixture) string {
	t.Helper()

	var data []byte
	var userData10Base, slotBase int64
	switch content.platform {
	case saveengine.PlatformPC:
		data = make([]byte, reportTestPCFixtureSize)
		copy(data, reportTestPCHeader())
		userData10Base = reportTestPCUserData10Base
		slotBase = reportTestPCSlotBase + int64(reportTestSlot)*reportTestPCSlotStride
	case saveengine.PlatformPS4:
		data = make([]byte, reportTestPS4FixtureSize)
		copy(data, reportTestPS4Header())
		userData10Base = reportTestPS4UserData10Base
		slotBase = reportTestPS4SlotBase + int64(reportTestSlot)*reportTestPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", content.platform)
	}

	if !content.inactive {
		data[userData10Base+reportTestActiveFlagsOffset+int64(reportTestSlot)] = 1
	}
	binary.LittleEndian.PutUint32(data[slotBase:], reportTestSlotVersion)
	copy(data[slotBase+reportTestAnchorAt:], reportTestAnchor)

	anchor := slotBase + reportTestAnchorAt
	put := func(at int64, value uint32) {
		binary.LittleEndian.PutUint32(data[anchor+at:], value)
	}
	putRow := func(sectionAt int64, row reportTestRow) {
		record := sectionAt + int64(row.index)*reportTestRecordSize
		put(record, row.handle)
		put(record+4, row.quantity)
	}

	attributes := reportTestVagabond
	if content.attributes != nil {
		attributes = *content.attributes
	}
	for index, value := range attributes {
		put(reportTestVigorOffset+int64(index)*4, value)
	}
	level := reportTestLevel
	if content.level != nil {
		level = *content.level
	}
	put(reportTestLevelOffset, level)
	soulMemory := uint32(100_000)
	if content.soulMemory != nil {
		soulMemory = *content.soulMemory
	}
	put(reportTestSoulMemoryOffset, soulMemory)
	data[anchor+reportTestClassOffset] = content.startingClass
	data[anchor+reportTestTalismanOffset] = 0

	for _, row := range content.inventory {
		putRow(reportTestInventoryAt, row)
	}
	for _, row := range content.storage {
		putRow(reportTestStorageAt, row)
	}
	put(reportTestStorageCountAt, 0)

	for index := 0; index < 14; index++ {
		put(reportTestSpellsAt+int64(index*8), reportTestSpellEmpty)
		put(reportTestSpellsAt+int64(index*8+4), 0)
	}
	for index, raw := range content.spells {
		put(reportTestSpellsAt+int64(index*8), raw)
		put(reportTestSpellsAt+int64(index*8+4), reportTestSpellOccFollower)
	}

	for slot := 0; slot < 22; slot++ {
		put(reportTestEquipHandlesAt+int64(slot)*4, 0)
		put(reportTestEquipRowsAt+int64(slot)*4, reportTestInvalidRow)
	}
	for slot := 0; slot < 10; slot++ {
		put(reportTestQuickItemsAt+int64(slot)*8, 0)
		put(reportTestQuickItemsAt+int64(slot)*8+4, reportTestInvalidRow)
	}
	for slot := 0; slot < 6; slot++ {
		put(reportTestPouchAt+int64(slot)*8, 0)
		put(reportTestPouchAt+int64(slot)*8+4, reportTestInvalidRow)
	}
	for _, reference := range content.references {
		put(reportTestEquipHandlesAt+int64(reference.slot)*4, reference.handle)
		put(reportTestEquipRowsAt+int64(reference.slot)*4, reference.row)
	}

	path := filepath.Join(t.TempDir(), "report.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadReportFixture(t *testing.T, content reportTestFixture) (*saveengine.Engine, string) {
	t.Helper()

	if content.platform == "" {
		content.platform = saveengine.PlatformPC
	}
	engine := saveengine.New()
	session, err := engine.LoadSave(writeReportFixture(t, content), string(content.platform))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, session.SaveSessionID
}

// reportTestCatalog builds a catalog from the stored catalog data, so every
// limit this test asserts against is the real document and not a local
// invention.
func reportTestCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	reportCatalogOnce.Do(func() {
		reportCatalogData, reportCatalogErr = loader.LoadFS(catalogdata.Files())
	})
	if reportCatalogErr != nil {
		t.Fatalf("loader.LoadFS: %v", reportCatalogErr)
	}
	gameCatalog, err := gamecatalog.New(reportCatalogData.Manifest, reportCatalogData.Resources())
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	return gameCatalog
}

var (
	reportCatalogOnce sync.Once
	reportCatalogData loader.Data
	reportCatalogErr  error
)

// issueCodes lists the codes of a report in the order they were emitted.
func issueCodes(result diagnostics.GetSaveValidationReportResult) []string {
	codes := make([]string, 0, len(result.Issues))
	for _, issue := range result.Issues {
		codes = append(codes, issue.Code)
	}
	return codes
}

func requireIssue(t *testing.T, result diagnostics.GetSaveValidationReportResult, code, severity string) diagnostics.SaveValidationIssue {
	t.Helper()

	for _, issue := range result.Issues {
		if issue.Code != code {
			continue
		}
		if issue.Severity != severity {
			t.Errorf("issue %q severity = %q, want %q", code, issue.Severity, severity)
		}
		return issue
	}
	t.Fatalf("issue %q is missing; got %v", code, issueCodes(result))
	return diagnostics.SaveValidationIssue{}
}

func requireNoIssue(t *testing.T, result diagnostics.GetSaveValidationReportResult, code string) {
	t.Helper()

	for _, issue := range result.Issues {
		if issue.Code == code {
			t.Fatalf("unexpected issue %q: %s", code, issue.Message)
		}
	}
}

// TestGetSaveValidationReport_KnownGoodSlotIsClean is the regression that
// protects known-good native data on both platforms: a slot satisfying every
// confirmed rule must produce no error, no warning and full coverage.
func TestGetSaveValidationReport_KnownGoodSlotIsClean(t *testing.T) {
	gameCatalog := reportTestCatalog(t)

	for _, platform := range []saveengine.Platform{saveengine.PlatformPC, saveengine.PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine, session := loadReportFixture(t, reportTestFixture{
				platform: platform,
				inventory: []reportTestRow{
					{index: 0, handle: reportTestGoodsSmallStack, quantity: 10},
					{index: 1, handle: reportTestAccessory, quantity: 1},
				},
				storage: []reportTestRow{
					{index: 0, handle: reportTestGoodsBigStack, quantity: 99},
				},
				spells: map[int]uint32{0: reportTestSpellRaw, 1: reportTestSpellRawB},
			})

			result, err := diagnostics.GetSaveValidationReport(engine, gameCatalog, session, reportTestSlot, "")
			if err != nil {
				t.Fatalf("GetSaveValidationReport: %v", err)
			}
			if !result.Active {
				t.Fatal("Active = false, want true")
			}
			if len(result.Issues) != 0 {
				t.Fatalf("issues = %v, want none: %+v", issueCodes(result), result.Issues)
			}
			if result.ErrorCount != 0 || result.WarningCount != 0 {
				t.Errorf("counts = %d errors / %d warnings, want 0 / 0", result.ErrorCount, result.WarningCount)
			}
			if len(result.Coverage) != 5 {
				t.Fatalf("Coverage = %+v, want all five scopes", result.Coverage)
			}
			for _, coverage := range result.Coverage {
				if !coverage.Checked {
					t.Errorf("scope %q was not checked: %s", coverage.Scope, coverage.Reason)
				}
				if coverage.UnresolvedRecords != 0 {
					t.Errorf("scope %q left %d records unresolved", coverage.Scope, coverage.UnresolvedRecords)
				}
			}
			if result.SaveRevision == "" {
				t.Error("SaveRevision is empty, want the revision the report describes")
			}
		})
	}
}

// TestGetSaveValidationReport_InactiveSlotIsReportedUnchecked protects the
// coverage contract at its most dangerous point: an empty issue list for an
// inactive slot must not read as a clean save.
func TestGetSaveValidationReport_InactiveSlotIsReportedUnchecked(t *testing.T) {
	engine, session := loadReportFixture(t, reportTestFixture{inactive: true})

	result, err := diagnostics.GetSaveValidationReport(
		engine, reportTestCatalog(t), session, reportTestSlot, "")
	if err != nil {
		t.Fatalf("GetSaveValidationReport: %v", err)
	}
	if result.Active {
		t.Fatal("Active = true, want false")
	}
	if len(result.Issues) != 0 {
		t.Errorf("issues = %v, want none", issueCodes(result))
	}
	if len(result.Coverage) != 5 {
		t.Fatalf("Coverage = %+v, want all five scopes", result.Coverage)
	}
	for _, coverage := range result.Coverage {
		if coverage.Checked {
			t.Errorf("scope %q was checked on an inactive slot", coverage.Scope)
		}
		if !strings.Contains(coverage.Reason, "not active") {
			t.Errorf("scope %q reason = %q, want the inactivity of the slot", coverage.Scope, coverage.Reason)
		}
	}
}

func TestGetSaveValidationReport_ContainerLimits(t *testing.T) {
	gameCatalog := reportTestCatalog(t)

	t.Run("quantity above the per-record stack limit", func(t *testing.T) {
		engine, session := loadReportFixture(t, reportTestFixture{
			storage: []reportTestRow{{index: 0, handle: reportTestGoodsBigStack, quantity: 100}},
		})

		result, err := diagnostics.GetSaveValidationReport(engine, gameCatalog, session, reportTestSlot, "")
		if err != nil {
			t.Fatalf("GetSaveValidationReport: %v", err)
		}
		issue := requireIssue(t, result, "quantity_above_stack_limit", "error")
		if issue.Scope != "storage" || issue.OwnedItemID == "" {
			t.Errorf("issue = %+v, want the storage scope and the record identity", issue)
		}
		requireNoIssue(t, result, "quantity_above_container_limit")
	})

	t.Run("quantity above the container total", func(t *testing.T) {
		engine, session := loadReportFixture(t, reportTestFixture{
			inventory: []reportTestRow{
				{index: 0, handle: reportTestGoodsSmallStack, quantity: 6},
				{index: 1, handle: reportTestGoodsSmallStack, quantity: 6},
			},
		})

		result, err := diagnostics.GetSaveValidationReport(engine, gameCatalog, session, reportTestSlot, "")
		if err != nil {
			t.Fatalf("GetSaveValidationReport: %v", err)
		}
		issue := requireIssue(t, result, "quantity_above_container_limit", "error")
		if !strings.Contains(issue.Message, "12") {
			t.Errorf("message = %q, want the summed quantity", issue.Message)
		}
		requireNoIssue(t, result, "quantity_above_stack_limit")
		// The container total is judged once per item, never once per record.
		if got := strings.Count(strings.Join(issueCodes(result), " "), "quantity_above_container_limit"); got != 1 {
			t.Errorf("container issues = %d, want exactly 1", got)
		}
	})

	t.Run("separate-instance record holding more than one", func(t *testing.T) {
		engine, session := loadReportFixture(t, reportTestFixture{
			inventory: []reportTestRow{{index: 0, handle: reportTestAccessory, quantity: 2}},
		})

		result, err := diagnostics.GetSaveValidationReport(engine, gameCatalog, session, reportTestSlot, "")
		if err != nil {
			t.Fatalf("GetSaveValidationReport: %v", err)
		}
		requireIssue(t, result, "quantity_above_stack_limit", "error")
	})

	t.Run("item that does not belong in the container", func(t *testing.T) {
		engine, session := loadReportFixture(t, reportTestFixture{
			storage: []reportTestRow{{index: 0, handle: reportTestGoodsNoStorage, quantity: 1}},
		})

		result, err := diagnostics.GetSaveValidationReport(engine, gameCatalog, session, reportTestSlot, "")
		if err != nil {
			t.Fatalf("GetSaveValidationReport: %v", err)
		}
		requireIssue(t, result, "item_not_allowed_in_container", "error")
	})

	t.Run("occupied record with quantity zero", func(t *testing.T) {
		engine, session := loadReportFixture(t, reportTestFixture{
			inventory: []reportTestRow{{index: 0, handle: reportTestGoodsSmallStack, quantity: 0}},
		})

		result, err := diagnostics.GetSaveValidationReport(engine, gameCatalog, session, reportTestSlot, "")
		if err != nil {
			t.Fatalf("GetSaveValidationReport: %v", err)
		}
		requireIssue(t, result, "quantity_zero", "error")
	})
}

// TestGetSaveValidationReport_UnknownDataBecomesAWarning protects the fail-safe
// rule: data this build cannot resolve is reported and counted as unresolved,
// never turned into a defect, a deletion or a repair.
func TestGetSaveValidationReport_UnknownDataBecomesAWarning(t *testing.T) {
	engine, session := loadReportFixture(t, reportTestFixture{
		inventory: []reportTestRow{
			{index: 0, handle: 0x80000001, quantity: 1}, // A weapon handle without a GaItem record.
			{index: 1, handle: 0xB000FFFE, quantity: 1}, // A goods handle GameCatalog does not know.
		},
	})

	result, err := diagnostics.GetSaveValidationReport(
		engine, reportTestCatalog(t), session, reportTestSlot, "")
	if err != nil {
		t.Fatalf("GetSaveValidationReport: %v", err)
	}
	requireIssue(t, result, "unresolved_item", "warning")
	requireIssue(t, result, "unknown_item", "warning")
	if result.ErrorCount != 0 {
		t.Errorf("ErrorCount = %d, want unknown data to raise no error", result.ErrorCount)
	}

	for _, coverage := range result.Coverage {
		if coverage.Scope != "inventory" {
			continue
		}
		if !coverage.Checked || coverage.RecordsChecked != 2 || coverage.UnresolvedRecords != 2 {
			t.Errorf("inventory coverage = %+v, want 2 records with 2 of them unresolved", coverage)
		}
	}
}

func TestGetSaveValidationReport_StatsRules(t *testing.T) {
	gameCatalog := reportTestCatalog(t)

	belowClassMinimum := reportTestVagabond
	belowClassMinimum[0] = 14
	belowClassMinimum[1] = 11

	outOfRange := reportTestVagabond
	outOfRange[0] = 0

	wrongLevel := uint32(40)
	noSoulMemory := uint32(0)

	tests := []struct {
		name    string
		fixture reportTestFixture
		want    string
		absent  string
	}{
		{
			name:    "stored level disagreeing with the attributes",
			fixture: reportTestFixture{level: &wrongLevel},
			want:    "level_mismatch",
		},
		{
			name:    "attribute below the starting-class minimum",
			fixture: reportTestFixture{attributes: &belowClassMinimum},
			want:    "attribute_below_class_minimum",
		},
		{
			name:    "attribute outside the absolute range",
			fixture: reportTestFixture{attributes: &outOfRange},
			want:    "attribute_out_of_range",
			// Without a legal attribute set there is no expected level, so the
			// report must not invent a level mismatch on top of the range error.
			absent: "level_mismatch",
		},
		{
			name:    "lifetime runes below the level minimum",
			fixture: reportTestFixture{soulMemory: &noSoulMemory},
			want:    "soul_memory_below_minimum",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine, session := loadReportFixture(t, tc.fixture)

			result, err := diagnostics.GetSaveValidationReport(
				engine, gameCatalog, session, reportTestSlot, "stats")
			if err != nil {
				t.Fatalf("GetSaveValidationReport: %v", err)
			}
			requireIssue(t, result, tc.want, "error")
			if tc.absent != "" {
				requireNoIssue(t, result, tc.absent)
			}
			if len(result.Coverage) != 1 || result.Coverage[0].Scope != "stats" {
				t.Errorf("Coverage = %+v, want the stats scope only", result.Coverage)
			}
		})
	}
}

func TestGetSaveValidationReport_DanglingEquipmentReference(t *testing.T) {
	engine, session := loadReportFixture(t, reportTestFixture{
		references: []reportTestRef{
			{slot: 1, handle: reportTestGoodsSmallStack, row: reportTestRowBase + 5},
		},
	})

	result, err := diagnostics.GetSaveValidationReport(
		engine, reportTestCatalog(t), session, reportTestSlot, "equipment")
	if err != nil {
		t.Fatalf("GetSaveValidationReport: %v", err)
	}
	issue := requireIssue(t, result, "dangling_equipment_reference", "error")
	if !strings.Contains(issue.Message, "equipment slot 1") {
		t.Errorf("message = %q, want the referencing structure and slot", issue.Message)
	}
}

func TestGetSaveValidationReport_SpellRules(t *testing.T) {
	gameCatalog := reportTestCatalog(t)

	t.Run("reserved position occupied", func(t *testing.T) {
		engine, session := loadReportFixture(t, reportTestFixture{
			spells: map[int]uint32{12: reportTestSpellRaw},
		})

		result, err := diagnostics.GetSaveValidationReport(engine, gameCatalog, session, reportTestSlot, "spells")
		if err != nil {
			t.Fatalf("GetSaveValidationReport: %v", err)
		}
		requireIssue(t, result, "reserved_spell_position_occupied", "error")
		// A reserved position costs no memory, so it must not also produce a
		// capacity error.
		requireNoIssue(t, result, "memory_slots_exceeded")
	})

	t.Run("more memory used than available", func(t *testing.T) {
		engine, session := loadReportFixture(t, reportTestFixture{
			spells: map[int]uint32{
				0: reportTestSpellRaw, 1: reportTestSpellRawB, 2: reportTestSpellRawC,
			},
		})

		result, err := diagnostics.GetSaveValidationReport(engine, gameCatalog, session, reportTestSlot, "spells")
		if err != nil {
			t.Fatalf("GetSaveValidationReport: %v", err)
		}
		requireIssue(t, result, "memory_slots_exceeded", "error")
	})

	t.Run("unresolvable spell is a warning and costs nothing", func(t *testing.T) {
		engine, session := loadReportFixture(t, reportTestFixture{
			spells: map[int]uint32{
				0: reportTestSpellRaw, 1: reportTestSpellRawB, 2: reportTestUnknownSpellRaw,
			},
		})

		result, err := diagnostics.GetSaveValidationReport(engine, gameCatalog, session, reportTestSlot, "spells")
		if err != nil {
			t.Fatalf("GetSaveValidationReport: %v", err)
		}
		requireIssue(t, result, "unresolved_equipped_spell", "warning")
		// The unresolved spell must not be costed, and the capacity must not be
		// judged from an incomplete sum.
		requireNoIssue(t, result, "memory_slots_exceeded")
		if result.ErrorCount != 0 {
			t.Errorf("ErrorCount = %d, want unknown spell data to raise no error", result.ErrorCount)
		}
	})
}

func TestGetSaveValidationReport_ScopeSelection(t *testing.T) {
	gameCatalog := reportTestCatalog(t)
	engine, session := loadReportFixture(t, reportTestFixture{
		level:     func() *uint32 { level := uint32(40); return &level }(),
		inventory: []reportTestRow{{index: 0, handle: reportTestAccessory, quantity: 2}},
	})

	t.Run("a narrowed scope reports only that scope", func(t *testing.T) {
		result, err := diagnostics.GetSaveValidationReport(engine, gameCatalog, session, reportTestSlot, "inventory")
		if err != nil {
			t.Fatalf("GetSaveValidationReport: %v", err)
		}
		if len(result.Coverage) != 1 || result.Coverage[0].Scope != "inventory" {
			t.Fatalf("Coverage = %+v, want the inventory scope only", result.Coverage)
		}
		requireIssue(t, result, "quantity_above_stack_limit", "error")
		requireNoIssue(t, result, "level_mismatch")
	})

	t.Run("the full report contains both findings", func(t *testing.T) {
		result, err := diagnostics.GetSaveValidationReport(engine, gameCatalog, session, reportTestSlot, "")
		if err != nil {
			t.Fatalf("GetSaveValidationReport: %v", err)
		}
		requireIssue(t, result, "quantity_above_stack_limit", "error")
		requireIssue(t, result, "level_mismatch", "error")
	})

	t.Run("an unknown scope is rejected", func(t *testing.T) {
		if _, err := diagnostics.GetSaveValidationReport(
			engine, gameCatalog, session, reportTestSlot, "Inventory"); err == nil {
			t.Error("the scope was normalised instead of rejected")
		}
		if _, err := diagnostics.GetSaveValidationReport(
			engine, gameCatalog, session, reportTestSlot, "world"); err == nil {
			t.Error("an unknown scope was accepted")
		}
	})
}

// TestGetSaveValidationReport_ChangesNothing proves the getter is non-mutating
// at the endpoint boundary: the session revision and its unsaved-changes state
// must survive a report that found errors.
func TestGetSaveValidationReport_ChangesNothing(t *testing.T) {
	engine, session := loadReportFixture(t, reportTestFixture{
		inventory: []reportTestRow{{index: 0, handle: reportTestAccessory, quantity: 2}},
		spells:    map[int]uint32{12: reportTestSpellRaw},
	})

	before, err := engine.GetSessionInfo(session)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}

	result, err := diagnostics.GetSaveValidationReport(
		engine, reportTestCatalog(t), session, reportTestSlot, "")
	if err != nil {
		t.Fatalf("GetSaveValidationReport: %v", err)
	}
	if result.ErrorCount == 0 {
		t.Fatal("the fixture produced no error, so this test would prove nothing")
	}

	after, err := engine.GetSessionInfo(session)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if after != before {
		t.Errorf("session info = %+v, want the unchanged %+v", after, before)
	}
	if after.UnsavedChanges {
		t.Error("UnsavedChanges = true, want a report to leave the session clean")
	}

	// A second pass over the untouched session must describe the same revision
	// and find exactly the same problems.
	repeated, err := diagnostics.GetSaveValidationReport(
		engine, reportTestCatalog(t), session, reportTestSlot, "")
	if err != nil {
		t.Fatalf("GetSaveValidationReport: %v", err)
	}
	if repeated.SaveRevision != result.SaveRevision {
		t.Errorf("SaveRevision = %q, want the unchanged %q", repeated.SaveRevision, result.SaveRevision)
	}
	if !reflect.DeepEqual(repeated, result) {
		t.Errorf("the second report differs from the first:\n%+v\n%+v", repeated, result)
	}
}

func TestGetSaveValidationReport_RejectsMissingDependencies(t *testing.T) {
	engine, session := loadReportFixture(t, reportTestFixture{})

	if _, err := diagnostics.GetSaveValidationReport(
		nil, reportTestCatalog(t), session, reportTestSlot, ""); err == nil {
		t.Error("a missing save engine was accepted")
	}
	if _, err := diagnostics.GetSaveValidationReport(
		engine, nil, session, reportTestSlot, ""); err == nil {
		t.Error("a missing game catalog was accepted")
	}
	if _, err := diagnostics.GetSaveValidationReport(
		engine, reportTestCatalog(t), "unknown", reportTestSlot, ""); err == nil {
		t.Error("an unknown save session was accepted")
	}
	if _, err := diagnostics.GetSaveValidationReport(
		engine, reportTestCatalog(t), session, 10, ""); err == nil {
		t.Error("a characterID outside the slot range was accepted")
	}
}

func TestGetSaveValidationReport_StartingClassMinima(t *testing.T) {
	gameCatalog := reportTestCatalog(t)

	classes := []struct {
		name          string
		startingClass uint8
		level         uint32
		attributes    [8]uint32
		belowAttr     [8]uint32
	}{
		{
			name:          "Confessor class 6",
			startingClass: 6,
			level:         10,
			attributes:    [8]uint32{10, 13, 10, 12, 12, 9, 14, 9},
			belowAttr:     [8]uint32{9, 13, 10, 12, 12, 9, 14, 9},
		},
		{
			name:          "Samurai class 7",
			startingClass: 7,
			level:         9,
			attributes:    [8]uint32{12, 11, 13, 12, 15, 9, 8, 8},
			belowAttr:     [8]uint32{12, 11, 13, 12, 14, 9, 8, 8},
		},
		{
			name:          "Prisoner class 8",
			startingClass: 8,
			level:         9,
			attributes:    [8]uint32{11, 12, 11, 11, 14, 14, 6, 9},
			belowAttr:     [8]uint32{11, 12, 11, 11, 14, 13, 6, 9},
		},
	}

	for _, tc := range classes {
		t.Run(tc.name+" legal base produces no issue", func(t *testing.T) {
			engine, session := loadReportFixture(t, reportTestFixture{
				startingClass: tc.startingClass,
				level:         &tc.level,
				attributes:    &tc.attributes,
			})
			result, err := diagnostics.GetSaveValidationReport(
				engine, gameCatalog, session, reportTestSlot, "stats")
			if err != nil {
				t.Fatalf("GetSaveValidationReport: %v", err)
			}
			requireNoIssue(t, result, "attribute_below_class_minimum")
			requireNoIssue(t, result, "level_mismatch")
			if result.ErrorCount != 0 {
				t.Errorf("ErrorCount = %d, want 0", result.ErrorCount)
			}
		})

		t.Run(tc.name+" below minimum produces error", func(t *testing.T) {
			engine, session := loadReportFixture(t, reportTestFixture{
				startingClass: tc.startingClass,
				attributes:    &tc.belowAttr,
			})
			result, err := diagnostics.GetSaveValidationReport(
				engine, gameCatalog, session, reportTestSlot, "stats")
			if err != nil {
				t.Fatalf("GetSaveValidationReport: %v", err)
			}
			requireIssue(t, result, "attribute_below_class_minimum", "error")
		})
	}
}
