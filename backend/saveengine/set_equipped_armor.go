package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	equippedArmorFirstSlot = 12
	equippedArmorSlotCount = 4
)

var equippedArmorEmptyGameIDs = [equippedArmorSlotCount]uint32{
	0x10002710,
	0x10002774,
	0x100027D8,
	0x1000283C,
}

// SetEquippedArmorResult reports one committed four-slot armor assignment.
// A zero GameID means the corresponding public slot was cleared.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type SetEquippedArmorResult struct {
	MutationReceipt
	CharacterID int                            `json:"characterID"`
	GameIDs     [equippedArmorSlotCount]uint32 `json:"gameIDs"`
}

// SetEquippedArmor atomically replaces the head, chest, arms and legs slots.
// Each non-nil assignment is an opaque OwnedItemID from Inventory common. A
// nil assignment selects the existing native bare-body record for that slot;
// this operation never allocates or repacks the GaItem table.
func (engine *Engine) SetEquippedArmor(
	saveSessionID string,
	characterID int,
	slotAssignments [equippedArmorSlotCount]*string,
	expectedRevision string,
	validateGameID func(slot int, gameID uint32) error,
) (SetEquippedArmorResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetEquippedArmorResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	var targetGameIDs [equippedArmorSlotCount]uint32
	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetEquippedArmor, characterID, func(loaded *loadedSave) error {
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

		flag, err := loaded.snapshot.readAt(
			userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
		}
		if flag[0] != userData10ActiveFlagValue {
			return fmt.Errorf("character %d is not active", characterID)
		}

		inventoryAt, err := inventoryHeldSectionAt(loaded, characterID)
		if err != nil {
			return err
		}
		inventoryRows, err := loaded.snapshot.readAt(inventoryAt, inventoryHeldCommonSize)
		if err != nil {
			return fmt.Errorf("cannot read Inventory common of character %d: %w", characterID, err)
		}
		anchor := inventoryAt - inventoryHeldCommonOffset
		_, slotEnd := inventorySlotBounds(loaded.session.platform, characterID)
		byHandle, err := readGaItemMap(loaded.snapshot, loaded.session.platform, characterID)
		if err != nil {
			return fmt.Errorf("cannot resolve armor of character %d: %w", characterID, err)
		}

		countAt := anchor + equipmentProjectileCountOffset
		if countAt+4 > slotEnd {
			return fmt.Errorf("projectile count of character %d lies outside its slot", characterID)
		}
		rawCount, err := loaded.snapshot.readAt(countAt, 4)
		if err != nil {
			return fmt.Errorf("cannot read projectile count of character %d: %w", characterID, err)
		}
		projectileCount := int64(binary.LittleEndian.Uint32(rawCount))
		if projectileCount > equipmentMaxProjectileRecords {
			return fmt.Errorf("character %d declares %d projectile records, want at most %d",
				characterID, projectileCount, equipmentMaxProjectileRecords)
		}
		armamentsAt := countAt + 4 + projectileCount*equipmentProjectileRecordSize
		if armamentsAt+equipmentBlockSize > slotEnd {
			return fmt.Errorf("equipment block of character %d does not fit into its slot", characterID)
		}

		fieldOffset := int64(equippedArmorFirstSlot * 4)
		ranges := [4]int64{
			anchor + removeEquipmentIndexesOffset + fieldOffset,
			anchor + removeEquipmentIndexesOffset + equipmentSlotCount*4 + 0x1C + fieldOffset,
			anchor + removeEquipmentHandlesOffset + fieldOffset,
			armamentsAt + fieldOffset,
		}
		var before [4][]byte
		for index, at := range ranges {
			before[index], err = loaded.snapshot.readAt(at, equippedArmorSlotCount*4)
			if err != nil {
				return fmt.Errorf("cannot read armor representation %d of character %d: %w",
					index+1, characterID, err)
			}
		}
		if err := validateExistingArmorFields(
			inventoryRows, byHandle, before, characterID); err != nil {
			return err
		}

		var after [4][]byte
		for index := range after {
			after[index] = make([]byte, equippedArmorSlotCount*4)
		}
		seenHandles := make(map[uint32]int, equippedArmorSlotCount)
		for slot, assignment := range slotAssignments {
			physicalIndex, handle, gameID, err := resolveArmorAssignment(
				loaded, inventoryRows, byHandle, characterID, slot, assignment)
			if err != nil {
				return err
			}
			if assignment != nil {
				if validateGameID != nil {
					if err := validateGameID(slot, gameID); err != nil {
						return fmt.Errorf("slotAssignments[%d]: %w", slot, err)
					}
				}
				targetGameIDs[slot] = gameID
			}
			if previous, duplicate := seenHandles[handle]; duplicate {
				return fmt.Errorf(
					"owned armor record 0x%08X is assigned to both slot %d and slot %d",
					handle, previous, slot)
			}
			seenHandles[handle] = slot

			binary.LittleEndian.PutUint32(after[0][slot*4:],
				removeReferenceInventoryRowBase+uint32(physicalIndex))
			binary.LittleEndian.PutUint32(after[1][slot*4:], gameID&0x0FFFFFFF)
			binary.LittleEndian.PutUint32(after[2][slot*4:], handle)
			binary.LittleEndian.PutUint32(after[3][slot*4:], gameID)
		}

		unchanged := true
		for index := range before {
			unchanged = unchanged && bytes.Equal(before[index], after[index])
		}
		if unchanged {
			return nil
		}
		return writeEquippedArmorFields(loaded.snapshot, ranges, before, after, characterID)
	})
	if err != nil {
		return SetEquippedArmorResult{}, err
	}

	return SetEquippedArmorResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		GameIDs:         targetGameIDs,
	}, nil
}

func validateExistingArmorFields(
	inventoryRows []byte,
	byHandle map[uint32]uint32,
	fields [4][]byte,
	characterID int,
) error {
	for slot := 0; slot < equippedArmorSlotCount; slot++ {
		row := binary.LittleEndian.Uint32(fields[0][slot*4:])
		bare := binary.LittleEndian.Uint32(fields[1][slot*4:])
		handle := binary.LittleEndian.Uint32(fields[2][slot*4:])
		gameID := binary.LittleEndian.Uint32(fields[3][slot*4:])
		if row < removeReferenceInventoryRowBase || handle&gaItemHandleTypeMask != gaItemArmorHandle {
			return fmt.Errorf("armor slot %d of character %d has inconsistent existing representations",
				slot, characterID)
		}
		physicalIndex := int(row - removeReferenceInventoryRowBase)
		if physicalIndex >= inventoryHeldCommonRecords {
			return fmt.Errorf("armor slot %d of character %d addresses inventory row %d out of range",
				slot, characterID, physicalIndex)
		}
		inventoryRow := inventoryRows[physicalIndex*inventoryHeldRecordSize:]
		resolved, err := resolveGaItemHandle(byHandle, handle)
		if err != nil || binary.LittleEndian.Uint32(inventoryRow) != handle ||
			binary.LittleEndian.Uint32(inventoryRow[4:])&inventoryHeldQuantityMask == 0 ||
			resolved&gaItemHandleTypeMask != 0x10000000 ||
			bare != resolved&0x0FFFFFFF || gameID != resolved {
			return fmt.Errorf("armor slot %d of character %d has inconsistent existing representations",
				slot, characterID)
		}
	}
	return nil
}

func resolveArmorAssignment(
	loaded *loadedSave,
	inventoryRows []byte,
	byHandle map[uint32]uint32,
	characterID int,
	slot int,
	assignment *string,
) (physicalIndex int, handle uint32, gameID uint32, err error) {
	if assignment == nil {
		physicalIndex, handle, err = findNativeEmptyArmorRecord(
			inventoryRows, byHandle, equippedArmorEmptyGameIDs[slot])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("slotAssignments[%d]: %w", slot, err)
		}
		return physicalIndex, handle, equippedArmorEmptyGameIDs[slot], nil
	}

	token := *assignment
	if token == "" {
		return 0, 0, 0, fmt.Errorf("slotAssignments[%d] is empty string", slot)
	}
	locator, err := loaded.session.resolveOwnedItemID(characterID, token)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("slotAssignments[%d]: %w", slot, err)
	}
	if locator.container != ownedContainerInventory ||
		locator.containerSection != InventorySectionCommon {
		return 0, 0, 0, fmt.Errorf(
			"slotAssignments[%d]: ownedItemID %q must address Inventory common", slot, token)
	}

	row := inventoryRows[locator.physicalIndex*inventoryHeldRecordSize:]
	handle = binary.LittleEndian.Uint32(row)
	quantity := binary.LittleEndian.Uint32(row[4:]) & inventoryHeldQuantityMask
	if handle&gaItemHandleTypeMask != gaItemArmorHandle || quantity == 0 {
		return 0, 0, 0, fmt.Errorf(
			"slotAssignments[%d]: ownedItemID %q is not a positive-quantity armor record",
			slot, token)
	}
	gameID, err = resolveGaItemHandle(byHandle, handle)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("slotAssignments[%d] ownedItemID %q: %w", slot, token, err)
	}
	return locator.physicalIndex, handle, gameID, nil
}

func findNativeEmptyArmorRecord(
	inventoryRows []byte,
	byHandle map[uint32]uint32,
	wantedGameID uint32,
) (int, uint32, error) {
	for physicalIndex := 0; physicalIndex < inventoryHeldCommonRecords; physicalIndex++ {
		row := inventoryRows[physicalIndex*inventoryHeldRecordSize:]
		handle := binary.LittleEndian.Uint32(row)
		quantity := binary.LittleEndian.Uint32(row[4:]) & inventoryHeldQuantityMask
		if quantity == 0 || handle&gaItemHandleTypeMask != gaItemArmorHandle {
			continue
		}
		gameID, err := resolveGaItemHandle(byHandle, handle)
		if err == nil && gameID == wantedGameID {
			return physicalIndex, handle, nil
		}
	}
	return 0, 0, fmt.Errorf(
		"native empty armor item 0x%08X is not present in Inventory common; GaItem allocation is unsupported",
		wantedGameID)
}

func writeEquippedArmorFields(
	snapshot *codec,
	ranges [4]int64,
	before, after [4][]byte,
	characterID int,
) error {
	written := 0
	for index, at := range ranges {
		if err := snapshot.writeAt(at, after[index]); err != nil {
			for rollback := 0; rollback < written; rollback++ {
				if rollbackErr := snapshot.writeAt(ranges[rollback], before[rollback]); rollbackErr != nil {
					return fmt.Errorf(
						"cannot write armor representation %d of character %d; rollback failed: %v: %w",
						index+1, characterID, rollbackErr, err)
				}
			}
			return fmt.Errorf("cannot write armor representation %d of character %d: %w",
				index+1, characterID, err)
		}
		written++
	}

	for index, at := range ranges {
		stored, err := snapshot.readAt(at, len(after[index]))
		if err == nil && bytes.Equal(stored, after[index]) {
			continue
		}
		var rollbackErrors []error
		for rollback, rollbackAt := range ranges {
			if rollbackErr := snapshot.writeAt(rollbackAt, before[rollback]); rollbackErr != nil {
				rollbackErrors = append(rollbackErrors, rollbackErr)
			}
		}
		if len(rollbackErrors) != 0 {
			return fmt.Errorf(
				"armor representation %d of character %d could not be verified and rollback failed: %v",
				index+1, characterID, rollbackErrors)
		}
		return errors.New("armor mutation could not be verified; the save is unchanged")
	}
	return nil
}
