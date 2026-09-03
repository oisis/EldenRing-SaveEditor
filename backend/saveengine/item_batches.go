package saveengine

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// This file holds the five atomic batch mutations of the Items workspace: one
// batch add, the two batch moves, one batch removal and the anchored group
// reorder of Inventory.
//
// Every one of them shares the same shape and the same guarantee:
//
//   - the complete request is validated before the session is touched;
//   - one expectedRevision covers the whole batch;
//   - the steps are applied to a private candidate image of the snapshot, never
//     to the session's own snapshot, so a step that fails leaves nothing behind
//     and no rollback of a partial write is needed;
//   - the candidate replaces the session snapshot only after every step
//     succeeded, which is what makes the batch one revision, one history entry
//     and one receipt;
//   - the writers, planners and validators of the single-record mutations are
//     reused as they are. No limit, no reference rule and no binary layout is
//     restated here.
//
// ponytail: a candidate byte image is the whole transaction mechanism. The
// snapshot is one contiguous buffer that the central commit path already copies
// for the operation history, so a second copy per batch costs one allocation and
// removes every partial-write path there could be.

// candidate returns a private copy of the session image the batch steps write
// into. Everything except the snapshot is shared, because a batch changes only
// save bytes: the session, the baseline and the operation history stay owned by
// the central commit path.
//
// The caller holds Engine.mutex.
func (loaded *loadedSave) candidate() *loadedSave {
	image := *loaded
	image.snapshot = &codec{data: append([]byte(nil), loaded.snapshot.data...)}
	return &image
}

// ItemAddition is one requested addition of a batch. The two limits, the record
// mode and the ban-risk fact are catalog decisions the endpoint resolved;
// SaveEngine never derives, defaults or widens them.
//
// BanRisk is item.safety.banRisk of the resolved ItemDocument. It reaches the
// operation history through the central commit path, so a batch that writes at
// least one such resource is recorded as a ban-risk operation and Review
// Changes demands its own confirmation before the session may be saved. The
// confirmation the Add itself required is a separate, earlier gate and does not
// stand in for it.
type ItemAddition struct {
	GameID            uint32
	Quantity          uint32
	SeparateInstances bool
	MaxPerRecord      uint32
	MaxContainerTotal uint32
	BanRisk           bool
}

// AddedItem reports one committed addition of a batch.
type AddedItem struct {
	GameID           uint32 `json:"gameID"`
	Container        string `json:"container"`
	ContainerSection string `json:"containerSection"`
	Added            uint32 `json:"added"`
	Quantity         uint32 `json:"quantity"`
	CreatedRecord    bool   `json:"createdRecord"`
	PhysicalIndex    int    `json:"physicalIndex"`
}

// AddItemsToContainersResult reports one committed batch add.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their usual JSON names and the batch
// members join them flat.
type AddItemsToContainersResult struct {
	MutationReceipt
	CharacterID int         `json:"characterID"`
	Added       []AddedItem `json:"added"`
}

// AddItemsToContainers adds every requested item to common Inventory, to common
// Storage, or to both, as one atomic mutation.
//
// The two lists are independent requests for two containers, not one list with a
// destination switch: a caller states what it wants in each container, and the
// changed scopes of the receipt name exactly the containers it actually used.
func (engine *Engine) AddItemsToContainers(
	saveSessionID string,
	characterID int,
	inventoryItems []ItemAddition,
	storageItems []ItemAddition,
	expectedRevision string,
) (AddItemsToContainersResult, error) {
	if len(inventoryItems) == 0 && len(storageItems) == 0 {
		return AddItemsToContainersResult{}, fmt.Errorf(
			"a batch add must request at least one item")
	}
	if err := validateItemAdditions("inventoryItems", inventoryItems); err != nil {
		return AddItemsToContainersResult{}, err
	}
	if err := validateItemAdditions("storageItems", storageItems); err != nil {
		return AddItemsToContainersResult{}, err
	}
	if !isCanonicalRevision(expectedRevision) {
		return AddItemsToContainersResult{}, apperror.InvalidRevision(expectedRevision)
	}

	// The containers this batch writes are known from the request, so the exact
	// scopes are resolved before the mutation runs and never assembled by hand.
	extraScopes := make([]string, 0, 2)
	if len(inventoryItems) > 0 {
		extraScopes = append(extraScopes, ScopeInventory)
	}
	if len(storageItems) > 0 {
		extraScopes = append(extraScopes, ScopeStorage)
	}

	// The risk of this execution is the catalog fact the endpoint resolved, not a
	// property of the kind: an ordinary batch add stays normal and one that
	// writes a ban-risk resource is recorded as a ban risk.
	banRisk := false
	for _, addition := range inventoryItems {
		banRisk = banRisk || addition.BanRisk
	}
	for _, addition := range storageItems {
		banRisk = banRisk || addition.BanRisk
	}

	var added []AddedItem
	committed, err := engine.commitCharacterRevisionWithExecution(
		saveSessionID, kindAddItemsToContainers, characterID,
		execution{extraScopes: extraScopes, banRisk: banRisk},
		func(loaded *loadedSave) error {
			if err := requireActiveCharacterAt(loaded, characterID, expectedRevision); err != nil {
				return err
			}
			candidate := loaded.candidate()
			results := make([]AddedItem, 0, len(inventoryItems)+len(storageItems))
			for index, addition := range inventoryItems {
				outcome, err := addItemToInventoryRecord(
					candidate, characterID, addition.GameID, addition.Quantity,
					addition.SeparateInstances, addition.MaxPerRecord, addition.MaxContainerTotal)
				if err != nil {
					return fmt.Errorf("inventoryItems[%d]: %w", index, err)
				}
				results = append(results, AddedItem{
					GameID:           addition.GameID,
					Container:        ownedContainerInventory,
					ContainerSection: InventorySectionCommon,
					Added:            addition.Quantity,
					Quantity:         outcome.quantity,
					CreatedRecord:    outcome.created,
					PhysicalIndex:    outcome.physicalIndex,
				})
			}
			for index, addition := range storageItems {
				// The Storage writer knows one limit only: the container total.
				// A per-record limit is an Inventory concept and is deliberately
				// not smuggled in here.
				outcome, err := addItemToStorageRecord(
					candidate, characterID, addition.GameID, addition.Quantity,
					addition.SeparateInstances, addition.MaxContainerTotal)
				if err != nil {
					return fmt.Errorf("storageItems[%d]: %w", index, err)
				}
				results = append(results, AddedItem{
					GameID:           addition.GameID,
					Container:        ownedContainerStorage,
					ContainerSection: StorageSectionCommon,
					Added:            addition.Quantity,
					Quantity:         outcome.quantity,
					CreatedRecord:    outcome.created,
					PhysicalIndex:    outcome.physicalIndex,
				})
			}
			// Every step succeeded, so the complete image replaces the session
			// snapshot in one assignment. There is no point at which the session
			// holds a partially applied batch.
			loaded.snapshot = candidate.snapshot
			added = results
			return nil
		})
	if err != nil {
		return AddItemsToContainersResult{}, err
	}
	return AddItemsToContainersResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		Added:           added,
	}, nil
}

// validateItemAdditions rejects an empty, contradictory or ambiguous request
// before the session is touched. Two entries for the same item in one container
// are rejected rather than summed: the caller states one addition per item, and
// silently merging two would hide which of them exceeded a limit.
func validateItemAdditions(field string, additions []ItemAddition) error {
	seen := make(map[uint32]int, len(additions))
	for index, addition := range additions {
		if previous, duplicate := seen[addition.GameID]; duplicate {
			return fmt.Errorf(
				"%s repeats item 0x%08X at positions %d and %d",
				field, addition.GameID, previous, index)
		}
		seen[addition.GameID] = index
		if addition.Quantity == 0 {
			return fmt.Errorf(
				"%s[%d]: quantity must be at least 1; it is the amount added, not a target total",
				field, index)
		}
		if addition.Quantity > ^ownedItemQuantityFlag {
			return fmt.Errorf("%s[%d]: quantity %d exceeds the %d the record can store",
				field, index, addition.Quantity, ^ownedItemQuantityFlag)
		}
		if addition.SeparateInstances && addition.Quantity != 1 {
			return fmt.Errorf(
				"%s[%d]: item 0x%08X stores every copy in its own record, so quantity must be 1; got %d",
				field, index, addition.GameID, addition.Quantity)
		}
		if addition.MaxPerRecord == 0 || addition.MaxContainerTotal == 0 {
			return fmt.Errorf(
				"%s[%d]: maxPerRecord and maxContainerTotal must both be at least 1; got %d and %d",
				field, index, addition.MaxPerRecord, addition.MaxContainerTotal)
		}
		if addition.Quantity > addition.MaxPerRecord {
			return fmt.Errorf("%s[%d]: quantity %d exceeds the limit of %d per record",
				field, index, addition.Quantity, addition.MaxPerRecord)
		}
	}
	return nil
}

// OwnedItemMove is one requested move of a batch. ExpectedGameID is the item the
// caller believes the record denotes; the writer refuses the move when the
// record turns out to denote another one.
type OwnedItemMove struct {
	OwnedItemID       string
	ExpectedGameID    uint32
	MaxQuantity       uint32
	SeparateInstances bool
}

// MovedItem reports one committed move of a batch. OwnedItemID is the now-stale
// source identity; the destination receives a new identity on the next read.
type MovedItem struct {
	OwnedItemID      string `json:"ownedItemID"`
	GameID           uint32 `json:"gameID"`
	Quantity         uint32 `json:"quantity"`
	ContainerSection string `json:"containerSection"`
	PhysicalIndex    int    `json:"physicalIndex"`
	AcquisitionIndex uint32 `json:"acquisitionIndex"`
}

// MoveOwnedItemsResult reports one committed batch move.
type MoveOwnedItemsResult struct {
	MutationReceipt
	CharacterID int         `json:"characterID"`
	Moved       []MovedItem `json:"moved"`
}

// MoveOwnedItemsToStorage moves every named Inventory common record into common
// Storage as one atomic mutation. Each record is appended behind the records
// already there, in the order the caller listed them.
func (engine *Engine) MoveOwnedItemsToStorage(
	saveSessionID string,
	characterID int,
	moves []OwnedItemMove,
	expectedRevision string,
) (MoveOwnedItemsResult, error) {
	if err := validateOwnedItemMoves(moves); err != nil {
		return MoveOwnedItemsResult{}, err
	}
	if !isCanonicalRevision(expectedRevision) {
		return MoveOwnedItemsResult{}, apperror.InvalidRevision(expectedRevision)
	}

	var moved []MovedItem
	committed, err := engine.commitCharacterRevision(
		saveSessionID, kindMoveOwnedItemsToStorage, characterID,
		func(loaded *loadedSave) error {
			if err := requireActiveCharacterAt(loaded, characterID, expectedRevision); err != nil {
				return err
			}
			candidate := loaded.candidate()
			results := make([]MovedItem, 0, len(moves))
			for index, move := range moves {
				locator, err := loaded.session.resolveOwnedItemID(characterID, move.OwnedItemID)
				if err != nil {
					return fmt.Errorf("moves[%d]: %w", index, err)
				}
				position, err := commonStorageRecordCount(candidate, characterID)
				if err != nil {
					return fmt.Errorf("moves[%d]: %w", index, err)
				}
				outcome, err := moveOwnedItemToStorageRecord(
					candidate, locator, move.OwnedItemID, position,
					move.ExpectedGameID, move.MaxQuantity, move.SeparateInstances)
				if err != nil {
					return fmt.Errorf("moves[%d]: %w", index, err)
				}
				results = append(results, MovedItem{
					OwnedItemID:      move.OwnedItemID,
					GameID:           outcome.gameID,
					Quantity:         outcome.quantity,
					ContainerSection: StorageSectionCommon,
					PhysicalIndex:    outcome.physicalIndex,
					AcquisitionIndex: outcome.acquisitionIndex,
				})
			}
			loaded.snapshot = candidate.snapshot
			moved = results
			return nil
		})
	if err != nil {
		return MoveOwnedItemsResult{}, err
	}
	return MoveOwnedItemsResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		Moved:           moved,
	}, nil
}

// MoveOwnedItemsToInventory is MoveOwnedItemsToStorage in the other direction:
// every named Storage common record becomes an Inventory common record, in the
// order the caller listed them and behind the records already held.
func (engine *Engine) MoveOwnedItemsToInventory(
	saveSessionID string,
	characterID int,
	moves []OwnedItemMove,
	expectedRevision string,
) (MoveOwnedItemsResult, error) {
	if err := validateOwnedItemMoves(moves); err != nil {
		return MoveOwnedItemsResult{}, err
	}
	if !isCanonicalRevision(expectedRevision) {
		return MoveOwnedItemsResult{}, apperror.InvalidRevision(expectedRevision)
	}

	var moved []MovedItem
	committed, err := engine.commitCharacterRevision(
		saveSessionID, kindMoveOwnedItemsToInventory, characterID,
		func(loaded *loadedSave) error {
			if err := requireActiveCharacterAt(loaded, characterID, expectedRevision); err != nil {
				return err
			}
			candidate := loaded.candidate()
			results := make([]MovedItem, 0, len(moves))
			for index, move := range moves {
				locator, err := loaded.session.resolveOwnedItemID(characterID, move.OwnedItemID)
				if err != nil {
					return fmt.Errorf("moves[%d]: %w", index, err)
				}
				position, err := commonInventoryRecordCount(candidate, characterID)
				if err != nil {
					return fmt.Errorf("moves[%d]: %w", index, err)
				}
				outcome, err := moveOwnedItemToInventoryRecord(
					candidate, locator, move.OwnedItemID, position,
					move.ExpectedGameID, move.MaxQuantity, move.SeparateInstances)
				if err != nil {
					return fmt.Errorf("moves[%d]: %w", index, err)
				}
				results = append(results, MovedItem{
					OwnedItemID:      move.OwnedItemID,
					GameID:           outcome.gameID,
					Quantity:         outcome.quantity,
					ContainerSection: InventorySectionCommon,
					PhysicalIndex:    outcome.physicalIndex,
					AcquisitionIndex: outcome.acquisitionIndex,
				})
			}
			loaded.snapshot = candidate.snapshot
			moved = results
			return nil
		})
	if err != nil {
		return MoveOwnedItemsResult{}, err
	}
	return MoveOwnedItemsResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		Moved:           moved,
	}, nil
}

func validateOwnedItemMoves(moves []OwnedItemMove) error {
	if len(moves) == 0 {
		return fmt.Errorf("a batch move must name at least one ownedItemID")
	}
	seen := make(map[string]int, len(moves))
	for index, move := range moves {
		if move.OwnedItemID == "" {
			return fmt.Errorf("moves[%d]: ownedItemID is required", index)
		}
		if previous, duplicate := seen[move.OwnedItemID]; duplicate {
			return fmt.Errorf("moves repeats ownedItemID %q at positions %d and %d",
				move.OwnedItemID, previous, index)
		}
		seen[move.OwnedItemID] = index
		if move.MaxQuantity == 0 {
			return fmt.Errorf("moves[%d]: maxQuantity must be at least 1", index)
		}
	}
	return nil
}

// RemovedItem reports one committed removal of a batch. The echoed identity is
// already stale: the batch advanced the revision and the record is gone.
type RemovedItem struct {
	OwnedItemID string `json:"ownedItemID"`
	GameID      uint32 `json:"gameID"`
}

// RemoveOwnedItemsResult reports one committed batch removal.
type RemoveOwnedItemsResult struct {
	MutationReceipt
	CharacterID int           `json:"characterID"`
	Removed     []RemovedItem `json:"removed"`
}

// RemoveOwnedItems removes every named record as one atomic mutation. A record
// an Equipment, Quick Item or Pouch slot references rejects the whole batch,
// because the shared removal planner refuses it and no step is kept.
func (engine *Engine) RemoveOwnedItems(
	saveSessionID string,
	characterID int,
	ownedItemIDs []string,
	expectedRevision string,
) (RemoveOwnedItemsResult, error) {
	if len(ownedItemIDs) == 0 {
		return RemoveOwnedItemsResult{}, fmt.Errorf(
			"a batch removal must name at least one ownedItemID")
	}
	seen := make(map[string]int, len(ownedItemIDs))
	for index, ownedItemID := range ownedItemIDs {
		if ownedItemID == "" {
			return RemoveOwnedItemsResult{}, fmt.Errorf(
				"ownedItemIDs[%d] is empty", index)
		}
		if previous, duplicate := seen[ownedItemID]; duplicate {
			return RemoveOwnedItemsResult{}, fmt.Errorf(
				"ownedItemIDs repeats ownedItemID %q at positions %d and %d",
				ownedItemID, previous, index)
		}
		seen[ownedItemID] = index
	}
	if !isCanonicalRevision(expectedRevision) {
		return RemoveOwnedItemsResult{}, apperror.InvalidRevision(expectedRevision)
	}

	var removed []RemovedItem
	committed, err := engine.commitCharacterRevision(
		saveSessionID, kindRemoveOwnedItems, characterID,
		func(loaded *loadedSave) error {
			if err := requireActiveCharacterAt(loaded, characterID, expectedRevision); err != nil {
				return err
			}
			candidate := loaded.candidate()
			results := make([]RemovedItem, 0, len(ownedItemIDs))
			for index, ownedItemID := range ownedItemIDs {
				locator, err := loaded.session.resolveOwnedItemID(characterID, ownedItemID)
				if err != nil {
					return fmt.Errorf("ownedItemIDs[%d]: %w", index, err)
				}
				gameID, err := removeOwnedItemRecord(candidate, locator, ownedItemID)
				if err != nil {
					return fmt.Errorf("ownedItemIDs[%d]: %w", index, err)
				}
				results = append(results, RemovedItem{OwnedItemID: ownedItemID, GameID: gameID})
			}
			loaded.snapshot = candidate.snapshot
			removed = results
			return nil
		})
	if err != nil {
		return RemoveOwnedItemsResult{}, err
	}
	return RemoveOwnedItemsResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		Removed:         removed,
	}, nil
}

// ReorderInventoryItemsResult reports one committed anchored group move.
type ReorderInventoryItemsResult struct {
	MutationReceipt
	CharacterID         int      `json:"characterID"`
	OrderedOwnedItemIDs []string `json:"orderedOwnedItemIDs"`
	GameIDs             []uint32 `json:"gameIDs"`
	AcquisitionIndices  []uint32 `json:"acquisitionIndices"`
}

// ReorderInventoryItems moves a selected group of supported Inventory common
// records to a new position, anchored on one member of that group.
//
// The rule is the interface's own, stated once here: the anchor lands at
// targetPosition of the resulting order, the selected records that were in
// front of it stay in front of it, and the selected records that were behind it
// stay behind it. The group keeps its internal order, and every record outside
// the group keeps its relative order too.
//
// The result is one complete supported order, written through the same planner
// SetInventoryOrder uses, so the acquisition allocator, the retained buckets and
// the unsafe-index bound have exactly one implementation.
func (engine *Engine) ReorderInventoryItems(
	saveSessionID string,
	characterID int,
	anchorOwnedItemID string,
	groupOwnedItemIDs []string,
	targetPosition int,
	expectedRevision string,
	classifyGameID func(uint32) (bool, error),
) (ReorderInventoryItemsResult, error) {
	if anchorOwnedItemID == "" {
		return ReorderInventoryItemsResult{}, fmt.Errorf("anchorOwnedItemID is required")
	}
	if len(groupOwnedItemIDs) == 0 {
		return ReorderInventoryItemsResult{}, fmt.Errorf(
			"groupOwnedItemIDs must name at least one record")
	}
	anchorInGroup := false
	seen := make(map[string]int, len(groupOwnedItemIDs))
	for index, ownedItemID := range groupOwnedItemIDs {
		if ownedItemID == "" {
			return ReorderInventoryItemsResult{}, fmt.Errorf(
				"groupOwnedItemIDs[%d] is empty", index)
		}
		if previous, duplicate := seen[ownedItemID]; duplicate {
			return ReorderInventoryItemsResult{}, fmt.Errorf(
				"groupOwnedItemIDs repeats ownedItemID %q at positions %d and %d",
				ownedItemID, previous, index)
		}
		seen[ownedItemID] = index
		if ownedItemID == anchorOwnedItemID {
			anchorInGroup = true
		}
	}
	if !anchorInGroup {
		return ReorderInventoryItemsResult{}, fmt.Errorf(
			"anchorOwnedItemID %q is not part of groupOwnedItemIDs", anchorOwnedItemID)
	}
	if targetPosition < 0 {
		return ReorderInventoryItemsResult{}, fmt.Errorf(
			"targetPosition must not be negative; got %d", targetPosition)
	}
	if classifyGameID == nil {
		return ReorderInventoryItemsResult{}, fmt.Errorf("Inventory order classifier is required")
	}
	if !isCanonicalRevision(expectedRevision) {
		return ReorderInventoryItemsResult{}, apperror.InvalidRevision(expectedRevision)
	}

	var ordered []string
	var gameIDs, acquisitionIndices []uint32
	committed, err := engine.commitCharacterRevision(
		saveSessionID, kindReorderInventoryItems, characterID,
		func(loaded *loadedSave) error {
			if err := requireActiveCharacterAt(loaded, characterID, expectedRevision); err != nil {
				return err
			}
			current, err := supportedInventoryOrder(loaded, characterID, classifyGameID)
			if err != nil {
				return err
			}
			planned, err := planAnchoredInventoryOrder(
				current, anchorOwnedItemID, seen, targetPosition)
			if err != nil {
				return err
			}
			candidate := loaded.candidate()
			ids, indices, err := applyInventoryOrder(
				candidate, characterID, planned, classifyGameID)
			if err != nil {
				return err
			}
			loaded.snapshot = candidate.snapshot
			ordered = planned
			gameIDs = ids
			acquisitionIndices = indices
			return nil
		})
	if err != nil {
		return ReorderInventoryItemsResult{}, err
	}
	return ReorderInventoryItemsResult{
		MutationReceipt:     committed,
		CharacterID:         characterID,
		OrderedOwnedItemIDs: ordered,
		GameIDs:             gameIDs,
		AcquisitionIndices:  acquisitionIndices,
	}, nil
}

// planAnchoredInventoryOrder turns the current order plus one anchored group
// into the complete order the writer applies. It is pure: it reads no snapshot
// and writes nothing, so the whole placement rule can be reasoned about and
// rejected before the session is touched.
func planAnchoredInventoryOrder(
	current []string,
	anchorOwnedItemID string,
	group map[string]int,
	targetPosition int,
) ([]string, error) {
	inGroup := make([]string, 0, len(group))
	remaining := make([]string, 0, len(current))
	present := make(map[string]bool, len(current))
	for _, ownedItemID := range current {
		present[ownedItemID] = true
		if _, selected := group[ownedItemID]; selected {
			inGroup = append(inGroup, ownedItemID)
			continue
		}
		remaining = append(remaining, ownedItemID)
	}
	for ownedItemID := range group {
		if !present[ownedItemID] {
			return nil, fmt.Errorf(
				"ownedItemID %q is not a supported Inventory common record", ownedItemID)
		}
	}
	if targetPosition > len(current)-1 && len(current) > 0 {
		return nil, fmt.Errorf(
			"targetPosition %d is outside the range 0..%d for the supported Inventory order",
			targetPosition, len(current)-1)
	}

	anchorInGroup := -1
	for index, ownedItemID := range inGroup {
		if ownedItemID == anchorOwnedItemID {
			anchorInGroup = index
			break
		}
	}
	if anchorInGroup < 0 {
		return nil, fmt.Errorf(
			"anchorOwnedItemID %q is not a supported Inventory common record", anchorOwnedItemID)
	}
	// The anchor lands on targetPosition, so the members that precede it inside
	// the group occupy the positions directly in front of that index.
	insertAt := targetPosition - anchorInGroup
	if insertAt < 0 || insertAt > len(remaining) {
		return nil, fmt.Errorf(
			"targetPosition %d cannot place the anchored group of %d records",
			targetPosition, len(inGroup))
	}

	planned := make([]string, 0, len(current))
	planned = append(planned, remaining[:insertAt]...)
	planned = append(planned, inGroup...)
	planned = append(planned, remaining[insertAt:]...)
	return planned, nil
}

// commonInventoryRecordCount and commonStorageRecordCount report how many
// non-empty common records a container holds, which is also the append position
// of the next move into it.
//
// The caller holds Engine.mutex.
func commonInventoryRecordCount(loaded *loadedSave, characterID int) (int, error) {
	records, err := readInventoryRecords(loaded, characterID)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, record := range records {
		if record.ContainerSection == InventorySectionCommon {
			count++
		}
	}
	return count, nil
}

func commonStorageRecordCount(loaded *loadedSave, characterID int) (int, error) {
	records, err := readStorageRecords(loaded, characterID)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, record := range records {
		if record.ContainerSection == StorageSectionCommon {
			count++
		}
	}
	return count, nil
}
