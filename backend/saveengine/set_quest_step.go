package saveengine

import (
	"fmt"
	"maps"
	"slices"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// QuestFlagTarget represents one target event flag state in a quest step plan.
type QuestFlagTarget struct {
	ID    uint32
	Value bool
}

// SetQuestStepResult reports one committed quest step change.
//
// SaveRevision is the revision the change committed under, which is the previous
// one plus exactly 1. SaveSessionID and CharacterID match the request. Catalog
// identity does not belong to SaveEngine and is added by the endpoint receipt.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type SetQuestStepResult struct {
	MutationReceipt
	CharacterID int `json:"characterID"`
}

type questByteMask struct {
	set   byte
	clear byte
}

// SetQuestStep sets or clears the event flag bits declared by a curated quest
// step plan in one character slot of an existing session.
//
// saveSessionID is matched exactly. characterID is the physical slot 0..9.
//
// expectedRevision must be a canonical decimal saveRevision and match the
// session's current revision exactly.
//
// flags must be a non-empty, deduplicated, canonical list of target event flags
// whose positions all resolve in confirmedEventFlagBlocks.
//
// An inactive slot is rejected fail-closed, leaving the session and snapshot
// untouched.
//
// The mutation runs inside one critical section under Engine.mutex. Every target
// bit is resolved before the slot is touched. Flags sharing one byte are merged
// into a single non-overlapping write, and applyByteWrites verifies the result
// and restores every previous byte if verification fails. Persisting the
// snapshot stays the job of WriteSave; no file on disk is opened here.
func (engine *Engine) SetQuestStep(
	saveSessionID string,
	characterID int,
	flags []QuestFlagTarget,
	expectedRevision string,
) (SetQuestStepResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetQuestStepResult{}, apperror.InvalidRevision(expectedRevision)
	}
	if len(flags) == 0 {
		return SetQuestStepResult{}, fmt.Errorf("quest step requires at least one target flag")
	}

	seenIDs := make(map[uint32]struct{}, len(flags))
	masks := make(map[int64]questByteMask, len(flags))

	for _, flag := range flags {
		if flag.ID == 0 {
			return SetQuestStepResult{}, fmt.Errorf("target event flag ID must be non-zero")
		}
		if _, duplicate := seenIDs[flag.ID]; duplicate {
			return SetQuestStepResult{}, fmt.Errorf(
				"target event flag ID %d appears more than once in the canonical plan", flag.ID)
		}
		seenIDs[flag.ID] = struct{}{}

		position, err := resolveEventFlag(flag.ID)
		if err != nil {
			return SetQuestStepResult{}, err
		}

		mask := masks[position.offset]
		bitMask := byte(1 << position.bit)
		if flag.Value {
			mask.set |= bitMask
			mask.clear &^= bitMask
		} else {
			mask.clear |= bitMask
			mask.set &^= bitMask
		}
		masks[position.offset] = mask
	}

	offsets := slices.Sorted(maps.Keys(masks))

	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetQuestStep, characterID, func(loaded *loadedSave) error {
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

		sectionAt, err := eventFlagSectionStart(loaded, characterID)
		if err != nil {
			return err
		}

		writes := make([]byteWrite, 0, len(offsets))
		for _, offset := range offsets {
			at := sectionAt + offset
			raw, err := loaded.snapshot.readAt(at, 1)
			if err != nil {
				return fmt.Errorf(
					"cannot read the quest event flags of character %d: %w", characterID, err)
			}
			mask := masks[offset]
			value := (raw[0] &^ mask.clear) | mask.set
			writes = append(writes, byteWrite{at: at, data: []byte{value}})
		}
		return applyByteWrites(loaded.snapshot, writes)
	})
	if err != nil {
		return SetQuestStepResult{}, err
	}

	return SetQuestStepResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
	}, nil
}
