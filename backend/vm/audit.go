package vm

import "fmt"

// Severity mirrors spec/32 ban-risk Tier (0=info, 1=caution, 2=high risk).
type Severity int

const (
	SeverityInfo Severity = 0
	SeverityWarn Severity = 1
	SeverityRisk Severity = 2
)

// Confidence is the umbrella tier for ban-detection rule provenance — see spec/35.
// Confirmed: vendor docs OR independently RE'd from binary/protocol.
// Reported:  multiple independent community sources, no technical proof.
// Speculated: single source, theoretical model, or heuristic.
type Confidence string

const (
	ConfidenceConfirmed  Confidence = "confirmed"
	ConfidenceReported   Confidence = "reported"
	ConfidenceSpeculated Confidence = "speculated"
)

const (
	MaxRunes            uint32 = 999_999_999
	MaxAttribute        uint32 = 99
	MaxLevelCap         uint32 = 713
	MaxTalismanSlots    uint8  = 3
	MaxSpiritAshUpgrade uint32 = 10
)

// AuditIssue is a single detected anomaly.
type AuditIssue struct {
	Severity   Severity   `json:"severity"`
	Confidence Confidence `json:"confidence"`
	Category   string     `json:"category"`
	Field      string     `json:"field"`
	Message    string     `json:"message"`
	Mitigation string     `json:"mitigation"`
	RiskKey    string     `json:"riskKey"`
}

// AuditReport is the full result of a save audit run.
// PassedChecks / TotalChecks count check categories, not individual items —
// scanning 1500 inventory entries is one check, not 1500.
type AuditReport struct {
	Issues       []AuditIssue `json:"issues"`
	PassedChecks int          `json:"passedChecks"`
	TotalChecks  int          `json:"totalChecks"`
}

// AuditCharacter runs all Phase-1 checks (4A numeric + 4B partial flag scan)
// against an already-mapped CharacterViewModel. Operates purely on the VM —
// does not touch raw save bytes.
func AuditCharacter(c *CharacterViewModel) AuditReport {
	report := AuditReport{Issues: []AuditIssue{}}

	checkRunes(c, &report)
	checkAttributes(c, &report)
	checkLevel(c, &report)
	checkTalismanSlots(c, &report)
	checkItemQuantities(c, &report)
	checkSpiritAshUpgrades(c, &report)
	checkItemFlags(c, &report)

	return report
}

func checkRunes(c *CharacterViewModel, r *AuditReport) {
	r.TotalChecks++
	if c.Souls > MaxRunes {
		r.Issues = append(r.Issues, AuditIssue{
			Severity:   SeverityRisk,
			Confidence: ConfidenceReported,
			Category:   "stats",
			Field:      "Souls",
			Message:    fmt.Sprintf("Runes %d exceed cap of 999,999,999", c.Souls),
			Mitigation: "Lower runes to ≤999,999,999 before going online",
			RiskKey:    "runes_above_999m",
		})
		return
	}
	r.PassedChecks++
}

func checkAttributes(c *CharacterViewModel, r *AuditReport) {
	r.TotalChecks++
	attrs := []struct {
		name string
		val  uint32
	}{
		{"Vigor", c.Vigor},
		{"Mind", c.Mind},
		{"Endurance", c.Endurance},
		{"Strength", c.Strength},
		{"Dexterity", c.Dexterity},
		{"Intelligence", c.Intelligence},
		{"Faith", c.Faith},
		{"Arcane", c.Arcane},
	}
	any := false
	for _, a := range attrs {
		if a.val > MaxAttribute {
			any = true
			r.Issues = append(r.Issues, AuditIssue{
				Severity:   SeverityRisk,
				Confidence: ConfidenceReported,
				Category:   "stats",
				Field:      a.name,
				Message:    fmt.Sprintf("%s = %d exceeds cap of 99", a.name, a.val),
				Mitigation: "Lower attribute to ≤99",
				RiskKey:    "stat_above_99",
			})
		}
	}
	if !any {
		r.PassedChecks++
	}
}

func checkLevel(c *CharacterViewModel, r *AuditReport) {
	r.TotalChecks++
	sum := c.Vigor + c.Mind + c.Endurance + c.Strength +
		c.Dexterity + c.Intelligence + c.Faith + c.Arcane
	expected := uint32(1)
	if sum > 79 {
		expected = sum - 79
	}

	if c.Level > MaxLevelCap {
		r.Issues = append(r.Issues, AuditIssue{
			Severity:   SeverityRisk,
			Confidence: ConfidenceReported,
			Category:   "stats",
			Field:      "Level",
			Message:    fmt.Sprintf("Level %d exceeds maximum 713", c.Level),
			Mitigation: "Level should equal sum(attributes) − 79",
			RiskKey:    "level_above_713",
		})
		return
	}
	if c.Level != expected {
		r.Issues = append(r.Issues, AuditIssue{
			Severity:   SeverityRisk,
			Confidence: ConfidenceSpeculated,
			Category:   "consistency",
			Field:      "Level",
			Message:    fmt.Sprintf("Level %d does not match attribute sum (expected %d)", c.Level, expected),
			Mitigation: "Re-save to recalculate level from attributes",
			RiskKey:    "derived_stat_manual",
		})
		return
	}
	r.PassedChecks++
}

func checkTalismanSlots(c *CharacterViewModel, r *AuditReport) {
	r.TotalChecks++
	if c.TalismanSlots > MaxTalismanSlots {
		r.Issues = append(r.Issues, AuditIssue{
			Severity:   SeverityRisk,
			Confidence: ConfidenceReported,
			Category:   "stats",
			Field:      "TalismanSlots",
			Message:    fmt.Sprintf("Talisman pouch slots = %d, cap is 3", c.TalismanSlots),
			Mitigation: "Lower to ≤3",
			RiskKey:    "talisman_pouch_above_3",
		})
		return
	}
	r.PassedChecks++
}

// itemEffectiveCap mirrors frontend DatabaseTab.effectiveCap() — see spec/34.
// scales_with_ng items: cap = base × (ClearCount + 1). Others: cap = base.
// Returns 0 if base is 0 (item has no inventory/storage cap defined — typically
// non-stackable singletons; quantity is forced to 1 elsewhere).
func itemEffectiveCap(item ItemViewModel, isStorage bool, clearCount uint32) uint32 {
	base := item.MaxInventory
	if isStorage {
		base = item.MaxStorage
	}
	if base == 0 {
		return 0
	}
	for _, f := range item.Flags {
		if f == "scales_with_ng" {
			return base * (clearCount + 1)
		}
	}
	return base
}

func checkItemQuantities(c *CharacterViewModel, r *AuditReport) {
	r.TotalChecks++
	any := false
	scan := func(items []ItemViewModel, isStorage bool) {
		bucket := "inventory"
		if isStorage {
			bucket = "storage"
		}
		for _, item := range items {
			cap := itemEffectiveCap(item, isStorage, c.ClearCount)
			if cap == 0 || item.Quantity <= cap {
				continue
			}
			any = true
			r.Issues = append(r.Issues, AuditIssue{
				Severity:   SeverityRisk,
				Confidence: ConfidenceReported,
				Category:   bucket,
				Field:      fmt.Sprintf("%s (handle 0x%X)", item.Name, item.Handle),
				Message:    fmt.Sprintf("Quantity %d exceeds effective cap %d (NG+%d)", item.Quantity, cap, c.ClearCount),
				Mitigation: fmt.Sprintf("Lower quantity to ≤%d", cap),
				RiskKey:    "quantity_above_max",
			})
		}
	}
	scan(c.Inventory, false)
	scan(c.Storage, true)
	if !any {
		r.PassedChecks++
	}
}

func checkSpiritAshUpgrades(c *CharacterViewModel, r *AuditReport) {
	r.TotalChecks++
	any := false
	scan := func(items []ItemViewModel) {
		for _, item := range items {
			if item.SubCategory != "standard_ashes" {
				continue
			}
			if item.CurrentUpgrade <= MaxSpiritAshUpgrade {
				continue
			}
			any = true
			r.Issues = append(r.Issues, AuditIssue{
				Severity:   SeverityRisk,
				Confidence: ConfidenceReported,
				Category:   "inventory",
				Field:      fmt.Sprintf("%s (handle 0x%X)", item.Name, item.Handle),
				Message:    fmt.Sprintf("Spirit ash upgrade +%d exceeds cap of +10", item.CurrentUpgrade),
				Mitigation: "Lower upgrade to ≤+10 or remove the item",
				RiskKey:    "spirit_ash_above_10",
			})
		}
	}
	scan(c.Inventory)
	scan(c.Storage)
	if !any {
		r.PassedChecks++
	}
}

// checkItemFlags scans inventory + storage for items carrying ban-risk flags
// (cut_content, pre_order, dlc_duplicate, ban_risk). cut_content is Confirmed
// per spec/35 master table; the rest are Reported.
func checkItemFlags(c *CharacterViewModel, r *AuditReport) {
	r.TotalChecks++
	any := false

	flagToRiskKey := map[string]struct {
		riskKey    string
		confidence Confidence
		mitigation string
	}{
		"cut_content":   {"cut_content", ConfidenceConfirmed, "Remove from inventory + clear acquisition flag before going online"},
		"pre_order":     {"pre_order", ConfidenceReported, "Safe only if your account owns the pre-order entitlement"},
		"dlc_duplicate": {"dlc_duplicate", ConfidenceReported, "Replace with the canonical (non-duplicate) variant"},
		"ban_risk":      {"ban_risk", ConfidenceReported, "Generic high-risk flag — use offline only"},
	}

	scan := func(items []ItemViewModel, bucket string) {
		for _, item := range items {
			for _, f := range item.Flags {
				meta, ok := flagToRiskKey[f]
				if !ok {
					continue
				}
				any = true
				r.Issues = append(r.Issues, AuditIssue{
					Severity:   SeverityRisk,
					Confidence: meta.confidence,
					Category:   bucket,
					Field:      fmt.Sprintf("%s (handle 0x%X)", item.Name, item.Handle),
					Message:    fmt.Sprintf("Item carries flag '%s'", f),
					Mitigation: meta.mitigation,
					RiskKey:    meta.riskKey,
				})
			}
		}
	}
	scan(c.Inventory, "inventory")
	scan(c.Storage, "storage")

	if !any {
		r.PassedChecks++
	}
}
