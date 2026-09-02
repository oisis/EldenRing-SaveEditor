package saveengine

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// Repair operation names are shared with the diagnostics endpoint. They name
// only existing SaveEngine mutations; a repair plan never creates a new writer.
const (
	RepairOperationSetOwnedItemQuantity = "set_owned_item_quantity"
	RepairOperationRemoveOwnedItem      = "remove_owned_item"
	RepairOperationSetCharacterStats    = "set_character_stats"
)

// RepairAction is the executable projection of a diagnostics repair plan. It
// is internal input from the endpoint after that endpoint has re-derived the
// plan from GameCatalog and verified its token; no transport caller supplies it.
type RepairAction struct {
	Operation   string
	OwnedItemID string
	TargetValue uint32
	Attributes  *CharacterAttributes
}

// ApplyRepairPlanResult is the SaveEngine receipt of one repair execution.
//
// Applied is the discriminator of the two success variants:
//
//   - true is one committed, atomic transaction. The embedded receipt is the
//     complete one the central commit path produced, so its operationKind is
//     always apply_repairs and its changedScopes come from the one central map.
//   - false is a verified selection whose freshly derived plan has no executable
//     action. It commits nothing, so no operationID is minted, the revision, the
//     snapshot, the unsaved-changes flag and the undo point stay as they were, no
//     session event is published, and the three execution members of the receipt
//     are absent.
type ApplyRepairPlanResult struct {
	MutationReceipt
	CharacterID int  `json:"characterID"`
	Applied     bool `json:"applied"`
}

// MarshalJSON isolates the no-action wire variant from the shared, strict
// MutationReceipt used by every committed mutation.
func (result ApplyRepairPlanResult) MarshalJSON() ([]byte, error) {
	if result.Applied {
		type appliedResult ApplyRepairPlanResult
		return json.Marshal(appliedResult(result))
	}
	return json.Marshal(struct {
		SaveSessionID string `json:"saveSessionID"`
		SaveRevision  string `json:"saveRevision"`
		CharacterID   int    `json:"characterID"`
		Applied       bool   `json:"applied"`
	}{
		SaveSessionID: result.SaveSessionID,
		SaveRevision:  result.SaveRevision,
		CharacterID:   result.CharacterID,
		Applied:       result.Applied,
	})
}

// ApplyRepairPlan applies the already re-derived executable actions as one
// character transaction. It does not decide whether an item is repairable or
// derive catalog limits: those decisions belong to diagnostics/GetRepairPlan.
// It validates every physical target again under the commit lock, prepares all
// byte writes before the first write, and commits them together.
func (engine *Engine) ApplyRepairPlan(
	saveSessionID string,
	characterID int,
	actions []RepairAction,
	expectedRevision string,
) (ApplyRepairPlanResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return ApplyRepairPlanResult{}, apperror.InvalidRevision(expectedRevision)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return ApplyRepairPlanResult{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}
	if len(actions) == 0 {
		return engine.confirmNoRepairActions(saveSessionID, characterID, expectedRevision)
	}

	committed, err := engine.commitCharacterRevisionWithHook(
		saveSessionID,
		kindApplyRepairs,
		characterID,
		func(loaded *loadedSave) error {
			if expectedRevision != loaded.session.revisionString() {
				return apperror.RevisionConflict(
					expectedRevision, loaded.session.revisionString())

			}
			if err := requireActiveCharacter(loaded, characterID); err != nil {
				return err
			}

			writes, err := planRepairWrites(loaded, characterID, actions)
			if err != nil {
				return err
			}
			return applyByteWrites(loaded.snapshot, writes)
		},
		func(loaded *loadedSave, rev string) {
			slot := characterID
			loaded.session.appendDiagnosticRecord(
				engine.nowUTC(),
				DiagnosticScopeRepairs,
				DiagnosticSeverityInfo,
				DiagnosticEventRepairsApplied,
				DiagnosticMessageRepairsApplied,
				&slot,
				rev,
			)
		},
	)
	if err != nil {
		return ApplyRepairPlanResult{}, err
	}

	return ApplyRepairPlanResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		Applied:         true,
	}, nil
}

func (engine *Engine) confirmNoRepairActions(
	saveSessionID string,
	characterID int,
	expectedRevision string,
) (ApplyRepairPlanResult, error) {
	if saveSessionID == "" {
		return ApplyRepairPlanResult{}, apperror.MissingField("saveSessionID")
	}
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return ApplyRepairPlanResult{}, apperror.UnknownSaveSession(saveSessionID)
	}
	if expectedRevision != loaded.session.revisionString() {
		return ApplyRepairPlanResult{}, apperror.RevisionConflict(
			expectedRevision, loaded.session.revisionString())

	}
	if err := requireActiveCharacter(loaded, characterID); err != nil {
		return ApplyRepairPlanResult{}, err
	}
	return ApplyRepairPlanResult{
		MutationReceipt: noCommitReceipt(saveSessionID, expectedRevision),
		CharacterID:     characterID,
		Applied:         false,
	}, nil
}

func requireActiveCharacter(loaded *loadedSave, characterID int) error {
	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}
	if flag[0] != userData10ActiveFlagValue {
		return fmt.Errorf("character %d is not active", characterID)
	}
	return nil
}

type repairCountWrite struct {
	count   uint32
	removed uint32
}

func planRepairWrites(loaded *loadedSave, characterID int, actions []RepairAction) ([]byteWrite, error) {
	writes := make([]byteWrite, 0, len(actions)*2)
	removalCounts := make(map[int64]repairCountWrite)
	owned := make(map[string]struct{})
	statsPlanned := false

	for index, action := range actions {
		switch action.Operation {
		case RepairOperationRemoveOwnedItem:
			if action.OwnedItemID == "" {
				return nil, fmt.Errorf("repair action %d: remove_owned_item requires ownedItemID", index)
			}
			if _, duplicate := owned[action.OwnedItemID]; duplicate {
				return nil, fmt.Errorf("repair plan addresses ownedItemID %q more than once", action.OwnedItemID)
			}
			owned[action.OwnedItemID] = struct{}{}
			locator, err := loaded.session.resolveOwnedItemID(characterID, action.OwnedItemID)
			if err != nil {
				return nil, err
			}
			_, removal, err := planOwnedItemRemovalWrite(loaded, locator, action.OwnedItemID)
			if err != nil {
				return nil, err
			}
			writes = append(writes, removal.record)
			if removal.countAt >= 0 && removal.count > 0 {
				group := removalCounts[removal.countAt]
				if group.removed != 0 && group.count != removal.count {
					return nil, fmt.Errorf("repair plan reads inconsistent record counts at offset %d", removal.countAt)
				}
				group.count = removal.count
				group.removed++
				if group.removed > group.count {
					return nil, fmt.Errorf("repair plan removes %d records from a section whose count is %d", group.removed, group.count)
				}
				removalCounts[removal.countAt] = group
			}

		case RepairOperationSetOwnedItemQuantity:
			if action.OwnedItemID == "" {
				return nil, fmt.Errorf("repair action %d: set_owned_item_quantity requires ownedItemID", index)
			}
			if _, duplicate := owned[action.OwnedItemID]; duplicate {
				return nil, fmt.Errorf("repair plan addresses ownedItemID %q more than once", action.OwnedItemID)
			}
			owned[action.OwnedItemID] = struct{}{}
			locator, err := loaded.session.resolveOwnedItemID(characterID, action.OwnedItemID)
			if err != nil {
				return nil, err
			}
			write, err := planOwnedItemQuantityWrite(loaded, locator, action.OwnedItemID, action.TargetValue, nil, nil)
			if err != nil {
				return nil, err
			}
			writes = append(writes, write)

		case RepairOperationSetCharacterStats:
			if statsPlanned || action.Attributes == nil {
				return nil, fmt.Errorf("repair action %d: set_character_stats requires exactly one attributes value", index)
			}
			statsPlanned = true
			values, level, err := prepareCharacterAttributes(*action.Attributes)
			if err != nil {
				return nil, err
			}
			ctx, err := planCharacterStatsState(loaded, characterID, values, level)
			if err != nil {
				return nil, err
			}
			blockAfter := bytes.Clone(ctx.blockBefore)
			for attributeIndex, value := range values {
				binary.LittleEndian.PutUint32(blockAfter[attributeIndex*4:], value)
			}
			binary.LittleEndian.PutUint32(blockAfter[statsBlockLevelPosition:], level)
			binary.LittleEndian.PutUint32(blockAfter[statsBlockTotalGetSoulPosition:], ctx.soulMemory)
			summaryLevelAfter := make([]byte, summaryLevelSize)
			binary.LittleEndian.PutUint32(summaryLevelAfter, level)
			writes = append(writes,
				byteWrite{at: ctx.blockAt, data: blockAfter},
				byteWrite{at: ctx.summaryLevelAt, data: summaryLevelAfter},
			)

		default:
			return nil, fmt.Errorf("repair action %d names unsupported operation %q", index, action.Operation)
		}
	}

	countOffsets := make([]int64, 0, len(removalCounts))
	for at := range removalCounts {
		countOffsets = append(countOffsets, at)
	}
	sort.Slice(countOffsets, func(left, right int) bool { return countOffsets[left] < countOffsets[right] })
	for _, at := range countOffsets {
		group := removalCounts[at]
		writes = append(writes, byteWrite{at: at, data: littleEndianUint32(group.count - group.removed)})
	}
	if err := requireNonOverlappingRepairWrites(writes); err != nil {
		return nil, err
	}
	return writes, nil
}

func requireNonOverlappingRepairWrites(writes []byteWrite) error {
	ordered := append([]byteWrite(nil), writes...)
	sort.Slice(ordered, func(left, right int) bool { return ordered[left].at < ordered[right].at })
	for index := 1; index < len(ordered); index++ {
		previous := ordered[index-1]
		current := ordered[index]
		if previous.at+int64(len(previous.data)) > current.at {
			return fmt.Errorf("repair plan contains overlapping writes at offsets %d and %d", previous.at, current.at)
		}
	}
	return nil
}
