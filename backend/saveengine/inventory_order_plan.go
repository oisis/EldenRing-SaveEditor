package saveengine

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// This file holds the shared planner of the supported Inventory order. Both the
// complete permutation (SetInventoryOrder) and the anchored group move
// (ReorderInventoryItems) write the same acquisition indices through it, so the
// eligibility rule, the retained buckets and the index allocator exist exactly
// once.

// requireActiveCharacterAt is the guard every character-scoped item mutation
// runs before it reads a single record: the slot has to exist, the caller has to
// address the current revision, and the slot has to be active. A residual slot
// of a deleted character is never located or written.
//
// The caller holds Engine.mutex.
func requireActiveCharacterAt(loaded *loadedSave, characterID int, expectedRevision string) error {
	if characterID < 0 || characterID >= characterSlotCount {
		return fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}
	current := loaded.session.revisionString()
	if expectedRevision != current {
		return apperror.RevisionConflict(expectedRevision, current)
	}
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

// eligibleInventoryOrderRecords resolves the supported common records of one
// character together with the acquisition buckets every unsupported or key
// record keeps. It is the single implementation of "which records take part in
// the Inventory order".
//
// The caller holds Engine.mutex.
func eligibleInventoryOrderRecords(
	loaded *loadedSave,
	characterID int,
	classifyGameID func(uint32) (bool, error),
) (map[string]inventoryOrderEntry, map[uint32]struct{}, error) {
	records, err := readInventoryRecords(loaded, characterID)
	if err != nil {
		return nil, nil, err
	}
	byHandle, err := readGaItemMap(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot resolve items of character %d: %w", characterID, err)
	}

	eligible := make(map[string]inventoryOrderEntry)
	retainedBuckets := make(map[uint32]struct{})
	for _, record := range records {
		if record.ContainerSection != InventorySectionCommon {
			retainedBuckets[record.AcquisitionIndex>>1] = struct{}{}
			continue
		}
		gameID, err := resolveGaItemHandle(byHandle, record.GaItemHandle)
		if err != nil {
			return nil, nil, fmt.Errorf("Inventory common record %d of character %d: %w",
				record.PhysicalIndex, characterID, err)
		}
		supported, err := classifyGameID(gameID)
		if err != nil {
			return nil, nil, fmt.Errorf("Inventory common record %d of character %d: %w",
				record.PhysicalIndex, characterID, err)
		}
		if !supported {
			retainedBuckets[record.AcquisitionIndex>>1] = struct{}{}
			continue
		}
		eligible[record.OwnedItemID] = inventoryOrderEntry{
			physicalIndex: record.PhysicalIndex,
			gameID:        gameID,
			acquisition:   record.AcquisitionIndex,
		}
	}
	return eligible, retainedBuckets, nil
}

// supportedInventoryOrder returns the identities of every supported common
// record in their current logical order, which is ascending acquisition index.
// It is what an anchored group move rearranges.
//
// The caller holds Engine.mutex.
func supportedInventoryOrder(
	loaded *loadedSave,
	characterID int,
	classifyGameID func(uint32) (bool, error),
) ([]string, error) {
	eligible, _, err := eligibleInventoryOrderRecords(loaded, characterID, classifyGameID)
	if err != nil {
		return nil, err
	}
	ordered := make([]string, 0, len(eligible))
	for ownedItemID := range eligible {
		ordered = append(ordered, ownedItemID)
	}
	// The map iteration is not an order, so the list is sorted by the one value
	// that is the logical order of the container.
	for index := 1; index < len(ordered); index++ {
		value := ordered[index]
		position := index
		for position > 0 &&
			eligible[ordered[position-1]].acquisition > eligible[value].acquisition {
			ordered[position] = ordered[position-1]
			position--
		}
		ordered[position] = value
	}
	return ordered, nil
}

// applyInventoryOrder writes the complete supported order of common Inventory.
// orderedOwnedItemIDs must name every supported record exactly once; a missing,
// repeated, foreign or unsupported identity rejects the whole plan before a
// single byte changes.
//
// The caller holds Engine.mutex.
func applyInventoryOrder(
	loaded *loadedSave,
	characterID int,
	orderedOwnedItemIDs []string,
	classifyGameID func(uint32) (bool, error),
) ([]uint32, []uint32, error) {
	eligible, retainedBuckets, err := eligibleInventoryOrderRecords(
		loaded, characterID, classifyGameID)
	if err != nil {
		return nil, nil, err
	}
	if len(orderedOwnedItemIDs) != len(eligible) {
		return nil, nil, fmt.Errorf(
			"orderedOwnedItemIDs contains %d records, but Inventory common has %d supported records",
			len(orderedOwnedItemIDs), len(eligible))
	}

	ordered := make([]inventoryOrderEntry, len(orderedOwnedItemIDs))
	seen := make(map[string]int, len(orderedOwnedItemIDs))
	for position, ownedItemID := range orderedOwnedItemIDs {
		if ownedItemID == "" {
			return nil, nil, fmt.Errorf("orderedOwnedItemIDs[%d] is empty", position)
		}
		if previous, duplicate := seen[ownedItemID]; duplicate {
			return nil, nil, fmt.Errorf(
				"orderedOwnedItemIDs repeats ownedItemID %q at positions %d and %d",
				ownedItemID, previous, position)
		}
		seen[ownedItemID] = position

		locator, err := loaded.session.resolveOwnedItemID(characterID, ownedItemID)
		if err != nil {
			return nil, nil, fmt.Errorf("orderedOwnedItemIDs[%d]: %w", position, err)
		}
		if locator.container != ownedContainerInventory ||
			locator.containerSection != InventorySectionCommon {
			return nil, nil, fmt.Errorf(
				"orderedOwnedItemIDs[%d]: ownedItemID %q must address Inventory common",
				position, ownedItemID)
		}
		entry, found := eligible[ownedItemID]
		if !found {
			return nil, nil, fmt.Errorf(
				"orderedOwnedItemIDs[%d]: ownedItemID %q is not supported by Inventory order",
				position, ownedItemID)
		}
		ordered[position] = entry
	}

	inventoryAt, err := inventoryHeldSectionAt(loaded, characterID)
	if err != nil {
		return nil, nil, err
	}
	nextAcquisitionAt := inventoryAt + addItemNextAcquisitionOffset
	storedNext, err := loaded.snapshot.uint32At(nextAcquisitionAt)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"cannot read NextAcquisitionSortId of character %d: %w", characterID, err)
	}
	indices, err := planItemOrderIndices(storedNext, len(ordered), retainedBuckets)
	if err != nil {
		return nil, nil, fmt.Errorf("character %d: %w", characterID, err)
	}

	writes := make([]byteWrite, 0, len(ordered)+1)
	for position, entry := range ordered {
		if entry.acquisition == indices[position] {
			continue
		}
		writes = append(writes, byteWrite{
			at:   inventoryAt + int64(entry.physicalIndex)*inventoryHeldRecordSize + 8,
			data: littleEndianUint32(indices[position]),
		})
	}
	writes = append(writes, byteWrite{
		at:   nextAcquisitionAt,
		data: littleEndianUint32(indices[len(indices)-1] + 1),
	})
	if err := applyByteWrites(loaded.snapshot, writes); err != nil {
		return nil, nil, fmt.Errorf("cannot set Inventory order: %w", err)
	}

	gameIDs := make([]uint32, len(ordered))
	for index, entry := range ordered {
		gameIDs[index] = entry.gameID
	}
	return gameIDs, indices, nil
}
