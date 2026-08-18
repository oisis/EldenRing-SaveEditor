package templates_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/templates"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func createTestTemplate(t *testing.T, store *buildtemplates.Store, tpl *buildtemplates.BuildTemplate) (string, string) {
	t.Helper()
	id, rev, err := store.CreateTemplate(tpl)
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	return id, rev
}

func hasIssueCode(issues []templates.BuildTemplatePreviewIssue, code string) bool {
	for _, it := range issues {
		if it.Code == code {
			return true
		}
	}
	return false
}

func TestGetBuildTemplatePreview_Success(t *testing.T) {
	engine := saveengine.New()
	savePath, rawBefore := writeTestSaveFixture(t)
	loaded, err := engine.LoadSave(savePath, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := newTestCatalog(t)
	storeDir := filepath.Join(t.TempDir(), "templates")
	store := buildtemplates.NewStore(storeDir)

	vigTarget := uint32(60)
	strTarget := uint32(40)
	targetName := "Archmage"

	// Fixture attributes: Vigor:50, Mind:25, Endurance:30, Strength:40, Dex:20, Int:15, Faith:12, Arc:10
	// Target attributes with Vigor 60: sum = 60 + 25 + 30 + 40 + 20 + 15 + 12 + 10 = 212. Level = 212 - 79 = 133.
	expectedTargetLevel := uint32(133)

	tpl := &buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: "Sorcerer Preview Build",
		},
		Selection: &buildtemplates.TemplateSelection{
			Profile: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"name": true, "level": true},
			},
			Stats: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"vigor": true, "strength": true},
			},
			Spells: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"spell1": true, "spell2": true},
			},
		},
		Sections: buildtemplates.TemplateSections{
			Profile: &buildtemplates.ProfileSection{
				Name:  &targetName,
				Level: &expectedTargetLevel,
			},
			Stats: &buildtemplates.StatsSection{
				Vigor:    &vigTarget,
				Strength: &strTarget,
			},
			Spells: &buildtemplates.SpellsSection{
				Spell1: &buildtemplates.SpellSlotRef{
					BaseItemID: 0x40000FA0, // Glintstone Pebble (fixture slot 0 -> changed: false)
					Name:       "Glintstone Pebble",
				},
				Spell2: &buildtemplates.SpellSlotRef{
					BaseItemID: 0x40001068, // Comet Azur (fixture slot 1 is empty -> changed: true)
					Name:       "Comet Azur",
				},
			},
		},
	}
	tplID, _ := createTestTemplate(t, store, tpl)

	sessionInfoBefore, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	undoBefore, err := engine.GetUndoState(loaded.SaveSessionID, testSlotActive)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}

	storeEntriesBefore, err := os.ReadDir(storeDir)
	if err != nil {
		t.Fatalf("ReadDir store: %v", err)
	}
	storeFilesMapBefore := make(map[string][]byte)
	for _, entry := range storeEntriesBefore {
		content, readErr := os.ReadFile(filepath.Join(storeDir, entry.Name()))
		if readErr != nil {
			t.Fatalf("ReadFile %s: %v", entry.Name(), readErr)
		}
		storeFilesMapBefore[entry.Name()] = content
	}

	res, err := templates.GetBuildTemplatePreview(store, engine, catalog, templates.GetBuildTemplatePreviewRequest{
		SaveSessionID: loaded.SaveSessionID,
		CharacterID:   testSlotActive,
		TemplateID:    tplID,
	})
	if err != nil {
		t.Fatalf("GetBuildTemplatePreview: %v", err)
	}

	if !res.Executable {
		t.Fatalf("expected Executable=true, got false (blockingIssues: %+v)", res.BlockingIssues)
	}
	if res.TemplateID != tplID {
		t.Errorf("TemplateID = %q, want %q", res.TemplateID, tplID)
	}
	if res.TemplateRevision != "1" {
		t.Errorf("TemplateRevision = %q, want %q", res.TemplateRevision, "1")
	}
	if res.CharacterID != testSlotActive {
		t.Errorf("CharacterID = %d, want %d", res.CharacterID, testSlotActive)
	}

	// Verify Profile Plan
	if res.Plan.Profile == nil || res.Plan.Profile.Name == nil || res.Plan.Profile.Level == nil {
		t.Fatalf("Plan.Profile missing: %+v", res.Plan.Profile)
	}
	if res.Plan.Profile.Name.Current != "Tarnished Hero" || res.Plan.Profile.Name.Target != "Archmage" || !res.Plan.Profile.Name.Changed {
		t.Errorf("Plan.Profile.Name = %+v", res.Plan.Profile.Name)
	}
	if res.Plan.Profile.Level.Current != 125 || res.Plan.Profile.Level.Target != expectedTargetLevel || !res.Plan.Profile.Level.Changed {
		t.Errorf("Plan.Profile.Level = %+v", res.Plan.Profile.Level)
	}

	// Verify Stats Plan
	if res.Plan.Stats == nil || res.Plan.Stats.Vigor == nil || res.Plan.Stats.Strength == nil {
		t.Fatalf("Plan.Stats missing: %+v", res.Plan.Stats)
	}
	if res.Plan.Stats.Vigor.Current != 50 || res.Plan.Stats.Vigor.Target != vigTarget || !res.Plan.Stats.Vigor.Changed {
		t.Errorf("Plan.Stats.Vigor = %+v", res.Plan.Stats.Vigor)
	}
	if res.Plan.Stats.Strength.Current != 40 || res.Plan.Stats.Strength.Target != strTarget || res.Plan.Stats.Strength.Changed {
		t.Errorf("Plan.Stats.Strength = %+v (expected Changed=false)", res.Plan.Stats.Strength)
	}
	if res.Plan.Stats.ResultLevel != expectedTargetLevel {
		t.Errorf("Plan.Stats.ResultLevel = %d, want %d", res.Plan.Stats.ResultLevel, expectedTargetLevel)
	}
	if res.Plan.Stats.ResultSoulMemory == 0 {
		t.Errorf("Plan.Stats.ResultSoulMemory is 0")
	}

	// Verify Spells Plan
	if res.Plan.Spells == nil {
		t.Fatalf("Plan.Spells is nil")
	}
	if len(res.Plan.Spells.Slots) != 12 {
		t.Fatalf("Plan.Spells slots length = %d, want 12", len(res.Plan.Spells.Slots))
	}
	slot1 := res.Plan.Spells.Slots[0]
	if slot1.SlotNumber != 1 || slot1.Changed {
		t.Errorf("slot 1 = %+v (expected Changed=false)", slot1)
	}
	slot2 := res.Plan.Spells.Slots[1]
	if slot2.SlotNumber != 2 || !slot2.Changed || slot2.Current != nil || slot2.Target == nil {
		t.Errorf("slot 2 = %+v (expected Changed=true from nil to Comet Azur)", slot2)
	}
	if len(res.Plan.Spells.EquippedSpells) != 2 {
		t.Errorf("EquippedSpells length = %d, want 2", len(res.Plan.Spells.EquippedSpells))
	}
	if res.Plan.Spells.UsedMemorySlots != 4 {
		t.Errorf("UsedMemorySlots = %d, want 4", res.Plan.Spells.UsedMemorySlots)
	}

	// Verify exact match between Slots targets and EquippedSpells
	if slot1.Target != nil && (slot1.Target.BaseItemID != res.Plan.Spells.EquippedSpells[0].BaseItemID || slot1.Target.Name != res.Plan.Spells.EquippedSpells[0].Name) {
		t.Errorf("slot1.Target %+v does not match EquippedSpells[0] %+v", slot1.Target, res.Plan.Spells.EquippedSpells[0])
	}
	if slot2.Target != nil && (slot2.Target.BaseItemID != res.Plan.Spells.EquippedSpells[1].BaseItemID || slot2.Target.Name != res.Plan.Spells.EquippedSpells[1].Name) {
		t.Errorf("slot2.Target %+v does not match EquippedSpells[1] %+v", slot2.Target, res.Plan.Spells.EquippedSpells[1])
	}

	// Verify session and disk immutability
	sessionInfoAfter, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo after: %v", err)
	}
	if sessionInfoAfter.UnsavedChanges != sessionInfoBefore.UnsavedChanges {
		t.Errorf("UnsavedChanges changed from %v to %v", sessionInfoBefore.UnsavedChanges, sessionInfoAfter.UnsavedChanges)
	}

	undoAfter, err := engine.GetUndoState(loaded.SaveSessionID, testSlotActive)
	if err != nil {
		t.Fatalf("GetUndoState after: %v", err)
	}
	if undoBefore != undoAfter {
		t.Errorf("UndoState mutated: before %+v, after %+v", undoBefore, undoAfter)
	}

	rawAfter, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatalf("ReadFile save: %v", err)
	}
	if !bytes.Equal(rawBefore, rawAfter) {
		t.Error("save file on disk was modified by GetBuildTemplatePreview")
	}

	storeEntriesAfter, err := os.ReadDir(storeDir)
	if err != nil {
		t.Fatalf("ReadDir store after: %v", err)
	}
	if len(storeEntriesBefore) != len(storeEntriesAfter) {
		t.Fatalf("store directory modified: before %d files, after %d files", len(storeEntriesBefore), len(storeEntriesAfter))
	}
	for _, entry := range storeEntriesAfter {
		contentBefore, exists := storeFilesMapBefore[entry.Name()]
		if !exists {
			t.Fatalf("unexpected new file in store: %s", entry.Name())
		}
		contentAfter, readErr := os.ReadFile(filepath.Join(storeDir, entry.Name()))
		if readErr != nil {
			t.Fatalf("ReadFile %s: %v", entry.Name(), readErr)
		}
		if !bytes.Equal(contentBefore, contentAfter) {
			t.Fatalf("store file %s content mutated", entry.Name())
		}
	}
}

func TestGetBuildTemplatePreview_NarrowingAndValidation(t *testing.T) {
	engine := saveengine.New()
	savePath, _ := writeTestSaveFixture(t)
	loaded, err := engine.LoadSave(savePath, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := newTestCatalog(t)
	store := buildtemplates.NewStore(t.TempDir())

	vig := uint32(50)
	str := uint32(40)
	lvl := uint32(125)
	name := "Hero"

	tpl := &buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: "Stats Only Template",
		},
		Selection: &buildtemplates.TemplateSelection{
			Stats: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"vigor": true},
			},
			Profile: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"name": true, "level": true},
			},
		},
		Sections: buildtemplates.TemplateSections{
			Stats: &buildtemplates.StatsSection{
				Vigor:    &vig,
				Strength: &str,
			},
			Profile: &buildtemplates.ProfileSection{
				Name:  &name,
				Level: &lvl,
			},
		},
	}
	tplID, _ := createTestTemplate(t, store, tpl)

	tests := []struct {
		name         string
		reqSelection *buildtemplates.TemplateSelection
		reqOptions   *buildtemplates.ApplyOptions
		expectedCode string
	}{
		{
			name: "attempt to expand section not in template",
			reqSelection: &buildtemplates.TemplateSelection{
				Stats:     &buildtemplates.SectionSelection{Fields: map[string]bool{"vigor": true}},
				Equipment: &buildtemplates.SectionSelection{All: true},
			},
			expectedCode: templates.IssueCodeSelectionNotInTemplate,
		},
		{
			name: "attempt to expand field not in template",
			reqSelection: &buildtemplates.TemplateSelection{
				Stats: &buildtemplates.SectionSelection{Fields: map[string]bool{"vigor": true, "strength": true}},
			},
			expectedCode: templates.IssueCodeSelectionNotInTemplate,
		},
		{
			name: "attempt All=true when template has specific fields",
			reqSelection: &buildtemplates.TemplateSelection{
				Stats: &buildtemplates.SectionSelection{All: true},
			},
			expectedCode: templates.IssueCodeSelectionNotInTemplate,
		},
		{
			name: "empty selection",
			reqSelection: &buildtemplates.TemplateSelection{
				Stats: &buildtemplates.SectionSelection{Fields: map[string]bool{}},
			},
			expectedCode: templates.IssueCodeEmptySelection,
		},
		{
			name: "unsupported option provided in request",
			reqOptions: &buildtemplates.ApplyOptions{
				Items: &buildtemplates.ItemApplyOptions{Mode: "replace"},
			},
			expectedCode: templates.IssueCodeUnsupportedOption,
		},
		{
			name: "profile.level selected without stats",
			reqSelection: &buildtemplates.TemplateSelection{
				Profile: &buildtemplates.SectionSelection{Fields: map[string]bool{"level": true}},
			},
			expectedCode: templates.IssueCodeUnsupportedField,
		},
		{
			name: "unknown stats field in selection",
			reqSelection: &buildtemplates.TemplateSelection{
				Stats: &buildtemplates.SectionSelection{Fields: map[string]bool{"unknownStat": true}},
			},
			expectedCode: templates.IssueCodeUnsupportedField,
		},
		{
			name: "unknown spells slot in selection",
			reqSelection: &buildtemplates.TemplateSelection{
				Spells: &buildtemplates.SectionSelection{Fields: map[string]bool{"spell99": true}},
			},
			expectedCode: templates.IssueCodeUnsupportedField,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := templates.GetBuildTemplatePreview(store, engine, catalog, templates.GetBuildTemplatePreviewRequest{
				SaveSessionID: loaded.SaveSessionID,
				CharacterID:   testSlotActive,
				TemplateID:    tplID,
				Selection:     tc.reqSelection,
				Options:       tc.reqOptions,
			})
			if err != nil {
				t.Fatalf("unexpected endpoint error: %v", err)
			}
			if res.Executable {
				t.Errorf("expected Executable=false")
			}
			if !hasIssueCode(res.BlockingIssues, tc.expectedCode) {
				t.Errorf("expected blocking issue code %q, got issues: %+v", tc.expectedCode, res.BlockingIssues)
			}
		})
	}
}

func TestGetBuildTemplatePreview_TemplateApplyOptionsUnsupported(t *testing.T) {
	engine := saveengine.New()
	savePath, _ := writeTestSaveFixture(t)
	loaded, err := engine.LoadSave(savePath, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := newTestCatalog(t)
	store := buildtemplates.NewStore(t.TempDir())

	vig := uint32(50)
	tpl := &buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: "Template with ApplyOptions",
		},
		ApplyOptions: &buildtemplates.ApplyOptions{
			Items: &buildtemplates.ItemApplyOptions{Mode: "replace"},
		},
		Selection: &buildtemplates.TemplateSelection{
			Stats: &buildtemplates.SectionSelection{Fields: map[string]bool{"vigor": true}},
		},
		Sections: buildtemplates.TemplateSections{
			Stats: &buildtemplates.StatsSection{Vigor: &vig},
		},
	}
	tplID, _ := createTestTemplate(t, store, tpl)

	res, err := templates.GetBuildTemplatePreview(store, engine, catalog, templates.GetBuildTemplatePreviewRequest{
		SaveSessionID: loaded.SaveSessionID,
		CharacterID:   testSlotActive,
		TemplateID:    tplID,
	})
	if err != nil {
		t.Fatalf("GetBuildTemplatePreview: %v", err)
	}
	if res.Executable {
		t.Error("expected Executable=false when template has ApplyOptions")
	}
	if !hasIssueCode(res.BlockingIssues, templates.IssueCodeUnsupportedOption) {
		t.Errorf("expected IssueCodeUnsupportedOption, got %+v", res.BlockingIssues)
	}
}

func TestGetBuildTemplatePreview_LevelMismatch(t *testing.T) {
	engine := saveengine.New()
	savePath, _ := writeTestSaveFixture(t)
	loaded, err := engine.LoadSave(savePath, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := newTestCatalog(t)
	store := buildtemplates.NewStore(t.TempDir())

	vig := uint32(60)
	wrongLevel := uint32(100)
	name := "Hero"

	tpl := &buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: "Mismatch Template",
		},
		Selection: &buildtemplates.TemplateSelection{
			Stats: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"vigor": true},
			},
			Profile: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"name": true, "level": true},
			},
		},
		Sections: buildtemplates.TemplateSections{
			Stats: &buildtemplates.StatsSection{
				Vigor: &vig,
			},
			Profile: &buildtemplates.ProfileSection{
				Name:  &name,
				Level: &wrongLevel,
			},
		},
	}
	tplID, _ := createTestTemplate(t, store, tpl)

	res, err := templates.GetBuildTemplatePreview(store, engine, catalog, templates.GetBuildTemplatePreviewRequest{
		SaveSessionID: loaded.SaveSessionID,
		CharacterID:   testSlotActive,
		TemplateID:    tplID,
	})
	if err != nil {
		t.Fatalf("GetBuildTemplatePreview: %v", err)
	}
	if res.Executable {
		t.Error("expected Executable=false on level mismatch")
	}
	if !hasIssueCode(res.BlockingIssues, templates.IssueCodeLevelMismatch) {
		t.Errorf("expected IssueCodeLevelMismatch, got %+v", res.BlockingIssues)
	}
}

func TestGetBuildTemplatePreview_SpellCompactionAndAlignment(t *testing.T) {
	engine := saveengine.New()
	savePath, _ := writeTestSaveFixture(t)
	loaded, err := engine.LoadSave(savePath, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := newTestCatalog(t)
	store := buildtemplates.NewStore(t.TempDir())

	// Fixture has Glintstone Pebble (0x40000FA0) at slot 0 (spell1) and empty slot 1 (spell2).
	// Template clears slot 1 and sets slot 2 to Comet Azur (0x40001068).
	// After compaction, Comet Azur must become the target of slot 1, while slot 2 target is nil.
	tpl := &buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: "Spell Compaction Template",
		},
		Selection: &buildtemplates.TemplateSelection{
			Spells: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"spell1": true, "spell2": true},
			},
		},
		Sections: buildtemplates.TemplateSections{
			Spells: &buildtemplates.SpellsSection{
				Spell1: nil, // clear slot 1
				Spell2: &buildtemplates.SpellSlotRef{
					BaseItemID: 0x40001068, // Comet Azur
					Name:       "Comet Azur",
				},
			},
		},
	}
	tplID, _ := createTestTemplate(t, store, tpl)

	res, err := templates.GetBuildTemplatePreview(store, engine, catalog, templates.GetBuildTemplatePreviewRequest{
		SaveSessionID: loaded.SaveSessionID,
		CharacterID:   testSlotActive,
		TemplateID:    tplID,
	})
	if err != nil {
		t.Fatalf("GetBuildTemplatePreview: %v", err)
	}

	if !res.Executable {
		t.Fatalf("expected Executable=true, got false: %+v", res.BlockingIssues)
	}
	if len(res.Plan.Spells.EquippedSpells) != 1 {
		t.Fatalf("EquippedSpells length = %d, want 1 (compacted)", len(res.Plan.Spells.EquippedSpells))
	}
	if res.Plan.Spells.EquippedSpells[0].BaseItemID != 0x40001068 {
		t.Errorf("compacted spell[0] = 0x%08X, want Comet Azur", res.Plan.Spells.EquippedSpells[0].BaseItemID)
	}

	// Verify Slots alignment with EquippedSpells
	if len(res.Plan.Spells.Slots) != 12 {
		t.Fatalf("Slots length = %d, want 12", len(res.Plan.Spells.Slots))
	}
	// Slot 1: Current=Glintstone Pebble, Target=Comet Azur, Changed=true
	slot1 := res.Plan.Spells.Slots[0]
	if slot1.Current == nil || slot1.Current.BaseItemID != 0x40000FA0 {
		t.Errorf("slot1.Current = %+v, want Glintstone Pebble", slot1.Current)
	}
	if slot1.Target == nil || slot1.Target.BaseItemID != 0x40001068 {
		t.Errorf("slot1.Target = %+v, want Comet Azur", slot1.Target)
	}
	if slot1.Target != nil && (slot1.Target.BaseItemID != res.Plan.Spells.EquippedSpells[0].BaseItemID || slot1.Target.Name != res.Plan.Spells.EquippedSpells[0].Name) {
		t.Errorf("slot1.Target %+v does not match EquippedSpells[0] %+v", slot1.Target, res.Plan.Spells.EquippedSpells[0])
	}
	if !slot1.Changed {
		t.Errorf("slot1.Changed = false, want true")
	}

	// Slot 2: Current=nil, Target=nil, Changed=false
	slot2 := res.Plan.Spells.Slots[1]
	if slot2.Current != nil {
		t.Errorf("slot2.Current = %+v, want nil", slot2.Current)
	}
	if slot2.Target != nil {
		t.Errorf("slot2.Target = %+v, want nil", slot2.Target)
	}
	if slot2.Changed {
		t.Errorf("slot2.Changed = true, want false")
	}
}

func TestGetBuildTemplatePreview_PhysicalSlots13And14Occupied(t *testing.T) {
	engine := saveengine.New()
	_, rawData := writeTestSaveFixture(t)

	// Prepare corrupted fixture where physical slot 13 (index 12) is occupied
	corrupted := bytes.Clone(rawData)
	slotBase := int64(testFixtureHeaderSize) + 0x10 + int64(testSlotActive)*testFixtureSlotBlockSize
	anchorBase := slotBase + testAnchorAt
	at12 := anchorBase + testSectionSpellsAt + int64(12)*8
	binary.LittleEndian.PutUint32(corrupted[at12:], 0x0FA0)
	binary.LittleEndian.PutUint32(corrupted[at12+4:], 0xFFFFFFFF)

	corruptedPath := filepath.Join(t.TempDir(), "corrupted_slot13.sl2")
	if err := os.WriteFile(corruptedPath, corrupted, 0o600); err != nil {
		t.Fatalf("write corrupted fixture: %v", err)
	}

	loaded, err := engine.LoadSave(corruptedPath, "")
	if err != nil {
		t.Fatalf("LoadSave corrupted: %v", err)
	}
	catalog := newTestCatalog(t)
	store := buildtemplates.NewStore(t.TempDir())

	tpl := &buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: "Spells Template",
		},
		Selection: &buildtemplates.TemplateSelection{
			Spells: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"spell1": true},
			},
		},
		Sections: buildtemplates.TemplateSections{
			Spells: &buildtemplates.SpellsSection{
				Spell1: &buildtemplates.SpellSlotRef{
					BaseItemID: 0x40000FA0,
					Name:       "Glintstone Pebble",
				},
			},
		},
	}
	tplID, _ := createTestTemplate(t, store, tpl)

	res, err := templates.GetBuildTemplatePreview(store, engine, catalog, templates.GetBuildTemplatePreviewRequest{
		SaveSessionID: loaded.SaveSessionID,
		CharacterID:   testSlotActive,
		TemplateID:    tplID,
	})
	if err != nil {
		t.Fatalf("GetBuildTemplatePreview: %v", err)
	}
	if res.Executable {
		t.Error("expected Executable=false when physical slots 13-14 are occupied")
	}
	if !hasIssueCode(res.BlockingIssues, templates.IssueCodeInvalidSpellLoadout) {
		t.Errorf("expected IssueCodeInvalidSpellLoadout, got %+v", res.BlockingIssues)
	}
}

func TestGetBuildTemplatePreview_InvalidSpellLoadoutDuplicate(t *testing.T) {
	engine := saveengine.New()
	savePath, _ := writeTestSaveFixture(t)
	loaded, err := engine.LoadSave(savePath, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := newTestCatalog(t)
	store := buildtemplates.NewStore(t.TempDir())

	// Duplicate spell in slot 1 and slot 2
	tpl := &buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: "Duplicate Spell Template",
		},
		Selection: &buildtemplates.TemplateSelection{
			Spells: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"spell1": true, "spell2": true},
			},
		},
		Sections: buildtemplates.TemplateSections{
			Spells: &buildtemplates.SpellsSection{
				Spell1: &buildtemplates.SpellSlotRef{
					BaseItemID: 0x40000FA0, // Glintstone Pebble
					Name:       "Glintstone Pebble",
				},
				Spell2: &buildtemplates.SpellSlotRef{
					BaseItemID: 0x40000FA0, // Duplicate Glintstone Pebble
					Name:       "Glintstone Pebble",
				},
			},
		},
	}
	tplID, _ := createTestTemplate(t, store, tpl)

	res, err := templates.GetBuildTemplatePreview(store, engine, catalog, templates.GetBuildTemplatePreviewRequest{
		SaveSessionID: loaded.SaveSessionID,
		CharacterID:   testSlotActive,
		TemplateID:    tplID,
	})
	if err != nil {
		t.Fatalf("GetBuildTemplatePreview: %v", err)
	}
	if res.Executable {
		t.Error("expected Executable=false for duplicate spell loadout")
	}
	if res.Plan.Spells != nil {
		t.Errorf("expected Plan.Spells=nil on invalid spell loadout, got %+v", res.Plan.Spells)
	}
	if !hasIssueCode(res.BlockingIssues, templates.IssueCodeInvalidSpellLoadout) {
		t.Errorf("expected IssueCodeInvalidSpellLoadout, got %+v", res.BlockingIssues)
	}
}
