package saveengine

import (
	"bytes"
	"errors"
	"fmt"
)

// CloneCharacterResult identifies both physical slots and the unique name
// assigned to one committed clone.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat.
type CloneCharacterResult struct {
	MutationReceipt
	SourceCharacterID int    `json:"sourceCharacterID"`
	TargetSlotID      int    `json:"targetSlotID"`
	Name              string `json:"name"`
}

// CloneCharacter copies one active character into one completely empty target
// slot. The source and every unrelated slot remain byte-exact.
func (engine *Engine) CloneCharacter(
	saveSessionID string,
	sourceCharacterID int,
	targetSlotID int,
	expectedRevision string,
) (CloneCharacterResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return CloneCharacterResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	// Both slot indices are checked before the commit, because the undo point of
	// the target slot is captured there and must not report the generic
	// characterID name for a rejected targetSlotID.
	if sourceCharacterID < 0 || sourceCharacterID >= characterSlotCount {
		return CloneCharacterResult{}, fmt.Errorf("sourceCharacterID %d is outside the range 0..%d",
			sourceCharacterID, characterSlotCount-1)
	}
	if targetSlotID < 0 || targetSlotID >= characterSlotCount {
		return CloneCharacterResult{}, fmt.Errorf("targetSlotID %d is outside the range 0..%d",
			targetSlotID, characterSlotCount-1)
	}

	cloneName := ""
	committed, err := engine.commitCharacterRevision(saveSessionID, kindCloneCharacter, targetSlotID, func(loaded *loadedSave) error {
		if sourceCharacterID == targetSlotID {
			return errors.New("sourceCharacterID and targetSlotID must differ")
		}
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return fmt.Errorf(
				"expectedRevision %q does not match the current saveRevision %q",
				expectedRevision, current)
		}

		sourceSlotAt := slotDataBase(loaded.session.platform, sourceCharacterID)
		sourceFlagAt := userData10Base(loaded.session.platform) +
			userData10ActiveFlagsOffset + int64(sourceCharacterID)
		sourceSummaryAt := userData10Base(loaded.session.platform) + userData10SummaryOffset +
			int64(sourceCharacterID)*userData10SummaryStride
		targetSlotAt := slotDataBase(loaded.session.platform, targetSlotID)
		targetFlagAt := userData10Base(loaded.session.platform) +
			userData10ActiveFlagsOffset + int64(targetSlotID)
		targetSummaryAt := userData10Base(loaded.session.platform) + userData10SummaryOffset +
			int64(targetSlotID)*userData10SummaryStride

		sourceFlag, err := loaded.snapshot.readAt(sourceFlagAt, 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of source character %d: %w",
				sourceCharacterID, err)
		}
		if sourceFlag[0] != userData10ActiveFlagValue {
			return fmt.Errorf("source character %d is not active", sourceCharacterID)
		}
		sourceSlot, err := loaded.snapshot.readAt(sourceSlotAt, characterSlotDataSize)
		if err != nil {
			return fmt.Errorf("cannot read source character %d: %w", sourceCharacterID, err)
		}
		sourceSummary, err := loaded.snapshot.readAt(sourceSummaryAt, userData10SummaryStride)
		if err != nil {
			return fmt.Errorf("cannot read profile summary of source character %d: %w",
				sourceCharacterID, err)
		}
		sourceAnchor, err := findStatsAnchor(
			loaded.snapshot, loaded.session.platform, sourceCharacterID)
		if err != nil {
			return err
		}
		sourceNameAt := sourceAnchor + playerCharacterNameOffset
		rawSourceName, err := loaded.snapshot.readAt(sourceNameAt, summaryNameSize)
		if err != nil {
			return fmt.Errorf("cannot read name of source character %d: %w",
				sourceCharacterID, err)
		}
		sourceName := decodeCharacterName(rawSourceName)
		if sourceName == "" {
			return fmt.Errorf("source character %d has no name", sourceCharacterID)
		}

		targetFlag, err := loaded.snapshot.readAt(targetFlagAt, 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of target slot %d: %w", targetSlotID, err)
		}
		if targetFlag[0] != userData10InactiveFlagValue {
			if targetFlag[0] == userData10ActiveFlagValue {
				return fmt.Errorf("targetSlotID %d is active", targetSlotID)
			}
			return fmt.Errorf("targetSlotID %d has unsupported activity flag 0x%02X",
				targetSlotID, targetFlag[0])
		}
		targetSlot, err := loaded.snapshot.readAt(targetSlotAt, characterSlotDataSize)
		if err != nil {
			return fmt.Errorf("cannot read target slot %d: %w", targetSlotID, err)
		}
		targetSummary, err := loaded.snapshot.readAt(targetSummaryAt, userData10SummaryStride)
		if err != nil {
			return fmt.Errorf("cannot read profile summary of target slot %d: %w", targetSlotID, err)
		}
		if !bytesAreZero(targetSlot) || !bytesAreZero(targetSummary) {
			return fmt.Errorf("targetSlotID %d is not completely empty", targetSlotID)
		}

		usedNames, err := cloneUsedCharacterNames(loaded)
		if err != nil {
			return err
		}
		cloneName = uniqueClonedCharacterName(sourceName, usedNames)
		encodedName, err := encodeCharacterName(cloneName)
		if err != nil {
			return fmt.Errorf("cannot encode cloned character name: %w", err)
		}

		clonedSlot := bytes.Clone(sourceSlot)
		nameOffset := sourceNameAt - sourceSlotAt
		copy(clonedSlot[nameOffset:nameOffset+summaryNameSize], encodedName)
		clonedSummary := bytes.Clone(sourceSummary)
		copy(clonedSummary[summaryNameOffset:summaryNameOffset+summaryNameSize], encodedName)

		if err := loaded.snapshot.writeAt(targetSlotAt, clonedSlot); err != nil {
			return fmt.Errorf("cannot write target slot %d: %w", targetSlotID, err)
		}
		if err := loaded.snapshot.writeAt(targetSummaryAt, clonedSummary); err != nil {
			return restoreCharacterSlotRanges(
				loaded.snapshot, targetSlotID, targetSlotAt, targetSlot,
				targetSummaryAt, targetSummary, targetFlagAt, targetFlag,
				fmt.Sprintf("source character %d was not cloned into target slot %d: the profile summary could not be written",
					sourceCharacterID, targetSlotID))
		}
		if err := loaded.snapshot.writeAt(targetFlagAt, []byte{userData10ActiveFlagValue}); err != nil {
			return restoreCharacterSlotRanges(
				loaded.snapshot, targetSlotID, targetSlotAt, targetSlot,
				targetSummaryAt, targetSummary, targetFlagAt, targetFlag,
				fmt.Sprintf("source character %d was not cloned into target slot %d: the activity flag could not be written",
					sourceCharacterID, targetSlotID))
		}

		slotWritten, slotErr := loaded.snapshot.readAt(targetSlotAt, characterSlotDataSize)
		summaryWritten, summaryErr := loaded.snapshot.readAt(
			targetSummaryAt, userData10SummaryStride)
		flagWritten, flagErr := loaded.snapshot.readAt(targetFlagAt, 1)
		if slotErr == nil && summaryErr == nil && flagErr == nil &&
			bytes.Equal(slotWritten, clonedSlot) &&
			bytes.Equal(summaryWritten, clonedSummary) &&
			flagWritten[0] == userData10ActiveFlagValue {
			return nil
		}
		return restoreCharacterSlotRanges(
			loaded.snapshot, targetSlotID, targetSlotAt, targetSlot,
			targetSummaryAt, targetSummary, targetFlagAt, targetFlag,
			fmt.Sprintf("source character %d was not cloned into target slot %d: the clone could not be verified",
				sourceCharacterID, targetSlotID))
	})
	if err != nil {
		return CloneCharacterResult{}, err
	}

	return CloneCharacterResult{
		MutationReceipt:   committed,
		SourceCharacterID: sourceCharacterID,
		TargetSlotID:      targetSlotID,
		Name:              cloneName,
	}, nil
}

func cloneUsedCharacterNames(loaded *loadedSave) (map[string]struct{}, error) {
	used := make(map[string]struct{}, characterSlotCount)
	for characterID := 0; characterID < characterSlotCount; characterID++ {
		flagAt := userData10Base(loaded.session.platform) +
			userData10ActiveFlagsOffset + int64(characterID)
		flag, err := loaded.snapshot.readAt(flagAt, 1)
		if err != nil {
			return nil, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
		}
		if flag[0] != userData10InactiveFlagValue && flag[0] != userData10ActiveFlagValue {
			return nil, fmt.Errorf("character %d has unsupported activity flag 0x%02X",
				characterID, flag[0])
		}

		summaryAt := userData10Base(loaded.session.platform) + userData10SummaryOffset +
			int64(characterID)*userData10SummaryStride
		rawSummaryName, err := loaded.snapshot.readAt(
			summaryAt+summaryNameOffset, summaryNameSize)
		if err != nil {
			return nil, fmt.Errorf("cannot read profile summary name of character %d: %w",
				characterID, err)
		}
		summaryName := decodeCharacterName(rawSummaryName)

		if flag[0] == userData10InactiveFlagValue {
			slotAt := slotDataBase(loaded.session.platform, characterID)
			slotData, err := loaded.snapshot.readAt(slotAt, characterSlotDataSize)
			if err != nil {
				return nil, fmt.Errorf("cannot inspect inactive character %d: %w", characterID, err)
			}
			if bytesAreZero(slotData) {
				if summaryName != "" {
					used[summaryName] = struct{}{}
				}
				continue
			}
		}

		anchor, err := findStatsAnchor(loaded.snapshot, loaded.session.platform, characterID)
		if err != nil {
			if flag[0] == userData10InactiveFlagValue && summaryName != "" {
				used[summaryName] = struct{}{}
				continue
			}
			return nil, fmt.Errorf("cannot resolve the name of character %d: %w", characterID, err)
		}
		rawPlayerName, err := loaded.snapshot.readAt(
			anchor+playerCharacterNameOffset, summaryNameSize)
		if err != nil {
			return nil, fmt.Errorf("cannot read PlayerGameData name of character %d: %w",
				characterID, err)
		}
		name := decodeCharacterName(rawPlayerName)
		if name == "" {
			name = summaryName
		}
		if name != "" {
			used[name] = struct{}{}
		}
	}
	return used, nil
}

func uniqueClonedCharacterName(base string, used map[string]struct{}) string {
	for suffixNumber := 2; ; suffixNumber++ {
		suffix := fmt.Sprintf(" %d", suffixNumber)
		candidate := trimToUTF16Units(base, summaryNameUnits-len(suffix)) + suffix
		if _, exists := used[candidate]; !exists {
			return candidate
		}
	}
}

func trimToUTF16Units(value string, maximum int) string {
	if maximum <= 0 {
		return ""
	}
	units := 0
	runes := make([]rune, 0, len(value))
	for _, value := range value {
		needed := 1
		if value > 0xFFFF {
			needed = 2
		}
		if units+needed > maximum {
			break
		}
		runes = append(runes, value)
		units += needed
	}
	return string(runes)
}
