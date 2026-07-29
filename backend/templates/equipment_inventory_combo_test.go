package templates

import "testing"

// Phase 7b.1 (P3 follow-up) — the equipment ↔ Inventory mutual-exclusion
// rule must fire for every Inventory-mutating section, not just
// inventory.workspace. These table-driven preview tests cover the four
// disallowed combinations for imported / hand-authored v2 templates.

// equipmentComboBase returns a v2 template that selects equipment (All)
// with one populated slot. Each conflict case layers a conflicting
// section on top via the mutate callback and re-validates so the combo
// guard — not a structural error — is the surfacing failure.
func equipmentComboBase(t *testing.T, mutate func(tpl *BuildTemplate)) *BuildTemplate {
	t.Helper()
	tpl := &BuildTemplate{
		Schema:    SchemaKey,
		Version:   2,
		CreatedAt: "2026-06-01T00:00:00Z",
		Selection: &TemplateSelection{Equipment: &SectionSelection{All: true}},
		Sections: TemplateSections{
			Equipment: &EquipmentSection{WeaponRightHand1: &EquipmentItemRef{BaseItemID: 0x100000}},
		},
	}
	mutate(tpl)
	if err := ValidateBuildTemplate(tpl); err != nil {
		t.Fatalf("fixture must validate before the combo guard runs: %v", err)
	}
	return tpl
}

func TestPreview_EquipmentInventoryCombo_RejectedFamily(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(tpl *BuildTemplate)
	}{
		{
			name: "equipment+inventory.workspace",
			mutate: func(tpl *BuildTemplate) {
				tpl.Selection.InventoryWorkspace = &SectionSelection{All: true}
				tpl.Sections.InventoryWorkspace = &InventoryWorkspaceSection{
					InventoryItems: []TemplateItem{},
					StorageItems:   []TemplateItem{},
				}
			},
		},
		{
			name: "equipment+items",
			mutate: func(tpl *BuildTemplate) {
				tpl.Selection.Items = &SectionSelection{All: true}
				tpl.Sections.Items = &ItemsSection{Entries: []TemplateItemEntryV2{validMeleeEntry("w1")}}
			},
		},
		{
			name: "equipment+inventoryLayout",
			mutate: func(tpl *BuildTemplate) {
				// inventoryLayout references items.entries but items is NOT
				// selected — isolates inventoryLayout as the sole conflict.
				tpl.Selection.InventoryLayout = &SectionSelection{All: true}
				tpl.Sections.Items = &ItemsSection{Entries: []TemplateItemEntryV2{validMeleeEntry("w1")}}
				tpl.Sections.InventoryLayout = &InventoryLayoutSection{
					Entries: []LayoutEntry{{EntryRef: "w1", Position: 0}},
				}
			},
		},
		{
			name: "equipment+storageLayout",
			mutate: func(tpl *BuildTemplate) {
				tpl.Selection.StorageLayout = &SectionSelection{All: true}
				tpl.Sections.Items = &ItemsSection{Entries: []TemplateItemEntryV2{validMeleeEntry("w1")}}
				tpl.Sections.StorageLayout = &StorageLayoutSection{
					Entries: []LayoutEntry{{EntryRef: "w1", Position: 0}},
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tpl := equipmentComboBase(t, tc.mutate)
			rep := PreviewBuildTemplateImport(tpl, ImportPreviewOptions{})

			if rep.OK {
				t.Fatalf("preview must reject %s, got OK=true", tc.name)
			}
			// Exactly the combo hard error, as an error (not a warning).
			combos := 0
			for _, e := range rep.Errors {
				if e.Code == IssueCodeEquipmentInventoryComboUnsupported {
					if e.Severity != "error" {
						t.Errorf("combo issue must be severity=error, got %q", e.Severity)
					}
					combos++
				}
			}
			if combos != 1 {
				t.Errorf("expected exactly one %s hard error, got %d (errors=%+v)",
					IssueCodeEquipmentInventoryComboUnsupported, combos, rep.Errors)
			}
			for _, w := range rep.Warnings {
				if w.Code == IssueCodeEquipmentInventoryComboUnsupported {
					t.Errorf("combo must be a hard error, not a warning: %+v", w)
				}
			}
		})
	}
}

// Negative case — equipment coexists with profile / stats / spells; the
// guard must NOT fire and the preview stays OK.
func TestPreview_EquipmentWithProfileStatsSpells_Allowed(t *testing.T) {
	tpl := &BuildTemplate{
		Schema:    SchemaKey,
		Version:   2,
		CreatedAt: "2026-06-01T00:00:00Z",
		Selection: &TemplateSelection{
			Equipment: &SectionSelection{All: true},
			Profile:   &SectionSelection{All: true},
			Stats:     &SectionSelection{All: true},
			Spells:    &SectionSelection{All: true},
		},
		Sections: TemplateSections{
			Equipment: &EquipmentSection{WeaponRightHand1: &EquipmentItemRef{BaseItemID: 0x100000}},
			Profile:   &ProfileSection{Level: equipU32Ptr(150), Runes: equipU32Ptr(0)},
			Stats: &StatsSection{
				Vigor: equipU32Ptr(60), Mind: equipU32Ptr(25), Endurance: equipU32Ptr(25),
				Strength: equipU32Ptr(12), Dexterity: equipU32Ptr(18), Intelligence: equipU32Ptr(80),
				Faith: equipU32Ptr(9), Arcane: equipU32Ptr(7),
			},
			Spells: &SpellsSection{Spell1: &SpellSlotRef{BaseItemID: 0x40001770, Name: "Catch Flame"}},
		},
	}
	if err := ValidateBuildTemplate(tpl); err != nil {
		t.Fatalf("equipment+profile+stats+spells must validate: %v", err)
	}
	rep := PreviewBuildTemplateImport(tpl, ImportPreviewOptions{})
	for _, e := range rep.Errors {
		if e.Code == IssueCodeEquipmentInventoryComboUnsupported {
			t.Fatalf("combo guard must NOT fire for equipment+profile+stats+spells: %+v", rep.Errors)
		}
	}
	if !rep.OK {
		t.Errorf("preview should be OK for equipment+profile+stats+spells, errors=%+v", rep.Errors)
	}
}
