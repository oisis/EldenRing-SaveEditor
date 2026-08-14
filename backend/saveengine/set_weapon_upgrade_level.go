package saveengine

import (
	"encoding/binary"
	"fmt"
)

// SetWeaponUpgradeLevelResult reports one committed upgrade-level change.
// OwnedItemID identifies the record that was changed but is stale after the
// returned revision advances, like every other owned-item mutation receipt.
type SetWeaponUpgradeLevelResult struct {
	SaveSessionID  string `json:"saveSessionID"`
	SaveRevision   string `json:"saveRevision"`
	OwnedItemID    string `json:"ownedItemID"`
	CharacterID    int    `json:"characterID"`
	Container      string `json:"container"`
	PreviousGameID uint32 `json:"previousGameID"`
	GameID         uint32 `json:"gameID"`
	UpgradeLevel   uint8  `json:"upgradeLevel"`
}

// SetWeaponUpgradeLevel changes the exact save-side game ID of one existing
// weapon record. It allocates and repacks nothing: the record keeps its handle,
// container row, quantity and acquisition index. When the same handle is
// equipped in a hand slot, both item-ID representations are changed in the same
// atomic plan. GaItemData gains the target ID while the previous ID is retained.
func (engine *Engine) SetWeaponUpgradeLevel(
	saveSessionID string,
	characterID int,
	ownedItemID string,
	upgradeLevel uint8,
	expectedRevision string,
	expectedGameID uint32,
	targetGameID uint32,
) (SetWeaponUpgradeLevelResult, error) {
	if expectedGameID&gaItemHandleTypeMask != 0 || targetGameID&gaItemHandleTypeMask != 0 {
		return SetWeaponUpgradeLevelResult{}, fmt.Errorf(
			"weapon game IDs must use prefix 0; got 0x%08X and 0x%08X",
			expectedGameID, targetGameID)
	}
	if !isCanonicalRevision(expectedRevision) {
		return SetWeaponUpgradeLevelResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	var container string
	saveRevision, err := engine.commitRevision(saveSessionID, func(loaded *loadedSave) error {
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
		if targetGameID == expectedGameID {
			return nil
		}

		writes := []byteWrite{{at: gaItemAt + 4, data: littleEndianUint32(targetGameID)}}
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
		if err := applyByteWrites(loaded.snapshot, writes); err != nil {
			return fmt.Errorf("cannot change the upgrade level of ownedItemID %q: %w", ownedItemID, err)
		}
		return nil
	})
	if err != nil {
		return SetWeaponUpgradeLevelResult{}, err
	}
	return SetWeaponUpgradeLevelResult{
		SaveSessionID: saveSessionID, SaveRevision: saveRevision, OwnedItemID: ownedItemID,
		CharacterID: characterID, Container: container, PreviousGameID: expectedGameID,
		GameID: targetGameID, UpgradeLevel: upgradeLevel,
	}, nil
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
