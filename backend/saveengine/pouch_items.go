package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Slot-data layout of the confirmed EquipItemData section, shared by PC and PS4.
// The section has no fixed position inside a slot, so it is located through the
// confirmed anchor of the slot and read forwards from it across a chain of fixed
// structures only.
const (
	// pouchItemSlotCount is the number of raw pouch records. The section stores
	// exactly these six pairs; the bytes behind them are never touched by this
	// getter.
	pouchItemSlotCount  = 6
	pouchItemRecordSize = 8

	// pouchItemsSectionOffset is the distance from the anchor to the start of the
	// pouch records. It is the distance from the anchor to EquipItemData:
	//
	//	0x00D0 SpEffect
	//	0x0058 EquipedItemIndex
	//	0x001C ActiveEquipedItems
	//	0x0058 EquipedItemsID
	//	0x0058 ActiveEquipedItemsGa
	//	0x9011 InventoryHeld
	//	0x0074 EquippedSpells
	//
	// that is 0x9279, plus the fixed head of that section: the ten eight-byte
	// quick-item records (0x50) and the four-byte active-quick value behind them.
	// Every one of those structures has a fixed size, so this distance is
	// constant: unlike the equipped-armaments block, nothing variable-length lies
	// in front of the pouch records.
	pouchItemsSectionOffset = 0x9279 + 0x50 + 4

	// pouchItemsReadSize is the range this getter needs inside the slot: the six
	// records and nothing behind them.
	pouchItemsReadSize = pouchItemSlotCount * pouchItemRecordSize
)

// pouchItemsAnchor is the confirmed 65-byte marker the pouch chain is measured
// from: one leading 0x00 byte, then four full repetitions of a 16-byte block
// made of 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes. It is stated here
// for this getter alone, the way every other slot reader states the marker it
// depends on.
var pouchItemsAnchor = []byte{
	0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// PouchItemSlot is one raw pouch record exactly as stored: the item field
// followed by the equip-index field, both little-endian uint32. Neither value is
// normalised, masked, validated or resolved to an item, so the empty-slot
// sentinel and every other stored combination is reported as written.
type PouchItemSlot struct {
	ItemID     uint32 `json:"itemID"`
	EquipIndex uint32 `json:"equipIndex"`
}

// CharacterPouchItems is the raw pouch state of one physical save slot: the six
// EquipItemData records that follow the quick items, exactly as stored. No value
// is normalised, validated or resolved to a name, and no GameCatalog is read.
//
// Items holds the records in their stored order, which is the order of the six
// pouch positions in the game.
//
// Every field except SaveSessionID and CharacterID describes an active slot
// only. An inactive slot — including a residual one, whose deleted character's
// pouch items are still in the file — reports Active false and six zeroed
// records, and its slot data is never searched or read.
type CharacterPouchItems struct {
	SaveSessionID string                            `json:"saveSessionID"`
	CharacterID   int                               `json:"characterID"`
	Active        bool                              `json:"active"`
	Items         [pouchItemSlotCount]PouchItemSlot `json:"items"`
}

// GetPouchItems returns the raw pouch state stored in one physical character
// slot of an existing session. Like the other character readers it reads the
// session's private snapshot through the codec only: it opens no file, writes
// nothing, changes no session and returns no snapshot byte.
//
// saveSessionID is matched exactly. It is never trimmed, normalised or guessed,
// so an empty, unknown or already closed identifier is rejected instead of
// resolving to a session.
//
// characterID is the slot index 0..9. An inactive or residual slot is a normal
// result, not an error, and its slot data is never read.
//
// For an active slot the records are located through the confirmed anchor of
// that one slot and read at a constant distance behind it. A missing anchor and
// a required range reaching past the end of the slot or of the snapshot are hard
// errors. There is no fallback position, no partial result and nothing is
// guessed.
func (engine *Engine) GetPouchItems(saveSessionID string, characterID int) (CharacterPouchItems, error) {
	if saveSessionID == "" {
		return CharacterPouchItems{}, errors.New("saveSessionID is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterPouchItems{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterPouchItems{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterPouchItems{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}

	pouchItems := CharacterPouchItems{SaveSessionID: saveSessionID, CharacterID: characterID}
	if flag[0] != userData10ActiveFlagValue {
		// An inactive slot is reported from its flag alone, so the residual pouch
		// items of a deleted character are never located or decoded.
		return pouchItems, nil
	}

	items, err := readPouchItems(loaded, characterID)
	if err != nil {
		return CharacterPouchItems{}, err
	}

	pouchItems.Active = true
	pouchItems.Items = items
	return pouchItems, nil
}

// readPouchItems is the single decoder of the six EquipItemData pouch pairs.
// The caller must hold Engine.mutex and establish that the slot is active.
func readPouchItems(loaded *loadedSave, characterID int) ([pouchItemSlotCount]PouchItemSlot, error) {
	var items [pouchItemSlotCount]PouchItemSlot
	base := slotDataBase(loaded.session.platform, characterID)
	slotEnd := base + characterSlotDataSize

	anchor, err := loaded.snapshot.indexIn(base, characterSlotDataSize, pouchItemsAnchor)
	if err != nil {
		return items, fmt.Errorf("cannot search the pouch items of character %d: %w", characterID, err)
	}
	if anchor < 0 {
		return items, fmt.Errorf("character %d carries no pouch-items anchor", characterID)
	}

	sectionAt := anchor + pouchItemsSectionOffset
	if sectionAt+pouchItemsReadSize > slotEnd {
		return items, fmt.Errorf("pouch items of character %d do not fit into its slot", characterID)
	}
	section, err := loaded.snapshot.readAt(sectionAt, pouchItemsReadSize)
	if err != nil {
		return items, fmt.Errorf("cannot read pouch items of character %d: %w", characterID, err)
	}

	for index := range items {
		record := section[index*pouchItemRecordSize:]
		items[index] = PouchItemSlot{
			ItemID:     binary.LittleEndian.Uint32(record),
			EquipIndex: binary.LittleEndian.Uint32(record[4:]),
		}
	}
	return items, nil
}
