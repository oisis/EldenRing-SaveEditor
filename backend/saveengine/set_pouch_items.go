package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// PouchEmptyGameID is the confirmed native value of one empty Pouch position.
	PouchEmptyGameID     uint32 = 0xFFFFFFFF
	pouchEmptyItemID     uint32 = 0x00000000
	pouchEmptyEquipIndex uint32 = 0xFFFFFFFF
	pouchItemsTailOffset        = 0x80
	pouchItemsTailSize          = pouchItemSlotCount * 4
)

// SetPouchItemsResult reports one committed six-slot Pouch assignment.
type SetPouchItemsResult struct {
	SaveSessionID string    `json:"saveSessionID"`
	SaveRevision  string    `json:"saveRevision"`
	CharacterID   int       `json:"characterID"`
	GameIDs       [6]uint32 `json:"gameIDs"`
}

// SetPouchItems atomically replaces the six Pouch slot positions of one character.
func (engine *Engine) SetPouchItems(
	saveSessionID string,
	characterID int,
	slotAssignments [6]*string,
	expectedRevision string,
	validateGameID func(gameID uint32) error,
) (SetPouchItemsResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetPouchItemsResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	var targetGoodsIDs [6]uint32
	saveRevision, err := engine.commitCharacterRevision(saveSessionID, opSetPouchItems, characterID, func(loaded *loadedSave) error {
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

		sectionAt, err := inventoryHeldSectionAt(loaded, characterID)
		if err != nil {
			return err
		}

		anchor := sectionAt - inventoryHeldCommonOffset
		_, slotEnd := inventorySlotBounds(loaded.session.platform, characterID)
		pairAt := anchor + pouchItemsSectionOffset
		if pairAt+pouchItemsReadSize > slotEnd {
			return fmt.Errorf("pouch items of character %d do not fit into its slot", characterID)
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

		armamentsOff := countAt + 4 + count*equipmentProjectileRecordSize
		tailAt := armamentsOff + pouchItemsTailOffset
		if tailAt+pouchItemsTailSize > slotEnd {
			return fmt.Errorf("equipped-armaments tail of character %d does not fit into its slot", characterID)
		}

		beforePairs, err := loaded.snapshot.readAt(pairAt, pouchItemsReadSize)
		if err != nil {
			return fmt.Errorf("cannot read pouch EquipItemData of character %d: %w", characterID, err)
		}
		beforeTail, err := loaded.snapshot.readAt(tailAt, pouchItemsTailSize)
		if err != nil {
			return fmt.Errorf("cannot read pouch equipped-armaments tail of character %d: %w", characterID, err)
		}

		// Validate all six existing pouch positions before writing anything.
		for i := 0; i < pouchItemSlotCount; i++ {
			existingHandle := binary.LittleEndian.Uint32(beforePairs[i*8:])
			existingEquipIndex := binary.LittleEndian.Uint32(beforePairs[i*8+4:])
			existingTailGameID := binary.LittleEndian.Uint32(beforeTail[i*4:])

			if existingHandle == 0 && existingEquipIndex == 0xFFFFFFFF && existingTailGameID == 0xFFFFFFFF {
				continue
			}

			if existingHandle != 0 && existingHandle != 0xFFFFFFFF &&
				(existingHandle&gaItemHandleTypeMask) == gaItemGoodsHandle &&
				existingEquipIndex >= removeReferenceInventoryRowBase {

				physicalIndex := int(existingEquipIndex - removeReferenceInventoryRowBase)
				if physicalIndex >= 0 && physicalIndex < inventoryHeldCommonRecords {
					rowAt := sectionAt + int64(physicalIndex)*inventoryHeldRecordSize
					rowBytes, err := loaded.snapshot.readAt(rowAt, inventoryHeldRecordSize)
					if err == nil {
						rowHandle := binary.LittleEndian.Uint32(rowBytes)
						rowQuantity := binary.LittleEndian.Uint32(rowBytes[4:]) & inventoryHeldQuantityMask
						gameID, resolveErr := resolveGaItemHandle(nil, existingHandle)
						if resolveErr == nil && rowHandle == existingHandle && rowQuantity > 0 && gameID == existingTailGameID {
							continue
						}
					}
				}
			}

			return fmt.Errorf("pouch slot %d: inconsistent existing save state (handle=0x%08X, equipIndex=0x%08X, tailGameID=0x%08X)",
				i, existingHandle, existingEquipIndex, existingTailGameID)
		}

		seenIDs := make(map[string]int, 6)
		var targetHandles [6]uint32
		var targetEquipIndexes [6]uint32

		for i, ptr := range slotAssignments {
			if ptr == nil {
				targetHandles[i] = pouchEmptyItemID
				targetEquipIndexes[i] = pouchEmptyEquipIndex
				targetGoodsIDs[i] = PouchEmptyGameID
				continue
			}

			token := *ptr
			if token == "" {
				return fmt.Errorf("slotAssignments[%d] is empty string", i)
			}
			if prev, exists := seenIDs[token]; exists {
				return fmt.Errorf("ownedItemID %q is assigned to both slot %d and slot %d", token, prev, i)
			}
			seenIDs[token] = i

			locator, err := loaded.session.resolveOwnedItemID(characterID, token)
			if err != nil {
				return fmt.Errorf("slotAssignments[%d]: %w", i, err)
			}
			if locator.container != ownedContainerInventory {
				return fmt.Errorf("slotAssignments[%d]: ownedItemID %q addresses a %s record; active Inventory record required",
					i, token, locator.container)
			}
			if locator.containerSection != InventorySectionCommon {
				return fmt.Errorf("slotAssignments[%d]: ownedItemID %q sits in section %q; Pouch items must be in common inventory",
					i, token, locator.containerSection)
			}
			rowAt := sectionAt + int64(locator.physicalIndex)*inventoryHeldRecordSize
			recordBytes, err := loaded.snapshot.readAt(rowAt, inventoryHeldRecordSize)
			if err != nil {
				return fmt.Errorf("cannot read inventory row %d of character %d: %w",
					locator.physicalIndex, characterID, err)
			}
			handle := binary.LittleEndian.Uint32(recordBytes)
			quantity := binary.LittleEndian.Uint32(recordBytes[4:]) & inventoryHeldQuantityMask

			if handle == inventoryHeldEmptyHandle || handle == inventoryHeldInvalidHandle {
				return fmt.Errorf("slotAssignments[%d]: ownedItemID %q addresses an empty inventory row", i, token)
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

			targetHandles[i] = handle
			targetEquipIndexes[i] = removeReferenceInventoryRowBase + uint32(locator.physicalIndex)
			targetGoodsIDs[i] = gameID
		}

		afterPairs := make([]byte, pouchItemsReadSize)
		afterTail := make([]byte, pouchItemsTailSize)
		for i := 0; i < pouchItemSlotCount; i++ {
			binary.LittleEndian.PutUint32(afterPairs[i*8:], targetHandles[i])
			binary.LittleEndian.PutUint32(afterPairs[i*8+4:], targetEquipIndexes[i])
			binary.LittleEndian.PutUint32(afterTail[i*4:], targetGoodsIDs[i])
		}

		if bytes.Equal(beforePairs, afterPairs) && bytes.Equal(beforeTail, afterTail) {
			return nil
		}

		if err := loaded.snapshot.writeAt(pairAt, afterPairs); err != nil {
			return fmt.Errorf("cannot write pouch EquipItemData of character %d: %w", characterID, err)
		}
		if err := loaded.snapshot.writeAt(tailAt, afterTail); err != nil {
			if rollbackErr := loaded.snapshot.writeAt(pairAt, beforePairs); rollbackErr != nil {
				return fmt.Errorf("cannot write pouch equipped-armaments tail of character %d (rollback of pairs failed: %v): %w",
					characterID, rollbackErr, err)
			}
			return fmt.Errorf("cannot write pouch equipped-armaments tail of character %d: %w", characterID, err)
		}

		writtenPairs, errPairs := loaded.snapshot.readAt(pairAt, pouchItemsReadSize)
		writtenTail, errTail := loaded.snapshot.readAt(tailAt, pouchItemsTailSize)
		if errPairs == nil && errTail == nil && bytes.Equal(writtenPairs, afterPairs) && bytes.Equal(writtenTail, afterTail) {
			return nil
		}

		r1 := loaded.snapshot.writeAt(pairAt, beforePairs)
		r2 := loaded.snapshot.writeAt(tailAt, beforeTail)
		if r1 != nil || r2 != nil {
			return fmt.Errorf("pouch items of character %d could not be verified and could not be restored: %v / %v",
				characterID, r1, r2)
		}
		return errors.New("pouch items mutation could not be verified; the save is unchanged")
	})

	if err != nil {
		return SetPouchItemsResult{}, err
	}

	return SetPouchItemsResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  saveRevision,
		CharacterID:   characterID,
		GameIDs:       targetGoodsIDs,
	}, nil
}
