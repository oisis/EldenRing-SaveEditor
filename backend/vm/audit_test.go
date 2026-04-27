package vm

import (
	"testing"
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
	if report.TotalChecks != 7 {
		t.Errorf("expected 7 check categories, got %d", report.TotalChecks)
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
