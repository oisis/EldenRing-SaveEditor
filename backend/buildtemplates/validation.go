package buildtemplates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf16"
)

// DecodeTemplate decodes and strictly validates a template document from raw JSON.
// It fails closed on unknown JSON fields, malformed syntax, or schema violations.
func DecodeTemplate(data []byte) (*BuildTemplate, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var tpl BuildTemplate
	if err := dec.Decode(&tpl); err != nil {
		return nil, fmt.Errorf("decode template: %w", err)
	}
	var trailing json.RawMessage
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("extra data after JSON payload")
		}
		return nil, fmt.Errorf("decode trailing JSON data: %w", err)
	}
	if err := ValidateTemplate(&tpl); err != nil {
		return nil, err
	}
	return &tpl, nil
}

// ValidateTemplate enforces all confirmed schema invariants for v1 and v2 templates.
func ValidateTemplate(tpl *BuildTemplate) error {
	if tpl == nil {
		return fmt.Errorf("ValidateTemplate: nil template")
	}
	if tpl.Schema != SchemaKey {
		return fmt.Errorf("ValidateTemplate: wrong schema %q (expected %q)", tpl.Schema, SchemaKey)
	}
	if tpl.Version <= 0 {
		return fmt.Errorf("ValidateTemplate: invalid version %d", tpl.Version)
	}
	if tpl.Version > MaxSchemaVersion {
		return fmt.Errorf("ValidateTemplate: unsupported version %d (max supported %d)", tpl.Version, MaxSchemaVersion)
	}
	if tpl.Version == SchemaVersionV1 {
		return validateBuildTemplateV1(tpl)
	}
	return validateBuildTemplateV2(tpl)
}

func validateBuildTemplateV1(tpl *BuildTemplate) error {
	if tpl.Sections.InventoryWorkspace == nil {
		return fmt.Errorf("ValidateTemplate: missing inventory.workspace section")
	}
	sec := tpl.Sections.InventoryWorkspace
	if len(sec.InventoryItems) == 0 && len(sec.StorageItems) == 0 {
		return fmt.Errorf("ValidateTemplate: inventory.workspace is empty")
	}
	if err := validateItems(sec.InventoryItems, ContainerInventory); err != nil {
		return err
	}
	if err := validateItems(sec.StorageItems, ContainerStorage); err != nil {
		return err
	}
	return nil
}

// ValidateTemplateSelection validates that a selection contains only supported fields and formats.
func ValidateTemplateSelection(sel *TemplateSelection) error {
	if sel == nil {
		return nil
	}
	if err := validateProfileSelection(sel.Profile); err != nil {
		return err
	}
	if err := validateStatsSelection(sel.Stats); err != nil {
		return err
	}
	if err := validateInventoryWorkspaceSelection(sel.InventoryWorkspace); err != nil {
		return err
	}
	if err := validateEquipmentSelection(sel.Equipment); err != nil {
		return err
	}
	if err := validateSpellsSelection(sel.Spells); err != nil {
		return err
	}
	if err := validateBooleanOnlySelection("selection.items", sel.Items); err != nil {
		return err
	}
	if err := validateBooleanOnlySelection("selection.inventoryLayout", sel.InventoryLayout); err != nil {
		return err
	}
	if err := validateBooleanOnlySelection("selection.storageLayout", sel.StorageLayout); err != nil {
		return err
	}
	return nil
}

func validateBuildTemplateV2(tpl *BuildTemplate) error {
	if tpl.Selection == nil {
		return fmt.Errorf("ValidateTemplate: v2 template requires a selection object")
	}
	if !tpl.Selection.HasAnySelected() {
		return fmt.Errorf("ValidateTemplate: v2 template selection has no selected fields")
	}

	if err := ValidateTemplateSelection(tpl.Selection); err != nil {
		return err
	}

	if tpl.Selection.Profile.HasAny() && tpl.Sections.Profile == nil {
		return fmt.Errorf("ValidateTemplate: selection.profile is selected but sections.profile is missing")
	}
	if tpl.Selection.Stats.HasAny() && tpl.Sections.Stats == nil {
		return fmt.Errorf("ValidateTemplate: selection.stats is selected but sections.stats is missing")
	}
	if tpl.Selection.InventoryWorkspace.HasAny() && tpl.Sections.InventoryWorkspace == nil {
		return fmt.Errorf("ValidateTemplate: selection.inventory.workspace is selected but sections.inventory.workspace is missing")
	}
	if tpl.Selection.Equipment.HasAny() && tpl.Sections.Equipment == nil {
		return fmt.Errorf("ValidateTemplate: selection.equipment is selected but sections.equipment is missing")
	}
	if tpl.Selection.Spells.HasAny() && tpl.Sections.Spells == nil {
		return fmt.Errorf("ValidateTemplate: selection.spells is selected but sections.spells is missing")
	}
	if tpl.Selection.Items.HasAny() && tpl.Sections.Items == nil {
		return fmt.Errorf("ValidateTemplate: selection.items is selected but sections.items is missing")
	}
	if tpl.Selection.InventoryLayout.HasAny() && tpl.Sections.InventoryLayout == nil {
		return fmt.Errorf("ValidateTemplate: selection.inventoryLayout is selected but sections.inventoryLayout is missing")
	}
	if tpl.Selection.StorageLayout.HasAny() && tpl.Sections.StorageLayout == nil {
		return fmt.Errorf("ValidateTemplate: selection.storageLayout is selected but sections.storageLayout is missing")
	}

	if tpl.Sections.Profile != nil {
		if err := validateProfileSection(tpl.Sections.Profile); err != nil {
			return err
		}
	}
	if tpl.Sections.Stats != nil {
		if err := validateStatsSection(tpl.Sections.Stats); err != nil {
			return err
		}
	}
	if tpl.Sections.InventoryWorkspace != nil {
		if err := validateItems(tpl.Sections.InventoryWorkspace.InventoryItems, ContainerInventory); err != nil {
			return err
		}
		if err := validateItems(tpl.Sections.InventoryWorkspace.StorageItems, ContainerStorage); err != nil {
			return err
		}
	}
	if tpl.Sections.Equipment != nil {
		if err := validateEquipmentSection(tpl.Sections.Equipment); err != nil {
			return err
		}
	}
	if tpl.Sections.Spells != nil {
		if err := validateSpellsSection(tpl.Sections.Spells); err != nil {
			return err
		}
	}
	if tpl.Sections.Items != nil {
		if err := validateItemsSection(tpl.Sections.Items); err != nil {
			return err
		}
	}
	if tpl.Sections.InventoryLayout != nil {
		if err := validateLayoutSection("inventoryLayout.entries", tpl.Sections.InventoryLayout.Entries, tpl.Sections.Items); err != nil {
			return err
		}
	}
	if tpl.Sections.StorageLayout != nil {
		if err := validateLayoutSection("storageLayout.entries", tpl.Sections.StorageLayout.Entries, tpl.Sections.Items); err != nil {
			return err
		}
	}
	if tpl.ApplyOptions != nil {
		if err := validateApplyOptions(tpl.ApplyOptions); err != nil {
			return err
		}
	}
	return nil
}

func validateItems(items []TemplateItem, expectedContainer string) error {
	for i, it := range items {
		if it.BaseItemID == 0 {
			return fmt.Errorf("validateItems[%s][%d]: baseItemID=0", expectedContainer, i)
		}
		if it.Quantity == 0 {
			return fmt.Errorf("validateItems[%s][%d]: quantity=0 (baseItemID=0x%08X)", expectedContainer, i, it.BaseItemID)
		}
		if it.Container != expectedContainer {
			return fmt.Errorf("validateItems[%s][%d]: container=%q does not match section", expectedContainer, i, it.Container)
		}
	}
	return nil
}

func validateProfileSelection(sel *SectionSelection) error {
	if sel == nil {
		return nil
	}
	for key := range sel.Fields {
		if !profileSelectionFields[key] {
			return fmt.Errorf("ValidateTemplate: selection.profile has unknown field %q", key)
		}
	}
	return nil
}

func validateStatsSelection(sel *SectionSelection) error {
	if sel == nil {
		return nil
	}
	for key := range sel.Fields {
		if !statsSelectionFields[key] {
			return fmt.Errorf("ValidateTemplate: selection.stats has unknown field %q", key)
		}
	}
	return nil
}

func validateInventoryWorkspaceSelection(sel *SectionSelection) error {
	if sel == nil {
		return nil
	}
	if sel.Fields != nil {
		return fmt.Errorf("ValidateTemplate: selection.inventory.workspace accepts only a boolean (got a field map)")
	}
	return nil
}

func validateEquipmentSelection(sel *SectionSelection) error {
	if sel == nil {
		return nil
	}
	for key := range sel.Fields {
		if !equipmentSelectionFields[key] {
			return fmt.Errorf("ValidateTemplate: selection.equipment has unknown slot %q", key)
		}
	}
	return nil
}

func validateSpellsSelection(sel *SectionSelection) error {
	if sel == nil {
		return nil
	}
	for key := range sel.Fields {
		if !spellsSelectionFields[key] {
			return fmt.Errorf("ValidateTemplate: selection.spells has unknown slot %q", key)
		}
	}
	return nil
}

func validateBooleanOnlySelection(label string, sel *SectionSelection) error {
	if sel == nil {
		return nil
	}
	if sel.Fields != nil {
		return fmt.Errorf("ValidateTemplate: %s accepts only a boolean (got a field map)", label)
	}
	return nil
}

func validateProfileSection(p *ProfileSection) error {
	if p.Name != nil {
		name := *p.Name
		if name == "" {
			return fmt.Errorf("ValidateTemplate: profile.name is empty")
		}
		if len(utf16.Encode([]rune(name))) > MaxProfileNameUTF16Units {
			return fmt.Errorf("ValidateTemplate: profile.name exceeds %d UTF-16 code units", MaxProfileNameUTF16Units)
		}
	}
	if p.Level != nil {
		if *p.Level < 1 || *p.Level > MaxProfileLevel {
			return fmt.Errorf("ValidateTemplate: profile.level=%d out of range [1, %d]", *p.Level, MaxProfileLevel)
		}
	}
	if p.Class != nil && *p.Class == "" {
		return fmt.Errorf("ValidateTemplate: profile.class is empty")
	}
	if p.ClearCount != nil && *p.ClearCount > MaxProfileClearCount {
		return fmt.Errorf("ValidateTemplate: profile.clearCount=%d out of range [0, %d]", *p.ClearCount, MaxProfileClearCount)
	}
	if p.ScadutreeBlessing != nil && *p.ScadutreeBlessing > MaxProfileScadutreeBlessing {
		return fmt.Errorf("ValidateTemplate: profile.scadutreeBlessing=%d out of range [0, %d]", *p.ScadutreeBlessing, MaxProfileScadutreeBlessing)
	}
	if p.ShadowRealmBlessing != nil && *p.ShadowRealmBlessing > MaxProfileShadowRealmBlessing {
		return fmt.Errorf("ValidateTemplate: profile.shadowRealmBlessing=%d out of range [0, %d]", *p.ShadowRealmBlessing, MaxProfileShadowRealmBlessing)
	}
	if p.TalismanSlots != nil && *p.TalismanSlots > MaxProfileTalismanSlots {
		return fmt.Errorf("ValidateTemplate: profile.talismanSlots=%d out of range [0, %d]", *p.TalismanSlots, MaxProfileTalismanSlots)
	}
	return nil
}

func validateStatsSection(s *StatsSection) error {
	stats := []struct {
		name string
		val  *uint32
	}{
		{"vigor", s.Vigor},
		{"mind", s.Mind},
		{"endurance", s.Endurance},
		{"strength", s.Strength},
		{"dexterity", s.Dexterity},
		{"intelligence", s.Intelligence},
		{"faith", s.Faith},
		{"arcane", s.Arcane},
	}
	for _, st := range stats {
		if st.val == nil {
			continue
		}
		if *st.val < MinStatValue || *st.val > MaxStatValue {
			return fmt.Errorf("ValidateTemplate: stats.%s=%d out of range [%d, %d]", st.name, *st.val, MinStatValue, MaxStatValue)
		}
	}
	return nil
}

func validateEquipmentSection(eq *EquipmentSection) error {
	for _, slotKey := range equipmentSlotOrder {
		ref := equipmentSlotRef(eq, slotKey)
		if ref == nil {
			continue
		}
		if err := validateEquipmentItemRef(slotKey, ref); err != nil {
			return err
		}
	}
	return nil
}

func validateEquipmentItemRef(slotKey string, ref *EquipmentItemRef) error {
	if ref.Upgrade != nil {
		if *ref.Upgrade < 0 {
			return fmt.Errorf("ValidateTemplate: equipment.%s.upgrade=%d is negative", slotKey, *ref.Upgrade)
		}
		if *ref.Upgrade > MaxEquipmentItemUpgrade {
			return fmt.Errorf("ValidateTemplate: equipment.%s.upgrade=%d out of range [0, %d]", slotKey, *ref.Upgrade, MaxEquipmentItemUpgrade)
		}
	}
	if ref.AoWItemID != nil && *ref.AoWItemID == 0 {
		return fmt.Errorf("ValidateTemplate: equipment.%s.aowItemID=0 is invalid (omit the field to mean any-AoW)", slotKey)
	}
	return nil
}

func validateSpellsSection(s *SpellsSection) error {
	for _, slotKey := range spellSlotOrder {
		ref := spellSlotRef(s, slotKey)
		if ref == nil {
			continue
		}
		if err := validateSpellSlotRef(slotKey, ref); err != nil {
			return err
		}
	}
	return nil
}

func validateSpellSlotRef(slotKey string, ref *SpellSlotRef) error {
	if ref.BaseItemID == 0 {
		return nil
	}
	if (ref.BaseItemID & SpellItemIDPrefixMask) != SpellItemIDPrefix {
		return fmt.Errorf("ValidateTemplate: spells.%s.baseItemID=0x%08X has wrong prefix (expected 0x4XXXXXXX)", slotKey, ref.BaseItemID)
	}
	return nil
}

func validateItemsSection(s *ItemsSection) error {
	if s == nil {
		return nil
	}
	seen := make(map[string]int, len(s.Entries))
	for i := range s.Entries {
		e := &s.Entries[i]
		if e.EntryID == "" {
			return fmt.Errorf("ValidateTemplate: items.entries[%d]: entryID is empty", i)
		}
		if prev, dup := seen[e.EntryID]; dup {
			return fmt.Errorf("ValidateTemplate: items.entries[%d]: entryID %q already used at index %d", i, e.EntryID, prev)
		}
		seen[e.EntryID] = i
		if e.ItemID == 0 {
			return fmt.Errorf("ValidateTemplate: items.entries[%d]: itemID=0 (entryID=%q)", i, e.EntryID)
		}
		if !itemCategoryAllowlist[e.Category] {
			return fmt.Errorf("ValidateTemplate: items.entries[%d]: unknown category %q (entryID=%q)", i, e.Category, e.EntryID)
		}
		if e.Quantity == 0 {
			return fmt.Errorf("ValidateTemplate: items.entries[%d]: quantity=0 not allowed (entryID=%q); clear/remove semantics belong to applyOptions.items.mode, not entry payload", i, e.EntryID)
		}
		if !itemLocationAllowlist[e.Location] {
			return fmt.Errorf("ValidateTemplate: items.entries[%d]: unknown location %q (entryID=%q; allowed: inventory, storage, both)", i, e.Location, e.EntryID)
		}
		if err := validateUpgradeKindAndLevel(i, e); err != nil {
			return err
		}
		if e.AshOfWarItemID != nil && *e.AshOfWarItemID == 0 {
			return fmt.Errorf("ValidateTemplate: items.entries[%d]: ashOfWarItemID=0 (entryID=%q; omit the field to mean no custom AoW)", i, e.EntryID)
		}
	}
	return nil
}

func validateUpgradeKindAndLevel(i int, e *TemplateItemEntryV2) error {
	switch e.UpgradeKind {
	case "", UpgradeKindNone:
		if e.UpgradeLevel != nil {
			return fmt.Errorf("ValidateTemplate: items.entries[%d]: upgradeLevel=%d set but upgradeKind=%q (entryID=%q; omit upgradeLevel for non-upgradable items)", i, *e.UpgradeLevel, e.UpgradeKind, e.EntryID)
		}
		return nil
	case UpgradeKindStandard:
		if e.UpgradeLevel == nil {
			return fmt.Errorf("ValidateTemplate: items.entries[%d]: upgradeKind=standard requires upgradeLevel (entryID=%q)", i, e.EntryID)
		}
		if *e.UpgradeLevel > MaxItemUpgradeStandard {
			return fmt.Errorf("ValidateTemplate: items.entries[%d]: upgradeLevel=%d out of range [0, %d] for upgradeKind=standard (entryID=%q)", i, *e.UpgradeLevel, MaxItemUpgradeStandard, e.EntryID)
		}
		return nil
	case UpgradeKindSomber:
		if e.UpgradeLevel == nil {
			return fmt.Errorf("ValidateTemplate: items.entries[%d]: upgradeKind=somber requires upgradeLevel (entryID=%q)", i, e.EntryID)
		}
		if *e.UpgradeLevel > MaxItemUpgradeSomber {
			return fmt.Errorf("ValidateTemplate: items.entries[%d]: upgradeLevel=%d out of range [0, %d] for upgradeKind=somber (entryID=%q)", i, *e.UpgradeLevel, MaxItemUpgradeSomber, e.EntryID)
		}
		return nil
	default:
		return fmt.Errorf("ValidateTemplate: items.entries[%d]: unknown upgradeKind %q (entryID=%q; allowed: standard, somber, none)", i, e.UpgradeKind, e.EntryID)
	}
}

func validateLayoutSection(label string, entries []LayoutEntry, items *ItemsSection) error {
	knownRefs := map[string]bool{}
	if items != nil {
		for i := range items.Entries {
			knownRefs[items.Entries[i].EntryID] = true
		}
	}
	seenRefs := make(map[string]int, len(entries))
	seenPositions := make(map[int]int, len(entries))
	for i, le := range entries {
		if le.EntryRef == "" {
			return fmt.Errorf("ValidateTemplate: %s[%d]: entryRef is empty", label, i)
		}
		if !knownRefs[le.EntryRef] {
			return fmt.Errorf("ValidateTemplate: %s[%d]: entryRef %q does not match any items.entries.entryID", label, i, le.EntryRef)
		}
		if prev, dup := seenRefs[le.EntryRef]; dup {
			return fmt.Errorf("ValidateTemplate: %s[%d]: entryRef %q already used at index %d", label, i, le.EntryRef, prev)
		}
		seenRefs[le.EntryRef] = i
		if prev, dup := seenPositions[le.Position]; dup {
			return fmt.Errorf("ValidateTemplate: %s[%d]: position=%d already used at index %d (entryRef=%q)", label, i, le.Position, prev, le.EntryRef)
		}
		seenPositions[le.Position] = i
	}
	return nil
}

func validateApplyOptions(o *ApplyOptions) error {
	if o == nil {
		return nil
	}
	if o.Items != nil {
		if !itemApplyModeAllowlist[o.Items.Mode] {
			return fmt.Errorf("ValidateTemplate: applyOptions.items.mode=%q is invalid (allowed: addMissing, updateExisting, merge, replace)", o.Items.Mode)
		}
	}
	if o.InventoryLayout != nil {
		if !layoutApplyModeAllowlist[o.InventoryLayout.Mode] {
			return fmt.Errorf("ValidateTemplate: applyOptions.inventoryLayout.mode=%q is invalid (allowed: ignore, append, reorderOnly, replace)", o.InventoryLayout.Mode)
		}
	}
	if o.StorageLayout != nil {
		if !layoutApplyModeAllowlist[o.StorageLayout.Mode] {
			return fmt.Errorf("ValidateTemplate: applyOptions.storageLayout.mode=%q is invalid (allowed: ignore, append, reorderOnly, replace)", o.StorageLayout.Mode)
		}
	}
	if o.WeaponLevelOverride != nil {
		if err := validateWeaponLevelOverride(o.WeaponLevelOverride); err != nil {
			return err
		}
	}
	return nil
}

func validateWeaponLevelOverride(o *WeaponLevelOverride) error {
	if o.UseTemplateLevels {
		if o.StandardOverride != nil || o.SomberOverride != nil {
			return fmt.Errorf("ValidateTemplate: applyOptions.weaponLevelOverride: useTemplateLevels=true is mutually exclusive with standardOverride/somberOverride")
		}
		return nil
	}
	if o.StandardOverride != nil && *o.StandardOverride > MaxItemUpgradeStandard {
		return fmt.Errorf("ValidateTemplate: applyOptions.weaponLevelOverride.standardOverride=%d out of range [0, %d]", *o.StandardOverride, MaxItemUpgradeStandard)
	}
	if o.SomberOverride != nil && *o.SomberOverride > MaxItemUpgradeSomber {
		return fmt.Errorf("ValidateTemplate: applyOptions.weaponLevelOverride.somberOverride=%d out of range [0, %d]", *o.SomberOverride, MaxItemUpgradeSomber)
	}
	return nil
}
