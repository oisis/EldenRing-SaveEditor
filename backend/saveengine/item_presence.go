package saveengine

import "fmt"

// This file owns the one InventoryHeld presence rule the domain mutations that
// pair an event flag with a single companion item share: SetWhetbladeUnlocked
// and SetMapRegionRevealed. Both need the same thing — make one goods item
// present or absent — and both have to express it as a byte plan so their flag
// writes and their item write commit or roll back together.
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

// planItemPresence describes the writes that make gameID present in or absent
// from the InventoryHeld of one character, without touching the snapshot. label
// names the domain the item belongs to so a rejection reads as the caller's own
// failure rather than as a generic inventory error.
//
// A record is matched on the raw game ID as well as on the computed handle,
// because the game stores key items under the ID itself while an editor-created
// common record carries the derived handle. More than one record of the item is
// a fail-closed error: this plan owns exactly one record and never guesses which
// of two copies the state belongs to.
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
	handle, err := gaItemHandleForGameID(gameID)
	if err != nil {
		return nil, err
	}
	matches := make([]InventoryRecord, 0, 1)
	for _, record := range records {
		if record.GaItemHandle != gameID && record.GaItemHandle != handle {
			continue
		}
		if record.Quantity == 0 {
			return nil, fmt.Errorf(
				"%s 0x%08X has a zero-quantity Inventory record", label, gameID)
		}
		matches = append(matches, record)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("%s 0x%08X has %d Inventory records, want at most 1",
			label, gameID, len(matches))
	}
	if present {
		if len(matches) == 1 {
			return nil, nil
		}
		_, writes, err := planInventoryRecordCreation(
			loaded, characterID, handle, gameID, 1, false)
		return writes, err
	}
	if len(matches) == 0 {
		return nil, nil
	}

	record := matches[0]
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
	writes := []byteWrite{{at: recordAt, data: plan.cleared}}
	if !plan.maintainsCount {
		return writes, nil
	}
	count, err := loaded.snapshot.uint32At(countAt)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s Inventory count: %w", label, err)
	}
	if count > 0 {
		writes = append(writes, byteWrite{at: countAt, data: littleEndianUint32(count - 1)})
	}
	return writes, nil
}
