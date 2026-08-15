package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Slot-data layout of the confirmed event-flag bitfield, shared by PC and PS4.
// The bitfield has no fixed position inside a slot: everything in front of it
// depends on four lengths the save itself declares — the acquired projectiles,
// the unlocked regions, the menu profile and the tutorial data — so the section
// is located through the confirmed anchor of that one slot and walked forwards
// across the whole chain. This reader owns its own anchor, its own layout
// constants and its own bounds checks; it borrows no position, helper or parsing
// function from another getter. The cookbook mutation reuses this exact locator
// and resolver so reads and writes cannot drift.
const (
	// eventFlagProjectileCountOffset is the distance from the anchor to the
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
	eventFlagProjectileCountOffset = 0xD0 + 0x58 + 0x1C + 0x58 + 0x58 + 0x9011 + 0x74 + 0x8C + 0x18

	// eventFlagProjectileRecordSize is the stride of one acquired-projectile
	// record and eventFlagMaxProjectileRecords is the highest count accepted
	// before the declared length is treated as corrupt instead of followed.
	eventFlagProjectileRecordSize = 8
	eventFlagMaxProjectileRecords = 200000

	// eventFlagBlocksBeforeStorage is the distance from the end of the projectile
	// records to the first byte of the Storage Box: EquipedArmaments (0x9C),
	// EquipPhysicsData (0x0C) and FaceData (0x12F).
	eventFlagBlocksBeforeStorage = 0x9C + 0x0C + 0x12F

	// eventFlagStorageBoxSize and eventFlagGestureSectionSize are the two fixed
	// blocks behind the face data. Both are distances to walk over here, not
	// sections this reader parses.
	eventFlagStorageBoxSize     = 0x6010
	eventFlagGestureSectionSize = 0x100

	// eventFlagRegionRecordSize is the stride of one unlocked-region record and
	// eventFlagMaxRegionRecords is the highest count accepted before the declared
	// length is treated as corrupt.
	eventFlagRegionRecordSize = 4
	eventFlagMaxRegionRecords = 20000

	// eventFlagHorseSize covers RideGameData plus its trailing control byte, and
	// eventFlagBloodStainSize covers the blood stain plus its trailing padding.
	eventFlagHorseSize      = 0x29
	eventFlagBloodStainSize = 0x4C

	// A variable block — the menu profile and the tutorial data — is stored as an
	// eight-byte header (two uint16 the reader does not interpret, then the uint32
	// payload size) followed by that many payload bytes. The declared size is
	// accepted up to eventFlagMaxDynamicSize only; the legacy assumption that the
	// tutorial payload is always 0x400 bytes long is deliberately not used, the
	// size is always read from the header.
	eventFlagDynamicHeaderSize = 8
	eventFlagMaxDynamicSize    = 0x10000

	// eventFlagTrophyEquipSize and eventFlagGaItemGameDataSize are the two fixed
	// blocks between the menu profile and the tutorial data: TrophyEquipData, and
	// GaItemGameData with its int64 count in front of 7000 sixteen-byte entries.
	eventFlagTrophyEquipSize    = 0x34
	eventFlagGaItemGameDataSize = 8 + 7000*16

	// eventFlagScalarsSize is the confirmed scalar block between the end of the
	// tutorial data and the first byte of the bitfield: three gameman bytes,
	// total deaths, character type, the online session flag, the online character
	// type, the last rested grace, the not-alone flag, the in-game timer and one
	// trailing uint32.
	eventFlagScalarsSize = 3 + 4 + 4 + 1 + 4 + 4 + 1 + 4 + 4

	// eventFlagSectionSize is the fixed length of the bitfield and
	// eventFlagTerminatorSize the single byte behind it. Both must lie inside the
	// slot before one flag is read.
	eventFlagSectionSize    = 0x1BF99F
	eventFlagTerminatorSize = 1

	// A flag ID is split into a block of eventFlagsPerBlock flags and an index
	// inside it. A block occupies eventFlagBlockSize bytes of the bitfield.
	eventFlagsPerBlock = 1000
	eventFlagBlockSize = 125
)

// eventFlagAnchor is the confirmed 65-byte marker this reader is measured from:
// one leading 0x00 byte, then four full repetitions of a 16-byte block made of
// 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes.
var eventFlagAnchor = []byte{
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

// CharacterEventFlags is the state of the requested event flags of one physical
// save slot. It carries no name, no meaning and no catalog data: deciding which
// flag belongs to which unlock is the caller's job.
//
// Active reports the slot's UserData10 activity flag. An inactive slot —
// including a residual one, whose deleted character's bitfield is still in the
// file — reports Active false and an empty, non-nil map, and its slot data is
// never searched or read. For an active slot Flags holds one entry per distinct
// requested identifier; a flag that cannot be located or read is an error, never
// a false.
type CharacterEventFlags struct {
	SaveSessionID string          `json:"saveSessionID"`
	CharacterID   int             `json:"characterID"`
	Active        bool            `json:"active"`
	Flags         map[uint32]bool `json:"flags"`
}

// GetEventFlags reports the state of the requested event flags in one physical
// character slot of an existing session. Like the other character readers it
// reads the session's private snapshot through the codec only: it opens no file,
// writes nothing, changes no session and returns no snapshot byte.
//
// saveSessionID is matched exactly. characterID is the slot index 0..9. An
// inactive or residual slot is a normal result, not an error, and its slot data
// is never read.
//
// Every requested identifier is resolved before the slot is touched, so an
// identifier this reader cannot place is rejected instead of being answered.
// For an active slot the bitfield is located dynamically along the confirmed
// chain behind the anchor. A missing anchor, a declared count or size above the
// accepted maximum, an incomplete dynamic header and a bitfield or terminator
// reaching past the end of the slot or of the snapshot are hard errors. There is
// no fallback position, no partial result and nothing is guessed.
func (engine *Engine) GetEventFlags(
	saveSessionID string, characterID int, eventFlagIDs []uint32,
) (CharacterEventFlags, error) {
	if saveSessionID == "" {
		return CharacterEventFlags{}, errors.New("saveSessionID is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterEventFlags{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterEventFlags{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	positions := make(map[uint32]eventFlagPosition, len(eventFlagIDs))
	for _, id := range eventFlagIDs {
		position, err := resolveEventFlag(id)
		if err != nil {
			return CharacterEventFlags{}, err
		}
		positions[id] = position
	}

	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterEventFlags{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}

	result := CharacterEventFlags{
		SaveSessionID: saveSessionID,
		CharacterID:   characterID,
		Flags:         map[uint32]bool{},
	}
	if flag[0] != userData10ActiveFlagValue {
		// An inactive slot is reported from its flag alone, so the residual
		// bitfield of a deleted character is never located or decoded.
		return result, nil
	}

	sectionAt, err := eventFlagSectionStart(loaded, characterID)
	if err != nil {
		return CharacterEventFlags{}, err
	}

	for id, position := range positions {
		raw, err := loaded.snapshot.readAt(sectionAt+position.offset, 1)
		if err != nil {
			return CharacterEventFlags{}, fmt.Errorf(
				"cannot read event flag %d of character %d: %w", id, characterID, err)
		}
		result.Flags[id] = raw[0]&(1<<position.bit) != 0
	}

	result.Active = true
	return result, nil
}

// eventFlagPosition is where one flag lives inside the bitfield: the byte
// counted from the first byte of the section, and the bit inside that byte.
type eventFlagPosition struct {
	offset int64
	bit    uint8
}

// resolveEventFlag places one identifier inside the bitfield. Only blocks with
// confirmed cookbook, whetblade, bell bearing, colosseum, summoning pool, grace,
// boss or map region evidence are supported; every other block is rejected
// instead of being answered from a guessed position, so an unsupported
// identifier can never be reported as false.
func resolveEventFlag(id uint32) (eventFlagPosition, error) {
	var blockPosition int64
	switch block := id / eventFlagsPerBlock; block {
	// Synchronized boss defeat flags occupy block 9, whose BST position is 9 in
	// SaveForge 1.5.8 and 1.6.8 alike. Only block 9 is added: the neighbouring
	// blocks 8 and 10 carry no curated resource, so they stay rejected.
	case 9:
		blockPosition = 9
	case 60:
		blockPosition = 10
	// Map region visibility flags occupy block 62, whose BST position is 12 in
	// SaveForge 1.5.8 and 1.6.8 alike. Only block 62 is added: block 63 carries
	// the transient map fragment pickup triggers and block 82 the system-level
	// map display switches, neither of which is a map region, so both stay
	// rejected.
	case 62:
		blockPosition = 12
	case 65:
		blockPosition = 15
	// Grace visit flags occupy the blocks 71 to 74 and 76, whose BST positions
	// are identical in SaveForge 1.5.8 and 1.6.8. Block 75 has a BST position
	// too, but no grace of the curated table lies in it, so it is deliberately
	// not resolved here.
	case 71:
		blockPosition = 21
	case 72:
		blockPosition = 22
	case 73:
		blockPosition = 23
	case 74:
		blockPosition = 24
	case 76:
		blockPosition = 26
	case 1042378:
		// Whetstone Knife's confirmed system-affinity flag 1042378601
		// occupies byte 0xA0D0C in both SaveForge 1.5.8 and 1.6.8.
		blockPosition = 5269
	case 67:
		blockPosition = 17
	case 68:
		blockPosition = 18
	case 670:
		// Summoning pool activation flags occupy BST position 107, confirmed for
		// the whole block in both SaveForge 1.5.8 and 1.6.8.
		blockPosition = 107
	case 11109:
		blockPosition = 11129
	default:
		return eventFlagPosition{}, fmt.Errorf(
			"event flag %d lies in block %d, which this reader does not support", id, block)
	}

	index := int64(id % eventFlagsPerBlock)
	return eventFlagPosition{
		offset: blockPosition*eventFlagBlockSize + index/8,
		bit:    uint8(7 - index%8),
	}, nil
}

// eventFlagSectionStart walks the confirmed chain from the anchor of one slot to
// the first byte of the bitfield. Every declared length is widened to int64
// before it is multiplied or added, so a corrupt value can never wrap into a
// small, seemingly valid offset.
func eventFlagSectionStart(loaded *loadedSave, characterID int) (int64, error) {
	at, slotEnd, err := eventFlagGaItemGameDataAt(loaded, characterID)
	if err != nil {
		return 0, err
	}
	at += eventFlagGaItemGameDataSize

	tutorialSize, err := eventFlagDeclaredValue(loaded, characterID,
		at+4, slotEnd, "tutorial size", eventFlagMaxDynamicSize)
	if err != nil {
		return 0, err
	}
	// The tutorial block ends behind its own header and payload; the legacy fixed
	// length is never used, so a save whose tutorial payload is not 0x400 bytes
	// long is placed correctly instead of being read at a shifted position.
	tutorialEnd := at + eventFlagDynamicHeaderSize + tutorialSize

	sectionAt := tutorialEnd + eventFlagScalarsSize
	if sectionAt+eventFlagSectionSize+eventFlagTerminatorSize > slotEnd {
		return 0, fmt.Errorf(
			"event flags of character %d do not fit into their slot", characterID)
	}
	// The bitfield and the terminator behind it must also lie inside the snapshot
	// itself, so a truncated file is rejected before one flag is read. The value
	// of the terminator is not interpreted, so no known-good save is rejected over
	// a byte this reader does not own.
	if !loaded.snapshot.covers(sectionAt, eventFlagSectionSize+eventFlagTerminatorSize) {
		return 0, fmt.Errorf(
			"event flags of character %d do not fit into the save file", characterID)
	}
	return sectionAt, nil
}

// eventFlagGaItemGameDataAt walks the same confirmed chain up to the first byte
// of GaItemGameData and reports it together with the end of the slot.
//
// It is the single owner of that walk. The bitfield reader continues from here
// over the fixed GaItemGameData block, and the GaItemData mutation addresses the
// block itself from the same value, while the cookbook mutation reuses the
// bitfield locator, so readers and writers cannot disagree about where either
// section starts. Whether the block fits into the slot is the caller's check:
// the bitfield locator proves that through the sections behind it, and the
// GaItemData mutation proves it for the block itself.
func eventFlagGaItemGameDataAt(loaded *loadedSave, characterID int) (int64, int64, error) {
	base, slotEnd := eventFlagSlotBounds(loaded.session.platform, characterID)

	anchor, err := loaded.snapshot.indexIn(base, slotEnd-base, eventFlagAnchor)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot search the event flags of character %d: %w", characterID, err)
	}
	if anchor < 0 {
		return 0, 0, fmt.Errorf("character %d carries no event flag anchor", characterID)
	}

	projectiles, err := eventFlagDeclaredValue(loaded, characterID,
		anchor+eventFlagProjectileCountOffset, slotEnd, "projectile count", eventFlagMaxProjectileRecords)
	if err != nil {
		return 0, 0, err
	}
	at := anchor + eventFlagProjectileCountOffset + 4 +
		projectiles*eventFlagProjectileRecordSize +
		eventFlagBlocksBeforeStorage + eventFlagStorageBoxSize + eventFlagGestureSectionSize

	regions, err := eventFlagDeclaredValue(loaded, characterID,
		at, slotEnd, "region count", eventFlagMaxRegionRecords)
	if err != nil {
		return 0, 0, err
	}
	at += 4 + regions*eventFlagRegionRecordSize + eventFlagHorseSize + eventFlagBloodStainSize

	menuSize, err := eventFlagDeclaredValue(loaded, characterID,
		at+4, slotEnd, "menu profile size", eventFlagMaxDynamicSize)
	if err != nil {
		return 0, 0, err
	}
	return at + eventFlagDynamicHeaderSize + menuSize + eventFlagTrophyEquipSize, slotEnd, nil
}

// eventFlagDeclaredValue reads one little-endian uint32 the save declares — a
// record count or a payload size — and rejects it when it does not lie inside
// the slot or exceeds the accepted maximum.
func eventFlagDeclaredValue(
	loaded *loadedSave, characterID int, at, slotEnd int64, field string, maximum int64,
) (int64, error) {
	if at < 0 || at+4 > slotEnd {
		return 0, fmt.Errorf("%s of character %d lies outside its slot", field, characterID)
	}
	raw, err := loaded.snapshot.readAt(at, 4)
	if err != nil {
		return 0, fmt.Errorf("cannot read %s of character %d: %w", field, characterID, err)
	}
	value := int64(binary.LittleEndian.Uint32(raw))
	if value > maximum {
		return 0, fmt.Errorf("character %d declares a %s of %d, want at most %d",
			characterID, field, value, maximum)
	}
	return value, nil
}

// eventFlagSlotBounds selects the platform entry point of this reader. PC and
// PS4 differ in the container only, so the platform files supply the bounds of
// the slot data and everything inside it is decoded identically.
func eventFlagSlotBounds(platform Platform, characterID int) (int64, int64) {
	if platform == PlatformPS4 {
		return ps4EventFlagSlotBounds(characterID)
	}
	return pcEventFlagSlotBounds(characterID)
}
