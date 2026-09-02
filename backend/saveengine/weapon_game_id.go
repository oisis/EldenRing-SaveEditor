package saveengine

import (
	"encoding/binary"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// setOwnedWeaponGameID is the single binary mutation used by weapon upgrade
// and infusion setters. It changes no allocation, handle, row or AoW state.
//
// operationKind names the public setter that requested the change, so the undo
// point of a shared write is never attributed to the wrong endpoint, and the
// returned receipt reports that same kind. The receipt is the one the central
// commit path produced; this writer never builds one of its own.
func (engine *Engine) setOwnedWeaponGameID(
	saveSessionID string,
	characterID int,
	ownedItemID string,
	expectedRevision string,
	expectedGameID uint32,
	targetGameID uint32,
	operationKind string,
	matchmakingLevel uint8,
) (MutationReceipt, string, error) {
	if expectedGameID&gaItemHandleTypeMask != 0 || targetGameID&gaItemHandleTypeMask != 0 {
		return MutationReceipt{}, "", fmt.Errorf(
			"weapon game IDs must use prefix 0; got 0x%08X and 0x%08X",
			expectedGameID, targetGameID)
	}
	if !isCanonicalRevision(expectedRevision) {
		return MutationReceipt{}, "", apperror.InvalidRevision(expectedRevision)
	}

	var container string
	committed, err := engine.commitCharacterRevision(saveSessionID, operationKind, characterID, func(loaded *loadedSave) error {
		if characterID < 0 || characterID >= characterSlotCount {
			return fmt.Errorf("characterID %d is outside the range 0..%d",
				characterID, characterSlotCount-1)
		}
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return apperror.RevisionConflict(expectedRevision, current)
		}
		flag, err := loaded.snapshot.readAt(
			userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
		}
		if flag[0] != userData10ActiveFlagValue {
			return fmt.Errorf("character %d is not active", characterID)
		}

		locator, err := loaded.session.resolveOwnedItemID(characterID, ownedItemID)
		if err != nil {
			return err
		}
		if locator.containerSection != InventorySectionCommon {
			return fmt.Errorf("ownedItemID %q must address a common container section", ownedItemID)
		}
		container = locator.container
		records, err := readOwnedRecords(loaded, characterID, locator.container)
		if err != nil {
			return err
		}
		var target *ownedRecord
		for index := range records {
			record := &records[index]
			if record.containerSection == locator.containerSection &&
				record.physicalIndex == locator.physicalIndex && record.ownedItemID == ownedItemID {
				target = record
				break
			}
		}
		if target == nil {
			return fmt.Errorf("ownedItemID %q no longer addresses a record of character %d",
				ownedItemID, characterID)
		}
		if target.gaItemHandle&gaItemHandleTypeMask != gaItemWeaponHandle || target.quantity == 0 {
			return fmt.Errorf("ownedItemID %q is not a positive-quantity weapon record", ownedItemID)
		}

		var gaItemAt int64
		matches := 0
		err = walkGaItemRecords(loaded.snapshot, loaded.session.platform, characterID,
			func(record gaItemRecord) error {
				if record.handle == target.gaItemHandle {
					gaItemAt, matches = record.at, matches+1
					if record.gameID != expectedGameID {
						return fmt.Errorf(
							"ownedItemID %q now denotes item 0x%08X, not the expected 0x%08X",
							ownedItemID, record.gameID, expectedGameID)
					}
				}
				return nil
			})
		if err != nil {
			return fmt.Errorf("cannot resolve items of character %d: %w", characterID, err)
		}
		if matches != 1 {
			return fmt.Errorf("weapon handle 0x%08X has %d GaItem records, want exactly 1",
				target.gaItemHandle, matches)
		}

		var writes []byteWrite
		if targetGameID != expectedGameID {
			writes = append(writes, byteWrite{at: gaItemAt + 4, data: littleEndianUint32(targetGameID)})
			equippedWrites, err := planEquippedWeaponIDWrites(
				loaded, characterID, locator, target.gaItemHandle, expectedGameID, targetGameID)
			if err != nil {
				return err
			}
			writes = append(writes, equippedWrites...)
			gaItemDataWrites, err := planGaItemDataInsertion(loaded, characterID, targetGameID)
			if err != nil {
				return err
			}
			writes = append(writes, gaItemDataWrites...)
		}

		if operationKind == kindSetWeaponUpgradeLevel {
			matchmakingWrites, err := planWeaponMatchmakingLevelWrite(loaded, characterID, matchmakingLevel)
			if err != nil {
				return err
			}
			writes = append(writes, matchmakingWrites...)
		}

		if len(writes) == 0 {
			return nil
		}

		if err := applyByteWrites(loaded.snapshot, writes); err != nil {
			return fmt.Errorf("cannot change owned weapon %q: %w", ownedItemID, err)
		}
		return nil
	})
	if err != nil {
		return MutationReceipt{}, "", err
	}
	return committed, container, nil
}

// planWeaponMatchmakingLevelWrite locates the character's stats anchor and plans
// a write to the durable matchmaking level byte at MagicOffset - 0xD5 if the
// target level is higher than the current value. The rule is strictly monotonic.
func planWeaponMatchmakingLevelWrite(
	loaded *loadedSave,
	characterID int,
	matchmakingLevel uint8,
) ([]byteWrite, error) {
	if matchmakingLevel > 25 {
		return nil, fmt.Errorf("matchmaking level %d exceeds maximum 25", matchmakingLevel)
	}
	anchor, err := findStatsAnchor(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return nil, err
	}
	slotBase := slotDataBase(loaded.session.platform, characterID)
	slotEnd := slotBase + characterSlotDataSize
	targetAt := anchor + statsMatchmakingWeaponLevelOffset
	if targetAt < slotBase || targetAt >= slotEnd {
		return nil, fmt.Errorf("matchmaking level offset %d for character %d is outside slot bounds [%d, %d)",
			targetAt, characterID, slotBase, slotEnd)
	}
	currentByte, err := loaded.snapshot.readAt(targetAt, 1)
	if err != nil {
		return nil, fmt.Errorf("cannot read matchmaking level of character %d: %w", characterID, err)
	}
	currentLevel := currentByte[0]
	if matchmakingLevel > currentLevel {
		return []byteWrite{{at: targetAt, data: []byte{matchmakingLevel}}}, nil
	}
	return nil, nil
}

// planEquippedWeaponIDWrites validates and updates only hand slots carrying the
// target handle. Unrelated equipment is outside this mutation's scope.
func planEquippedWeaponIDWrites(
	loaded *loadedSave,
	characterID int,
	locator ownedItemLocator,
	handle, currentGameID, targetGameID uint32,
) ([]byteWrite, error) {
	inventoryAt, err := inventoryHeldSectionAt(loaded, characterID)
	if err != nil {
		return nil, err
	}
	anchor := inventoryAt - inventoryHeldCommonOffset
	_, slotEnd := inventorySlotBounds(loaded.session.platform, characterID)
	countAt := anchor + equipmentProjectileCountOffset
	if countAt+4 > slotEnd {
		return nil, fmt.Errorf("projectile count of character %d lies outside its slot", characterID)
	}
	count, err := loaded.snapshot.uint32At(countAt)
	if err != nil {
		return nil, fmt.Errorf("cannot read projectile count of character %d: %w", characterID, err)
	}
	if count > equipmentMaxProjectileRecords {
		return nil, fmt.Errorf("character %d declares %d projectile records, want at most %d",
			characterID, count, equipmentMaxProjectileRecords)
	}
	dynamicAt := countAt + 4 + int64(count)*equipmentProjectileRecordSize
	if dynamicAt+equipmentBlockSize > slotEnd {
		return nil, fmt.Errorf("equipment block of character %d does not fit into its slot", characterID)
	}

	indexAt := anchor + removeEquipmentIndexesOffset
	bareAt := indexAt + equipmentSlotCount*4 + 0x1C
	handleAt := anchor + removeEquipmentHandlesOffset
	indexes, err := loaded.snapshot.readAt(indexAt, equippedArmamentSlotCount*4)
	if err != nil {
		return nil, fmt.Errorf("cannot read equipped weapon rows of character %d: %w", characterID, err)
	}
	bareIDs, err := loaded.snapshot.readAt(bareAt, equippedArmamentSlotCount*4)
	if err != nil {
		return nil, fmt.Errorf("cannot read equipped weapon IDs of character %d: %w", characterID, err)
	}
	handles, err := loaded.snapshot.readAt(handleAt, equippedArmamentSlotCount*4)
	if err != nil {
		return nil, fmt.Errorf("cannot read equipped weapon handles of character %d: %w", characterID, err)
	}
	dynamicIDs, err := loaded.snapshot.readAt(dynamicAt, equippedArmamentSlotCount*4)
	if err != nil {
		return nil, fmt.Errorf("cannot read dynamic equipped weapon IDs of character %d: %w", characterID, err)
	}

	var writes []byteWrite
	for slot := 0; slot < equippedArmamentSlotCount; slot++ {
		field := slot * 4
		if binary.LittleEndian.Uint32(handles[field:]) != handle {
			continue
		}
		if locator.container != ownedContainerInventory ||
			locator.containerSection != InventorySectionCommon {
			return nil, fmt.Errorf("equipped weapon handle 0x%08X does not belong to Inventory common", handle)
		}
		wantRow := removeReferenceInventoryRowBase + uint32(locator.physicalIndex)
		if binary.LittleEndian.Uint32(indexes[field:]) != wantRow ||
			binary.LittleEndian.Uint32(bareIDs[field:]) != currentGameID ||
			binary.LittleEndian.Uint32(dynamicIDs[field:]) != currentGameID {
			return nil, fmt.Errorf(
				"armament slot %d of character %d has inconsistent weapon representations",
				slot, characterID)
		}
		writes = append(writes,
			byteWrite{at: bareAt + int64(field), data: littleEndianUint32(targetGameID)},
			byteWrite{at: dynamicAt + int64(field), data: littleEndianUint32(targetGameID)},
		)
	}
	return writes, nil
}
