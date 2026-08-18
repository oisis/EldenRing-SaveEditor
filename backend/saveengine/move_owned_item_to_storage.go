package saveengine

import (
	"encoding/binary"
	"fmt"
)

// MoveOwnedItemToStorageResult reports one committed Inventory-to-Storage move.
// OwnedItemID is the now-stale source identity. The destination receives a new
// identity on the next Storage read under SaveRevision.
type MoveOwnedItemToStorageResult struct {
	SaveSessionID    string `json:"saveSessionID"`
	SaveRevision     string `json:"saveRevision"`
	OwnedItemID      string `json:"ownedItemID"`
	CharacterID      int    `json:"characterID"`
	GameID           uint32 `json:"gameID"`
	Quantity         uint32 `json:"quantity"`
	ContainerSection string `json:"containerSection"`
	TargetPosition   int    `json:"targetPosition"`
	PhysicalIndex    int    `json:"physicalIndex"`
	AcquisitionIndex uint32 `json:"acquisitionIndex"`
}

// MoveOwnedItemToStorage moves one complete physical Inventory common record
// into the common section of Storage. It never merges, rehandles, allocates or
// repacks. targetPosition is the zero-based position among the non-empty common
// Storage records in their physical order; len(records) appends.
func (engine *Engine) MoveOwnedItemToStorage(
	saveSessionID string,
	characterID int,
	ownedItemID string,
	targetPosition int,
	expectedRevision string,
	expectedGameID uint32,
	maxStorage uint32,
) (MoveOwnedItemToStorageResult, error) {
	if targetPosition < 0 {
		return MoveOwnedItemToStorageResult{}, fmt.Errorf(
			"targetPosition must not be negative; got %d", targetPosition)
	}
	if maxStorage == 0 {
		return MoveOwnedItemToStorageResult{}, fmt.Errorf("maxStorage must be at least 1")
	}
	if !isCanonicalRevision(expectedRevision) {
		return MoveOwnedItemToStorageResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	var moved movedStorageRecord
	saveRevision, err := engine.commitCharacterRevision(saveSessionID, opMoveOwnedItemToStorage, characterID, func(loaded *loadedSave) error {
		if characterID < 0 || characterID >= characterSlotCount {
			return fmt.Errorf("characterID %d is outside the range 0..%d",
				characterID, characterSlotCount-1)
		}
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return fmt.Errorf(
				"expectedRevision %q does not match the current saveRevision %q",
				expectedRevision, current)
		}

		locator, err := loaded.session.resolveOwnedItemID(characterID, ownedItemID)
		if err != nil {
			return err
		}
		moved, err = moveOwnedItemToStorageRecord(
			loaded, locator, ownedItemID, targetPosition, expectedGameID, maxStorage)
		return err
	})
	if err != nil {
		return MoveOwnedItemToStorageResult{}, err
	}

	return MoveOwnedItemToStorageResult{
		SaveSessionID:    saveSessionID,
		SaveRevision:     saveRevision,
		OwnedItemID:      ownedItemID,
		CharacterID:      characterID,
		GameID:           moved.gameID,
		Quantity:         moved.quantity,
		ContainerSection: StorageSectionCommon,
		TargetPosition:   targetPosition,
		PhysicalIndex:    moved.physicalIndex,
		AcquisitionIndex: moved.acquisitionIndex,
	}, nil
}

type movedStorageRecord struct {
	gameID           uint32
	quantity         uint32
	physicalIndex    int
	acquisitionIndex uint32
}

// moveOwnedItemToStorageRecord validates the complete plan before applying its
// five non-overlapping ranges. The caller holds Engine.mutex.
func moveOwnedItemToStorageRecord(
	loaded *loadedSave,
	locator ownedItemLocator,
	ownedItemID string,
	targetPosition int,
	expectedGameID uint32,
	maxStorage uint32,
) (movedStorageRecord, error) {
	if locator.container != ownedContainerInventory {
		return movedStorageRecord{}, fmt.Errorf(
			"ownedItemID %q is not in Inventory", ownedItemID)
	}
	if locator.containerSection != InventorySectionCommon {
		return movedStorageRecord{}, fmt.Errorf(
			"ownedItemID %q addresses an Inventory key record; moving key records to Storage is not supported",
			ownedItemID)
	}

	records, err := readInventoryRecords(loaded, locator.characterID)
	if err != nil {
		return movedStorageRecord{}, err
	}
	var source *InventoryRecord
	for index := range records {
		record := &records[index]
		if record.ContainerSection == locator.containerSection &&
			record.PhysicalIndex == locator.physicalIndex && record.OwnedItemID == ownedItemID {
			source = record
			break
		}
	}
	if source == nil {
		return movedStorageRecord{}, fmt.Errorf(
			"ownedItemID %q no longer addresses a record of character %d",
			ownedItemID, locator.characterID)
	}

	byHandle, err := readGaItemMap(loaded.snapshot, loaded.session.platform, locator.characterID)
	if err != nil {
		return movedStorageRecord{}, fmt.Errorf(
			"cannot resolve items of character %d: %w", locator.characterID, err)
	}
	gameID, err := resolveGaItemHandle(byHandle, source.GaItemHandle)
	if err != nil {
		return movedStorageRecord{}, fmt.Errorf("ownedItemID %q: %w", ownedItemID, err)
	}
	if gameID != expectedGameID {
		return movedStorageRecord{}, fmt.Errorf(
			"ownedItemID %q now denotes item 0x%08X, not the expected 0x%08X",
			ownedItemID, gameID, expectedGameID)
	}
	if err := rejectReferencedOwnedItem(loaded, locator, source.GaItemHandle, ownedItemID); err != nil {
		return movedStorageRecord{}, err
	}

	stored, err := readStorageRecords(loaded, locator.characterID)
	if err != nil {
		return movedStorageRecord{}, err
	}
	commonCount := 0
	total := uint64(source.Quantity)
	for _, record := range stored {
		if record.ContainerSection == StorageSectionCommon {
			commonCount++
		}
		storedGameID, err := resolveGaItemHandle(byHandle, record.GaItemHandle)
		if err != nil {
			return movedStorageRecord{}, fmt.Errorf(
				"storage record %s#%d of character %d: %w",
				record.ContainerSection, record.PhysicalIndex, locator.characterID, err)
		}
		if storedGameID == gameID {
			total += uint64(record.Quantity)
		}
	}
	if total > uint64(maxStorage) {
		return movedStorageRecord{}, fmt.Errorf(
			"moving ownedItemID %q would store %d units of item 0x%08X, above its storage limit of %d",
			ownedItemID, total, gameID, maxStorage)
	}
	if targetPosition > commonCount {
		return movedStorageRecord{}, fmt.Errorf(
			"targetPosition %d is outside the range 0..%d for common Storage",
			targetPosition, commonCount)
	}

	inventoryAt, inventoryCountAt, recordSize, err := ownedItemRowAndCountAt(loaded, locator)
	if err != nil {
		return movedStorageRecord{}, err
	}
	if recordSize != inventoryHeldRecordSize {
		return movedStorageRecord{}, fmt.Errorf(
			"ownedItemID %q addresses a record of unexpected size %d", ownedItemID, recordSize)
	}
	rawSource, err := loaded.snapshot.readAt(inventoryAt, inventoryHeldRecordSize)
	if err != nil {
		return movedStorageRecord{}, fmt.Errorf("cannot read ownedItemID %q: %w", ownedItemID, err)
	}
	if binary.LittleEndian.Uint32(rawSource) != source.GaItemHandle {
		return movedStorageRecord{}, fmt.Errorf(
			"ownedItemID %q changed while its move was being planned", ownedItemID)
	}

	storageAt, err := storageBoxSectionAt(loaded, locator.characterID)
	if err != nil {
		return movedStorageRecord{}, err
	}
	common, err := loaded.snapshot.readAt(storageAt+storageCommonAt, storageCommonSize)
	if err != nil {
		return movedStorageRecord{}, fmt.Errorf(
			"cannot read common Storage of character %d: %w", locator.characterID, err)
	}
	storageCount, err := loaded.snapshot.uint32At(storageAt)
	if err != nil {
		return movedStorageRecord{}, fmt.Errorf(
			"cannot read common Storage count of character %d: %w", locator.characterID, err)
	}
	if storageCount == ^uint32(0) {
		return movedStorageRecord{}, fmt.Errorf(
			"common Storage count of character %d cannot be advanced", locator.characterID)
	}

	countersAt := storageAt + storageKeyAt + storageKeySize
	nextEquip, err := loaded.snapshot.uint32At(countersAt)
	if err != nil {
		return movedStorageRecord{}, fmt.Errorf("cannot read Storage NextEquipIndex of character %d: %w",
			locator.characterID, err)
	}
	storedNext, err := loaded.snapshot.uint32At(countersAt + 4)
	if err != nil {
		return movedStorageRecord{}, fmt.Errorf(
			"cannot read Storage NextAcquisitionSortId of character %d: %w", locator.characterID, err)
	}

	acquisitionIndex, nextAcquisition, updatedEquip, err := nextStorageAcquisitionAndCounters(
		storedNext, nextEquip, stored, locator.characterID)
	if err != nil {
		return movedStorageRecord{}, err
	}
	movedRaw := append([]byte(nil), rawSource...)
	binary.LittleEndian.PutUint32(movedRaw[8:], acquisitionIndex)
	updatedCommon, physicalIndex, err := insertStorageCommonRecord(
		common, movedRaw, targetPosition)
	if err != nil {
		return movedStorageRecord{}, fmt.Errorf("character %d: %w", locator.characterID, err)
	}

	inventoryCount, err := loaded.snapshot.uint32At(inventoryCountAt)
	if err != nil {
		return movedStorageRecord{}, fmt.Errorf(
			"cannot read Inventory common count of character %d: %w", locator.characterID, err)
	}
	cleared := make([]byte, inventoryHeldRecordSize)
	binary.LittleEndian.PutUint32(cleared[8:], uint32(locator.physicalIndex))
	newInventoryCount := inventoryCount
	if newInventoryCount > 0 {
		newInventoryCount--
	}
	writes := []byteWrite{
		{at: inventoryAt, data: cleared},
		{at: inventoryCountAt, data: littleEndianUint32(newInventoryCount)},
		{at: storageAt + storageCommonAt, data: updatedCommon},
		{at: storageAt, data: littleEndianUint32(storageCount + 1)},
		{at: countersAt, data: littleEndianUint32(updatedEquip)},
		{at: countersAt + 4, data: littleEndianUint32(nextAcquisition)},
	}
	if err := applyByteWrites(loaded.snapshot, writes); err != nil {
		return movedStorageRecord{}, fmt.Errorf("ownedItemID %q: %w", ownedItemID, err)
	}

	return movedStorageRecord{
		gameID:           gameID,
		quantity:         source.Quantity,
		physicalIndex:    physicalIndex,
		acquisitionIndex: acquisitionIndex,
	}, nil
}

// insertStorageCommonRecord inserts one row into the physical order of the
// non-empty common records. It rotates one contiguous span into an empty row;
// bytes outside that span stay identical.
func insertStorageCommonRecord(common, record []byte, targetPosition int) ([]byte, int, error) {
	if len(common) != storageCommonSize || len(record) != storageRecordSize {
		return nil, 0, fmt.Errorf("common Storage carries an unexpected record layout")
	}
	active := make([]int, 0)
	for row := 0; row < storageCommonRecords; row++ {
		handle := binary.LittleEndian.Uint32(common[row*storageRecordSize:])
		if handle != storageEmptyHandle && handle != storageInvalidHandle {
			active = append(active, row)
		}
	}
	if targetPosition < 0 || targetPosition > len(active) {
		return nil, 0, fmt.Errorf(
			"targetPosition %d is outside the range 0..%d for common Storage",
			targetPosition, len(active))
	}
	if len(active) == storageCommonRecords {
		return nil, 0, fmt.Errorf("common Storage holds no free record")
	}

	updated := append([]byte(nil), common...)
	writeAt := func(row int) {
		copy(updated[row*storageRecordSize:(row+1)*storageRecordSize], record)
	}
	if len(active) == 0 {
		writeAt(0)
		return updated, 0, nil
	}

	if targetPosition == len(active) {
		last := active[len(active)-1]
		for row := last + 1; row < storageCommonRecords; row++ {
			handle := binary.LittleEndian.Uint32(updated[row*storageRecordSize:])
			if handle == storageEmptyHandle || handle == storageInvalidHandle {
				writeAt(row)
				return updated, row, nil
			}
		}
		for empty := last - 1; empty >= 0; empty-- {
			handle := binary.LittleEndian.Uint32(updated[empty*storageRecordSize:])
			if handle != storageEmptyHandle && handle != storageInvalidHandle {
				continue
			}
			copy(updated[empty*storageRecordSize:last*storageRecordSize],
				updated[(empty+1)*storageRecordSize:(last+1)*storageRecordSize])
			writeAt(last)
			return updated, last, nil
		}
	}

	anchor := active[targetPosition]
	for empty := anchor + 1; empty < storageCommonRecords; empty++ {
		handle := binary.LittleEndian.Uint32(updated[empty*storageRecordSize:])
		if handle != storageEmptyHandle && handle != storageInvalidHandle {
			continue
		}
		copy(updated[(anchor+1)*storageRecordSize:(empty+1)*storageRecordSize],
			updated[anchor*storageRecordSize:empty*storageRecordSize])
		writeAt(anchor)
		return updated, anchor, nil
	}
	for empty := anchor - 1; empty >= 0; empty-- {
		handle := binary.LittleEndian.Uint32(updated[empty*storageRecordSize:])
		if handle != storageEmptyHandle && handle != storageInvalidHandle {
			continue
		}
		copy(updated[empty*storageRecordSize:(anchor-1)*storageRecordSize],
			updated[(empty+1)*storageRecordSize:anchor*storageRecordSize])
		writeAt(anchor - 1)
		return updated, anchor - 1, nil
	}
	return nil, 0, fmt.Errorf("common Storage holds no free record")
}
