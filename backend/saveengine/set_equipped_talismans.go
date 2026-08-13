package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	equippedTalismanFirstSlot = 17
	equippedTalismanSlotCount = 4
	equippedTalismanBareMask  = 0x0FFFFFFF
)

// SetEquippedTalismansResult reports one committed compact talisman loadout.
type SetEquippedTalismansResult struct {
	SaveSessionID string   `json:"saveSessionID"`
	SaveRevision  string   `json:"saveRevision"`
	CharacterID   int      `json:"characterID"`
	GameIDs       []uint32 `json:"gameIDs"`
	UnlockedSlots int      `json:"unlockedSlots"`
}

// SetEquippedTalismans atomically replaces the four player-visible talisman fields.
func (engine *Engine) SetEquippedTalismans(
	saveSessionID string,
	characterID int,
	orderedOwnedItemIDs []string,
	expectedRevision string,
	validateGameID func(uint32) error,
) (SetEquippedTalismansResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetEquippedTalismansResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if len(orderedOwnedItemIDs) > equippedTalismanSlotCount {
		return SetEquippedTalismansResult{}, fmt.Errorf(
			"orderedOwnedItemIDs contains %d talismans, want at most %d",
			len(orderedOwnedItemIDs), equippedTalismanSlotCount)
	}

	gameIDs := make([]uint32, len(orderedOwnedItemIDs))
	unlockedSlots := 0
	saveRevision, err := engine.commitRevision(saveSessionID, func(loaded *loadedSave) error {
		if characterID < 0 || characterID >= characterSlotCount {
			return fmt.Errorf("characterID %d is outside the range 0..%d",
				characterID, characterSlotCount-1)
		}
		if expectedRevision != loaded.session.revisionString() {
			return fmt.Errorf(
				"expectedRevision %q does not match the current saveRevision %q",
				expectedRevision, loaded.session.revisionString())
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
		base, slotEnd := inventorySlotBounds(loaded.session.platform, characterID)
		unlockedSlots, err = readUnlockedTalismanFields(
			loaded.snapshot, anchor, base, characterID)
		if err != nil {
			return err
		}
		if len(orderedOwnedItemIDs) > unlockedSlots {
			return fmt.Errorf("character %d has %d unlocked talisman slot(s), cannot equip %d",
				characterID, unlockedSlots, len(orderedOwnedItemIDs))
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

		fieldOffset := int64(equippedTalismanFirstSlot * 4)
		ranges := [4]int64{
			anchor + removeEquipmentIndexesOffset + fieldOffset,
			anchor + removeEquipmentIndexesOffset + equipmentSlotCount*4 + 0x1C + fieldOffset,
			anchor + removeEquipmentHandlesOffset + fieldOffset,
			armamentsAt + fieldOffset,
		}
		var before [4][]byte
		for index, at := range ranges {
			before[index], err = loaded.snapshot.readAt(at, equippedTalismanSlotCount*4)
			if err != nil {
				return fmt.Errorf("cannot read talisman representation %d of character %d: %w",
					index+1, characterID, err)
			}
		}

		if err := validateExistingTalismanFields(loaded, inventoryAt, before, characterID); err != nil {
			return err
		}

		var after [4][]byte
		for index := range after {
			after[index] = make([]byte, equippedTalismanSlotCount*4)
		}
		for slot := 0; slot < equippedTalismanSlotCount; slot++ {
			binary.LittleEndian.PutUint32(after[0][slot*4:], removeReferenceInvalidRow)
			binary.LittleEndian.PutUint32(after[1][slot*4:], 0xFFFFFFFF)
			binary.LittleEndian.PutUint32(after[2][slot*4:], 0)
			binary.LittleEndian.PutUint32(after[3][slot*4:], 0xFFFFFFFF)
		}

		seen := make(map[uint32]int, len(orderedOwnedItemIDs))
		for slot, token := range orderedOwnedItemIDs {
			if token == "" {
				return fmt.Errorf("orderedOwnedItemIDs[%d] is empty", slot)
			}
			locator, err := loaded.session.resolveOwnedItemID(characterID, token)
			if err != nil {
				return fmt.Errorf("orderedOwnedItemIDs[%d]: %w", slot, err)
			}
			if locator.container != ownedContainerInventory ||
				locator.containerSection != InventorySectionCommon {
				return fmt.Errorf(
					"orderedOwnedItemIDs[%d]: ownedItemID %q must address Inventory common",
					slot, token)
			}

			rowAt := inventoryAt + int64(locator.physicalIndex)*inventoryHeldRecordSize
			row, err := loaded.snapshot.readAt(rowAt, inventoryHeldRecordSize)
			if err != nil {
				return fmt.Errorf("cannot read inventory row %d of character %d: %w",
					locator.physicalIndex, characterID, err)
			}
			handle := binary.LittleEndian.Uint32(row)
			quantity := binary.LittleEndian.Uint32(row[4:]) & inventoryHeldQuantityMask
			if handle&gaItemHandleTypeMask != gaItemAccessoryHandle || quantity == 0 {
				return fmt.Errorf(
					"orderedOwnedItemIDs[%d]: ownedItemID %q is not a positive-quantity talisman record",
					slot, token)
			}
			gameID, err := resolveGaItemHandle(nil, handle)
			if err != nil {
				return fmt.Errorf("orderedOwnedItemIDs[%d] ownedItemID %q: %w", slot, token, err)
			}
			if validateGameID != nil {
				if err := validateGameID(gameID); err != nil {
					return fmt.Errorf("orderedOwnedItemIDs[%d]: %w", slot, err)
				}
			}
			if previous, exists := seen[gameID]; exists {
				return fmt.Errorf("talisman 0x%08X is assigned to both slot %d and slot %d",
					gameID, previous, slot)
			}
			seen[gameID] = slot
			gameIDs[slot] = gameID

			binary.LittleEndian.PutUint32(
				after[0][slot*4:], removeReferenceInventoryRowBase+uint32(locator.physicalIndex))
			binary.LittleEndian.PutUint32(after[1][slot*4:], handle&equippedTalismanBareMask)
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
		return writeTalismanFields(loaded.snapshot, ranges, before, after, characterID)
	})
	if err != nil {
		return SetEquippedTalismansResult{}, err
	}

	return SetEquippedTalismansResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  saveRevision,
		CharacterID:   characterID,
		GameIDs:       gameIDs,
		UnlockedSlots: unlockedSlots,
	}, nil
}

func validateExistingTalismanFields(
	loaded *loadedSave,
	inventoryAt int64,
	fields [4][]byte,
	characterID int,
) error {
	for slot := 0; slot < equippedTalismanSlotCount; slot++ {
		row := binary.LittleEndian.Uint32(fields[0][slot*4:])
		bare := binary.LittleEndian.Uint32(fields[1][slot*4:])
		handle := binary.LittleEndian.Uint32(fields[2][slot*4:])
		gameID := binary.LittleEndian.Uint32(fields[3][slot*4:])
		if row == removeReferenceInvalidRow && bare == 0xFFFFFFFF && handle == 0 && gameID == 0xFFFFFFFF {
			continue
		}
		if row < removeReferenceInventoryRowBase || handle&gaItemHandleTypeMask != gaItemAccessoryHandle {
			return fmt.Errorf("talisman slot %d of character %d has inconsistent existing representations",
				slot, characterID)
		}
		physicalIndex := int(row - removeReferenceInventoryRowBase)
		if physicalIndex >= inventoryHeldCommonRecords {
			return fmt.Errorf("talisman slot %d of character %d addresses inventory row %d out of range",
				slot, characterID, physicalIndex)
		}
		inventoryRow, err := loaded.snapshot.readAt(
			inventoryAt+int64(physicalIndex)*inventoryHeldRecordSize, inventoryHeldRecordSize)
		if err != nil {
			return fmt.Errorf("cannot validate talisman slot %d of character %d: %w", slot, characterID, err)
		}
		resolved, err := resolveGaItemHandle(nil, handle)
		if err != nil || binary.LittleEndian.Uint32(inventoryRow) != handle ||
			binary.LittleEndian.Uint32(inventoryRow[4:])&inventoryHeldQuantityMask == 0 ||
			bare != handle&equippedTalismanBareMask || gameID != resolved {
			return fmt.Errorf("talisman slot %d of character %d has inconsistent existing representations",
				slot, characterID)
		}
	}
	return nil
}

func writeTalismanFields(
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
					return fmt.Errorf("cannot write talisman representation %d of character %d; rollback failed: %v: %w",
						index+1, characterID, rollbackErr, err)
				}
			}
			return fmt.Errorf("cannot write talisman representation %d of character %d: %w",
				index+1, characterID, err)
		}
		written++
	}

	for index, at := range ranges {
		got, err := snapshot.readAt(at, len(after[index]))
		if err == nil && bytes.Equal(got, after[index]) {
			continue
		}
		for rollback := range ranges {
			if rollbackErr := snapshot.writeAt(ranges[rollback], before[rollback]); rollbackErr != nil {
				return fmt.Errorf("talisman mutation of character %d could not be verified or restored: %v",
					characterID, rollbackErr)
			}
		}
		return errors.New("talisman mutation could not be verified; the save is unchanged")
	}
	return nil
}
