package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	equippedArmamentSlotCount = 6
	unarmedEquipmentGameID    = uint32(0x0001ADB0)
)

// SetEquippedArmamentsResult reports one committed six-slot hand-armament assignment.
// A zero GameID means the corresponding public slot was cleared.
type SetEquippedArmamentsResult struct {
	SaveSessionID string                            `json:"saveSessionID"`
	SaveRevision  string                            `json:"saveRevision"`
	CharacterID   int                               `json:"characterID"`
	GameIDs       [equippedArmamentSlotCount]uint32 `json:"gameIDs"`
}

// SetEquippedArmaments atomically replaces left 1, right 1, left 2, right 2,
// left 3 and right 3 in stored order. Each non-nil assignment is an opaque
// OwnedItemID from Inventory common. A nil assignment selects the existing
// native Unarmed record; this operation never allocates or repacks GaItem.
func (engine *Engine) SetEquippedArmaments(
	saveSessionID string,
	characterID int,
	slotAssignments [equippedArmamentSlotCount]*string,
	expectedRevision string,
	validateGameID func(slot int, gameID uint32) error,
) (SetEquippedArmamentsResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetEquippedArmamentsResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	var targetGameIDs [equippedArmamentSlotCount]uint32
	saveRevision, err := engine.commitCharacterRevision(saveSessionID, opSetEquippedArmaments, characterID, func(loaded *loadedSave) error {
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
			return fmt.Errorf("cannot resolve armaments of character %d: %w", characterID, err)
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

		ranges := [4]int64{
			anchor + removeEquipmentIndexesOffset,
			anchor + removeEquipmentIndexesOffset + equipmentSlotCount*4 + 0x1C,
			anchor + removeEquipmentHandlesOffset,
			armamentsAt,
		}
		var before [4][]byte
		for index, at := range ranges {
			before[index], err = loaded.snapshot.readAt(at, equippedArmamentSlotCount*4)
			if err != nil {
				return fmt.Errorf("cannot read armament representation %d of character %d: %w",
					index+1, characterID, err)
			}
		}
		if err := validateExistingArmamentFields(
			inventoryRows, byHandle, before, characterID); err != nil {
			return err
		}

		var after [4][]byte
		for index := range after {
			after[index] = make([]byte, equippedArmamentSlotCount*4)
		}
		seenHandles := make(map[uint32]int, equippedArmamentSlotCount)
		for slot, assignment := range slotAssignments {
			physicalIndex, handle, gameID, err := resolveArmamentAssignment(
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
				if previous, duplicate := seenHandles[handle]; duplicate {
					return fmt.Errorf(
						"owned weapon record 0x%08X is assigned to both slot %d and slot %d",
						handle, previous, slot)
				}
				seenHandles[handle] = slot
			}

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
		return writeEquippedArmamentFields(loaded.snapshot, ranges, before, after, characterID)
	})
	if err != nil {
		return SetEquippedArmamentsResult{}, err
	}

	return SetEquippedArmamentsResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  saveRevision,
		CharacterID:   characterID,
		GameIDs:       targetGameIDs,
	}, nil
}

func validateExistingArmamentFields(
	inventoryRows []byte,
	byHandle map[uint32]uint32,
	fields [4][]byte,
	characterID int,
) error {
	for slot := 0; slot < equippedArmamentSlotCount; slot++ {
		row := binary.LittleEndian.Uint32(fields[0][slot*4:])
		bare := binary.LittleEndian.Uint32(fields[1][slot*4:])
		handle := binary.LittleEndian.Uint32(fields[2][slot*4:])
		gameID := binary.LittleEndian.Uint32(fields[3][slot*4:])
		if row < removeReferenceInventoryRowBase ||
			handle&gaItemHandleTypeMask != gaItemWeaponHandle {
			return fmt.Errorf(
				"armament slot %d of character %d has inconsistent existing representations",
				slot, characterID)
		}
		physicalIndex := int(row - removeReferenceInventoryRowBase)
		if physicalIndex >= inventoryHeldCommonRecords {
			return fmt.Errorf("armament slot %d of character %d addresses inventory row %d out of range",
				slot, characterID, physicalIndex)
		}
		inventoryRow := inventoryRows[physicalIndex*inventoryHeldRecordSize:]
		if binary.LittleEndian.Uint32(inventoryRow) != handle {
			return fmt.Errorf("armament slot %d of character %d does not match its inventory row",
				slot, characterID)
		}
		if binary.LittleEndian.Uint32(inventoryRow[4:])&inventoryHeldQuantityMask == 0 {
			return fmt.Errorf("armament slot %d of character %d references a zero-quantity record",
				slot, characterID)
		}
		resolved, err := resolveGaItemHandle(byHandle, handle)
		if err != nil {
			return fmt.Errorf("armament slot %d of character %d cannot resolve its handle: %w",
				slot, characterID, err)
		}
		if resolved&gaItemHandleTypeMask != 0 || bare != resolved || gameID != resolved {
			return fmt.Errorf(
				"armament slot %d of character %d has inconsistent item ID representations",
				slot, characterID)
		}
	}
	return nil
}

func resolveArmamentAssignment(
	loaded *loadedSave,
	inventoryRows []byte,
	byHandle map[uint32]uint32,
	characterID int,
	slot int,
	assignment *string,
) (physicalIndex int, handle uint32, gameID uint32, err error) {
	if assignment == nil {
		physicalIndex, handle, err = findNativeUnarmedRecord(inventoryRows, byHandle)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("slotAssignments[%d]: %w", slot, err)
		}
		return physicalIndex, handle, unarmedEquipmentGameID, nil
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
	if handle&gaItemHandleTypeMask != gaItemWeaponHandle || quantity == 0 {
		return 0, 0, 0, fmt.Errorf(
			"slotAssignments[%d]: ownedItemID %q is not a positive-quantity weapon record",
			slot, token)
	}
	gameID, err = resolveGaItemHandle(byHandle, handle)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("slotAssignments[%d] ownedItemID %q: %w", slot, token, err)
	}
	if gameID&gaItemHandleTypeMask != 0 {
		return 0, 0, 0, fmt.Errorf(
			"slotAssignments[%d]: ownedItemID %q resolves to non-weapon game ID 0x%08X",
			slot, token, gameID)
	}
	return locator.physicalIndex, handle, gameID, nil
}

func findNativeUnarmedRecord(
	inventoryRows []byte,
	byHandle map[uint32]uint32,
) (int, uint32, error) {
	for physicalIndex := 0; physicalIndex < inventoryHeldCommonRecords; physicalIndex++ {
		row := inventoryRows[physicalIndex*inventoryHeldRecordSize:]
		handle := binary.LittleEndian.Uint32(row)
		quantity := binary.LittleEndian.Uint32(row[4:]) & inventoryHeldQuantityMask
		if quantity == 0 || handle&gaItemHandleTypeMask != gaItemWeaponHandle {
			continue
		}
		gameID, err := resolveGaItemHandle(byHandle, handle)
		if err == nil && gameID == unarmedEquipmentGameID {
			return physicalIndex, handle, nil
		}
	}
	return 0, 0, fmt.Errorf(
		"native Unarmed item 0x%08X is not present in Inventory common; GaItem allocation is unsupported",
		unarmedEquipmentGameID)
}

func writeEquippedArmamentFields(
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
						"cannot write armament representation %d of character %d; rollback failed: %v: %w",
						index+1, characterID, rollbackErr, err)
				}
			}
			return fmt.Errorf("cannot write armament representation %d of character %d: %w",
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
				"armament representation %d of character %d could not be verified and rollback failed: %v",
				index+1, characterID, rollbackErrors)
		}
		return errors.New("armament mutation could not be verified; the save is unchanged")
	}
	return nil
}
