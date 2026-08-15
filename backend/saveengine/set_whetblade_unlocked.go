package saveengine

import (
	"errors"
	"fmt"
	"sort"
)

// WhetbladeState is the save-side identity of one catalog Whetblade. Catalog
// names and public resource keys remain the endpoint's responsibility.
type WhetbladeState struct {
	EventFlagID uint32
	GameID      uint32
}

// SetWhetbladeUnlockedResult reports one committed Whetblade state change.
type SetWhetbladeUnlockedResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	Unlocked      bool   `json:"unlocked"`
}

// SetWhetbladeUnlocked synchronises one Whetblade's main and related event
// flags with its Inventory item. The shared Ashes of War menu flag is set while
// any other Whetblade remains represented by its main flag or Inventory item.
// Bundled pickups are separate resources and are not created or removed here.
func (engine *Engine) SetWhetbladeUnlocked(
	saveSessionID string,
	characterID int,
	target WhetbladeState,
	relatedEventFlagIDs []uint32,
	otherWhetblades []WhetbladeState,
	aowMenuEventFlagID uint32,
	unlocked bool,
	expectedRevision string,
) (SetWhetbladeUnlockedResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetWhetbladeUnlockedResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if err := validateWhetbladeState(target, "target Whetblade"); err != nil {
		return SetWhetbladeUnlockedResult{}, err
	}
	if len(relatedEventFlagIDs) == 0 {
		return SetWhetbladeUnlockedResult{}, errors.New(
			"target Whetblade declares no related event flags")
	}
	for _, flagID := range relatedEventFlagIDs {
		block := flagID / eventFlagsPerBlock
		if block != 65 && block != 1042378 {
			return SetWhetbladeUnlockedResult{}, fmt.Errorf(
				"related Whetblade event flag %d lies in block %d, want block 65 or 1042378",
				flagID, block)
		}
	}
	if block := aowMenuEventFlagID / eventFlagsPerBlock; block != 65 {
		return SetWhetbladeUnlockedResult{}, fmt.Errorf(
			"AoW menu event flag %d lies in block %d, want block 65",
			aowMenuEventFlagID, block)
	}

	flagIDs := make([]uint32, 0, len(relatedEventFlagIDs)+len(otherWhetblades)+2)
	flagIDs = append(flagIDs, target.EventFlagID)
	flagIDs = append(flagIDs, relatedEventFlagIDs...)
	flagIDs = append(flagIDs, aowMenuEventFlagID)
	seenFlags := make(map[uint32]struct{}, len(flagIDs)+len(otherWhetblades))
	for _, flagID := range flagIDs {
		if _, duplicate := seenFlags[flagID]; duplicate {
			return SetWhetbladeUnlockedResult{}, fmt.Errorf(
				"Whetblade mutation repeats event flag %d", flagID)
		}
		seenFlags[flagID] = struct{}{}
		if _, err := resolveEventFlag(flagID); err != nil {
			return SetWhetbladeUnlockedResult{}, err
		}
	}
	seenGameIDs := map[uint32]struct{}{target.GameID: {}}
	for index, state := range otherWhetblades {
		if err := validateWhetbladeState(state, fmt.Sprintf("other Whetblade %d", index)); err != nil {
			return SetWhetbladeUnlockedResult{}, err
		}
		if _, duplicate := seenFlags[state.EventFlagID]; duplicate {
			return SetWhetbladeUnlockedResult{}, fmt.Errorf(
				"Whetblade mutation repeats event flag %d", state.EventFlagID)
		}
		seenFlags[state.EventFlagID] = struct{}{}
		if _, duplicate := seenGameIDs[state.GameID]; duplicate {
			return SetWhetbladeUnlockedResult{}, fmt.Errorf(
				"Whetblade mutation repeats game ID 0x%08X", state.GameID)
		}
		seenGameIDs[state.GameID] = struct{}{}
	}

	saveRevision, err := engine.commitCharacterRevision(saveSessionID, opSetWhetbladeUnlocked, characterID, func(loaded *loadedSave) error {
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
		active, err := loaded.snapshot.readAt(
			userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
		}
		if active[0] != userData10ActiveFlagValue {
			return fmt.Errorf("character %d is not active", characterID)
		}

		inventory, err := whetbladeInventoryRecords(loaded, characterID)
		if err != nil {
			return err
		}
		inventoryWrites, err := planWhetbladeInventoryState(
			loaded, characterID, inventory, target.GameID, unlocked)
		if err != nil {
			return err
		}

		sectionAt, err := eventFlagSectionStart(loaded, characterID)
		if err != nil {
			return err
		}
		desired := make(map[uint32]bool, len(relatedEventFlagIDs)+2)
		desired[target.EventFlagID] = unlocked
		for _, flagID := range relatedEventFlagIDs {
			desired[flagID] = unlocked
		}
		menuUnlocked := unlocked
		if !menuUnlocked {
			menuUnlocked, err = anotherWhetbladeIsUnlocked(
				loaded, sectionAt, inventory, otherWhetblades)
			if err != nil {
				return err
			}
		}
		desired[aowMenuEventFlagID] = menuUnlocked
		flagWrites, err := planWhetbladeEventFlagWrites(loaded, sectionAt, desired)
		if err != nil {
			return err
		}

		writes := append(inventoryWrites, flagWrites...)
		if err := applyByteWrites(loaded.snapshot, writes); err != nil {
			return fmt.Errorf("cannot set Whetblade 0x%08X: %w", target.GameID, err)
		}
		return nil
	})
	if err != nil {
		return SetWhetbladeUnlockedResult{}, err
	}
	return SetWhetbladeUnlockedResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  saveRevision,
		CharacterID:   characterID,
		Unlocked:      unlocked,
	}, nil
}

func validateWhetbladeState(state WhetbladeState, field string) error {
	block := state.EventFlagID / eventFlagsPerBlock
	if block != 60 && block != 65 {
		return fmt.Errorf("%s event flag %d lies in block %d, want block 60 or 65",
			field, state.EventFlagID, block)
	}
	if _, err := resolveEventFlag(state.EventFlagID); err != nil {
		return err
	}
	if state.GameID&gaItemHandleTypeMask != 0x40000000 {
		return fmt.Errorf("%s game ID 0x%08X is not a goods ID", field, state.GameID)
	}
	return nil
}

func whetbladeInventoryRecords(
	loaded *loadedSave, characterID int,
) ([]InventoryRecord, error) {
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

func planWhetbladeInventoryState(
	loaded *loadedSave,
	characterID int,
	records []InventoryRecord,
	gameID uint32,
	unlocked bool,
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
				"Whetblade 0x%08X has a zero-quantity Inventory record", gameID)
		}
		matches = append(matches, record)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("Whetblade 0x%08X has %d Inventory records, want at most 1",
			gameID, len(matches))
	}
	if unlocked {
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
	description := fmt.Sprintf("Whetblade 0x%08X", gameID)
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
		return nil, fmt.Errorf("Whetblade record states %d bytes and Inventory stores %d",
			len(plan.cleared), recordSize)
	}
	writes := []byteWrite{{at: recordAt, data: plan.cleared}}
	if !plan.maintainsCount {
		return writes, nil
	}
	count, err := loaded.snapshot.uint32At(countAt)
	if err != nil {
		return nil, fmt.Errorf("cannot read Whetblade Inventory count: %w", err)
	}
	if count > 0 {
		writes = append(writes, byteWrite{at: countAt, data: littleEndianUint32(count - 1)})
	}
	return writes, nil
}

func anotherWhetbladeIsUnlocked(
	loaded *loadedSave,
	sectionAt int64,
	records []InventoryRecord,
	others []WhetbladeState,
) (bool, error) {
	handles := make(map[uint32]struct{}, len(others)*2)
	for _, state := range others {
		position, _ := resolveEventFlag(state.EventFlagID)
		raw, err := loaded.snapshot.readAt(sectionAt+position.offset, 1)
		if err != nil {
			return false, fmt.Errorf("cannot read event flag %d: %w", state.EventFlagID, err)
		}
		if raw[0]&(1<<position.bit) != 0 {
			return true, nil
		}
		handle, _ := gaItemHandleForGameID(state.GameID)
		handles[state.GameID] = struct{}{}
		handles[handle] = struct{}{}
	}
	for _, record := range records {
		if record.Quantity == 0 {
			continue
		}
		if _, present := handles[record.GaItemHandle]; present {
			return true, nil
		}
	}
	return false, nil
}

func planWhetbladeEventFlagWrites(
	loaded *loadedSave,
	sectionAt int64,
	desired map[uint32]bool,
) ([]byteWrite, error) {
	byOffset := make(map[int64]byte)
	for flagID, value := range desired {
		position, _ := resolveEventFlag(flagID)
		at := sectionAt + position.offset
		current, exists := byOffset[at]
		if !exists {
			raw, err := loaded.snapshot.readAt(at, 1)
			if err != nil {
				return nil, fmt.Errorf("cannot read event flag %d: %w", flagID, err)
			}
			current = raw[0]
		}
		if value {
			current |= 1 << position.bit
		} else {
			current &^= 1 << position.bit
		}
		byOffset[at] = current
	}
	offsets := make([]int64, 0, len(byOffset))
	for at := range byOffset {
		offsets = append(offsets, at)
	}
	sort.Slice(offsets, func(i, j int) bool { return offsets[i] < offsets[j] })
	writes := make([]byteWrite, 0, len(offsets))
	for _, at := range offsets {
		writes = append(writes, byteWrite{at: at, data: []byte{byOffset[at]}})
	}
	return writes, nil
}
