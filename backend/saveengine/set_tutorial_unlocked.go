package saveengine

import (
	"encoding/binary"
	"fmt"
)

// SetTutorialUnlockedResult reports one committed TutorialData membership
// change. Catalog identity does not belong to SaveEngine and is added by the
// endpoint receipt.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type SetTutorialUnlockedResult struct {
	MutationReceipt
	CharacterID int  `json:"characterID"`
	Unlocked    bool `json:"unlocked"`
}

// SetTutorialUnlocked adds or removes one TutorialParam row ID in the
// TutorialData list of one active character slot. The public endpoint resolves
// catalog identity to the row ID; SaveEngine alone interprets and mutates the
// block.
//
// The declared payload size is never changed, so the mutation is one in-place
// write of the count followed by the part of the ID array either list uses; the
// header, the payload size, unknown IDs and every byte outside that range stay
// as they are.
//
// Native lists are ascending, and a new ID is inserted at its ascending
// position. SaveForge 1.5.8 and 1.6.8 appended instead, so a save they touched
// may hold an unsorted list; such a list is left exactly as it is and a new ID
// is appended at the end. Nothing is ever sorted, deduplicated or normalised.
//
// unlocked false removes every occurrence of the exact ID, shifts the remaining
// entries left, zeroes the freed entries and lowers count accordingly. Whether
// the game re-triggers the popup or re-spawns a tutorial-bound world item after
// a removal is not measured; see the endpoint documentation.
//
// An idempotent call writes no byte. It still advances saveRevision and marks
// the session dirty under the existing commitCharacterRevision contract, but it
// records no undo point.
func (engine *Engine) SetTutorialUnlocked(
	saveSessionID string,
	characterID int,
	tutorialID uint32,
	unlocked bool,
	expectedRevision string,
) (SetTutorialUnlockedResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetTutorialUnlockedResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if tutorialID == 0 {
		return SetTutorialUnlockedResult{}, fmt.Errorf("tutorial ID must be non-zero")
	}

	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetTutorialUnlocked, characterID, func(loaded *loadedSave) error {
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

		layout, err := readTutorialData(loaded, characterID)
		if err != nil {
			return err
		}
		next, err := assignTutorialMembership(layout.ids, tutorialID, unlocked, layout.capacity)
		if err != nil {
			return fmt.Errorf("character %d: %w", characterID, err)
		}

		// Every real operation changes the list length: an insertion grows it and
		// a removal shrinks it. An equal length therefore means nothing to do.
		if len(next) == len(layout.ids) {
			return nil
		}

		// One contiguous range starting at the count and covering every entry
		// either list uses, so a removal also owns and zeroes the entries it frees.
		span := max(len(layout.ids), len(next))
		payload := make([]byte, tutorialDataCountSize+span*tutorialDataIDSize)
		binary.LittleEndian.PutUint32(payload, uint32(len(next)))
		for index, id := range next {
			binary.LittleEndian.PutUint32(
				payload[tutorialDataCountSize+index*tutorialDataIDSize:], id)
		}
		return applyByteWrites(loaded.snapshot, []byteWrite{
			{at: layout.countAt, data: payload},
		})
	})
	if err != nil {
		return SetTutorialUnlockedResult{}, err
	}

	return SetTutorialUnlockedResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		Unlocked:        unlocked,
	}, nil
}

// assignTutorialMembership returns the list the request asks for. It never
// reorders, deduplicates or normalises the entries it keeps.
func assignTutorialMembership(
	ids []uint32, tutorialID uint32, unlocked bool, capacity int64,
) ([]uint32, error) {
	if !unlocked {
		next := make([]uint32, 0, len(ids))
		for _, id := range ids {
			if id != tutorialID {
				next = append(next, id)
			}
		}
		return next, nil
	}

	ascending := true
	for index, id := range ids {
		if id == tutorialID {
			return ids, nil
		}
		if index > 0 && ids[index-1] >= id {
			ascending = false
		}
	}
	if int64(len(ids)) >= capacity {
		return nil, fmt.Errorf(
			"TutorialData is full at %d of %d entries", len(ids), capacity)
	}

	// A native list is ascending, so the new ID takes its ascending position. A
	// list a SaveForge 1.x writer appended to may be unsorted; it keeps its exact
	// physical order and receives the new ID at the end.
	at := len(ids)
	if ascending {
		for index, id := range ids {
			if id > tutorialID {
				at = index
				break
			}
		}
	}
	next := make([]uint32, 0, len(ids)+1)
	next = append(next, ids[:at]...)
	next = append(next, tutorialID)
	return append(next, ids[at:]...), nil
}
