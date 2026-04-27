package vm

import (
	"testing"

	"github.com/oisis/EldenRing-SaveEditor/backend/core"
	"github.com/oisis/EldenRing-SaveEditor/backend/db"
	gamedata "github.com/oisis/EldenRing-SaveEditor/backend/db/data"
)

// cleanCharacter returns a baseline VM that should pass every audit check.
// Adamant class baseline: Vigor 14, Mind 9, Endurance 12, Strength 14, Dex 13,
// Int 9, Faith 9, Arcane 7 → sum 87 → level 8.
func cleanCharacter() *CharacterViewModel {
	return &CharacterViewModel{
		Name:          "Test",
		Level:         8,
		Souls:         50_000,
		Vigor:         14,
		Mind:          9,
		Endurance:     12,
		Strength:      14,
		Dexterity:     13,
		Intelligence:  9,
		Faith:         9,
		Arcane:        7,
		TalismanSlots: 1,
		ClearCount:    0,
		Inventory:     []ItemViewModel{},
		Storage:       []ItemViewModel{},
	}
}

func findIssue(report AuditReport, riskKey string) *AuditIssue {
	for i := range report.Issues {
		if report.Issues[i].RiskKey == riskKey {
			return &report.Issues[i]
		}
	}
	return nil
}

func TestAuditCharacter_CleanSavePasses(t *testing.T) {
	c := cleanCharacter()
	report := AuditCharacter(c)
	if len(report.Issues) != 0 {
		t.Fatalf("expected 0 issues, got %d: %+v", len(report.Issues), report.Issues)
	}
	if report.PassedChecks != report.TotalChecks {
		t.Errorf("expected all %d checks to pass, got %d", report.TotalChecks, report.PassedChecks)
	}
	if report.TotalChecks != 8 {
		t.Errorf("expected 8 check categories, got %d", report.TotalChecks)
	}
}

func TestAuditCharacter_RunesOver999M(t *testing.T) {
	c := cleanCharacter()
	c.Souls = 1_500_000_000
	report := AuditCharacter(c)
	issue := findIssue(report, "runes_above_999m")
	if issue == nil {
		t.Fatal("expected runes_above_999m issue")
	}
	if issue.Severity != SeverityRisk {
		t.Errorf("expected SeverityRisk, got %d", issue.Severity)
	}
	if issue.Confidence != ConfidenceReported {
		t.Errorf("expected ConfidenceReported, got %s", issue.Confidence)
	}
}

func TestAuditCharacter_AttributeOver99(t *testing.T) {
	c := cleanCharacter()
	c.Strength = 100
	c.Level = c.Level + 86 // recalibrate level so we don't trip derived_stat_manual
	report := AuditCharacter(c)
	issue := findIssue(report, "stat_above_99")
	if issue == nil {
		t.Fatal("expected stat_above_99 issue")
	}
	if issue.Field != "Strength" {
		t.Errorf("expected Field=Strength, got %s", issue.Field)
	}
	if issue.Confidence != ConfidenceReported {
		t.Errorf("expected ConfidenceReported, got %s", issue.Confidence)
	}
}

func TestAuditCharacter_LevelOver713(t *testing.T) {
	c := cleanCharacter()
	c.Vigor = 99
	c.Mind = 99
	c.Endurance = 99
	c.Strength = 99
	c.Dexterity = 99
	c.Intelligence = 99
	c.Faith = 99
	c.Arcane = 99
	c.Level = 800 // > 713 cap
	report := AuditCharacter(c)
	issue := findIssue(report, "level_above_713")
	if issue == nil {
		t.Fatalf("expected level_above_713 issue, got: %+v", report.Issues)
	}
	if issue.Confidence != ConfidenceReported {
		t.Errorf("expected ConfidenceReported, got %s", issue.Confidence)
	}
}

func TestAuditCharacter_LevelInconsistent(t *testing.T) {
	c := cleanCharacter()
	c.Level = 500 // attrs sum is 87, expected level 8 — mismatch
	report := AuditCharacter(c)
	issue := findIssue(report, "derived_stat_manual")
	if issue == nil {
		t.Fatalf("expected derived_stat_manual issue, got: %+v", report.Issues)
	}
	if issue.Confidence != ConfidenceSpeculated {
		t.Errorf("expected ConfidenceSpeculated, got %s", issue.Confidence)
	}
}

func TestAuditCharacter_TalismanPouchOver3(t *testing.T) {
	c := cleanCharacter()
	c.TalismanSlots = 4
	report := AuditCharacter(c)
	issue := findIssue(report, "talisman_pouch_above_3")
	if issue == nil {
		t.Fatal("expected talisman_pouch_above_3 issue")
	}
	if issue.Severity != SeverityRisk {
		t.Errorf("expected SeverityRisk, got %d", issue.Severity)
	}
}

func TestAuditCharacter_QuantityOverEffectiveCap(t *testing.T) {
	c := cleanCharacter()
	c.ClearCount = 0
	c.Inventory = []ItemViewModel{
		{
			Handle:       0xB0000001,
			ID:           10070,
			Name:         "Stonesword Key",
			Category:     "Goods",
			SubCategory:  "key_items",
			Quantity:     100,
			MaxInventory: 55,
			Flags:        []string{"stackable", "scales_with_ng"},
		},
	}
	report := AuditCharacter(c)
	issue := findIssue(report, "quantity_above_max")
	if issue == nil {
		t.Fatal("expected quantity_above_max issue")
	}
}

func TestAuditCharacter_QuantityWithinNgScaling(t *testing.T) {
	c := cleanCharacter()
	c.ClearCount = 2 // NG+2 → cap = 55 × 3 = 165
	c.Inventory = []ItemViewModel{
		{
			Handle:       0xB0000001,
			Name:         "Stonesword Key",
			Quantity:     160,
			MaxInventory: 55,
			Flags:        []string{"stackable", "scales_with_ng"},
		},
	}
	report := AuditCharacter(c)
	if findIssue(report, "quantity_above_max") != nil {
		t.Fatalf("expected NO quantity issue (160 ≤ 165), got: %+v", report.Issues)
	}
}

func TestAuditCharacter_CutContentDetected(t *testing.T) {
	c := cleanCharacter()
	c.Inventory = []ItemViewModel{
		{
			Handle:   0xA0000099,
			Name:     "Pavel (test talisman)",
			Category: "Talisman",
			Flags:    []string{"cut_content"},
		},
	}
	report := AuditCharacter(c)
	issue := findIssue(report, "cut_content")
	if issue == nil {
		t.Fatal("expected cut_content issue")
	}
	if issue.Confidence != ConfidenceConfirmed {
		t.Errorf("expected ConfidenceConfirmed for cut_content, got %s", issue.Confidence)
	}
	if issue.Severity != SeverityRisk {
		t.Errorf("expected SeverityRisk, got %d", issue.Severity)
	}
}

func TestAuditCharacter_PreOrderItemReported(t *testing.T) {
	c := cleanCharacter()
	c.Storage = []ItemViewModel{
		{
			Handle: 0xB0000010,
			Name:   "Carian Oath gesture",
			Flags:  []string{"pre_order"},
		},
	}
	report := AuditCharacter(c)
	issue := findIssue(report, "pre_order")
	if issue == nil {
		t.Fatal("expected pre_order issue")
	}
	if issue.Confidence != ConfidenceReported {
		t.Errorf("expected ConfidenceReported, got %s", issue.Confidence)
	}
	if issue.Category != "storage" {
		t.Errorf("expected Category=storage, got %s", issue.Category)
	}
}

func TestAuditCharacter_SpiritAshOver10(t *testing.T) {
	c := cleanCharacter()
	c.Inventory = []ItemViewModel{
		{
			Handle:         0xB0000020,
			Name:           "Mimic Tear Ash",
			SubCategory:    "standard_ashes",
			CurrentUpgrade: 15,
			MaxUpgrade:     10,
			Flags:          []string{},
		},
	}
	report := AuditCharacter(c)
	issue := findIssue(report, "spirit_ash_above_10")
	if issue == nil {
		t.Fatal("expected spirit_ash_above_10 issue")
	}
}

func TestAuditCharacter_PassedChecksMath(t *testing.T) {
	c := cleanCharacter()
	c.Souls = 1_500_000_000 // fail runes only
	report := AuditCharacter(c)
	if report.PassedChecks != report.TotalChecks-1 {
		t.Errorf("expected PassedChecks = TotalChecks-1, got %d/%d", report.PassedChecks, report.TotalChecks)
	}
}

func TestAuditCharacter_MultipleAttributesOver99(t *testing.T) {
	c := cleanCharacter()
	c.Strength = 100
	c.Dexterity = 150
	c.Level = c.Level + 86 + 137 // re-balance to avoid level mismatch
	report := AuditCharacter(c)
	count := 0
	for _, iss := range report.Issues {
		if iss.RiskKey == "stat_above_99" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 stat_above_99 issues, got %d", count)
	}
	// Both attributes failing still counts as ONE failed check category
	if report.PassedChecks != report.TotalChecks-1 {
		t.Errorf("expected only attributes-check category to fail (PassedChecks=%d, TotalChecks=%d)", report.PassedChecks, report.TotalChecks)
	}
}

func TestAuditCharacter_WeaponUpgradeAboveMax(t *testing.T) {
	c := cleanCharacter()
	c.Inventory = []ItemViewModel{
		{
			Handle:         0x80000010,
			Name:           "Longsword",
			Category:       "Weapon",
			CurrentUpgrade: 30,
			MaxUpgrade:     25,
		},
	}
	report := AuditCharacter(c)
	issue := findIssue(report, "weapon_upgrade_above_max")
	if issue == nil {
		t.Fatal("expected weapon_upgrade_above_max issue")
	}
	if issue.Confidence != ConfidenceReported {
		t.Errorf("expected ConfidenceReported, got %s", issue.Confidence)
	}
}

func TestAuditCharacter_WeaponUpgradeWithinSomberCap(t *testing.T) {
	c := cleanCharacter()
	c.Inventory = []ItemViewModel{
		{
			Handle:         0x80000011,
			Name:           "Sacred Relic Sword",
			Category:       "Weapon",
			CurrentUpgrade: 10,
			MaxUpgrade:     10,
		},
	}
	report := AuditCharacter(c)
	if findIssue(report, "weapon_upgrade_above_max") != nil {
		t.Fatalf("expected NO weapon upgrade issue (10 == max 10)")
	}
}

// minimalSlot returns a SaveSlot with an empty inventory + GaMap, sufficient
// for AuditSlot tests that only care about specific items being present.
func minimalSlot() *core.SaveSlot {
	return &core.SaveSlot{
		GaMap: map[uint32]uint32{},
		Inventory: core.EquipInventoryData{
			CommonItems: []core.InventoryItem{},
			KeyItems:    []core.InventoryItem{},
		},
		Storage: core.EquipInventoryData{
			CommonItems: []core.InventoryItem{},
			KeyItems:    []core.InventoryItem{},
		},
	}
}

func TestAuditSlot_CleanSlotPasses(t *testing.T) {
	slot := minimalSlot()
	report := AuditReport{Issues: []AuditIssue{}}
	AuditSlot(slot, &report)
	if len(report.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d: %+v", len(report.Issues), report.Issues)
	}
	if report.TotalChecks != 4 {
		t.Errorf("expected 4 raw check categories, got %d", report.TotalChecks)
	}
	if report.PassedChecks != 4 {
		t.Errorf("expected 4 passed, got %d", report.PassedChecks)
	}
}

func TestAuditSlot_UnknownItemIDFlagged(t *testing.T) {
	slot := minimalSlot()
	// 0xB0... = goods handle prefix → resolved via HandleToItemID.
	// Lower bits encode an itemID that is not in the catalogue.
	slot.Inventory.CommonItems = []core.InventoryItem{
		{GaItemHandle: 0xB0BEEFFE, Quantity: 1},
	}
	report := AuditReport{Issues: []AuditIssue{}}
	AuditSlot(slot, &report)
	issue := findIssue(report, "unknown_item_id")
	if issue == nil {
		t.Fatalf("expected unknown_item_id issue, got: %+v", report.Issues)
	}
	if issue.Confidence != ConfidenceSpeculated {
		t.Errorf("expected ConfidenceSpeculated, got %s", issue.Confidence)
	}
}

func TestAuditSlot_BadHandlePrefixFlagged(t *testing.T) {
	slot := minimalSlot()
	// 0x70... is none of the known prefixes (0x80/0xA0/0xB0/0xC0).
	slot.Inventory.CommonItems = []core.InventoryItem{
		{GaItemHandle: 0x70000001, Quantity: 1},
	}
	report := AuditReport{Issues: []AuditIssue{}}
	AuditSlot(slot, &report)
	issue := findIssue(report, "gaitem_handle_invalid")
	if issue == nil {
		t.Fatalf("expected gaitem_handle_invalid issue, got: %+v", report.Issues)
	}
	if issue.Confidence != ConfidenceConfirmed {
		t.Errorf("expected ConfidenceConfirmed, got %s", issue.Confidence)
	}
}

func TestAuditSlot_OrphanedWeaponHandleFlagged(t *testing.T) {
	slot := minimalSlot()
	// Weapon handle (0x80...) but missing from GaMap → orphaned reference.
	slot.Inventory.CommonItems = []core.InventoryItem{
		{GaItemHandle: 0x80000099, Quantity: 1},
	}
	report := AuditReport{Issues: []AuditIssue{}}
	AuditSlot(slot, &report)
	issue := findIssue(report, "gaitem_handle_invalid")
	if issue == nil {
		t.Fatalf("expected gaitem_handle_invalid issue for orphan, got: %+v", report.Issues)
	}
}

func TestAuditSlot_EmptySentinelHandlesIgnored(t *testing.T) {
	slot := minimalSlot()
	slot.Inventory.CommonItems = []core.InventoryItem{
		{GaItemHandle: 0xFFFFFFFF, Quantity: 0},
		{GaItemHandle: 0x00000000, Quantity: 0},
	}
	report := AuditReport{Issues: []AuditIssue{}}
	AuditSlot(slot, &report)
	if len(report.Issues) != 0 {
		t.Errorf("expected sentinel handles to be skipped, got: %+v", report.Issues)
	}
}

// playerSlot builds a SaveSlot with slot.Data sized to cover PlayerGameData
// plus the event-flag region (large enough for flag IDs 50-57 wherever the
// lookup table maps them). Caller can populate Player fields and write raw
// bytes via core.SlotAccessor.
func playerSlot(t *testing.T) *core.SaveSlot {
	t.Helper()
	const bufSize = 2_000_000 // covers PlayerGameDataOffset + entire flag region
	slot := minimalSlot()
	slot.Data = make([]byte, bufSize)
	slot.EventFlagsOffset = PlayerGameDataOffset + 0x100 // arbitrary, after PGD
	return slot
}

// writeRawU32 writes a uint32 into slot.Data at the given offset for tests.
func writeRawU32(t *testing.T, slot *core.SaveSlot, off int, val uint32) {
	t.Helper()
	sa := core.NewSlotAccessor(slot.Data)
	if err := sa.WriteU32(off, val); err != nil {
		t.Fatalf("writeRawU32 at 0x%X: %v", off, err)
	}
}

func TestAuditSlot_DerivedStatsConsistent(t *testing.T) {
	slot := playerSlot(t)
	slot.Player.Vigor = 25
	slot.Player.Mind = 20
	slot.Player.Endurance = 15
	// Write expected MaxHP / MaxFP / MaxSP from the lookup tables.
	writeRawU32(t, slot, PlayerGameDataOffset+maxHPOffset, uint32(gamedataHP(25)))
	writeRawU32(t, slot, PlayerGameDataOffset+maxFPOffset, uint32(gamedataFP(20)))
	writeRawU32(t, slot, PlayerGameDataOffset+maxSPOffset, uint32(gamedataSP(15)))

	report := AuditReport{Issues: []AuditIssue{}}
	checkDerivedStats(slot, &report)
	if findIssue(report, "derived_stat_manual") != nil {
		t.Fatalf("expected NO derived_stat_manual issue, got: %+v", report.Issues)
	}
}

func TestAuditSlot_DerivedStatMismatch_HP(t *testing.T) {
	slot := playerSlot(t)
	slot.Player.Vigor = 25
	slot.Player.Mind = 20
	slot.Player.Endurance = 15
	// Write WRONG MaxHP (off by 500) to trip the mismatch check.
	writeRawU32(t, slot, PlayerGameDataOffset+maxHPOffset, uint32(gamedataHP(25))+500)
	writeRawU32(t, slot, PlayerGameDataOffset+maxFPOffset, uint32(gamedataFP(20)))
	writeRawU32(t, slot, PlayerGameDataOffset+maxSPOffset, uint32(gamedataSP(15)))

	report := AuditReport{Issues: []AuditIssue{}}
	checkDerivedStats(slot, &report)
	issue := findIssue(report, "derived_stat_manual")
	if issue == nil {
		t.Fatalf("expected derived_stat_manual issue, got: %+v", report.Issues)
	}
	if issue.Confidence != ConfidenceSpeculated {
		t.Errorf("expected ConfidenceSpeculated, got %s", issue.Confidence)
	}
	if issue.Field != "MaxHP" {
		t.Errorf("expected Field=MaxHP, got %s", issue.Field)
	}
}

func TestAuditSlot_DerivedStatsSkipWhenDataTooSmall(t *testing.T) {
	slot := minimalSlot()
	slot.Data = make([]byte, 100) // way too small for PlayerGameDataOffset
	slot.Player.Vigor = 25
	report := AuditReport{Issues: []AuditIssue{}}
	checkDerivedStats(slot, &report)
	if findIssue(report, "derived_stat_manual") != nil {
		t.Fatalf("expected silent skip when data too small, got issues: %+v", report.Issues)
	}
	if report.PassedChecks != 1 {
		t.Errorf("expected PassedChecks=1 (silent skip), got %d", report.PassedChecks)
	}
}

func TestAuditSlot_ClearCountFlagsMatch(t *testing.T) {
	slot := playerSlot(t)
	slot.Player.ClearCount = 2
	flags := slot.Data[slot.EventFlagsOffset:]
	if err := db.SetEventFlag(flags, 50+2, true); err != nil {
		t.Fatalf("SetEventFlag(52): %v", err)
	}

	report := AuditReport{Issues: []AuditIssue{}}
	checkClearCountFlags(slot, &report)
	if findIssue(report, "clearcount_flag_mismatch") != nil {
		t.Fatalf("expected NO clearcount_flag_mismatch issue, got: %+v", report.Issues)
	}
}

func TestAuditSlot_ClearCountFlagsMismatch(t *testing.T) {
	slot := playerSlot(t)
	slot.Player.ClearCount = 2
	flags := slot.Data[slot.EventFlagsOffset:]
	// Set the wrong flag — ClearCount=2 but flag 50 is set instead of 52.
	if err := db.SetEventFlag(flags, 50, true); err != nil {
		t.Fatalf("SetEventFlag(50): %v", err)
	}

	report := AuditReport{Issues: []AuditIssue{}}
	checkClearCountFlags(slot, &report)
	issue := findIssue(report, "clearcount_flag_mismatch")
	if issue == nil {
		t.Fatalf("expected clearcount_flag_mismatch issue, got: %+v", report.Issues)
	}
	if issue.Confidence != ConfidenceSpeculated {
		t.Errorf("expected ConfidenceSpeculated, got %s", issue.Confidence)
	}
}

func TestAuditSlot_ClearCountFlagsNoneSet(t *testing.T) {
	slot := playerSlot(t)
	slot.Player.ClearCount = 1
	// Do NOT set any flag — ClearCount=1 so flag 51 should have been set.

	report := AuditReport{Issues: []AuditIssue{}}
	checkClearCountFlags(slot, &report)
	if findIssue(report, "clearcount_flag_mismatch") == nil {
		t.Fatalf("expected mismatch issue when flag not set, got: %+v", report.Issues)
	}
}

func TestAuditSlot_ClearCountSkipWhenNoFlagsRegion(t *testing.T) {
	slot := minimalSlot()
	slot.EventFlagsOffset = 0 // signal: no flag region parsed
	slot.Player.ClearCount = 3

	report := AuditReport{Issues: []AuditIssue{}}
	checkClearCountFlags(slot, &report)
	if findIssue(report, "clearcount_flag_mismatch") != nil {
		t.Fatalf("expected silent skip when EventFlagsOffset=0, got: %+v", report.Issues)
	}
	if report.PassedChecks != 1 {
		t.Errorf("expected PassedChecks=1 (silent skip), got %d", report.PassedChecks)
	}
}

// gamedataHP / gamedataFP / gamedataSP read the same lookup tables that
// audit.go uses, so the tests' "expected" values stay in sync if the tables
// ever change.
func gamedataHP(attr uint32) float32 { return statTableEntry(gamedata.HP, attr) }
func gamedataFP(attr uint32) float32 { return statTableEntry(gamedata.FP, attr) }
func gamedataSP(attr uint32) float32 { return statTableEntry(gamedata.SP, attr) }

func statTableEntry(table []float32, attr uint32) float32 {
	if int(attr) < len(table) {
		return table[attr]
	}
	return 0
}
