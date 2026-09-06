package saveengine

// The slot-management projection of GetSaveCharacters.
//
// It exists beside CharacterSummary and never replaces it: the summary keeps
// its deliberate contract that an inactive slot reveals nothing of the deleted
// character it may still carry. This projection adds only what the interface
// needs in order to present ten slots and to offer the operations the writers
// would actually accept, and it derives every rule from the same helpers the
// writers use, so a capability cannot drift away from the mutation it enables.
//
// A capability is a hint for the interface. Every writer still revalidates the
// slot and the expected revision, so a stale capability can never turn into an
// accepted mutation.

// Slot states. `unknown` is the fail-safe state: it is reported whenever the
// slot cannot be classified with confidence, and it carries no capability, so
// an unclassified slot is never presented as empty and never offers a
// destructive action.
const (
	CharacterSlotStateActive   = "active"
	CharacterSlotStateResidual = "residual"
	CharacterSlotStateEmpty    = "empty"
	CharacterSlotStateUnknown  = "unknown"
)

// CharacterSlotCapabilities names the slot operations the current state allows.
// CloneFrom marks a usable clone source, CloneInto a usable clone target;
// Delete covers both deleting an active character and clearing a residual slot,
// because DeleteCharacter is the one writer that performs either.
type CharacterSlotCapabilities struct {
	Activate   bool `json:"activate"`
	Deactivate bool `json:"deactivate"`
	CloneFrom  bool `json:"cloneFrom"`
	CloneInto  bool `json:"cloneInto"`
	Delete     bool `json:"delete"`
}

// CharacterSlot is one physical slot as the slot management sees it.
//
// StartingClassID is meaningful only while StartingClassKnown is true. The
// identifier is read from the profile summary of an active slot alone: a
// residual summary can be zeroed while the slot data is not, so its class byte
// would be a default value invented for a character that never had it.
type CharacterSlot struct {
	CharacterID        int                       `json:"characterID"`
	State              string                    `json:"state"`
	StartingClassID    uint8                     `json:"startingClassID"`
	StartingClassKnown bool                      `json:"startingClassKnown"`
	Capabilities       CharacterSlotCapabilities `json:"capabilities"`
}

// describeCharacterSlot classifies one slot from its activity flag and, for an
// inactive slot, from the same evidence DeleteCharacter and SetCharacterActive
// evaluate. A read or classification failure is not an error of the getter: it
// leaves the slot in the unknown state with no capability at all.
//
// ponytail: the inactive branch reads the whole slot block to prove it is
// zeroed, which is one full slot copy per inactive slot per call. The result is
// cached per session revision by the caller; introduce a zero-scan on the codec
// only if that read ever shows up in a measurement.
func describeCharacterSlot(loaded *loadedSave, characterID int, flag byte) CharacterSlot {
	slot := CharacterSlot{CharacterID: characterID, State: CharacterSlotStateUnknown}
	summaryAt := userData10Base(loaded.session.platform) + userData10SummaryOffset +
		int64(characterID)*userData10SummaryStride

	switch flag {
	case userData10ActiveFlagValue:
		slot.State = CharacterSlotStateActive
		slot.Capabilities = CharacterSlotCapabilities{
			Deactivate: true,
			CloneFrom:  true,
			Delete:     true,
		}
		if raw, err := loaded.snapshot.readAt(
			summaryAt+summaryStartingClassOffset, summaryIdentifierSize); err == nil {
			slot.StartingClassID = raw[0]
			slot.StartingClassKnown = true
		}
		return slot
	case userData10InactiveFlagValue:
	default:
		// An unsupported flag is exactly what every writer refuses, so the slot
		// is reported as unknown instead of being guessed into a state.
		return slot
	}

	slotAt := slotDataBase(loaded.session.platform, characterID)
	slotData, slotErr := loaded.snapshot.readAt(slotAt, characterSlotDataSize)
	summary, summaryErr := loaded.snapshot.readAt(summaryAt, userData10SummaryStride)
	if slotErr != nil || summaryErr != nil {
		return slot
	}

	// Completely zeroed is the one condition CloneCharacter accepts as a target,
	// so it is also the only condition that may be presented as `Empty`.
	if bytesAreZero(slotData) && bytesAreZero(summary) {
		slot.State = CharacterSlotStateEmpty
		slot.Capabilities.CloneInto = true
		return slot
	}

	occupied, err := characterCanBeDeleted(loaded, characterID, flag, slotData, summary)
	if err != nil || !occupied {
		// Data is present but no residual character can be established. The slot
		// is neither a clone target nor a deletion target, which is precisely
		// what both writers decide as well.
		return slot
	}

	slot.State = CharacterSlotStateResidual
	slot.Capabilities.Delete = true
	// Reactivation carries its own precondition, so the capability answers the
	// writer's own check instead of assuming that residual data is reactivatable.
	slot.Capabilities.Activate = validateResidualCharacter(loaded, characterID) == nil
	return slot
}
