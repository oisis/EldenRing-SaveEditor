package saveengine

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

const bellBearingEventFlagBlock = uint32(11109)

// SetBellBearingUnlockedResult reports one committed handed-in state change.
// Catalog identity remains the endpoint's responsibility.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type SetBellBearingUnlockedResult struct {
	MutationReceipt
	CharacterID int  `json:"characterID"`
	Unlocked    bool `json:"unlocked"`
}

// SetBellBearingUnlocked changes the acquisition flag of one Bell Bearing.
// Unlocking represents handing it to the Twin Maiden Husks and therefore also
// removes every matching raw-ID or goods-handle record from Inventory common,
// Inventory key and Storage common. Locking clears only the flag and never
// creates an item.
//
// The entire plan is validated before the first write. A matching record in
// Storage key is rejected because this project has no confirmed write contract
// for that section. Every accepted write is verified; any failure restores all
// bytes changed by this operation before the revision can advance.
func (engine *Engine) SetBellBearingUnlocked(
	saveSessionID string,
	characterID int,
	eventFlagID uint32,
	gameID uint32,
	unlocked bool,
	expectedRevision string,
) (SetBellBearingUnlockedResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetBellBearingUnlockedResult{}, apperror.InvalidRevision(expectedRevision)
	}
	block := eventFlagID / eventFlagsPerBlock
	if block != bellBearingEventFlagBlock {
		return SetBellBearingUnlockedResult{}, fmt.Errorf(
			"event flag %d lies in block %d, want Bell Bearing block %d",
			eventFlagID, block, bellBearingEventFlagBlock)
	}
	if gameID&gaItemHandleTypeMask != 0x40000000 {
		return SetBellBearingUnlockedResult{}, fmt.Errorf(
			"Bell Bearing game ID 0x%08X is not a goods ID", gameID)
	}
	goodsHandle, err := gaItemHandleForGameID(gameID)
	if err != nil {
		return SetBellBearingUnlockedResult{}, err
	}
	position, err := resolveEventFlag(eventFlagID)
	if err != nil {
		return SetBellBearingUnlockedResult{}, err
	}

	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetBellBearingUnlocked, characterID, func(loaded *loadedSave) error {
		if characterID < 0 || characterID >= characterSlotCount {
			return fmt.Errorf("characterID %d is outside the range 0..%d",
				characterID, characterSlotCount-1)
		}
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return apperror.RevisionConflict(expectedRevision, current)
		}

		flag, err := loaded.snapshot.readAt(
			userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
		}
		if flag[0] != userData10ActiveFlagValue {
			return fmt.Errorf("character %d is not active", characterID)
		}

		var writes []byteWrite
		if unlocked {
			writes, err = planBellBearingRemovals(loaded, characterID, gameID, goodsHandle)
			if err != nil {
				return err
			}
		}

		sectionAt, err := eventFlagSectionStart(loaded, characterID)
		if err != nil {
			return err
		}
		flagAt := sectionAt + position.offset
		oldFlag, err := loaded.snapshot.readAt(flagAt, 1)
		if err != nil {
			return fmt.Errorf("cannot read event flag %d of character %d: %w",
				eventFlagID, characterID, err)
		}
		newFlag := oldFlag[0]
		if unlocked {
			newFlag |= 1 << position.bit
		} else {
			newFlag &^= 1 << position.bit
		}
		writes = append(writes, byteWrite{at: flagAt, data: []byte{newFlag}})
		return applyByteWrites(loaded.snapshot, writes)
	})
	if err != nil {
		return SetBellBearingUnlockedResult{}, err
	}

	return SetBellBearingUnlockedResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		Unlocked:        unlocked,
	}, nil
}

// planBellBearingRemovals uses the existing container readers and removal plans
// so anchors, sections, sentinels and cleared-row representations keep their
// established owners. It returns only writes; the caller applies them together
// with the event flag after every range has been validated.
func planBellBearingRemovals(
	loaded *loadedSave,
	characterID int,
	gameID uint32,
	goodsHandle uint32,
) ([]byteWrite, error) {
	writes := make([]byteWrite, 0)
	counts := make(map[int64]int)
	for _, container := range []string{ownedContainerInventory, ownedContainerStorage} {
		records, err := readOwnedRecords(loaded, characterID, container)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			if record.gaItemHandle != gameID && record.gaItemHandle != goodsHandle {
				continue
			}
			locator := ownedItemLocator{
				characterID:      characterID,
				container:        container,
				containerSection: record.containerSection,
				physicalIndex:    record.physicalIndex,
			}
			plan, err := planOwnedItemRemoval(locator, record.ownedItemID)
			if err != nil {
				return nil, fmt.Errorf("cannot consume Bell Bearing 0x%08X: %w", gameID, err)
			}
			if err := rejectReferencedOwnedItem(
				loaded, locator, record.gaItemHandle, record.ownedItemID); err != nil {
				return nil, err
			}
			recordAt, countAt, recordSize, err := ownedItemRowAndCountAt(loaded, locator)
			if err != nil {
				return nil, err
			}
			if int64(len(plan.cleared)) != recordSize {
				return nil, fmt.Errorf(
					"Bell Bearing record states %d bytes and its container stores %d",
					len(plan.cleared), recordSize)
			}
			writes = append(writes, byteWrite{at: recordAt, data: plan.cleared})
			if plan.maintainsCount {
				counts[countAt]++
			}
		}
	}

	for countAt, removedCount := range counts {
		count, err := loaded.snapshot.uint32At(countAt)
		if err != nil {
			return nil, fmt.Errorf("cannot read Bell Bearing container count: %w", err)
		}
		removed := uint32(removedCount)
		if count < removed {
			// Both legacy versions left an already-inconsistent count unchanged
			// instead of wrapping it or repairing unrelated data.
			continue
		}
		writes = append(writes, byteWrite{at: countAt, data: littleEndianUint32(count - removed)})
	}
	return writes, nil
}
