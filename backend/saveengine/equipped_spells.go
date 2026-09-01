package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Slot-data layout of the confirmed EquippedSpells section and of the two
// structures the active spell capacity is derived from, shared by PC and PS4.
// None of them has a fixed position inside a slot, so all of them are located
// through the confirmed anchor of the slot and read relative to it.
const (
	// equippedSpellSlotCount is the number of physical EquippedSpells records.
	// The section stores exactly these fourteen pairs of raw MagicParam ID and
	// follower field, followed by the active-index uint32.
	equippedSpellSlotCount  = 14
	equippedSpellRecordSize = 8
	equippedSpellsReadSize  = equippedSpellSlotCount * equippedSpellRecordSize
	equippedSpellsActiveAt  = equippedSpellsReadSize

	// equippedSpellsSectionOffset is the distance from the anchor to the start of
	// EquippedSpells. It is the sum of the confirmed fixed structures between the
	// two positions:
	//
	//	0x00D0 SpEffect
	//	0x0058 EquipedItemIndex
	//	0x001C ActiveEquipedItems
	//	0x0058 EquipedItemsID
	//	0x0058 ActiveEquipedItemsGa
	//	0x9011 InventoryHeld
	//
	// Every one of them has a fixed size, so this distance is constant:
	// EquippedSpells is the section the variable-length acquired-projectiles
	// records lie behind, not in front of.
	equippedSpellsSectionOffset = 0x9205

	// The two native record pairs. An empty record is the sentinel ID with a zero
	// follower and an occupied record is a stored ID with an all-ones follower;
	// the game writes no third combination, so any other pair is corrupt state
	// rather than something to reinterpret.
	equippedSpellEmptyID          uint32 = 0xFFFFFFFF
	equippedSpellEmptyFollower    uint32 = 0x00000000
	equippedSpellOccupiedFollower uint32 = 0xFFFFFFFF
	equippedSpellRawIDLimit       uint32 = 0x10000000

	// Active spell capacity. A character starts with two memory slots, every
	// Memory Stone adds one, the game caps the stones at eight, and Moon of
	// Nokstella adds two more while it is equipped in an unlocked talisman field.
	// Twelve is the highest number the game grants, which is below the fourteen
	// physical records the save always keeps.
	spellBaseMemorySlots       = 2
	spellMaxMemoryStones       = 8
	spellMaxMemorySlots        = 12
	moonOfNokstellaMemorySlots = 2
	moonOfNokstellaItemID      = 0x20000474

	// Layout of the InventoryHeld section the Memory Stone stack lives in,
	// measured from the anchor. The common records come first, a four-byte key
	// count separates them from the key records, and every record is a triple of
	// GaItem handle, quantity and acquisition index.
	inventoryCommonOffset   = 505
	inventoryRecordSize     = 12
	inventoryCommonRecords  = 0xA80
	inventoryKeyCountHeader = 4
	inventoryKeyRecords     = 0x180
	inventorySectionSize    = inventoryCommonRecords*inventoryRecordSize +
		inventoryKeyCountHeader + inventoryKeyRecords*inventoryRecordSize

	// memoryStoneHandle is the GaItem handle of the Memory Stone stack, and
	// inventoryQuantityMask drops the high bit the game sets on a stored
	// quantity.
	memoryStoneHandle     uint32 = 0xB000272E
	inventoryQuantityMask uint32 = 0x7FFFFFFF

	// talismanSlotsOffset is where the additional-talisman-slots byte lies,
	// measured backwards from the anchor, and talismanSlotsMax is the highest
	// value the game stores there. The total number of unlocked talisman fields
	// is one more than that byte.
	talismanSlotsOffset = -241
	talismanSlotsMax    = 3

	// Layout of the equipped-armaments block the talisman fields live in. It is
	// the same chain GetEquipment follows: the projectile count sits at a fixed
	// distance behind the anchor and the block starts behind the records that
	// count declares.
	equippedSpellsProjectileCountOffset = 0x931D
	equippedSpellsProjectileRecordSize  = 8
	equippedSpellsMaxProjectileRecords  = 200000
	equippedSpellsEquipmentFieldCount   = 22
	equippedSpellsEquipmentBlockSize    = equippedSpellsEquipmentFieldCount * 4
	equippedSpellsFirstTalismanField    = 17
	equippedSpellsTalismanFieldCount    = 5
)

// equippedSpellsAnchor is the confirmed 65-byte marker the whole chain of this
// getter is measured from: one leading 0x00 byte, then four full repetitions of
// a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes. It
// is stated here for this getter alone, the way every other slot reader states
// the marker it depends on.
var equippedSpellsAnchor = []byte{
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

// CharacterEquippedSpells is the raw equipped-spell state of one physical save
// slot: the fourteen stored MagicParam identifiers and the number of memory
// slots the character may fill. No identifier is normalised, masked, converted
// into a full item ID, resolved to a name or checked against a catalog; this
// type reads no GameCatalog at all.
//
// Spells holds the identifiers in their stored order, which is the order of the
// fourteen physical records. The empty sentinel is preserved like every other
// value, so an unused record stays visible as the sentinel instead of becoming a
// zero or disappearing.
//
// AvailableMemorySlots is the active capacity of the character, which is smaller
// than the fourteen physical records: the base capacity, plus the Memory Stones
// the character holds, capped by the game maximum, plus the Moon of Nokstella
// bonus while that talisman sits in an unlocked talisman field.
//
// Every field except SaveSessionID and CharacterID describes an active slot
// only. An inactive slot — including a residual one, whose deleted character's
// spells are still in the file — reports Active false, fourteen zeros and a zero
// capacity, and its slot data is never searched or read.
type CharacterEquippedSpells struct {
	SaveSessionID        string                         `json:"saveSessionID"`
	SaveRevision         string                         `json:"saveRevision"`
	CharacterID          int                            `json:"characterID"`
	Active               bool                           `json:"active"`
	Spells               [equippedSpellSlotCount]uint32 `json:"spells"`
	AvailableMemorySlots int                            `json:"availableMemorySlots"`
}

// CharacterEquippedSpellsMutation reports one committed equipped-spells update.
type CharacterEquippedSpellsMutation struct {
	SaveSessionID        string   `json:"saveSessionID"`
	SaveRevision         string   `json:"saveRevision"`
	CharacterID          int      `json:"characterID"`
	RawMagicParamIDs     []uint32 `json:"rawMagicParamIDs"`
	UsedMemorySlots      int      `json:"usedMemorySlots"`
	AvailableMemorySlots int      `json:"availableMemorySlots"`
}

// GetEquippedSpells returns the raw equipped-spell state stored in one physical
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
// For an active slot everything is located through the confirmed anchor of that
// one slot: the fourteen records at a constant distance behind it, the Memory
// Stone stack inside the InventoryHeld section behind it, the unlocked talisman
// count in front of it, and the talisman fields behind the acquired-projectile
// records the save itself declares. A missing anchor, a record pair the game
// never writes, a declared projectile count above the accepted maximum and any
// required range reaching past the end of the slot or of the snapshot are hard
// errors. There is no fallback position, no partial result and nothing is
// guessed.
func (engine *Engine) GetEquippedSpells(saveSessionID string, characterID int) (CharacterEquippedSpells, error) {
	if saveSessionID == "" {
		return CharacterEquippedSpells{}, errors.New("saveSessionID is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterEquippedSpells{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterEquippedSpells{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterEquippedSpells{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}

	spells := CharacterEquippedSpells{
		SaveSessionID: saveSessionID,
		SaveRevision:  loaded.session.revisionString(),
		CharacterID:   characterID,
	}
	if flag[0] != userData10ActiveFlagValue {
		// An inactive slot is reported from its flag alone, so the residual
		// spells of a deleted character are never located or decoded.
		return spells, nil
	}

	state, err := readEquippedSpellsState(loaded, characterID)
	if err != nil {
		return CharacterEquippedSpells{}, err
	}

	spells.Active = true
	spells.Spells = state.records
	spells.AvailableMemorySlots = state.availableMemorySlots
	return spells, nil
}

type equippedSpellsState struct {
	records               [equippedSpellSlotCount]uint32
	activeSpellIndex      int
	availableMemorySlots  int
	unlockedTalismanSlots int
}

// readEquippedSpellsState is the single read-only decoder of the physical spell
// records, their active index and both capacity inputs needed by a character
// loadout. GetEquippedSpells deliberately projects only its established fields.
// The caller must hold Engine.mutex and establish that the slot is active.
func readEquippedSpellsState(loaded *loadedSave, characterID int) (equippedSpellsState, error) {
	base := slotDataBase(loaded.session.platform, characterID)
	slotEnd := base + characterSlotDataSize

	anchor, err := loaded.snapshot.indexIn(base, characterSlotDataSize, equippedSpellsAnchor)
	if err != nil {
		return equippedSpellsState{}, fmt.Errorf(
			"cannot search the equipped spells of character %d: %w", characterID, err)
	}
	if anchor < 0 {
		return equippedSpellsState{}, fmt.Errorf("character %d carries no equipped-spells anchor", characterID)
	}

	records, err := readEquippedSpellRecords(loaded.snapshot, anchor, slotEnd, characterID)
	if err != nil {
		return equippedSpellsState{}, err
	}
	available, err := readAvailableMemorySlots(loaded.snapshot, anchor, base, slotEnd, characterID)
	if err != nil {
		return equippedSpellsState{}, err
	}
	unlocked, err := readUnlockedTalismanFields(loaded.snapshot, anchor, base, characterID)
	if err != nil {
		return equippedSpellsState{}, err
	}

	activeAt := anchor + equippedSpellsSectionOffset + equippedSpellsActiveAt
	if activeAt+4 > slotEnd {
		return equippedSpellsState{}, fmt.Errorf(
			"active spell index of character %d lies outside its slot", characterID)
	}
	activeRaw, err := loaded.snapshot.uint32At(activeAt)
	if err != nil {
		return equippedSpellsState{}, fmt.Errorf(
			"cannot read active spell index of character %d: %w", characterID, err)
	}
	active := -1
	if activeRaw != equippedSpellEmptyID {
		active = int(activeRaw)
	}

	return equippedSpellsState{
		records:               records,
		activeSpellIndex:      active,
		availableMemorySlots:  available,
		unlockedTalismanSlots: unlocked,
	}, nil
}

// SetEquippedSpells atomically replaces the first 12 spell memory positions of
// one active character slot and updates the active spell index if needed.
func (engine *Engine) SetEquippedSpells(
	saveSessionID string,
	characterID int,
	rawSpellIDs []uint32,
	usedMemorySlots int,
	expectedRevision string,
) (CharacterEquippedSpellsMutation, error) {
	if !isCanonicalRevision(expectedRevision) {
		return CharacterEquippedSpellsMutation{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if len(rawSpellIDs) > spellMaxMemorySlots {
		return CharacterEquippedSpellsMutation{}, fmt.Errorf(
			"cannot equip more than %d spells; got %d", spellMaxMemorySlots, len(rawSpellIDs))
	}

	seen := make(map[uint32]struct{}, len(rawSpellIDs))
	for index, rawID := range rawSpellIDs {
		if rawID == 0 || rawID >= equippedSpellRawIDLimit {
			return CharacterEquippedSpellsMutation{}, fmt.Errorf(
				"spell slot %d: 0x%08X is not a raw MagicParam ID", index, rawID)
		}
		if _, duplicate := seen[rawID]; duplicate {
			return CharacterEquippedSpellsMutation{}, fmt.Errorf(
				"spell slot %d: raw MagicParam ID 0x%08X is duplicated", index, rawID)
		}
		seen[rawID] = struct{}{}
	}

	var availableSlots int
	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetEquippedSpells, characterID, func(loaded *loadedSave) error {
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

		flag, err := loaded.snapshot.readAt(
			userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
		}
		if flag[0] != userData10ActiveFlagValue {
			return fmt.Errorf("character %d is not active", characterID)
		}

		base := slotDataBase(loaded.session.platform, characterID)
		slotEnd := base + characterSlotDataSize

		anchor, err := loaded.snapshot.indexIn(base, characterSlotDataSize, equippedSpellsAnchor)
		if err != nil {
			return fmt.Errorf(
				"cannot search the equipped spells of character %d: %w", characterID, err)
		}
		if anchor < 0 {
			return fmt.Errorf("character %d carries no equipped-spells anchor", characterID)
		}

		sectionAt := anchor + equippedSpellsSectionOffset
		if sectionAt+116 > slotEnd {
			return fmt.Errorf("equipped spells of character %d do not fit into its slot", characterID)
		}

		existingRecords, err := readEquippedSpellRecords(loaded.snapshot, anchor, slotEnd, characterID)
		if err != nil {
			return err
		}
		if existingRecords[12] != equippedSpellEmptyID || existingRecords[13] != equippedSpellEmptyID {
			return fmt.Errorf(
				"physical spell position 13 or 14 of character %d is not empty; mutation aborted", characterID)
		}

		section, err := loaded.snapshot.readAt(sectionAt, 116)
		if err != nil {
			return fmt.Errorf("cannot read equipped spells section of character %d: %w", characterID, err)
		}

		available, err := readAvailableMemorySlots(loaded.snapshot, anchor, base, slotEnd, characterID)
		if err != nil {
			return err
		}
		if usedMemorySlots > available {
			return fmt.Errorf(
				"used memory slots %d exceeds available capacity %d for character %d",
				usedMemorySlots, available, characterID)
		}
		availableSlots = available

		origActiveIndex := binary.LittleEndian.Uint32(section[112:])
		var newActiveIndex uint32
		if len(rawSpellIDs) == 0 {
			newActiveIndex = equippedSpellEmptyID
		} else if origActiveIndex != equippedSpellEmptyID && origActiveIndex < uint32(len(rawSpellIDs)) {
			newActiveIndex = origActiveIndex
		} else {
			newActiveIndex = 0
		}

		beforeSpells := section[:96]
		beforeTail := section[96:112]
		beforeActive := section[112:116]

		afterSpells := make([]byte, 96)
		for i := 0; i < 12; i++ {
			if i < len(rawSpellIDs) {
				binary.LittleEndian.PutUint32(afterSpells[i*8:], rawSpellIDs[i])
				binary.LittleEndian.PutUint32(afterSpells[i*8+4:], equippedSpellOccupiedFollower)
			} else {
				binary.LittleEndian.PutUint32(afterSpells[i*8:], equippedSpellEmptyID)
				binary.LittleEndian.PutUint32(afterSpells[i*8+4:], equippedSpellEmptyFollower)
			}
		}

		afterActive := make([]byte, 4)
		binary.LittleEndian.PutUint32(afterActive, newActiveIndex)

		if bytes.Equal(beforeSpells, afterSpells) && bytes.Equal(beforeActive, afterActive) {
			return nil
		}

		if err := loaded.snapshot.writeAt(sectionAt, afterSpells); err != nil {
			return fmt.Errorf("cannot write equipped spells of character %d: %w", characterID, err)
		}
		if err := loaded.snapshot.writeAt(sectionAt+112, afterActive); err != nil {
			loaded.snapshot.writeAt(sectionAt, beforeSpells)
			return fmt.Errorf("cannot write active spell index of character %d: %w", characterID, err)
		}

		written, verifyErr := loaded.snapshot.readAt(sectionAt, 116)
		if verifyErr == nil &&
			bytes.Equal(written[:96], afterSpells) &&
			bytes.Equal(written[96:112], beforeTail) &&
			bytes.Equal(written[112:116], afterActive) {
			return nil
		}

		rb1 := loaded.snapshot.writeAt(sectionAt, beforeSpells)
		rb2 := loaded.snapshot.writeAt(sectionAt+112, beforeActive)
		if rb1 != nil || rb2 != nil {
			return fmt.Errorf(
				"equipped spells of character %d could not be verified and could not be restored: %v, %v",
				characterID, rb1, rb2)
		}
		return errors.New("equipped spells mutation could not be verified; the save is unchanged")
	})
	if err != nil {
		return CharacterEquippedSpellsMutation{}, err
	}

	if rawSpellIDs == nil {
		rawSpellIDs = []uint32{}
	}

	return CharacterEquippedSpellsMutation{
		SaveSessionID:        saveSessionID,
		SaveRevision:         committed.SaveRevision,
		CharacterID:          characterID,
		RawMagicParamIDs:     rawSpellIDs,
		UsedMemorySlots:      usedMemorySlots,
		AvailableMemorySlots: availableSlots,
	}, nil
}

// readEquippedSpellRecords reads the fourteen physical records and validates
// every pair against the two combinations the game writes. The stored identifier
// is returned unchanged for both of them, including the empty sentinel; a pair
// that is neither is corrupt state and fails the whole read.
func readEquippedSpellRecords(
	source *codec,
	anchor, slotEnd int64,
	characterID int,
) ([equippedSpellSlotCount]uint32, error) {
	var records [equippedSpellSlotCount]uint32

	sectionAt := anchor + equippedSpellsSectionOffset
	if sectionAt+equippedSpellsReadSize > slotEnd {
		return records, fmt.Errorf("equipped spells of character %d do not fit into its slot", characterID)
	}
	section, err := source.readAt(sectionAt, equippedSpellsReadSize)
	if err != nil {
		return records, fmt.Errorf("cannot read equipped spells of character %d: %w", characterID, err)
	}

	for index := range records {
		record := section[index*equippedSpellRecordSize:]
		spellID := binary.LittleEndian.Uint32(record)
		follower := binary.LittleEndian.Uint32(record[4:])
		switch {
		case spellID == equippedSpellEmptyID && follower == equippedSpellEmptyFollower:
		case spellID != equippedSpellEmptyID && follower == equippedSpellOccupiedFollower:
		default:
			return [equippedSpellSlotCount]uint32{}, fmt.Errorf(
				"spell record %d of character %d stores the pair (0x%08X, 0x%08X), which is neither empty nor occupied",
				index, characterID, spellID, follower)
		}
		records[index] = spellID
	}
	return records, nil
}

// readAvailableMemorySlots computes how many memory slots the character may
// fill. The rule is the confirmed one: the base capacity plus the effective
// Memory Stones, capped by the game maximum, plus the Moon of Nokstella bonus
// when that talisman sits in an unlocked talisman field. Every input is read
// from the slot itself, and a missing range is a hard error rather than a
// silently reduced capacity.
func readAvailableMemorySlots(
	source *codec,
	anchor, base, slotEnd int64,
	characterID int,
) (int, error) {
	stones, err := readMemoryStones(source, anchor, slotEnd, characterID)
	if err != nil {
		return 0, err
	}
	if stones > spellMaxMemoryStones {
		stones = spellMaxMemoryStones
	}
	slots := spellBaseMemorySlots + int(stones)

	moon, err := wearsMoonOfNokstella(source, anchor, base, slotEnd, characterID)
	if err != nil {
		return 0, err
	}
	if moon {
		slots += moonOfNokstellaMemorySlots
	}
	if slots > spellMaxMemorySlots {
		return spellMaxMemorySlots, nil
	}
	return slots, nil
}

// readMemoryStones reports the effective Memory Stone count of the slot: the
// quantity of the common-item stack, falling back to the key-item stack only
// when no common stack holds the stone. A slot that holds no Memory Stone at all
// is a normal result of zero, but a section that does not fit into the slot is a
// hard error.
//
// The whole InventoryHeld section is read in one copy and scanned linearly; it
// is 0x9004 bytes, so an index would cost more than it saves.
func readMemoryStones(source *codec, anchor, slotEnd int64, characterID int) (uint32, error) {
	sectionAt := anchor + inventoryCommonOffset
	if sectionAt+inventorySectionSize > slotEnd {
		return 0, fmt.Errorf(
			"memory stones of character %d do not fit into its slot", characterID)
	}
	section, err := source.readAt(sectionAt, inventorySectionSize)
	if err != nil {
		return 0, fmt.Errorf("cannot read memory stones of character %d: %w", characterID, err)
	}

	commonEnd := inventoryCommonRecords * inventoryRecordSize
	if quantity, found := findMemoryStoneQuantity(section[:commonEnd]); found && quantity > 0 {
		return quantity, nil
	}
	keyStart := commonEnd + inventoryKeyCountHeader
	if quantity, found := findMemoryStoneQuantity(section[keyStart:]); found {
		return quantity, nil
	}
	return 0, nil
}

// findMemoryStoneQuantity returns the quantity of the first Memory Stone record
// in one inventory range. The stored quantity keeps its high bit, which is not
// part of the count, so it is masked off here and nowhere else.
func findMemoryStoneQuantity(records []byte) (uint32, bool) {
	for offset := 0; offset+inventoryRecordSize <= len(records); offset += inventoryRecordSize {
		if binary.LittleEndian.Uint32(records[offset:]) != memoryStoneHandle {
			continue
		}
		return binary.LittleEndian.Uint32(records[offset+4:]) & inventoryQuantityMask, true
	}
	return 0, false
}

// wearsMoonOfNokstella reports whether Moon of Nokstella sits in one of the
// unlocked talisman fields. A talisman that occupies a field the character has
// not unlocked grants nothing, which is why the unlocked count is read first and
// the locked fields are never inspected.
func wearsMoonOfNokstella(source *codec, anchor, base, slotEnd int64, characterID int) (bool, error) {
	unlocked, err := readUnlockedTalismanFields(source, anchor, base, characterID)
	if err != nil {
		return false, err
	}

	countAt := anchor + equippedSpellsProjectileCountOffset
	if countAt+4 > slotEnd {
		return false, fmt.Errorf("projectile count of character %d lies outside its slot", characterID)
	}
	rawCount, err := source.readAt(countAt, 4)
	if err != nil {
		return false, fmt.Errorf("cannot read projectile count of character %d: %w", characterID, err)
	}
	// The count is widened to int64 before it is multiplied, so a declared
	// length can never wrap into a small, seemingly valid offset.
	count := int64(binary.LittleEndian.Uint32(rawCount))
	if count > equippedSpellsMaxProjectileRecords {
		return false, fmt.Errorf(
			"character %d declares %d projectile records, want at most %d",
			characterID, count, equippedSpellsMaxProjectileRecords)
	}

	blockAt := countAt + 4 + count*equippedSpellsProjectileRecordSize
	if blockAt+equippedSpellsEquipmentBlockSize > slotEnd {
		return false, fmt.Errorf("talisman fields of character %d do not fit into its slot", characterID)
	}
	block, err := source.readAt(blockAt, equippedSpellsEquipmentBlockSize)
	if err != nil {
		return false, fmt.Errorf("cannot read talisman fields of character %d: %w", characterID, err)
	}

	for field := 0; field < unlocked && field < equippedSpellsTalismanFieldCount; field++ {
		raw := binary.LittleEndian.Uint32(block[(equippedSpellsFirstTalismanField+field)*4:])
		// Native saves normally store the complete game ID. Keep the family-bit
		// normalization for compatibility with already supported fixtures whose
		// upper nibble is absent.
		if (raw&0x0FFFFFFF)|0x20000000 == moonOfNokstellaItemID {
			return true, nil
		}
	}
	return false, nil
}

// readUnlockedTalismanFields turns the stored additional-slots byte into the
// number of talisman fields the character may use. The byte lies in front of the
// anchor, so an anchor without room for it is a hard error instead of a default
// of one field.
func readUnlockedTalismanFields(source *codec, anchor, base int64, characterID int) (int, error) {
	slotsAt := anchor + talismanSlotsOffset
	if slotsAt < base {
		return 0, fmt.Errorf("talisman slot count of character %d lies outside its slot", characterID)
	}
	raw, err := source.readAt(slotsAt, 1)
	if err != nil {
		return 0, fmt.Errorf("cannot read talisman slot count of character %d: %w", characterID, err)
	}
	additional := int(raw[0])
	if additional > talismanSlotsMax {
		additional = talismanSlotsMax
	}
	return additional + 1, nil
}
