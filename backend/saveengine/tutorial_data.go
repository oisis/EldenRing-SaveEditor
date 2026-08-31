package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// TutorialData stores a uint32 count followed by TutorialParam row IDs inside
// its declared payload. The header and the physical maximum are identical on PC
// and PS4 and are confirmed by SaveForge 1.5.8 and 1.6.8.
const (
	tutorialDataCountSize = 4
	tutorialDataIDSize    = 4
	tutorialDataMaxIDs    = 0xFF
)

// CharacterTutorialIDs is the raw TutorialData membership of one character.
// IDs remain in physical order and are not deduplicated or interpreted. An
// inactive slot returns Active false and an empty, non-nil list without reading
// the residual slot data.
type CharacterTutorialIDs struct {
	SaveSessionID string   `json:"saveSessionID"`
	SaveRevision  string   `json:"saveRevision"`
	CharacterID   int      `json:"characterID"`
	Active        bool     `json:"active"`
	IDs           []uint32 `json:"ids"`
}

// GetTutorialIDs reads the TutorialParam row IDs registered in TutorialData. It
// opens no file and changes no session state. The dynamic block is located from
// the same confirmed chain that owns GaItemGameData and the event-flag section;
// the save-declared payload size is always used instead of the legacy 0x400
// assumption.
func (engine *Engine) GetTutorialIDs(
	saveSessionID string, characterID int,
) (CharacterTutorialIDs, error) {
	if saveSessionID == "" {
		return CharacterTutorialIDs{}, errors.New("saveSessionID is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterTutorialIDs{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterTutorialIDs{}, fmt.Errorf(
			"characterID %d is outside the range 0..%d", characterID, characterSlotCount-1)
	}

	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterTutorialIDs{}, fmt.Errorf(
			"cannot read activity of character %d: %w", characterID, err)
	}
	result := CharacterTutorialIDs{
		SaveSessionID: saveSessionID,
		SaveRevision:  loaded.session.revisionString(),
		CharacterID:   characterID,
		IDs:           []uint32{},
	}
	if flag[0] != userData10ActiveFlagValue {
		return result, nil
	}

	layout, err := readTutorialData(loaded, characterID)
	if err != nil {
		return CharacterTutorialIDs{}, err
	}
	result.Active = true
	result.IDs = layout.ids
	return result, nil
}

// tutorialDataLayout is one character's validated TutorialData list: where its
// count lives, how many IDs the declared payload may hold and the IDs currently
// stored, in physical order.
type tutorialDataLayout struct {
	countAt  int64
	capacity int64
	ids      []uint32
}

// readTutorialData locates and validates the TutorialData list of one active
// character. It is the single owner of the block's bounds, count and capacity
// rules, so the getter and the mutation cannot drift apart. The caller must
// already hold Engine.mutex and must have established that the slot is active.
func readTutorialData(loaded *loadedSave, characterID int) (tutorialDataLayout, error) {
	sectionAt, slotEnd, err := tutorialDataStart(loaded, characterID)
	if err != nil {
		return tutorialDataLayout{}, err
	}
	payloadSize, err := eventFlagDeclaredValue(
		loaded, characterID, sectionAt+4, slotEnd, "tutorial size", eventFlagMaxDynamicSize)
	if err != nil {
		return tutorialDataLayout{}, err
	}
	if sectionAt+eventFlagDynamicHeaderSize+payloadSize > slotEnd {
		return tutorialDataLayout{}, fmt.Errorf(
			"tutorial data of character %d do not fit into its slot", characterID)
	}
	if !loaded.snapshot.covers(sectionAt, eventFlagDynamicHeaderSize+payloadSize) {
		return tutorialDataLayout{}, fmt.Errorf(
			"tutorial data of character %d do not fit into the save file", characterID)
	}

	// The count is part of the declared payload, so a payload too small to hold
	// it is malformed. Reading the four bytes anyway would read past the payload.
	if payloadSize < tutorialDataCountSize {
		return tutorialDataLayout{}, fmt.Errorf(
			"tutorial data of character %d declare a payload of %d bytes, which does not hold the %d-byte tutorial count field",
			characterID, payloadSize, tutorialDataCountSize)
	}

	countAt := sectionAt + eventFlagDynamicHeaderSize
	rawCount, err := loaded.snapshot.readAt(countAt, tutorialDataCountSize)
	if err != nil {
		return tutorialDataLayout{}, fmt.Errorf(
			"cannot read tutorial count of character %d: %w", characterID, err)
	}
	count := int64(binary.LittleEndian.Uint32(rawCount))
	maximumFromPayload := (payloadSize - tutorialDataCountSize) / tutorialDataIDSize
	if count > maximumFromPayload || count > tutorialDataMaxIDs {
		return tutorialDataLayout{}, fmt.Errorf(
			"tutorial count %d of character %d exceeds the declared payload capacity %d or hard cap %d",
			count, characterID, maximumFromPayload, tutorialDataMaxIDs)
	}

	rawIDs, err := loaded.snapshot.readAt(
		countAt+tutorialDataCountSize, int(count)*tutorialDataIDSize)
	if err != nil {
		return tutorialDataLayout{}, fmt.Errorf(
			"cannot read tutorial IDs of character %d: %w", characterID, err)
	}
	ids := make([]uint32, count)
	for index := range ids {
		ids[index] = binary.LittleEndian.Uint32(rawIDs[index*tutorialDataIDSize:])
	}

	capacity := maximumFromPayload
	if capacity > tutorialDataMaxIDs {
		capacity = tutorialDataMaxIDs
	}
	return tutorialDataLayout{countAt: countAt, capacity: capacity, ids: ids}, nil
}

// tutorialDataStart returns the first byte of TutorialData and its slot bound.
// The fixed GaItemGameData block immediately precedes it, so this small adapter
// reuses the existing owner of the dynamic walk instead of duplicating it.
func tutorialDataStart(loaded *loadedSave, characterID int) (int64, int64, error) {
	gaItemDataAt, slotEnd, err := eventFlagGaItemGameDataAt(loaded, characterID)
	if err != nil {
		return 0, 0, err
	}
	sectionAt := gaItemDataAt + eventFlagGaItemGameDataSize
	if sectionAt+eventFlagDynamicHeaderSize > slotEnd {
		return 0, 0, fmt.Errorf(
			"tutorial data header of character %d does not fit into its slot",
			characterID)
	}
	if !loaded.snapshot.covers(sectionAt, eventFlagDynamicHeaderSize) {
		return 0, 0, fmt.Errorf(
			"tutorial data header of character %d does not fit into the save file", characterID)
	}
	return sectionAt, slotEnd, nil
}
