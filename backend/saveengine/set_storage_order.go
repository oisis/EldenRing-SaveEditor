package saveengine

import "fmt"

// SetStorageOrderResult reports one committed supported Storage order.
type SetStorageOrderResult struct {
	SaveSessionID      string   `json:"saveSessionID"`
	SaveRevision       string   `json:"saveRevision"`
	CharacterID        int      `json:"characterID"`
	GameIDs            []uint32 `json:"gameIDs"`
	AcquisitionIndices []uint32 `json:"acquisitionIndices"`
}

type storageOrderEntry struct {
	physicalIndex int
	gameID        uint32
	acquisition   uint32
}

// SetStorageOrder atomically replaces the logical order of every supported
// Storage common record. classifyGameID identifies the immutable catalog subset
// owned by the endpoint; SaveEngine owns identities and binary layout.
func (engine *Engine) SetStorageOrder(
	saveSessionID string,
	characterID int,
	orderedOwnedItemIDs []string,
	expectedRevision string,
	classifyGameID func(uint32) (bool, error),
) (SetStorageOrderResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetStorageOrderResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if len(orderedOwnedItemIDs) == 0 {
		return SetStorageOrderResult{}, fmt.Errorf("orderedOwnedItemIDs must not be empty")
	}
	if len(orderedOwnedItemIDs) > storageCommonRecords {
		return SetStorageOrderResult{}, fmt.Errorf(
			"orderedOwnedItemIDs contains %d records, want at most %d",
			len(orderedOwnedItemIDs), storageCommonRecords)
	}
	if classifyGameID == nil {
		return SetStorageOrderResult{}, fmt.Errorf("Storage order classifier is required")
	}

	var gameIDs, acquisitionIndices []uint32
	saveRevision, err := engine.commitCharacterRevision(saveSessionID, opSetStorageOrder, characterID, func(loaded *loadedSave) error {
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

		records, err := readStorageRecords(loaded, characterID)
		if err != nil {
			return err
		}
		byHandle, err := readGaItemMap(loaded.snapshot, loaded.session.platform, characterID)
		if err != nil {
			return fmt.Errorf("cannot resolve items of character %d: %w", characterID, err)
		}

		eligible := make(map[string]storageOrderEntry)
		retainedBuckets := make(map[uint32]struct{})
		for _, record := range records {
			if record.ContainerSection != StorageSectionCommon {
				retainedBuckets[record.AcquisitionIndex>>1] = struct{}{}
				continue
			}
			gameID, err := resolveGaItemHandle(byHandle, record.GaItemHandle)
			if err != nil {
				return fmt.Errorf("Storage common record %d of character %d: %w",
					record.PhysicalIndex, characterID, err)
			}
			supported, err := classifyGameID(gameID)
			if err != nil {
				return fmt.Errorf("Storage common record %d of character %d: %w",
					record.PhysicalIndex, characterID, err)
			}
			if !supported {
				retainedBuckets[record.AcquisitionIndex>>1] = struct{}{}
				continue
			}
			eligible[record.OwnedItemID] = storageOrderEntry{
				physicalIndex: record.PhysicalIndex,
				gameID:        gameID,
				acquisition:   record.AcquisitionIndex,
			}
		}

		if len(orderedOwnedItemIDs) != len(eligible) {
			return fmt.Errorf(
				"orderedOwnedItemIDs contains %d records, but Storage common has %d supported records",
				len(orderedOwnedItemIDs), len(eligible))
		}
		ordered := make([]storageOrderEntry, len(orderedOwnedItemIDs))
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
			if locator.container != ownedContainerStorage ||
				locator.containerSection != StorageSectionCommon {
				return fmt.Errorf(
					"orderedOwnedItemIDs[%d]: ownedItemID %q must address Storage common",
					position, ownedItemID)
			}
			entry, found := eligible[ownedItemID]
			if !found {
				return fmt.Errorf(
					"orderedOwnedItemIDs[%d]: ownedItemID %q is not supported by Storage order",
					position, ownedItemID)
			}
			ordered[position] = entry
		}

		storageAt, err := storageBoxSectionAt(loaded, characterID)
		if err != nil {
			return err
		}
		nextAcquisitionAt := storageAt + storageKeyAt + storageKeySize + 4
		storedNext, err := loaded.snapshot.uint32At(nextAcquisitionAt)
		if err != nil {
			return fmt.Errorf(
				"cannot read Storage NextAcquisitionSortId of character %d: %w", characterID, err)
		}
		indices, err := planItemOrderIndices(storedNext, len(ordered), retainedBuckets)
		if err != nil {
			return fmt.Errorf("character %d Storage: %w", characterID, err)
		}

		writes := make([]byteWrite, 0, len(ordered)+1)
		for position, entry := range ordered {
			if entry.acquisition == indices[position] {
				continue
			}
			writes = append(writes, byteWrite{
				at: storageAt + storageCommonAt +
					int64(entry.physicalIndex)*storageRecordSize + 8,
				data: littleEndianUint32(indices[position]),
			})
		}
		writes = append(writes, byteWrite{
			at:   nextAcquisitionAt,
			data: littleEndianUint32(indices[len(indices)-1] + 1),
		})
		if err := applyByteWrites(loaded.snapshot, writes); err != nil {
			return fmt.Errorf("cannot set Storage order: %w", err)
		}

		gameIDs = make([]uint32, len(ordered))
		for index, entry := range ordered {
			gameIDs[index] = entry.gameID
		}
		acquisitionIndices = indices
		return nil
	})
	if err != nil {
		return SetStorageOrderResult{}, err
	}

	return SetStorageOrderResult{
		SaveSessionID:      saveSessionID,
		SaveRevision:       saveRevision,
		CharacterID:        characterID,
		GameIDs:            gameIDs,
		AcquisitionIndices: acquisitionIndices,
	}, nil
}
