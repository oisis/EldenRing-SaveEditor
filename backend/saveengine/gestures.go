package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Slot-data layout of the confirmed GestureGameData section, shared by its
// reader and writer on PC and PS4. The section has no fixed position inside a
// slot: it sits directly behind
// the Storage Box, which sits behind the face-data block, which sits behind
// EquipPhysicsData and the equipped-armaments block, which themselves sit behind
// the variable-length acquired-projectiles section. Everything in front of
// GestureGameData therefore depends on how many projectiles the character has
// acquired, so the section is located through the confirmed anchor of that one
// slot and walked forwards from it across the one dynamic length the save itself
// declares. The locator below is the single source of this position for both
// gesture operations; neither borrows an offset from another save section.
const (
	// gestureProjectileCountOffset is the distance from the anchor to the uint32
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
	gestureProjectileCountOffset = 0x931D

	// gestureProjectileRecordSize is the stride of one acquired-projectile
	// record, and gestureMaxProjectileRecords is the highest count accepted
	// before the declared length is treated as corrupt instead of followed. The
	// limit is far above the counts native saves carry and far below what would
	// let a declared length wrap or reach past the container.
	gestureProjectileRecordSize = 8
	gestureMaxProjectileRecords = 200000

	// gestureBlocksBeforeStorage is the distance from the end of the projectile
	// records to the first byte of the Storage Box. It is the sum of the three
	// confirmed fixed blocks in between:
	//
	//	0x009C EquipedArmaments
	//	0x000C EquipPhysicsData
	//	0x012F FaceData
	//
	// The Storage Box starts immediately behind the face data.
	gestureBlocksBeforeStorage = 0x9C + 0x0C + 0x12F

	// gestureStorageBoxSize is the confirmed size of the Storage Box that stands
	// between the face data and GestureGameData. It is a distance to walk over
	// here, not a section this getter parses: nothing inside the Storage Box is
	// read, decoded or validated by the gesture operations.
	gestureStorageBoxSize = 0x6010

	// GestureSlotCount is the number of raw records GestureGameData holds. The
	// block is a fixed 64 × uint32 little-endian array and is always read whole.
	GestureSlotCount = 64

	// gestureRecordSize is the stride of one raw record and gestureSectionSize is
	// the size of the whole block, the 0x100 bytes the next section is measured
	// from.
	gestureRecordSize  = 4
	gestureSectionSize = GestureSlotCount * gestureRecordSize
)

// gestureAnchor is the confirmed 65-byte marker this getter is measured from:
// one leading 0x00 byte, then four full repetitions of a 16-byte block made of
// 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes. It is stated here for this
// getter alone, the way every other slot reader states the marker it depends on.
var gestureAnchor = []byte{
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

// CharacterGestures is the raw GestureGameData block of one physical save slot.
// This is the save side of the gesture surface: it carries no name, no category,
// no kind, no key and no unlock decision, and it reads no GameCatalog at all.
//
// Slots keeps the physical native order and the stored values exactly as
// written. Nothing is sorted, deduplicated, filtered, masked or normalised, so a
// native empty sentinel, a zero and a value this stage cannot explain all stay
// visible where the game put them. Deciding what a stored value means belongs to
// the caller that owns the gesture definitions.
//
// Active reports the slot's UserData10 activity flag. An inactive slot —
// including a residual one, whose deleted character's gesture block is still in
// the file — reports Active false and an empty, non-nil list, and its slot data
// is never searched or read. For an active slot Slots always holds exactly
// GestureSlotCount values; a block that is not completely present is an error,
// never a short result.
type CharacterGestures struct {
	SaveSessionID string   `json:"saveSessionID"`
	SaveRevision  string   `json:"saveRevision"`
	CharacterID   int      `json:"characterID"`
	Active        bool     `json:"active"`
	Slots         []uint32 `json:"slots"`
}

// GetGestures returns the raw GestureGameData records stored in one physical
// character slot of an existing session. Like the other character readers it
// reads the session's private snapshot through the codec only: it opens no file,
// writes nothing, changes no session and returns no snapshot byte. It calls no
// other getter and no endpoint.
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
// distance behind it, the Storage Box starts behind the records that count
// declares plus the three fixed blocks in between, and GestureGameData starts
// behind the Storage Box. A missing anchor, a count above the accepted maximum
// and a block reaching past the end of the slot or of the snapshot are hard
// errors. There is no fallback position, no partial result and nothing is
// guessed.
func (engine *Engine) GetGestures(saveSessionID string, characterID int) (CharacterGestures, error) {
	if saveSessionID == "" {
		return CharacterGestures{}, errors.New("saveSessionID is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterGestures{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterGestures{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterGestures{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}

	gestures := CharacterGestures{
		SaveSessionID: saveSessionID,
		SaveRevision:  loaded.session.revisionString(),
		CharacterID:   characterID,
		Slots:         []uint32{},
	}
	if flag[0] != userData10ActiveFlagValue {
		// An inactive slot is reported from its flag alone, so the residual
		// gesture block of a deleted character is never located or decoded.
		return gestures, nil
	}

	sectionAt, err := gestureSectionStart(loaded, characterID)
	if err != nil {
		return CharacterGestures{}, err
	}
	section, err := loaded.snapshot.readAt(sectionAt, gestureSectionSize)
	if err != nil {
		return CharacterGestures{}, fmt.Errorf("cannot read gestures of character %d: %w", characterID, err)
	}

	slots := make([]uint32, GestureSlotCount)
	for index := range slots {
		slots[index] = binary.LittleEndian.Uint32(section[index*gestureRecordSize:])
	}

	gestures.Active = true
	gestures.Slots = slots
	return gestures, nil
}

// gestureSectionStart resolves the first byte of GestureGameData for one
// character. Callers validate the character index and activity before using it.
// Every offset and bound used by both the getter and setter lives here, so the
// two operations cannot drift to different interpretations of the slot layout.
func gestureSectionStart(loaded *loadedSave, characterID int) (int64, error) {
	base, slotEnd := gestureSlotBounds(loaded.session.platform, characterID)

	anchor, err := loaded.snapshot.indexIn(base, slotEnd-base, gestureAnchor)
	if err != nil {
		return 0, fmt.Errorf(
			"cannot search the gestures of character %d: %w", characterID, err)
	}
	if anchor < 0 {
		return 0, fmt.Errorf("character %d carries no gesture anchor", characterID)
	}

	countAt := anchor + gestureProjectileCountOffset
	if countAt+4 > slotEnd {
		return 0, fmt.Errorf(
			"projectile count of character %d lies outside its slot", characterID)
	}
	rawCount, err := loaded.snapshot.readAt(countAt, 4)
	if err != nil {
		return 0, fmt.Errorf(
			"cannot read projectile count of character %d: %w", characterID, err)
	}
	// The count is widened to int64 before it is multiplied, so a declared
	// length can never wrap into a small, seemingly valid offset.
	count := int64(binary.LittleEndian.Uint32(rawCount))
	if count > gestureMaxProjectileRecords {
		return 0, fmt.Errorf(
			"character %d declares %d projectile records, want at most %d",
			characterID, count, gestureMaxProjectileRecords)
	}

	sectionAt := countAt + 4 + count*gestureProjectileRecordSize +
		gestureBlocksBeforeStorage + gestureStorageBoxSize
	if sectionAt+gestureSectionSize > slotEnd {
		return 0, fmt.Errorf(
			"gestures of character %d do not fit into their slot", characterID)
	}
	return sectionAt, nil
}

// gestureSlotBounds selects the platform entry point of the gesture operations.
// PC and PS4 differ in the container only, so the platform files supply the
// bounds of the slot data and everything inside it is decoded identically.
func gestureSlotBounds(platform Platform, characterID int) (int64, int64) {
	if platform == PlatformPS4 {
		return ps4GestureSlotBounds(characterID)
	}
	return pcGestureSlotBounds(characterID)
}
