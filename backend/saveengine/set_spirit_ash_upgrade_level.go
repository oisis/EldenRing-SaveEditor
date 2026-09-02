package saveengine

import (
	"encoding/binary"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// SetSpiritAshUpgradeLevelResult reports one committed Spirit Ash upgrade.
// OwnedItemID is stale after the returned revision advances.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type SetSpiritAshUpgradeLevelResult struct {
	MutationReceipt
	OwnedItemID    string `json:"ownedItemID"`
	CharacterID    int    `json:"characterID"`
	Container      string `json:"container"`
	PreviousGameID uint32 `json:"previousGameID"`
	GameID         uint32 `json:"gameID"`
	UpgradeLevel   uint8  `json:"upgradeLevel"`
}

// SetSpiritAshUpgradeLevel changes the exact goods handle of one existing
// Spirit Ash record and keeps matching Inventory loadout references coherent.
func (engine *Engine) SetSpiritAshUpgradeLevel(
	saveSessionID string,
	characterID int,
	ownedItemID string,
	upgradeLevel uint8,
	expectedRevision string,
	expectedGameID uint32,
	targetGameID uint32,
) (SetSpiritAshUpgradeLevelResult, error) {
	if expectedGameID&gaItemHandleTypeMask != 0x40000000 {
		return SetSpiritAshUpgradeLevelResult{}, fmt.Errorf(
			"current Spirit Ash game ID 0x%08X must use the goods prefix", expectedGameID)
	}
	if targetGameID&gaItemHandleTypeMask != 0x40000000 {
		return SetSpiritAshUpgradeLevelResult{}, fmt.Errorf(
			"target Spirit Ash game ID 0x%08X must use the goods prefix", targetGameID)
	}
	expectedHandle, err := gaItemHandleForGameID(expectedGameID)
	if err != nil {
		return SetSpiritAshUpgradeLevelResult{}, err
	}
	targetHandle, err := gaItemHandleForGameID(targetGameID)
	if err != nil {
		return SetSpiritAshUpgradeLevelResult{}, err
	}
	if !isCanonicalRevision(expectedRevision) {
		return SetSpiritAshUpgradeLevelResult{}, apperror.InvalidRevision(expectedRevision)
	}

	var container string
	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetSpiritAshUpgradeLevel, characterID, func(loaded *loadedSave) error {
		if characterID < 0 || characterID >= characterSlotCount {
			return fmt.Errorf("characterID %d is outside the range 0..%d",
				characterID, characterSlotCount-1)
		}
		currentRevision := loaded.session.revisionString()
		if expectedRevision != currentRevision {
			return apperror.RevisionConflict(expectedRevision, currentRevision)
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
		if target.gaItemHandle != expectedHandle || target.quantity == 0 {
			return fmt.Errorf(
				"ownedItemID %q now denotes handle 0x%08X with quantity %d, not Spirit Ash 0x%08X",
				ownedItemID, target.gaItemHandle, target.quantity, expectedGameID)
		}
		if targetGameID == expectedGameID {
			return nil
		}

		recordAt, _, _, err := ownedItemRowAndCountAt(loaded, locator)
		if err != nil {
			return err
		}
		writes := []byteWrite{{at: recordAt, data: littleEndianUint32(targetHandle)}}
		referenceWrites, err := planSpiritAshReferenceWrites(
			loaded, characterID, locator, expectedHandle, targetHandle, expectedGameID, targetGameID)
		if err != nil {
			return err
		}
		writes = append(writes, referenceWrites...)
		gaItemDataWrites, err := planGaItemDataInsertion(loaded, characterID, targetGameID)
		if err != nil {
			return err
		}
		writes = append(writes, gaItemDataWrites...)
		if err := applyByteWrites(loaded.snapshot, writes); err != nil {
			return fmt.Errorf("cannot change owned Spirit Ash %q: %w", ownedItemID, err)
		}
		return nil
	})
	if err != nil {
		return SetSpiritAshUpgradeLevelResult{}, err
	}
	return SetSpiritAshUpgradeLevelResult{
		MutationReceipt: committed, OwnedItemID: ownedItemID,
		CharacterID: characterID, Container: container, PreviousGameID: expectedGameID,
		GameID: targetGameID, UpgradeLevel: upgradeLevel,
	}, nil
}

// planSpiritAshReferenceWrites updates only Quick Items and Pouch positions
// whose stored Inventory row is the record being changed. Storage has no active
// loadout reference.
func planSpiritAshReferenceWrites(
	loaded *loadedSave,
	characterID int,
	locator ownedItemLocator,
	currentHandle, targetHandle, currentGameID, targetGameID uint32,
) ([]byteWrite, error) {
	if locator.container != ownedContainerInventory {
		return nil, nil
	}
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

	quick, err := planSpiritAshSlots(
		loaded, characterID, "quick item", anchor+quickItemsSectionOffset,
		dynamicAt+quickItemsTailOffset, quickItemSlotCount, locator.physicalIndex,
		currentHandle, targetHandle, currentGameID, targetGameID, slotEnd)
	if err != nil {
		return nil, err
	}
	pouch, err := planSpiritAshSlots(
		loaded, characterID, "pouch", anchor+pouchItemsSectionOffset,
		dynamicAt+pouchItemsTailOffset, pouchItemSlotCount, locator.physicalIndex,
		currentHandle, targetHandle, currentGameID, targetGameID, slotEnd)
	if err != nil {
		return nil, err
	}
	return append(quick, pouch...), nil
}

func planSpiritAshSlots(
	loaded *loadedSave,
	characterID int,
	label string,
	pairAt, tailAt int64,
	slotCount int,
	physicalIndex int,
	currentHandle, targetHandle, currentGameID, targetGameID uint32,
	slotEnd int64,
) ([]byteWrite, error) {
	pairSize, tailSize := slotCount*8, slotCount*4
	if pairAt+int64(pairSize) > slotEnd || tailAt+int64(tailSize) > slotEnd {
		return nil, fmt.Errorf("%s data of character %d does not fit into its slot", label, characterID)
	}
	pairs, err := loaded.snapshot.readAt(pairAt, pairSize)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s pairs of character %d: %w", label, characterID, err)
	}
	tail, err := loaded.snapshot.readAt(tailAt, tailSize)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s IDs of character %d: %w", label, characterID, err)
	}

	wantRow := removeReferenceInventoryRowBase + uint32(physicalIndex)
	var writes []byteWrite
	for slot := 0; slot < slotCount; slot++ {
		field := slot * 8
		if binary.LittleEndian.Uint32(pairs[field+4:]) != wantRow {
			continue
		}
		if binary.LittleEndian.Uint32(pairs[field:]) != currentHandle ||
			binary.LittleEndian.Uint32(tail[slot*4:]) != currentGameID {
			return nil, fmt.Errorf(
				"%s slot %d of character %d has inconsistent Spirit Ash representations",
				label, slot, characterID)
		}
		writes = append(writes,
			byteWrite{at: pairAt + int64(field), data: littleEndianUint32(targetHandle)},
			byteWrite{at: tailAt + int64(slot*4), data: littleEndianUint32(targetGameID)})
	}
	return writes, nil
}
