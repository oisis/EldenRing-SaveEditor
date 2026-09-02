package saveengine

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

const userData10InactiveFlagValue = 0

var errCharacterActivityUnchanged = errors.New("character activity is unchanged")

// SetCharacterActiveResult reports the accepted activity state.
//
// Changed is the discriminator of the two success variants of this endpoint:
//
//   - true is a committed mutation. The embedded receipt is the complete one the
//     central commit path produced, and SaveRevision is the new revision.
//   - false is an idempotent request that found the slot already in the wanted
//     state. It commits nothing, so no operationID is minted, the revision, the
//     snapshot, the unsaved-changes flag, the undo point and the OwnedItemID
//     registry all stay exactly as they were, no session event is published, and
//     the three execution members of the receipt are absent. It is a domain
//     success and not an error.
type SetCharacterActiveResult struct {
	MutationReceipt
	Changed     bool `json:"changed"`
	CharacterID int  `json:"characterID"`
	Active      bool `json:"active"`
}

// MarshalJSON keeps MutationReceipt strict for committed mutations while the
// endpoint's idempotent success carries only the session and unchanged revision.
func (result SetCharacterActiveResult) MarshalJSON() ([]byte, error) {
	if result.Changed {
		type committedResult SetCharacterActiveResult
		return json.Marshal(committedResult(result))
	}
	return json.Marshal(struct {
		SaveSessionID string `json:"saveSessionID"`
		SaveRevision  string `json:"saveRevision"`
		Changed       bool   `json:"changed"`
		CharacterID   int    `json:"characterID"`
		Active        bool   `json:"active"`
	}{
		SaveSessionID: result.SaveSessionID,
		SaveRevision:  result.SaveRevision,
		Changed:       result.Changed,
		CharacterID:   result.CharacterID,
		Active:        result.Active,
	})
}

// SetCharacterActive changes only the confirmed UserData10 activity byte of
// one physical character slot. Deactivation preserves every residual byte.
// Reactivation is allowed only when the slot still carries a statistics anchor
// and a non-empty name in PlayerGameData or its profile summary.
func (engine *Engine) SetCharacterActive(
	saveSessionID string,
	characterID int,
	active bool,
	expectedRevision string,
) (SetCharacterActiveResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetCharacterActiveResult{}, apperror.InvalidRevision(expectedRevision)
	}

	target := byte(userData10InactiveFlagValue)
	if active {
		target = userData10ActiveFlagValue
	}

	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetCharacterActive, characterID, func(loaded *loadedSave) error {
		if characterID < 0 || characterID >= characterSlotCount {
			return fmt.Errorf("characterID %d is outside the range 0..%d",
				characterID, characterSlotCount-1)
		}

		current := loaded.session.revisionString()
		if expectedRevision != current {
			return apperror.RevisionConflict(expectedRevision, current)
		}

		flagAt := userData10Base(loaded.session.platform) +
			userData10ActiveFlagsOffset + int64(characterID)
		before, err := loaded.snapshot.readAt(flagAt, 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
		}
		if before[0] != userData10InactiveFlagValue &&
			before[0] != userData10ActiveFlagValue {
			return fmt.Errorf("character %d has unsupported activity flag 0x%02X",
				characterID, before[0])
		}
		if before[0] == target {
			return errCharacterActivityUnchanged
		}

		if active {
			if err := validateResidualCharacter(loaded, characterID); err != nil {
				return err
			}
		}

		if err := loaded.snapshot.writeAt(flagAt, []byte{target}); err != nil {
			return fmt.Errorf("cannot write activity of character %d: %w", characterID, err)
		}
		written, verifyErr := loaded.snapshot.readAt(flagAt, 1)
		if verifyErr == nil && written[0] == target {
			return nil
		}

		if restoreErr := loaded.snapshot.writeAt(flagAt, before); restoreErr != nil {
			return fmt.Errorf(
				"activity of character %d could not be verified and the prior flag could not be restored: %w",
				characterID, restoreErr)
		}
		return fmt.Errorf("activity mutation of character %d could not be verified; the save is unchanged",
			characterID)
	})
	// An idempotent request commits nothing, so it keeps the revision it matched
	// and carries the no-commit receipt of a mutation that never happened.
	receipt := committed
	changed := true
	if errors.Is(err, errCharacterActivityUnchanged) {
		receipt = noCommitReceipt(saveSessionID, expectedRevision)
		changed = false
	} else if err != nil {
		return SetCharacterActiveResult{}, err
	}

	return SetCharacterActiveResult{
		MutationReceipt: receipt,
		Changed:         changed,
		CharacterID:     characterID,
		Active:          active,
	}, nil
}

func validateResidualCharacter(loaded *loadedSave, characterID int) error {
	anchor, err := findStatsAnchor(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return fmt.Errorf("cannot reactivate character %d: %w", characterID, err)
	}

	playerName, err := loaded.snapshot.readAt(
		anchor+playerCharacterNameOffset, summaryNameSize)
	if err != nil {
		return fmt.Errorf("cannot read residual PlayerGameData name of character %d: %w",
			characterID, err)
	}
	summaryAt := userData10Base(loaded.session.platform) + userData10SummaryOffset +
		int64(characterID)*userData10SummaryStride + summaryNameOffset
	summaryName, err := loaded.snapshot.readAt(summaryAt, summaryNameSize)
	if err != nil {
		return fmt.Errorf("cannot read residual profile summary name of character %d: %w",
			characterID, err)
	}

	if decodeCharacterName(playerName) == "" && decodeCharacterName(summaryName) == "" {
		return fmt.Errorf("character %d has no residual character data to reactivate", characterID)
	}
	return nil
}
