package saveengine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// OperationRisk is the closed backend vocabulary shown by Review Changes.
// Frontends display it but never infer or upgrade it locally.
type OperationRisk string

const (
	OperationRiskNormal   OperationRisk = "normal"
	OperationRiskWarning  OperationRisk = "warning"
	OperationRiskBanRisk  OperationRisk = "ban risk"
	OperationRiskCritical OperationRisk = "critical"
)

// OperationRecord is the safe, public description of one logical mutation
// currently applied on top of the session baseline. Replay bytes never leave
// SaveEngine; Review Changes receives only this projection.
type OperationRecord struct {
	OperationID      string        `json:"operationID"`
	OperationKind    string        `json:"operationKind"`
	SaveSessionID    string        `json:"saveSessionID"`
	SaveRevision     string        `json:"saveRevision"`
	Order            string        `json:"order"`
	CharacterID      *int          `json:"characterID,omitempty"`
	Area             string        `json:"area"`
	Description      string        `json:"description"`
	RelatedResource  string        `json:"relatedResource,omitempty"`
	BeforeState      string        `json:"beforeState"`
	AfterState       string        `json:"afterState"`
	Risk             OperationRisk `json:"risk"`
	RiskReason       string        `json:"riskReason"`
	ChangedByteCount int           `json:"changedByteCount"`
	ChangedScopes    []string      `json:"changedScopes"`
}

// OperationHistory is the authoritative Review Changes snapshot for one save
// revision. Its arrays are independent copies and cannot mutate the session.
type OperationHistory struct {
	SaveSessionID string            `json:"saveSessionID"`
	SaveRevision  string            `json:"saveRevision"`
	Operations    []OperationRecord `json:"operations"`
	UndoCount     int               `json:"undoCount"`
	RedoCount     int               `json:"redoCount"`
}

// bytePatch is the replayable delta of one contiguous changed range. Forward
// replay requires Before and writes After; reverse replay requires After and
// writes Before. Matching the preimage first is the dependency check that makes
// selective revert fail closed rather than overwriting a later operation.
type bytePatch struct {
	Offset int    `json:"offset"`
	Before []byte `json:"before"`
	After  []byte `json:"after"`
}

type operationEntry struct {
	Record  OperationRecord `json:"record"`
	Patches []bytePatch     `json:"patches"`
}

func fingerprintBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func buildBytePatches(before []byte, after []byte) ([]bytePatch, error) {
	if len(before) != len(after) {
		return nil, fmt.Errorf("snapshot length changed from %d to %d", len(before), len(after))
	}
	patches := make([]bytePatch, 0)
	for index := 0; index < len(before); {
		if before[index] == after[index] {
			index++
			continue
		}
		start := index
		for index < len(before) && before[index] != after[index] {
			index++
		}
		patches = append(patches, bytePatch{
			Offset: start,
			Before: append([]byte(nil), before[start:index]...),
			After:  append([]byte(nil), after[start:index]...),
		})
	}
	return patches, nil
}

func changedByteCount(patches []bytePatch) int {
	count := 0
	for _, patch := range patches {
		count += len(patch.After)
	}
	return count
}

func applyPatches(data []byte, patches []bytePatch, forward bool) error {
	for patchIndex, patch := range patches {
		expected, replacement := patch.Before, patch.After
		if !forward {
			expected, replacement = patch.After, patch.Before
		}
		if patch.Offset < 0 || patch.Offset > len(data) || len(expected) > len(data)-patch.Offset {
			return fmt.Errorf("operation patch %d is outside the snapshot", patchIndex)
		}
		end := patch.Offset + len(expected)
		if !bytes.Equal(data[patch.Offset:end], expected) {
			return fmt.Errorf("operation patch %d no longer matches its required state", patchIndex)
		}
		if len(expected) != len(replacement) {
			return fmt.Errorf("operation patch %d changes the snapshot length", patchIndex)
		}
		copy(data[patch.Offset:end], replacement)
	}
	return nil
}

func cloneOperationEntry(entry operationEntry) operationEntry {
	cloned := operationEntry{Record: cloneOperationRecord(entry.Record)}
	cloned.Patches = make([]bytePatch, len(entry.Patches))
	for index, patch := range entry.Patches {
		cloned.Patches[index] = bytePatch{
			Offset: patch.Offset,
			Before: append([]byte(nil), patch.Before...),
			After:  append([]byte(nil), patch.After...),
		}
	}
	return cloned
}

func cloneOperationRecord(record OperationRecord) OperationRecord {
	cloned := record
	cloned.ChangedScopes = append([]string(nil), record.ChangedScopes...)
	if record.CharacterID != nil {
		characterID := *record.CharacterID
		cloned.CharacterID = &characterID
	}
	return cloned
}

// executionBanRiskReason is the one reason text of a risk that belongs to a
// concrete execution rather than to its kind. It lives here, beside the kind
// table, so the history contract stays the single place a risk is named.
const executionBanRiskReason = "This operation writes a resource the GameCatalog marks as a ban risk."

func operationEntryForCommit(
	receipt MutationReceipt,
	order int,
	characterScoped bool,
	characterID int,
	banRisk bool,
	before []byte,
	after []byte,
) (operationEntry, error) {
	patches, err := buildBytePatches(before, after)
	if err != nil {
		return operationEntry{}, err
	}
	area, risk, riskReason := operationPresentation(receipt.OperationKind, receipt.ChangedScopes)
	// The kind states the risk every execution of it carries; banRisk is what the
	// backend derived from authoritative GameCatalog data about this one
	// execution. A ban risk outranks a warning and is outranked by a critical, so
	// the elevated value replaces the baseline in exactly that case.
	if banRisk && risk != OperationRiskCritical {
		risk, riskReason = OperationRiskBanRisk, executionBanRiskReason
	}
	var character *int
	if characterScoped {
		value := characterID
		character = &value
	}
	return operationEntry{
		Record: OperationRecord{
			OperationID:      receipt.OperationID,
			OperationKind:    receipt.OperationKind,
			SaveSessionID:    receipt.SaveSessionID,
			SaveRevision:     receipt.SaveRevision,
			Order:            strconv.Itoa(order),
			CharacterID:      character,
			Area:             area,
			Description:      humanizeOperationKind(receipt.OperationKind),
			BeforeState:      "Revision " + previousRevision(receipt.SaveRevision),
			AfterState:       "Revision " + receipt.SaveRevision,
			Risk:             risk,
			RiskReason:       riskReason,
			ChangedByteCount: changedByteCount(patches),
			ChangedScopes:    append([]string(nil), receipt.ChangedScopes...),
		},
		Patches: patches,
	}, nil
}

func previousRevision(revision string) string {
	value, err := strconv.ParseUint(revision, 10, 64)
	if err != nil || value == 0 {
		return revision
	}
	return strconv.FormatUint(value-1, 10)
}

func humanizeOperationKind(kind string) string {
	parts := strings.Split(kind, "_")
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func operationPresentation(kind string, scopes []string) (string, OperationRisk, string) {
	area := "Session"
	for _, scope := range scopes {
		switch scope {
		case ScopeCharacterStats:
			area = "Stats"
		case ScopeCharacterProfile, ScopeCharacterAppearance, ScopeCharacterList:
			if area == "Session" {
				area = "Profile"
			}
		case ScopeInventory:
			area = "Inventory"
		case ScopeStorage:
			if area == "Session" {
				area = "Storage"
			}
		case ScopeEquipmentLoadout:
			if area == "Session" || area == "Profile" {
				area = "Equipment"
			}
		case ScopeWorldFlags:
			area = "World"
		case ScopeNetwork:
			area = "Network"
		case ScopeFavorites:
			area = "Favorites"
		}
	}

	switch kind {
	case kindSetSaveAccountID:
		return area, OperationRiskBanRisk,
			"Changing the save owner identifier can affect account compatibility."
	case kindApplyRepairs:
		return area, OperationRiskWarning,
			"Repairs can remove or rewrite records that failed validation."
	case kindSetNetworkSettings, kindApplyNetworkPreset:
		return area, OperationRiskWarning,
			"Non-default network values can change online matchmaking behavior."
	case kindSetBellBearingUnlocked, kindSetBossDefeated, kindSetColosseumUnlocked,
		kindSetCookbookUnlocked, kindSetFogOfWarRemoved, kindSetGestureUnlocked,
		kindSetGraceVisited, kindSetMapRegionRevealed, kindSetQuestStep,
		kindSetRegionUnlocked, kindSetSpectralSteedAttire,
		kindLockAllSpectralSteedAttires, kindSetSummoningPoolActivated,
		kindSetTutorialUnlocked, kindSetWhetbladeUnlocked:
		return area, OperationRiskWarning,
			"This operation changes world progression state."
	default:
		return area, OperationRiskNormal, "The backend accepted this ordinary operation."
	}
}

// GetOperationHistory returns the authoritative ordered history of one session.
func (engine *Engine) GetOperationHistory(saveSessionID string) (OperationHistory, error) {
	if saveSessionID == "" {
		return OperationHistory{}, apperror.MissingField("saveSessionID")
	}
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return OperationHistory{}, apperror.UnknownSaveSession(saveSessionID)
	}
	records := make([]OperationRecord, len(loaded.operations))
	for index, entry := range loaded.operations {
		records[index] = cloneOperationRecord(entry.Record)
	}
	return OperationHistory{
		SaveSessionID: saveSessionID,
		SaveRevision:  loaded.session.revisionString(),
		Operations:    records,
		UndoCount:     len(loaded.operations),
		RedoCount:     len(loaded.redo),
	}, nil
}

// HistoryMutationResult is returned by global Undo, Redo and selective revert.
type HistoryMutationResult struct {
	MutationReceipt
	AffectedOperationID   string `json:"affectedOperationID"`
	AffectedOperationKind string `json:"affectedOperationKind"`
}

func (engine *Engine) UndoLastOperation(
	saveSessionID string,
	expectedRevision string,
) (HistoryMutationResult, error) {
	return engine.changeHistory(saveSessionID, expectedRevision, kindUndoLastOperation, "")
}

func (engine *Engine) RedoLastOperation(
	saveSessionID string,
	expectedRevision string,
) (HistoryMutationResult, error) {
	return engine.changeHistory(saveSessionID, expectedRevision, kindRedoLastOperation, "")
}

func (engine *Engine) RevertOperation(
	saveSessionID string,
	operationID string,
	expectedRevision string,
) (HistoryMutationResult, error) {
	if operationID == "" {
		return HistoryMutationResult{}, apperror.MissingField("operationID")
	}
	return engine.changeHistory(saveSessionID, expectedRevision, kindRevertOperation, operationID)
}

func (engine *Engine) changeHistory(
	saveSessionID string,
	expectedRevision string,
	kind string,
	operationID string,
) (HistoryMutationResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return HistoryMutationResult{}, apperror.InvalidRevision(expectedRevision)
	}
	if saveSessionID == "" {
		return HistoryMutationResult{}, apperror.MissingField("saveSessionID")
	}

	defer engine.publishSessionChanged()
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return HistoryMutationResult{}, apperror.UnknownSaveSession(saveSessionID)
	}
	if expectedRevision != loaded.session.revisionString() {
		return HistoryMutationResult{}, apperror.RevisionConflict(
			expectedRevision, loaded.session.revisionString())
	}

	var affected operationEntry
	nextOperations := append([]operationEntry(nil), loaded.operations...)
	nextRedo := append([]operationEntry(nil), loaded.redo...)
	candidate := append([]byte(nil), loaded.snapshot.data...)

	switch kind {
	case kindUndoLastOperation:
		if len(nextOperations) == 0 {
			return HistoryMutationResult{}, errors.New("no operation is available to undo")
		}
		affected = nextOperations[len(nextOperations)-1]
		if err := applyPatches(candidate, affected.Patches, false); err != nil {
			return HistoryMutationResult{}, fmt.Errorf("cannot undo operation %q: %w",
				affected.Record.OperationID, err)
		}
		nextOperations = nextOperations[:len(nextOperations)-1]
		nextRedo = append(nextRedo, affected)
	case kindRedoLastOperation:
		if len(nextRedo) == 0 {
			return HistoryMutationResult{}, errors.New("no operation is available to redo")
		}
		affected = nextRedo[len(nextRedo)-1]
		if err := applyPatches(candidate, affected.Patches, true); err != nil {
			return HistoryMutationResult{}, fmt.Errorf("cannot redo operation %q: %w",
				affected.Record.OperationID, err)
		}
		nextRedo = nextRedo[:len(nextRedo)-1]
		nextOperations = append(nextOperations, affected)
	case kindRevertOperation:
		targetIndex := -1
		for index, entry := range nextOperations {
			if entry.Record.OperationID == operationID {
				targetIndex = index
				affected = entry
				break
			}
		}
		if targetIndex < 0 {
			return HistoryMutationResult{}, fmt.Errorf("unknown operationID %q", operationID)
		}
		candidate = append([]byte(nil), loaded.baseline.data...)
		replayed := make([]operationEntry, 0, len(nextOperations)-1)
		for index, entry := range nextOperations {
			if index == targetIndex {
				continue
			}
			if err := applyPatches(candidate, entry.Patches, true); err != nil {
				return HistoryMutationResult{}, fmt.Errorf(
					"cannot revert operation %q because later operation %q depends on it: %w",
					operationID, entry.Record.OperationID, err)
			}
			replayed = append(replayed, entry)
		}
		nextOperations = replayed
		nextRedo = nil
	default:
		return HistoryMutationResult{}, fmt.Errorf("unsupported history mutation %q", kind)
	}

	if err := validateSnapshotCandidate(loaded, candidate); err != nil {
		return HistoryMutationResult{}, fmt.Errorf("history replay failed validation: %w", err)
	}
	pending, err := engine.prepareMutation(kind, affected.Record.ChangedScopes...)
	if err != nil {
		return HistoryMutationResult{}, err
	}
	nextRevision := strconv.FormatUint(loaded.session.revision+1, 10)
	receipt := pending.receipt(saveSessionID, nextRevision)
	if err := engine.persistRecoveryState(loaded, nextOperations, nextRevision); err != nil {
		return HistoryMutationResult{}, err
	}

	loaded.snapshot = &codec{data: candidate}
	loaded.operations = nextOperations
	loaded.redo = nextRedo
	loaded.session.undo = nil
	loaded.session.reviewAuthorization = nil
	loaded.session.dirty = len(nextOperations) > 0
	actualRevision := loaded.session.advanceRevision()
	if actualRevision != nextRevision {
		return HistoryMutationResult{}, errors.New("history revision changed unexpectedly")
	}
	if len(nextOperations) == 0 {
		loaded.baselineRevision = nextRevision
	}
	engine.enqueueCommitted(loaded.session, receipt)
	return HistoryMutationResult{
		MutationReceipt:       receipt,
		AffectedOperationID:   affected.Record.OperationID,
		AffectedOperationKind: affected.Record.OperationKind,
	}, nil
}

// DiscardChangesResult reports the new revision created by restoring baseline.
type DiscardChangesResult struct {
	MutationReceipt
	DiscardedOperations int `json:"discardedOperations"`
}

func (engine *Engine) DiscardChanges(
	saveSessionID string,
	expectedRevision string,
) (DiscardChangesResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return DiscardChangesResult{}, apperror.InvalidRevision(expectedRevision)
	}
	if saveSessionID == "" {
		return DiscardChangesResult{}, apperror.MissingField("saveSessionID")
	}
	defer engine.publishSessionChanged()
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return DiscardChangesResult{}, apperror.UnknownSaveSession(saveSessionID)
	}
	if expectedRevision != loaded.session.revisionString() {
		return DiscardChangesResult{}, apperror.RevisionConflict(
			expectedRevision, loaded.session.revisionString())
	}
	if !loaded.session.dirty || len(loaded.operations) == 0 {
		return DiscardChangesResult{}, errors.New("the session has no changes to discard")
	}

	extraScopes := unionOperationScopes(loaded.operations)
	pending, err := engine.prepareMutation(kindDiscardChanges, extraScopes...)
	if err != nil {
		return DiscardChangesResult{}, err
	}
	candidate := append([]byte(nil), loaded.baseline.data...)
	if err := validateSnapshotCandidate(loaded, candidate); err != nil {
		return DiscardChangesResult{}, fmt.Errorf("baseline failed validation: %w", err)
	}
	if err := engine.removeRecoveryJournal(loaded.session.id); err != nil {
		return DiscardChangesResult{}, err
	}

	discarded := len(loaded.operations)
	loaded.snapshot = &codec{data: candidate}
	loaded.operations = nil
	loaded.redo = nil
	loaded.session.undo = nil
	loaded.session.reviewAuthorization = nil
	loaded.session.dirty = false
	receipt := pending.receipt(saveSessionID, loaded.session.advanceRevision())
	loaded.baselineRevision = receipt.SaveRevision
	engine.enqueueCommitted(loaded.session, receipt)
	return DiscardChangesResult{MutationReceipt: receipt, DiscardedOperations: discarded}, nil
}

func unionOperationScopes(entries []operationEntry) []string {
	selected := make(map[string]bool, len(changedScopeOrder))
	for _, entry := range entries {
		for _, scope := range entry.Record.ChangedScopes {
			selected[scope] = true
		}
	}
	result := make([]string, 0, len(selected))
	for _, scope := range changedScopeOrder {
		if selected[scope] {
			result = append(result, scope)
		}
	}
	return result
}

func validateSnapshotCandidate(loaded *loadedSave, data []byte) error {
	temporary := &loadedSave{
		session:  &Session{platform: loaded.session.platform},
		snapshot: &codec{data: append([]byte(nil), data...)},
	}
	serialized, err := serializeContainer(temporary)
	if err != nil {
		return err
	}
	return validateSerialized(serialized, loaded.session.platform)
}
