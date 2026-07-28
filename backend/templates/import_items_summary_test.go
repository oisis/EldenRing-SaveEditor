package templates

import "testing"

// v2 items summary counting is derived exclusively from
// TemplateItemEntryV2.Location (container) and .Category (bucket) — never
// from the layout sections. These tests lock that contract in.

// entry is a small builder for a valid TemplateItemEntryV2.
func entry(id string, itemID uint32, category, location string) TemplateItemEntryV2 {
	return TemplateItemEntryV2{
		EntryID:  id,
		ItemID:   itemID,
		Category: category,
		Quantity: 1,
		Location: location,
	}
}

// makeV2ItemsTemplate wraps a set of item entries (and optional layouts)
// into a valid, selection-items-only v2 template.
func makeV2ItemsTemplate(entries []TemplateItemEntryV2) *BuildTemplate {
	return &BuildTemplate{
		Schema:  SchemaKey,
		Version: 2,
		Selection: &TemplateSelection{
			Items: &SectionSelection{All: true},
		},
		Sections: TemplateSections{
			Items: &ItemsSection{Entries: entries},
		},
	}
}

func TestSummarizeItemsSection_NilAndEmpty(t *testing.T) {
	if got := summarizeItemsSection(nil); got != (itemsSummaryCounts{}) {
		t.Errorf("nil section = %+v, want zero", got)
	}
	if got := summarizeItemsSection(&ItemsSection{}); got != (itemsSummaryCounts{}) {
		t.Errorf("empty section = %+v, want zero", got)
	}
}

func TestSummarizeItemsSection_LocationBuckets(t *testing.T) {
	c := summarizeItemsSection(&ItemsSection{Entries: []TemplateItemEntryV2{
		entry("a", 0x1000, ItemCategoryTools, ItemLocationInventory),
		entry("b", 0x1001, ItemCategoryTools, ItemLocationStorage),
		entry("c", 0x1002, ItemCategoryTools, ItemLocationBoth),
	}})
	// inventory: a + c ; storage: b + c
	if c.InventoryItems != 2 {
		t.Errorf("InventoryItems = %d, want 2", c.InventoryItems)
	}
	if c.StorageItems != 2 {
		t.Errorf("StorageItems = %d, want 2", c.StorageItems)
	}
}

func TestSummarizeItemsSection_CategoryBuckets(t *testing.T) {
	c := summarizeItemsSection(&ItemsSection{Entries: []TemplateItemEntryV2{
		entry("w1", 0x1, ItemCategoryMeleeArmaments, ItemLocationInventory),
		entry("w2", 0x2, ItemCategoryRangedAndCatalysts, ItemLocationInventory),
		entry("w3", 0x3, ItemCategoryShields, ItemLocationInventory),
		entry("a1", 0x4, ItemCategoryArmorHead, ItemLocationInventory),
		entry("a2", 0x5, ItemCategoryArmorChest, ItemLocationInventory),
		entry("a3", 0x6, ItemCategoryArmorArms, ItemLocationInventory),
		entry("a4", 0x7, ItemCategoryArmorLegs, ItemLocationInventory),
		entry("t1", 0x8, ItemCategoryTalismans, ItemLocationInventory),
		entry("s1", 0x9, ItemCategoryCraftingMaterials, ItemLocationInventory),
		entry("s2", 0xA, ItemCategoryKeyItems, ItemLocationInventory),
	}})
	if c.Weapons != 3 {
		t.Errorf("Weapons = %d, want 3", c.Weapons)
	}
	if c.Armor != 4 {
		t.Errorf("Armor = %d, want 4", c.Armor)
	}
	if c.Talismans != 1 {
		t.Errorf("Talismans = %d, want 1", c.Talismans)
	}
	if c.Stackables != 2 {
		t.Errorf("Stackables = %d, want 2", c.Stackables)
	}
}

func TestSummarizeItemsSection_CategoryCountedOncePerBothLocation(t *testing.T) {
	// A weapon in both containers is one weapon, but two container slots.
	c := summarizeItemsSection(&ItemsSection{Entries: []TemplateItemEntryV2{
		entry("w", 0x1, ItemCategoryMeleeArmaments, ItemLocationBoth),
	}})
	if c.Weapons != 1 {
		t.Errorf("Weapons = %d, want 1 (category counted once)", c.Weapons)
	}
	if c.InventoryItems != 1 || c.StorageItems != 1 {
		t.Errorf("containers = inv %d / stor %d, want 1/1", c.InventoryItems, c.StorageItems)
	}
}

func TestSummarizeItemsSection_AoWCountedOncePerEntry(t *testing.T) {
	aow := uint32(0xC0DE)
	weaponWithAoW := entry("w", 0x1, ItemCategoryMeleeArmaments, ItemLocationBoth)
	weaponWithAoW.AshOfWarItemID = &aow
	c := summarizeItemsSection(&ItemsSection{Entries: []TemplateItemEntryV2{
		weaponWithAoW,
		entry("plain", 0x2, ItemCategoryMeleeArmaments, ItemLocationInventory),
	}})
	if c.AoWAssignments != 1 {
		t.Errorf("AoWAssignments = %d, want 1 (both-location entry counts once, plain entry none)", c.AoWAssignments)
	}
}

// PreviewBuildTemplateImport wiring — the summary the UI actually reads.

func TestPreviewBuildTemplateImport_V2ItemsSummary(t *testing.T) {
	aow := uint32(0xC0DE)
	w := entry("w", 0x1, ItemCategoryMeleeArmaments, ItemLocationInventory)
	w.AshOfWarItemID = &aow
	tpl := makeV2ItemsTemplate([]TemplateItemEntryV2{
		w,
		entry("armor", 0x2, ItemCategoryArmorChest, ItemLocationStorage),
		entry("tal", 0x3, ItemCategoryTalismans, ItemLocationBoth),
		entry("mat", 0x4, ItemCategoryCraftingMaterials, ItemLocationInventory),
	})
	rep := PreviewBuildTemplateImport(tpl, ImportPreviewOptions{})
	if !rep.OK {
		t.Fatalf("report not OK: %+v", rep.Errors)
	}
	s := rep.Summary
	// inventory: w + tal + mat = 3 ; storage: armor + tal = 2
	if s.InventoryItems != 3 || s.StorageItems != 2 {
		t.Errorf("containers = inv %d / stor %d, want 3/2", s.InventoryItems, s.StorageItems)
	}
	if s.Weapons != 1 || s.Armor != 1 || s.Talismans != 1 || s.Stackables != 1 {
		t.Errorf("buckets = W%d A%d T%d S%d, want 1/1/1/1", s.Weapons, s.Armor, s.Talismans, s.Stackables)
	}
	if s.AoWAssignments != 1 {
		t.Errorf("AoWAssignments = %d, want 1", s.AoWAssignments)
	}
	if s.ItemsEntries != 4 {
		t.Errorf("ItemsEntries = %d, want 4", s.ItemsEntries)
	}
}

func TestPreviewBuildTemplateImport_EmptyItemsSectionZeros(t *testing.T) {
	tpl := makeV2ItemsTemplate([]TemplateItemEntryV2{})
	rep := PreviewBuildTemplateImport(tpl, ImportPreviewOptions{})
	s := rep.Summary
	if s.InventoryItems != 0 || s.StorageItems != 0 || s.Weapons != 0 ||
		s.Armor != 0 || s.Talismans != 0 || s.Stackables != 0 || s.AoWAssignments != 0 {
		t.Errorf("empty items summary not all zero: %+v", s)
	}
}

func TestPreviewBuildTemplateImport_ItemsNotCountedFromLayout(t *testing.T) {
	// One entry, referenced from BOTH layouts. Layout refs must never
	// inflate the container/category/AoW counts.
	tpl := makeV2ItemsTemplate([]TemplateItemEntryV2{
		entry("only", 0x1, ItemCategoryMeleeArmaments, ItemLocationInventory),
	})
	tpl.Selection.InventoryLayout = &SectionSelection{All: true}
	tpl.Selection.StorageLayout = &SectionSelection{All: true}
	tpl.Sections.InventoryLayout = &InventoryLayoutSection{Entries: []LayoutEntry{{EntryRef: "only", Position: 0}}}
	tpl.Sections.StorageLayout = &StorageLayoutSection{Entries: []LayoutEntry{{EntryRef: "only", Position: 0}}}

	rep := PreviewBuildTemplateImport(tpl, ImportPreviewOptions{})
	if !rep.OK {
		t.Fatalf("report not OK: %+v", rep.Errors)
	}
	s := rep.Summary
	if s.InventoryItems != 1 || s.StorageItems != 0 || s.Weapons != 1 {
		t.Errorf("layout inflated counts: inv %d stor %d weapons %d, want 1/0/1", s.InventoryItems, s.StorageItems, s.Weapons)
	}
}

func TestPreviewBuildTemplateImport_LegacyWorkspaceTakesPriority(t *testing.T) {
	// A template that ships BOTH inventory.workspace and items. Legacy
	// path wins; items are not summed on top.
	tpl := &BuildTemplate{
		Schema:  SchemaKey,
		Version: 2,
		Selection: &TemplateSelection{
			InventoryWorkspace: &SectionSelection{All: true},
			Items:              &SectionSelection{All: true},
		},
		Sections: TemplateSections{
			InventoryWorkspace: &InventoryWorkspaceSection{
				InventoryItems: []TemplateItem{{
					BaseItemID: 0x000F4240,
					Name:       "Greatsword",
					Category:   "melee_armaments",
					Quantity:   1,
					Upgrade:    10,
					Container:  ContainerInventory,
					Position:   0,
				}},
				StorageItems: []TemplateItem{},
			},
			Items: &ItemsSection{Entries: []TemplateItemEntryV2{
				entry("a", 0x1, ItemCategoryTools, ItemLocationInventory),
				entry("b", 0x2, ItemCategoryTools, ItemLocationInventory),
			}},
		},
	}
	rep := PreviewBuildTemplateImport(tpl, ImportPreviewOptions{})
	// Legacy workspace has exactly 1 inventory item; items would have
	// added 2 more if double-counted.
	if rep.Summary.InventoryItems != 1 {
		t.Errorf("InventoryItems = %d, want 1 (legacy priority, no double count)", rep.Summary.InventoryItems)
	}
	if rep.Summary.StorageItems != 0 {
		t.Errorf("StorageItems = %d, want 0", rep.Summary.StorageItems)
	}
}
