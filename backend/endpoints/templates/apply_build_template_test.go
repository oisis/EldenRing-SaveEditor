package templates_test

import (
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/templates"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestApplyBuildTemplate_Success(t *testing.T) {
	engine := saveengine.New()
	savePath, _ := writeTestSaveFixture(t)
	loaded, err := engine.LoadSave(savePath, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := newTestCatalog(t)
	storeDir := filepath.Join(t.TempDir(), "templates")
	store := buildtemplates.NewStore(storeDir)

	vigTarget := uint32(60)
	strTarget := uint32(40)
	targetName := "Archmage"
	expectedTargetLevel := uint32(133)

	tpl := &buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: "Archmage Build",
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
					BaseItemID: 0x40000FA0, // Glintstone Pebble
					Name:       "Glintstone Pebble",
				},
				Spell2: &buildtemplates.SpellSlotRef{
					BaseItemID: 0x40000FA1, // Glintstone Stars
					Name:       "Glintstone Stars",
				},
			},
		},
	}
	tplID, tplRev := createTestTemplate(t, store, tpl)

	initialRev := "0"
	res, err := templates.ApplyBuildTemplate(store, engine, catalog, templates.ApplyBuildTemplateRequest{
		SaveSessionID:    loaded.SaveSessionID,
		CharacterID:      testSlotActive,
		TemplateID:       tplID,
		ExpectedRevision: initialRev,
	})
	if err != nil {
		t.Fatalf("ApplyBuildTemplate: %v", err)
	}

	if res.TemplateID != tplID {
		t.Errorf("TemplateID = %q, want %q", res.TemplateID, tplID)
	}
	if res.TemplateRevision != tplRev {
		t.Errorf("TemplateRevision = %q, want %q", res.TemplateRevision, tplRev)
	}
	if res.SaveSessionID != loaded.SaveSessionID {
		t.Errorf("SaveSessionID = %q, want %q", res.SaveSessionID, loaded.SaveSessionID)
	}
	if res.SaveRevision == initialRev {
		t.Errorf("SaveRevision was not incremented, got %q", res.SaveRevision)
	}
	if res.CharacterID != testSlotActive {
		t.Errorf("CharacterID = %d, want %d", res.CharacterID, testSlotActive)
	}

	// Verify plan details in receipt.
	if res.Plan.Profile == nil || res.Plan.Profile.Name == nil || res.Plan.Profile.Name.Target != targetName {
		t.Errorf("Plan.Profile.Name mismatch: %+v", res.Plan.Profile)
	}
	if res.Plan.Stats == nil {
		t.Errorf("Plan.Stats is nil")
	} else if res.Plan.Stats.ResultLevel != expectedTargetLevel {
		t.Errorf("Plan.Stats.ResultLevel = %d, want %d", res.Plan.Stats.ResultLevel, expectedTargetLevel)
	}
	if res.Plan.Spells == nil || len(res.Plan.Spells.EquippedSpells) != 2 {
		t.Errorf("Plan.Spells mismatch: %+v", res.Plan.Spells)
	}

	// Verify live save state in engine.
	prof, err := engine.GetCharacterProfile(loaded.SaveSessionID, testSlotActive)
	if err != nil {
		t.Fatalf("GetCharacterProfile: %v", err)
	}
	if prof.Name != targetName {
		t.Errorf("persisted profile name = %q, want %q", prof.Name, targetName)
	}
	if prof.Level != expectedTargetLevel {
		t.Errorf("persisted profile level = %d, want %d", prof.Level, expectedTargetLevel)
	}

	stats, err := engine.GetCharacterStats(loaded.SaveSessionID, testSlotActive)
	if err != nil {
		t.Fatalf("GetCharacterStats: %v", err)
	}
	if stats.Vigor != vigTarget || stats.Strength != strTarget || stats.Level != expectedTargetLevel {
		t.Errorf("persisted stats mismatch: vigor=%d strength=%d level=%d", stats.Vigor, stats.Strength, stats.Level)
	}

	spells, err := engine.GetEquippedSpells(loaded.SaveSessionID, testSlotActive)
	if err != nil {
		t.Fatalf("GetEquippedSpells: %v", err)
	}
	if spells.Spells[0] != 0x00000FA0 || spells.Spells[1] != 0x00000FA1 {
		t.Errorf("persisted spells[0..1] mismatch: %v", spells.Spells[:2])
	}
	if spells.Spells[12] != 0xFFFFFFFF || spells.Spells[13] != 0xFFFFFFFF {
		t.Errorf("physical slots 13-14 modified: %v", spells.Spells[12:])
	}

	// Verify undo state.
	undo, err := engine.GetUndoState(loaded.SaveSessionID, testSlotActive)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if !undo.Available {
		t.Errorf("expected undo point available")
	}
	if undo.OperationID != "apply_build_template" {
		t.Errorf("undo operationID = %q, want 'apply_build_template'", undo.OperationID)
	}
}

func TestApplyBuildTemplate_BlockingPreviewCausesZeroMutation(t *testing.T) {
	engine := saveengine.New()
	savePath, _ := writeTestSaveFixture(t)
	loaded, err := engine.LoadSave(savePath, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := newTestCatalog(t)
	store := buildtemplates.NewStore(t.TempDir())

	// Level mismatch template (profile.level 100 != calculated 133)
	vigTarget := uint32(60)
	strTarget := uint32(40)
	targetName := "Mismatch"
	wrongLevel := uint32(100)

	tpl := &buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: "Mismatch Template",
		},
		Selection: &buildtemplates.TemplateSelection{
			Profile: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"name": true, "level": true},
			},
			Stats: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"vigor": true, "strength": true},
			},
		},
		Sections: buildtemplates.TemplateSections{
			Profile: &buildtemplates.ProfileSection{
				Name:  &targetName,
				Level: &wrongLevel,
			},
			Stats: &buildtemplates.StatsSection{
				Vigor:    &vigTarget,
				Strength: &strTarget,
			},
		},
	}
	tplID, _ := createTestTemplate(t, store, tpl)

	initialRev := "0"
	_, err = templates.ApplyBuildTemplate(store, engine, catalog, templates.ApplyBuildTemplateRequest{
		SaveSessionID:    loaded.SaveSessionID,
		CharacterID:      testSlotActive,
		TemplateID:       tplID,
		ExpectedRevision: initialRev,
	})
	if err == nil {
		t.Fatal("expected error on blocking issue, got nil")
	}

	// Verify revision unchanged and no undo point created.
	undo, err := engine.GetUndoState(loaded.SaveSessionID, testSlotActive)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if undo.SaveRevision != initialRev {
		t.Errorf("saveRevision changed: %q != %q", undo.SaveRevision, initialRev)
	}
	if undo.Available {
		t.Errorf("expected no undo point available")
	}
}

func TestApplyBuildTemplate_StaleExpectedRevisionRejection(t *testing.T) {
	engine := saveengine.New()
	savePath, _ := writeTestSaveFixture(t)
	loaded, err := engine.LoadSave(savePath, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := newTestCatalog(t)
	store := buildtemplates.NewStore(t.TempDir())

	targetName := "Lord"
	tpl := &buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: "Lord Template",
		},
		Selection: &buildtemplates.TemplateSelection{
			Profile: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"name": true},
			},
		},
		Sections: buildtemplates.TemplateSections{
			Profile: &buildtemplates.ProfileSection{
				Name: &targetName,
			},
		},
	}
	tplID, _ := createTestTemplate(t, store, tpl)

	_, err = templates.ApplyBuildTemplate(store, engine, catalog, templates.ApplyBuildTemplateRequest{
		SaveSessionID:    loaded.SaveSessionID,
		CharacterID:      testSlotActive,
		TemplateID:       tplID,
		ExpectedRevision: "9999", // Stale revision
	})
	if err == nil {
		t.Fatal("expected error for stale expectedRevision, got nil")
	}

	// Verify revision unchanged.
	undo, err := engine.GetUndoState(loaded.SaveSessionID, testSlotActive)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if undo.SaveRevision != "0" {
		t.Errorf("saveRevision changed: %q != %q", undo.SaveRevision, "0")
	}
}

func TestApplyBuildTemplate_NonCanonicalExpectedRevisionRejection(t *testing.T) {
	engine := saveengine.New()
	savePath, _ := writeTestSaveFixture(t)
	loaded, err := engine.LoadSave(savePath, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := newTestCatalog(t)
	store := buildtemplates.NewStore(t.TempDir())

	targetName := "Lord"
	tpl := &buildtemplates.BuildTemplate{
		Schema:  buildtemplates.SchemaKey,
		Version: buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name: "Lord Template",
		},
		Selection: &buildtemplates.TemplateSelection{
			Profile: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"name": true},
			},
		},
		Sections: buildtemplates.TemplateSections{
			Profile: &buildtemplates.ProfileSection{
				Name: &targetName,
			},
		},
	}
	tplID, _ := createTestTemplate(t, store, tpl)

	for _, badRev := range []string{"01", "+1", "", "abc"} {
		t.Run("revision_"+badRev, func(t *testing.T) {
			_, err = templates.ApplyBuildTemplate(store, engine, catalog, templates.ApplyBuildTemplateRequest{
				SaveSessionID:    loaded.SaveSessionID,
				CharacterID:      testSlotActive,
				TemplateID:       tplID,
				ExpectedRevision: badRev,
			})
			if err == nil {
				t.Fatalf("expected error for non-canonical expectedRevision %q, got nil", badRev)
			}

			undo, err := engine.GetUndoState(loaded.SaveSessionID, testSlotActive)
			if err != nil {
				t.Fatalf("GetUndoState: %v", err)
			}
			if undo.SaveRevision != "0" {
				t.Errorf("saveRevision changed: %q != %q", undo.SaveRevision, "0")
			}
			if undo.Available {
				t.Errorf("expected no undo point available")
			}
		})
	}
}
