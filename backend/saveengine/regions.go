package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Slot-data layout of the confirmed unlocked-regions section. The section has no
// fixed position inside a slot: it begins immediately behind GestureGameData,
// which itself sits behind the Storage Box and the variable-length
// acquired-projectiles section. The position of GestureGameData is therefore the
// only thing that has to be resolved, and gestureSectionStart already owns that
// walk for this exact slot with these exact bounds. This getter measures its own
// section from the end of that block instead of restating the chain, so the two
// readers cannot drift to different interpretations of the same slot layout.
const (
	// regionRecordSize is the stride of one stored region ID, and regionMaxCount
	// is the highest count accepted before the declared length is treated as
	// corrupt instead of followed. The limit is the confirmed one the save format
	// is read with: far above the count any native save carries and far below what
	// would let a declared length wrap or reach past the container.
	regionRecordSize = 4
	regionMaxCount   = 20000
)

// CharacterRegions is the raw unlocked-regions list of one physical save slot.
// This is the save side of the region surface: it carries no name, no area, no
// kind, no key and no unlock decision, and it reads no GameCatalog at all.
//
// RegionIDs keeps the physical native order and the stored values exactly as
// written. Nothing is sorted, deduplicated, filtered, masked or normalised, so a
// repeated ID and an ID this stage cannot explain both stay visible where the
// game put them. Deciding what a stored ID means belongs to the caller that owns
// the region definitions.
//
// Active reports the slot's UserData10 activity flag. An inactive slot —
// including a residual one, whose deleted character's region list is still in
// the file — reports Active false and an empty, non-nil list, and its slot data
// is never searched or read.
type CharacterRegions struct {
	SaveSessionID string   `json:"saveSessionID"`
	CharacterID   int      `json:"characterID"`
	Active        bool     `json:"active"`
	RegionIDs     []uint32 `json:"regionIDs"`
}

// GetRegions returns the raw unlocked-region IDs stored in one physical
// character slot of an existing session. Like the other character readers it
// reads the session's private snapshot through the codec only: it opens no file,
// writes nothing, changes no session and returns no snapshot byte. It calls no
// other getter and no endpoint, and it leaves revision, dirty state, undo and
// snapshot untouched.
//
// saveSessionID is matched exactly. It is never trimmed, normalised or guessed,
// so an empty, unknown or already closed identifier is rejected instead of
// resolving to a session.
//
// characterID is the slot index 0..9. An inactive or residual slot is a normal
// result, not an error, and its slot data is never read.
//
// For an active slot the list is located dynamically: GestureGameData is
// resolved through its confirmed anchor and the declared projectile length in
// front of it, the region count is read directly behind that fixed block, and
// the IDs follow the count. A count above the accepted maximum and a list
// reaching past the end of the slot or of the snapshot are hard errors. There is
// no fallback position, no partial result and nothing is guessed.
func (engine *Engine) GetRegions(saveSessionID string, characterID int) (CharacterRegions, error) {
	if saveSessionID == "" {
		return CharacterRegions{}, errors.New("saveSessionID is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterRegions{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterRegions{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterRegions{}, fmt.Errorf(
			"cannot read activity of character %d: %w", characterID, err)
	}

	regions := CharacterRegions{
		SaveSessionID: saveSessionID,
		CharacterID:   characterID,
		RegionIDs:     []uint32{},
	}
	if flag[0] != userData10ActiveFlagValue {
		// An inactive slot is reported from its flag alone, so the residual
		// region list of a deleted character is never located or decoded.
		return regions, nil
	}

	ids, err := readUnlockedRegions(loaded, characterID)
	if err != nil {
		return CharacterRegions{}, err
	}

	regions.Active = true
	regions.RegionIDs = ids
	return regions, nil
}

// readUnlockedRegions decodes the region list of one active character. Callers
// validate the character index and activity before using it.
func readUnlockedRegions(loaded *loadedSave, characterID int) ([]uint32, error) {
	gestureAt, err := gestureSectionStart(loaded, characterID)
	if err != nil {
		return nil, err
	}
	_, slotEnd := gestureSlotBounds(loaded.session.platform, characterID)

	countAt := gestureAt + gestureSectionSize
	if countAt+4 > slotEnd {
		return nil, fmt.Errorf(
			"unlocked region count of character %d lies outside its slot", characterID)
	}
	rawCount, err := loaded.snapshot.readAt(countAt, 4)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot read unlocked region count of character %d: %w", characterID, err)
	}
	// The count is widened to int64 before it is multiplied, so a declared
	// length can never wrap into a small, seemingly valid length.
	count := int64(binary.LittleEndian.Uint32(rawCount))
	if count > regionMaxCount {
		return nil, fmt.Errorf(
			"character %d declares %d unlocked regions, want at most %d",
			characterID, count, regionMaxCount)
	}
	size := count * regionRecordSize
	if countAt+4+size > slotEnd {
		return nil, fmt.Errorf(
			"unlocked regions of character %d do not fit into their slot", characterID)
	}

	ids := make([]uint32, count)
	if count == 0 {
		return ids, nil
	}
	// count is at most regionMaxCount here, so the size fits an int on every
	// platform the application builds for.
	section, err := loaded.snapshot.readAt(countAt+4, int(size))
	if err != nil {
		return nil, fmt.Errorf(
			"cannot read unlocked regions of character %d: %w", characterID, err)
	}
	for index := range ids {
		ids[index] = binary.LittleEndian.Uint32(section[index*regionRecordSize:])
	}
	return ids, nil
}
