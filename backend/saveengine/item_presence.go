package saveengine

import (
	"fmt"
	"sort"
)

// This file owns the one InventoryHeld presence rule the domain mutations that
// pair event flags with companion items share: SetWhetbladeUnlocked,
// SetMapRegionRevealed and the two Spectral Steed Attire mutations. They all
// need the same thing — make goods items present or absent — and they all have
// to express it as a byte plan so their flag writes and their item writes commit
// or roll back together.
//
// It allocates no GaItem record and writes no key record, so it inherits every
// limit of planInventoryRecordCreation and planOwnedItemRemoval. It is not a
// general item API: AddItemToInventory and RemoveOwnedItem stay the public ones.

// inventoryHeldRecords reads every InventoryHeld record of one character, the
// common section followed by the key section, through the shared record reader.
//
// The caller must already hold Engine.mutex.
func inventoryHeldRecords(loaded *loadedSave, characterID int) ([]InventoryRecord, error) {
	sectionAt, err := inventoryHeldSectionAt(loaded, characterID)
	if err != nil {
		return nil, err
	}
	section, err := loaded.snapshot.readAt(sectionAt, inventoryHeldSectionSize)
	if err != nil {
		return nil, fmt.Errorf("cannot read inventory of character %d: %w", characterID, err)
	}
	keyEnd := inventoryHeldSectionSize - inventoryHeldTrailingCounters
	records := appendInventoryRecords(
		make([]InventoryRecord, 0), section[:inventoryHeldCommonSize], InventorySectionCommon)
	return appendInventoryRecords(
		records, section[inventoryHeldKeyAt:keyEnd], InventorySectionKey), nil
}

// matchInventoryHeldRecord finds the single InventoryHeld record of one goods
// game ID. label names the domain the item belongs to so a rejection reads as
// the caller's own failure rather than as a generic inventory error.
//
// A record is matched on the raw game ID as well as on the computed handle,
// because the game stores key items under the ID itself while an editor-created
// common record carries the derived handle. More than one record of the item, and
// a record with quantity zero, are fail-closed errors: this rule owns exactly one
// positive record and never guesses which of two copies a state belongs to.
func matchInventoryHeldRecord(
	records []InventoryRecord, gameID uint32, label string,
) (InventoryRecord, bool, error) {
	handle, err := gaItemHandleForGameID(gameID)
	if err != nil {
		return InventoryRecord{}, false, err
	}
	matches := make([]InventoryRecord, 0, 1)
	for _, record := range records {
		if record.GaItemHandle != gameID && record.GaItemHandle != handle {
			continue
		}
		if record.Quantity == 0 {
			return InventoryRecord{}, false, fmt.Errorf(
				"%s 0x%08X has a zero-quantity Inventory record", label, gameID)
		}
		matches = append(matches, record)
	}
	if len(matches) > 1 {
		return InventoryRecord{}, false, fmt.Errorf("%s 0x%08X has %d Inventory records, want at most 1",
			label, gameID, len(matches))
	}
	if len(matches) == 0 {
		return InventoryRecord{}, false, nil
	}
	return matches[0], true, nil
}

// planItemPresence describes the writes that make gameID present in or absent
// from the InventoryHeld of one character, without touching the snapshot.
//
// The item is already in the wanted state when no write is needed, and the empty
// plan the caller then receives is what keeps a repeated call from moving a byte.
//
// The caller must already hold Engine.mutex.
func planItemPresence(
	loaded *loadedSave,
	characterID int,
	records []InventoryRecord,
	gameID uint32,
	present bool,
	label string,
) ([]byteWrite, error) {
	if !present {
		return planItemRemovals(loaded, characterID, records, []uint32{gameID}, label)
	}
	_, found, err := matchInventoryHeldRecord(records, gameID, label)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, nil
	}
	handle, err := gaItemHandleForGameID(gameID)
	if err != nil {
		return nil, err
	}
	_, writes, err := planInventoryRecordCreation(loaded, characterID, handle, gameID, 1, false)
	return writes, err
}

// planItemRemovals describes the writes that remove every listed goods game ID
// from the InventoryHeld of one character as one plan. A game ID that has no
// record contributes nothing, so the plan stays empty when nothing is there.
//
// The section counters are folded per counter instead of per record: two removals
// from the same section share one count field, and two writes to that one field
// would neither be a non-overlapping plan nor arrive at the right number.
//
// The caller must already hold Engine.mutex.
func planItemRemovals(
	loaded *loadedSave,
	characterID int,
	records []InventoryRecord,
	gameIDs []uint32,
	label string,
) ([]byteWrite, error) {
	writes := make([]byteWrite, 0, len(gameIDs)+1)
	removalsPerCount := make(map[int64]uint32, len(gameIDs))
	for _, gameID := range gameIDs {
		record, found, err := matchInventoryHeldRecord(records, gameID, label)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}

		locator := ownedItemLocator{
			characterID:      characterID,
			container:        ownedContainerInventory,
			containerSection: record.ContainerSection,
			physicalIndex:    record.PhysicalIndex,
		}
		description := fmt.Sprintf("%s 0x%08X", label, gameID)
		plan, err := planOwnedItemRemoval(locator, description)
		if err != nil {
			return nil, err
		}
		if err := rejectReferencedOwnedItem(
			loaded, locator, record.GaItemHandle, description); err != nil {
			return nil, err
		}
		recordAt, countAt, recordSize, err := ownedItemRowAndCountAt(loaded, locator)
		if err != nil {
			return nil, err
		}
		if int64(len(plan.cleared)) != recordSize {
			return nil, fmt.Errorf("%s record states %d bytes and Inventory stores %d",
				label, len(plan.cleared), recordSize)
		}
		writes = append(writes, byteWrite{at: recordAt, data: plan.cleared})
		if plan.maintainsCount {
			removalsPerCount[countAt]++
		}
	}

	countOffsets := make([]int64, 0, len(removalsPerCount))
	for at := range removalsPerCount {
		countOffsets = append(countOffsets, at)
	}
	sort.Slice(countOffsets, func(i, j int) bool { return countOffsets[i] < countOffsets[j] })
	for _, countAt := range countOffsets {
		count, err := loaded.snapshot.uint32At(countAt)
		if err != nil {
			return nil, fmt.Errorf("cannot read %s Inventory count: %w", label, err)
		}
		if count == 0 {
			continue
		}
		removed := removalsPerCount[countAt]
		if removed > count {
			removed = count
		}
		writes = append(writes, byteWrite{at: countAt, data: littleEndianUint32(count - removed)})
	}
	return writes, nil
}
