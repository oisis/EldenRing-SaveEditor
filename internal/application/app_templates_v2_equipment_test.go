package application

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/editor"
)

// allEmptyEquipped returns a 22-slot equipped-armaments array whose every
// writable slot holds its class's real in-game-empty value, the way
// ReadEquippedState reads back an unequipped character: Unarmed for hand
// slots, the invalid sentinel for ammo / talismans, and the technical
// bare-armor IDs for armor. isEmptyEquipSlot is the single source of truth for
// recognising these — the export path never invents a second contract.
func allEmptyEquipped() [core.ChrAsmFieldCount]uint32 {
	var eq [core.ChrAsmFieldCount]uint32
	for _, i := range []int{0, 1, 2, 3, 4, 5} { // hand armaments → Unarmed
		eq[i] = core.ItemTypeWeapon | unarmedItemID
	}
	for _, i := range []int{6, 7, 8, 9} { // arrows / bolts → invalid sentinel
		eq[i] = core.GaHandleInvalid
	}
	eq[12] = 0x10002710                       // bare head
	eq[13] = 0x10002774                       // bare chest
	eq[14] = 0x100027D8                       // bare arms
	eq[15] = 0x1000283C                       // bare legs
	for _, i := range []int{17, 18, 19, 20} { // talismans → invalid sentinel
		eq[i] = core.GaHandleInvalid
	}
	return eq
}

const allTalismansActive = 4

func TestBuildEquipmentSection_EmptyReturnsNil(t *testing.T) {
	got := buildEquipmentSectionFromEquipped(allEmptyEquipped(), nil, allTalismansActive, false)
	if got != nil {
		t.Errorf("empty equipment should return nil section, got %+v", got)
	}
}

func TestBuildEquipmentSection_WeaponMatchesEditableItem(t *testing.T) {
	eq := allEmptyEquipped()
	// Equip a weapon in RH1 (idx 1): encoded form = itemID | 0x80000000.
	eq[1] = 0x80100020

	items := []editor.EditableItem{{
		BaseItemID:     0x100000,
		ItemID:         0x00100020,
		Name:           "Uchigatana",
		IsWeapon:       true,
		CurrentUpgrade: 25,
		InfusionName:   "Cold",
	}}
	sec := buildEquipmentSectionFromEquipped(eq, items, allTalismansActive, false)
	if sec == nil || sec.WeaponRightHand1 == nil {
		t.Fatalf("expected WeaponRightHand1 populated, got %+v", sec)
	}
	if sec.WeaponRightHand1.BaseItemID != 0x100000 {
		t.Errorf("baseItemID mismatch: %v", sec.WeaponRightHand1)
	}
	if sec.WeaponRightHand1.Upgrade == nil || *sec.WeaponRightHand1.Upgrade != 25 {
		t.Errorf("upgrade not populated for weapon")
	}
	if sec.WeaponRightHand1.InfusionName != "Cold" {
		t.Errorf("infusion lost: %q", sec.WeaponRightHand1.InfusionName)
	}
}

func TestBuildEquipmentSection_AmmoMatchesGoodsItem(t *testing.T) {
	eq := allEmptyEquipped()
	// Equip Arrows1 (idx 6): goods item ID 0x40100050 (already 0x40-prefixed).
	eq[6] = 0x40100050
	items := []editor.EditableItem{{
		BaseItemID: 0x40100050,
		ItemID:     0x40100050,
		Name:       "Standard Arrow",
	}}
	sec := buildEquipmentSectionFromEquipped(eq, items, allTalismansActive, false)
	if sec == nil || sec.Arrows1 == nil {
		t.Fatalf("expected Arrows1 populated, got %+v", sec)
	}
	if sec.Arrows1.BaseItemID != 0x40100050 {
		t.Errorf("arrows baseItemID mismatch: %v", sec.Arrows1)
	}
	if sec.Arrows1.Upgrade != nil {
		t.Errorf("ammo should not carry an upgrade pointer")
	}
}

func TestBuildEquipmentSection_ArmorMatchesEditableItem(t *testing.T) {
	eq := allEmptyEquipped()
	// Armor head idx 12: encoded = itemID | 0x80000000.
	eq[12] = 0x90100040
	items := []editor.EditableItem{{
		BaseItemID: 0x10100040,
		ItemID:     0x10100040,
		Name:       "Knight Helm",
		IsArmor:    true,
	}}
	sec := buildEquipmentSectionFromEquipped(eq, items, allTalismansActive, false)
	if sec == nil || sec.ArmorHead == nil {
		t.Fatalf("expected ArmorHead populated, got %+v", sec)
	}
	if sec.ArmorHead.BaseItemID != 0x10100040 {
		t.Errorf("armor baseItemID mismatch")
	}
}

func TestBuildEquipmentSection_GreatRuneAndUnknownSlotsNotExported(t *testing.T) {
	eq := allEmptyEquipped()
	eq[10] = 0x80000001 // EquippedGreatRune — out of scope
	eq[11] = 0x80000004 // unk0x2C — out of scope
	eq[16] = 0x80000005 // unk0x40 — out of scope

	sec := buildEquipmentSectionFromEquipped(eq, nil, allTalismansActive, false)
	if sec != nil {
		t.Errorf("section should be nil — only out-of-scope slots populated, got %+v", sec)
	}
}

func TestBuildEquipmentSection_UnknownItemEmitsNormalizedBaseID(t *testing.T) {
	eq := allEmptyEquipped()
	// Equip RH1 with an item ID that won't resolve to anything in the
	// editable inventory or the DB; the scanner should still emit a ref
	// rather than silently drop the slot.
	eq[1] = 0xDEADBEEF
	sec := buildEquipmentSectionFromEquipped(eq, nil, allTalismansActive, false)
	if sec == nil || sec.WeaponRightHand1 == nil {
		t.Fatalf("unknown equipped item should still emit a ref, got %+v", sec)
	}
	if sec.WeaponRightHand1.BaseItemID == 0 {
		t.Errorf("unknown item ref should carry the normalized itemID as baseItemID")
	}
}

func TestBuildEquipmentSection_MultiSlotPopulated(t *testing.T) {
	eq := allEmptyEquipped()
	eq[1] = 0x80100020  // RH1 weapon
	eq[6] = 0x40100050  // Arrows1
	eq[12] = 0x90100040 // Head armor
	items := []editor.EditableItem{
		{BaseItemID: 0x100000, ItemID: 0x00100020, Name: "Uchi", IsWeapon: true, CurrentUpgrade: 0},
		{BaseItemID: 0x40100050, ItemID: 0x40100050, Name: "Arrow"},
		{BaseItemID: 0x10100040, ItemID: 0x10100040, Name: "Helm", IsArmor: true},
	}
	sec := buildEquipmentSectionFromEquipped(eq, items, allTalismansActive, false)
	if sec == nil {
		t.Fatal("section nil")
	}
	if sec.WeaponRightHand1 == nil || sec.Arrows1 == nil || sec.ArmorHead == nil {
		t.Errorf("expected three slots populated, got %+v", sec)
	}
	if sec.ArmorChest != nil {
		t.Errorf("ArmorChest should remain nil for empty slot")
	}
}

// ─── talisman export tests ──────────────────────────────────────────────

func TestBuildEquipmentSection_TalismanMatchesEditableItem(t *testing.T) {
	eq := allEmptyEquipped()
	// Talisman1 (idx 17): stored bare (0x20-prefixed), no 0x80 mask.
	eq[17] = 0x20100001
	items := []editor.EditableItem{{
		BaseItemID: 0x20100001,
		ItemID:     0x20100001,
		Name:       "Radagon's Soreseal",
		IsTalisman: true,
	}}
	sec := buildEquipmentSectionFromEquipped(eq, items, allTalismansActive, false)
	if sec == nil || sec.Talisman1 == nil {
		t.Fatalf("expected Talisman1 populated, got %+v", sec)
	}
	if sec.Talisman1.BaseItemID != 0x20100001 {
		t.Errorf("talisman1 baseItemID mismatch: %v", sec.Talisman1)
	}
	if sec.Talisman1.Name != "Radagon's Soreseal" {
		t.Errorf("talisman1 name mismatch: %q", sec.Talisman1.Name)
	}
	if sec.Talisman1.Upgrade != nil || sec.Talisman1.InfusionName != "" || sec.Talisman1.AoWItemID != nil {
		t.Errorf("talisman ref should carry no weapon metadata: %+v", sec.Talisman1)
	}
}

// Source active-slot semantics: a source with one active pouch slot exports
// only talisman1; occupied-but-locked talisman slots stay nil (never clears).
func TestBuildEquipmentSection_SourceOneActiveSlot_ExportsOnlyTalisman1(t *testing.T) {
	eq := allEmptyEquipped()
	eq[17] = 0x20100001
	eq[18] = 0x20100002 // occupied but locked on a 1-slot source
	items := []editor.EditableItem{
		{BaseItemID: 0x20100001, ItemID: 0x20100001, Name: "T1", IsTalisman: true},
		{BaseItemID: 0x20100002, ItemID: 0x20100002, Name: "T2", IsTalisman: true},
	}
	sec := buildEquipmentSectionFromEquipped(eq, items, 1, true)
	if sec == nil || sec.Talisman1 == nil {
		t.Fatalf("talisman1 should be exported, got %+v", sec)
	}
	if sec.Talisman2 != nil || sec.Talisman3 != nil || sec.Talisman4 != nil {
		t.Errorf("talisman slots beyond the source capacity must stay nil, got %+v", sec)
	}
}

func TestBuildEquipmentSection_SourceFourActiveSlots_ExportsTalisman1Through4(t *testing.T) {
	eq := allEmptyEquipped()
	eq[17] = 0x20100001
	eq[18] = 0x20100002
	eq[19] = 0x20100003
	eq[20] = 0x20100004
	eq[21] = 0x20100005 // talisman5 — never exported
	items := []editor.EditableItem{
		{BaseItemID: 0x20100001, ItemID: 0x20100001, Name: "T1", IsTalisman: true},
		{BaseItemID: 0x20100002, ItemID: 0x20100002, Name: "T2", IsTalisman: true},
		{BaseItemID: 0x20100003, ItemID: 0x20100003, Name: "T3", IsTalisman: true},
		{BaseItemID: 0x20100004, ItemID: 0x20100004, Name: "T4", IsTalisman: true},
		{BaseItemID: 0x20100005, ItemID: 0x20100005, Name: "T5", IsTalisman: true},
	}
	sec := buildEquipmentSectionFromEquipped(eq, items, allTalismansActive, false)
	if sec == nil || sec.Talisman1 == nil || sec.Talisman2 == nil || sec.Talisman3 == nil || sec.Talisman4 == nil {
		t.Fatalf("expected talisman1..4 populated, got %+v", sec)
	}
	if sec.Talisman5 != nil {
		t.Errorf("talisman5 must never be exported, got %+v", sec.Talisman5)
	}
}

func TestBuildEquipmentSection_TalismanUnknownItemEmitsRawBaseID(t *testing.T) {
	eq := allEmptyEquipped()
	eq[17] = 0x2DEADBEE // unfamiliar talisman ID — not in inventory nor DB
	sec := buildEquipmentSectionFromEquipped(eq, nil, allTalismansActive, false)
	if sec == nil || sec.Talisman1 == nil {
		t.Fatalf("unknown talisman should still emit a ref, got %+v", sec)
	}
	if sec.Talisman1.BaseItemID == 0 {
		t.Errorf("unknown talisman ref should carry the normalized itemID")
	}
}

// ─── full-loadout export (emitEmptyAsClear=true) ────────────────────────

func TestBuildEquipmentSection_FullLoadout_NativeEmptyBecomesClears(t *testing.T) {
	sec := buildEquipmentSectionFromEquipped(allEmptyEquipped(), nil, allTalismansActive, true)
	if sec == nil {
		t.Fatal("full-loadout export of an empty character must return a non-nil (all-clear) section")
	}
	if sec.WeaponRightHand1 == nil || sec.WeaponRightHand1.BaseItemID != 0 {
		t.Errorf("weaponRightHand1 (Unarmed) should be explicit clear, got %+v", sec.WeaponRightHand1)
	}
	if sec.ArmorChest == nil || sec.ArmorChest.BaseItemID != 0 {
		t.Errorf("armorChest (bare) should be explicit clear, got %+v", sec.ArmorChest)
	}
	if sec.Arrows1 == nil || sec.Arrows1.BaseItemID != 0 {
		t.Errorf("arrows1 (sentinel) should be explicit clear, got %+v", sec.Arrows1)
	}
	if sec.Talisman1 == nil || sec.Talisman1.BaseItemID != 0 {
		t.Errorf("talisman1 (sentinel) should be explicit clear, got %+v", sec.Talisman1)
	}
	if sec.Talisman5 != nil {
		t.Errorf("talisman5 must not be exported, got %+v", sec.Talisman5)
	}
}

func TestBuildEquipmentSection_FullLoadout_OccupiedAndEmptyMix(t *testing.T) {
	eq := allEmptyEquipped()
	eq[1] = 0x80100020 // RH1 weapon occupied
	items := []editor.EditableItem{{
		BaseItemID: 0x100000, ItemID: 0x00100020, Name: "Uchi", IsWeapon: true, CurrentUpgrade: 3,
	}}
	sec := buildEquipmentSectionFromEquipped(eq, items, allTalismansActive, true)
	if sec == nil || sec.WeaponRightHand1 == nil {
		t.Fatal("occupied slot must be populated")
	}
	if sec.WeaponRightHand1.BaseItemID != 0x100000 {
		t.Errorf("occupied slot baseItemID wrong: %+v", sec.WeaponRightHand1)
	}
	// A different, empty slot must be an explicit clear, not omitted.
	if sec.WeaponLeftHand1 == nil || sec.WeaponLeftHand1.BaseItemID != 0 {
		t.Errorf("empty weaponLeftHand1 should be explicit clear, got %+v", sec.WeaponLeftHand1)
	}
}

func TestEquipmentSlotEquipClass(t *testing.T) {
	cases := map[string]equipClass{
		"weaponRightHand1": classHandArmament,
		"weaponLeftHand3":  classHandArmament,
		"arrows1":          classAmmo,
		"bolts2":           classAmmo,
		"armorHead":        classArmor,
		"armorLegs":        classArmor,
		"talisman1":        classTalisman,
		"talisman5":        classTalisman,
	}
	for key, want := range cases {
		if got := equipmentSlotEquipClass(key); got != want {
			t.Errorf("equipmentSlotEquipClass(%q) = %d, want %d", key, got, want)
		}
	}
}

func TestTalismanOrdinal(t *testing.T) {
	for key, wantOrd := range map[string]int{"talisman1": 0, "talisman2": 1, "talisman3": 2, "talisman4": 3} {
		if got, ok := talismanOrdinal(key); !ok || got != wantOrd {
			t.Errorf("talismanOrdinal(%q) = (%d,%v), want (%d,true)", key, got, ok, wantOrd)
		}
	}
	if _, ok := talismanOrdinal("talisman5"); ok {
		t.Error("talisman5 must not be a writable talisman ordinal")
	}
	if _, ok := talismanOrdinal("armorHead"); ok {
		t.Error("armorHead is not a talisman")
	}
}
