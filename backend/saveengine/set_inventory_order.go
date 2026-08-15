package saveengine

import "fmt"

const (
	itemOrderReservedIndexMax uint32 = 432
	itemOrderUnsafeIndex      uint32 = 10000
)

// SetInventoryOrderResult reports one committed supported Inventory order.
type SetInventoryOrderResult struct {
	SaveSessionID      string   `json:"saveSessionID"`
	SaveRevision       string   `json:"saveRevision"`
	CharacterID        int      `json:"characterID"`
	GameIDs            []uint32 `json:"gameIDs"`
	AcquisitionIndices []uint32 `json:"acquisitionIndices"`
}

type inventoryOrderEntry struct {
	physicalIndex int
	gameID        uint32
	acquisition   uint32
}

// SetInventoryOrder atomically replaces the logical order of every supported
// Inventory common record. classifyGameID identifies the immutable catalog
// subset owned by the endpoint; SaveEngine owns identities and binary layout.
func (engine *Engine) SetInventoryOrder(
	saveSessionID string,
	characterID int,
	orderedOwnedItemIDs []string,
	expectedRevision string,
	classifyGameID func(uint32) (bool, error),
) (SetInventoryOrderResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetInventoryOrderResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if len(orderedOwnedItemIDs) == 0 {
		return SetInventoryOrderResult{}, fmt.Errorf("orderedOwnedItemIDs must not be empty")
	}
	if len(orderedOwnedItemIDs) > inventoryHeldCommonRecords {
		return SetInventoryOrderResult{}, fmt.Errorf(
			"orderedOwnedItemIDs contains %d records, want at most %d",
			len(orderedOwnedItemIDs), inventoryHeldCommonRecords)
	}
	if classifyGameID == nil {
		return SetInventoryOrderResult{}, fmt.Errorf("Inventory order classifier is required")
	}

	var gameIDs, acquisitionIndices []uint32
	saveRevision, err := engine.commitCharacterRevision(saveSessionID, opSetInventoryOrder, characterID, func(loaded *loadedSave) error {
		if characterID < 0 || characterID >= characterSlotCount {
			return fmt.Errorf("characterID %d is outside the range 0..%d",
				characterID, characterSlotCount-1)
		}
		if expectedRevision != loaded.session.revisionString() {
			return fmt.Errorf(
				"expectedRevision %q does not match the current saveRevision %q",
				expectedRevision, loaded.session.revisionString())
		}

		flag, err := loaded.snapshot.readAt(
			userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
		}
		if flag[0] != userData10ActiveFlagValue {
			return fmt.Errorf("character %d is not active", characterID)
		}

		records, err := readInventoryRecords(loaded, characterID)
		if err != nil {
			return err
		}
		byHandle, err := readGaItemMap(loaded.snapshot, loaded.session.platform, characterID)
		if err != nil {
			return fmt.Errorf("cannot resolve items of character %d: %w", characterID, err)
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
				return fmt.Errorf("Inventory common record %d of character %d: %w",
					record.PhysicalIndex, characterID, err)
			}
			supported, err := classifyGameID(gameID)
			if err != nil {
				return fmt.Errorf("Inventory common record %d of character %d: %w",
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

		if len(orderedOwnedItemIDs) != len(eligible) {
			return fmt.Errorf(
				"orderedOwnedItemIDs contains %d records, but Inventory common has %d supported records",
				len(orderedOwnedItemIDs), len(eligible))
		}
		ordered := make([]inventoryOrderEntry, len(orderedOwnedItemIDs))
		seen := make(map[string]int, len(orderedOwnedItemIDs))
		for position, ownedItemID := range orderedOwnedItemIDs {
			if ownedItemID == "" {
				return fmt.Errorf("orderedOwnedItemIDs[%d] is empty", position)
			}
			if previous, duplicate := seen[ownedItemID]; duplicate {
				return fmt.Errorf(
					"orderedOwnedItemIDs repeats ownedItemID %q at positions %d and %d",
					ownedItemID, previous, position)
			}
			seen[ownedItemID] = position

			locator, err := loaded.session.resolveOwnedItemID(characterID, ownedItemID)
			if err != nil {
				return fmt.Errorf("orderedOwnedItemIDs[%d]: %w", position, err)
			}
			if locator.container != ownedContainerInventory ||
				locator.containerSection != InventorySectionCommon {
				return fmt.Errorf(
					"orderedOwnedItemIDs[%d]: ownedItemID %q must address Inventory common",
					position, ownedItemID)
			}
			entry, found := eligible[ownedItemID]
			if !found {
				return fmt.Errorf(
					"orderedOwnedItemIDs[%d]: ownedItemID %q is not supported by Inventory order",
					position, ownedItemID)
			}
			ordered[position] = entry
		}

		inventoryAt, err := inventoryHeldSectionAt(loaded, characterID)
		if err != nil {
			return err
		}
		nextAcquisitionAt := inventoryAt + addItemNextAcquisitionOffset
		storedNext, err := loaded.snapshot.uint32At(nextAcquisitionAt)
		if err != nil {
			return fmt.Errorf(
				"cannot read NextAcquisitionSortId of character %d: %w", characterID, err)
		}
		indices, err := planItemOrderIndices(storedNext, len(ordered), retainedBuckets)
		if err != nil {
			return fmt.Errorf("character %d: %w", characterID, err)
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
			return fmt.Errorf("cannot set Inventory order: %w", err)
		}

		gameIDs = make([]uint32, len(ordered))
		for index, entry := range ordered {
			gameIDs[index] = entry.gameID
		}
		acquisitionIndices = indices
		return nil
	})
	if err != nil {
		return SetInventoryOrderResult{}, err
	}

	return SetInventoryOrderResult{
		SaveSessionID:      saveSessionID,
		SaveRevision:       saveRevision,
		CharacterID:        characterID,
		GameIDs:            gameIDs,
		AcquisitionIndices: acquisitionIndices,
	}, nil
}

func planItemOrderIndices(
	storedNext uint32,
	count int,
	retainedBuckets map[uint32]struct{},
) ([]uint32, error) {
	base := uint64(storedNext)
	if base <= uint64(itemOrderReservedIndexMax) {
		base = uint64(itemOrderReservedIndexMax) + 2
	}
	if base%2 != 0 {
		base++
	}
	for bucket := range retainedBuckets {
		after := (uint64(bucket) + 1) * 2
		if after > base {
			base = after
		}
	}

	last := base + uint64(count-1)*2
	if last >= uint64(itemOrderUnsafeIndex) {
		return nil, fmt.Errorf(
			"item order would assign acquisition index %d, want at most %d",
			last, itemOrderUnsafeIndex-1)
	}

	indices := make([]uint32, count)
	for position := range indices {
		indices[position] = uint32(base) + uint32(position)*2
	}
	return indices, nil
}
