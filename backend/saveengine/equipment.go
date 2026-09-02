package saveengine

import (
	"encoding/binary"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// Slot-data layout of the confirmed equipped-armaments block, shared by PC and
// PS4. The block has no fixed position inside a slot: it sits behind the
// variable-length acquired-projectiles section, so everything in front of it
// depends on how many projectiles the character has acquired. It is therefore
// located through the confirmed anchor of the slot and read forwards from it,
// across one dynamic length that the save itself declares.
const (
	// equipmentSlotCount is the number of raw ChrAsmEquipment fields. The block
	// is read as exactly these 22 little-endian uint32 values; the bytes behind
	// them are never touched.
	equipmentSlotCount = 22
	equipmentBlockSize = equipmentSlotCount * 4

	// equipmentProjectileCountOffset is the distance from the anchor to the
	// uint32 that declares how many acquired-projectile records follow it. It is
	// the sum of the confirmed fixed structures between the two positions:
	//
	//	0x00D0 SpEffect
	//	0x0058 EquipedItemIndex
	//	0x001C ActiveEquipedItems
	//	0x0058 EquipedItemsID
	//	0x0058 ActiveEquipedItemsGa
	//	0x9011 InventoryHeld
	//	0x0074 EquippedSpells
	//	0x008C EquipItemData
	//	0x0018 EquippedGestures
	//
	// Every one of them has a fixed size, so this distance is constant; only the
	// projectile section behind it varies.
	equipmentProjectileCountOffset = 0x931D

	// equipmentProjectileRecordSize is the stride of one acquired-projectile
	// record, and equipmentMaxProjectileRecords is the highest count accepted
	// before the declared length is treated as corrupt instead of followed. The
	// limit is far above the counts native saves carry and far below what would
	// let a declared length wrap or reach past the container.
	equipmentProjectileRecordSize = 8
	equipmentMaxProjectileRecords = 200000
)

// equipmentAnchor is the confirmed 65-byte marker the equipment chain is
// measured from: one leading 0x00 byte, then four full repetitions of a 16-byte
// block made of 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes. It is stated
// here for this getter alone, the way every other slot reader states the marker
// it depends on.
var equipmentAnchor = []byte{
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

// CharacterEquipment is the raw equipped state of one physical save slot: the 22
// ChrAsmEquipment fields exactly as stored. No value is normalised, masked,
// turned into a handle, stripped of its type bits, validated or resolved to a
// name, and no GameCatalog is read.
//
// Slots holds the fields in their stored order, which is fixed:
//
//	 0 leftHandArmament1    11 unknown0x2C
//	 1 rightHandArmament1   12 head
//	 2 leftHandArmament2    13 chest
//	 3 rightHandArmament2   14 arms
//	 4 leftHandArmament3    15 legs
//	 5 rightHandArmament3   16 unknown0x40
//	 6 arrows1              17 talisman1
//	 7 bolts1               18 talisman2
//	 8 arrows2              19 talisman3
//	 9 bolts2               20 talisman4
//	10 unknown0x28          21 unknown0x54
//
// The unknown fields are reported as stored, like every other field.
//
// Every field except SaveSessionID and CharacterID describes an active slot
// only. An inactive slot — including a residual one, whose deleted character's
// equipment is still in the file — reports Active false and an all-zero Slots,
// and its slot data is never searched or read.
type CharacterEquipment struct {
	SaveSessionID string     `json:"saveSessionID"`
	SaveRevision  string     `json:"saveRevision"`
	CharacterID   int        `json:"characterID"`
	Active        bool       `json:"active"`
	Slots         [22]uint32 `json:"slots"`
}

// GetEquipment returns the raw equipped state stored in one physical character
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
// For an active slot the block is located dynamically: the confirmed anchor is
// searched inside that one slot, the projectile count is read at a fixed
// distance behind it, and the equipment block starts behind the records that
// count declares. A missing anchor, a count above the accepted maximum and a
// position reaching past the end of the slot or of the snapshot are hard errors.
// There is no fallback position, no partial result and nothing is guessed.
func (engine *Engine) GetEquipment(saveSessionID string, characterID int) (CharacterEquipment, error) {
	if saveSessionID == "" {
		return CharacterEquipment{}, apperror.MissingField("saveSessionID")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterEquipment{}, apperror.UnknownSaveSession(saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterEquipment{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterEquipment{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}

	equipment := CharacterEquipment{
		SaveSessionID: saveSessionID,
		SaveRevision:  loaded.session.revisionString(),
		CharacterID:   characterID,
	}
	if flag[0] != userData10ActiveFlagValue {
		// An inactive slot is reported from its flag alone, so the residual
		// equipment of a deleted character is never located or decoded.
		return equipment, nil
	}

	slots, _, _, err := readEquipmentSlots(loaded, characterID)
	if err != nil {
		return CharacterEquipment{}, err
	}

	equipment.Active = true
	equipment.Slots = slots
	return equipment, nil
}

// readEquipmentSlots is the single read-only decoder of the dynamic
// ChrAsmEquipment block. GetEquipment and the aggregate character-loadout
// reader share it, so presentation can never drift to a second offset chain.
// The caller must hold Engine.mutex and establish that the slot is active.
func readEquipmentSlots(
	loaded *loadedSave,
	characterID int,
) ([equipmentSlotCount]uint32, int64, int64, error) {
	var slots [equipmentSlotCount]uint32
	blockAt, slotEnd, err := equipmentBlockAt(loaded, characterID)
	if err != nil {
		return slots, 0, 0, err
	}
	block, err := loaded.snapshot.readAt(blockAt, equipmentBlockSize)
	if err != nil {
		return slots, 0, 0, fmt.Errorf("cannot read equipment of character %d: %w", characterID, err)
	}
	for index := range slots {
		slots[index] = binary.LittleEndian.Uint32(block[index*4:])
	}
	return slots, blockAt, slotEnd, nil
}

// equipmentBlockAt locates the dynamic equipped-armaments block for one active
// slot. Readers that need the confirmed tails behind the 22 fields reuse the
// same projectile-count validation and bounds checks.
func equipmentBlockAt(loaded *loadedSave, characterID int) (int64, int64, error) {
	base := slotDataBase(loaded.session.platform, characterID)
	slotEnd := base + characterSlotDataSize

	anchor, err := loaded.snapshot.indexIn(base, characterSlotDataSize, equipmentAnchor)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot search the equipment of character %d: %w", characterID, err)
	}
	if anchor < 0 {
		return 0, 0, fmt.Errorf("character %d carries no equipment anchor", characterID)
	}

	countAt := anchor + equipmentProjectileCountOffset
	if countAt+4 > slotEnd {
		return 0, 0, fmt.Errorf("projectile count of character %d lies outside its slot", characterID)
	}
	rawCount, err := loaded.snapshot.readAt(countAt, 4)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot read projectile count of character %d: %w", characterID, err)
	}
	count := int64(binary.LittleEndian.Uint32(rawCount))
	if count > equipmentMaxProjectileRecords {
		return 0, 0, fmt.Errorf(
			"character %d declares %d projectile records, want at most %d",
			characterID, count, equipmentMaxProjectileRecords)
	}

	blockAt := countAt + 4 + count*equipmentProjectileRecordSize
	if blockAt+equipmentBlockSize > slotEnd {
		return 0, 0, fmt.Errorf("equipment block of character %d does not fit into its slot", characterID)
	}
	return blockAt, slotEnd, nil
}
