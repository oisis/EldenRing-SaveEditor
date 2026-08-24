package diagnostics_test

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// planFor runs a report of the whole slot and returns it together with the
// engine, the session and the revision the plan must be bound to. Every plan
// test starts from a real report, because an issue identifier that was not
// produced by one is not a thing a caller can have.
func planFor(t *testing.T, content reportTestFixture) (
	*saveengine.Engine, string, diagnostics.GetSaveValidationReportResult,
) {
	t.Helper()

	gameCatalog := reportTestCatalog(t)
	engine, session := loadReportFixture(t, content)
	report, err := diagnostics.GetSaveValidationReport(engine, gameCatalog, session, reportTestSlot, "")
	if err != nil {
		t.Fatalf("GetSaveValidationReport: %v", err)
	}
	return engine, session, report
}

// issueIDFor returns the identifier of the first finding carrying one code.
func issueIDFor(t *testing.T, report diagnostics.GetSaveValidationReportResult, code string) string {
	t.Helper()

	for _, issue := range report.Issues {
		if issue.Code == code {
			return issue.ID
		}
	}
	t.Fatalf("the report carries no %q finding; got %v", code, issueCodes(report))
	return ""
}

// requireAction returns the planned action resolving one issue code, failing
// when the plan refused it instead. An action may resolve several findings, so
// it is matched by the identifier it carries rather than by a single code.
func requireAction(
	t *testing.T,
	report diagnostics.GetSaveValidationReportResult,
	plan diagnostics.GetRepairPlanResult,
	code string,
) diagnostics.RepairAction {
	t.Helper()

	id := issueIDFor(t, report, code)
	for _, action := range plan.Actions {
		for _, planned := range action.IssueIDs {
			if planned == id {
				return action
			}
		}
	}
	for _, rejection := range plan.Rejected {
		if rejection.IssueID == id {
			t.Fatalf("issue %q was rejected instead of planned: %s", code, rejection.Reason)
		}
	}
	t.Fatalf("issue %q appears in neither the actions nor the rejections of the plan", code)
	return diagnostics.RepairAction{}
}

// countPlanned returns how many requested identifiers the plan accounted for,
// across shared actions and rejections alike.
func countPlanned(plan diagnostics.GetRepairPlanResult) int {
	total := len(plan.Rejected)
	for _, action := range plan.Actions {
		total += len(action.IssueIDs)
	}
	return total
}

// requireRejection returns the rejection of one issue code, failing when the
// plan produced an action for it. A wrongly planned repair is the dangerous
// direction, so this helper never accepts an action as a substitute.
func requireRejection(t *testing.T, plan diagnostics.GetRepairPlanResult, code string) diagnostics.RepairRejection {
	t.Helper()

	for _, action := range plan.Actions {
		for _, planned := range action.IssueIDs {
			if strings.Contains(planned, ":"+code+":") {
				t.Fatalf("issue %q was planned as %q, but it carries no confirmed repair",
					code, action.Operation)
			}
		}
	}
	for _, rejection := range plan.Rejected {
		if rejection.Code == code {
			if strings.TrimSpace(rejection.Reason) == "" {
				t.Errorf("issue %q was rejected without a reason", code)
			}
			return rejection
		}
	}
	t.Fatalf("issue %q appears in neither the actions nor the rejections of the plan", code)
	return diagnostics.RepairRejection{}
}

// allIssueIDs returns every identifier of a report.
func allIssueIDs(report diagnostics.GetSaveValidationReportResult) []string {
	ids := make([]string, 0, len(report.Issues))
	for _, issue := range report.Issues {
		ids = append(ids, issue.ID)
	}
	return ids
}

// TestSaveValidationReport_IssueIdentity protects the identifier contract the
// plan addresses findings by: every issue carries a non-empty identifier, no two
// findings of one report share one, and narrowing the scope does not renumber
// what it returns.
func TestSaveValidationReport_IssueIdentity(t *testing.T) {
	gameCatalog := reportTestCatalog(t)
	engine, session := loadReportFixture(t, reportTestFixture{
		// One level above the Vagabond base, so the stored zero really is below
		// the class-relative lifetime-rune minimum, while the stored level 40
		// still disagrees with the level the attributes produce.
		attributes: &reportTestOneLevelAboveBase,
		level:      func() *uint32 { level := uint32(40); return &level }(),
		soulMemory: func() *uint32 { memory := uint32(0); return &memory }(),
		inventory: []reportTestRow{
			{index: 0, handle: reportTestAccessory, quantity: 2},
			{index: 1, handle: reportTestGoodsSmallStack, quantity: 0},
		},
		storage: []reportTestRow{{index: 0, handle: reportTestGoodsNoStorage, quantity: 1}},
		spells:  map[int]uint32{12: reportTestSpellRaw},
	})

	full, err := diagnostics.GetSaveValidationReport(engine, gameCatalog, session, reportTestSlot, "")
	if err != nil {
		t.Fatalf("GetSaveValidationReport: %v", err)
	}
	if len(full.Issues) < 4 {
		t.Fatalf("the fixture produced %d issues, too few to prove identity", len(full.Issues))
	}

	seen := make(map[string]string, len(full.Issues))
	for _, issue := range full.Issues {
		if issue.ID == "" {
			t.Fatalf("issue %q carries no identifier", issue.Code)
		}
		if !strings.HasPrefix(issue.ID, issue.Scope+":"+issue.Code+":") {
			t.Errorf("issue ID %q does not name its own scope and code", issue.ID)
		}
		if previous, exists := seen[issue.ID]; exists {
			t.Fatalf("issue ID %q is shared by %q and %q", issue.ID, previous, issue.Code)
		}
		seen[issue.ID] = issue.Code
	}

	// The identifier is counted inside its scope, so a narrowed report must hand
	// back exactly the identifiers the full report gave for that scope. This is
	// the invariant that lets a caller take an ID from any report and use it.
	narrowed, err := diagnostics.GetSaveValidationReport(engine, gameCatalog, session, reportTestSlot, "stats")
	if err != nil {
		t.Fatalf("GetSaveValidationReport: %v", err)
	}
	if len(narrowed.Issues) == 0 {
		t.Fatal("the stats scope produced no issue, so this case would prove nothing")
	}
	for _, issue := range narrowed.Issues {
		if seen[issue.ID] != issue.Code {
			t.Errorf("narrowing the scope changed the identifier of %q to %q", issue.Code, issue.ID)
		}
	}
}

// TestGetRepairPlan_PlansOnlyConfirmedRepairs is the core regression: the four
// findings whose target state confirmed data determines become actions with the
// exact target value, and every finding whose resolution would need an invented
// policy is refused with a reason instead.
func TestGetRepairPlan_PlansOnlyConfirmedRepairs(t *testing.T) {
	engine, session, report := planFor(t, reportTestFixture{
		// One level above the Vagabond base, so the stored zero really is below
		// the class-relative lifetime-rune minimum, while the stored level 40
		// still disagrees with the level the attributes produce.
		attributes: &reportTestOneLevelAboveBase,
		level:      func() *uint32 { level := uint32(40); return &level }(),
		soulMemory: func() *uint32 { memory := uint32(0); return &memory }(),
		inventory: []reportTestRow{
			{index: 0, handle: reportTestAccessory, quantity: 2},
			{index: 1, handle: reportTestGoodsSmallStack, quantity: 0},
		},
		storage: []reportTestRow{{index: 0, handle: reportTestGoodsNoStorage, quantity: 1}},
		spells:  map[int]uint32{12: reportTestSpellRaw},
	})

	plan, err := diagnostics.GetRepairPlan(
		engine, reportTestCatalog(t), session, reportTestSlot, report.SaveRevision, allIssueIDs(report))
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}

	if countPlanned(plan) != len(report.Issues) {
		t.Fatalf("the plan accounts for %d of %d requested findings",
			countPlanned(plan), len(report.Issues))
	}

	t.Run("an accessory above its per-record limit is clamped to that limit", func(t *testing.T) {
		action := requireAction(t, report, plan, "quantity_above_stack_limit")
		if action.Operation != "set_owned_item_quantity" {
			t.Errorf("Operation = %q, want set_owned_item_quantity", action.Operation)
		}
		// The accessory is a separate-instance item, so its confirmed limit is
		// exactly one record per instance.
		if action.TargetValue != 1 {
			t.Errorf("TargetValue = %d, want the confirmed per-record limit 1", action.TargetValue)
		}
		if action.OwnedItemID == "" {
			t.Error("the action names no record")
		}
	})

	t.Run("a record with quantity 0 is planned for removal", func(t *testing.T) {
		action := requireAction(t, report, plan, "quantity_zero")
		if action.Operation != "remove_owned_item" {
			t.Errorf("Operation = %q, want remove_owned_item", action.Operation)
		}
		if action.OwnedItemID == "" {
			t.Error("the action names no record")
		}
	})

	t.Run("a level mismatch carries no repair contract of its own", func(t *testing.T) {
		requireRejection(t, plan, "level_mismatch")
	})

	t.Run("lifetime runes below the minimum carry no repair contract of their own", func(t *testing.T) {
		requireRejection(t, plan, "soul_memory_below_minimum")
	})

	t.Run("an item in a container that rejects it is refused", func(t *testing.T) {
		requireRejection(t, plan, "item_not_allowed_in_container")
	})

	t.Run("a reserved spell position is refused", func(t *testing.T) {
		requireRejection(t, plan, "reserved_spell_position_occupied")
	})
}

// TestGetRepairPlan_UnresolvedDataIsNeverRepaired is the safety regression for
// the rule that unknown data fails safely: a record this build cannot resolve
// must never turn into a deletion, a clamp or any other mutation.
func TestGetRepairPlan_UnresolvedDataIsNeverRepaired(t *testing.T) {
	engine, session, report := planFor(t, reportTestFixture{
		inventory: []reportTestRow{{index: 0, handle: 0xDEADBEEF, quantity: 1}},
		spells:    map[int]uint32{0: reportTestUnknownSpellRaw},
	})

	plan, err := diagnostics.GetRepairPlan(
		engine, reportTestCatalog(t), session, reportTestSlot, report.SaveRevision, allIssueIDs(report))
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}
	if len(plan.Actions) != 0 {
		t.Fatalf("unresolved data produced %d action(s): %+v", len(plan.Actions), plan.Actions)
	}
	if len(plan.Rejected) == 0 {
		t.Fatal("unresolved data produced no rejection, so the finding was silently dropped")
	}
	requireRejection(t, plan, "unresolved_equipped_spell")
}

// TestGetRepairPlan_IsBoundToTheSaveRevision proves a plan cannot be derived
// against a save version its identifiers no longer describe.
func TestGetRepairPlan_IsBoundToTheSaveRevision(t *testing.T) {
	engine, session, report := planFor(t, reportTestFixture{
		inventory: []reportTestRow{{index: 0, handle: reportTestAccessory, quantity: 2}},
	})
	gameCatalog := reportTestCatalog(t)
	ids := allIssueIDs(report)

	t.Run("a missing revision is rejected", func(t *testing.T) {
		if _, err := diagnostics.GetRepairPlan(
			engine, gameCatalog, session, reportTestSlot, "", ids); err == nil {
			t.Error("a plan was built without binding it to a save revision")
		}
	})

	t.Run("a foreign revision is rejected", func(t *testing.T) {
		if _, err := diagnostics.GetRepairPlan(
			engine, gameCatalog, session, reportTestSlot, "999999", ids); err == nil {
			t.Error("a plan was built against a revision the session is not at")
		}
	})

	t.Run("a revision that a mutation advanced past is rejected", func(t *testing.T) {
		if _, err := engine.SetCharacterRunes(session, reportTestSlot, 1234, report.SaveRevision); err != nil {
			t.Fatalf("SetCharacterRunes: %v", err)
		}
		if _, err := diagnostics.GetRepairPlan(
			engine, gameCatalog, session, reportTestSlot, report.SaveRevision, ids); err == nil {
			t.Error("a stale plan was built after the save advanced")
		}
	})
}

// TestGetRepairPlan_RejectsUnusableRequests covers the input boundary. Each case
// must fail rather than quietly produce a smaller plan than the caller asked
// for.
func TestGetRepairPlan_RejectsUnusableRequests(t *testing.T) {
	engine, session, report := planFor(t, reportTestFixture{
		inventory: []reportTestRow{{index: 0, handle: reportTestAccessory, quantity: 2}},
	})
	gameCatalog := reportTestCatalog(t)
	ids := allIssueIDs(report)
	if len(ids) == 0 {
		t.Fatal("the fixture produced no finding, so this test would prove nothing")
	}

	cases := []struct {
		name string
		ids  []string
	}{
		{name: "no identifier", ids: nil},
		{name: "an empty identifier", ids: []string{""}},
		{name: "a repeated identifier", ids: []string{ids[0], ids[0]}},
		{name: "an unknown identifier", ids: []string{"inventory:quantity_zero:99"}},
		{name: "an identifier of another scope", ids: []string{"stats:level_mismatch:0"}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := diagnostics.GetRepairPlan(
				engine, gameCatalog, session, reportTestSlot, report.SaveRevision, testCase.ids); err == nil {
				t.Errorf("%s was accepted", testCase.name)
			}
		})
	}

	t.Run("a missing engine is rejected", func(t *testing.T) {
		if _, err := diagnostics.GetRepairPlan(
			nil, gameCatalog, session, reportTestSlot, report.SaveRevision, ids); err == nil {
			t.Error("a plan was built without a save engine")
		}
	})

	t.Run("a missing catalog is rejected", func(t *testing.T) {
		if _, err := diagnostics.GetRepairPlan(
			engine, nil, session, reportTestSlot, report.SaveRevision, ids); err == nil {
			t.Error("a plan was built without a game catalog")
		}
	})

	t.Run("an inactive slot is rejected", func(t *testing.T) {
		inactiveEngine, inactiveSession := loadReportFixture(t, reportTestFixture{inactive: true})
		inactiveReport, err := diagnostics.GetSaveValidationReport(
			inactiveEngine, gameCatalog, inactiveSession, reportTestSlot, "")
		if err != nil {
			t.Fatalf("GetSaveValidationReport: %v", err)
		}
		if _, err := diagnostics.GetRepairPlan(inactiveEngine, gameCatalog, inactiveSession,
			reportTestSlot, inactiveReport.SaveRevision, ids); err == nil {
			t.Error("a plan was built for an inactive slot")
		}
	})
}

// TestGetRepairPlan_TokenSealsThePlan proves the token identifies this plan of
// this save version and nothing else, which is what lets ApplyRepairs execute a
// plan without any server-side reservation.
func TestGetRepairPlan_TokenSealsThePlan(t *testing.T) {
	engine, session, report := planFor(t, reportTestFixture{
		level: func() *uint32 { level := uint32(40); return &level }(),
		inventory: []reportTestRow{
			{index: 0, handle: reportTestAccessory, quantity: 2},
			{index: 1, handle: reportTestGoodsSmallStack, quantity: 0},
		},
	})
	gameCatalog := reportTestCatalog(t)

	full, err := diagnostics.GetRepairPlan(
		engine, gameCatalog, session, reportTestSlot, report.SaveRevision, allIssueIDs(report))
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}
	if full.PlanToken == "" {
		t.Fatal("the plan carries no token")
	}

	repeated, err := diagnostics.GetRepairPlan(
		engine, gameCatalog, session, reportTestSlot, report.SaveRevision, allIssueIDs(report))
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}
	if repeated.PlanToken != full.PlanToken {
		t.Errorf("the same plan produced two tokens: %q and %q", full.PlanToken, repeated.PlanToken)
	}

	// The identifiers are requested in reverse, which must not change the plan:
	// the order of a plan comes from the report, never from the request.
	reversed := allIssueIDs(report)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	unordered, err := diagnostics.GetRepairPlan(
		engine, gameCatalog, session, reportTestSlot, report.SaveRevision, reversed)
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}
	if unordered.PlanToken != full.PlanToken {
		t.Error("the request order changed the plan token")
	}

	subset, err := diagnostics.GetRepairPlan(
		engine, gameCatalog, session, reportTestSlot, report.SaveRevision, allIssueIDs(report)[:1])
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}
	if subset.PlanToken == full.PlanToken {
		t.Error("a smaller plan produced the token of the full plan")
	}
}

// TestGetRepairPlan_ChangesNothing proves the getter is non-mutating at the
// endpoint boundary: building a plan for real defects must leave the session
// revision and its unsaved-changes state exactly as they were.
func TestGetRepairPlan_ChangesNothing(t *testing.T) {
	engine, session, report := planFor(t, reportTestFixture{
		level:     func() *uint32 { level := uint32(40); return &level }(),
		inventory: []reportTestRow{{index: 0, handle: reportTestAccessory, quantity: 2}},
	})

	before, err := engine.GetSessionInfo(session)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}

	plan, err := diagnostics.GetRepairPlan(
		engine, reportTestCatalog(t), session, reportTestSlot, report.SaveRevision, allIssueIDs(report))
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}
	if len(plan.Actions) == 0 {
		t.Fatal("the fixture produced no action, so this test would prove nothing")
	}

	after, err := engine.GetSessionInfo(session)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if after != before {
		t.Errorf("the session changed: before %+v, after %+v", before, after)
	}

	// The same revision must still build the same plan, which it could not if
	// the first call had moved the save.
	again, err := diagnostics.GetRepairPlan(
		engine, reportTestCatalog(t), session, reportTestSlot, report.SaveRevision, allIssueIDs(report))
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}
	if again.PlanToken != plan.PlanToken {
		t.Error("the plan changed after being built once")
	}
}

// TestGetRepairPlan_AttributeRepairIsOneSharedAction is the regression for the
// atomic statistics contract: both attribute findings of one character are
// answered by a single SetCharacterStats over all eight attributes, never by two
// competing writes, and the target is the nearest legal set rather than an
// invented one.
func TestGetRepairPlan_AttributeRepairIsOneSharedAction(t *testing.T) {
	// Vagabond minima are {15, 10, 11, 14, 13, 9, 9, 7}. Vigor 0 is below both
	// the absolute minimum 1 and the class minimum 15; Mind 200 is above the
	// absolute maximum 99; Strength 12 is legal in 1..99 but below its class
	// minimum 14. Endurance 40 is legal on both rules and must not move.
	illegal := [8]uint32{0, 200, 40, 12, 13, 9, 9, 7}
	engine, session, report := planFor(t, reportTestFixture{attributes: &illegal})

	plan, err := diagnostics.GetRepairPlan(
		engine, reportTestCatalog(t), session, reportTestSlot, report.SaveRevision, allIssueIDs(report))
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}
	if countPlanned(plan) != len(report.Issues) {
		t.Fatalf("the plan accounts for %d of %d findings", countPlanned(plan), len(report.Issues))
	}

	statsActions := 0
	for _, action := range plan.Actions {
		if action.Operation == "set_character_stats" {
			statsActions++
		}
	}
	if statsActions != 1 {
		t.Fatalf("the plan carries %d set_character_stats actions, want exactly 1", statsActions)
	}

	action := requireAction(t, report, plan, "attribute_out_of_range")
	if action.Operation != "set_character_stats" {
		t.Fatalf("Operation = %q, want set_character_stats", action.Operation)
	}
	if action.Attributes == nil {
		t.Fatal("the statistics action carries no attribute set")
	}

	want := saveengine.CharacterAttributes{
		Vigor: 15, Mind: 99, Endurance: 40, Strength: 14,
		Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7,
	}
	if *action.Attributes != want {
		t.Errorf("Attributes = %+v, want %+v", *action.Attributes, want)
	}

	// Every attribute finding of the report must hang off this one action, so no
	// second write can be planned for the same block.
	for _, issue := range report.Issues {
		if issue.Code != "attribute_out_of_range" && issue.Code != "attribute_below_class_minimum" {
			continue
		}
		found := false
		for _, planned := range action.IssueIDs {
			if planned == issue.ID {
				found = true
			}
		}
		if !found {
			t.Errorf("finding %q is not resolved by the shared statistics action", issue.ID)
		}
	}
}

// TestGetRepairPlan_UnknownStartingClassIsNeverRepaired proves an attribute set
// whose class minima are not confirmed produces no write. An unknown class has
// no known legal target, and guessing one would corrupt a save the build does
// not understand.
func TestGetRepairPlan_UnknownStartingClassIsNeverRepaired(t *testing.T) {
	illegal := [8]uint32{0, 200, 40, 12, 13, 9, 9, 7}
	engine, session, report := planFor(t, reportTestFixture{
		attributes:    &illegal,
		startingClass: 200,
	})

	plan, err := diagnostics.GetRepairPlan(
		engine, reportTestCatalog(t), session, reportTestSlot, report.SaveRevision, allIssueIDs(report))
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}
	for _, action := range plan.Actions {
		if action.Operation == "set_character_stats" {
			t.Fatalf("an unknown starting class produced a statistics write: %+v", action)
		}
	}
	if len(plan.Rejected) == 0 {
		t.Fatal("an unknown starting class produced no rejection")
	}
}

func TestGetRepairPlan_DuplicateStackableRecordIsRefused(t *testing.T) {
	engine, session, report := planFor(t, reportTestFixture{
		inventory: []reportTestRow{
			{index: 0, handle: reportTestGoodsSmallStack, quantity: 4},
			{index: 1, handle: reportTestGoodsSmallStack, quantity: 4},
		},
	})

	plan, err := diagnostics.GetRepairPlan(
		engine, reportTestCatalog(t), session, reportTestSlot, report.SaveRevision, allIssueIDs(report))
	if err != nil {
		t.Fatalf("GetRepairPlan: %v", err)
	}

	rejection := requireRejection(t, plan, "duplicate_stackable_record")
	if !strings.Contains(rejection.Reason, "no confirmed safe automatic repair contract") {
		t.Errorf("rejection reason = %q, want safe contract message", rejection.Reason)
	}
}
