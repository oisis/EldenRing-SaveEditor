package application

import (
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/editor"
	"github.com/oisis/EldenRing-SaveForge/backend/templates"
	"github.com/oisis/EldenRing-SaveForge/backend/vm"
)

// seedSpell writes a raw MagicParam ID into the given 0-indexed spell slot of
// slot 0 (occupied follower), mirroring how a live save stores an equipped
// spell. Used to give the export path something to read.
func seedSpell(app *App, slotIdx int, rawID uint32) {
	slot := &app.save.Slots[0]
	off := slot.EquippedSpellsOffset + slotIdx*core.EquippedSpellSlotSize
	binary.LittleEndian.PutUint32(slot.Data[off:], rawID)
	binary.LittleEndian.PutUint32(slot.Data[off+4:], core.EquippedSpellOccupiedFollower)
}

// decodeExport unmarshals an export payload into a BuildTemplate.
func decodeExport(t *testing.T, payload string) templates.BuildTemplate {
	t.Helper()
	var tpl templates.BuildTemplate
	if err := json.Unmarshal([]byte(payload), &tpl); err != nil {
		t.Fatalf("export JSON does not decode: %v\n%s", err, payload)
	}
	return tpl
}

// ─── spells-only export ─────────────────────────────────────────────────

func TestExportV2_SpellsOnly_FullLoadoutWithClears(t *testing.T) {
	app := spellsApplyFixture(t)
	seedSpell(app, 0, 0x1770)       // Catch Flame
	seedSpell(app, 5, 0x40000+0x22) // arbitrary occupied slot
	// slots 1-4, 6-13 stay at the empty sentinel from the fixture.

	out, err := app.ExportBuildTemplateV2JSONFromCharacter(0, `{"spells":true}`, BuildTemplateV2ExportOptions{Name: "spells"})
	if err != nil {
		t.Fatalf("export spells: %v", err)
	}
	tpl := decodeExport(t, out)
	if tpl.Selection == nil || tpl.Selection.Spells == nil || !tpl.Selection.Spells.All {
		t.Fatalf("selection.spells.All not set: %+v", tpl.Selection)
	}
	if tpl.Sections.Spells == nil {
		t.Fatal("spells section missing")
	}
	// Occupied slot 0 → full DB-style ID (0x40000000 | raw).
	if tpl.Sections.Spells.Spell1 == nil || tpl.Sections.Spells.Spell1.BaseItemID != (templates.SpellItemIDPrefix|0x1770) {
		t.Errorf("spell1 wrong: %+v", tpl.Sections.Spells.Spell1)
	}
	// Empty slot 2 → explicit clear (BaseItemID 0), present not omitted.
	if tpl.Sections.Spells.Spell2 == nil || tpl.Sections.Spells.Spell2.BaseItemID != 0 {
		t.Errorf("spell2 (empty) should be explicit clear, got %+v", tpl.Sections.Spells.Spell2)
	}
	// Full loadout = all 14 slots present.
	if tpl.Sections.Spells.Spell14 == nil {
		t.Error("spell14 should be present in a full loadout")
	}
}

func TestExportV2_SpellsOnly_YAMLRoundTrip(t *testing.T) {
	app := spellsApplyFixture(t)
	seedSpell(app, 0, 0x1770)

	yamlOut, err := app.ExportBuildTemplateV2YAMLFromCharacter(0, `{"spells":true}`, BuildTemplateV2ExportOptions{Name: "spells"})
	if err != nil {
		t.Fatalf("export spells yaml: %v", err)
	}
	tpl, err := templates.ParseBuildTemplateYAML([]byte(yamlOut))
	if err != nil {
		t.Fatalf("parse yaml round-trip: %v", err)
	}
	report := templates.PreviewBuildTemplateImport(tpl, templates.ImportPreviewOptions{Mode: "append"})
	if !report.OK {
		t.Errorf("preview of round-tripped spells template not OK: %+v", report.Errors)
	}
	if tpl.Sections.Spells == nil || tpl.Sections.Spells.Spell1 == nil ||
		tpl.Sections.Spells.Spell1.BaseItemID != (templates.SpellItemIDPrefix|0x1770) {
		t.Errorf("spell1 lost across yaml round-trip: %+v", tpl.Sections.Spells)
	}
}

func TestExportV2_SpellsOnly_DoesNotMutateSlot(t *testing.T) {
	app := spellsApplyFixture(t)
	seedSpell(app, 0, 0x1770)
	before := make([]byte, len(app.save.Slots[0].Data))
	copy(before, app.save.Slots[0].Data)

	if _, err := app.ExportBuildTemplateV2JSONFromCharacter(0, `{"spells":true}`, BuildTemplateV2ExportOptions{}); err != nil {
		t.Fatalf("export: %v", err)
	}
	for i := range before {
		if before[i] != app.save.Slots[0].Data[i] {
			t.Fatalf("export mutated source slot at byte %d", i)
		}
	}
}

// ─── equipment-only + equipment+spells export ───────────────────────────

// seedFixtureEquipped writes the 22-slot equipped-armaments block of the
// calibrated fixture (the real Equipment source), past a NON-ZERO projectile
// count, and fills the ChrAsm2 header at EquipItemsIDOffset with decoy values
// the export must ignore. TalismanSlots=3 (four active pouch slots) so a full
// loadout exports talisman1..4.
func seedFixtureEquipped(t *testing.T, app *App, equipped [core.ChrAsmFieldCount]uint32) {
	t.Helper()
	slot := &app.save.Slots[0]
	slot.Player.TalismanSlots = 3
	for i := 0; i < core.ChrAsmFieldCount; i++ {
		binary.LittleEndian.PutUint32(slot.Data[slot.EquipItemsIDOffset+i*4:], 0x8001FF00|uint32(i))
	}
	const projCount = 3
	projHeaderOff := slot.EquippedSpellsOffset + core.DynEquipedSpells + core.DynEquipedItems + core.DynEquipedGestures
	binary.LittleEndian.PutUint32(slot.Data[projHeaderOff:], projCount)
	armamentsOff := projHeaderOff + 4 + projCount*8
	for i := 0; i < core.ChrAsmFieldCount; i++ {
		binary.LittleEndian.PutUint32(slot.Data[armamentsOff+i*4:], equipped[i])
	}
}

func TestExportV2_EquipmentOnly_FullLoadoutAllClears(t *testing.T) {
	app := spellsApplyFixture(t)
	// An unequipped character reads back its class empties in the
	// equipped-armaments block; a full loadout exports those as clears.
	seedFixtureEquipped(t, app, allEmptyEquipped())
	out, err := app.ExportBuildTemplateV2JSONFromCharacter(0, `{"equipment":true}`, BuildTemplateV2ExportOptions{Name: "equip"})
	if err != nil {
		t.Fatalf("export equipment: %v", err)
	}
	tpl := decodeExport(t, out)
	if tpl.Selection == nil || tpl.Selection.Equipment == nil || !tpl.Selection.Equipment.All {
		t.Fatalf("selection.equipment.All not set: %+v", tpl.Selection)
	}
	if tpl.Sections.Equipment == nil {
		t.Fatal("equipment section missing")
	}
	if tpl.Sections.Equipment.WeaponRightHand1 == nil || tpl.Sections.Equipment.WeaponRightHand1.BaseItemID != 0 {
		t.Errorf("weaponRightHand1 should be explicit clear, got %+v", tpl.Sections.Equipment.WeaponRightHand1)
	}
	if tpl.Sections.Equipment.ArmorLegs == nil || tpl.Sections.Equipment.ArmorLegs.BaseItemID != 0 {
		t.Errorf("armorLegs should be explicit clear, got %+v", tpl.Sections.Equipment.ArmorLegs)
	}
	if tpl.Sections.Equipment.Talisman4 == nil || tpl.Sections.Equipment.Talisman4.BaseItemID != 0 {
		t.Errorf("talisman4 should be explicit clear, got %+v", tpl.Sections.Equipment.Talisman4)
	}
	if tpl.Sections.Equipment.Talisman5 != nil {
		t.Errorf("talisman5 must not be exported, got %+v", tpl.Sections.Equipment.Talisman5)
	}
}

// The export reads the equipped-armaments block and ignores the ChrAsm2
// GaItem-handle header at EquipItemsIDOffset (populated with decoy values).
func TestExportV2_Equipment_ReadsArmamentsIgnoresDecoyHeader(t *testing.T) {
	app := spellsApplyFixture(t)
	eq := allEmptyEquipped()
	eq[1] = 0x80100020 // real weapon in the armaments block, RH1
	seedFixtureEquipped(t, app, eq)

	out, err := app.ExportBuildTemplateV2JSONFromCharacter(0, `{"equipment":true}`, BuildTemplateV2ExportOptions{Name: "decoy"})
	if err != nil {
		t.Fatalf("export equipment: %v", err)
	}
	tpl := decodeExport(t, out)
	if tpl.Sections.Equipment.WeaponRightHand1 == nil {
		t.Fatal("weaponRightHand1 must be populated from the armaments block")
	}
	// Decoy header value for idx 1 was 0x8001FF01 → normalized 0x0001FF01.
	// The export must not carry that; it must reflect the armaments weapon.
	if got := tpl.Sections.Equipment.WeaponRightHand1.BaseItemID; got == 0x0001FF01 {
		t.Errorf("export read the decoy ChrAsm2 header (0x%08X) instead of the armaments block", got)
	}
	if got := tpl.Sections.Equipment.WeaponRightHand1.BaseItemID; got != 0x00100020 {
		t.Errorf("weaponRightHand1 baseItemID = 0x%08X, want normalized armaments value 0x00100020", got)
	}
}

// Equipment is mutually exclusive with items / layout — the backend rejects
// the combination even on a direct export call, not only in the UI.
func TestExportV2_EquipmentWithItems_Rejected(t *testing.T) {
	app := spellsApplyFixture(t)
	seedFixtureEquipped(t, app, allEmptyEquipped())
	_, err := app.ExportBuildTemplateV2JSONFromCharacter(0, `{"equipment":true,"items":true}`, BuildTemplateV2ExportOptions{Name: "bad"})
	if err == nil {
		t.Fatal("expected rejection when equipment is combined with items")
	}
	if !strings.Contains(err.Error(), "equipment cannot be combined") {
		t.Errorf("error should explain the equipment/items exclusion, got %q", err.Error())
	}
}

func TestExportV2_EquipmentAndSpells_BothSectionsLand(t *testing.T) {
	app := spellsApplyFixture(t)
	seedFixtureEquipped(t, app, allEmptyEquipped())
	seedSpell(app, 0, 0x1770)

	out, err := app.ExportBuildTemplateV2JSONFromCharacter(0, `{"equipment":true,"spells":true}`, BuildTemplateV2ExportOptions{Name: "both"})
	if err != nil {
		t.Fatalf("export equipment+spells: %v", err)
	}
	tpl := decodeExport(t, out)
	if tpl.Sections.Equipment == nil {
		t.Error("equipment section missing in combined export")
	}
	if tpl.Sections.Spells == nil || tpl.Sections.Spells.Spell1 == nil ||
		tpl.Sections.Spells.Spell1.BaseItemID != (templates.SpellItemIDPrefix|0x1770) {
		t.Errorf("spells section wrong in combined export: %+v", tpl.Sections.Spells)
	}
	if tpl.Selection.Equipment == nil || !tpl.Selection.Equipment.All ||
		tpl.Selection.Spells == nil || !tpl.Selection.Spells.All {
		t.Errorf("combined selection wrong: %+v", tpl.Selection)
	}
}

func TestExportV2_EquipmentAndSpells_JSONYAMLEquivalent(t *testing.T) {
	app := spellsApplyFixture(t)
	seedFixtureEquipped(t, app, allEmptyEquipped())
	seedSpell(app, 0, 0x1770)
	sel := `{"equipment":true,"spells":true}`
	opts := BuildTemplateV2ExportOptions{Name: "both"}

	jsonOut, err := app.ExportBuildTemplateV2JSONFromCharacter(0, sel, opts)
	if err != nil {
		t.Fatalf("json export: %v", err)
	}
	yamlOut, err := app.ExportBuildTemplateV2YAMLFromCharacter(0, sel, opts)
	if err != nil {
		t.Fatalf("yaml export: %v", err)
	}
	fromJSON := decodeExport(t, jsonOut)
	fromYAML, err := templates.ParseBuildTemplateYAML([]byte(yamlOut))
	if err != nil {
		t.Fatalf("parse yaml: %v", err)
	}
	if fromJSON.Sections.Spells.Spell1.BaseItemID != fromYAML.Sections.Spells.Spell1.BaseItemID {
		t.Error("spell1 diverges between JSON and YAML export")
	}
	if (fromJSON.Sections.Equipment.WeaponRightHand1 == nil) != (fromYAML.Sections.Equipment.WeaponRightHand1 == nil) {
		t.Error("equipment weaponRightHand1 presence diverges between JSON and YAML export")
	}
}

// ─── real production-path fixture ───────────────────────────────────────

// loadoutFixtureItems carries the DB IDs and inventory GaItem handles the
// loadout fixture equips, so the tests can assert against real, DB-resolvable
// values rather than re-encoding a hand-built assumption.
type loadoutFixtureItems struct {
	weaponBaseID uint32 // Longsword base
	weaponFullID uint32 // Longsword +7 Cold (base + Cold offset 900 + level 7)
	armorID      uint32 // Iron Kasa
	ammoID       uint32 // Arrow
	talismanID   uint32 // Crimson Amber Medallion (item ID form)

	weaponHandle   uint32
	armorHandle    uint32
	ammoHandle     uint32
	talismanHandle uint32
}

// newLoadoutFixture builds an App whose slot 0 carries a real, resolvable
// inventory — both as the parsed slot.Inventory.CommonItems (consumed by
// SaveSlot.WriteEquipment) AND as raw inventory bytes at
// MagicOffset+InvStartFromMagic (consumed by editor.BuildSnapshot on the
// export path) — plus one equipped spell. It then equips the four item
// families through the real SaveEquipment path, so the equipped-armaments
// block is written exactly the way the app writes it live.
//
// Arrows2 is deliberately equipped as well, so a caller can later clear it
// through SaveEquipment and prove the clear-writer path, distinct from a slot
// that was empty to begin with.
func newLoadoutFixture(t *testing.T) (*App, loadoutFixtureItems) {
	t.Helper()
	app := spellsApplyFixture(t)
	slot := &app.save.Slots[0]
	slot.Player.TalismanSlots = 2 // → 3 active pouch slots

	items := loadoutFixtureItems{
		weaponBaseID:   0x001E8480,       // Longsword (melee_armaments, MaxUpgrade 25)
		weaponFullID:   0x001E8480 + 907, // +7 Cold (Cold infusion offset 900 + level 7)
		armorID:        0x100249F0,       // Iron Kasa
		ammoID:         0x02FAF080,       // Arrow
		talismanID:     0x200003E8,       // Crimson Amber Medallion
		weaponHandle:   core.ItemTypeWeapon | 0x000000A1,
		armorHandle:    core.ItemTypeArmor | 0x000000A2,
		ammoHandle:     core.ItemTypeWeapon | 0x000000A3,
		talismanHandle: core.ItemTypeAccessory | 0x000003E8,
	}

	// GaMap resolves the instance-backed families (weapon / armor / ammo) to
	// their DB item IDs. Talismans are handle-encoded (db.HandleToItemID), so
	// they need no GaMap entry.
	slot.GaMap = map[uint32]uint32{
		items.weaponHandle: items.weaponFullID,
		items.armorHandle:  items.armorID,
		items.ammoHandle:   items.ammoID,
	}

	inv := []core.InventoryItem{
		{GaItemHandle: items.weaponHandle, Quantity: 1},
		{GaItemHandle: items.armorHandle, Quantity: 1},
		{GaItemHandle: items.ammoHandle, Quantity: 1},
		{GaItemHandle: items.talismanHandle, Quantity: 1},
	}
	slot.Inventory.CommonItems = inv
	// Storage carries one record so the snapshot-independence test has a
	// non-empty container to prove the deep copy against.
	slot.Storage.CommonItems = []core.InventoryItem{{GaItemHandle: items.talismanHandle, Quantity: 5}}

	// Mirror the parsed inventory into the raw bytes editor.BuildSnapshot reads.
	invStart := slot.MagicOffset + core.InvStartFromMagic
	for i, it := range inv {
		off := invStart + i*core.InvRecordLen
		binary.LittleEndian.PutUint32(slot.Data[off:], it.GaItemHandle)
		binary.LittleEndian.PutUint32(slot.Data[off+4:], 1)              // quantity
		binary.LittleEndian.PutUint32(slot.Data[off+8:], uint32(1000+i)) // acquisition index
	}

	seedSpell(app, 0, 0x1770) // Catch Flame

	if err := app.SaveEquipment(0, []EquipmentChange{
		{Slot: core.EquipSlotRightHandArmament1, Handle: items.weaponHandle},
		{Slot: core.EquipSlotHead, Handle: items.armorHandle},
		{Slot: core.EquipSlotArrows1, Handle: items.ammoHandle},
		{Slot: core.EquipSlotArrows2, Handle: items.ammoHandle},
		{Slot: core.EquipSlotTalisman1, Handle: items.talismanHandle},
	}); err != nil {
		t.Fatalf("SaveEquipment: %v", err)
	}
	return app, items
}

// TestCloneCharacterSlot_SnapshotIndependence proves cloneCharacterSlot
// detaches every mutable slot field from the live slot, and that the section
// builders driven off the clone still reflect the snapshot moment after the
// live slot is mutated.
func TestCloneCharacterSlot_SnapshotIndependence(t *testing.T) {
	app, items := newLoadoutFixture(t)
	live := &app.save.Slots[0]

	clone, err := app.cloneCharacterSlot(0)
	if err != nil {
		t.Fatalf("cloneCharacterSlot: %v", err)
	}

	// ── independence of the raw state ──────────────────────────────────
	if &clone.Data[0] == &live.Data[0] {
		t.Fatal("clone.Data shares the live slot's backing array")
	}
	invStart := live.MagicOffset + core.InvStartFromMagic
	cloneByteBefore := clone.Data[invStart]
	if clone.GaMap[0xDEADBEEF] != 0 {
		t.Fatal("clone.GaMap unexpectedly already carries the probe key")
	}
	clonePlayerBefore := clone.Player.TalismanSlots
	cloneInvQtyBefore := clone.Inventory.CommonItems[0].Quantity
	cloneStoQtyBefore := clone.Storage.CommonItems[0].Quantity

	// Build every export section from the captured clone BEFORE mutating the
	// live slot, so we have the snapshot-moment values to compare against.
	profileBefore, statsBefore := loadoutSourcesFromSlot(t, clone)
	itemsBefore, err := buildItemsSourceFromSlot(clone, 0)
	if err != nil {
		t.Fatalf("items source (before): %v", err)
	}
	equipBefore, spellsBefore, err := buildEquipmentSpellsSourcesFromSlot(clone, 0, true, true)
	if err != nil {
		t.Fatalf("equipment/spells source (before): %v", err)
	}

	// ── mutate the live slot in every dimension ────────────────────────
	live.Data[invStart] ^= 0xFF
	live.GaMap[0xDEADBEEF] = 0x12345678
	live.Inventory.CommonItems[0].Quantity = 999
	live.Storage.CommonItems[0].Quantity = 999
	live.Player.TalismanSlots = 0
	live.Player.Level = 1
	// Re-equip the weapon slot on the live slot with a different (owned) item so
	// the live equipped-armaments block changes; the clone must not follow.
	if err := app.SaveEquipment(0, []EquipmentChange{
		{Slot: core.EquipSlotRightHandArmament1, Handle: items.ammoHandle},
	}); err != nil {
		t.Fatalf("mutate live equipment: %v", err)
	}
	seedSpell(app, 0, 0x0FA0) // overwrite live spell slot 0 (Glintstone Pebble)

	// ── the clone is untouched ─────────────────────────────────────────
	if clone.Data[invStart] != cloneByteBefore {
		t.Errorf("clone.Data mutated with the live slot (byte 0x%X)", invStart)
	}
	if _, ok := clone.GaMap[0xDEADBEEF]; ok {
		t.Error("clone.GaMap is not an independent map")
	}
	if clone.Inventory.CommonItems[0].Quantity != cloneInvQtyBefore {
		t.Error("clone.Inventory is not a deep copy")
	}
	if clone.Storage.CommonItems[0].Quantity != cloneStoQtyBefore {
		t.Error("clone.Storage is not a deep copy")
	}
	if clone.Player.TalismanSlots != clonePlayerBefore {
		t.Error("clone.Player did not come from the snapshot moment")
	}

	// ── section builders off the clone still reflect the snapshot ───────
	profileAfter, statsAfter := loadoutSourcesFromSlot(t, clone)
	if *profileAfter.Level != *profileBefore.Level || *profileAfter.TalismanSlots != *profileBefore.TalismanSlots {
		t.Errorf("profile diverged after live mutation: before=%+v after=%+v", profileBefore, profileAfter)
	}
	if *statsAfter.Vigor != *statsBefore.Vigor {
		t.Errorf("stats diverged after live mutation")
	}

	itemsAfter, err := buildItemsSourceFromSlot(clone, 0)
	if err != nil {
		t.Fatalf("items source (after): %v", err)
	}
	wBefore := findInvWeapon(t, itemsBefore.InventoryItems, items.weaponBaseID)
	wAfter := findInvWeapon(t, itemsAfter.InventoryItems, items.weaponBaseID)
	if wBefore.CurrentUpgrade != 7 || wBefore.InfusionName != "Cold" {
		t.Fatalf("fixture weapon did not resolve with metadata: %+v", wBefore)
	}
	if wAfter.CurrentUpgrade != wBefore.CurrentUpgrade || wAfter.InfusionName != wBefore.InfusionName {
		t.Errorf("items source diverged after live mutation: before=%+v after=%+v", wBefore, wAfter)
	}

	equipAfter, spellsAfter, err := buildEquipmentSpellsSourcesFromSlot(clone, 0, true, true)
	if err != nil {
		t.Fatalf("equipment/spells source (after): %v", err)
	}
	if equipBefore.WeaponRightHand1 == nil || equipAfter.WeaponRightHand1 == nil ||
		equipAfter.WeaponRightHand1.BaseItemID != equipBefore.WeaponRightHand1.BaseItemID {
		t.Errorf("equipment source diverged after live mutation: before=%+v after=%+v",
			equipBefore.WeaponRightHand1, equipAfter.WeaponRightHand1)
	}
	// The source builder carries the RAW MagicParam IDs (the 0x40 prefix is
	// applied later by BuildV2Template), so the seeded 0x1770 stays raw here.
	if spellsBefore[0] != 0x1770 || spellsAfter[0] != spellsBefore[0] {
		t.Errorf("spells source diverged after live mutation: before=0x%X after=0x%X", spellsBefore[0], spellsAfter[0])
	}
}

// loadoutSourcesFromSlot maps a detached slot through the real VM + source
// builders the export path uses for profile / stats.
func loadoutSourcesFromSlot(t *testing.T, slot *core.SaveSlot) (*templates.ProfileSource, *templates.StatsSource) {
	t.Helper()
	charVM, err := vm.MapParsedSlotToVM(slot)
	if err != nil {
		t.Fatalf("MapParsedSlotToVM: %v", err)
	}
	sel := &templates.TemplateSelection{
		Profile: &templates.SectionSelection{All: true},
		Stats:   &templates.SectionSelection{All: true},
	}
	return buildTemplateV2SourcesFromCharacter(charVM, sel)
}

func findInvWeapon(t *testing.T, itemsList []editor.EditableItem, baseID uint32) editor.EditableItem {
	t.Helper()
	for _, it := range itemsList {
		if it.BaseItemID == baseID {
			return it
		}
	}
	t.Fatalf("weapon baseItemID 0x%08X not found in inventory source (%d items)", baseID, len(itemsList))
	return editor.EditableItem{}
}

// End-to-end: SaveEquipment writes the native equipped-armaments block, then
// the export reads that same block back through the real production chain
// (cloneCharacterSlot → ReadEquippedState → editor.BuildSnapshot →
// BuildV2Template → JSON) for a weapon, armor, ammo, a talisman, and a slot
// cleared by the writer. No hand-built []editor.EditableItem is fed to the
// builder; the metadata comes from the real BuildSnapshot resolution.
func TestExportV2_SaveEquipmentRoundTripsThroughExport(t *testing.T) {
	app, items := newLoadoutFixture(t)

	// A genuine clear of a slot that WAS occupied — proves the clear-writer
	// path, not merely an initially-empty slot.
	if err := app.SaveEquipment(0, []EquipmentChange{
		{Slot: core.EquipSlotArrows2, Handle: 0},
	}); err != nil {
		t.Fatalf("SaveEquipment clear: %v", err)
	}

	out, err := app.ExportBuildTemplateV2JSONFromCharacter(0, `{"equipment":true}`, BuildTemplateV2ExportOptions{Name: "loadout"})
	if err != nil {
		t.Fatalf("ExportBuildTemplateV2JSONFromCharacter: %v", err)
	}
	tpl := decodeExport(t, out)
	sec := tpl.Sections.Equipment
	if sec == nil {
		t.Fatal("equipment section missing from export")
	}

	// Weapon carries the metadata resolved by the real BuildSnapshot.
	if sec.WeaponRightHand1 == nil || sec.WeaponRightHand1.BaseItemID != items.weaponBaseID ||
		sec.WeaponRightHand1.Upgrade == nil || *sec.WeaponRightHand1.Upgrade != 7 ||
		sec.WeaponRightHand1.InfusionName != "Cold" {
		t.Errorf("weapon round-trip lost metadata: %+v", sec.WeaponRightHand1)
	}
	if sec.ArmorHead == nil || sec.ArmorHead.BaseItemID != items.armorID {
		t.Errorf("armor round-trip wrong: %+v", sec.ArmorHead)
	}
	if sec.Arrows1 == nil || sec.Arrows1.BaseItemID != items.ammoID {
		t.Errorf("ammo round-trip wrong: %+v", sec.Arrows1)
	}
	if sec.Talisman1 == nil || sec.Talisman1.BaseItemID != items.talismanID {
		t.Errorf("talisman round-trip wrong: %+v", sec.Talisman1)
	}
	// The writer-cleared Arrows2 exports as an explicit clear (BaseItemID 0).
	if sec.Arrows2 == nil || sec.Arrows2.BaseItemID != 0 {
		t.Errorf("writer-cleared arrows2 should be explicit clear, got %+v", sec.Arrows2)
	}
}

// ─── fail-closed wiring when the source slot is not parseable ────────────

func TestExportV2_SpellsSelected_UnparsedSlot_FailsClosed(t *testing.T) {
	app := profileStatsFixture() // no EquippedSpellsOffset / Data calibration
	_, err := app.ExportBuildTemplateV2JSONFromCharacter(0, `{"spells":true}`, BuildTemplateV2ExportOptions{})
	if err == nil {
		t.Fatal("expected fail-closed error when the spell section is unparsed")
	}
	if !strings.Contains(err.Error(), "spell") {
		t.Errorf("error should mention spells, got %q", err.Error())
	}
}

func TestExportV2_EquipmentSelected_UnparsedSlot_FailsClosed(t *testing.T) {
	app := profileStatsFixture()
	_, err := app.ExportBuildTemplateV2JSONFromCharacter(0, `{"equipment":true}`, BuildTemplateV2ExportOptions{})
	if err == nil {
		t.Fatal("expected fail-closed error when the equipment section is unparsed")
	}
}
