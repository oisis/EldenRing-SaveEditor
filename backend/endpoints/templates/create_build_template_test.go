package templates_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"unicode/utf16"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/templates"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	testFixtureHeaderSize       = 0x300
	testFixtureEntryCountOffset = 0x0C
	testFixtureEntryCount       = 12
	testFixtureSlotBlockSize    = 0x280010
	testFixtureSize             = int64(testFixtureHeaderSize) + 10*testFixtureSlotBlockSize + 0x60010

	testFixtureUserData10Offset = int64(testFixtureHeaderSize) + 10*testFixtureSlotBlockSize + 0x10
	testFixtureFlagsOffset      = 0x1954
	testFixtureSummaryOffset    = 0x195E
	testFixtureSummaryStride    = 0x24C
	testFixtureLevelOffset      = 0x22
	testFixtureSecondsOffset    = 0x26
	testFixtureGenderOffset     = 0x242
	testFixtureClassOffset      = 0x243

	testSlotActive   = 2
	testSlotInactive = 5

	testAnchorAt                = 0x0640
	testSectionSpellsAt         = 0x9205
	testCountAt                 = 0x931D
	testProjectileCount         = 17
	testStatsVigorOffset        = -379
	testStatsMindOffset         = -375
	testStatsEnduranceOffset    = -371
	testStatsStrengthOffset     = -367
	testStatsDexterityOffset    = -363
	testStatsIntelligenceOffset = -359
	testStatsFaithOffset        = -355
	testStatsArcaneOffset       = -351
	testStatsLevelOffset        = -335
	testInventoryAt             = 505
	testMemoryStones            = 2
	testMemoryStoneHandle       = 0xB000272E
)

var testAnchor = []byte{
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

func writeTestSaveFixture(t *testing.T) (string, []byte) {
	t.Helper()
	data := make([]byte, testFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[testFixtureEntryCountOffset:], testFixtureEntryCount)

	// Set active flag for slot 2
	data[testFixtureUserData10Offset+testFixtureFlagsOffset+testSlotActive] = 1

	// Summary for slot 2
	summary := testFixtureUserData10Offset + testFixtureSummaryOffset + int64(testSlotActive)*testFixtureSummaryStride
	for index, unit := range utf16.Encode([]rune("Tarnished Hero")) {
		binary.LittleEndian.PutUint16(data[summary+int64(index)*2:], unit)
	}
	binary.LittleEndian.PutUint32(data[summary+testFixtureLevelOffset:], 125)
	binary.LittleEndian.PutUint32(data[summary+testFixtureSecondsOffset:], 50000)
	data[summary+testFixtureGenderOffset] = 1
	data[summary+testFixtureClassOffset] = 1

	slotBase := int64(testFixtureHeaderSize) + 0x10 + int64(testSlotActive)*testFixtureSlotBlockSize
	anchorBase := slotBase + testAnchorAt
	copy(data[anchorBase:], testAnchor)

	// Stats block backwards from anchor
	binary.LittleEndian.PutUint32(data[anchorBase+testStatsVigorOffset:], 50)
	binary.LittleEndian.PutUint32(data[anchorBase+testStatsMindOffset:], 25)
	binary.LittleEndian.PutUint32(data[anchorBase+testStatsEnduranceOffset:], 30)
	binary.LittleEndian.PutUint32(data[anchorBase+testStatsStrengthOffset:], 40)
	binary.LittleEndian.PutUint32(data[anchorBase+testStatsDexterityOffset:], 20)
	binary.LittleEndian.PutUint32(data[anchorBase+testStatsIntelligenceOffset:], 15)
	binary.LittleEndian.PutUint32(data[anchorBase+testStatsFaithOffset:], 12)
	binary.LittleEndian.PutUint32(data[anchorBase+testStatsArcaneOffset:], 10)
	binary.LittleEndian.PutUint32(data[anchorBase+testStatsLevelOffset:], 125)

	// Equipped spells (first slot Glintstone Pebble 0x0FA0, second empty)
	at0 := anchorBase + testSectionSpellsAt
	binary.LittleEndian.PutUint32(data[at0:], 0x0FA0)
	binary.LittleEndian.PutUint32(data[at0+4:], 0xFFFFFFFF)
	for i := 1; i < 14; i++ {
		at := anchorBase + testSectionSpellsAt + int64(i)*8
		binary.LittleEndian.PutUint32(data[at:], 0xFFFFFFFF)
		binary.LittleEndian.PutUint32(data[at+4:], 0x00000000)
	}

	binary.LittleEndian.PutUint32(data[anchorBase+testCountAt:], testProjectileCount)

	// One Memory Stone stack in inventory (base 2 + 2 stones = exactly 4 available slots; tested loadout consumes exactly 4 slots)
	stoneAt := anchorBase + testInventoryAt + 97*12
	binary.LittleEndian.PutUint32(data[stoneAt:], testMemoryStoneHandle)
	binary.LittleEndian.PutUint32(data[stoneAt+4:], testMemoryStones)

	path := filepath.Join(t.TempDir(), "test_save.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path, data
}

var (
	testCatalogOnce sync.Once
	testCatalogData loader.Data
	testCatalogErr  error
)

func newTestCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()
	testCatalogOnce.Do(func() {
		testCatalogData, testCatalogErr = loader.LoadFS(catalogdata.Files())
	})
	if testCatalogErr != nil {
		t.Fatalf("loader.LoadFS: %v", testCatalogErr)
	}
	cat, err := gamecatalog.New(testCatalogData.Manifest, testCatalogData.Resources())
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	return cat
}

func TestCreateBuildTemplate_Success(t *testing.T) {
	engine := saveengine.New()
	savePath, rawBefore := writeTestSaveFixture(t)
	loaded, err := engine.LoadSave(savePath, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := newTestCatalog(t)
	storeDir := filepath.Join(t.TempDir(), "templates")
	store := buildtemplates.NewStore(storeDir)

	req := templates.CreateBuildTemplateRequest{
		SaveSessionID:     loaded.SaveSessionID,
		SourceCharacterID: testSlotActive,
		Name:              "Sorcerer STR Hybrid",
		Description:       "Test build template description",
		Tags:              []string{"pvp", "str", "sorcery"},
		Selection: buildtemplates.TemplateSelection{
			Profile: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"name": true, "level": true},
			},
			Stats: &buildtemplates.SectionSelection{
				All: true,
			},
			Spells: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"spell1": true},
			},
		},
	}

	res, err := templates.CreateBuildTemplate(store, engine, catalog, "2.0.0", req)
	if err != nil {
		t.Fatalf("CreateBuildTemplate: %v", err)
	}
	if res.TemplateID == "" {
		t.Fatal("expected non-empty TemplateID")
	}
	if res.TemplateRevision != "1" {
		t.Fatalf("TemplateRevision = %q, want %q", res.TemplateRevision, "1")
	}

	// Verify the saved template document in Store
	tpl, rev, err := store.GetTemplate(res.TemplateID)
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if rev != "1" {
		t.Errorf("rev = %q, want %q", rev, "1")
	}
	if tpl.Metadata == nil || tpl.Metadata.Name != "Sorcerer STR Hybrid" {
		t.Errorf("tpl.Metadata.Name = %v", tpl.Metadata)
	}
	if tpl.Sections.Profile == nil || *tpl.Sections.Profile.Name != "Tarnished Hero" || *tpl.Sections.Profile.Level != 125 {
		t.Errorf("tpl.Sections.Profile = %+v", tpl.Sections.Profile)
	}
	if tpl.Sections.Stats == nil || *tpl.Sections.Stats.Vigor != 50 || *tpl.Sections.Stats.Strength != 40 {
		t.Errorf("tpl.Sections.Stats = %+v", tpl.Sections.Stats)
	}
	if tpl.Sections.Spells == nil || tpl.Sections.Spells.Spell1 == nil || tpl.Sections.Spells.Spell1.BaseItemID != 0x40000FA0 {
		t.Errorf("tpl.Sections.Spells = %+v", tpl.Sections.Spells)
	}

	// Verify save state was untouched
	undo, err := engine.GetUndoState(loaded.SaveSessionID, testSlotActive)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if undo.SaveRevision != "0" {
		t.Errorf("SaveRevision mutated to %q", undo.SaveRevision)
	}
	rawAfter, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatalf("ReadFile save: %v", err)
	}
	if !bytes.Equal(rawBefore, rawAfter) {
		t.Error("save file on disk was modified by CreateBuildTemplate")
	}
}

func TestCreateBuildTemplate_RejectsInvalidSelections(t *testing.T) {
	engine := saveengine.New()
	savePath, _ := writeTestSaveFixture(t)
	loaded, err := engine.LoadSave(savePath, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := newTestCatalog(t)
	store := buildtemplates.NewStore(t.TempDir())

	tests := []struct {
		name string
		req  templates.CreateBuildTemplateRequest
	}{
		{
			name: "empty name",
			req: templates.CreateBuildTemplateRequest{
				SaveSessionID:     loaded.SaveSessionID,
				SourceCharacterID: testSlotActive,
				Name:              "",
				Selection: buildtemplates.TemplateSelection{
					Stats: &buildtemplates.SectionSelection{All: true},
				},
			},
		},
		{
			name: "empty selection",
			req: templates.CreateBuildTemplateRequest{
				SaveSessionID:     loaded.SaveSessionID,
				SourceCharacterID: testSlotActive,
				Name:              "No Selection",
				Selection:         buildtemplates.TemplateSelection{},
			},
		},
		{
			name: "inactive character slot",
			req: templates.CreateBuildTemplateRequest{
				SaveSessionID:     loaded.SaveSessionID,
				SourceCharacterID: testSlotInactive,
				Name:              "Inactive",
				Selection: buildtemplates.TemplateSelection{
					Stats: &buildtemplates.SectionSelection{All: true},
				},
			},
		},
		{
			name: "profile.All shortcut is rejected",
			req: templates.CreateBuildTemplateRequest{
				SaveSessionID:     loaded.SaveSessionID,
				SourceCharacterID: testSlotActive,
				Name:              "Profile All",
				Selection: buildtemplates.TemplateSelection{
					Profile: &buildtemplates.SectionSelection{All: true},
				},
			},
		},
		{
			name: "unsupported profile field runes",
			req: templates.CreateBuildTemplateRequest{
				SaveSessionID:     loaded.SaveSessionID,
				SourceCharacterID: testSlotActive,
				Name:              "Profile Runes",
				Selection: buildtemplates.TemplateSelection{
					Profile: &buildtemplates.SectionSelection{
						Fields: map[string]bool{"runes": true},
					},
				},
			},
		},
		{
			name: "unsupported section equipment",
			req: templates.CreateBuildTemplateRequest{
				SaveSessionID:     loaded.SaveSessionID,
				SourceCharacterID: testSlotActive,
				Name:              "Equipment Section",
				Selection: buildtemplates.TemplateSelection{
					Equipment: &buildtemplates.SectionSelection{All: true},
				},
			},
		},
		{
			name: "unsupported spell slot 13",
			req: templates.CreateBuildTemplateRequest{
				SaveSessionID:     loaded.SaveSessionID,
				SourceCharacterID: testSlotActive,
				Name:              "Spell 13",
				Selection: buildtemplates.TemplateSelection{
					Spells: &buildtemplates.SectionSelection{
						Fields: map[string]bool{"spell13": true},
					},
				},
			},
		},
		{
			name: "spells.All shortcut is rejected",
			req: templates.CreateBuildTemplateRequest{
				SaveSessionID:     loaded.SaveSessionID,
				SourceCharacterID: testSlotActive,
				Name:              "Spells All",
				Selection: buildtemplates.TemplateSelection{
					Spells: &buildtemplates.SectionSelection{All: true},
				},
			},
		},
		{
			name: "unsupported section items",
			req: templates.CreateBuildTemplateRequest{
				SaveSessionID:     loaded.SaveSessionID,
				SourceCharacterID: testSlotActive,
				Name:              "Items Section",
				Selection: buildtemplates.TemplateSelection{
					Items: &buildtemplates.SectionSelection{All: true},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := templates.CreateBuildTemplate(store, engine, catalog, "2.0.0", tc.req)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}

func TestCreateBuildTemplate_NilDependencies(t *testing.T) {
	store := buildtemplates.NewStore(t.TempDir())
	engine := saveengine.New()
	catalog := newTestCatalog(t)
	req := templates.CreateBuildTemplateRequest{
		SaveSessionID:     "session-1",
		SourceCharacterID: 0,
		Name:              "Test",
		Selection: buildtemplates.TemplateSelection{
			Stats: &buildtemplates.SectionSelection{All: true},
		},
	}

	if _, err := templates.CreateBuildTemplate(nil, engine, catalog, "2.0.0", req); err == nil {
		t.Error("expected error on nil store")
	}
	if _, err := templates.CreateBuildTemplate(store, nil, catalog, "2.0.0", req); err == nil {
		t.Error("expected error on nil engine")
	}
	if _, err := templates.CreateBuildTemplate(store, engine, nil, "2.0.0", req); err == nil {
		t.Error("expected error on nil catalog")
	}
}
