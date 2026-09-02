package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

const (
	// statsTotalGetSoulOffset is the confirmed lifetime-runes field of
	// PlayerGameData, called SoulMemory by the legacy implementation. It sits
	// directly behind the held-runes field.
	statsTotalGetSoulOffset = int64(-327)

	// statsWritableBlockOffset and statsWritableBlockSize describe the single
	// contiguous range this mutation owns: from the first attribute through the
	// last byte of TotalGetSoul. The held-runes field and the three unknown words
	// inside that range are read and written back byte for byte, so one write and
	// one rollback cover the whole PlayerGameData part of the mutation.
	statsWritableBlockOffset = int64(statsVigorOffset)
	statsWritableBlockSize   = int(statsTotalGetSoulOffset + 4 - statsWritableBlockOffset)

	// Positions of the written fields inside that range.
	statsBlockLevelPosition        = int(statsLevelOffset - statsWritableBlockOffset)
	statsBlockTotalGetSoulPosition = int(statsTotalGetSoulOffset - statsWritableBlockOffset)

	// statsClassOffset is the confirmed starting-class byte of PlayerGameData,
	// counted backwards from the same anchor as every other field of that struct.
	// It is the authoritative copy of the class and the only one this mutation
	// validates against; the ProfileSummary copy is menu data that may be stale.
	// It lies behind TotalGetSoul, outside the range this mutation writes.
	statsClassOffset = int64(-248)

	statsMinimumAttribute = uint32(1)
	statsMaximumAttribute = uint32(99)

	// statsLevelBase is the constant of the confirmed level formula
	// level = sum(attributes) - 79.
	statsLevelBase    = int64(79)
	statsMinimumLevel = int64(1)
	statsMaximumLevel = 8*int64(statsMaximumAttribute) - statsLevelBase

	// LevelPolicyRecalculate is the only accepted levelPolicy. It is matched
	// exactly: the value is never trimmed, lower-cased or otherwise normalised.
	LevelPolicyRecalculate = "recalculate"
)

// characterAttributeCount is the number of writable attributes. Their order is
// the confirmed save order and is shared by every table in this file.
const characterAttributeCount = 8

var characterAttributeNames = [characterAttributeCount]string{
	"vigor", "mind", "endurance", "strength",
	"dexterity", "intelligence", "faith", "arcane",
}

// startingClassDefinition is the complete, immutable definition of one starting
// class as the embedded GameCatalog declares it: the eight base attributes in
// the confirmed save order and the base level of the class.
//
// It is the single source of truth for both. Nothing in SaveEngine keeps a
// second class table, and Level is the catalog fact, never the value the level
// formula would derive from the attributes.
type startingClassDefinition struct {
	attributes [characterAttributeCount]uint32
	level      uint32
}

var (
	startingClassOnce  sync.Once
	startingClassTable map[uint8]startingClassDefinition
	startingClassErr   error
)

func loadStartingClassDefinitions() (map[uint8]startingClassDefinition, error) {
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		return nil, fmt.Errorf("load embedded catalog data: %w", err)
	}
	table := make(map[uint8]startingClassDefinition)
	for _, res := range data.Resources() {
		if res.Kind != schema.ResourceKindClass || res.Class == nil {
			continue
		}
		classDoc := res.Class
		if !classDoc.StartingClassID.Known {
			return nil, fmt.Errorf("class resource %q has unknown startingClassID", res.Key)
		}
		id := classDoc.StartingClassID.Value
		if id > 255 {
			return nil, fmt.Errorf("class resource %q startingClassID %d exceeds uint8", res.Key, id)
		}
		if !classDoc.Vigor.Known || classDoc.Vigor.Value == 0 ||
			!classDoc.Mind.Known || classDoc.Mind.Value == 0 ||
			!classDoc.Endurance.Known || classDoc.Endurance.Value == 0 ||
			!classDoc.Strength.Known || classDoc.Strength.Value == 0 ||
			!classDoc.Dexterity.Known || classDoc.Dexterity.Value == 0 ||
			!classDoc.Intelligence.Known || classDoc.Intelligence.Value == 0 ||
			!classDoc.Faith.Known || classDoc.Faith.Value == 0 ||
			!classDoc.Arcane.Known || classDoc.Arcane.Value == 0 {
			return nil, fmt.Errorf("class resource %q has missing, unknown or zero attribute fact", res.Key)
		}
		if !classDoc.Level.Known || classDoc.Level.Value == 0 {
			return nil, fmt.Errorf("class resource %q has missing, unknown or zero level fact", res.Key)
		}
		table[uint8(id)] = startingClassDefinition{
			attributes: [characterAttributeCount]uint32{
				classDoc.Vigor.Value,
				classDoc.Mind.Value,
				classDoc.Endurance.Value,
				classDoc.Strength.Value,
				classDoc.Dexterity.Value,
				classDoc.Intelligence.Value,
				classDoc.Faith.Value,
				classDoc.Arcane.Value,
			},
			level: classDoc.Level.Value,
		}
	}
	return table, nil
}

// startingClass resolves the shared definition of one starting class. An
// identifier outside the twelve confirmed classes is an error, never a skipped
// check: an unknown class carries neither confirmed minima nor a confirmed base
// level, so no rule that needs them may proceed.
func startingClass(startingClassID uint8) (startingClassDefinition, error) {
	startingClassOnce.Do(func() {
		startingClassTable, startingClassErr = loadStartingClassDefinitions()
	})
	if startingClassErr != nil {
		return startingClassDefinition{}, startingClassErr
	}
	definition, exists := startingClassTable[startingClassID]
	if !exists {
		return startingClassDefinition{}, fmt.Errorf(
			"starting class %d is unknown; its attribute minima are not confirmed",
			startingClassID)
	}
	return definition, nil
}

// LegalAttributesFor returns the attribute set closest to the supplied one that
// satisfies both confirmed attribute rules: the absolute range 1..99 and the
// per-attribute minimum of the character's own starting class resolved from the
// GameCatalog class documents. Each attribute is moved the smallest distance
// that makes it legal, and an already legal attribute is returned unchanged.
//
// It exists so a consumer deriving a corrected attribute set — currently
// GetRepairPlan — applies exactly the rules SetCharacterStats enforces, instead
// of keeping a second copy of the range and the class table. The two would
// otherwise be free to drift, and a plan built against a stale copy would be
// rejected by the very endpoint meant to execute it.
//
// An unknown starting class is an error, not a skipped check: a class outside
// the twelve confirmed ones carries no known minima, so no legal set can be derived
// for it. The class minimum is applied after the range, because every confirmed
// class minimum already lies inside 1..99 and is therefore the stricter bound.
//
// This applies rules; it decides no policy. Whether a caller may write the
// result is the caller's contract, not this function's.
func LegalAttributesFor(
	attributes CharacterAttributes,
	startingClassID uint8,
) (CharacterAttributes, error) {
	definition, err := startingClass(startingClassID)
	if err != nil {
		return CharacterAttributes{}, err
	}
	minima := definition.attributes

	values := attributes.ordered()
	for index, value := range values {
		if value < statsMinimumAttribute {
			value = statsMinimumAttribute
		}
		if value > statsMaximumAttribute {
			value = statsMaximumAttribute
		}
		if value < minima[index] {
			value = minima[index]
		}
		values[index] = value
	}
	return CharacterAttributes{
		Vigor: values[0], Mind: values[1], Endurance: values[2], Strength: values[3],
		Dexterity: values[4], Intelligence: values[5], Faith: values[6], Arcane: values[7],
	}, nil
}

// CharacterAttributes is the complete writable attribute set of one character.
// All eight fields are mandatory: the transport rejects a request that omits one
// instead of reading the omission as the illegal value zero.
type CharacterAttributes struct {
	Vigor        uint32 `json:"vigor"`
	Mind         uint32 `json:"mind"`
	Endurance    uint32 `json:"endurance"`
	Strength     uint32 `json:"strength"`
	Dexterity    uint32 `json:"dexterity"`
	Intelligence uint32 `json:"intelligence"`
	Faith        uint32 `json:"faith"`
	Arcane       uint32 `json:"arcane"`
}

// ordered returns the eight attributes in the confirmed save order, so range
// checks, class minima and the physical write all walk the same sequence.
func (attributes CharacterAttributes) ordered() [characterAttributeCount]uint32 {
	return [characterAttributeCount]uint32{
		attributes.Vigor, attributes.Mind, attributes.Endurance, attributes.Strength,
		attributes.Dexterity, attributes.Intelligence, attributes.Faith, attributes.Arcane,
	}
}

// SetCharacterStatsResult reports one committed statistics assignment. It
// returns the accepted attributes together with the two values SaveEngine
// derived from them, and exposes no offset, raw byte or starting class.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat.
type SetCharacterStatsResult struct {
	MutationReceipt
	CharacterID int                 `json:"characterID"`
	Attributes  CharacterAttributes `json:"attributes"`
	Level       uint32              `json:"level"`
	SoulMemory  uint32              `json:"soulMemory"`
}

// SetCharacterStats atomically assigns the eight attributes of one active
// character, together with the two values the save keeps consistent with them:
// the level, which is always recalculated from the attributes and stored both in
// PlayerGameData and in the character's ProfileSummary, and TotalGetSoul, which
// is raised to the minimum lifetime runes that level requires and otherwise left
// exactly as it is.
//
// Nothing else changes. HP, FP and SP with their maximum and base maximum, the
// held runes, the starting class, the name, the appearance, the inventory and
// every unrelated byte are preserved; the combat statistics the game derives
// from the attributes are not recomputed here.
func (engine *Engine) SetCharacterStats(
	saveSessionID string,
	characterID int,
	attributes CharacterAttributes,
	levelPolicy string,
	expectedRevision string,
) (SetCharacterStatsResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetCharacterStatsResult{}, apperror.InvalidRevision(expectedRevision)
	}
	if levelPolicy != LevelPolicyRecalculate {
		return SetCharacterStatsResult{}, fmt.Errorf(
			"levelPolicy must be %q; got %q", LevelPolicyRecalculate, levelPolicy)
	}

	values, level, err := prepareCharacterAttributes(attributes)
	if err != nil {
		return SetCharacterStatsResult{}, err
	}

	var soulMemory uint32
	committed, err := engine.commitCharacterRevision(saveSessionID, kindSetCharacterStats, characterID, func(loaded *loadedSave) error {
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return apperror.RevisionConflict(expectedRevision, current)
		}

		ctx, err := planCharacterStatsState(loaded, characterID, values, level)
		if err != nil {
			return err
		}
		soulMemory = ctx.soulMemory

		blockAfter := bytes.Clone(ctx.blockBefore)
		for index, value := range values {
			binary.LittleEndian.PutUint32(blockAfter[index*4:], value)
		}
		binary.LittleEndian.PutUint32(blockAfter[statsBlockLevelPosition:], level)
		binary.LittleEndian.PutUint32(blockAfter[statsBlockTotalGetSoulPosition:], soulMemory)

		summaryLevelAfter := make([]byte, summaryLevelSize)
		binary.LittleEndian.PutUint32(summaryLevelAfter, level)

		if bytes.Equal(ctx.blockBefore, blockAfter) && bytes.Equal(ctx.summaryLevelBefore, summaryLevelAfter) {
			return nil
		}

		if err := loaded.snapshot.writeAt(ctx.blockAt, blockAfter); err != nil {
			return fmt.Errorf("cannot write statistics of character %d: %w", characterID, err)
		}
		if err := loaded.snapshot.writeAt(ctx.summaryLevelAt, summaryLevelAfter); err != nil {
			return restoreCharacterStats(loaded.snapshot, characterID,
				ctx.blockAt, ctx.blockBefore, ctx.summaryLevelAt, ctx.summaryLevelBefore,
				fmt.Sprintf("cannot write profile summary level of character %d: %v", characterID, err))
		}

		blockWritten, blockErr := loaded.snapshot.readAt(ctx.blockAt, statsWritableBlockSize)
		summaryWritten, summaryErr := loaded.snapshot.readAt(ctx.summaryLevelAt, summaryLevelSize)
		if blockErr == nil && summaryErr == nil &&
			bytes.Equal(blockWritten, blockAfter) && bytes.Equal(summaryWritten, summaryLevelAfter) {
			return nil
		}

		return restoreCharacterStats(loaded.snapshot, characterID,
			ctx.blockAt, ctx.blockBefore, ctx.summaryLevelAt, ctx.summaryLevelBefore,
			fmt.Sprintf("statistics of character %d could not be verified", characterID))
	})
	if err != nil {
		return SetCharacterStatsResult{}, err
	}

	return SetCharacterStatsResult{
		MutationReceipt: committed,
		CharacterID:     characterID,
		Attributes:      attributes,
		Level:           level,
		SoulMemory:      soulMemory,
	}, nil
}

// prepareCharacterAttributes is a pure validator for ordered values and the
// recalculated level.
//
// The minimum SoulMemory is deliberately not derived here: it depends on the
// base level of the character's own starting class, which is known only once the
// class byte has been read from the locked snapshot. It is therefore computed in
// planCharacterStatsState, next to that read.
func prepareCharacterAttributes(attributes CharacterAttributes) (values [characterAttributeCount]uint32, level uint32, err error) {
	values = attributes.ordered()
	lvl, err := recalculateCharacterLevel(values)
	if err != nil {
		return [characterAttributeCount]uint32{}, 0, err
	}
	return values, lvl, nil
}

type plannedStatsContext struct {
	blockAt            int64
	blockBefore        []byte
	summaryLevelAt     int64
	summaryLevelBefore []byte
	soulMemory         uint32
}

// planCharacterStatsState reads and validates starting class and soul memory on a locked save snapshot.
func planCharacterStatsState(
	loaded *loadedSave,
	characterID int,
	values [characterAttributeCount]uint32,
	level uint32,
) (plannedStatsContext, error) {
	if characterID < 0 || characterID >= characterSlotCount {
		return plannedStatsContext{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	base := userData10Base(loaded.session.platform)
	flag, err := loaded.snapshot.readAt(base+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return plannedStatsContext{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}
	if flag[0] != userData10ActiveFlagValue {
		return plannedStatsContext{}, fmt.Errorf("character %d is not active", characterID)
	}

	anchor, err := findStatsAnchor(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return plannedStatsContext{}, err
	}

	rawClass, err := loaded.snapshot.readAt(anchor+statsClassOffset, 1)
	if err != nil {
		return plannedStatsContext{}, fmt.Errorf("cannot read starting class of character %d: %w", characterID, err)
	}
	if err := validateAgainstStartingClass(values, rawClass[0]); err != nil {
		return plannedStatsContext{}, err
	}
	definition, err := startingClass(rawClass[0])
	if err != nil {
		return plannedStatsContext{}, err
	}
	requiredSoulMemory := minimumSoulMemoryForLevel(level, definition.level)

	summary := base + userData10SummaryOffset + int64(characterID)*userData10SummaryStride
	blockAt := anchor + statsWritableBlockOffset
	summaryLevelAt := summary + summaryLevelOffset
	blockBefore, err := loaded.snapshot.readAt(blockAt, statsWritableBlockSize)
	if err != nil {
		return plannedStatsContext{}, fmt.Errorf("cannot read statistics of character %d: %w", characterID, err)
	}
	summaryLevelBefore, err := loaded.snapshot.readAt(summaryLevelAt, summaryLevelSize)
	if err != nil {
		return plannedStatsContext{}, fmt.Errorf("cannot read profile summary level of character %d: %w", characterID, err)
	}

	storedSoulMemory := binary.LittleEndian.Uint32(blockBefore[statsBlockTotalGetSoulPosition:])
	sm := storedSoulMemory
	if sm < requiredSoulMemory {
		sm = requiredSoulMemory
	}

	return plannedStatsContext{
		blockAt:            blockAt,
		blockBefore:        blockBefore,
		summaryLevelAt:     summaryLevelAt,
		summaryLevelBefore: summaryLevelBefore,
		soulMemory:         sm,
	}, nil
}

// PlanCharacterStats calculates the resulting level and required SoulMemory for a
// proposed attribute set against the starting class of an active character without mutating the save.
func (engine *Engine) PlanCharacterStats(
	saveSessionID string,
	characterID int,
	attributes CharacterAttributes,
) (level uint32, soulMemory uint32, err error) {
	if saveSessionID == "" {
		return 0, 0, apperror.MissingField("saveSessionID")
	}

	values, lvl, err := prepareCharacterAttributes(attributes)
	if err != nil {
		return 0, 0, err
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return 0, 0, apperror.UnknownSaveSession(saveSessionID)
	}

	ctx, err := planCharacterStatsState(loaded, characterID, values, lvl)
	if err != nil {
		return 0, 0, err
	}

	return lvl, ctx.soulMemory, nil
}

// recalculateCharacterLevel validates the absolute range of every attribute and
// returns the level the confirmed formula derives from them. The sum is taken in
// a type that cannot overflow, and the resulting level must itself be legal.
func recalculateCharacterLevel(values [characterAttributeCount]uint32) (uint32, error) {
	total := int64(0)
	for index, value := range values {
		if value < statsMinimumAttribute || value > statsMaximumAttribute {
			return 0, fmt.Errorf("attributes.%s %d is outside the range %d..%d",
				characterAttributeNames[index], value,
				statsMinimumAttribute, statsMaximumAttribute)
		}
		total += int64(value)
	}
	level := total - statsLevelBase
	if level < statsMinimumLevel || level > statsMaximumLevel {
		return 0, fmt.Errorf("recalculated level %d is outside the range %d..%d",
			level, statsMinimumLevel, statsMaximumLevel)
	}
	return uint32(level), nil
}

// validateAgainstStartingClass rejects any attribute below the base value of the
// character's own starting class, as stored in its PlayerGameData and resolved
// from the GameCatalog class documents. An identifier outside the twelve confirmed
// classes is a hard rejection, not a skipped check: an unknown class carries no
// known minima, so its save must not be written.
func validateAgainstStartingClass(values [characterAttributeCount]uint32, startingClassID uint8) error {
	definition, err := startingClass(startingClassID)
	if err != nil {
		return err
	}
	minima := definition.attributes
	for index, value := range values {
		if value < minima[index] {
			return fmt.Errorf("attributes.%s %d is below the starting-class minimum %d",
				characterAttributeNames[index], value, minima[index])
		}
	}
	return nil
}

// minimumSoulMemoryForLevel returns the total runes a character must have earned
// to climb from the base level of its own starting class to the given level: the
// sum of the per-level costs
// cost(n) = floor(0.02*n^3 + 3.06*n^2 + 105.6*n - 895), clamped to zero, taken
// over the levels classBaseLevel+1 .. level.
//
// The base level of the class is the floor of that sum because a character never
// paid for it: the native vanilla save shows every freshly created class sitting
// at its own base level 1..10 with TotalGetSoul exactly zero. A level at or below
// that base therefore requires nothing, and summing from level 1 would report a
// legal untouched character as being below the minimum.
//
// The sum is evaluated in integers so the result cannot depend on the host's
// floating-point behaviour. Six per-level costs are corrected by one to keep the
// established results at the boundaries where the historical floating-point
// evaluation rounded down; without them the totals from level 45 upwards would
// drift from the confirmed reference values. Each correction stays attached to
// its own level, so a partial sum applies exactly the ones it covers. The
// maximum possible result, 1692560963 at level 713 above base level 1, fits into
// uint32, so no clamp is needed.
func minimumSoulMemoryForLevel(level uint32, classBaseLevel uint32) uint32 {
	total := int64(0)
	for step := int64(classBaseLevel) + 1; step <= int64(level); step++ {
		cost := (2*step*step*step + 306*step*step + 10_560*step - 89_500) / 100
		switch step {
		case 45, 205, 257, 282, 410, 707:
			cost--
		}
		if cost > 0 {
			total += cost
		}
	}
	return uint32(total)
}

// restoreCharacterStats puts both mutated ranges back and reports the failure
// that caused the rollback. A rollback that cannot be written or verified is
// reported instead, so a partially mutated snapshot is never presented as
// unchanged.
func restoreCharacterStats(
	snapshot *codec,
	characterID int,
	blockAt int64,
	blockBefore []byte,
	summaryLevelAt int64,
	summaryLevelBefore []byte,
	failure string,
) error {
	if err := errors.Join(
		snapshot.writeAt(blockAt, blockBefore),
		snapshot.writeAt(summaryLevelAt, summaryLevelBefore),
	); err != nil {
		return fmt.Errorf("%s and the prior statistics could not be restored: %w", failure, err)
	}

	blockRestored, blockErr := snapshot.readAt(blockAt, len(blockBefore))
	summaryRestored, summaryErr := snapshot.readAt(summaryLevelAt, len(summaryLevelBefore))
	if blockErr != nil || summaryErr != nil ||
		!bytes.Equal(blockRestored, blockBefore) ||
		!bytes.Equal(summaryRestored, summaryLevelBefore) {
		return fmt.Errorf("%s and the rollback of character %d could not be verified",
			failure, characterID)
	}
	return fmt.Errorf("%s; the save is unchanged", failure)
}
