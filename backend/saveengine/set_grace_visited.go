package saveengine

import (
	"fmt"
	"maps"
	"slices"
)

// SetGraceVisitedResult reports one committed visit state change.
//
// SaveRevision is the revision the change committed under, which is the previous
// one plus exactly 1. SaveSessionID and CharacterID match the request. Visited is
// the new state stored in the save event flags. Catalog identity does not belong
// to SaveEngine and is therefore added by the endpoint receipt.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type SetGraceVisitedResult struct {
	MutationReceipt
	CharacterID int  `json:"characterID"`
	Visited     bool `json:"visited"`
}

// graceFlagBlocks are the confirmed event-flag blocks the visit flag of a curated
// grace may lie in. The catalog enforces the same bound on its documents; this
// method restates it so a flag of another block can never be flipped through the
// grace entry point.
var graceFlagBlocks = []uint32{71, 72, 73, 74, 76}

// graceDoorFlagBlocks are the eighteen confirmed overworld ObjAct blocks the
// nineteen non-zero doorEventFlagID values of the curated Graces table lie in.
// resolveEventFlag also answers blocks that carry no door at all — the Gatefront
// companion blocks 4, 60 and 710 and every other supported resource block — so
// the door identifier is bounded here first and a progression flag can never be
// written under the name of a dungeon door.
var graceDoorFlagBlocks = []uint32{
	1033438, 1036518, 1037538, 1038528, 1039418, 1039488, 1040528, 1041378,
	1043338, 1043388, 1043398, 1045348, 1045518, 1045528, 1047408, 1048368,
	1050538, 1050558,
}

// gatefrontVisitEventFlagID is the visit flag of the Gatefront grace, the single
// confirmed grace whose activation the game accompanies with further flags.
//
// gatefrontCompanionEventFlagIDs is that confirmed set: the Spectral Steed
// Whistle unlock, the Melina give step, its world-state counterpart and the
// accept/refuse popup marker. They are SET-only, exactly as in SaveForge 1.5.8
// and 1.6.8, because normal progression or an item path may also have set them.
// This is a single closed exception, not a dependency system: no other grace has
// confirmed companions, and the Roundtable Hold flags are deliberately excluded.
// SaveEngine applies the set itself so no caller can name companion flags.
const gatefrontVisitEventFlagID uint32 = 76111

var gatefrontCompanionEventFlagIDs = []uint32{60100, 4680, 710520, 4681}

// SetGraceVisited sets or clears the visit state of one Site of Grace together
// with its confirmed dependencies.
//
// saveSessionID is matched exactly. characterID is the physical slot 0..9.
//
// visitEventFlagID is the confirmed visit flag of the grace and is always
// written. doorEventFlagID is the optional overworld ObjAct flag of a sealed
// dungeon entrance; zero means the grace has no door and no door byte is written.
//
// expectedRevision must be a canonical decimal saveRevision and match the
// session's current revision exactly.
//
// The visit flag and, when one is given, the door flag follow visited
// symmetrically. Companion flags are never accepted from a caller: the single
// confirmed case, Gatefront on an activation, is applied here and is SET-only,
// because another progression path may also have set those flags.
// LastRestedGrace, the map, the regions and the inventory are not touched — the
// game owns them.
//
// An inactive slot is rejected fail-closed, leaving the session and snapshot
// untouched.
//
// The mutation runs inside one critical section under Engine.mutex. Every target
// bit is planned before the first write, flags sharing one byte are merged into a
// single non-overlapping write, and applyByteWrites verifies the result and
// restores every previous byte when it cannot. Persisting the snapshot stays the
// job of WriteSave; no file on disk is opened here.
func (engine *Engine) SetGraceVisited(
	saveSessionID string,
	characterID int,
	visitEventFlagID uint32,
	doorEventFlagID uint32,
	visited bool,
	expectedRevision string,
) (SetGraceVisitedResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetGraceVisitedResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if block := visitEventFlagID / eventFlagsPerBlock; !slices.Contains(graceFlagBlocks, block) {
		return SetGraceVisitedResult{}, fmt.Errorf(
			"event flag %d lies in block %d, which is not a confirmed grace block %v",
			visitEventFlagID, block, graceFlagBlocks)
	}
	if block := doorEventFlagID / eventFlagsPerBlock; doorEventFlagID != 0 &&
		!slices.Contains(graceDoorFlagBlocks, block) {
		return SetGraceVisitedResult{}, fmt.Errorf(
			"event flag %d lies in block %d, which is not a confirmed grace door block",
			doorEventFlagID, block)
	}

	// The three accepted ranges are disjoint — visit blocks 71, 72, 73, 74 and 76,
	// the eighteen door blocks and the companion blocks 4, 60 and 710 — so no
	// identifier can land in targets twice.
	targets := map[uint32]bool{visitEventFlagID: visited}
	if doorEventFlagID != 0 {
		targets[doorEventFlagID] = visited
	}
	if visitEventFlagID == gatefrontVisitEventFlagID && visited {
		for _, id := range gatefrontCompanionEventFlagIDs {
			targets[id] = true
		}
	}

	// Every identifier is placed before the slot is touched, and flags sharing a
	// byte — 4680 and 4681 do — are merged here, so the plan handed to
	// applyByteWrites covers each byte exactly once.
	type byteMask struct{ set, clear byte }
	masks := make(map[int64]byteMask, len(targets))
	for id, state := range targets {
		position, err := resolveEventFlag(id)
		if err != nil {
			return SetGraceVisitedResult{}, err
		}
		mask := masks[position.offset]
		if state {
			mask.set |= 1 << position.bit
		} else {
			mask.clear |= 1 << position.bit
		}
		masks[position.offset] = mask
	}
	offsets := slices.Sorted(maps.Keys(masks))

	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetGraceVisited, characterID, func(loaded *loadedSave) error {
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
					"cannot read the grace event flags of character %d: %w", characterID, err)
			}
			mask := masks[offset]
			writes = append(writes, byteWrite{at: at, data: []byte{raw[0]&^mask.clear | mask.set}})
		}
		return applyByteWrites(loaded.snapshot, writes)
	})
	if err != nil {
		return SetGraceVisitedResult{}, err
	}

	return SetGraceVisitedResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		Visited:         visited,
	}, nil
}
