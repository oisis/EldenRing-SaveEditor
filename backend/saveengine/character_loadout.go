package saveengine

import (
	"encoding/binary"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// CharacterLoadoutOwnedItem is one validated Quick Items or Pouch reference.
// An empty position is the zero value. An occupied position carries the exact
// revision-scoped Inventory common identity and quantity of the referenced row,
// plus the game ID confirmed by both the GaItem handle and the armaments tail.
type CharacterLoadoutOwnedItem struct {
	GameID      uint32
	OwnedItemID string
	Quantity    uint32
}

// CharacterLoadoutSnapshot is one coherent, read-only snapshot of every
// confirmed player-facing loadout group. It is internal SaveEngine domain data:
// GameCatalog presentation and public slot labels belong to the endpoint.
type CharacterLoadoutSnapshot struct {
	SaveSessionID string
	SaveRevision  string
	CharacterID   int
	Active        bool
	Equipment     [equipmentSlotCount]uint32
	// EquipmentOwned carries the revision-scoped Inventory common identity of
	// every occupied hand, armor and talisman position, indexed exactly like
	// Equipment. Positions without an owned identity — the ammunition fields,
	// technically empty records and talisman positions above the unlocked count
	// — stay the empty string. It is resolved from the confirmed EquipedItemIndex
	// and ActiveEquipedItemsGa references the three armament writers already
	// maintain and is accepted only when the GaItem table resolves that handle to
	// the very game ID the position presents, never by searching the container
	// for a matching game ID.
	EquipmentOwned       [equipmentSlotCount]string
	QuickItems           [quickItemSlotCount]CharacterLoadoutOwnedItem
	Pouch                [pouchItemSlotCount]CharacterLoadoutOwnedItem
	ActiveQuickItem      int32
	Physick              [physickTearCount]uint32
	Spells               [spellMaxMemorySlots]uint32
	ActiveSpellIndex     int
	AvailableMemorySlots int
	// MemoryStones is the effective Memory Stone count AvailableMemorySlots was
	// computed from, already capped by the game maximum. It is the same value,
	// read once, never a second count of the same items.
	MemoryStones          uint32
	UnlockedTalismanSlots int
}

// IsTechnicalEmptyEquipmentSlot reports the confirmed native empty value for
// one public ChrAsmEquipment position. SaveEngine owns this save-format rule;
// presentation layers must not duplicate the technical game IDs.
func IsTechnicalEmptyEquipmentSlot(index int, gameID uint32) bool {
	switch {
	case index >= 0 && index < 6:
		return gameID == unarmedEquipmentGameID
	case index >= 6 && index < 10:
		return gameID == loadoutEquipmentEmptyID
	case index >= equippedArmorFirstSlot && index < equippedArmorFirstSlot+equippedArmorSlotCount:
		return gameID == equippedArmorEmptyGameIDs[index-equippedArmorFirstSlot]
	case index >= 17 && index < 21:
		return gameID == loadoutEquipmentEmptyID
	default:
		return false
	}
}

const loadoutEquipmentEmptyID uint32 = 0xFFFFFFFF

// GetCharacterLoadoutSnapshot reads one character loadout under one Engine
// mutex acquisition. It reuses the same low-level decoders as the five public
// raw getters and validates the cross-structure Quick Items and Pouch
// references before returning anything. It opens no file and mutates no state.
func (engine *Engine) GetCharacterLoadoutSnapshot(
	saveSessionID string,
	characterID int,
) (CharacterLoadoutSnapshot, error) {
	if saveSessionID == "" {
		return CharacterLoadoutSnapshot{}, apperror.MissingField("saveSessionID")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterLoadoutSnapshot{}, apperror.UnknownSaveSession(saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterLoadoutSnapshot{}, fmt.Errorf(
			"characterID %d is outside the range 0..%d", characterID, characterSlotCount-1)
	}

	result := CharacterLoadoutSnapshot{
		SaveSessionID:    saveSessionID,
		SaveRevision:     loaded.session.revisionString(),
		CharacterID:      characterID,
		ActiveSpellIndex: -1,
	}
	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterLoadoutSnapshot{}, fmt.Errorf(
			"cannot read activity of character %d: %w", characterID, err)
	}
	if flag[0] != userData10ActiveFlagValue {
		return result, nil
	}

	equipment, blockAt, slotEnd, err := readEquipmentSlots(loaded, characterID)
	if err != nil {
		return CharacterLoadoutSnapshot{}, err
	}
	quick, activeQuick, err := readQuickItems(loaded, characterID)
	if err != nil {
		return CharacterLoadoutSnapshot{}, err
	}
	pouch, err := readPouchItems(loaded, characterID)
	if err != nil {
		return CharacterLoadoutSnapshot{}, err
	}
	physick, err := readPhysickMixture(loaded, characterID)
	if err != nil {
		return CharacterLoadoutSnapshot{}, err
	}
	spellState, err := readEquippedSpellsState(loaded, characterID)
	if err != nil {
		return CharacterLoadoutSnapshot{}, err
	}
	if spellState.activeSpellIndex >= spellMaxMemorySlots ||
		(spellState.activeSpellIndex >= 0 &&
			spellState.records[spellState.activeSpellIndex] == equippedSpellEmptyID) {
		return CharacterLoadoutSnapshot{}, fmt.Errorf(
			"active spell index of character %d is %d, which does not address an occupied public spell position",
			characterID, spellState.activeSpellIndex)
	}

	quickResolved, err := resolveCharacterLoadoutOwnedItems(
		loaded, characterID, "quick item", quickItemSlots(quick), blockAt+quickItemsTailOffset,
		slotEnd, quickItemSlotCount, QuickItemEmptyGameID)
	if err != nil {
		return CharacterLoadoutSnapshot{}, err
	}
	pouchResolved, err := resolveCharacterLoadoutOwnedItems(
		loaded, characterID, "pouch", pouchItemSlots(pouch), blockAt+pouchItemsTailOffset,
		slotEnd, pouchItemSlotCount, PouchEmptyGameID)
	if err != nil {
		return CharacterLoadoutSnapshot{}, err
	}

	owned, err := resolveCharacterLoadoutEquipmentOwners(
		loaded, characterID, equipment, spellState.unlockedTalismanSlots)
	if err != nil {
		return CharacterLoadoutSnapshot{}, err
	}

	result.Active = true
	result.Equipment = equipment
	result.EquipmentOwned = owned
	copy(result.QuickItems[:], quickResolved)
	copy(result.Pouch[:], pouchResolved)
	result.ActiveQuickItem = activeQuick
	result.Physick = physick
	copy(result.Spells[:], spellState.records[:spellMaxMemorySlots])
	result.ActiveSpellIndex = spellState.activeSpellIndex
	result.AvailableMemorySlots = spellState.availableMemorySlots
	result.MemoryStones = spellState.memoryStones
	result.UnlockedTalismanSlots = spellState.unlockedTalismanSlots
	return result, nil
}

type characterLoadoutOwnedPair struct {
	itemID     uint32
	equipIndex uint32
}

func quickItemSlots(items [quickItemSlotCount]QuickItemSlot) []characterLoadoutOwnedPair {
	result := make([]characterLoadoutOwnedPair, len(items))
	for index, item := range items {
		result[index] = characterLoadoutOwnedPair{itemID: item.ItemID, equipIndex: item.EquipIndex}
	}
	return result
}

func pouchItemSlots(items [pouchItemSlotCount]PouchItemSlot) []characterLoadoutOwnedPair {
	result := make([]characterLoadoutOwnedPair, len(items))
	for index, item := range items {
		result[index] = characterLoadoutOwnedPair{itemID: item.ItemID, equipIndex: item.EquipIndex}
	}
	return result
}

// resolveCharacterLoadoutOwnedItems validates the native triple shared by Quick
// Items and Pouch: EquipItemData handle, Inventory common row and armaments-tail
// game ID. There is no fallback search and no partial result.
func resolveCharacterLoadoutOwnedItems(
	loaded *loadedSave,
	characterID int,
	label string,
	pairs []characterLoadoutOwnedPair,
	tailAt int64,
	slotEnd int64,
	slotCount int,
	emptyGameID uint32,
) ([]CharacterLoadoutOwnedItem, error) {
	if len(pairs) != slotCount {
		return nil, fmt.Errorf("%s positions contain %d records, want %d", label, len(pairs), slotCount)
	}
	if tailAt < 0 || tailAt+int64(slotCount*4) > slotEnd {
		return nil, fmt.Errorf("%s armaments tail of character %d does not fit into its slot", label, characterID)
	}
	tail, err := loaded.snapshot.readAt(tailAt, slotCount*4)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s armaments tail of character %d: %w", label, characterID, err)
	}
	records, err := readInventoryRecords(loaded, characterID)
	if err != nil {
		return nil, err
	}
	common := make(map[int]InventoryRecord)
	for _, record := range records {
		if record.ContainerSection == InventorySectionCommon {
			common[record.PhysicalIndex] = record
		}
	}

	resolved := make([]CharacterLoadoutOwnedItem, slotCount)
	for index, pair := range pairs {
		gameID := binary.LittleEndian.Uint32(tail[index*4:])
		if pair.itemID == inventoryHeldEmptyHandle &&
			pair.equipIndex == removeReferenceInvalidRow && gameID == emptyGameID {
			continue
		}
		if pair.itemID == inventoryHeldEmptyHandle || pair.itemID == inventoryHeldInvalidHandle ||
			pair.itemID&gaItemHandleTypeMask != gaItemGoodsHandle ||
			pair.equipIndex < removeReferenceInventoryRowBase {
			return nil, inconsistentLoadoutOwnedItemError(label, index, pair, gameID)
		}
		physicalIndex := int(pair.equipIndex - removeReferenceInventoryRowBase)
		if physicalIndex < 0 || physicalIndex >= inventoryHeldCommonRecords {
			return nil, inconsistentLoadoutOwnedItemError(label, index, pair, gameID)
		}
		record, exists := common[physicalIndex]
		if !exists || record.GaItemHandle != pair.itemID || record.Quantity == 0 {
			return nil, inconsistentLoadoutOwnedItemError(label, index, pair, gameID)
		}
		resolvedGameID, err := resolveGaItemHandle(nil, pair.itemID)
		if err != nil || resolvedGameID != gameID {
			return nil, inconsistentLoadoutOwnedItemError(label, index, pair, gameID)
		}
		resolved[index] = CharacterLoadoutOwnedItem{
			GameID:      gameID,
			OwnedItemID: record.OwnedItemID,
			Quantity:    record.Quantity,
		}
	}
	return resolved, nil
}

func inconsistentLoadoutOwnedItemError(
	label string,
	index int,
	pair characterLoadoutOwnedPair,
	gameID uint32,
) error {
	return fmt.Errorf(
		"%s slot %d: inconsistent existing save state (handle=0x%08X, equipIndex=0x%08X, tailGameID=0x%08X)",
		label, index, pair.itemID, pair.equipIndex, gameID)
}

// characterLoadoutOwnedPosition is one public equipment position backed by an
// Inventory common record, with the GaItem handle type its group requires. The
// ammunition fields are deliberately absent: no confirmed writer addresses them,
// so this getter states no owned identity for them either.
type characterLoadoutOwnedPosition struct {
	index      int
	handleType uint32
}

var characterLoadoutOwnedPositions = []characterLoadoutOwnedPosition{
	{index: 0, handleType: gaItemWeaponHandle},
	{index: 1, handleType: gaItemWeaponHandle},
	{index: 2, handleType: gaItemWeaponHandle},
	{index: 3, handleType: gaItemWeaponHandle},
	{index: 4, handleType: gaItemWeaponHandle},
	{index: 5, handleType: gaItemWeaponHandle},
	{index: equippedArmorFirstSlot, handleType: gaItemArmorHandle},
	{index: equippedArmorFirstSlot + 1, handleType: gaItemArmorHandle},
	{index: equippedArmorFirstSlot + 2, handleType: gaItemArmorHandle},
	{index: equippedArmorFirstSlot + 3, handleType: gaItemArmorHandle},
	{index: equippedTalismanFirstSlot, handleType: gaItemAccessoryHandle},
	{index: equippedTalismanFirstSlot + 1, handleType: gaItemAccessoryHandle},
	{index: equippedTalismanFirstSlot + 2, handleType: gaItemAccessoryHandle},
	{index: equippedTalismanFirstSlot + 3, handleType: gaItemAccessoryHandle},
}

// resolveCharacterLoadoutEquipmentOwners resolves the owned-item identity of
// every occupied hand, armor and talisman position from the two confirmed
// reference blocks the three armament writers maintain in front of
// InventoryHeld: EquipedItemIndex, which stores removeReferenceInventoryRowBase
// plus the physical row of the referenced Inventory common record, and
// ActiveEquipedItemsGa, which stores that record's exact GaItem handle.
//
// Both must agree with the record they address, the handle type must be the one
// the group requires, and the handle must resolve to exactly the game ID the
// ChrAsmEquipment field of that position carries. Resolution goes through the
// shared resolveGaItemHandle, which reads the GaItem table for the handle types
// that require a record — the weapon and armor handles of the hand and armor
// positions — while an accessory handle is resolved from the handle itself
// without a table record. The armament and armor writers read that same table;
// the talisman writer uses the same resolver but resolves its handle without it.
//
// This is not the complete pre-write validation those writers perform: the bare
// representation of the equipped item is not checked here.
//
// Nothing is matched by game ID, nothing falls back to a similar record and
// nothing is guessed on a duplicate: a position whose reference, handle, handle
// type, quantity or resolved game ID disagrees fails the whole getter, so a name
// is never reported next to the owned identity of a different item.
func resolveCharacterLoadoutEquipmentOwners(
	loaded *loadedSave,
	characterID int,
	equipment [equipmentSlotCount]uint32,
	unlockedTalismanSlots int,
) ([equipmentSlotCount]string, error) {
	var owned [equipmentSlotCount]string

	sectionAt, err := inventoryHeldSectionAt(loaded, characterID)
	if err != nil {
		return owned, err
	}
	anchor := sectionAt - inventoryHeldCommonOffset
	rawIndexes, err := loaded.snapshot.readAt(anchor+removeEquipmentIndexesOffset, equipmentBlockSize)
	if err != nil {
		return owned, fmt.Errorf(
			"cannot read equipment references of character %d: %w", characterID, err)
	}
	rawHandles, err := loaded.snapshot.readAt(anchor+removeEquipmentHandlesOffset, equipmentBlockSize)
	if err != nil {
		return owned, fmt.Errorf(
			"cannot read equipment handles of character %d: %w", characterID, err)
	}
	byHandle, err := readGaItemMap(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return owned, fmt.Errorf(
			"cannot resolve equipped items of character %d: %w", characterID, err)
	}
	records, err := readInventoryRecords(loaded, characterID)
	if err != nil {
		return owned, err
	}
	common := make(map[int]InventoryRecord, len(records))
	for _, record := range records {
		if record.ContainerSection == InventorySectionCommon {
			common[record.PhysicalIndex] = record
		}
	}

	for _, position := range characterLoadoutOwnedPositions {
		index := position.index
		gameID := equipment[index]
		if IsTechnicalEmptyEquipmentSlot(index, gameID) {
			continue
		}
		// A talisman position above the unlocked count is reported as locked and
		// its residual value is never presented, so no identity is minted for it.
		if index >= equippedTalismanFirstSlot &&
			index-equippedTalismanFirstSlot >= unlockedTalismanSlots {
			continue
		}
		row := binary.LittleEndian.Uint32(rawIndexes[index*4:])
		handle := binary.LittleEndian.Uint32(rawHandles[index*4:])
		if row < removeReferenceInventoryRowBase ||
			handle&gaItemHandleTypeMask != position.handleType {
			return owned, inconsistentLoadoutEquipmentError(index, row, handle, gameID)
		}
		physicalIndex := int(row - removeReferenceInventoryRowBase)
		if physicalIndex >= inventoryHeldCommonRecords {
			return owned, inconsistentLoadoutEquipmentError(index, row, handle, gameID)
		}
		record, exists := common[physicalIndex]
		if !exists || record.Quantity == 0 || record.GaItemHandle != handle {
			return owned, inconsistentLoadoutEquipmentError(index, row, handle, gameID)
		}
		// The referenced record must be the item the position actually presents:
		// its handle has to resolve to the very game ID stored in ChrAsmEquipment.
		resolved, resolveErr := resolveGaItemHandle(byHandle, handle)
		if resolveErr != nil || resolved != gameID {
			return owned, inconsistentLoadoutEquipmentError(index, row, handle, gameID)
		}
		owned[index] = record.OwnedItemID
	}
	return owned, nil
}

func inconsistentLoadoutEquipmentError(index int, row, handle, gameID uint32) error {
	return fmt.Errorf(
		"equipment position %d: inconsistent existing save state "+
			"(rowField=0x%08X, handle=0x%08X, equipmentGameID=0x%08X)",
		index, row, handle, gameID)
}
