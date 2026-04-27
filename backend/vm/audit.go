package vm

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveEditor/backend/core"
	"github.com/oisis/EldenRing-SaveEditor/backend/db"
	gamedata "github.com/oisis/EldenRing-SaveEditor/backend/db/data"
)

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

// AuditCharacter runs VM-level checks (4A numeric + 4B partial flag scan +
// weapon upgrade) against an already-mapped CharacterViewModel. Operates
// purely on the VM — does not touch raw save bytes. Phase-2 raw checks
// (unknown item IDs, GaItem handle integrity) live in AuditSlot.
func AuditCharacter(c *CharacterViewModel) AuditReport {
	report := AuditReport{Issues: []AuditIssue{}}

	checkRunes(c, &report)
	checkAttributes(c, &report)
	checkLevel(c, &report)
	checkTalismanSlots(c, &report)
	checkItemQuantities(c, &report)
	checkSpiritAshUpgrades(c, &report)
	checkWeaponUpgrades(c, &report)
	checkItemFlags(c, &report)

	return report
}

// AuditSlot runs raw-data checks against the SaveSlot and APPENDS results to
// the existing report. Use this for checks that need access to raw inventory
// + GaMap (unknown item IDs, malformed handles) — these never reach the VM
// because MapParsedSlotToVM filters them out at parse time.
func AuditSlot(slot *core.SaveSlot, report *AuditReport) {
	checkUnknownItemIDs(slot, report)
	checkGaItemHandleIntegrity(slot, report)
	checkDerivedStats(slot, report)
	checkClearCountFlags(slot, report)
	checkDlcOwnership(slot, report)
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

// checkWeaponUpgrades flags weapons whose CurrentUpgrade exceeds MaxUpgrade.
// Smithing-stone weapons cap at +25, somber-stone at +10. IDs above the cap
// don't exist in regulation params, so server-side detection is plausible
// (analogous to spirit_ash_above_10).
func checkWeaponUpgrades(c *CharacterViewModel, r *AuditReport) {
	r.TotalChecks++
	any := false
	scan := func(items []ItemViewModel) {
		for _, item := range items {
			if item.Category != "Weapon" {
				continue
			}
			if item.MaxUpgrade == 0 || item.CurrentUpgrade <= item.MaxUpgrade {
				continue
			}
			any = true
			r.Issues = append(r.Issues, AuditIssue{
				Severity:   SeverityRisk,
				Confidence: ConfidenceReported,
				Category:   "inventory",
				Field:      fmt.Sprintf("%s (handle 0x%X)", item.Name, item.Handle),
				Message:    fmt.Sprintf("Weapon upgrade +%d exceeds max +%d", item.CurrentUpgrade, item.MaxUpgrade),
				Mitigation: fmt.Sprintf("Lower upgrade to ≤+%d", item.MaxUpgrade),
				RiskKey:    "weapon_upgrade_above_max",
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

// resolveItemID converts a GaItem handle to the catalogue itemID using the
// same logic as mapItems in character_vm.go. Returns 0 if the handle cannot
// be resolved (caller treats this as a separate handle-integrity issue).
func resolveItemID(handle uint32, gaMap map[uint32]uint32) uint32 {
	category := db.GetItemCategoryFromHandle(handle)
	if category == "Weapon" || category == "Armor" || category == "Ash of War" {
		id, ok := gaMap[handle]
		if !ok {
			return 0
		}
		return id
	}
	if category == "Unknown" {
		return 0
	}
	return db.HandleToItemID(handle)
}

// checkUnknownItemIDs walks raw inventory + storage and flags items whose
// resolved itemID is not present in the editor's catalogue. Speculated
// confidence — we don't know whether the server keeps an allowlist of valid
// IDs, but unknown IDs are at minimum a strong "this came from elsewhere"
// signal and at worst a cut-content marker we haven't classified yet.
func checkUnknownItemIDs(slot *core.SaveSlot, r *AuditReport) {
	r.TotalChecks++
	any := false
	scan := func(items []core.InventoryItem, bucket string) {
		for i, item := range items {
			if item.GaItemHandle == 0 || item.GaItemHandle == 0xFFFFFFFF {
				continue
			}
			itemID := resolveItemID(item.GaItemHandle, slot.GaMap)
			if itemID == 0 || itemID == 110000 {
				continue // unresolved handle / unarmed sentinel — handled elsewhere
			}
			data, _ := db.GetItemDataFuzzy(itemID)
			if data.Name != "" {
				continue
			}
			any = true
			r.Issues = append(r.Issues, AuditIssue{
				Severity:   SeverityWarn,
				Confidence: ConfidenceSpeculated,
				Category:   bucket,
				Field:      fmt.Sprintf("[%d] handle 0x%X (id 0x%X)", i, item.GaItemHandle, itemID),
				Message:    fmt.Sprintf("Item ID 0x%X not in catalogue — possibly cut content or fabricated ID", itemID),
				Mitigation: "Remove the item if you do not recognise it",
				RiskKey:    "unknown_item_id",
			})
		}
	}
	scan(slot.Inventory.CommonItems, "inventory")
	scan(slot.Inventory.KeyItems, "inventory")
	scan(slot.Storage.CommonItems, "storage")
	scan(slot.Storage.KeyItems, "storage")
	if !any {
		r.PassedChecks++
	}
}

// checkGaItemHandleIntegrity flags structural problems in inventory handles:
// (a) handles whose prefix is none of the four known categories (0x80 weapon,
// 0xA0 talisman, 0xB0 goods, 0xC0 AoW) — these will fail to load in-game;
// (b) Weapon/Armor/AoW handles that are not present in slot.GaMap — orphaned
// references that crash the slot loader. Confirmed confidence: this is a
// binary-level integrity requirement, not a ban-risk heuristic.
func checkGaItemHandleIntegrity(slot *core.SaveSlot, r *AuditReport) {
	r.TotalChecks++
	any := false
	scan := func(items []core.InventoryItem, bucket string) {
		for i, item := range items {
			if item.GaItemHandle == 0 || item.GaItemHandle == 0xFFFFFFFF {
				continue
			}
			category := db.GetItemCategoryFromHandle(item.GaItemHandle)
			if category == "Unknown" {
				any = true
				r.Issues = append(r.Issues, AuditIssue{
					Severity:   SeverityWarn,
					Confidence: ConfidenceConfirmed,
					Category:   bucket,
					Field:      fmt.Sprintf("[%d] handle 0x%X", i, item.GaItemHandle),
					Message:    "Handle prefix does not match any known category (expected 0x80 / 0xA0 / 0xB0 / 0xC0)",
					Mitigation: "Save will fail to load — restore from backup",
					RiskKey:    "gaitem_handle_invalid",
				})
				continue
			}
			if category == "Weapon" || category == "Armor" || category == "Ash of War" {
				if _, ok := slot.GaMap[item.GaItemHandle]; !ok {
					any = true
					r.Issues = append(r.Issues, AuditIssue{
						Severity:   SeverityWarn,
						Confidence: ConfidenceConfirmed,
						Category:   bucket,
						Field:      fmt.Sprintf("[%d] handle 0x%X", i, item.GaItemHandle),
						Message:    "Handle not present in GaItem map — orphaned reference",
						Mitigation: "Save will likely crash on load — restore from backup",
						RiskKey:    "gaitem_handle_invalid",
					})
				}
			}
		}
	}
	scan(slot.Inventory.CommonItems, "inventory")
	scan(slot.Inventory.KeyItems, "inventory")
	scan(slot.Storage.CommonItems, "storage")
	scan(slot.Storage.KeyItems, "storage")
	if !any {
		r.PassedChecks++
	}
}

// derived stat raw offsets within PlayerGameData (per spec/04)
const (
	maxHPOffset = 0x0C
	maxFPOffset = 0x18
	maxSPOffset = 0x28
)

// derivedStatTolerance accounts for float32 → uint32 rounding when comparing
// stored values against table-derived expectations.
const derivedStatTolerance uint32 = 1

// statTableLookup looks up the expected derived stat for an attribute value,
// converting the float32 table entry to uint32. Returns (0, false) if the
// attribute is out of the table's valid range.
func statTableLookup(table []float32, attr uint32) (uint32, bool) {
	if attr == 0 || int(attr) >= len(table) {
		return 0, false
	}
	return uint32(table[attr]), true
}

// absDiffU32 returns |a - b| for unsigned values without underflow.
func absDiffU32(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

// checkDerivedStats verifies that stored MaxHP / MaxFP / MaxSP match the
// values derived from Vigor / Mind / Endurance via the data.HP / FP / SP
// lookup tables. Mismatch is a "manually edited stored stat" marker.
// Confidence: Speculated — server-side stat-consistency checks are plausible
// but not publicly confirmed.
func checkDerivedStats(slot *core.SaveSlot, r *AuditReport) {
	r.TotalChecks++

	if !validatePlayerGameDataReachable(slot) {
		// Cannot read raw bytes — skip silently rather than false-positive.
		r.PassedChecks++
		return
	}

	sa := core.NewSlotAccessor(slot.Data)
	checks := []struct {
		field    string
		offset   int
		attrName string
		attrVal  uint32
		table    []float32
	}{
		{"MaxHP", maxHPOffset, "Vigor", slot.Player.Vigor, gamedata.HP},
		{"MaxFP", maxFPOffset, "Mind", slot.Player.Mind, gamedata.FP},
		{"MaxSP", maxSPOffset, "Endurance", slot.Player.Endurance, gamedata.SP},
	}

	any := false
	for _, c := range checks {
		stored, err := sa.ReadU32(PlayerGameDataOffset + c.offset)
		if err != nil {
			continue
		}
		expected, ok := statTableLookup(c.table, c.attrVal)
		if !ok {
			continue
		}
		if absDiffU32(stored, expected) <= derivedStatTolerance {
			continue
		}
		any = true
		r.Issues = append(r.Issues, AuditIssue{
			Severity:   SeverityRisk,
			Confidence: ConfidenceSpeculated,
			Category:   "consistency",
			Field:      c.field,
			Message:    fmt.Sprintf("%s = %d does not match expected %d (from %s = %d)", c.field, stored, expected, c.attrName, c.attrVal),
			Mitigation: fmt.Sprintf("Edit %s instead — %s is derived automatically", c.attrName, c.field),
			RiskKey:    "derived_stat_manual",
		})
	}
	if !any {
		r.PassedChecks++
	}
}

// validatePlayerGameDataReachable checks whether slot.Data is large enough
// to read the derived-stat region without bounds error.
func validatePlayerGameDataReachable(slot *core.SaveSlot) bool {
	return slot != nil && len(slot.Data) >= PlayerGameDataOffset+0x2C
}

// checkClearCountFlags verifies that event flags 50..57 mirror slot.Player.ClearCount —
// exactly one of those flags should be set, and its index should equal ClearCount.
// app.go's SaveCharacter writes this mirror automatically; a mismatch indicates
// the save was edited externally without going through this editor's write path.
// Confidence: Speculated — no public report of the server checking this, but
// it's a strong "edited save" marker because legitimate gameplay can never
// produce a desync between ClearCount and these flags.
func checkClearCountFlags(slot *core.SaveSlot, r *AuditReport) {
	r.TotalChecks++

	if slot == nil || slot.EventFlagsOffset == 0 || slot.EventFlagsOffset >= len(slot.Data) {
		// Event flags region not parsed — skip silently.
		r.PassedChecks++
		return
	}
	flags := slot.Data[slot.EventFlagsOffset:]

	setIndices := []uint32{}
	for i := uint32(0); i <= 7; i++ {
		v, err := db.GetEventFlag(flags, 50+i)
		if err != nil {
			// Flag region malformed — let other checks (or core diagnostics) report it.
			r.PassedChecks++
			return
		}
		if v {
			setIndices = append(setIndices, i)
		}
	}

	expected := slot.Player.ClearCount
	if len(setIndices) == 1 && setIndices[0] == expected {
		r.PassedChecks++
		return
	}

	var msg string
	switch {
	case len(setIndices) == 0:
		msg = fmt.Sprintf("ClearCount = %d but none of event flags 50-57 are set", expected)
	case len(setIndices) > 1:
		msg = fmt.Sprintf("ClearCount = %d but multiple NG+ flags are set (%v)", expected, setIndices)
	default:
		msg = fmt.Sprintf("ClearCount = %d but flag %d is set (expected flag %d)", expected, 50+setIndices[0], 50+expected)
	}

	r.Issues = append(r.Issues, AuditIssue{
		Severity:   SeverityRisk,
		Confidence: ConfidenceSpeculated,
		Category:   "consistency",
		Field:      "ClearCount / EventFlags 50-57",
		Message:    msg,
		Mitigation: "Save once via the editor — ClearCount ↔ event flag sync is automatic on write",
		RiskKey:    "clearcount_flag_mismatch",
	})
}

// dlcMismatchThreshold is the minimum DLC item count that triggers an
// ownership-mismatch issue. A single DLC item could come from a coop
// pickup or invader drop on a non-DLC character, so we wait until the
// inventory clearly reflects "stuff that needed DLC content to obtain".
const dlcMismatchThreshold = 3

// preOrderItemIDs are bonus DLC-tagged entries shipped with the base game
// pre-order — having them on a non-DLC save is legit. Excluded from the
// DLC item count.
var preOrderItemIDs = map[uint32]bool{
	0x401EA7A8: true, // Ring of Miquella gesture (pre-order bonus)
}

func checkDlcOwnership(slot *core.SaveSlot, r *AuditReport) {
	r.TotalChecks++

	flagOff := core.DlcSectionOffset + core.DlcEntryFlagByte
	if flagOff < 0 || flagOff+1 > len(slot.Data) {
		r.PassedChecks++
		return
	}
	entryFlag := slot.Data[flagOff]

	count := countDlcItems(slot.Inventory.CommonItems, slot.GaMap) +
		countDlcItems(slot.Inventory.KeyItems, slot.GaMap) +
		countDlcItems(slot.Storage.CommonItems, slot.GaMap)

	if count >= dlcMismatchThreshold && entryFlag == 0 {
		r.Issues = append(r.Issues, AuditIssue{
			Severity:   SeverityRisk,
			Confidence: ConfidenceReported,
			Category:   "ownership",
			Field:      "DLC items / SotE entry flag",
			Message: fmt.Sprintf("Inventory holds %d DLC-tagged items but the Shadow of the Erdtree entry flag is unset — character has not entered the DLC",
				count),
			Mitigation: "Either remove the DLC items, or load the save in-game and enter Shadow of the Erdtree once to legitimize the flag",
			RiskKey:    "dlc_ownership_mismatch",
		})
		return
	}
	r.PassedChecks++
}

// countDlcItems counts entries whose DB metadata carries the "dlc" flag,
// excluding pre-order bonuses that ship on non-DLC saves.
func countDlcItems(items []core.InventoryItem, gaMap map[uint32]uint32) int {
	count := 0
	for _, it := range items {
		if it.GaItemHandle == core.GaHandleEmpty || it.GaItemHandle == core.GaHandleInvalid {
			continue
		}
		id, ok := gaMap[it.GaItemHandle]
		if !ok {
			id = (it.GaItemHandle & 0x0FFFFFFF) | 0x40000000
		}
		if preOrderItemIDs[id] {
			continue
		}
		meta := db.GetItemData(id)
		if meta.Name == "" {
			continue
		}
		for _, f := range meta.Flags {
			if f == "dlc" {
				count++
				break
			}
		}
	}
	return count
}
