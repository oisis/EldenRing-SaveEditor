/*
Endpoint: GetRepairPlan
EndpointID: get_repair_plan
Purpose: Builds a non-mutating plan of selected repairs bound to the current save version.
How it works: The runtime handler reads the validation facts of one character slot once, judges them with GetSaveValidationReport, resolves the requested findings against that report and turns only the findings with a confirmed, uniquely determined resolution into actions. Every other requested finding is returned as an explicit rejection with the reason it carries no plan. The result is sealed with a plan token derived from the session, the revision and the actions themselves, so ApplyRepairs can prove it is executing exactly this plan of exactly this save version.
Supported resource types: GameResource references.
Input variables: saveSessionID, characterID, saveRevision, issueIDs.
GameCatalog variables read: the container limits and stack rules GetSaveValidationReport already reads, reused to derive the target quantity of a clamping action.
Save variables read: the validation facts of one character slot, read once under one lock; the getter is non-mutating, reserves nothing and writes nothing.
Implementation status: implemented
*/
package diagnostics

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetRepairPlanEndpointID is the stable backend identifier of GetRepairPlan.
const GetRepairPlanEndpointID = "get_repair_plan"

// GetRepairPlanDefinition describes the public getter contract.
//
// characterID joins the variables the endpoint map lists because a plan is
// per-character: a validation issue identifier is unique inside one character
// slot and nowhere else, and ApplyRepairs already carries the same variable.
var GetRepairPlanDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetRepairPlan",
	ID:                         GetRepairPlanEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "saveRevision", "issueIDs"},
	Description:                "Builds a non-mutating plan of selected repairs bound to the current save version.",
})

// The repair operations a plan may contain. Each one names an existing mutating
// contract of this backend; the plan never invents a new way to write the save.
const (
	RepairOperationSetOwnedItemQuantity = saveengine.RepairOperationSetOwnedItemQuantity
	RepairOperationRemoveOwnedItem      = saveengine.RepairOperationRemoveOwnedItem
	RepairOperationSetCharacterStats    = saveengine.RepairOperationSetCharacterStats
)

// RepairAction is one planned change. It is a description, not an execution:
// building it changes nothing and reserves nothing.
//
// IssueIDs lists every finding this one action resolves, in report order. It is
// a list because the statistics block is repaired by a single atomic
// SetCharacterStats over all eight attributes, which can answer several findings
// at once; a container action always names exactly one. Each requested
// identifier still points at exactly one action or one rejection, so nothing is
// ever silently dropped — but an action may be shared.
//
// ponytail: TargetValue is one untyped number instead of a per-operation payload
// union, and Attributes is the one payload that needs more than a number. Split
// them the day a third shape appears.
type RepairAction struct {
	IssueIDs    []string                        `json:"issueIDs"`
	Scope       string                          `json:"scope"`
	Operation   string                          `json:"operation"`
	OwnedItemID string                          `json:"ownedItemID,omitempty"`
	TargetValue uint32                          `json:"targetValue,omitempty"`
	Attributes  *saveengine.CharacterAttributes `json:"attributes,omitempty"`
	Description string                          `json:"description"`
}

// RepairRejection is one requested finding this build refuses to plan for, with
// the reason. It exists so a caller never has to read an absent action as either
// "already fine" or "silently handled": a finding is planned or it is refused,
// never omitted.
type RepairRejection struct {
	IssueID string `json:"issueID"`
	Code    string `json:"code"`
	Scope   string `json:"scope"`
	Reason  string `json:"reason"`
}

// GetRepairPlanResult is the typed result of GetRepairPlan.
//
// Actions and Rejected together account for every requested identifier exactly
// once, both ordered by the order the validation report lists their findings, so
// two plans of the same revision and the same request are identical.
//
// PlanToken seals the plan. It is derived from the session, the revision, the
// character and every action, so ApplyRepairs can recompute it from a freshly
// derived plan and refuse anything that is not byte-for-byte the same plan of
// the same save version. It carries no server-side reservation and expires with
// nothing: a changed save simply produces a different token.
type GetRepairPlanResult struct {
	SaveSessionID string            `json:"saveSessionID"`
	SaveRevision  string            `json:"saveRevision"`
	CharacterID   int               `json:"characterID"`
	PlanToken     string            `json:"planToken"`
	Actions       []RepairAction    `json:"actions"`
	Rejected      []RepairRejection `json:"rejected"`
}

// GetRepairPlan builds a non-mutating repair plan for selected findings of one
// character slot of an existing save session.
//
// saveSessionID and saveRevision are both required and both matched exactly. The
// revision is what binds the plan to a save version: a plan derived from issue
// identifiers of an older revision would address findings that have since moved,
// so a revision that no longer matches the session is rejected rather than
// reinterpreted.
//
// issueIDs are the IDs of SaveValidationIssue values of a report of this same
// character and revision. An empty list, a repeated identifier and an identifier
// that names no finding of the current report are all rejected: this endpoint
// never guesses which defect a caller meant.
//
// A finding becomes an action only when its resolution is uniquely determined by
// confirmed data. Everything else becomes a rejection carrying the reason, which
// is deliberately the larger group: a defect whose repair needs a policy this
// project has not confirmed must stay a reported defect. Nothing is written,
// reserved, repaired or proposed beyond the returned description.
func GetRepairPlan(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	saveRevision string,
	issueIDs []string,
) (GetRepairPlanResult, error) {
	if engine == nil {
		return GetRepairPlanResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetRepairPlanResult{}, errors.New("game catalog is not available")
	}
	if saveRevision == "" {
		return GetRepairPlanResult{}, errors.New("saveRevision is required")
	}
	requestedIssues, err := requestedIssueIDs(issueIDs)
	if err != nil {
		return GetRepairPlanResult{}, err
	}

	facts, err := engine.GetSaveValidationFacts(saveSessionID, characterID)
	if err != nil {
		return GetRepairPlanResult{}, err
	}
	if facts.SaveRevision != saveRevision {
		return GetRepairPlanResult{}, fmt.Errorf(
			"saveRevision %q does not match the current revision %q of session %s",
			saveRevision, facts.SaveRevision, facts.SaveSessionID)
	}
	if !facts.Active {
		return GetRepairPlanResult{}, fmt.Errorf(
			"character %d is not active, so it has nothing to repair", facts.CharacterID)
	}

	// The whole report is judged, never a narrowed scope: an identifier may name
	// any scope, and a plan that could not see a scope could not reject its
	// identifiers with a reason either.
	report := buildSaveValidationReport(gameCatalog, facts, allValidationScopes())

	result := GetRepairPlanResult{
		SaveSessionID: facts.SaveSessionID,
		SaveRevision:  facts.SaveRevision,
		CharacterID:   facts.CharacterID,
		Actions:       []RepairAction{},
		Rejected:      []RepairRejection{},
	}
	// The statistics block is written by exactly one confirmed operation over all
	// eight attributes, so every attribute finding of one character shares a
	// single action. statsAction remembers where that action was placed, so the
	// second and later findings join it instead of planning a conflicting write.
	statsAction := -1
	for _, issue := range report.Issues {
		if !requestedIssues[issue.ID] {
			continue
		}
		delete(requestedIssues, issue.ID)

		if isAttributeIssue(issue.Code) {
			if statsAction >= 0 {
				result.Actions[statsAction].IssueIDs =
					append(result.Actions[statsAction].IssueIDs, issue.ID)
				continue
			}
			action, reason := planAttributeRepair(facts, issue)
			if reason != "" {
				result.Rejected = append(result.Rejected, reject(issue, reason))
				continue
			}
			statsAction = len(result.Actions)
			result.Actions = append(result.Actions, action)
			continue
		}

		action, reason := planRepairAction(gameCatalog, facts, issue)
		if reason != "" {
			result.Rejected = append(result.Rejected, reject(issue, reason))
			continue
		}
		result.Actions = append(result.Actions, action)
	}
	if len(requestedIssues) > 0 {
		return GetRepairPlanResult{}, fmt.Errorf(
			"issueIDs %v name no finding of the current report of character %d at revision %s",
			slices.Sorted(maps.Keys(requestedIssues)), facts.CharacterID, facts.SaveRevision)
	}

	result.PlanToken = repairPlanToken(result)
	return result, nil
}

// requestedIssueIDs turns the requested identifiers into a set. An empty request
// and a repeated identifier are rejected: an empty plan is never what a caller
// meant, and a repeated identifier would ask for the same repair twice.
func requestedIssueIDs(issueIDs []string) (map[string]bool, error) {
	if len(issueIDs) == 0 {
		return nil, errors.New("issueIDs is required and must name at least one finding")
	}
	requested := make(map[string]bool, len(issueIDs))
	for _, id := range issueIDs {
		if id == "" {
			return nil, errors.New("issueIDs must not contain an empty identifier")
		}
		if requested[id] {
			return nil, fmt.Errorf("issueIDs repeats %q", id)
		}
		requested[id] = true
	}
	return requested, nil
}

// allValidationScopes selects every scope of a validation pass.
func allValidationScopes() map[string]bool {
	requested := make(map[string]bool, len(saveValidationScopes))
	for _, name := range saveValidationScopes {
		requested[name] = true
	}
	return requested
}

// reject builds the refusal of one finding.
func reject(issue SaveValidationIssue, reason string) RepairRejection {
	return RepairRejection{
		IssueID: issue.ID,
		Code:    issue.Code,
		Scope:   issue.Scope,
		Reason:  reason,
	}
}

// isAttributeIssue reports whether a finding is answered by rewriting the eight
// attributes. Both codes are violations of an attribute rule and both are
// resolved by the same single write, so they can never be planned separately.
func isAttributeIssue(code string) bool {
	return code == IssueCodeAttributeOutOfRange || code == IssueCodeAttributeBelowClassMin
}

// planAttributeRepair derives the corrected attribute set of the character.
//
// The target comes from SaveEngine, which owns both the absolute range and the
// per-class minima, so the plan can never propose a set the executing endpoint
// would reject. Each attribute moves the smallest distance that makes it legal;
// a legal attribute is left alone.
//
// The action carries a deliberate consequence, stated in its description because
// it is not optional: SetCharacterStats always recalculates the level from the
// attributes it writes and raises the lifetime runes to the minimum the levels
// above the base level of the character's own starting class require.
// Repairing an attribute therefore moves the level too. There is no confirmed
// contract in this build that writes attributes without doing so.
func planAttributeRepair(
	facts saveengine.SaveValidationFacts,
	issue SaveValidationIssue,
) (RepairAction, string) {
	repaired, err := saveengine.LegalAttributesFor(
		facts.Stats.Attributes, facts.Stats.StartingClassID)
	if err != nil {
		return RepairAction{}, fmt.Sprintf(
			"no legal attribute set can be derived: %v", err)
	}
	if repaired == facts.Stats.Attributes {
		return RepairAction{}, "the stored attributes already satisfy both attribute rules, so the finding names no attribute this build can correct"
	}
	return RepairAction{
		IssueIDs:    []string{issue.ID},
		Scope:       issue.Scope,
		Operation:   RepairOperationSetCharacterStats,
		Attributes:  &repaired,
		Description: "set the eight attributes to the nearest set satisfying the range 1..99 and the minima of the character's starting class; this also recalculates the stored level from them and raises the lifetime runes to the minimum the levels above the base level of the character's own starting class require, which SetCharacterStats always does",
	}, ""
}

// planRepairAction turns one finding into either an action or the reason it
// carries none. It returns a non-empty reason for everything it refuses, and
// never both an action and a reason.
//
// The refusals are the point of this function. A repair is planned only where
// the target state is a single value that confirmed data already determines. A
// defect whose resolution needs a policy — which record of several to reduce,
// which spell of a loadout to unequip, whether an item in the wrong container is
// moved or destroyed, what an unknown identifier was meant to be — is refused,
// because inventing that policy here would turn a heuristic into a save
// mutation.
func planRepairAction(
	gameCatalog *gamecatalog.Catalog,
	facts saveengine.SaveValidationFacts,
	issue SaveValidationIssue,
) (RepairAction, string) {
	action := RepairAction{
		IssueIDs:    []string{issue.ID},
		Scope:       issue.Scope,
		OwnedItemID: issue.OwnedItemID,
	}

	switch issue.Code {
	case IssueCodeQuantityZero:
		action.Operation = RepairOperationRemoveOwnedItem
		action.Description = fmt.Sprintf(
			"remove the %s record %s, which occupies a slot with quantity 0",
			issue.Scope, issue.OwnedItemID)
		return action, ""

	case IssueCodeQuantityAboveStackLimit:
		limit, reason := recordStackLimit(gameCatalog, facts, issue)
		if reason != "" {
			return RepairAction{}, reason
		}
		action.Operation = RepairOperationSetOwnedItemQuantity
		action.TargetValue = limit
		action.Description = fmt.Sprintf(
			"set the quantity of the %s record %s to its confirmed per-record limit %d",
			issue.Scope, issue.OwnedItemID, limit)
		return action, ""

	case IssueCodeLevelMismatch, IssueCodeSoulMemoryBelowMinimum:
		// Neither has a repair contract of its own. SaveEngine writes the stored
		// level and the lifetime runes only as derived consequences of
		// SetCharacterStats, whose only accepted levelPolicy is "recalculate";
		// this build has no operation that writes either value independently.
		// Whether a save may have its level rewritten to match its attributes is
		// an open contract decision, not something a plan may settle on its own.
		return RepairAction{}, "the stored level and the lifetime runes are written only as derived consequences of SetCharacterStats, and whether this build may rewrite them to match the attributes is an unresolved contract decision"

	case IssueCodeUnresolvedItem, IssueCodeUnknownItem, IssueCodeUnresolvedSpell:
		return RepairAction{}, "this build cannot resolve the stored data, so no repair can be derived from it without guessing what it was meant to be"

	case IssueCodeQuantityAboveContainer:
		return RepairAction{}, "the finding names a container total, not one record, and no confirmed rule selects which of the records holding this item is reduced"

	case IssueCodeDuplicateStackableRecord:
		return RepairAction{}, "duplicate stackable records have no confirmed safe automatic repair contract"

	case IssueCodeItemNotAllowedInHere:
		return RepairAction{}, "no confirmed rule states whether an item stored in a container that does not accept it is moved or destroyed"

	case IssueCodeMemorySlotsExceeded:
		return RepairAction{}, "no confirmed rule selects which of the equipped spells is unequipped to fit the available memory slots"

	case IssueCodeDanglingReference, IssueCodeReservedSpellPosition:
		return RepairAction{}, "clearing the offending position is not yet a confirmed repair contract of this build"

	default:
		return RepairAction{}, fmt.Sprintf("issue code %q has no repair contract", issue.Code)
	}
}

// recordStackLimit re-derives the confirmed per-record limit of the record one
// finding names, through the same GameCatalog rule that produced the finding.
//
// It is deliberately a second derivation from the same facts rather than a value
// carried on the issue: the report states what is wrong and nothing about how to
// resolve it, and the limit is a repair target, not a defect.
func recordStackLimit(
	gameCatalog *gamecatalog.Catalog,
	facts saveengine.SaveValidationFacts,
	issue SaveValidationIssue,
) (uint32, string) {
	for _, record := range facts.Items {
		if record.OwnedItemID != issue.OwnedItemID || record.Container != issue.Scope {
			continue
		}
		if !record.Resolved {
			return 0, "the record the finding names no longer resolves"
		}
		resource, exists := gameCatalog.ItemByGameID(record.GameID)
		if !exists || resource.Kind != schema.ResourceKindItem || resource.Item == nil {
			return 0, "the record the finding names is not a known item"
		}
		_, perRecord, known := containerLimits(resource.Item, issue.Scope)
		if !known || perRecord == 0 {
			return 0, "the item the finding names carries no confirmed per-record limit"
		}
		return perRecord, ""
	}
	return 0, "the record the finding names is no longer present in its container"
}

// repairPlanToken seals a plan with a digest of everything that makes it this
// plan of this save version.
//
// It is derived and not random on purpose: ApplyRepairs recomputes it from a
// plan it derives itself and compares, so a plan needs no server-side storage,
// no expiry and no reservation. Any change to the save, the character or a
// single action produces a different token, and a token can therefore never
// authorise a mutation that was not the one described.
func repairPlanToken(result GetRepairPlanResult) string {
	var canonical strings.Builder
	fmt.Fprintf(&canonical, "%s\n%s\n%d\n",
		result.SaveSessionID, result.SaveRevision, result.CharacterID)
	for _, action := range result.Actions {
		fmt.Fprintf(&canonical, "%s\x1f%s\x1f%s\x1f%d",
			strings.Join(action.IssueIDs, ","), action.Operation,
			action.OwnedItemID, action.TargetValue)
		if attributes := action.Attributes; attributes != nil {
			// Spelled out in the confirmed save order rather than through %v on the
			// struct, so a future field added to CharacterAttributes cannot silently
			// change every previously issued token.
			fmt.Fprintf(&canonical, "\x1f%d,%d,%d,%d,%d,%d,%d,%d",
				attributes.Vigor, attributes.Mind, attributes.Endurance, attributes.Strength,
				attributes.Dexterity, attributes.Intelligence, attributes.Faith, attributes.Arcane)
		}
		canonical.WriteString("\x1e")
	}
	digest := sha256.Sum256([]byte(canonical.String()))
	return hex.EncodeToString(digest[:])
}
