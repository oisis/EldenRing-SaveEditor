package buildtemplates

import (
	"encoding/json"
	"fmt"
)

// SchemaKey identifies a build template payload.
const SchemaKey = "saveforge.build-template"

// Schema versions.
const (
	SchemaVersionV1  = 1
	MaxSchemaVersion = 2
)

// Player profile and stats constraints.
const (
	MaxProfileLevel               = 713
	MaxProfileClearCount          = 7
	MaxProfileScadutreeBlessing   = 20
	MaxProfileShadowRealmBlessing = 10
	MaxProfileTalismanSlots       = 3
	MaxProfileNameUTF16Units      = 16
	MinStatValue                  = 1
	MaxStatValue                  = 99
)

// Equipment and spell constants.
const (
	MaxEquipmentItemUpgrade        = 25
	SpellItemIDPrefix       uint32 = 0x40000000
	SpellItemIDPrefixMask   uint32 = 0xF0000000
)

// Item categories in v2 items section.
const (
	ItemCategoryMeleeArmaments      = "melee_armaments"
	ItemCategoryRangedAndCatalysts  = "ranged_and_catalysts"
	ItemCategoryShields             = "shields"
	ItemCategoryAshesOfWar          = "ashes_of_war"
	ItemCategoryArmorHead           = "head"
	ItemCategoryArmorChest          = "chest"
	ItemCategoryArmorArms           = "arms"
	ItemCategoryArmorLegs           = "legs"
	ItemCategoryTalismans           = "talismans"
	ItemCategorySorceries           = "sorceries"
	ItemCategoryIncantations        = "incantations"
	ItemCategoryTools               = "tools"
	ItemCategoryCraftingMaterials   = "crafting_materials"
	ItemCategoryBolsteringMaterials = "bolstering_materials"
	ItemCategoryArrowsAndBolts      = "arrows_and_bolts"
	ItemCategoryKeyItems            = "key_items"
	ItemCategoryGestures            = "gestures"
	ItemCategoryDLC                 = "dlc"
)

var itemCategoryAllowlist = map[string]bool{
	ItemCategoryMeleeArmaments:      true,
	ItemCategoryRangedAndCatalysts:  true,
	ItemCategoryShields:             true,
	ItemCategoryAshesOfWar:          true,
	ItemCategoryArmorHead:           true,
	ItemCategoryArmorChest:          true,
	ItemCategoryArmorArms:           true,
	ItemCategoryArmorLegs:           true,
	ItemCategoryTalismans:           true,
	ItemCategorySorceries:           true,
	ItemCategoryIncantations:        true,
	ItemCategoryTools:               true,
	ItemCategoryCraftingMaterials:   true,
	ItemCategoryBolsteringMaterials: true,
	ItemCategoryArrowsAndBolts:      true,
	ItemCategoryKeyItems:            true,
	ItemCategoryGestures:            true,
	ItemCategoryDLC:                 true,
}

// Upgrade kinds in v2 items section.
const (
	UpgradeKindNone     = "none"
	UpgradeKindStandard = "standard"
	UpgradeKindSomber   = "somber"
)

const (
	MaxItemUpgradeStandard = 25
	MaxItemUpgradeSomber   = 10
)

// Item locations in v2 items section.
const (
	ItemLocationInventory = "inventory"
	ItemLocationStorage   = "storage"
	ItemLocationBoth      = "both"
)

var itemLocationAllowlist = map[string]bool{
	ItemLocationInventory: true,
	ItemLocationStorage:   true,
	ItemLocationBoth:      true,
}

// Item apply modes in apply options.
const (
	ItemApplyModeAddMissing     = "addMissing"
	ItemApplyModeUpdateExisting = "updateExisting"
	ItemApplyModeMerge          = "merge"
	ItemApplyModeReplace        = "replace"
)

var itemApplyModeAllowlist = map[string]bool{
	ItemApplyModeAddMissing:     true,
	ItemApplyModeUpdateExisting: true,
	ItemApplyModeMerge:          true,
	ItemApplyModeReplace:        true,
}

// Layout apply modes in apply options.
const (
	LayoutApplyModeIgnore      = "ignore"
	LayoutApplyModeAppend      = "append"
	LayoutApplyModeReorderOnly = "reorderOnly"
	LayoutApplyModeReplace     = "replace"
)

var layoutApplyModeAllowlist = map[string]bool{
	LayoutApplyModeIgnore:      true,
	LayoutApplyModeAppend:      true,
	LayoutApplyModeReorderOnly: true,
	LayoutApplyModeReplace:     true,
}

// Container values in v1 workspace.
const (
	ContainerInventory = "inventory"
	ContainerStorage   = "storage"
)

// BuildTemplate is the portable on-wire and on-disk representation of a Build Template.
type BuildTemplate struct {
	Schema       string               `json:"schema"`
	Version      int                  `json:"version"`
	CreatedAt    string               `json:"createdAt"`
	AppVersion   string               `json:"appVersion,omitempty"`
	Metadata     *TemplateDocMetadata `json:"metadata,omitempty"`
	Selection    *TemplateSelection   `json:"selection,omitempty"`
	Sections     TemplateSections     `json:"sections"`
	ApplyOptions *ApplyOptions        `json:"applyOptions,omitempty"`
}

// TemplateDocMetadata carries human-readable metadata inside the template document.
type TemplateDocMetadata struct {
	Name                 string   `json:"name,omitempty"`
	Description          string   `json:"description,omitempty"`
	Author               string   `json:"author,omitempty"`
	Tags                 []string `json:"tags,omitempty"`
	SourceCharacterIndex int      `json:"sourceCharacterIndex,omitempty"`
	SourceCharacterName  string   `json:"sourceCharacterName,omitempty"`
}

// TemplateSections groups payload sections by stable key.
type TemplateSections struct {
	InventoryWorkspace *InventoryWorkspaceSection `json:"inventory.workspace,omitempty"`
	Profile            *ProfileSection            `json:"profile,omitempty"`
	Stats              *StatsSection              `json:"stats,omitempty"`
	Equipment          *EquipmentSection          `json:"equipment,omitempty"`
	Spells             *SpellsSection             `json:"spells,omitempty"`
	Items              *ItemsSection              `json:"items,omitempty"`
	InventoryLayout    *InventoryLayoutSection    `json:"inventoryLayout,omitempty"`
	StorageLayout      *StorageLayoutSection      `json:"storageLayout,omitempty"`
}

// InventoryWorkspaceSection is the v1 payload containing inventory and storage items.
type InventoryWorkspaceSection struct {
	InventoryItems []TemplateItem `json:"inventoryItems"`
	StorageItems   []TemplateItem `json:"storageItems"`
}

// TemplateItem describes a single item in v1 inventory.workspace.
type TemplateItem struct {
	BaseItemID   uint32  `json:"baseItemID"`
	Name         string  `json:"name,omitempty"`
	Category     string  `json:"category,omitempty"`
	Quantity     uint32  `json:"quantity"`
	Upgrade      int     `json:"upgrade,omitempty"`
	InfusionName string  `json:"infusionName,omitempty"`
	AoWItemID    *uint32 `json:"aowItemID,omitempty"`
	Container    string  `json:"container"`
	Position     int     `json:"position"`
}

// ProfileSection carries single-character profile fields.
type ProfileSection struct {
	Name                *string `json:"name,omitempty"`
	Level               *uint32 `json:"level,omitempty"`
	Runes               *uint32 `json:"runes,omitempty"`
	SoulMemory          *uint32 `json:"soulMemory,omitempty"`
	Class               *string `json:"class,omitempty"`
	ClearCount          *uint32 `json:"clearCount,omitempty"`
	ScadutreeBlessing   *uint8  `json:"scadutreeBlessing,omitempty"`
	ShadowRealmBlessing *uint8  `json:"shadowRealmBlessing,omitempty"`
	TalismanSlots       *uint8  `json:"talismanSlots,omitempty"`
}

// StatsSection carries the 8 character attributes.
type StatsSection struct {
	Vigor        *uint32 `json:"vigor,omitempty"`
	Mind         *uint32 `json:"mind,omitempty"`
	Endurance    *uint32 `json:"endurance,omitempty"`
	Strength     *uint32 `json:"strength,omitempty"`
	Dexterity    *uint32 `json:"dexterity,omitempty"`
	Intelligence *uint32 `json:"intelligence,omitempty"`
	Faith        *uint32 `json:"faith,omitempty"`
	Arcane       *uint32 `json:"arcane,omitempty"`
}

// EquipmentSection carries the equipped armaments, ammo, armor, and talismans.
type EquipmentSection struct {
	WeaponLeftHand1  *EquipmentItemRef `json:"weaponLeftHand1,omitempty"`
	WeaponRightHand1 *EquipmentItemRef `json:"weaponRightHand1,omitempty"`
	WeaponLeftHand2  *EquipmentItemRef `json:"weaponLeftHand2,omitempty"`
	WeaponRightHand2 *EquipmentItemRef `json:"weaponRightHand2,omitempty"`
	WeaponLeftHand3  *EquipmentItemRef `json:"weaponLeftHand3,omitempty"`
	WeaponRightHand3 *EquipmentItemRef `json:"weaponRightHand3,omitempty"`
	Arrows1          *EquipmentItemRef `json:"arrows1,omitempty"`
	Bolts1           *EquipmentItemRef `json:"bolts1,omitempty"`
	Arrows2          *EquipmentItemRef `json:"arrows2,omitempty"`
	Bolts2           *EquipmentItemRef `json:"bolts2,omitempty"`
	ArmorHead        *EquipmentItemRef `json:"armorHead,omitempty"`
	ArmorChest       *EquipmentItemRef `json:"armorChest,omitempty"`
	ArmorArms        *EquipmentItemRef `json:"armorArms,omitempty"`
	ArmorLegs        *EquipmentItemRef `json:"armorLegs,omitempty"`
	Talisman1        *EquipmentItemRef `json:"talisman1,omitempty"`
	Talisman2        *EquipmentItemRef `json:"talisman2,omitempty"`
	Talisman3        *EquipmentItemRef `json:"talisman3,omitempty"`
	Talisman4        *EquipmentItemRef `json:"talisman4,omitempty"`
	Talisman5        *EquipmentItemRef `json:"talisman5,omitempty"`
}

// EquipmentItemRef points to one item to equip into a slot.
type EquipmentItemRef struct {
	BaseItemID   uint32  `json:"baseItemID"`
	Name         string  `json:"name,omitempty"`
	Upgrade      *int    `json:"upgrade,omitempty"`
	InfusionName string  `json:"infusionName,omitempty"`
	AoWItemID    *uint32 `json:"aowItemID,omitempty"`
}

// SpellsSection captures up to 14 equipped-spell slots.
type SpellsSection struct {
	Spell1  *SpellSlotRef `json:"spell1,omitempty"`
	Spell2  *SpellSlotRef `json:"spell2,omitempty"`
	Spell3  *SpellSlotRef `json:"spell3,omitempty"`
	Spell4  *SpellSlotRef `json:"spell4,omitempty"`
	Spell5  *SpellSlotRef `json:"spell5,omitempty"`
	Spell6  *SpellSlotRef `json:"spell6,omitempty"`
	Spell7  *SpellSlotRef `json:"spell7,omitempty"`
	Spell8  *SpellSlotRef `json:"spell8,omitempty"`
	Spell9  *SpellSlotRef `json:"spell9,omitempty"`
	Spell10 *SpellSlotRef `json:"spell10,omitempty"`
	Spell11 *SpellSlotRef `json:"spell11,omitempty"`
	Spell12 *SpellSlotRef `json:"spell12,omitempty"`
	Spell13 *SpellSlotRef `json:"spell13,omitempty"`
	Spell14 *SpellSlotRef `json:"spell14,omitempty"`
}

// SpellSlotRef is one equipped-spell slot reference.
type SpellSlotRef struct {
	BaseItemID uint32 `json:"baseItemID"`
	Name       string `json:"name,omitempty"`
}

// ItemsSection is the v2 decoupled items section.
type ItemsSection struct {
	Entries []TemplateItemEntryV2 `json:"entries"`
}

// TemplateItemEntryV2 is a single item entry in v2 items.
type TemplateItemEntryV2 struct {
	EntryID        string  `json:"entryID"`
	ItemID         uint32  `json:"itemID"`
	Name           string  `json:"name,omitempty"`
	Category       string  `json:"category"`
	Quantity       uint32  `json:"quantity"`
	Location       string  `json:"location"`
	UpgradeKind    string  `json:"upgradeKind,omitempty"`
	UpgradeLevel   *uint8  `json:"upgradeLevel,omitempty"`
	InfusionName   string  `json:"infusionName,omitempty"`
	AshOfWarItemID *uint32 `json:"ashOfWarItemID,omitempty"`
}

// InventoryLayoutSection is the inventory container ordering section.
type InventoryLayoutSection struct {
	Entries []LayoutEntry `json:"entries"`
}

// StorageLayoutSection is the storage container ordering section.
type StorageLayoutSection struct {
	Entries []LayoutEntry `json:"entries"`
}

// LayoutEntry is one ordered entry in a layout section.
type LayoutEntry struct {
	EntryRef string `json:"entryRef"`
	Position int    `json:"position"`
}

// TemplateSelection is the v2 section selection descriptor.
type TemplateSelection struct {
	Profile            *SectionSelection `json:"profile,omitempty"`
	Stats              *SectionSelection `json:"stats,omitempty"`
	InventoryWorkspace *SectionSelection `json:"inventory.workspace,omitempty"`
	Equipment          *SectionSelection `json:"equipment,omitempty"`
	Spells             *SectionSelection `json:"spells,omitempty"`
	Items              *SectionSelection `json:"items,omitempty"`
	InventoryLayout    *SectionSelection `json:"inventoryLayout,omitempty"`
	StorageLayout      *SectionSelection `json:"storageLayout,omitempty"`
}

// SectionSelection is a per-section toggle: either boolean shortcut (All) or per-field map.
type SectionSelection struct {
	All    bool            `json:"-"`
	Fields map[string]bool `json:"-"`
}

// HasAny reports whether at least one field or the All shortcut is selected.
func (s *SectionSelection) HasAny() bool {
	if s == nil {
		return false
	}
	if s.All {
		return true
	}
	for _, v := range s.Fields {
		if v {
			return true
		}
	}
	return false
}

// HasAnySelected reports whether TemplateSelection selects at least one section.
func (t *TemplateSelection) HasAnySelected() bool {
	if t == nil {
		return false
	}
	return t.Profile.HasAny() ||
		t.Stats.HasAny() ||
		t.InventoryWorkspace.HasAny() ||
		t.Equipment.HasAny() ||
		t.Spells.HasAny() ||
		t.Items.HasAny() ||
		t.InventoryLayout.HasAny() ||
		t.StorageLayout.HasAny()
}

// UnmarshalJSON parses a boolean shortcut or a JSON map of field names to booleans.
func (s *SectionSelection) UnmarshalJSON(data []byte) error {
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		s.All = b
		s.Fields = nil
		return nil
	}
	var m map[string]bool
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("SectionSelection: must be a boolean or a map of field->bool: %w", err)
	}
	s.All = false
	s.Fields = m
	return nil
}

// MarshalJSON serializes as boolean or field map.
func (s SectionSelection) MarshalJSON() ([]byte, error) {
	if s.Fields != nil {
		return json.Marshal(s.Fields)
	}
	return json.Marshal(s.All)
}

// ApplyOptions configures how a template is applied.
type ApplyOptions struct {
	Items               *ItemApplyOptions    `json:"items,omitempty"`
	InventoryLayout     *LayoutApplyOptions  `json:"inventoryLayout,omitempty"`
	StorageLayout       *LayoutApplyOptions  `json:"storageLayout,omitempty"`
	WeaponLevelOverride *WeaponLevelOverride `json:"weaponLevelOverride,omitempty"`
}

// ItemApplyOptions controls item reconciliation mode.
type ItemApplyOptions struct {
	Mode               string `json:"mode"`
	PreserveExtraItems bool   `json:"preserveExtraItems,omitempty"`
}

// LayoutApplyOptions controls layout reconciliation mode.
type LayoutApplyOptions struct {
	Mode string `json:"mode"`
}

// WeaponLevelOverride configures weapon level overriding at apply time.
type WeaponLevelOverride struct {
	UseTemplateLevels bool   `json:"useTemplateLevels"`
	StandardOverride  *uint8 `json:"standardOverride,omitempty"`
	SomberOverride    *uint8 `json:"somberOverride,omitempty"`
}

// Canonical equipment slot keys in deterministic iteration order.
var equipmentSlotOrder = []string{
	"weaponLeftHand1",
	"weaponRightHand1",
	"weaponLeftHand2",
	"weaponRightHand2",
	"weaponLeftHand3",
	"weaponRightHand3",
	"arrows1",
	"bolts1",
	"arrows2",
	"bolts2",
	"armorHead",
	"armorChest",
	"armorArms",
	"armorLegs",
	"talisman1",
	"talisman2",
	"talisman3",
	"talisman4",
	"talisman5",
}

// Canonical spell slot keys in deterministic iteration order.
var spellSlotOrder = []string{
	"spell1", "spell2", "spell3", "spell4", "spell5", "spell6", "spell7",
	"spell8", "spell9", "spell10", "spell11", "spell12", "spell13", "spell14",
}

var profileSelectionFields = map[string]bool{
	"name":                true,
	"level":               true,
	"runes":               true,
	"soulMemory":          true,
	"class":               true,
	"clearCount":          true,
	"scadutreeBlessing":   true,
	"shadowRealmBlessing": true,
	"talismanSlots":       true,
}

var statsSelectionFields = map[string]bool{
	"vigor":        true,
	"mind":         true,
	"endurance":    true,
	"strength":     true,
	"dexterity":    true,
	"intelligence": true,
	"faith":        true,
	"arcane":       true,
}

var equipmentSelectionFields = map[string]bool{
	"weaponLeftHand1":  true,
	"weaponRightHand1": true,
	"weaponLeftHand2":  true,
	"weaponRightHand2": true,
	"weaponLeftHand3":  true,
	"weaponRightHand3": true,
	"arrows1":          true,
	"bolts1":           true,
	"arrows2":          true,
	"bolts2":           true,
	"armorHead":        true,
	"armorChest":       true,
	"armorArms":        true,
	"armorLegs":        true,
	"talisman1":        true,
	"talisman2":        true,
	"talisman3":        true,
	"talisman4":        true,
	"talisman5":        true,
}

var spellsSelectionFields = map[string]bool{
	"spell1": true, "spell2": true, "spell3": true, "spell4": true,
	"spell5": true, "spell6": true, "spell7": true, "spell8": true,
	"spell9": true, "spell10": true, "spell11": true, "spell12": true,
	"spell13": true, "spell14": true,
}

func equipmentSlotRef(eq *EquipmentSection, slotKey string) *EquipmentItemRef {
	if eq == nil {
		return nil
	}
	switch slotKey {
	case "weaponLeftHand1":
		return eq.WeaponLeftHand1
	case "weaponRightHand1":
		return eq.WeaponRightHand1
	case "weaponLeftHand2":
		return eq.WeaponLeftHand2
	case "weaponRightHand2":
		return eq.WeaponRightHand2
	case "weaponLeftHand3":
		return eq.WeaponLeftHand3
	case "weaponRightHand3":
		return eq.WeaponRightHand3
	case "arrows1":
		return eq.Arrows1
	case "bolts1":
		return eq.Bolts1
	case "arrows2":
		return eq.Arrows2
	case "bolts2":
		return eq.Bolts2
	case "armorHead":
		return eq.ArmorHead
	case "armorChest":
		return eq.ArmorChest
	case "armorArms":
		return eq.ArmorArms
	case "armorLegs":
		return eq.ArmorLegs
	case "talisman1":
		return eq.Talisman1
	case "talisman2":
		return eq.Talisman2
	case "talisman3":
		return eq.Talisman3
	case "talisman4":
		return eq.Talisman4
	case "talisman5":
		return eq.Talisman5
	default:
		return nil
	}
}

func spellSlotRef(s *SpellsSection, slotKey string) *SpellSlotRef {
	if s == nil {
		return nil
	}
	switch slotKey {
	case "spell1":
		return s.Spell1
	case "spell2":
		return s.Spell2
	case "spell3":
		return s.Spell3
	case "spell4":
		return s.Spell4
	case "spell5":
		return s.Spell5
	case "spell6":
		return s.Spell6
	case "spell7":
		return s.Spell7
	case "spell8":
		return s.Spell8
	case "spell9":
		return s.Spell9
	case "spell10":
		return s.Spell10
	case "spell11":
		return s.Spell11
	case "spell12":
		return s.Spell12
	case "spell13":
		return s.Spell13
	case "spell14":
		return s.Spell14
	default:
		return nil
	}
}
