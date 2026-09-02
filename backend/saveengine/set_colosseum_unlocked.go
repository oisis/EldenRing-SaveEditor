package saveengine

import (
	"fmt"
	"maps"
	"slices"
)

// SetColosseumUnlockedResult reports one committed colosseum unlock change.
//
// SaveRevision is the revision the change committed under, which is the previous
// one plus exactly 1. SaveSessionID and CharacterID match the request. Unlocked
// is the new state stored in the save event flags. Catalog identity does not
// belong to SaveEngine and is therefore added by the endpoint receipt.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type SetColosseumUnlockedResult struct {
	MutationReceipt
	CharacterID int  `json:"characterID"`
	Unlocked    bool `json:"unlocked"`
}

// colosseumFlagSets is the confirmed, closed set of flags one colosseum unlock
// consists of, keyed by the activation flag the catalog declares: the activation
// flag itself, the map POI marker of block 62, the NPC/event-memory marker of
// block 69 and the matchmaking gate marker of block 710. SaveForge 1.5.8 and
// 1.6.8 carry the byte-identical table and prove the same four members per
// arena. This is a single closed rule, not a dependency system: exactly these
// three activation flags are accepted and nothing else, so no caller can flip an
// arbitrary flag of a supported block through the colosseum entry point.
//
// The physical gate state lives in the WorldGeom blob and is not an event flag,
// so it is deliberately not touched.
var colosseumFlagSets = map[uint32][]uint32{
	60350: {60350, 62720, 69450, 710850}, // Caelid Colosseum
	60360: {60360, 62730, 69460, 710860}, // Limgrave Colosseum
	60370: {60370, 62740, 69470, 710870}, // Royal Colosseum
}

// colosseumGlobalEventFlagIDs fire once any colosseum is unlocked: the gameman
// marker, the shared event/map system flag and the block 69 global. They are
// SET-only, exactly as in SaveForge 1.5.8 and 1.6.8, because another unlocked
// colosseum may still own them and 60100 is shared with the Torrent progression
// the grace mutation also sets.
var colosseumGlobalEventFlagIDs = []uint32{6080, 60100, 69480}

// SetColosseumUnlocked sets or clears the unlock state of one colosseum together
// with the complete confirmed flag set the matchmaking and the map marker need.
//
// saveSessionID is matched exactly. characterID is the physical slot 0..9.
//
// unlockEventFlagID must be one of the three confirmed activation flags; every
// other identifier is rejected instead of being written at a guessed meaning.
// There is no fallback that would accept an arbitrary flag of block 60.
//
// expectedRevision must be a canonical decimal saveRevision and match the
// session's current revision exactly.
//
// unlocked true writes the four flags of the requested colosseum plus the three
// global flags; unlocked false clears the four flags of that colosseum only. The
// globals are never cleared. The physical gate in WorldGeom, summoning pools,
// graces, items, regions and every other flag stay untouched.
//
// An inactive slot is rejected fail-closed, leaving the session and snapshot
// untouched.
//
// The mutation runs inside one critical section under Engine.mutex. Every target
// bit is planned before the first write, flags sharing one byte are merged into a
// single non-overlapping write, and applyByteWrites verifies the result and
// restores every previous byte when it cannot. Persisting the snapshot stays the
// job of WriteSave; no file on disk is opened here.
func (engine *Engine) SetColosseumUnlocked(
	saveSessionID string,
	characterID int,
	unlockEventFlagID uint32,
	unlocked bool,
	expectedRevision string,
) (SetColosseumUnlockedResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetColosseumUnlockedResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	flagSet, confirmed := colosseumFlagSets[unlockEventFlagID]
	if !confirmed {
		return SetColosseumUnlockedResult{}, fmt.Errorf(
			"event flag %d is not a confirmed colosseum unlock flag %v",
			unlockEventFlagID, slices.Sorted(maps.Keys(colosseumFlagSets)))
	}

	// Every target of one call follows the same direction: unlocking writes the
	// arena set and the globals, locking clears the arena set alone. One mask per
	// byte is therefore enough — there is no mixed byte to plan for.
	targets := flagSet
	if unlocked {
		targets = append(append([]uint32{}, flagSet...), colosseumGlobalEventFlagIDs...)
	}

	// Every identifier is placed before the slot is touched, and flags sharing a
	// byte are merged here, so the plan handed to applyByteWrites covers each byte
	// exactly once.
	masks := make(map[int64]byte, len(targets))
	for _, id := range targets {
		position, err := resolveEventFlag(id)
		if err != nil {
			return SetColosseumUnlockedResult{}, err
		}
		masks[position.offset] |= 1 << position.bit
	}
	offsets := slices.Sorted(maps.Keys(masks))

	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetColosseumUnlocked, characterID, func(loaded *loadedSave) error {
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
					"cannot read the colosseum event flags of character %d: %w", characterID, err)
			}
			value := raw[0] &^ masks[offset]
			if unlocked {
				value = raw[0] | masks[offset]
			}
			writes = append(writes, byteWrite{at: at, data: []byte{value}})
		}
		return applyByteWrites(loaded.snapshot, writes)
	})
	if err != nil {
		return SetColosseumUnlockedResult{}, err
	}

	return SetColosseumUnlockedResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		Unlocked:        unlocked,
	}, nil
}
