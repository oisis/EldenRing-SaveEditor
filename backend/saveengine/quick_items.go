package saveengine

import (
	"encoding/binary"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// Slot-data layout of the confirmed EquipItemData section, shared by PC and PS4.
// The section has no fixed position inside a slot, so it is located through the
// confirmed anchor of the slot and read forwards from it across a chain of fixed
// structures only.
const (
	// quickItemSlotCount is the number of raw quick-item records. The section
	// stores exactly these ten pairs; the bytes behind them are never touched by
	// this getter.
	quickItemSlotCount  = 10
	quickItemRecordSize = 8

	// quickItemsSectionOffset is the distance from the anchor to the start of
	// EquipItemData. It is the sum of the confirmed fixed structures between the
	// two positions:
	//
	//	0x00D0 SpEffect
	//	0x0058 EquipedItemIndex
	//	0x001C ActiveEquipedItems
	//	0x0058 EquipedItemsID
	//	0x0058 ActiveEquipedItemsGa
	//	0x9011 InventoryHeld
	//	0x0074 EquippedSpells
	//
	// Every one of them has a fixed size, so this distance is constant: unlike
	// the equipped-armaments block, nothing variable-length lies in front of
	// EquipItemData.
	quickItemsSectionOffset = 0x9279

	// quickItemsActiveOffset is where the raw active-slot int32 lies, measured
	// from the start of EquipItemData: immediately behind the ten records.
	quickItemsActiveOffset = quickItemSlotCount * quickItemRecordSize

	// quickItemsReadSize is the range this getter needs inside the slot: the ten
	// records plus the active-slot value behind them.
	quickItemsReadSize = quickItemsActiveOffset + 4
)

// quickItemsAnchor is the confirmed 65-byte marker the quick-items chain is
// measured from: one leading 0x00 byte, then four full repetitions of a 16-byte
// block made of 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes. It is stated
// here for this getter alone, the way every other slot reader states the marker
// it depends on.
var quickItemsAnchor = []byte{
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

// QuickItemSlot is one raw quick-item record exactly as stored: the item field
// followed by the equip-index field, both little-endian uint32. Neither value is
// normalised, masked, validated or resolved to an item, so the empty-slot
// sentinel and every other stored combination is reported as written.
type QuickItemSlot struct {
	ItemID     uint32 `json:"itemID"`
	EquipIndex uint32 `json:"equipIndex"`
}

// CharacterQuickItems is the raw quick-item state of one physical save slot: the
// ten EquipItemData records and the active-slot value behind them, exactly as
// stored. No value is normalised, validated or resolved to a name, and no
// GameCatalog is read.
//
// Items holds the records in their stored order, which is the order of the ten
// quick-item positions in the game.
//
// ActiveQuick is the stored int32 and keeps its sign: a negative value is
// reported as it is written, not clamped to a valid position.
//
// Every field except SaveSessionID and CharacterID describes an active slot
// only. An inactive slot — including a residual one, whose deleted character's
// quick items are still in the file — reports Active false, ten zeroed records
// and ActiveQuick zero, and its slot data is never searched or read.
type CharacterQuickItems struct {
	SaveSessionID string                            `json:"saveSessionID"`
	SaveRevision  string                            `json:"saveRevision"`
	CharacterID   int                               `json:"characterID"`
	Active        bool                              `json:"active"`
	Items         [quickItemSlotCount]QuickItemSlot `json:"items"`
	ActiveQuick   int32                             `json:"activeQuick"`
}

// GetQuickItems returns the raw quick-item state stored in one physical
// character slot of an existing session. Like the other character readers it
// reads the session's private snapshot through the codec only: it opens no file,
// writes nothing, changes no session and returns no snapshot byte.
//
// saveSessionID is matched exactly. It is never trimmed, normalised or guessed,
// so an empty, unknown or already closed identifier is rejected instead of
// resolving to a session.
//
// characterID is the slot index 0..9. An inactive or residual slot is a normal
// result, not an error, and its slot data is never read.
//
// For an active slot the section is located through the confirmed anchor of that
// one slot and read at a constant distance behind it. A missing anchor and a
// required range reaching past the end of the slot or of the snapshot are hard
// errors. There is no fallback position, no partial result and nothing is
// guessed.
func (engine *Engine) GetQuickItems(saveSessionID string, characterID int) (CharacterQuickItems, error) {
	if saveSessionID == "" {
		return CharacterQuickItems{}, apperror.MissingField("saveSessionID")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterQuickItems{}, apperror.UnknownSaveSession(saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterQuickItems{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterQuickItems{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}

	quickItems := CharacterQuickItems{
		SaveSessionID: saveSessionID,
		SaveRevision:  loaded.session.revisionString(),
		CharacterID:   characterID,
	}
	if flag[0] != userData10ActiveFlagValue {
		// An inactive slot is reported from its flag alone, so the residual quick
		// items of a deleted character are never located or decoded.
		return quickItems, nil
	}

	items, activeQuick, err := readQuickItems(loaded, characterID)
	if err != nil {
		return CharacterQuickItems{}, err
	}

	quickItems.Active = true
	quickItems.Items = items
	quickItems.ActiveQuick = activeQuick
	return quickItems, nil
}

// readQuickItems is the single decoder of the ten EquipItemData quick-item
// pairs and their signed active index. The caller must hold Engine.mutex and
// establish that the slot is active.
func readQuickItems(
	loaded *loadedSave,
	characterID int,
) ([quickItemSlotCount]QuickItemSlot, int32, error) {
	var items [quickItemSlotCount]QuickItemSlot
	base := slotDataBase(loaded.session.platform, characterID)
	slotEnd := base + characterSlotDataSize

	anchor, err := loaded.snapshot.indexIn(base, characterSlotDataSize, quickItemsAnchor)
	if err != nil {
		return items, 0, fmt.Errorf("cannot search the quick items of character %d: %w", characterID, err)
	}
	if anchor < 0 {
		return items, 0, fmt.Errorf("character %d carries no quick-items anchor", characterID)
	}

	sectionAt := anchor + quickItemsSectionOffset
	if sectionAt+quickItemsReadSize > slotEnd {
		return items, 0, fmt.Errorf("quick items of character %d do not fit into its slot", characterID)
	}
	section, err := loaded.snapshot.readAt(sectionAt, quickItemsReadSize)
	if err != nil {
		return items, 0, fmt.Errorf("cannot read quick items of character %d: %w", characterID, err)
	}

	for index := range items {
		record := section[index*quickItemRecordSize:]
		items[index] = QuickItemSlot{
			ItemID:     binary.LittleEndian.Uint32(record),
			EquipIndex: binary.LittleEndian.Uint32(record[4:]),
		}
	}
	activeQuick := int32(binary.LittleEndian.Uint32(section[quickItemsActiveOffset:]))
	return items, activeQuick, nil
}
