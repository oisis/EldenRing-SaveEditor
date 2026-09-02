package saveengine

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// The scopes one validation pass is divided into. Each of them is read and
// judged independently, so a scope whose data cannot be decoded never hides the
// findings of the others.
const (
	ValidationScopeInventory = "inventory"
	ValidationScopeStorage   = "storage"
	ValidationScopeStats     = "stats"
	ValidationScopeEquipment = "equipment"
	ValidationScopeSpells    = "spells"
)

// ValidationItemRecord is one stored container record together with the result
// of resolving its GaItem handle.
//
// The resolution is deliberately lenient, which is the one difference to
// ResolveGaItemIDs: a handle that cannot be resolved is reported as an
// unresolved record with the reason it failed, instead of failing the whole
// read. A report exists to make exactly that record visible; a fail-closed
// batch would replace it with no report at all.
//
// GameID is meaningful only when Resolved is true. Nothing is guessed for an
// unresolved record: no game ID, no substitute item and no repair.
type ValidationItemRecord struct {
	Container        string `json:"container"`
	ContainerSection string `json:"containerSection"`
	PhysicalIndex    int    `json:"physicalIndex"`
	OwnedItemID      string `json:"ownedItemID"`
	GaItemHandle     uint32 `json:"gaItemHandle"`
	Quantity         uint32 `json:"quantity"`
	GameID           uint32 `json:"gameID"`
	Resolved         bool   `json:"resolved"`
	ResolutionError  string `json:"resolutionError"`
}

// ValidationReference is one stored {GaItem handle, InventoryHeld common row}
// pair whose row does not carry the item the pair names. Only inconsistent
// pairs are collected; a pair that matches its row is not a finding.
type ValidationReference struct {
	Structure string `json:"structure"`
	Slot      int    `json:"slot"`
	Handle    uint32 `json:"handle"`
	Row       uint32 `json:"row"`
	Reason    string `json:"reason"`
}

// ValidationStats is the statistics block of one character next to the values
// the confirmed rules derive from it.
//
// The derivation happens here and nowhere else: ExpectedLevel,
// MinimumSoulMemory and ClassMinimumError come from the same functions
// SetCharacterStats enforces, so a report can never accept a state the setter
// rejects or reject a state the setter accepts. MinimumSoulMemory is therefore
// counted from the base level of the character's own starting class, and stays
// zero when that class is unknown.
//
// LevelError and ClassMinimumError hold the reason a rule could not be
// satisfied, empty when it was. ExpectedLevel is meaningful only when LevelError
// is empty.
type ValidationStats struct {
	Attributes        CharacterAttributes `json:"attributes"`
	StartingClassID   uint8               `json:"startingClassID"`
	StoredLevel       uint32              `json:"storedLevel"`
	ExpectedLevel     uint32              `json:"expectedLevel"`
	LevelError        string              `json:"levelError"`
	StoredSoulMemory  uint32              `json:"storedSoulMemory"`
	MinimumSoulMemory uint32              `json:"minimumSoulMemory"`
	ClassMinimumError string              `json:"classMinimumError"`
}

// SaveValidationFacts is everything one non-mutating validation pass reads from
// a single character slot, gathered under one lock and one save revision.
//
// Every scope carries its own failure string. An empty failure means the scope
// was decoded and its facts are complete; a non-empty one names why the scope
// could not be read, and the facts of that scope are then absent rather than
// partial. Unreadable data therefore becomes a reported gap in coverage, never
// a silently clean result and never a whole-report error.
type SaveValidationFacts struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	Active        bool   `json:"active"`

	Items            []ValidationItemRecord `json:"items"`
	InventoryFailure string                 `json:"inventoryFailure"`
	StorageFailure   string                 `json:"storageFailure"`

	Stats        ValidationStats `json:"stats"`
	StatsFailure string          `json:"statsFailure"`

	DanglingReferences []ValidationReference `json:"danglingReferences"`
	EquipmentFailure   string                `json:"equipmentFailure"`

	Spells               [equippedSpellSlotCount]uint32 `json:"spells"`
	AvailableMemorySlots int                            `json:"availableMemorySlots"`
	SpellsFailure        string                         `json:"spellsFailure"`
}

// GetSaveValidationFacts reads every fact a validation report is built from for
// one physical character slot of an existing session.
//
// It is strictly non-mutating: it takes one lock, reads the session's private
// snapshot through the codec only, and writes no byte, no session field, no
// revision and no undo point. It opens no file and returns no raw snapshot byte.
//
// The whole pass runs under one lock so every scope describes the same save
// revision. A concurrent mutation can therefore not land between two scopes and
// leave a report that mixes two states.
//
// saveSessionID is matched exactly. characterID is the slot index 0..9; an
// inactive or residual slot is a normal result with no facts, not an error, and
// its slot data is never searched or read.
//
// This method judges nothing that needs GameCatalog. Naming an item, stating a
// container limit and costing a spell belong to the endpoint above it; locating,
// decoding and applying the save-side rules belong here.
func (engine *Engine) GetSaveValidationFacts(saveSessionID string, characterID int) (SaveValidationFacts, error) {
	if saveSessionID == "" {
		return SaveValidationFacts{}, apperror.MissingField("saveSessionID")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return SaveValidationFacts{}, apperror.UnknownSaveSession(saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return SaveValidationFacts{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	facts := SaveValidationFacts{
		SaveSessionID:      saveSessionID,
		SaveRevision:       loaded.session.revisionString(),
		CharacterID:        characterID,
		Items:              []ValidationItemRecord{},
		DanglingReferences: []ValidationReference{},
	}
	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return SaveValidationFacts{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}
	if flag[0] != userData10ActiveFlagValue {
		// An inactive slot is reported from its flag alone, so the residual data
		// of a deleted character is never located, decoded or judged.
		return facts, nil
	}
	facts.Active = true

	// The GaItem table is read once for the whole pass. A table that cannot be
	// parsed leaves both container scopes uncovered rather than turning every
	// record into an unresolved finding.
	byHandle, gaItemErr := readGaItemMap(loaded.snapshot, loaded.session.platform, characterID)

	inventory, err := readInventoryRecords(loaded, characterID)
	switch {
	case err != nil:
		facts.InventoryFailure = err.Error()
	case gaItemErr != nil:
		facts.InventoryFailure = gaItemErr.Error()
	default:
		for _, record := range inventory {
			facts.Items = append(facts.Items, validationItemRecord(
				byHandle, ownedContainerInventory, record.ContainerSection,
				record.PhysicalIndex, record.OwnedItemID, record.GaItemHandle, record.Quantity))
		}
	}

	storage, err := readStorageRecords(loaded, characterID)
	switch {
	case err != nil:
		facts.StorageFailure = err.Error()
	case gaItemErr != nil:
		facts.StorageFailure = gaItemErr.Error()
	default:
		for _, record := range storage {
			facts.Items = append(facts.Items, validationItemRecord(
				byHandle, ownedContainerStorage, record.ContainerSection,
				record.PhysicalIndex, record.OwnedItemID, record.GaItemHandle, record.Quantity))
		}
	}

	stats, err := readValidationStats(loaded, characterID)
	if err != nil {
		facts.StatsFailure = err.Error()
	} else {
		facts.Stats = stats
	}

	if facts.InventoryFailure != "" {
		// The reference rows are indices into InventoryHeld, so without the
		// decoded container there is nothing to compare them against.
		facts.EquipmentFailure = facts.InventoryFailure
	} else if references, err := ownedItemRowReferences(loaded, characterID); err != nil {
		facts.EquipmentFailure = err.Error()
	} else {
		facts.DanglingReferences = danglingReferences(references, inventory)
	}

	spells, available, err := readValidationSpells(loaded, characterID)
	if err != nil {
		facts.SpellsFailure = err.Error()
	} else {
		facts.Spells = spells
		facts.AvailableMemorySlots = available
	}
	return facts, nil
}

// validationItemRecord resolves one stored handle leniently and returns the
// record a report describes it with.
func validationItemRecord(
	byHandle map[uint32]uint32,
	container, containerSection string,
	physicalIndex int,
	ownedItemID string,
	handle, quantity uint32,
) ValidationItemRecord {
	record := ValidationItemRecord{
		Container:        container,
		ContainerSection: containerSection,
		PhysicalIndex:    physicalIndex,
		OwnedItemID:      ownedItemID,
		GaItemHandle:     handle,
		Quantity:         quantity,
	}
	gameID, err := resolveGaItemHandle(byHandle, handle)
	if err != nil {
		record.ResolutionError = err.Error()
		return record
	}
	record.GameID = gameID
	record.Resolved = true
	return record
}

// danglingReferences keeps the stored {handle, row} pairs whose row does not
// carry the item the pair names.
//
// The accepted shapes are the ones RemoveOwnedItem already relies on: the
// invalid-row sentinel and every row below the InventoryHeld base mean the slot
// references nothing, and a row that matches a non-empty common record carrying
// the same handle is a consistent reference. Everything else is reported.
func danglingReferences(references []ownedItemReference, inventory []InventoryRecord) []ValidationReference {
	byRow := make(map[int]uint32, len(inventory))
	for _, record := range inventory {
		if record.ContainerSection == InventorySectionCommon {
			byRow[record.PhysicalIndex] = record.GaItemHandle
		}
	}

	dangling := make([]ValidationReference, 0)
	for _, reference := range references {
		if reference.row == removeReferenceInvalidRow || reference.row < removeReferenceInventoryRowBase {
			continue
		}
		index := int(reference.row - removeReferenceInventoryRowBase)
		handle, occupied := byRow[index]
		switch {
		case !occupied:
			dangling = append(dangling, ValidationReference{
				Structure: reference.structure, Slot: reference.slot,
				Handle: reference.handle, Row: reference.row,
				Reason: fmt.Sprintf("inventory common row %d is empty", index),
			})
		case handle != reference.handle:
			dangling = append(dangling, ValidationReference{
				Structure: reference.structure, Slot: reference.slot,
				Handle: reference.handle, Row: reference.row,
				Reason: fmt.Sprintf(
					"inventory common row %d carries the different handle 0x%08X", index, handle),
			})
		}
	}
	return dangling
}

// readValidationStats reads the eight attributes, the stored level, the stored
// lifetime runes and the starting class of one active slot, and derives the
// values the confirmed rules expect from them.
//
// The caller must already hold Engine.mutex and must have established that the
// slot is active.
func readValidationStats(loaded *loadedSave, characterID int) (ValidationStats, error) {
	anchor, err := findStatsAnchor(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return ValidationStats{}, err
	}

	var values [characterAttributeCount]uint32
	for index := range values {
		value, err := loaded.snapshot.uint32At(anchor + statsVigorOffset + int64(index)*4)
		if err != nil {
			return ValidationStats{}, fmt.Errorf(
				"cannot read statistics of character %d: %w", characterID, err)
		}
		values[index] = value
	}
	level, err := loaded.snapshot.uint32At(anchor + statsLevelOffset)
	if err != nil {
		return ValidationStats{}, fmt.Errorf("cannot read level of character %d: %w", characterID, err)
	}
	soulMemory, err := loaded.snapshot.uint32At(anchor + statsTotalGetSoulOffset)
	if err != nil {
		return ValidationStats{}, fmt.Errorf(
			"cannot read lifetime runes of character %d: %w", characterID, err)
	}
	rawClass, err := loaded.snapshot.readAt(anchor+statsClassOffset, 1)
	if err != nil {
		return ValidationStats{}, fmt.Errorf(
			"cannot read starting class of character %d: %w", characterID, err)
	}

	stats := ValidationStats{
		Attributes: CharacterAttributes{
			Vigor: values[0], Mind: values[1], Endurance: values[2], Strength: values[3],
			Dexterity: values[4], Intelligence: values[5], Faith: values[6], Arcane: values[7],
		},
		StartingClassID:  rawClass[0],
		StoredLevel:      level,
		StoredSoulMemory: soulMemory,
	}
	definition, classErr := startingClass(rawClass[0])
	if expected, err := recalculateCharacterLevel(values); err != nil {
		stats.LevelError = err.Error()
	} else {
		stats.ExpectedLevel = expected
		// An unknown class has no confirmed base level, so no minimum can be
		// derived for it. The gap is reported through ClassMinimumError below
		// instead of being guessed from level 1, which would invent a floor the
		// character may never have owed.
		if classErr == nil {
			stats.MinimumSoulMemory = minimumSoulMemoryForLevel(expected, definition.level)
		}
	}
	if err := validateAgainstStartingClass(values, rawClass[0]); err != nil {
		stats.ClassMinimumError = err.Error()
	}
	return stats, nil
}

// readValidationSpells reads the fourteen physical spell records and the memory
// capacity of one active slot through the same readers GetEquippedSpells uses,
// so the report can never disagree with the getter about either value.
//
// The caller must already hold Engine.mutex and must have established that the
// slot is active.
func readValidationSpells(loaded *loadedSave, characterID int) ([equippedSpellSlotCount]uint32, int, error) {
	var records [equippedSpellSlotCount]uint32

	base := slotDataBase(loaded.session.platform, characterID)
	slotEnd := base + characterSlotDataSize

	anchor, err := loaded.snapshot.indexIn(base, characterSlotDataSize, equippedSpellsAnchor)
	if err != nil {
		return records, 0, fmt.Errorf(
			"cannot search the equipped spells of character %d: %w", characterID, err)
	}
	if anchor < 0 {
		return records, 0, fmt.Errorf("character %d carries no equipped-spells anchor", characterID)
	}

	records, err = readEquippedSpellRecords(loaded.snapshot, anchor, slotEnd, characterID)
	if err != nil {
		return records, 0, err
	}
	available, err := readAvailableMemorySlots(loaded.snapshot, anchor, base, slotEnd, characterID)
	if err != nil {
		return records, 0, err
	}
	return records, available, nil
}
