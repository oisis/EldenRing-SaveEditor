package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// QuickItemEmptyGameID is the confirmed native value of one empty Quick
	// Items position in the equipped-armaments tail.
	QuickItemEmptyGameID     uint32 = 0xFFFFFFFF
	quickItemEmptyItemID     uint32 = 0x00000000
	quickItemEmptyEquipIndex uint32 = 0xFFFFFFFF
	quickItemsTailOffset            = equipmentBlockSize
	quickItemsTailSize              = quickItemSlotCount * 4
)

// SetQuickItemsResult reports one committed ten-slot Quick Items assignment.
type SetQuickItemsResult struct {
	SaveSessionID string                     `json:"saveSessionID"`
	SaveRevision  string                     `json:"saveRevision"`
	CharacterID   int                        `json:"characterID"`
	GameIDs       [quickItemSlotCount]uint32 `json:"gameIDs"`
}

// SetQuickItems atomically replaces the ten Quick Items positions of one character.
func (engine *Engine) SetQuickItems(
	saveSessionID string,
	characterID int,
	slotAssignments [quickItemSlotCount]*string,
	expectedRevision string,
	validateGameID func(gameID uint32) error,
) (SetQuickItemsResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetQuickItemsResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	var targetGameIDs [quickItemSlotCount]uint32
	saveRevision, err := engine.commitCharacterRevision(saveSessionID, opSetQuickItems, characterID, func(loaded *loadedSave) error {
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
		anchor := inventoryAt - inventoryHeldCommonOffset
		_, slotEnd := inventorySlotBounds(loaded.session.platform, characterID)

		pairAt := anchor + quickItemsSectionOffset
		if pairAt+quickItemsReadSize > slotEnd {
			return fmt.Errorf("quick items of character %d do not fit into its slot", characterID)
		}

		countAt := anchor + equipmentProjectileCountOffset
		if countAt+4 > slotEnd {
			return fmt.Errorf("projectile count of character %d lies outside its slot", characterID)
		}
		rawCount, err := loaded.snapshot.readAt(countAt, 4)
		if err != nil {
			return fmt.Errorf("cannot read projectile count of character %d: %w", characterID, err)
		}
		count := int64(binary.LittleEndian.Uint32(rawCount))
		if count > equipmentMaxProjectileRecords {
			return fmt.Errorf("character %d declares %d projectile records, want at most %d",
				characterID, count, equipmentMaxProjectileRecords)
		}

		armamentsAt := countAt + 4 + count*equipmentProjectileRecordSize
		tailAt := armamentsAt + quickItemsTailOffset
		if tailAt+quickItemsTailSize > slotEnd {
			return fmt.Errorf(
				"quick-items equipped-armaments tail of character %d does not fit into its slot",
				characterID)
		}

		beforePairs, err := loaded.snapshot.readAt(pairAt, quickItemsActiveOffset)
		if err != nil {
			return fmt.Errorf("cannot read quick-items EquipItemData of character %d: %w", characterID, err)
		}
		beforeTail, err := loaded.snapshot.readAt(tailAt, quickItemsTailSize)
		if err != nil {
			return fmt.Errorf(
				"cannot read quick-items equipped-armaments tail of character %d: %w",
				characterID, err)
		}

		for i := 0; i < quickItemSlotCount; i++ {
			existingHandle := binary.LittleEndian.Uint32(beforePairs[i*quickItemRecordSize:])
			existingEquipIndex := binary.LittleEndian.Uint32(beforePairs[i*quickItemRecordSize+4:])
			existingGameID := binary.LittleEndian.Uint32(beforeTail[i*4:])

			if existingHandle == quickItemEmptyItemID &&
				existingEquipIndex == quickItemEmptyEquipIndex &&
				existingGameID == QuickItemEmptyGameID {
				continue
			}

			if existingHandle != inventoryHeldEmptyHandle &&
				existingHandle != inventoryHeldInvalidHandle &&
				existingHandle&gaItemHandleTypeMask == gaItemGoodsHandle &&
				existingEquipIndex >= removeReferenceInventoryRowBase {
				physicalIndex := int(existingEquipIndex - removeReferenceInventoryRowBase)
				if physicalIndex < inventoryHeldCommonRecords {
					rowAt := inventoryAt + int64(physicalIndex)*inventoryHeldRecordSize
					row, readErr := loaded.snapshot.readAt(rowAt, inventoryHeldRecordSize)
					if readErr == nil {
						rowHandle := binary.LittleEndian.Uint32(row)
						quantity := binary.LittleEndian.Uint32(row[4:]) & inventoryHeldQuantityMask
						gameID, resolveErr := resolveGaItemHandle(nil, existingHandle)
						if resolveErr == nil && rowHandle == existingHandle &&
							quantity > 0 && gameID == existingGameID {
							continue
						}
					}
				}
			}

			return fmt.Errorf(
				"quick item slot %d: inconsistent existing save state (handle=0x%08X, equipIndex=0x%08X, tailGameID=0x%08X)",
				i, existingHandle, existingEquipIndex, existingGameID)
		}

		seenHandles := make(map[uint32]int, quickItemSlotCount)
		var targetHandles [quickItemSlotCount]uint32
		var targetEquipIndexes [quickItemSlotCount]uint32

		for i, assignment := range slotAssignments {
			if assignment == nil {
				targetHandles[i] = quickItemEmptyItemID
				targetEquipIndexes[i] = quickItemEmptyEquipIndex
				targetGameIDs[i] = QuickItemEmptyGameID
				continue
			}

			token := *assignment
			if token == "" {
				return fmt.Errorf("slotAssignments[%d] is empty string", i)
			}
			locator, err := loaded.session.resolveOwnedItemID(characterID, token)
			if err != nil {
				return fmt.Errorf("slotAssignments[%d]: %w", i, err)
			}
			if locator.container != ownedContainerInventory {
				return fmt.Errorf(
					"slotAssignments[%d]: ownedItemID %q addresses a %s record; active Inventory record required",
					i, token, locator.container)
			}
			if locator.containerSection != InventorySectionCommon {
				return fmt.Errorf(
					"slotAssignments[%d]: ownedItemID %q sits in section %q; Quick Items must be in common inventory",
					i, token, locator.containerSection)
			}

			rowAt := inventoryAt + int64(locator.physicalIndex)*inventoryHeldRecordSize
			row, err := loaded.snapshot.readAt(rowAt, inventoryHeldRecordSize)
			if err != nil {
				return fmt.Errorf("cannot read inventory row %d of character %d: %w",
					locator.physicalIndex, characterID, err)
			}
			handle := binary.LittleEndian.Uint32(row)
			quantity := binary.LittleEndian.Uint32(row[4:]) & inventoryHeldQuantityMask
			if handle == inventoryHeldEmptyHandle || handle == inventoryHeldInvalidHandle {
				return fmt.Errorf(
					"slotAssignments[%d]: ownedItemID %q addresses an empty inventory row", i, token)
			}
			if quantity == 0 {
				return fmt.Errorf("slotAssignments[%d]: ownedItemID %q has 0 quantity", i, token)
			}

			gameID, err := resolveGaItemHandle(nil, handle)
			if err != nil {
				return fmt.Errorf("slotAssignments[%d] ownedItemID %q: %w", i, token, err)
			}
			if validateGameID != nil {
				if err := validateGameID(gameID); err != nil {
					return fmt.Errorf("slotAssignments[%d]: %w", i, err)
				}
			}
			if handle&gaItemHandleTypeMask != gaItemGoodsHandle {
				return fmt.Errorf(
					"slotAssignments[%d]: ownedItemID %q has non-goods handle 0x%08X",
					i, token, handle)
			}
			if previous, exists := seenHandles[handle]; exists {
				return fmt.Errorf(
					"item 0x%08X is assigned to both slot %d and slot %d", gameID, previous, i)
			}
			seenHandles[handle] = i

			targetHandles[i] = handle
			targetEquipIndexes[i] = removeReferenceInventoryRowBase + uint32(locator.physicalIndex)
			targetGameIDs[i] = gameID
		}

		afterPairs := make([]byte, quickItemsActiveOffset)
		afterTail := make([]byte, quickItemsTailSize)
		for i := 0; i < quickItemSlotCount; i++ {
			binary.LittleEndian.PutUint32(afterPairs[i*quickItemRecordSize:], targetHandles[i])
			binary.LittleEndian.PutUint32(afterPairs[i*quickItemRecordSize+4:], targetEquipIndexes[i])
			binary.LittleEndian.PutUint32(afterTail[i*4:], targetGameIDs[i])
		}

		if bytes.Equal(beforePairs, afterPairs) && bytes.Equal(beforeTail, afterTail) {
			return nil
		}

		if err := loaded.snapshot.writeAt(pairAt, afterPairs); err != nil {
			return fmt.Errorf("cannot write quick-items EquipItemData of character %d: %w", characterID, err)
		}
		if err := loaded.snapshot.writeAt(tailAt, afterTail); err != nil {
			if rollbackErr := loaded.snapshot.writeAt(pairAt, beforePairs); rollbackErr != nil {
				return fmt.Errorf(
					"cannot write quick-items equipped-armaments tail of character %d (rollback of pairs failed: %v): %w",
					characterID, rollbackErr, err)
			}
			return fmt.Errorf(
				"cannot write quick-items equipped-armaments tail of character %d: %w",
				characterID, err)
		}

		writtenPairs, pairsErr := loaded.snapshot.readAt(pairAt, quickItemsActiveOffset)
		writtenTail, tailErr := loaded.snapshot.readAt(tailAt, quickItemsTailSize)
		if pairsErr == nil && tailErr == nil &&
			bytes.Equal(writtenPairs, afterPairs) && bytes.Equal(writtenTail, afterTail) {
			return nil
		}

		pairsRollbackErr := loaded.snapshot.writeAt(pairAt, beforePairs)
		tailRollbackErr := loaded.snapshot.writeAt(tailAt, beforeTail)
		if pairsRollbackErr != nil || tailRollbackErr != nil {
			return fmt.Errorf(
				"quick items of character %d could not be verified and could not be restored: %v / %v",
				characterID, pairsRollbackErr, tailRollbackErr)
		}
		return errors.New("quick items mutation could not be verified; the save is unchanged")
	})
	if err != nil {
		return SetQuickItemsResult{}, err
	}

	return SetQuickItemsResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  saveRevision,
		CharacterID:   characterID,
		GameIDs:       targetGameIDs,
	}, nil
}
