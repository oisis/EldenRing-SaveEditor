package saveengine

import (
	"encoding/binary"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// AddItemToStorageResult reports one committed common-Storage add. The receipt
// carries physical coordinates instead of an OwnedItemID because committing the
// mutation retires every identity minted under the previous revision.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type AddItemToStorageResult struct {
	MutationReceipt
	CharacterID      int    `json:"characterID"`
	GameID           uint32 `json:"gameID"`
	Added            uint32 `json:"added"`
	Quantity         uint32 `json:"quantity"`
	CreatedRecord    bool   `json:"createdRecord"`
	ContainerSection string `json:"containerSection"`
	PhysicalIndex    int    `json:"physicalIndex"`
}

type addedStorageRecord struct {
	physicalIndex int
	quantity      uint32
	created       bool
}

// AddItemToStorage adds quantity of gameID to common Storage in one revision.
// quantity is a delta, never a target total. The caller owns catalog decisions;
// SaveEngine owns the complete physical preflight, write plan and rollback.
func (engine *Engine) AddItemToStorage(
	saveSessionID string,
	characterID int,
	gameID uint32,
	quantity uint32,
	expectedRevision string,
	separateInstances bool,
	maxStorage uint32,
) (AddItemToStorageResult, error) {
	if quantity == 0 {
		return AddItemToStorageResult{}, fmt.Errorf(
			"quantity must be at least 1; it is the amount added, not a target total")
	}
	if quantity > ^ownedItemQuantityFlag {
		return AddItemToStorageResult{}, fmt.Errorf(
			"quantity %d exceeds the %d the record can store", quantity, ^ownedItemQuantityFlag)
	}
	if separateInstances && quantity != 1 {
		return AddItemToStorageResult{}, fmt.Errorf(
			"item 0x%08X stores every copy in its own record, so quantity must be 1; got %d",
			gameID, quantity)
	}
	if maxStorage == 0 {
		return AddItemToStorageResult{}, fmt.Errorf(
			"maxStorage must be at least 1")
	}
	if quantity > maxStorage {
		return AddItemToStorageResult{}, fmt.Errorf(
			"quantity %d exceeds the limit of %d for Storage", quantity, maxStorage)
	}
	if !isCanonicalRevision(expectedRevision) {
		return AddItemToStorageResult{}, apperror.InvalidRevision(expectedRevision)
	}

	var outcome addedStorageRecord
	committed, err := engine.commitCharacterRevision(saveSessionID, kindAddItemToStorage, characterID, func(loaded *loadedSave) error {
		if characterID < 0 || characterID >= characterSlotCount {
			return fmt.Errorf("characterID %d is outside the range 0..%d",
				characterID, characterSlotCount-1)
		}
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return apperror.RevisionConflict(expectedRevision, current)
		}

		var err error
		outcome, err = addItemToStorageRecord(
			loaded, characterID, gameID, quantity, separateInstances, maxStorage)
		return err
	})
	if err != nil {
		return AddItemToStorageResult{}, err
	}

	return AddItemToStorageResult{
		MutationReceipt:  committed,
		CharacterID:      characterID,
		GameID:           gameID,
		Added:            quantity,
		Quantity:         outcome.quantity,
		CreatedRecord:    outcome.created,
		ContainerSection: StorageSectionCommon,
		PhysicalIndex:    outcome.physicalIndex,
	}, nil
}

// addItemToStorageRecord validates every destination and ownership fact before
// choosing the top-up or new-record plan. The caller holds Engine.mutex.
func addItemToStorageRecord(
	loaded *loadedSave,
	characterID int,
	gameID uint32,
	quantity uint32,
	separateInstances bool,
	maxStorage uint32,
) (addedStorageRecord, error) {
	handle, err := gaItemHandleForGameID(gameID)
	if err != nil {
		return addedStorageRecord{}, err
	}
	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return addedStorageRecord{}, fmt.Errorf(
			"cannot read activity of character %d: %w", characterID, err)
	}
	if flag[0] != userData10ActiveFlagValue {
		return addedStorageRecord{}, fmt.Errorf(
			"character %d is not active and receives no item", characterID)
	}

	inventory, err := readInventoryRecords(loaded, characterID)
	if err != nil {
		return addedStorageRecord{}, err
	}
	storage, err := readStorageRecords(loaded, characterID)
	if err != nil {
		return addedStorageRecord{}, err
	}
	byHandle, err := readGaItemMap(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return addedStorageRecord{}, fmt.Errorf(
			"cannot resolve items of character %d: %w", characterID, err)
	}

	owned := false
	for _, record := range inventory {
		itemID, err := resolveGaItemHandle(byHandle, record.GaItemHandle)
		if err != nil {
			return addedStorageRecord{}, fmt.Errorf(
				"inventory record %d of character %d: %w", record.PhysicalIndex, characterID, err)
		}
		if itemID != gameID {
			continue
		}
		if record.ContainerSection == InventorySectionKey {
			return addedStorageRecord{}, fmt.Errorf(
				"item 0x%08X already holds an Inventory key record of character %d, and this"+
					" mutation writes common records only", gameID, characterID)
		}
		owned = true
	}

	target := -1
	total := uint64(quantity)
	stackRecordCount := 0
	for index, record := range storage {
		itemID, err := resolveGaItemHandle(byHandle, record.GaItemHandle)
		if err != nil {
			return addedStorageRecord{}, fmt.Errorf(
				"storage record %d of character %d: %w", record.PhysicalIndex, characterID, err)
		}
		if itemID != gameID {
			continue
		}
		if record.ContainerSection == StorageSectionKey {
			return addedStorageRecord{}, fmt.Errorf(
				"item 0x%08X already holds a Storage key record of character %d, and this"+
					" mutation writes common records only", gameID, characterID)
		}
		owned = true
		total += uint64(record.Quantity)
		if !separateInstances {
			stackRecordCount++
			if target < 0 {
				target = index
			}
		}
	}
	if !separateInstances && stackRecordCount > 1 {
		return addedStorageRecord{}, fmt.Errorf(
			"item 0x%08X already holds %d duplicate records in Storage; adding to duplicate quantity_stack records is not supported",
			gameID, stackRecordCount)
	}
	if total > uint64(maxStorage) {
		return addedStorageRecord{}, fmt.Errorf(
			"adding %d would raise the Storage total of item 0x%08X to %d, above the limit of %d",
			quantity, gameID, total, maxStorage)
	}
	if target >= 0 {
		return topUpStorageRecord(
			loaded, characterID, storage[target], gameID, quantity, maxStorage, storage)
	}
	return createStorageRecord(loaded, characterID, handle, gameID, quantity, owned, storage)
}

func topUpStorageRecord(
	loaded *loadedSave,
	characterID int,
	record StorageRecord,
	gameID uint32,
	quantity uint32,
	maxStorage uint32,
	records []StorageRecord,
) (addedStorageRecord, error) {
	stacked := uint64(record.Quantity) + uint64(quantity)
	if stacked > uint64(maxStorage) {
		return addedStorageRecord{}, fmt.Errorf(
			"adding %d to the %d item 0x%08X already stores would store %d, above the limit"+
				" of %d per record", quantity, record.Quantity, gameID, stacked, maxStorage)
	}
	sectionAt, err := storageBoxSectionAt(loaded, characterID)
	if err != nil {
		return addedStorageRecord{}, err
	}
	recordAt := sectionAt + storageCommonAt + int64(record.PhysicalIndex)*storageRecordSize
	quantityAt := recordAt + 4
	raw, err := loaded.snapshot.uint32At(quantityAt)
	if err != nil {
		return addedStorageRecord{}, fmt.Errorf(
			"cannot read the quantity of Storage record %d of character %d: %w",
			record.PhysicalIndex, characterID, err)
	}
	countersAt := sectionAt + storageKeyAt + storageKeySize
	storedNext, err := loaded.snapshot.uint32At(countersAt + 4)
	if err != nil {
		return addedStorageRecord{}, fmt.Errorf(
			"cannot read Storage NextAcquisitionSortId of character %d: %w", characterID, err)
	}
	acquisitionIndex, nextAcquisition, err := nextStorageAcquisitionAndCounters(
		storedNext, records, characterID)
	if err != nil {
		return addedStorageRecord{}, err
	}
	updated := uint32(stacked)
	writes := []byteWrite{
		{at: quantityAt, data: littleEndianUint32((raw & ownedItemQuantityFlag) | updated)},
		{at: recordAt + 8, data: littleEndianUint32(acquisitionIndex)},
		{at: countersAt + 4, data: littleEndianUint32(nextAcquisition)},
	}
	if err := applyByteWrites(loaded.snapshot, writes); err != nil {
		return addedStorageRecord{}, fmt.Errorf("item 0x%08X: %w", gameID, err)
	}
	return addedStorageRecord{physicalIndex: record.PhysicalIndex, quantity: updated}, nil
}

func createStorageRecord(
	loaded *loadedSave,
	characterID int,
	handle uint32,
	gameID uint32,
	quantity uint32,
	owned bool,
	records []StorageRecord,
) (addedStorageRecord, error) {
	sectionAt, err := storageBoxSectionAt(loaded, characterID)
	if err != nil {
		return addedStorageRecord{}, err
	}
	common, err := loaded.snapshot.readAt(sectionAt+storageCommonAt, storageCommonSize)
	if err != nil {
		return addedStorageRecord{}, fmt.Errorf(
			"cannot read common Storage of character %d: %w", characterID, err)
	}
	row := -1
	occupied := 0
	for index := 0; index < storageCommonRecords; index++ {
		stored := binary.LittleEndian.Uint32(common[index*storageRecordSize:])
		if stored == storageEmptyHandle || stored == storageInvalidHandle {
			if row < 0 {
				row = index
			}
			continue
		}
		occupied++
	}
	if row < 0 {
		return addedStorageRecord{}, fmt.Errorf(
			"the common Storage section of character %d holds no free record", characterID)
	}
	declared, err := loaded.snapshot.uint32At(sectionAt)
	if err != nil {
		return addedStorageRecord{}, fmt.Errorf(
			"cannot read the common Storage count of character %d: %w", characterID, err)
	}
	if declared >= storageCommonRecords {
		return addedStorageRecord{}, fmt.Errorf(
			"Storage of character %d declares %d of %d common records and receives no item",
			characterID, declared, storageCommonRecords)
	}

	countersAt := sectionAt + storageKeyAt + storageKeySize
	storedNext, err := loaded.snapshot.uint32At(countersAt + 4)
	if err != nil {
		return addedStorageRecord{}, fmt.Errorf(
			"cannot read Storage NextAcquisitionSortId of character %d: %w", characterID, err)
	}

	acquisitionIndex, nextAcquisition, err := nextStorageAcquisitionAndCounters(
		storedNext, records, characterID)
	if err != nil {
		return addedStorageRecord{}, err
	}
	// The new record lands in row, so the layout behind the planned write is the
	// one that was read plus that row; the stored counter is never carried over.
	updatedEquip := storageNextEquipIndex(common, row)

	var gaItemData []byteWrite
	if !owned {
		gaItemData, err = planGaItemDataInsertion(loaded, characterID, gameID)
		if err != nil {
			return addedStorageRecord{}, err
		}
	}
	record := make([]byte, storageRecordSize)
	copy(record, littleEndianUint32(handle))
	copy(record[4:], littleEndianUint32(quantity))
	copy(record[8:], littleEndianUint32(acquisitionIndex))
	writes := append([]byteWrite{
		{at: sectionAt + storageCommonAt + int64(row)*storageRecordSize, data: record},
		{at: sectionAt, data: littleEndianUint32(declared + 1)},
		{at: countersAt, data: littleEndianUint32(updatedEquip)},
		{at: countersAt + 4, data: littleEndianUint32(nextAcquisition)},
	}, gaItemData...)
	if err := applyByteWrites(loaded.snapshot, writes); err != nil {
		return addedStorageRecord{}, fmt.Errorf("item 0x%08X: %w", gameID, err)
	}
	return addedStorageRecord{physicalIndex: row, quantity: quantity, created: true}, nil
}
