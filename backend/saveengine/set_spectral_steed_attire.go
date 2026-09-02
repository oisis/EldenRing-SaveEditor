package saveengine

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// SpectralSteedAttire is the save-side identity of one Torrent appearance: the
// event flag that selects it and the goods item that has to be held for it.
// Public keys, names and icons stay the endpoint's responsibility.
//
// GameID is 0 for the default appearance, which the game offers without an item.
// Exactly one entry of a set may carry that zero.
type SpectralSteedAttire struct {
	EventFlagID uint32
	GameID      uint32
}

// SpectralSteedAttireMutation reports one committed Torrent appearance change.
// It is shared by both mutations of this file, which commit the same save state
// under different operation identities.
//
// SaveRevision is the revision the change committed under, which is the previous
// one plus exactly 1.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type SpectralSteedAttireMutation struct {
	MutationReceipt
	CharacterID int `json:"characterID"`
}

// SetSpectralSteedAttire activates exactly one Torrent appearance in one
// character slot of an existing session.
//
// attires is the complete mutually exclusive set; selectedEventFlagID must name
// one of them. An appearance that declares an item requires a positive record of
// that item in Inventory — Storage does not count and this mutation never adds
// the item, which stays the caller's separate AddItemToInventory step.
//
// Every check runs before the first byte moves, so a missing item leaves the
// session, the snapshot and the revision untouched. The write clears the whole
// set and sets the selected flag as one verified plan, so a save whose flags were
// all cleared or contradictory is resolved into exactly one set flag.
func (engine *Engine) SetSpectralSteedAttire(
	saveSessionID string,
	characterID int,
	attires []SpectralSteedAttire,
	selectedEventFlagID uint32,
	expectedRevision string,
) (SpectralSteedAttireMutation, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SpectralSteedAttireMutation{}, apperror.InvalidRevision(expectedRevision)
	}
	if err := validateSpectralSteedAttires(attires); err != nil {
		return SpectralSteedAttireMutation{}, err
	}
	selected, found := findSpectralSteedAttire(attires, selectedEventFlagID)
	if !found {
		return SpectralSteedAttireMutation{}, fmt.Errorf(
			"event flag %d is not one of the Spectral Steed Attire appearances", selectedEventFlagID)
	}

	committed, err := engine.commitCharacterRevision(
		saveSessionID, kindSetSpectralSteedAttire, characterID,
		func(loaded *loadedSave) error {
			if err := checkSpectralSteedSlot(loaded, characterID, expectedRevision); err != nil {
				return err
			}
			if selected.GameID != 0 {
				records, err := inventoryHeldRecords(loaded, characterID)
				if err != nil {
					return err
				}
				_, held, err := matchInventoryHeldRecord(
					records, selected.GameID, spectralSteedAttireLabel)
				if err != nil {
					return err
				}
				if !held {
					return fmt.Errorf(
						"%s 0x%08X is not in the Inventory of character %d, so it cannot be worn",
						spectralSteedAttireLabel, selected.GameID, characterID)
				}
			}
			writes, err := planSpectralSteedAttireFlags(
				loaded, characterID, attires, selectedEventFlagID)
			if err != nil {
				return err
			}
			return applyByteWrites(loaded.snapshot, writes)
		})
	if err != nil {
		return SpectralSteedAttireMutation{}, err
	}
	return SpectralSteedAttireMutation{
		MutationReceipt: committed,
		CharacterID:     characterID,
	}, nil
}

// LockAllSpectralSteedAttires removes every attire item of the set from
// Inventory and selects the default appearance in one atomic mutation.
//
// It is deliberately one operation rather than a composition of the item and the
// appearance mutations: a caller that ran them one after the other could leave
// the save holding an appearance whose item is gone, or the reverse, whenever the
// second call failed. Item records and flags are planned together and applied as
// one verified plan, so the slot either reaches the complete locked state or does
// not move at all.
func (engine *Engine) LockAllSpectralSteedAttires(
	saveSessionID string,
	characterID int,
	attires []SpectralSteedAttire,
	expectedRevision string,
) (SpectralSteedAttireMutation, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SpectralSteedAttireMutation{}, apperror.InvalidRevision(expectedRevision)
	}
	if err := validateSpectralSteedAttires(attires); err != nil {
		return SpectralSteedAttireMutation{}, err
	}
	var defaultEventFlagID uint32
	gameIDs := make([]uint32, 0, len(attires)-1)
	for _, attire := range attires {
		if attire.GameID == 0 {
			defaultEventFlagID = attire.EventFlagID
			continue
		}
		gameIDs = append(gameIDs, attire.GameID)
	}

	committed, err := engine.commitCharacterRevision(
		saveSessionID, kindLockAllSpectralSteedAttires, characterID,
		func(loaded *loadedSave) error {
			if err := checkSpectralSteedSlot(loaded, characterID, expectedRevision); err != nil {
				return err
			}
			records, err := inventoryHeldRecords(loaded, characterID)
			if err != nil {
				return err
			}
			itemWrites, err := planItemRemovals(
				loaded, characterID, records, gameIDs, spectralSteedAttireLabel)
			if err != nil {
				return err
			}
			flagWrites, err := planSpectralSteedAttireFlags(
				loaded, characterID, attires, defaultEventFlagID)
			if err != nil {
				return err
			}
			return applyByteWrites(loaded.snapshot, append(itemWrites, flagWrites...))
		})
	if err != nil {
		return SpectralSteedAttireMutation{}, err
	}
	return SpectralSteedAttireMutation{
		MutationReceipt: committed,
		CharacterID:     characterID,
	}, nil
}

// spectralSteedAttireLabel names the domain in every Inventory rejection these
// two mutations can produce.
const spectralSteedAttireLabel = "Spectral Steed Attire"

// validateSpectralSteedAttires rejects a set that cannot describe a mutually
// exclusive appearance choice: an empty set, a repeated event flag or game ID, an
// identifier this engine cannot place, a game ID that is not a goods ID, or a
// number of item-free default appearances other than one.
func validateSpectralSteedAttires(attires []SpectralSteedAttire) error {
	if len(attires) == 0 {
		return errors.New("no Spectral Steed Attire appearances were supplied")
	}
	seenFlags := make(map[uint32]struct{}, len(attires))
	seenGameIDs := make(map[uint32]struct{}, len(attires))
	defaults := 0
	for _, attire := range attires {
		if _, err := resolveEventFlag(attire.EventFlagID); err != nil {
			return err
		}
		if _, duplicate := seenFlags[attire.EventFlagID]; duplicate {
			return fmt.Errorf("Spectral Steed Attire set repeats event flag %d", attire.EventFlagID)
		}
		seenFlags[attire.EventFlagID] = struct{}{}
		if attire.GameID == 0 {
			defaults++
			continue
		}
		if attire.GameID&gaItemHandleTypeMask != 0x40000000 {
			return fmt.Errorf("Spectral Steed Attire game ID 0x%08X is not a goods ID", attire.GameID)
		}
		if _, duplicate := seenGameIDs[attire.GameID]; duplicate {
			return fmt.Errorf("Spectral Steed Attire set repeats game ID 0x%08X", attire.GameID)
		}
		seenGameIDs[attire.GameID] = struct{}{}
	}
	if defaults != 1 {
		return fmt.Errorf(
			"Spectral Steed Attire set declares %d appearances without an item, want exactly 1", defaults)
	}
	return nil
}

func findSpectralSteedAttire(
	attires []SpectralSteedAttire, eventFlagID uint32,
) (SpectralSteedAttire, bool) {
	for _, attire := range attires {
		if attire.EventFlagID == eventFlagID {
			return attire, true
		}
	}
	return SpectralSteedAttire{}, false
}

// checkSpectralSteedSlot is the shared precondition of both mutations: a slot
// index in range, the exact expected revision and an active character. A residual
// slot is rejected fail-closed instead of having its leftover bitfield written.
func checkSpectralSteedSlot(loaded *loadedSave, characterID int, expectedRevision string) error {
	if characterID < 0 || characterID >= characterSlotCount {
		return fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}
	current := loaded.session.revisionString()
	if expectedRevision != current {
		return apperror.RevisionConflict(expectedRevision, current)
	}
	active, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}
	if active[0] != userData10ActiveFlagValue {
		return fmt.Errorf("character %d is not active", characterID)
	}
	return nil
}

// planSpectralSteedAttireFlags clears every appearance flag of the set and sets
// the selected one, through the shared event-flag write planner.
func planSpectralSteedAttireFlags(
	loaded *loadedSave,
	characterID int,
	attires []SpectralSteedAttire,
	selectedEventFlagID uint32,
) ([]byteWrite, error) {
	sectionAt, err := eventFlagSectionStart(loaded, characterID)
	if err != nil {
		return nil, err
	}
	desired := make(map[uint32]bool, len(attires))
	for _, attire := range attires {
		desired[attire.EventFlagID] = false
	}
	desired[selectedEventFlagID] = true
	return planEventFlagWrites(loaded, sectionAt, desired)
}
