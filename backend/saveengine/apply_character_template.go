package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// CharacterSpellsPlan describes the target equipped spell loadout for ApplyCharacterTemplate.
type CharacterSpellsPlan struct {
	RawMagicParamIDs []uint32 `json:"rawMagicParamIDs"`
	UsedMemorySlots  int      `json:"usedMemorySlots"`
}

// ApplyCharacterTemplatePlan describes the resolved, confirmed target values to apply to one character.
// Nil fields indicate sections not selected for mutation.
type ApplyCharacterTemplatePlan struct {
	Name       *string              `json:"name,omitempty"`
	Attributes *CharacterAttributes `json:"attributes,omitempty"`
	Spells     *CharacterSpellsPlan `json:"spells,omitempty"`
}

// ApplyCharacterTemplateResult reports the committed result of applying a template plan.
type ApplyCharacterTemplateResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
}

// ApplyCharacterTemplate atomically applies the resolved target plan (name, statistics,
// and/or equipped spells) to one active character slot under expectedRevision control.
//
// All preflight checks, starting-class minima, spell memory capacity, and physical positions
// 13-14 invariants are verified before any byte is written. The whole mutation is executed
// as a single atomic applyByteWrites transaction with verification and automatic rollback
// on error. Exactly one save revision is advanced and at most one undo point is recorded.
func (engine *Engine) ApplyCharacterTemplate(
	saveSessionID string,
	characterID int,
	plan ApplyCharacterTemplatePlan,
	expectedRevision string,
) (ApplyCharacterTemplateResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return ApplyCharacterTemplateResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return ApplyCharacterTemplateResult{}, fmt.Errorf(
			"characterID %d is outside the range 0..%d", characterID, characterSlotCount-1)
	}
	if plan.Name == nil && plan.Attributes == nil && plan.Spells == nil {
		return ApplyCharacterTemplateResult{}, errors.New("plan contains no character changes")
	}

	var encodedName []byte
	if plan.Name != nil {
		var err error
		encodedName, err = encodeCharacterName(*plan.Name)
		if err != nil {
			return ApplyCharacterTemplateResult{}, err
		}
	}

	var attrValues [characterAttributeCount]uint32
	var plannedLevel, requiredSoulMemory uint32
	if plan.Attributes != nil {
		var err error
		attrValues, plannedLevel, requiredSoulMemory, err = prepareCharacterAttributes(*plan.Attributes)
		if err != nil {
			return ApplyCharacterTemplateResult{}, err
		}
	}

	if plan.Spells != nil {
		if plan.Spells.UsedMemorySlots < 0 {
			return ApplyCharacterTemplateResult{}, fmt.Errorf(
				"used memory slots %d must not be negative", plan.Spells.UsedMemorySlots)
		}
		if len(plan.Spells.RawMagicParamIDs) > spellMaxMemorySlots {
			return ApplyCharacterTemplateResult{}, fmt.Errorf(
				"cannot equip more than %d spells; got %d", spellMaxMemorySlots, len(plan.Spells.RawMagicParamIDs))
		}
		seen := make(map[uint32]struct{}, len(plan.Spells.RawMagicParamIDs))
		for index, rawID := range plan.Spells.RawMagicParamIDs {
			if rawID == 0 || rawID >= equippedSpellRawIDLimit {
				return ApplyCharacterTemplateResult{}, fmt.Errorf(
					"spell slot %d: 0x%08X is not a raw MagicParam ID", index, rawID)
			}
			if _, duplicate := seen[rawID]; duplicate {
				return ApplyCharacterTemplateResult{}, fmt.Errorf(
					"spell slot %d: raw MagicParam ID 0x%08X is duplicated", index, rawID)
			}
			seen[rawID] = struct{}{}
		}
	}

	saveRevision, err := engine.commitCharacterRevision(saveSessionID, opApplyBuildTemplate, characterID, func(loaded *loadedSave) error {
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return fmt.Errorf(
				"expectedRevision %q does not match the current saveRevision %q",
				expectedRevision, current)
		}

		base := userData10Base(loaded.session.platform)
		flag, err := loaded.snapshot.readAt(base+userData10ActiveFlagsOffset+int64(characterID), 1)
		if err != nil {
			return fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
		}
		if flag[0] != userData10ActiveFlagValue {
			return fmt.Errorf("character %d is not active", characterID)
		}

		var writes []byteWrite

		// 1. Prepare profile name writes if planned.
		if plan.Name != nil {
			anchor, err := findStatsAnchor(loaded.snapshot, loaded.session.platform, characterID)
			if err != nil {
				return err
			}
			playerAt := anchor + playerCharacterNameOffset
			summaryAt := base + userData10SummaryOffset +
				int64(characterID)*userData10SummaryStride + summaryNameOffset

			writes = append(writes,
				byteWrite{at: playerAt, data: encodedName},
				byteWrite{at: summaryAt, data: encodedName},
			)
		}

		// 2. Prepare statistics writes if planned.
		if plan.Attributes != nil {
			ctx, err := planCharacterStatsState(loaded, characterID, attrValues, plannedLevel, requiredSoulMemory)
			if err != nil {
				return err
			}

			blockAfter := bytes.Clone(ctx.blockBefore)
			for index, value := range attrValues {
				binary.LittleEndian.PutUint32(blockAfter[index*4:], value)
			}
			binary.LittleEndian.PutUint32(blockAfter[statsBlockLevelPosition:], plannedLevel)
			binary.LittleEndian.PutUint32(blockAfter[statsBlockTotalGetSoulPosition:], ctx.soulMemory)

			summaryLevelAfter := make([]byte, summaryLevelSize)
			binary.LittleEndian.PutUint32(summaryLevelAfter, plannedLevel)

			writes = append(writes,
				byteWrite{at: ctx.blockAt, data: blockAfter},
				byteWrite{at: ctx.summaryLevelAt, data: summaryLevelAfter},
			)
		}

		// 3. Prepare equipped spells writes if planned.
		if plan.Spells != nil {
			slotBase := slotDataBase(loaded.session.platform, characterID)
			slotEnd := slotBase + characterSlotDataSize

			spellAnchor, err := loaded.snapshot.indexIn(slotBase, characterSlotDataSize, equippedSpellsAnchor)
			if err != nil {
				return fmt.Errorf("cannot search the equipped spells of character %d: %w", characterID, err)
			}
			if spellAnchor < 0 {
				return fmt.Errorf("character %d carries no equipped-spells anchor", characterID)
			}

			sectionAt := spellAnchor + equippedSpellsSectionOffset
			if sectionAt+116 > slotEnd {
				return fmt.Errorf("equipped spells of character %d do not fit into its slot", characterID)
			}

			existingRecords, err := readEquippedSpellRecords(loaded.snapshot, spellAnchor, slotEnd, characterID)
			if err != nil {
				return err
			}
			if existingRecords[12] != equippedSpellEmptyID || existingRecords[13] != equippedSpellEmptyID {
				return fmt.Errorf(
					"physical spell position 13 or 14 of character %d is not empty; mutation aborted", characterID)
			}

			available, err := readAvailableMemorySlots(loaded.snapshot, spellAnchor, slotBase, slotEnd, characterID)
			if err != nil {
				return err
			}
			if plan.Spells.UsedMemorySlots > available {
				return fmt.Errorf(
					"used memory slots %d exceeds available capacity %d for character %d",
					plan.Spells.UsedMemorySlots, available, characterID)
			}

			section, err := loaded.snapshot.readAt(sectionAt, 116)
			if err != nil {
				return fmt.Errorf("cannot read equipped spells section of character %d: %w", characterID, err)
			}

			origActiveIndex := binary.LittleEndian.Uint32(section[112:])
			var newActiveIndex uint32
			if len(plan.Spells.RawMagicParamIDs) == 0 {
				newActiveIndex = equippedSpellEmptyID
			} else if origActiveIndex != equippedSpellEmptyID && origActiveIndex < uint32(len(plan.Spells.RawMagicParamIDs)) {
				newActiveIndex = origActiveIndex
			} else {
				newActiveIndex = 0
			}

			afterSpells := make([]byte, 96)
			for i := 0; i < 12; i++ {
				if i < len(plan.Spells.RawMagicParamIDs) {
					binary.LittleEndian.PutUint32(afterSpells[i*8:], plan.Spells.RawMagicParamIDs[i])
					binary.LittleEndian.PutUint32(afterSpells[i*8+4:], equippedSpellOccupiedFollower)
				} else {
					binary.LittleEndian.PutUint32(afterSpells[i*8:], equippedSpellEmptyID)
					binary.LittleEndian.PutUint32(afterSpells[i*8+4:], equippedSpellEmptyFollower)
				}
			}

			afterActive := make([]byte, 4)
			binary.LittleEndian.PutUint32(afterActive, newActiveIndex)

			writes = append(writes,
				byteWrite{at: sectionAt, data: afterSpells},
				byteWrite{at: sectionAt + 112, data: afterActive},
			)
		}

		if err := applyByteWrites(loaded.snapshot, writes); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return ApplyCharacterTemplateResult{}, err
	}

	return ApplyCharacterTemplateResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  saveRevision,
		CharacterID:   characterID,
	}, nil
}
