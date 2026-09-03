package saveengine

import (
	"encoding/binary"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// Slot-data layout of the confirmed InventoryHeld section, shared by PC and PS4.
// The section has no fixed position inside a slot, so it is located through the
// confirmed anchor of the slot and read forwards from it. This getter owns its
// own anchor and its own layout constants; it borrows no position, helper or
// parsing function from another getter.
const (
	// InventorySectionCommon and InventorySectionKey are the two physical
	// sections of InventoryHeld and the only accepted containerSection values
	// besides the empty string, which means both sections.
	InventorySectionCommon = "common"
	InventorySectionKey    = "key"

	// InventoryDefaultPageSize is the page size used when the caller passes 0.
	// Like GetResources this getter has no maximum page size.
	InventoryDefaultPageSize = 50

	// InventoryHeldMaxRecords is the number of physical rows the two sections
	// hold together. A caller that has to filter over the whole container asks
	// for this page size instead of restating the physical capacities, which
	// stay private to this package.
	InventoryHeldMaxRecords = inventoryHeldCommonRecords + inventoryHeldKeyRecords

	// inventoryHeldCommonOffset is the distance from the anchor to the first
	// common record. The anchor is followed by the fixed structures in front of
	// InventoryHeld and by the four-byte common-item count, so the first record
	// itself starts here.
	inventoryHeldCommonOffset = 505

	// One physical record is a triple of GaItem handle, quantity and acquisition
	// index, each a little-endian uint32.
	inventoryHeldRecordSize = 12

	// The two physical sections and the fields between and behind them: the
	// common records, the four-byte key-item count, the key records and the two
	// trailing counters, NextEquipIndex and NextAcquisitionSortId.
	inventoryHeldCommonRecords    = 0xA80
	inventoryHeldKeyCountHeader   = 4
	inventoryHeldKeyRecords       = 0x180
	inventoryHeldTrailingCounters = 8

	inventoryHeldCommonSize  = inventoryHeldCommonRecords * inventoryHeldRecordSize
	inventoryHeldKeyAt       = inventoryHeldCommonSize + inventoryHeldKeyCountHeader
	inventoryHeldSectionSize = inventoryHeldKeyAt +
		inventoryHeldKeyRecords*inventoryHeldRecordSize +
		inventoryHeldTrailingCounters

	// inventoryHeldQuantityMask drops the high bit the game sets on a stored
	// quantity. It is not part of the count and is masked off here and nowhere
	// else.
	inventoryHeldQuantityMask uint32 = 0x7FFFFFFF

	// The two native sentinels of an absent record. They are the only handles
	// treated as "no item"; every other stored handle is reported as written,
	// including one this stage cannot resolve.
	inventoryHeldEmptyHandle   uint32 = 0x00000000
	inventoryHeldInvalidHandle uint32 = 0xFFFFFFFF
)

// inventoryHeldAnchor is the confirmed 65-byte marker this getter is measured
// from: one leading 0x00 byte, then four full repetitions of a 16-byte block
// made of 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes. It is stated here
// for this getter alone, the way every other slot reader states the marker it
// depends on.
var inventoryHeldAnchor = []byte{
	0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// InventoryRecord is one non-empty physical InventoryHeld record exactly as
// stored, plus the identity of the row it was read from.
//
// OwnedItemID is the opaque identity of this physical record, valid for the
// SaveRevision it was minted under and for nothing else. It is compared byte for
// byte and never parsed: it carries no handle, no acquisition index, no physical
// index and no slot address.
//
// ContainerSection is the physical section the record lives in and PhysicalIndex
// is its position inside that section, counted from 0. The pair identifies the
// row that was read; it is deliberately not a stable item identity, because a
// physical row moves when the game rewrites the section.
//
// GaItemHandle and AcquisitionIndex are the raw stored uint32 values: neither is
// normalised, masked, validated or resolved against a catalog. Quantity is the
// stored value with the high bit masked off, because that bit is not part of the
// count.
type InventoryRecord struct {
	OwnedItemID      string `json:"ownedItemID"`
	ContainerSection string `json:"containerSection"`
	PhysicalIndex    int    `json:"physicalIndex"`
	GaItemHandle     uint32 `json:"gaItemHandle"`
	Quantity         uint32 `json:"quantity"`
	AcquisitionIndex uint32 `json:"acquisitionIndex"`
}

// CharacterInventory is one page of the raw InventoryHeld records of one
// physical save slot. This is the first stage of the Inventory surface: it
// carries no name, no kind, no key, no family, no variant, no equipped state, no
// capacity and no Storage record, and it reads no GameCatalog at all. The only
// resolved value is the owned-item identity of each returned record.
//
// SaveRevision is the revision the result was read under, and the one every
// OwnedItemID in it was minted under. It is a decimal string compared exactly.
//
// Records keeps the physical native order: the common section first, then the
// key section, and inside each section the stored order of its rows. Only
// non-empty records are reported, so the two native absent sentinels never
// appear and never receive an identity. Total counts every non-empty record that
// passed the section filter, before paging.
//
// Every field except SaveSessionID, SaveRevision, CharacterID, Page and PageSize
// describes an active slot only. An inactive slot — including a residual one,
// whose deleted character's inventory is still in the file — reports Active
// false, an empty list and a zero total, mints no identity, and its slot data is
// never searched or read; it still reports the current SaveRevision.
type CharacterInventory struct {
	SaveSessionID string            `json:"saveSessionID"`
	SaveRevision  string            `json:"saveRevision"`
	CharacterID   int               `json:"characterID"`
	Active        bool              `json:"active"`
	Records       []InventoryRecord `json:"records"`
	Total         int               `json:"total"`
	Page          int               `json:"page"`
	PageSize      int               `json:"pageSize"`
}

// GetInventory returns one page of the raw InventoryHeld records stored in one
// physical character slot of an existing session. Like the other character
// readers it reads the session's private snapshot through the codec only: it
// opens no file, writes nothing, changes no session and returns no snapshot
// byte.
//
// saveSessionID is matched exactly. It is never trimmed, normalised or guessed,
// so an empty, unknown or already closed identifier is rejected instead of
// resolving to a session.
//
// characterID is the slot index 0..9. An inactive or residual slot is a normal
// result, not an error, and its slot data is never read.
//
// containerSection selects the physical section: the empty string means both,
// "common" and "key" mean that one section. The value is matched exactly and
// case-sensitively; it is never trimmed and has no alias, so anything else is
// rejected.
//
// page and pageSize follow the paging convention of GetResources: a negative
// value is rejected, 0 means page 1 and 50 entries, there is no maximum page
// size, and a valid page beyond the total returns an empty, non-nil list with
// the real total.
//
// For an active slot the section is located through the confirmed anchor of that
// one slot and read at a constant distance behind it. A missing anchor and a
// section reaching past the end of the slot or of the snapshot are hard errors.
// There is no fallback position, no partial result and nothing is guessed.
func (engine *Engine) GetInventory(
	saveSessionID string,
	characterID int,
	containerSection string,
	page int,
	pageSize int,
) (CharacterInventory, error) {
	if saveSessionID == "" {
		return CharacterInventory{}, apperror.MissingField("saveSessionID")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterInventory{}, apperror.UnknownSaveSession(saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterInventory{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}
	switch containerSection {
	case "", InventorySectionCommon, InventorySectionKey:
	default:
		return CharacterInventory{}, fmt.Errorf(
			"containerSection must be %q, %q or empty; got %q",
			InventorySectionCommon, InventorySectionKey, containerSection)
	}
	if page < 0 {
		return CharacterInventory{}, fmt.Errorf("page must not be negative; got %d", page)
	}
	if pageSize < 0 {
		return CharacterInventory{}, fmt.Errorf("pageSize must not be negative; got %d", pageSize)
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = InventoryDefaultPageSize
	}

	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterInventory{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}

	inventory := CharacterInventory{
		SaveSessionID: saveSessionID,
		SaveRevision:  loaded.session.revisionString(),
		CharacterID:   characterID,
		Records:       []InventoryRecord{},
		Page:          page,
		PageSize:      pageSize,
	}
	if flag[0] != userData10ActiveFlagValue {
		// An inactive slot is reported from its flag alone, so the residual
		// inventory of a deleted character is never located or decoded.
		return inventory, nil
	}

	records, err := readInventoryRecords(loaded, characterID)
	if err != nil {
		return CharacterInventory{}, err
	}

	matches := records
	if containerSection != "" {
		matches = make([]InventoryRecord, 0, len(records))
		for _, record := range records {
			if record.ContainerSection == containerSection {
				matches = append(matches, record)
			}
		}
	}

	inventory.Active = true
	inventory.Total = len(matches)
	inventory.Records = inventoryPage(matches, page, pageSize)
	return inventory, nil
}

// readInventoryRecords locates the InventoryHeld section of one active slot,
// decodes both of its physical sections and returns every non-empty record with
// its owned-item identity minted, in physical native order.
//
// It is the single source of truth for the anchor, the bounds checks, the
// section layout, the sentinels, the quantity mask, the physical index and the
// minting. GetInventory adds only the section filter and the paging on top, so
// containerSection, page and pageSize can never influence which identity a
// record gets. Both physical sections are always decoded and identified here.
//
// The caller must already hold Engine.mutex and must have established that the
// slot is active: an inactive slot is answered from its activity flag alone and
// its data is never searched.
func readInventoryRecords(loaded *loadedSave, characterID int) ([]InventoryRecord, error) {
	sectionAt, err := inventoryHeldSectionAt(loaded, characterID)
	if err != nil {
		return nil, err
	}
	section, err := loaded.snapshot.readAt(sectionAt, inventoryHeldSectionSize)
	if err != nil {
		return nil, fmt.Errorf("cannot read inventory of character %d: %w", characterID, err)
	}

	keyEnd := inventoryHeldSectionSize - inventoryHeldTrailingCounters
	records := appendInventoryRecords(
		make([]InventoryRecord, 0), section[:inventoryHeldCommonSize], InventorySectionCommon)
	records = appendInventoryRecords(records, section[inventoryHeldKeyAt:keyEnd], InventorySectionKey)
	for index := range records {
		records[index].OwnedItemID = loaded.session.mintOwnedItemID(ownedItemLocator{
			characterID:      characterID,
			container:        ownedContainerInventory,
			containerSection: records[index].ContainerSection,
			physicalIndex:    records[index].PhysicalIndex,
		})
	}
	return records, nil
}

// inventoryHeldSectionAt locates the InventoryHeld section of one active slot
// and returns the absolute offset of its first common record.
//
// It is the single source of truth for the anchor of this container and for the
// bounds of the section behind it: readInventoryRecords reads the section from
// here, and the quantity setter derives the offset of one record from the same
// value, so a reader and a writer can never disagree about where a row lives.
//
// The caller must already hold Engine.mutex and must have established that the
// slot is active.
func inventoryHeldSectionAt(loaded *loadedSave, characterID int) (int64, error) {
	base, slotEnd := inventorySlotBounds(loaded.session.platform, characterID)

	anchor, err := loaded.snapshot.indexIn(base, slotEnd-base, inventoryHeldAnchor)
	if err != nil {
		return 0, fmt.Errorf("cannot search the inventory of character %d: %w", characterID, err)
	}
	if anchor < 0 {
		return 0, fmt.Errorf("character %d carries no inventory anchor", characterID)
	}

	sectionAt := anchor + inventoryHeldCommonOffset
	if sectionAt+inventoryHeldSectionSize > slotEnd {
		return 0, fmt.Errorf("inventory of character %d does not fit into its slot", characterID)
	}
	return sectionAt, nil
}

// appendInventoryRecords decodes one physical section and appends its non-empty
// records in stored order. Only the two native sentinels count as absent; every
// other handle is kept exactly as stored, so an unknown value stays visible
// instead of being dropped or reinterpreted.
func appendInventoryRecords(
	into []InventoryRecord,
	records []byte,
	containerSection string,
) []InventoryRecord {
	for index := 0; index*inventoryHeldRecordSize < len(records); index++ {
		record := records[index*inventoryHeldRecordSize:]
		handle := binary.LittleEndian.Uint32(record)
		if handle == inventoryHeldEmptyHandle || handle == inventoryHeldInvalidHandle {
			continue
		}
		into = append(into, InventoryRecord{
			ContainerSection: containerSection,
			PhysicalIndex:    index,
			GaItemHandle:     handle,
			Quantity:         binary.LittleEndian.Uint32(record[4:]) & inventoryHeldQuantityMask,
			AcquisitionIndex: binary.LittleEndian.Uint32(record[8:]),
		})
	}
	return into
}

// inventoryPage cuts one page out of the matches without changing their order. A
// page beyond the last one is a normal result: it returns an empty, non-nil
// list. The first index is derived by division instead of multiplication, so a
// large page never overflows before it is compared with the match count.
func inventoryPage(matches []InventoryRecord, page, pageSize int) []InventoryRecord {
	total := len(matches)
	if total == 0 || page-1 > (total-1)/pageSize {
		return []InventoryRecord{}
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	return matches[start:end]
}

// inventorySlotBounds selects the platform entry point of this getter. PC and
// PS4 differ in the container only, so the platform files supply the bounds of
// the slot data and everything inside it is decoded identically.
func inventorySlotBounds(platform Platform, characterID int) (int64, int64) {
	if platform == PlatformPS4 {
		return ps4InventorySlotBounds(characterID)
	}
	return pcInventorySlotBounds(characterID)
}
