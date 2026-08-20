package saveengine

import (
	"encoding/binary"
	"fmt"
	"sort"
)

// MoveOwnedItemToInventoryResult reports one committed Storage-to-Inventory move.
// OwnedItemID is the now-stale source identity. The destination receives a new
// identity on the next Inventory read under SaveRevision.
type MoveOwnedItemToInventoryResult struct {
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

// MoveOwnedItemToInventory moves one complete physical Storage common record
// into the common section of Inventory. targetPosition is the zero-based
// position in the game's acquisition-index order; len(records) appends. Existing
// physical Inventory rows never move, so Equipment, Quick Item and Pouch row
// references stay valid.
func (engine *Engine) MoveOwnedItemToInventory(
	saveSessionID string,
	characterID int,
	ownedItemID string,
	targetPosition int,
	expectedRevision string,
	expectedGameID uint32,
	maxInventory uint32,
	separateInstances bool,
) (MoveOwnedItemToInventoryResult, error) {
	if targetPosition < 0 {
		return MoveOwnedItemToInventoryResult{}, fmt.Errorf(
			"targetPosition must not be negative; got %d", targetPosition)
	}
	if maxInventory == 0 {
		return MoveOwnedItemToInventoryResult{}, fmt.Errorf("maxInventory must be at least 1")
	}
	if !isCanonicalRevision(expectedRevision) {
		return MoveOwnedItemToInventoryResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	var moved movedInventoryRecord
	saveRevision, err := engine.commitCharacterRevision(saveSessionID, opMoveOwnedItemToInventory, characterID, func(loaded *loadedSave) error {
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
		moved, err = moveOwnedItemToInventoryRecord(
			loaded, locator, ownedItemID, targetPosition, expectedGameID, maxInventory, separateInstances)
		return err
	})
	if err != nil {
		return MoveOwnedItemToInventoryResult{}, err
	}

	return MoveOwnedItemToInventoryResult{
		SaveSessionID:    saveSessionID,
		SaveRevision:     saveRevision,
		OwnedItemID:      ownedItemID,
		CharacterID:      characterID,
		GameID:           moved.gameID,
		Quantity:         moved.quantity,
		ContainerSection: InventorySectionCommon,
		TargetPosition:   targetPosition,
		PhysicalIndex:    moved.physicalIndex,
		AcquisitionIndex: moved.acquisitionIndex,
	}, nil
}

type movedInventoryRecord struct {
	gameID           uint32
	quantity         uint32
	physicalIndex    int
	acquisitionIndex uint32
}

type inventoryOrderRecord struct {
	physicalIndex int
	acquisition   uint32
	moved         bool
}

// moveOwnedItemToInventoryRecord validates the complete plan before applying
// its non-overlapping ranges. The caller holds Engine.mutex.
func moveOwnedItemToInventoryRecord(
	loaded *loadedSave,
	locator ownedItemLocator,
	ownedItemID string,
	targetPosition int,
	expectedGameID uint32,
	maxInventory uint32,
	separateInstances bool,
) (movedInventoryRecord, error) {
	if locator.container != ownedContainerStorage {
		return movedInventoryRecord{}, fmt.Errorf(
			"ownedItemID %q is not in Storage", ownedItemID)
	}
	if locator.containerSection != StorageSectionCommon {
		return movedInventoryRecord{}, fmt.Errorf(
			"ownedItemID %q addresses a Storage key record; moving key records to Inventory is not supported",
			ownedItemID)
	}

	stored, err := readStorageRecords(loaded, locator.characterID)
	if err != nil {
		return movedInventoryRecord{}, err
	}
	var source *StorageRecord
	for index := range stored {
		record := &stored[index]
		if record.ContainerSection == locator.containerSection &&
			record.PhysicalIndex == locator.physicalIndex && record.OwnedItemID == ownedItemID {
			source = record
			break
		}
	}
	if source == nil {
		return movedInventoryRecord{}, fmt.Errorf(
			"ownedItemID %q no longer addresses a record of character %d",
			ownedItemID, locator.characterID)
	}

	byHandle, err := readGaItemMap(loaded.snapshot, loaded.session.platform, locator.characterID)
	if err != nil {
		return movedInventoryRecord{}, fmt.Errorf(
			"cannot resolve items of character %d: %w", locator.characterID, err)
	}
	gameID, err := resolveGaItemHandle(byHandle, source.GaItemHandle)
	if err != nil {
		return movedInventoryRecord{}, fmt.Errorf("ownedItemID %q: %w", ownedItemID, err)
	}
	if gameID != expectedGameID {
		return movedInventoryRecord{}, fmt.Errorf(
			"ownedItemID %q now denotes item 0x%08X, not the expected 0x%08X",
			ownedItemID, gameID, expectedGameID)
	}

	inventory, err := readInventoryRecords(loaded, locator.characterID)
	if err != nil {
		return movedInventoryRecord{}, err
	}
	total := uint64(source.Quantity)
	for _, record := range inventory {
		recordGameID, err := resolveGaItemHandle(byHandle, record.GaItemHandle)
		if err != nil {
			return movedInventoryRecord{}, fmt.Errorf(
				"inventory record %s#%d of character %d: %w",
				record.ContainerSection, record.PhysicalIndex, locator.characterID, err)
		}
		if recordGameID != gameID {
			continue
		}
		if record.ContainerSection == InventorySectionKey {
			return movedInventoryRecord{}, fmt.Errorf(
				"item 0x%08X already holds an Inventory key record of character %d;"+
					" this mutation writes common records only", gameID, locator.characterID)
		}
		if !separateInstances {
			return movedInventoryRecord{}, fmt.Errorf(
				"item 0x%08X already holds a stack record in Inventory; moving duplicate quantity_stack records is not supported",
				gameID)
		}
		total += uint64(record.Quantity)
	}
	if total > uint64(maxInventory) {
		return movedInventoryRecord{}, fmt.Errorf(
			"moving ownedItemID %q would carry %d units of item 0x%08X, above its Inventory limit of %d",
			ownedItemID, total, gameID, maxInventory)
	}
	storageAt, storageCountAt, recordSize, err := ownedItemRowAndCountAt(loaded, locator)
	if err != nil {
		return movedInventoryRecord{}, err
	}
	if recordSize != storageRecordSize {
		return movedInventoryRecord{}, fmt.Errorf(
			"ownedItemID %q addresses a record of unexpected size %d", ownedItemID, recordSize)
	}
	rawSource, err := loaded.snapshot.readAt(storageAt, storageRecordSize)
	if err != nil {
		return movedInventoryRecord{}, fmt.Errorf("cannot read ownedItemID %q: %w", ownedItemID, err)
	}
	if binary.LittleEndian.Uint32(rawSource) != source.GaItemHandle {
		return movedInventoryRecord{}, fmt.Errorf(
			"ownedItemID %q changed while its move was being planned", ownedItemID)
	}

	inventoryAt, err := inventoryHeldSectionAt(loaded, locator.characterID)
	if err != nil {
		return movedInventoryRecord{}, err
	}
	physicalIndex, err := firstFreeInventoryRow(loaded, inventoryAt, locator.characterID)
	if err != nil {
		return movedInventoryRecord{}, err
	}
	inventoryCountAt := inventoryAt - addItemCommonCountBackDistance
	inventoryCount, err := loaded.snapshot.uint32At(inventoryCountAt)
	if err != nil {
		return movedInventoryRecord{}, fmt.Errorf(
			"cannot read the common Inventory count of character %d: %w", locator.characterID, err)
	}
	if inventoryCount >= inventoryHeldCommonRecords {
		return movedInventoryRecord{}, fmt.Errorf(
			"common Inventory of character %d declares %d of %d records and receives no item",
			locator.characterID, inventoryCount, inventoryHeldCommonRecords)
	}
	storageCount, err := loaded.snapshot.uint32At(storageCountAt)
	if err != nil {
		return movedInventoryRecord{}, fmt.Errorf(
			"cannot read the common Storage count of character %d: %w", locator.characterID, err)
	}
	if storageCount == 0 {
		return movedInventoryRecord{}, fmt.Errorf(
			"common Storage count of character %d is zero although ownedItemID %q exists",
			locator.characterID, ownedItemID)
	}

	nextAcquisitionAt := inventoryAt + addItemNextAcquisitionOffset
	storedNextAcquisition, err := loaded.snapshot.uint32At(nextAcquisitionAt)
	if err != nil {
		return movedInventoryRecord{}, fmt.Errorf(
			"cannot read NextAcquisitionSortId of character %d: %w", locator.characterID, err)
	}
	freshAcquisition, err := nextAcquisitionIndex(storedNextAcquisition, locator.characterID)
	if err != nil {
		return movedInventoryRecord{}, err
	}
	ordered, pool, err := inventoryMoveOrder(inventory, targetPosition, freshAcquisition)
	if err != nil {
		return movedInventoryRecord{}, fmt.Errorf("character %d: %w", locator.characterID, err)
	}

	movedRaw := append([]byte(nil), rawSource...)
	movedAcquisition := pool[targetPosition]
	binary.LittleEndian.PutUint32(movedRaw[8:], movedAcquisition)
	writes := []byteWrite{
		{at: storageAt, data: make([]byte, storageRecordSize)},
		{at: storageCountAt, data: littleEndianUint32(storageCount - 1)},
		{at: inventoryAt + int64(physicalIndex)*inventoryHeldRecordSize, data: movedRaw},
		{at: inventoryCountAt, data: littleEndianUint32(inventoryCount + 1)},
		{at: nextAcquisitionAt, data: littleEndianUint32(freshAcquisition + 1)},
	}
	for position, record := range ordered {
		if record.moved || record.acquisition == pool[position] {
			continue
		}
		writes = append(writes, byteWrite{
			at:   inventoryAt + int64(record.physicalIndex)*inventoryHeldRecordSize + 8,
			data: littleEndianUint32(pool[position]),
		})
	}
	if err := applyByteWrites(loaded.snapshot, writes); err != nil {
		return movedInventoryRecord{}, fmt.Errorf("ownedItemID %q: %w", ownedItemID, err)
	}

	return movedInventoryRecord{
		gameID:           gameID,
		quantity:         source.Quantity,
		physicalIndex:    physicalIndex,
		acquisitionIndex: movedAcquisition,
	}, nil
}

// inventoryMoveOrder inserts the moved record into the logical order of common
// Inventory records and reuses their sorted acquisition-index pool plus one
// fresh index. It never changes a physical row.
func inventoryMoveOrder(
	records []InventoryRecord,
	targetPosition int,
	freshAcquisition uint32,
) ([]inventoryOrderRecord, []uint32, error) {
	ordered := make([]inventoryOrderRecord, 0, len(records)+1)
	pool := make([]uint32, 0, len(records)+1)
	seen := make(map[uint32]struct{}, len(records)+1)
	for _, record := range records {
		if _, exists := seen[record.AcquisitionIndex]; exists {
			return nil, nil, fmt.Errorf(
				"Inventory carries duplicate acquisition index %d", record.AcquisitionIndex)
		}
		if record.AcquisitionIndex >= freshAcquisition {
			return nil, nil, fmt.Errorf(
				"NextAcquisitionSortId does not follow Inventory index %d",
				record.AcquisitionIndex)
		}
		seen[record.AcquisitionIndex] = struct{}{}
		if record.ContainerSection != InventorySectionCommon {
			continue
		}
		ordered = append(ordered, inventoryOrderRecord{
			physicalIndex: record.PhysicalIndex,
			acquisition:   record.AcquisitionIndex,
		})
		pool = append(pool, record.AcquisitionIndex)
	}
	if targetPosition < 0 || targetPosition > len(ordered) {
		return nil, nil, fmt.Errorf(
			"targetPosition %d is outside the range 0..%d for common Inventory",
			targetPosition, len(ordered))
	}
	sort.Slice(ordered, func(left, right int) bool {
		return ordered[left].acquisition < ordered[right].acquisition
	})
	ordered = append(ordered, inventoryOrderRecord{})
	copy(ordered[targetPosition+1:], ordered[targetPosition:])
	ordered[targetPosition] = inventoryOrderRecord{moved: true}
	pool = append(pool, freshAcquisition)
	sort.Slice(pool, func(left, right int) bool { return pool[left] < pool[right] })
	return ordered, pool, nil
}
