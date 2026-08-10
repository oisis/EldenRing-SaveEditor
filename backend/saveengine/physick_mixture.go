package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Slot-data layout of the confirmed EquipPhysicsData block, shared by PC and
// PS4. The block has no fixed position inside a slot: it sits behind the
// equipped-armaments block, which itself sits behind the variable-length
// acquired-projectiles section, so everything in front of it depends on how many
// projectiles the character has acquired. It is therefore located through the
// confirmed anchor of the slot and read forwards from it, across one dynamic
// length that the save itself declares.
const (
	// physickTearCount is the number of raw Crystal Tear identifiers the current
	// mixture holds. They are the first two little-endian uint32 of
	// EquipPhysicsData; the bytes behind them are never touched by this getter.
	physickTearCount = 2
	physickReadSize  = physickTearCount * 4

	// physickProjectileCountOffset is the distance from the anchor to the uint32
	// that declares how many acquired-projectile records follow it. It is the sum
	// of the confirmed fixed structures between the two positions:
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
	physickProjectileCountOffset = 0x931D

	// physickProjectileRecordSize is the stride of one acquired-projectile
	// record, and physickMaxProjectileRecords is the highest count accepted
	// before the declared length is treated as corrupt instead of followed. The
	// limit is far above the counts native saves carry and far below what would
	// let a declared length wrap or reach past the container.
	physickProjectileRecordSize = 8
	physickMaxProjectileRecords = 200000

	// physickArmamentsBlockSize is the size of the equipped-armaments block that
	// lies between the projectile records and EquipPhysicsData. The Physick block
	// starts immediately behind it.
	physickArmamentsBlockSize = 0x9C
)

// physickMixtureAnchor is the confirmed 65-byte marker the Physick chain is
// measured from: one leading 0x00 byte, then four full repetitions of a 16-byte
// block made of 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes. It is stated
// here for this getter alone, the way every other slot reader states the marker
// it depends on.
var physickMixtureAnchor = []byte{
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

// CharacterPhysickMixture is the raw Flask of Wondrous Physick mixture of one
// physical save slot: the two Crystal Tear identifiers at the start of
// EquipPhysicsData, exactly as stored. Neither value is normalised, masked,
// validated or resolved to an item, and no GameCatalog is read.
//
// Tears holds the two identifiers in their stored order, which is the order of
// the two mixture positions in the game.
//
// Every field except SaveSessionID and CharacterID describes an active slot
// only. An inactive slot — including a residual one, whose deleted character's
// mixture is still in the file — reports Active false and two zeros, and its
// slot data is never searched or read.
type CharacterPhysickMixture struct {
	SaveSessionID string    `json:"saveSessionID"`
	CharacterID   int       `json:"characterID"`
	Active        bool      `json:"active"`
	Tears         [2]uint32 `json:"tears"`
}

// GetPhysickMixture returns the raw Physick mixture stored in one physical
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
// For an active slot the block is located dynamically: the confirmed anchor is
// searched inside that one slot, the projectile count is read at a fixed
// distance behind it, and EquipPhysicsData starts behind the records that count
// declares plus the equipped-armaments block. A missing anchor, a count above
// the accepted maximum and a position reaching past the end of the slot or of
// the snapshot are hard errors. There is no fallback position, no partial result
// and nothing is guessed.
func (engine *Engine) GetPhysickMixture(saveSessionID string, characterID int) (CharacterPhysickMixture, error) {
	if saveSessionID == "" {
		return CharacterPhysickMixture{}, errors.New("saveSessionID is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterPhysickMixture{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterPhysickMixture{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterPhysickMixture{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}

	mixture := CharacterPhysickMixture{SaveSessionID: saveSessionID, CharacterID: characterID}
	if flag[0] != userData10ActiveFlagValue {
		// An inactive slot is reported from its flag alone, so the residual
		// mixture of a deleted character is never located or decoded.
		return mixture, nil
	}

	base := slotDataBase(loaded.session.platform, characterID)
	slotEnd := base + characterSlotDataSize

	anchor, err := loaded.snapshot.indexIn(base, characterSlotDataSize, physickMixtureAnchor)
	if err != nil {
		return CharacterPhysickMixture{}, fmt.Errorf(
			"cannot search the physick mixture of character %d: %w", characterID, err)
	}
	if anchor < 0 {
		return CharacterPhysickMixture{}, fmt.Errorf("character %d carries no physick anchor", characterID)
	}

	countAt := anchor + physickProjectileCountOffset
	if countAt+4 > slotEnd {
		return CharacterPhysickMixture{}, fmt.Errorf(
			"projectile count of character %d lies outside its slot", characterID)
	}
	rawCount, err := loaded.snapshot.readAt(countAt, 4)
	if err != nil {
		return CharacterPhysickMixture{}, fmt.Errorf(
			"cannot read projectile count of character %d: %w", characterID, err)
	}
	// The count is widened to int64 before it is multiplied, so a declared
	// length can never wrap into a small, seemingly valid offset.
	count := int64(binary.LittleEndian.Uint32(rawCount))
	if count > physickMaxProjectileRecords {
		return CharacterPhysickMixture{}, fmt.Errorf(
			"character %d declares %d projectile records, want at most %d",
			characterID, count, physickMaxProjectileRecords)
	}

	blockAt := countAt + 4 + count*physickProjectileRecordSize + physickArmamentsBlockSize
	if blockAt+physickReadSize > slotEnd {
		return CharacterPhysickMixture{}, fmt.Errorf(
			"physick mixture of character %d does not fit into its slot", characterID)
	}
	block, err := loaded.snapshot.readAt(blockAt, physickReadSize)
	if err != nil {
		return CharacterPhysickMixture{}, fmt.Errorf(
			"cannot read physick mixture of character %d: %w", characterID, err)
	}

	mixture.Active = true
	for index := range mixture.Tears {
		mixture.Tears[index] = binary.LittleEndian.Uint32(block[index*4:])
	}
	return mixture, nil
}
