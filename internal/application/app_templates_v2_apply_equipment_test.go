package application

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/editor"
	"github.com/oisis/EldenRing-SaveForge/backend/templates"
)

// equipmentApplyTemplateJSON marshals a v2 template that selects equipment
// (All shortcut) and, optionally, profile.talismanSlots, so an apply
// integration test can exercise the effective-pouch-capacity gate.
func equipmentApplyTemplateJSON(t *testing.T, talismanSlots *uint8, sec *templates.EquipmentSection) string {
	t.Helper()
	sel := &templates.TemplateSelection{Equipment: &templates.SectionSelection{All: true}}
	sections := templates.TemplateSections{Equipment: sec}
	if talismanSlots != nil {
		sel.Profile = &templates.SectionSelection{Fields: map[string]bool{"talismanSlots": true}}
		sections.Profile = &templates.ProfileSection{TalismanSlots: talismanSlots}
	}
	tpl := &templates.BuildTemplate{
		Schema:    templates.SchemaKey,
		Version:   2,
		CreatedAt: "2026-06-02T00:00:00Z",
		Selection: sel,
		Sections:  sections,
	}
	out, err := json.Marshal(tpl)
	if err != nil {
		t.Fatalf("marshal template: %v", err)
	}
	return string(out)
}

func countWarningsByCode(warnings []templates.ImportPreviewIssue, code, container string) int {
	n := 0
	for _, w := range warnings {
		if w.Code == code && w.Container == container {
			n++
		}
	}
	return n
}

// A talisman slot the target has not unlocked is skipped with the
// pouch-insufficient code — not applied, not a generic slot-invalid warning.
func TestApplyV2_Equipment_LockedTalismanSlot_PouchInsufficient(t *testing.T) {
	app := spellsApplyFixture(t)
	app.save.Slots[0].Player.TalismanSlots = 0 // one active pouch slot
	jsonText := equipmentApplyTemplateJSON(t, nil, &templates.EquipmentSection{
		Talisman4: &templates.EquipmentItemRef{BaseItemID: 0x200003E8},
	})
	res, err := app.ApplyBuildTemplateV2ToCharacterJSON(0, jsonText, ApplyTemplateV2Options{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if n := countWarningsByCode(res.Preview.Warnings, templates.IssueCodeTalismanSlotPouchInsufficient, "talisman4"); n != 1 {
		t.Fatalf("expected one pouch-insufficient warning for talisman4, got %+v", res.Preview.Warnings)
	}
}

// The same template that raises profile.talismanSlots to 3 unlocks talisman4
// within the SAME apply and actually WRITES the talisman: the resolver uses the
// post-profile effective capacity for the resolve gate, and WriteEquipment
// runs after the Player flush, so slot.Player.TalismanSlots is already 3 when
// the writer checks the pouch capacity. This is the full happy path, not just
// the warning-code transition.
func TestApplyV2_Equipment_ProfileUnlocksTalismanSlotSameApply(t *testing.T) {
	app := spellsApplyFixture(t)
	slot := &app.save.Slots[0]
	slot.Player.TalismanSlots = 0 // one active pouch slot before the template

	const (
		talismanHandle = core.ItemTypeAccessory | 0x000003E8 // Crimson Amber Medallion
		talismanID     = 0x200003E8
	)
	// Own the talisman: as the parsed struct (WriteEquipment inventory-row
	// lookup) AND as the raw inventory bytes (editor.BuildSnapshot, which the
	// equipment resolver reads).
	slot.Inventory.CommonItems = []core.InventoryItem{{GaItemHandle: talismanHandle, Quantity: 1}}
	invStart := slot.MagicOffset + core.InvStartFromMagic
	binary.LittleEndian.PutUint32(slot.Data[invStart:], talismanHandle)
	binary.LittleEndian.PutUint32(slot.Data[invStart+4:], 1)
	binary.LittleEndian.PutUint32(slot.Data[invStart+8:], 1000)

	three := uint8(3)
	jsonText := equipmentApplyTemplateJSON(t, &three, &templates.EquipmentSection{
		Talisman4: &templates.EquipmentItemRef{BaseItemID: talismanID},
	})
	res, err := app.ApplyBuildTemplateV2ToCharacterJSON(0, jsonText, ApplyTemplateV2Options{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !res.Applied {
		t.Fatalf("apply must succeed, got preview %+v", res.Preview)
	}
	if n := countWarningsByCode(res.Preview.Warnings, templates.IssueCodeTalismanSlotPouchInsufficient, "talisman4"); n != 0 {
		t.Fatalf("talisman4 must not be pouch-insufficient after profile unlock, got %+v", res.Preview.Warnings)
	}
	if n := countWarningsByCode(res.Preview.Warnings, templates.IssueCodeEquipmentItemNotInInventory, "talisman4"); n != 0 {
		t.Fatalf("talisman4 is owned; it must not be reported not-in-inventory, got %+v", res.Preview.Warnings)
	}
	if res.EquipmentSlotsApplied != 1 {
		t.Fatalf("EquipmentSlotsApplied = %d, want 1 (talisman4 written)", res.EquipmentSlotsApplied)
	}
	if slot.Player.TalismanSlots != 3 {
		t.Fatalf("Player.TalismanSlots = %d, want 3 after profile unlock", slot.Player.TalismanSlots)
	}

	raw, err := slot.ReadEquippedState()
	if err != nil {
		t.Fatalf("ReadEquippedState: %v", err)
	}
	if got := raw.Equipped[20]; got != talismanID {
		t.Errorf("Equipped[20] (talisman4) = 0x%08X, want 0x%08X", got, uint32(talismanID))
	}
	// ChrAsm2 native handle for slot index 20 must carry the owned GaItem handle.
	handleOff := slot.EquipItemsIDOffset + 1 + 20*4
	if got := binary.LittleEndian.Uint32(slot.Data[handleOff:]); got != talismanHandle {
		t.Errorf("ChrAsm2 talisman4 handle = 0x%08X, want 0x%08X", got, uint32(talismanHandle))
	}
}

// talisman5 has no native write contract: apply skips it with a slot-invalid
// warning and never produces a write.
func TestApplyV2_Equipment_Talisman5NotWritten(t *testing.T) {
	app := spellsApplyFixture(t)
	app.save.Slots[0].Player.TalismanSlots = 3
	jsonText := equipmentApplyTemplateJSON(t, nil, &templates.EquipmentSection{
		Talisman5: &templates.EquipmentItemRef{BaseItemID: 0x200003E8},
	})
	res, err := app.ApplyBuildTemplateV2ToCharacterJSON(0, jsonText, ApplyTemplateV2Options{})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if n := countWarningsByCode(res.Preview.Warnings, templates.IssueCodeEquipmentSlotInvalid, "talisman5"); n != 1 {
		t.Fatalf("expected talisman5 slot-invalid warning, got %+v", res.Preview.Warnings)
	}
	for _, f := range res.AppliedFields {
		if f == "equipment.talisman5" {
			t.Fatal("talisman5 must never be applied")
		}
	}
}

// allTalismansUnlocked is the target pouch capacity used by tests that are not
// exercising the locked-slot gate (all four talisman slots active).
const allTalismansUnlocked = 4

func TestResolveEquipmentWrites_MatchesOwnedWeapon(t *testing.T) {
	items := []editor.EditableItem{{BaseItemID: 0x100000, OriginalHandle: 0x80000010, IsWeapon: true}}
	sec := &templates.EquipmentSection{WeaponRightHand1: &templates.EquipmentItemRef{BaseItemID: 0x100000}}
	writes, warnings, err := resolveEquipmentWritesFromItems(items, allTalismansUnlocked, &templates.SectionSelection{All: true}, sec)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if len(writes) != 1 || writes[0] != (core.EquipmentWrite{Slot: core.EquipSlotRightHandArmament1, Handle: 0x80000010}) {
		t.Fatalf("writes = %+v", writes)
	}
}

func TestResolveEquipmentWrites_EquipsTalisman(t *testing.T) {
	// Talisman handle carries the accessory (0xA0) type prefix.
	items := []editor.EditableItem{{BaseItemID: 0x200003E8, OriginalHandle: 0xA0000010, IsTalisman: true}}
	sec := &templates.EquipmentSection{Talisman1: &templates.EquipmentItemRef{BaseItemID: 0x200003E8}}
	writes, warnings, err := resolveEquipmentWritesFromItems(items, allTalismansUnlocked, &templates.SectionSelection{All: true}, sec)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if len(writes) != 1 || writes[0] != (core.EquipmentWrite{Slot: core.EquipSlotTalisman1, Handle: 0xA0000010}) {
		t.Fatalf("writes = %+v", writes)
	}
}

func TestResolveEquipmentWrites_ClearsTalisman(t *testing.T) {
	sec := &templates.EquipmentSection{Talisman2: &templates.EquipmentItemRef{BaseItemID: 0}}
	writes, warnings, err := resolveEquipmentWritesFromItems(nil, allTalismansUnlocked, &templates.SectionSelection{All: true}, sec)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if len(writes) != 1 || writes[0] != (core.EquipmentWrite{Slot: core.EquipSlotTalisman2, Handle: 0}) {
		t.Fatalf("writes = %+v", writes)
	}
}

func TestResolveEquipmentWrites_SkipsLockedTalismanSlot(t *testing.T) {
	// Only the first pouch slot is unlocked; talisman2 (equip) and talisman3
	// (clear) are both past capacity and must be skipped — not passed to the
	// writer, which would reject the whole batch on the locked equip.
	items := []editor.EditableItem{{BaseItemID: 0x200003E8, OriginalHandle: 0xA0000010, IsTalisman: true}}
	sec := &templates.EquipmentSection{
		Talisman1: &templates.EquipmentItemRef{BaseItemID: 0x200003E8},
		Talisman2: &templates.EquipmentItemRef{BaseItemID: 0x200003E9},
		Talisman3: &templates.EquipmentItemRef{BaseItemID: 0}, // clear on a locked slot
	}
	writes, warnings, err := resolveEquipmentWritesFromItems(items, 1, &templates.SectionSelection{All: true}, sec)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if len(writes) != 1 || writes[0].Slot != core.EquipSlotTalisman1 {
		t.Fatalf("only talisman1 should produce a write, got %+v", writes)
	}
	// talisman2 and talisman3 each warn about the locked pouch slot.
	locked := 0
	for _, w := range warnings {
		if w.Code == templates.IssueCodeTalismanSlotPouchInsufficient {
			locked++
		}
	}
	if locked != 2 {
		t.Fatalf("expected 2 pouch-insufficient warnings, got %+v", warnings)
	}
}

func TestResolveEquipmentWrites_SkipsTalisman5(t *testing.T) {
	sec := &templates.EquipmentSection{Talisman5: &templates.EquipmentItemRef{BaseItemID: 0x200003E8}}
	writes, warnings, err := resolveEquipmentWritesFromItems(nil, allTalismansUnlocked, &templates.SectionSelection{All: true}, sec)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if len(writes) != 0 {
		t.Fatalf("talisman5 has no write contract: %+v", writes)
	}
	if len(warnings) != 1 || warnings[0].Code != templates.IssueCodeEquipmentSlotInvalid {
		t.Fatalf("warnings = %+v", warnings)
	}
}

func TestResolveEquipmentWrites_ExplicitClear(t *testing.T) {
	sec := &templates.EquipmentSection{ArmorHead: &templates.EquipmentItemRef{BaseItemID: 0}}
	writes, warnings, err := resolveEquipmentWritesFromItems(nil, allTalismansUnlocked, &templates.SectionSelection{All: true}, sec)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if len(writes) != 1 || writes[0] != (core.EquipmentWrite{Slot: core.EquipSlotHead, Handle: 0}) {
		t.Fatalf("writes = %+v", writes)
	}
}

func TestResolveEquipmentWrites_AmbiguousMatchWarns(t *testing.T) {
	items := []editor.EditableItem{
		{BaseItemID: 0x100000, OriginalHandle: 0x80000010, IsWeapon: true},
		{BaseItemID: 0x100000, OriginalHandle: 0x80000011, IsWeapon: true},
	}
	sec := &templates.EquipmentSection{WeaponRightHand1: &templates.EquipmentItemRef{BaseItemID: 0x100000}}
	writes, warnings, err := resolveEquipmentWritesFromItems(items, allTalismansUnlocked, &templates.SectionSelection{All: true}, sec)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if len(writes) != 1 || writes[0].Handle != 0x80000010 {
		t.Fatalf("writes = %+v", writes)
	}
	if len(warnings) != 1 || warnings[0].Code != templates.IssueCodeEquipmentItemAmbiguous {
		t.Fatalf("warnings = %+v", warnings)
	}
}
