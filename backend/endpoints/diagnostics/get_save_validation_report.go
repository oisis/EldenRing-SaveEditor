/*
Endpoint: GetSaveValidationReport
EndpointID: get_save_validation_report
Purpose: Runs non-mutating save validation and returns detected problems without proposing unconfirmed repairs.
How it works: The runtime handler asks SaveEngine once for every fact one validation pass needs of a single character slot, gathered under one lock and one save revision, and then judges those facts against the GameCatalog rules the mutating endpoints already enforce. It opens no file, parses no save data of its own, writes nothing and proposes no repair: naming a defect is this endpoint's job, resolving one belongs to GetRepairPlan and ApplyRepairs.
Supported resource types: GameResource references.
Input variables: saveSessionID, characterID, scope.
GameCatalog variables read: for every resolved container record the resource kind, key, item family and presentation name, item.storage.recordMode, item.storage.maxInventory and item.storage.maxStorage, and item.capabilities.stack.known, item.capabilities.stack.enabled and item.capabilities.stack.rules.maxPerStack; for every occupied spell record the spell item family, the spell game-ID prefix, the spell Memory Slots cost and the spell-memory equipment capability.
Save variables read: the UserData10 activity flag of the requested slot and, for an active slot, the InventoryHeld and Storage Box records with the GaItem table their handles resolve through, the eight PlayerGameData attributes with the stored level, lifetime runes and starting class, the Equipment, Quick Item and Pouch reference pairs, and the fourteen EquippedSpells records with the memory capacity derived from the slot; the getter is non-mutating and writes nothing.
Implementation status: implemented
*/
package diagnostics

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetSaveValidationReportEndpointID is the stable backend identifier of GetSaveValidationReport.
const GetSaveValidationReportEndpointID = "get_save_validation_report"

// GetSaveValidationReportDefinition describes the public getter contract.
var GetSaveValidationReportDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetSaveValidationReport",
	ID:                         GetSaveValidationReportEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "scope"},
	Description:                "Runs non-mutating save validation and returns detected problems without proposing unconfirmed repairs.",
})

// The two severities a finding carries. An error names a state the save cannot
// legally be in; a warning names a state this build cannot judge, which is data
// the report has to make visible rather than silently accept.
const (
	SaveValidationSeverityError   = "error"
	SaveValidationSeverityWarning = "warning"
)

// The issue codes this report can emit. They are stable identifiers a caller may
// branch on; the human-readable Message beside them is not.
const (
	IssueCodeUnresolvedItem           = "unresolved_item"
	IssueCodeUnknownItem              = "unknown_item"
	IssueCodeQuantityZero             = "quantity_zero"
	IssueCodeQuantityAboveStackLimit  = "quantity_above_stack_limit"
	IssueCodeQuantityAboveContainer   = "quantity_above_container_limit"
	IssueCodeItemNotAllowedInHere     = "item_not_allowed_in_container"
	IssueCodeAttributeOutOfRange      = "attribute_out_of_range"
	IssueCodeLevelMismatch            = "level_mismatch"
	IssueCodeAttributeBelowClassMin   = "attribute_below_class_minimum"
	IssueCodeSoulMemoryBelowMinimum   = "soul_memory_below_minimum"
	IssueCodeDanglingReference        = "dangling_equipment_reference"
	IssueCodeDuplicateStackableRecord = "duplicate_stackable_record"
	IssueCodeReservedSpellPosition    = "reserved_spell_position_occupied"
	IssueCodeUnresolvedSpell          = "unresolved_equipped_spell"
	IssueCodeMemorySlotsExceeded      = "memory_slots_exceeded"
)

// saveValidationScopes lists every scope in the order a report presents them.
// The empty scope input means all of them.
var saveValidationScopes = []string{
	saveengine.ValidationScopeInventory,
	saveengine.ValidationScopeStorage,
	saveengine.ValidationScopeStats,
	saveengine.ValidationScopeEquipment,
	saveengine.ValidationScopeSpells,
}

// SaveValidationIssue is one detected problem. It states what is wrong and
// where, and deliberately nothing about how to resolve it: this getter proposes
// no action, no default action and no repair.
//
// ID addresses this one finding for GetRepairPlan. It is derived from the scope,
// the code and the position of the finding inside its own scope, so it is stable
// for a given SaveRevision and unaffected by the scope filter: a scope is always
// judged as a whole, so narrowing the report can never renumber what it returns.
// Like OwnedItemID it is valid for the SaveRevision of the report and for
// nothing else.
//
// OwnedItemID identifies the container record a record-scoped issue was found
// in, valid for the SaveRevision of the report and for nothing else. It is empty
// for an issue that names no record.
type SaveValidationIssue struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Scope       string `json:"scope"`
	Message     string `json:"message"`
	OwnedItemID string `json:"ownedItemID"`
}

// SaveValidationScopeCoverage reports what one scope actually checked.
//
// It exists so an empty issue list can be read correctly. Checked false with a
// Reason means the scope could not be decoded and its data was therefore not
// judged at all — the save is not thereby clean. UnresolvedRecords counts the
// records inside a checked scope whose stored handle resolved to nothing, so the
// part of a scope that stayed unjudged is visible too.
type SaveValidationScopeCoverage struct {
	Scope             string `json:"scope"`
	Checked           bool   `json:"checked"`
	Reason            string `json:"reason"`
	RecordsChecked    int    `json:"recordsChecked"`
	UnresolvedRecords int    `json:"unresolvedRecords"`
}

// GetSaveValidationReportResult is the typed result of GetSaveValidationReport.
//
// Issues are ordered by scope in the order Coverage lists them, and inside a
// scope in the stored order of the data they were found in, so two reports of
// the same revision are identical.
//
// An inactive slot — including a residual one — reports Active false, no issue
// and every requested scope unchecked with that reason: the residual data of a
// deleted character is never judged.
type GetSaveValidationReportResult struct {
	SaveSessionID string                        `json:"saveSessionID"`
	SaveRevision  string                        `json:"saveRevision"`
	CharacterID   int                           `json:"characterID"`
	Active        bool                          `json:"active"`
	Coverage      []SaveValidationScopeCoverage `json:"coverage"`
	Issues        []SaveValidationIssue         `json:"issues"`
	ErrorCount    int                           `json:"errorCount"`
	WarningCount  int                           `json:"warningCount"`
}

// GetSaveValidationReport runs one non-mutating validation pass over a single
// character slot of an existing save session and returns the problems it found.
//
// The endpoint is thin: it rejects a missing engine, a missing catalog and an
// unknown scope, asks SaveEngine once for the facts of the whole pass, and joins
// them with GameCatalog. Locating and decoding save data and applying the
// save-side rules belong to SaveEngine; container limits, stack limits and spell
// costs belong to GameCatalog. The session must already exist; this endpoint
// never creates one, opens no file and returns no raw save byte.
//
// scope selects one of "inventory", "storage", "stats", "equipment" and
// "spells"; the empty string means all five. It is matched exactly and never
// trimmed or normalised. A narrowed scope changes only which findings are
// reported, never how a scope is judged: the whole pass is read either way, so
// two reports can never disagree about the same data.
//
// Nothing is repaired, proposed or written, and unknown data is never converted
// into a defect: a record this build cannot resolve becomes a warning together
// with the coverage entry that says so.
func GetSaveValidationReport(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	scope string,
) (GetSaveValidationReportResult, error) {
	if engine == nil {
		return GetSaveValidationReportResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetSaveValidationReportResult{}, errors.New("game catalog is not available")
	}
	requested, err := requestedValidationScopes(scope)
	if err != nil {
		return GetSaveValidationReportResult{}, err
	}

	facts, err := engine.GetSaveValidationFacts(saveSessionID, characterID)
	if err != nil {
		return GetSaveValidationReportResult{}, err
	}
	return buildSaveValidationReport(gameCatalog, facts, requested), nil
}

// buildSaveValidationReport judges one already-read set of facts. It exists so
// GetRepairPlan can derive a plan and the report it explains from the same facts
// of the same lock and the same revision, instead of reading the save twice and
// risking two reports that disagree.
func buildSaveValidationReport(
	gameCatalog *gamecatalog.Catalog,
	facts saveengine.SaveValidationFacts,
	requested map[string]bool,
) GetSaveValidationReportResult {
	result := GetSaveValidationReportResult{
		SaveSessionID: facts.SaveSessionID,
		SaveRevision:  facts.SaveRevision,
		CharacterID:   facts.CharacterID,
		Active:        facts.Active,
		Coverage:      []SaveValidationScopeCoverage{},
		Issues:        []SaveValidationIssue{},
	}
	for _, name := range saveValidationScopes {
		if !requested[name] {
			continue
		}
		if !facts.Active {
			result.Coverage = append(result.Coverage, SaveValidationScopeCoverage{
				Scope:  name,
				Reason: fmt.Sprintf("character %d is not active", facts.CharacterID),
			})
			continue
		}

		coverage, issues := checkValidationScope(gameCatalog, facts, name)
		for index := range issues {
			issues[index].ID = validationIssueID(name, issues[index].Code, index)
		}
		result.Coverage = append(result.Coverage, coverage)
		result.Issues = append(result.Issues, issues...)
	}

	for _, issue := range result.Issues {
		if issue.Severity == SaveValidationSeverityError {
			result.ErrorCount++
			continue
		}
		result.WarningCount++
	}
	return result
}

// validationIssueID builds the stable identifier of one finding. The position is
// counted inside the scope and not inside the report, because only the scope is
// judged as an indivisible unit: numbering across the whole report would shift
// every identifier as soon as a caller narrowed the scope.
//
// The code travels inside the identifier so a caller replaying an identifier
// against a report it does not belong to is rejected instead of silently
// repairing a different finding.
func validationIssueID(scope string, code string, indexInScope int) string {
	return fmt.Sprintf("%s:%s:%d", scope, code, indexInScope)
}

// requestedValidationScopes turns the scope input into the set of scopes to
// report. An unknown scope is rejected instead of silently reporting nothing.
func requestedValidationScopes(scope string) (map[string]bool, error) {
	requested := make(map[string]bool, len(saveValidationScopes))
	if scope == "" {
		for _, name := range saveValidationScopes {
			requested[name] = true
		}
		return requested, nil
	}
	for _, name := range saveValidationScopes {
		if name == scope {
			requested[name] = true
			return requested, nil
		}
	}
	return nil, fmt.Errorf("scope must be one of %v or empty; got %q", saveValidationScopes, scope)
}

// checkValidationScope judges one scope of an active slot and returns its
// coverage entry together with the issues it found.
func checkValidationScope(
	gameCatalog *gamecatalog.Catalog,
	facts saveengine.SaveValidationFacts,
	scope string,
) (SaveValidationScopeCoverage, []SaveValidationIssue) {
	coverage := SaveValidationScopeCoverage{Scope: scope, Checked: true}

	switch scope {
	case saveengine.ValidationScopeInventory:
		if facts.InventoryFailure != "" {
			return SaveValidationScopeCoverage{Scope: scope, Reason: facts.InventoryFailure}, nil
		}
		return checkContainer(gameCatalog, facts, scope)
	case saveengine.ValidationScopeStorage:
		if facts.StorageFailure != "" {
			return SaveValidationScopeCoverage{Scope: scope, Reason: facts.StorageFailure}, nil
		}
		return checkContainer(gameCatalog, facts, scope)
	case saveengine.ValidationScopeStats:
		if facts.StatsFailure != "" {
			return SaveValidationScopeCoverage{Scope: scope, Reason: facts.StatsFailure}, nil
		}
		coverage.RecordsChecked = 1
		return coverage, checkStats(facts.Stats)
	case saveengine.ValidationScopeEquipment:
		if facts.EquipmentFailure != "" {
			return SaveValidationScopeCoverage{Scope: scope, Reason: facts.EquipmentFailure}, nil
		}
		issues := make([]SaveValidationIssue, 0, len(facts.DanglingReferences))
		for _, reference := range facts.DanglingReferences {
			issues = append(issues, SaveValidationIssue{
				Code:     IssueCodeDanglingReference,
				Severity: SaveValidationSeverityError,
				Scope:    scope,
				Message: fmt.Sprintf("%s %d references inventory row %d with handle 0x%08X, but %s",
					reference.Structure, reference.Slot, reference.Row, reference.Handle, reference.Reason),
			})
		}
		coverage.RecordsChecked = len(facts.DanglingReferences)
		return coverage, issues
	case saveengine.ValidationScopeSpells:
		if facts.SpellsFailure != "" {
			return SaveValidationScopeCoverage{Scope: scope, Reason: facts.SpellsFailure}, nil
		}
		return checkSpells(gameCatalog, facts)
	default:
		return SaveValidationScopeCoverage{Scope: scope, Reason: "scope is not implemented"}, nil
	}
}

// checkContainer judges every record of one container against the GameCatalog
// limits the mutating endpoints enforce.
//
// The rules are the confirmed ones: a quantity-stack record may hold at most
// min(maxPerStack, container total), a separate-instance record holds exactly
// one, and the sum of a game ID across the container may not exceed the
// container total. A container total of zero means the item does not belong in
// that container at all.
//
// A record whose handle resolved to nothing, and one whose game ID GameCatalog
// does not know, is a warning: it is reported and counted as unresolved instead
// of being judged against limits that were never established for it.
func checkContainer(
	gameCatalog *gamecatalog.Catalog,
	facts saveengine.SaveValidationFacts,
	container string,
) (SaveValidationScopeCoverage, []SaveValidationIssue) {
	coverage := SaveValidationScopeCoverage{Scope: container, Checked: true}
	issues := make([]SaveValidationIssue, 0)
	totals := make(map[uint32]uint64)
	limits := make(map[uint32]uint32)
	stackCounts := make(map[uint32]int)
	isStack := make(map[uint32]bool)

	for _, record := range facts.Items {
		if record.Container != container {
			continue
		}
		coverage.RecordsChecked++

		if !record.Resolved {
			coverage.UnresolvedRecords++
			issues = append(issues, SaveValidationIssue{
				Code:        IssueCodeUnresolvedItem,
				Severity:    SaveValidationSeverityWarning,
				Scope:       container,
				Message:     fmt.Sprintf("handle 0x%08X was not resolved: %s", record.GaItemHandle, record.ResolutionError),
				OwnedItemID: record.OwnedItemID,
			})
			continue
		}

		resource, exists := gameCatalog.ItemByGameID(record.GameID)
		if !exists || resource.Kind != schema.ResourceKindItem || resource.Item == nil || resource.Key == "" {
			coverage.UnresolvedRecords++
			issues = append(issues, SaveValidationIssue{
				Code:        IssueCodeUnknownItem,
				Severity:    SaveValidationSeverityWarning,
				Scope:       container,
				Message:     fmt.Sprintf("game ID 0x%08X is not a known item", record.GameID),
				OwnedItemID: record.OwnedItemID,
			})
			continue
		}

		if resource.Item.Storage.RecordMode.Known && resource.Item.Storage.RecordMode.Value == schema.RecordModeQuantityStack {
			isStack[record.GameID] = true
			stackCounts[record.GameID]++
		}

		if record.Quantity == 0 {
			issues = append(issues, SaveValidationIssue{
				Code:        IssueCodeQuantityZero,
				Severity:    SaveValidationSeverityError,
				Scope:       container,
				Message:     fmt.Sprintf("%q occupies a record with quantity 0", resource.Key),
				OwnedItemID: record.OwnedItemID,
			})
		}

		containerTotal, perRecord, known := containerLimits(resource.Item, container)
		if !known {
			coverage.UnresolvedRecords++
			issues = append(issues, SaveValidationIssue{
				Code:        IssueCodeUnresolvedItem,
				Severity:    SaveValidationSeverityWarning,
				Scope:       container,
				Message:     fmt.Sprintf("%q carries no confirmed %s limit", resource.Key, container),
				OwnedItemID: record.OwnedItemID,
			})
			continue
		}
		if containerTotal == 0 {
			issues = append(issues, SaveValidationIssue{
				Code:        IssueCodeItemNotAllowedInHere,
				Severity:    SaveValidationSeverityError,
				Scope:       container,
				Message:     fmt.Sprintf("%q does not belong in the %s", resource.Key, container),
				OwnedItemID: record.OwnedItemID,
			})
			continue
		}
		if record.Quantity > perRecord {
			issues = append(issues, SaveValidationIssue{
				Code:        IssueCodeQuantityAboveStackLimit,
				Severity:    SaveValidationSeverityError,
				Scope:       container,
				Message:     fmt.Sprintf("%q holds %d in one record, want at most %d", resource.Key, record.Quantity, perRecord),
				OwnedItemID: record.OwnedItemID,
			})
		}
		totals[record.GameID] += uint64(record.Quantity)
		limits[record.GameID] = containerTotal
	}

	// Duplicate records for a quantity_stack item are reported once per game ID,
	// keeping stored order of first appearance.
	reportedDuplicates := make(map[uint32]bool)
	for _, record := range facts.Items {
		if record.Container != container || !record.Resolved || reportedDuplicates[record.GameID] {
			continue
		}
		if isStack[record.GameID] && stackCounts[record.GameID] > 1 {
			reportedDuplicates[record.GameID] = true
			issues = append(issues, SaveValidationIssue{
				Code:     IssueCodeDuplicateStackableRecord,
				Severity: SaveValidationSeverityError,
				Scope:    container,
				Message: fmt.Sprintf("game ID 0x%08X holds %d duplicate stack records in the %s",
					record.GameID, stackCounts[record.GameID], container),
			})
		}
	}

	// The container total is a property of the game ID, not of one record, so it
	// is judged once per item after every record of the container was counted.
	// The result keeps the stored order by walking the records again.
	reported := make(map[uint32]bool, len(totals))
	for _, record := range facts.Items {
		if record.Container != container || !record.Resolved || reported[record.GameID] {
			continue
		}
		limit, judged := limits[record.GameID]
		if !judged || totals[record.GameID] <= uint64(limit) {
			continue
		}
		reported[record.GameID] = true
		issues = append(issues, SaveValidationIssue{
			Code:     IssueCodeQuantityAboveContainer,
			Severity: SaveValidationSeverityError,
			Scope:    container,
			Message: fmt.Sprintf("game ID 0x%08X holds %d in the %s, want at most %d",
				record.GameID, totals[record.GameID], container, limit),
		})
	}
	return coverage, issues
}

// containerLimits returns the confirmed container total and per-record limit of
// one item for one container, and whether both are established.
//
// The derivation is the one AddItemToInventory and SetOwnedItemQuantity use: a
// quantity stack is limited by min(maxPerStack, container total), and a
// separate-instance item occupies one record per instance. An unknown record
// mode, an unknown stack capability and an unknown container limit leave the
// item unjudged instead of being read as the limit zero.
func containerLimits(item *schema.ItemDocument, container string) (total uint32, perRecord uint32, known bool) {
	storage := item.Storage
	switch container {
	case saveengine.ValidationScopeInventory:
		if !storage.MaxInventory.Known {
			return 0, 0, false
		}
		total = storage.MaxInventory.Value
	case saveengine.ValidationScopeStorage:
		if !storage.MaxStorage.Known {
			return 0, 0, false
		}
		total = storage.MaxStorage.Value
	default:
		return 0, 0, false
	}
	if total == 0 {
		return 0, 0, true
	}

	if !storage.RecordMode.Known {
		return 0, 0, false
	}
	switch storage.RecordMode.Value {
	case schema.RecordModeSeparateInstances:
		return total, 1, true
	case schema.RecordModeQuantityStack:
		stack := item.Capabilities.Stack
		if !stack.Known || !stack.Enabled || stack.Rules == nil || stack.Rules.MaxPerStack == 0 {
			return 0, 0, false
		}
		return total, min(stack.Rules.MaxPerStack, total), true
	default:
		return 0, 0, false
	}
}

// checkStats judges the statistics block against the values SaveEngine derived
// from it with the confirmed rules.
func checkStats(stats saveengine.ValidationStats) []SaveValidationIssue {
	issues := make([]SaveValidationIssue, 0)
	if stats.LevelError != "" {
		// Without a legal attribute set there is no expected level to compare
		// the stored one with, so the range violation is the only finding.
		return append(issues, SaveValidationIssue{
			Code:     IssueCodeAttributeOutOfRange,
			Severity: SaveValidationSeverityError,
			Scope:    saveengine.ValidationScopeStats,
			Message:  stats.LevelError,
		})
	}
	if stats.StoredLevel != stats.ExpectedLevel {
		issues = append(issues, SaveValidationIssue{
			Code:     IssueCodeLevelMismatch,
			Severity: SaveValidationSeverityError,
			Scope:    saveengine.ValidationScopeStats,
			Message: fmt.Sprintf("the stored level %d does not match the level %d its attributes produce",
				stats.StoredLevel, stats.ExpectedLevel),
		})
	}
	if stats.ClassMinimumError != "" {
		issues = append(issues, SaveValidationIssue{
			Code:     IssueCodeAttributeBelowClassMin,
			Severity: SaveValidationSeverityError,
			Scope:    saveengine.ValidationScopeStats,
			Message:  stats.ClassMinimumError,
		})
	}
	if stats.StoredSoulMemory < stats.MinimumSoulMemory {
		issues = append(issues, SaveValidationIssue{
			Code:     IssueCodeSoulMemoryBelowMinimum,
			Severity: SaveValidationSeverityError,
			Scope:    saveengine.ValidationScopeStats,
			Message: fmt.Sprintf("the stored lifetime runes %d are below the %d level %d requires",
				stats.StoredSoulMemory, stats.MinimumSoulMemory, stats.ExpectedLevel),
		})
	}
	return issues
}

// checkSpells judges the fourteen physical spell records: the two positions the
// game keeps reserved must be empty, every occupied identifier must resolve to a
// spell, and the memory those spells consume must fit the capacity SaveEngine
// derived from the slot.
//
// An identifier that does not resolve is a warning and leaves its cost out of
// the sum, so an unknown spell can never create a capacity error of its own.
func checkSpells(
	gameCatalog *gamecatalog.Catalog,
	facts saveengine.SaveValidationFacts,
) (SaveValidationScopeCoverage, []SaveValidationIssue) {
	const (
		publicSpellPositions = 12
		emptySpellID         = uint32(0xFFFFFFFF)
	)

	coverage := SaveValidationScopeCoverage{Scope: saveengine.ValidationScopeSpells, Checked: true}
	issues := make([]SaveValidationIssue, 0)
	used := 0
	resolvedAll := true

	for index, raw := range facts.Spells {
		if raw == emptySpellID {
			continue
		}
		coverage.RecordsChecked++
		if index >= publicSpellPositions {
			issues = append(issues, SaveValidationIssue{
				Code:     IssueCodeReservedSpellPosition,
				Severity: SaveValidationSeverityError,
				Scope:    saveengine.ValidationScopeSpells,
				Message: fmt.Sprintf(
					"spell position %d is reserved and must be empty, but stores 0x%08X", index+1, raw),
			})
			continue
		}

		cost, err := spellMemoryCost(gameCatalog, raw)
		if err != nil {
			resolvedAll = false
			coverage.UnresolvedRecords++
			issues = append(issues, SaveValidationIssue{
				Code:     IssueCodeUnresolvedSpell,
				Severity: SaveValidationSeverityWarning,
				Scope:    saveengine.ValidationScopeSpells,
				Message:  fmt.Sprintf("spell position %d: %v", index+1, err),
			})
			continue
		}
		used += cost
	}

	if resolvedAll && used > facts.AvailableMemorySlots {
		issues = append(issues, SaveValidationIssue{
			Code:     IssueCodeMemorySlotsExceeded,
			Severity: SaveValidationSeverityError,
			Scope:    saveengine.ValidationScopeSpells,
			Message: fmt.Sprintf("the equipped spells use %d memory slots, but the character has %d",
				used, facts.AvailableMemorySlots),
		})
	}
	return coverage, issues
}

// spellMemoryCost resolves one raw MagicParam ID to the Memory Slots the spell
// consumes, through the same GameCatalog rule SetEquippedSpells validates
// against.
func spellMemoryCost(gameCatalog *gamecatalog.Catalog, raw uint32) (int, error) {
	if raw == 0 || raw >= gamecatalog.EquippedSpellRawIDLimit {
		return 0, fmt.Errorf("0x%08X is not a raw MagicParam ID", raw)
	}
	gameID := gamecatalog.EquippedSpellGameIDPrefix | raw
	resource, exists := gameCatalog.ItemByGameID(gameID)
	if !exists {
		return 0, fmt.Errorf("game ID 0x%08X is not a known item", gameID)
	}
	_, cost, err := gamecatalog.ValidateSpellResource(resource)
	if err != nil {
		return 0, err
	}
	return cost, nil
}
