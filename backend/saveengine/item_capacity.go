package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	ItemCapacityDestinationInventory = "inventory"
	ItemCapacityDestinationStorage   = "storage"

	ItemCapacityLimitPerRecord  = "per_record_quantity"
	ItemCapacityLimitContainer  = "container_quantity"
	ItemCapacityLimitRecords    = "physical_records"
	ItemCapacityLimitAllocator  = "acquisition_allocator"
	ItemCapacityLimitGaItemData = "gaitemdata_entries"
)

// ItemCapacity is a read-only preflight for one common-container item add.
// Empty LimitingFactor means the declared quantity fits every checked budget.
// The result reserves nothing: a later mutation must repeat the checks under
// its own lock and expected revision.
type ItemCapacity struct {
	SaveSessionID             string `json:"saveSessionID"`
	SaveRevision              string `json:"saveRevision"`
	CharacterID               int    `json:"characterID"`
	Active                    bool   `json:"active"`
	Destination               string `json:"destination"`
	GameID                    uint32 `json:"gameID"`
	Quantity                  uint32 `json:"quantity"`
	CanFit                    bool   `json:"canFit"`
	LimitingFactor            string `json:"limitingFactor"`
	CurrentQuantity           uint64 `json:"currentQuantity"`
	QuantityAfter             uint64 `json:"quantityAfter"`
	MaxContainerQuantity      uint32 `json:"maxContainerQuantity"`
	FreePhysicalRecords       int    `json:"freePhysicalRecords"`
	PhysicalRecordsRequired   int    `json:"physicalRecordsRequired"`
	FreeGaItemDataEntries     int    `json:"freeGaItemDataEntries"`
	GaItemDataEntriesRequired int    `json:"gaItemDataEntriesRequired"`
}

// GetItemCapacity reports whether one planned common-container addition fits
// the character's current snapshot. It reads Inventory, Storage, GaItemData and
// the destination allocators directly, mints no OwnedItemID and changes no
// session state.
func (engine *Engine) GetItemCapacity(
	saveSessionID string,
	characterID int,
	destination string,
	gameID uint32,
	quantity uint32,
	separateInstances bool,
	maxPerRecord uint32,
	maxContainerTotal uint32,
) (ItemCapacity, error) {
	if saveSessionID == "" {
		return ItemCapacity{}, errors.New("saveSessionID is required")
	}
	switch destination {
	case ItemCapacityDestinationInventory, ItemCapacityDestinationStorage:
	default:
		return ItemCapacity{}, fmt.Errorf(
			"destination must be %q or %q; got %q",
			ItemCapacityDestinationInventory, ItemCapacityDestinationStorage, destination)
	}
	if quantity == 0 {
		return ItemCapacity{}, errors.New("quantity must be at least 1")
	}
	if maxPerRecord == 0 || maxContainerTotal == 0 {
		return ItemCapacity{}, errors.New("item capacity limits must be positive")
	}
	if separateInstances && quantity != 1 {
		return ItemCapacity{}, fmt.Errorf(
			"a separate-instance item accepts quantity 1, got %d", quantity)
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return ItemCapacity{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return ItemCapacity{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	result := ItemCapacity{
		SaveSessionID:        saveSessionID,
		SaveRevision:         loaded.session.revisionString(),
		CharacterID:          characterID,
		Destination:          destination,
		GameID:               gameID,
		Quantity:             quantity,
		MaxContainerQuantity: maxContainerTotal,
	}
	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return ItemCapacity{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}
	if flag[0] != userData10ActiveFlagValue {
		return result, nil
	}
	result.Active = true

	inventory, inventoryAt, err := capacityInventoryRecords(loaded, characterID)
	if err != nil {
		return ItemCapacity{}, err
	}
	storage, storageAt, err := capacityStorageRecords(loaded, characterID)
	if err != nil {
		return ItemCapacity{}, err
	}
	byHandle, err := readGaItemMap(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return ItemCapacity{}, fmt.Errorf("cannot resolve items of character %d: %w", characterID, err)
	}

	ownedAnywhere := false
	targetQuantity := uint64(0)
	targetRecordQuantity := uint64(0)
	targetFound := false
	consume := func(container, section string, handle, storedQuantity uint32) error {
		itemID, err := resolveGaItemHandle(byHandle, handle)
		if err != nil {
			return err
		}
		if itemID != gameID {
			return nil
		}
		if section == InventorySectionKey || section == StorageSectionKey {
			return fmt.Errorf(
				"item 0x%08X already occupies the unsupported key section of %s", gameID, container)
		}
		ownedAnywhere = true
		if container == destination {
			targetQuantity += uint64(storedQuantity)
			if !targetFound {
				targetRecordQuantity = uint64(storedQuantity)
				targetFound = true
			}
		}
		return nil
	}
	for _, record := range inventory {
		if err := consume(ItemCapacityDestinationInventory, record.ContainerSection,
			record.GaItemHandle, record.Quantity); err != nil {
			return ItemCapacity{}, fmt.Errorf(
				"inventory record %d of character %d: %w", record.PhysicalIndex, characterID, err)
		}
	}
	for _, record := range storage {
		if err := consume(ItemCapacityDestinationStorage, record.ContainerSection,
			record.GaItemHandle, record.Quantity); err != nil {
			return ItemCapacity{}, fmt.Errorf(
				"storage record %d of character %d: %w", record.PhysicalIndex, characterID, err)
		}
	}

	result.CurrentQuantity = targetQuantity
	result.QuantityAfter = targetQuantity + uint64(quantity)

	var commonRecords, freeRecords int
	var declaredCount uint32
	switch destination {
	case ItemCapacityDestinationInventory:
		commonRecords = inventoryHeldCommonRecords
		for _, record := range inventory {
			if record.ContainerSection == InventorySectionCommon {
				freeRecords++
			}
		}
		freeRecords = commonRecords - freeRecords
		declaredCount, err = loaded.snapshot.uint32At(inventoryAt - addItemCommonCountBackDistance)
	case ItemCapacityDestinationStorage:
		commonRecords = storageCommonRecords
		for _, record := range storage {
			if record.ContainerSection == StorageSectionCommon {
				freeRecords++
			}
		}
		freeRecords = commonRecords - freeRecords
		declaredCount, err = loaded.snapshot.uint32At(storageAt)
	}
	if err != nil {
		return ItemCapacity{}, fmt.Errorf(
			"cannot read the common item count of %s for character %d: %w",
			destination, characterID, err)
	}
	if declaredCount > uint32(commonRecords) {
		return ItemCapacity{}, fmt.Errorf(
			"%s of character %d declares %d common records, want at most %d",
			destination, characterID, declaredCount, commonRecords)
	}
	result.FreePhysicalRecords = min(freeRecords, commonRecords-int(declaredCount))
	if separateInstances || !targetFound {
		result.PhysicalRecordsRequired = 1
	}

	freeGaItemData, hasGaItemData, err := capacityGaItemData(loaded, characterID, gameID)
	if err != nil {
		return ItemCapacity{}, err
	}
	result.FreeGaItemDataEntries = freeGaItemData
	if !ownedAnywhere && !hasGaItemData {
		result.GaItemDataEntriesRequired = 1
	}

	switch {
	case !separateInstances && targetRecordQuantity+uint64(quantity) > uint64(maxPerRecord):
		result.LimitingFactor = ItemCapacityLimitPerRecord
	case result.QuantityAfter > uint64(maxContainerTotal):
		result.LimitingFactor = ItemCapacityLimitContainer
	case result.PhysicalRecordsRequired > result.FreePhysicalRecords:
		result.LimitingFactor = ItemCapacityLimitRecords
	}
	if result.LimitingFactor == "" && result.PhysicalRecordsRequired > 0 {
		available, err := capacityAllocatorAvailable(
			loaded, characterID, destination, inventoryAt, storageAt, storage)
		if err != nil {
			return ItemCapacity{}, err
		}
		if !available {
			result.LimitingFactor = ItemCapacityLimitAllocator
		}
	}
	if result.LimitingFactor == "" &&
		result.GaItemDataEntriesRequired > result.FreeGaItemDataEntries {
		result.LimitingFactor = ItemCapacityLimitGaItemData
	}
	result.CanFit = result.LimitingFactor == ""
	return result, nil
}

func capacityInventoryRecords(
	loaded *loadedSave, characterID int,
) ([]InventoryRecord, int64, error) {
	sectionAt, err := inventoryHeldSectionAt(loaded, characterID)
	if err != nil {
		return nil, 0, err
	}
	section, err := loaded.snapshot.readAt(sectionAt, inventoryHeldSectionSize)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot read inventory of character %d: %w", characterID, err)
	}
	keyEnd := inventoryHeldSectionSize - inventoryHeldTrailingCounters
	records := appendInventoryRecords(nil, section[:inventoryHeldCommonSize], InventorySectionCommon)
	records = appendInventoryRecords(records, section[inventoryHeldKeyAt:keyEnd], InventorySectionKey)
	return records, sectionAt, nil
}

func capacityStorageRecords(
	loaded *loadedSave, characterID int,
) ([]StorageRecord, int64, error) {
	sectionAt, err := storageBoxSectionAt(loaded, characterID)
	if err != nil {
		return nil, 0, err
	}
	section, err := loaded.snapshot.readAt(sectionAt, storageSectionSize)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot read storage of character %d: %w", characterID, err)
	}
	records := appendStorageRecords(nil,
		section[storageCommonAt:storageCommonAt+storageCommonSize], StorageSectionCommon)
	records = appendStorageRecords(records,
		section[storageKeyAt:storageKeyAt+storageKeySize], StorageSectionKey)
	return records, sectionAt, nil
}

func capacityGaItemData(
	loaded *loadedSave, characterID int, gameID uint32,
) (int, bool, error) {
	sectionAt, slotEnd, err := eventFlagGaItemGameDataAt(loaded, characterID)
	if err != nil {
		return 0, false, err
	}
	if sectionAt+eventFlagGaItemGameDataSize > slotEnd {
		return 0, false, fmt.Errorf("GaItemData of character %d does not fit into its slot", characterID)
	}
	declared, err := loaded.snapshot.uint32At(sectionAt)
	if err != nil {
		return 0, false, fmt.Errorf("cannot read the GaItemData count of character %d: %w",
			characterID, err)
	}
	if declared > gaItemDataMaxCount {
		return 0, false, fmt.Errorf(
			"character %d declares %d active GaItemData entries, want at most %d",
			characterID, int32(declared), gaItemDataMaxCount)
	}
	active, err := loaded.snapshot.readAt(
		sectionAt+gaItemDataArrayOffset, int(declared)*gaItemDataActiveEntrySize)
	if err != nil {
		return 0, false, fmt.Errorf("cannot read the GaItemData entries of character %d: %w",
			characterID, err)
	}
	for index := 0; index < int(declared); index++ {
		if binary.LittleEndian.Uint32(active[index*gaItemDataActiveEntrySize:]) == gameID {
			return gaItemDataMaxCount - int(declared), true, nil
		}
	}
	return gaItemDataMaxCount - int(declared), false, nil
}

func capacityAllocatorAvailable(
	loaded *loadedSave,
	characterID int,
	destination string,
	inventoryAt int64,
	storageAt int64,
	storage []StorageRecord,
) (bool, error) {
	switch destination {
	case ItemCapacityDestinationInventory:
		equip, err := loaded.snapshot.uint32At(inventoryAt + addItemNextEquipIndexOffset)
		if err != nil {
			return false, fmt.Errorf("cannot read NextEquipIndex of character %d: %w", characterID, err)
		}
		if equip == ^uint32(0) {
			return false, nil
		}
		next, err := loaded.snapshot.uint32At(inventoryAt + addItemNextAcquisitionOffset)
		if err != nil {
			return false, fmt.Errorf(
				"cannot read NextAcquisitionSortId of character %d: %w", characterID, err)
		}
		_, err = nextAcquisitionIndex(next, characterID)
		return err == nil, nil
	case ItemCapacityDestinationStorage:
		// NextEquipIndex is bounded by the physical section and can never block a
		// deposit, so only the acquisition allocator is probed here.
		countersAt := storageAt + storageKeyAt + storageKeySize
		storedNext, err := loaded.snapshot.uint32At(countersAt + 4)
		if err != nil {
			return false, fmt.Errorf(
				"cannot read Storage NextAcquisitionSortId of character %d: %w", characterID, err)
		}
		_, _, err = nextStorageAcquisitionAndCounters(storedNext, storage, characterID)
		return err == nil, nil
	default:
		return false, nil
	}
}
