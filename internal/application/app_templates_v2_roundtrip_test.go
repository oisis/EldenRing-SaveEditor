package application

import (
	"encoding/binary"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/templates"
)

// Native "empty" item the equipment writer resolves a weapon-slot clear to
// (Unarmed). Mirrors core.unarmedEquipmentItemID, which is unexported.
const roundTripUnarmedItemID uint32 = 0x0001ADB0

// calibrateLoadoutSlot mirrors spellsApplyFixture's slot-0 calibration for an
// arbitrary slot so the full apply pipeline (WriteEquipment / WriteSpells /
// Player flush) can run against it. Every equipped-spell slot starts at the
// empty-slot sentinel.
func calibrateLoadoutSlot(t *testing.T, slot *core.SaveSlot) {
	t.Helper()
	slot.Data = make([]byte, core.SlotSize)
	copy(slot.Data[core.FallbackMagicBase:], core.MagicPattern)
	slot.MagicOffset = core.FallbackMagicBase
	if err := slot.CalculateDynamicOffsets(); err != nil {
		t.Fatalf("calibrate slot: %v", err)
	}
	for i := 0; i < core.EquippedSpellSlotCount; i++ {
		off := slot.EquippedSpellsOffset + i*core.EquippedSpellSlotSize
		binary.LittleEndian.PutUint32(slot.Data[off:], core.EquippedSpellEmptySentinel)
		binary.LittleEndian.PutUint32(slot.Data[off+4:], 0)
	}
}

// seedRoundTripTargetInventory equips the target with the same item families the
// source loadout carries PLUS the native technical rows the equipment writer
// needs to CLEAR weapon/armor slots (Unarmed + the four bare-armor items).
//
// Weapon/armor/talisman are editable categories, so mirroring them into the raw
// inventory bytes lets editor.BuildSnapshot (used by the equipment resolver)
// resolve them via snap.InventoryItems. Ammo is a pass-through category: its raw
// row is written too, but BuildSnapshot never surfaces it in InventoryItems — the
// resolver finds ammo exclusively through Inventory.CommonItems + GaMap
// (collectAmmoCandidates). The clear-only rows live only in the parsed inventory
// + GaMap the writer consults.
func seedRoundTripTargetInventory(target *core.SaveSlot, items loadoutFixtureItems) {
	const (
		unarmedHandle   = core.ItemTypeWeapon | 0x00000101
		weaponHandle    = core.ItemTypeWeapon | 0x00000102
		armorHandle     = core.ItemTypeArmor | 0x00000103
		ammoHandle      = core.ItemTypeWeapon | 0x00000104
		bareHeadHandle  = core.ItemTypeArmor | 0x00000110
		bareChestHandle = core.ItemTypeArmor | 0x00000111
		bareArmsHandle  = core.ItemTypeArmor | 0x00000112
		bareLegsHandle  = core.ItemTypeArmor | 0x00000113
	)
	talismanHandle := items.talismanHandle

	target.GaMap = map[uint32]uint32{
		unarmedHandle:   roundTripUnarmedItemID,
		weaponHandle:    items.weaponFullID,
		armorHandle:     items.armorID,
		ammoHandle:      items.ammoID,
		bareHeadHandle:  0x10002710,
		bareChestHandle: 0x10002774,
		bareArmsHandle:  0x100027D8,
		bareLegsHandle:  0x1000283C,
	}

	// Parsed inventory the writer scans (inventoryRowForHandle / handleForItemID)
	// for both equips and native clears.
	target.Inventory.CommonItems = []core.InventoryItem{
		{GaItemHandle: weaponHandle, Quantity: 1},
		{GaItemHandle: armorHandle, Quantity: 1},
		{GaItemHandle: ammoHandle, Quantity: 1},
		{GaItemHandle: talismanHandle, Quantity: 1},
		{GaItemHandle: unarmedHandle, Quantity: 1},
		{GaItemHandle: bareHeadHandle, Quantity: 1},
		{GaItemHandle: bareChestHandle, Quantity: 1},
		{GaItemHandle: bareArmsHandle, Quantity: 1},
		{GaItemHandle: bareLegsHandle, Quantity: 1},
	}

	// Raw inventory bytes the equipment resolver's BuildSnapshot reads. Only the
	// three editable families (weapon/armor/talisman) actually resolve through
	// snap.InventoryItems; the ammo row is written for parity but is resolved via
	// CommonItems + GaMap instead (pass-through category).
	invStart := target.MagicOffset + core.InvStartFromMagic
	raw := []uint32{weaponHandle, armorHandle, ammoHandle, talismanHandle}
	for i, h := range raw {
		off := invStart + i*core.InvRecordLen
		binary.LittleEndian.PutUint32(target.Data[off:], h)
		binary.LittleEndian.PutUint32(target.Data[off+4:], 1)
		binary.LittleEndian.PutUint32(target.Data[off+8:], uint32(2000+i))
	}
}

// seedSpellOnSlot writes a raw MagicParam ID into the given equipped-spell slot
// of an arbitrary slot (seedSpell in the loadout tests is slot-0 only).
func seedSpellOnSlot(slot *core.SaveSlot, slotIdx int, rawID uint32) {
	off := slot.EquippedSpellsOffset + slotIdx*core.EquippedSpellSlotSize
	binary.LittleEndian.PutUint32(slot.Data[off:], rawID)
	binary.LittleEndian.PutUint32(slot.Data[off+4:], core.EquippedSpellOccupiedFollower)
}

// TestTemplateV2_CreatePreviewLibraryApply_RoundTrip drives the whole
// production chain end to end with NO hand-authored template JSON:
//
//	source slot 0 → PreviewBuildTemplateV2FromCharacter (profile+stats+equipment+spells)
//	  → SaveImportedBuildTemplateJSONToLibrary (exact preview.JSON)
//	  → PreviewBuildTemplateFromLibrary (reload canonical JSON)
//	  → ApplyBuildTemplateV2ToCharacterJSON on target slot 1 (no edit session)
//
// It proves profile/stats + the full equipment/spell loadout (including explicit
// clears of the target's stale gear) transfer intact, the source slot is never
// touched, and equipment/spells apply without a Sort Order workspace session.
func TestTemplateV2_CreatePreviewLibraryApply_RoundTrip(t *testing.T) {
	// Source: slot 0, fully equipped through the real SaveEquipment path.
	// newLoadoutFixture leaves arrows1 + arrows2 occupied with owned ammo, so
	// the loadout carries live ammo through the whole pipeline; the target owns
	// the matching ammo below and the resolver equips it directly from the
	// target's Inventory + GaMap (ammo is a pass-through category and never
	// enters the editable workspace).
	app, items := newLoadoutFixture(t)
	app.templateLibrary = templates.NewTemplateLibrary(t.TempDir())

	// Target: slot 1, calibrated, owning the item families + native clear rows.
	app.save.ActiveSlots[1] = true
	target := &app.save.Slots[1]
	calibrateLoadoutSlot(t, target)
	target.Version = 1 // non-empty slot so editor.BuildSnapshot scans it
	target.Player = core.PlayerGameData{
		Class: 0, Level: 10, Souls: 5, SoulMemory: 5,
		Vigor: 30, Mind: 30, Endurance: 30, Strength: 30,
		Dexterity: 30, Intelligence: 30, Faith: 30, Arcane: 30,
		ClearCount: 0, ScadutreeBlessing: 0, ShadowRealmBlessing: 0,
		TalismanSlots: 2, // 3 active pouch slots — matches the source
	}
	seedRoundTripTargetInventory(target, items)

	// Different starting loadout on the target: a weapon in RH2 (source leaves
	// RH2 empty → the template must CLEAR it) and a spell the source lacks in
	// slot 3 (→ must be cleared), plus a stale spell in slot 0 (→ overwritten).
	if err := app.SaveEquipment(1, []EquipmentChange{
		{Slot: core.EquipSlotRightHandArmament2, Handle: core.ItemTypeWeapon | 0x00000102},
		{Slot: core.EquipSlotTalisman1, Handle: items.talismanHandle},
	}); err != nil {
		t.Fatalf("seed target equipment: %v", err)
	}
	seedSpellOnSlot(target, 0, 0x0FA0) // Glintstone Pebble (source has Catch Flame)
	seedSpellOnSlot(target, 3, 0x0FA0) // extra spell the source will clear

	// ── Create → Preview (source) ──────────────────────────────────────
	sel := `{"profile":true,"stats":true,"equipment":true,"spells":true}`
	preview, err := app.PreviewBuildTemplateV2FromCharacter(0, sel, BuildTemplateV2ExportOptions{Name: "loadout"})
	if err != nil {
		t.Fatalf("PreviewBuildTemplateV2FromCharacter: %v", err)
	}
	if !preview.Report.OK {
		t.Fatalf("source preview not OK: %+v", preview.Report.Errors)
	}

	// ── Save to Library → reload (canonical JSON preserved) ─────────────
	entry, err := app.SaveImportedBuildTemplateJSONToLibrary(preview.JSON)
	if err != nil {
		t.Fatalf("SaveImportedBuildTemplateJSONToLibrary: %v", err)
	}
	libPreview, err := app.PreviewBuildTemplateFromLibrary(entry.ID)
	if err != nil {
		t.Fatalf("PreviewBuildTemplateFromLibrary: %v", err)
	}
	// The docs promise the library round-trips the exact canonical JSON.
	if libPreview.JSON != preview.JSON {
		t.Fatalf("library JSON is not byte-identical to the create preview JSON:\nsource=%s\nlibrary=%s", preview.JSON, libPreview.JSON)
	}
	libTpl := decodeExport(t, libPreview.JSON)
	if libTpl.Selection == nil || !libTpl.Selection.Equipment.HasAny() || !libTpl.Selection.Spells.HasAny() {
		t.Fatalf("library template lost equipment/spells selection: %+v", libTpl.Selection)
	}
	if libTpl.Sections.Equipment == nil || libTpl.Sections.Equipment.WeaponRightHand1 == nil {
		t.Fatalf("library template lost equipment section: %+v", libTpl.Sections.Equipment)
	}
	if libTpl.Sections.Spells == nil || libTpl.Sections.Spells.Spell1 == nil {
		t.Fatalf("library template lost spells section: %+v", libTpl.Sections.Spells)
	}

	// Snapshot the source slot to prove the apply never touches it.
	srcBefore := make([]byte, len(app.save.Slots[0].Data))
	copy(srcBefore, app.save.Slots[0].Data)

	// ── Apply the library's canonical JSON to the target (no session) ───
	res, err := app.ApplyBuildTemplateV2ToCharacterJSON(1, libPreview.JSON, ApplyTemplateV2Options{})
	if err != nil {
		t.Fatalf("ApplyBuildTemplateV2ToCharacterJSON: %v", err)
	}
	if !res.Applied {
		t.Fatalf("apply not applied: errors=%+v warnings=%+v", res.Preview.Errors, res.Preview.Warnings)
	}
	if !res.Preview.OK {
		t.Fatalf("apply preview not OK: %+v", res.Preview.Errors)
	}
	for _, issue := range append(append([]templates.ImportPreviewIssue{}, res.Preview.Errors...), res.Preview.Warnings...) {
		if issue.Code == templates.IssueCodeInventorySessionRequired {
			t.Fatalf("equipment/spells apply must not require a Sort Order session, got %+v", issue)
		}
	}

	// Counts: 6 weapon + 4 ammo + 4 armor + 3 active talismans = 17 equipment
	// writes; all 14 spell slots written.
	if res.EquipmentSlotsApplied != 17 {
		t.Errorf("EquipmentSlotsApplied = %d, want 17", res.EquipmentSlotsApplied)
	}
	if res.SpellSlotsApplied != 14 {
		t.Errorf("SpellSlotsApplied = %d, want 14", res.SpellSlotsApplied)
	}

	// Profile/stats transferred from the source.
	if target.Player.Level != 50 {
		t.Errorf("target Level = %d, want 50 (from source)", target.Player.Level)
	}
	if target.Player.Vigor != 20 {
		t.Errorf("target Vigor = %d, want 20 (from source)", target.Player.Vigor)
	}

	// Equipment transferred: representative weapon, talisman, and an explicit
	// clear of the target's stale RH2 weapon.
	rawEquip, err := target.ReadEquippedState()
	if err != nil {
		t.Fatalf("target ReadEquippedState: %v", err)
	}
	if got := rawEquip.Equipped[1]; got != items.weaponFullID { // RH1
		t.Errorf("RH1 = 0x%08X, want weaponFullID 0x%08X", got, items.weaponFullID)
	}
	if got := rawEquip.Equipped[17]; got != items.talismanID { // talisman1
		t.Errorf("talisman1 = 0x%08X, want 0x%08X", got, items.talismanID)
	}
	if got := rawEquip.Equipped[3]; got != roundTripUnarmedItemID { // RH2 cleared → Unarmed
		t.Errorf("RH2 = 0x%08X, want Unarmed 0x%08X (explicit clear of stale gear)", got, roundTripUnarmedItemID)
	}
	// Occupied ammo transferred: arrows1 (source-equipped) resolved against the
	// target's owned ammo — proving the pass-through ammo category applies, not
	// just clears.
	if got := rawEquip.Equipped[6]; got != items.ammoID { // arrows1
		t.Errorf("arrows1 = 0x%08X, want ammoID 0x%08X (occupied ammo equipped from target inventory)", got, items.ammoID)
	}
	// No ammo slot may report not-in-inventory — the target owns the ammo.
	for _, slotKey := range []string{"arrows1", "bolts1", "arrows2", "bolts2"} {
		for _, w := range res.Preview.Warnings {
			if w.Code == templates.IssueCodeEquipmentItemNotInInventory && w.Container == slotKey {
				t.Errorf("unexpected not-in-inventory warning for %s: %+v", slotKey, w)
			}
		}
	}

	// Spells transferred: slot 0 = source's Catch Flame; slot 3 explicit clear.
	if id, _ := readSpellSlotOnSlot(target, 0); id != 0x1770 {
		t.Errorf("spell slot 0 = 0x%08X, want raw 0x1770 (Catch Flame)", id)
	}
	if id, _ := readSpellSlotOnSlot(target, 3); id != core.EquippedSpellEmptySentinel {
		t.Errorf("spell slot 3 = 0x%08X, want empty sentinel (explicit clear)", id)
	}

	// The source slot was never mutated by the apply.
	for i := range srcBefore {
		if srcBefore[i] != app.save.Slots[0].Data[i] {
			t.Fatalf("apply mutated the SOURCE slot at byte %d", i)
		}
	}
}

// readSpellSlotOnSlot reads the (spellID, follower) pair from an arbitrary slot.
func readSpellSlotOnSlot(slot *core.SaveSlot, slotIdx int) (uint32, uint32) {
	off := slot.EquippedSpellsOffset + slotIdx*core.EquippedSpellSlotSize
	return binary.LittleEndian.Uint32(slot.Data[off:]),
		binary.LittleEndian.Uint32(slot.Data[off+4:])
}
