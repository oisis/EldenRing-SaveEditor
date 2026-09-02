package saveengine

import (
	"encoding/binary"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// Slot-data layout of the confirmed Storage Box section, shared by PC and PS4.
// The section has no fixed position inside a slot: it sits behind the face-data
// block, which sits behind EquipPhysicsData and the equipped-armaments block,
// which themselves sit behind the variable-length acquired-projectiles section.
// Everything in front of Storage therefore depends on how many projectiles the
// character has acquired, so the section is located through the confirmed anchor
// of that one slot and walked forwards from it across the one dynamic length the
// save itself declares. This getter owns its own anchor, its own layout
// constants and its own bounds checks; it borrows no position, helper or parsing
// function from another getter.
const (
	// StorageSectionCommon and StorageSectionKey are the two physical sections
	// of the Storage Box and the only accepted containerSection values besides
	// the empty string, which means both sections.
	StorageSectionCommon = "common"
	StorageSectionKey    = "key"

	// StorageDefaultPageSize is the page size used when the caller passes 0. Like
	// the other paging getters this one has no maximum page size.
	StorageDefaultPageSize = 50

	// storageProjectileCountOffset is the distance from the anchor to the uint32
	// that declares how many acquired-projectile records follow it. It is the sum
	// of the confirmed fixed structures between the two positions:
	//
	//	0x00D0 SpEffect
	//	0x0058 EquipedItemIndex
	//	0x001C ActiveEquipedItems
	//	0x0058 EquipedItemsID
	//	0x0058 ActiveEquipedItemsGa
	//	0x9011 InventoryHeld
	//	0x0074 EquippedSpells
	//	0x008C EquipItemData
	//	0x0018 EquippedGestures
	//
	// Every one of them has a fixed size, so this distance is constant; only the
	// projectile section behind it varies.
	storageProjectileCountOffset = 0x931D

	// storageProjectileRecordSize is the stride of one acquired-projectile
	// record, and storageMaxProjectileRecords is the highest count accepted
	// before the declared length is treated as corrupt instead of followed. The
	// limit is far above the counts native saves carry and far below what would
	// let a declared length wrap or reach past the container.
	storageProjectileRecordSize = 8
	storageMaxProjectileRecords = 200000

	// storageBlocksBeforeSection is the distance from the end of the projectile
	// records to the first byte of the Storage Box. It is the sum of the three
	// confirmed fixed blocks in between:
	//
	//	0x009C EquipedArmaments
	//	0x000C EquipPhysicsData
	//	0x012F FaceData
	//
	// The Storage Box starts immediately behind the face data.
	storageBlocksBeforeSection = 0x9C + 0x0C + 0x12F

	// One physical record is a triple of GaItem handle, quantity and acquisition
	// index, each a little-endian uint32 — the same 12-byte record InventoryHeld
	// uses, confirmed independently for this section.
	storageRecordSize = 12

	// The two physical sections and the fields around them: the four-byte
	// non-empty count in front of the common records, the common records, the
	// four-byte key-item count, the key records and the two trailing counters,
	// NextEquipIndex and NextAcquisitionSortId.
	storageCountHeader      = 4
	storageCommonRecords    = 0x780
	storageKeyCountHeader   = 4
	storageKeyRecords       = 0x80
	storageTrailingCounters = 8

	storageCommonAt    = storageCountHeader
	storageCommonSize  = storageCommonRecords * storageRecordSize
	storageKeyAt       = storageCommonAt + storageCommonSize + storageKeyCountHeader
	storageKeySize     = storageKeyRecords * storageRecordSize
	storageSectionSize = storageKeyAt + storageKeySize + storageTrailingCounters

	// storageQuantityMask drops the high bit the game sets on a stored quantity.
	// The Storage record is the same 12-byte record InventoryHeld uses and the
	// bit carries the same meaning here, so it is not part of the count and is
	// masked off here and nowhere else.
	storageQuantityMask uint32 = 0x7FFFFFFF

	// The two native sentinels of an absent record. They are the only handles
	// treated as "no item"; every other stored handle is reported as written,
	// including one this stage cannot resolve.
	storageEmptyHandle   uint32 = 0x00000000
	storageInvalidHandle uint32 = 0xFFFFFFFF
)

// storageAnchor is the confirmed 65-byte marker this getter is measured from:
// one leading 0x00 byte, then four full repetitions of a 16-byte block made of
// 0xFF 0xFF 0xFF 0xFF followed by twelve 0x00 bytes. It is stated here for this
// getter alone, the way every other slot reader states the marker it depends on.
var storageAnchor = []byte{
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

// StorageRecord is one non-empty physical Storage Box record, plus the identity
// of the row it was read from.
//
// OwnedItemID is the opaque identity of this physical record, valid for the
// SaveRevision it was minted under and for nothing else. It is compared byte for
// byte and never parsed: it carries no handle, no acquisition index, no physical
// index and no slot address. A Storage record and an Inventory record at the
// same coordinates are two physical records and never share an identity.
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
type StorageRecord struct {
	OwnedItemID      string `json:"ownedItemID"`
	ContainerSection string `json:"containerSection"`
	PhysicalIndex    int    `json:"physicalIndex"`
	GaItemHandle     uint32 `json:"gaItemHandle"`
	Quantity         uint32 `json:"quantity"`
	AcquisitionIndex uint32 `json:"acquisitionIndex"`
}

// CharacterStorage is one page of the raw Storage Box records of one physical
// save slot. This is the first stage of the Storage surface: it carries no name,
// no kind, no key, no family, no variant, no capacity and no Inventory record,
// and it reads no GameCatalog at all. The only resolved value is the owned-item
// identity of each returned record.
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
// whose deleted character's storage is still in the file — reports Active false,
// an empty list and a zero total, mints no identity, and its slot data is never
// searched or read; it still reports the current SaveRevision.
type CharacterStorage struct {
	SaveSessionID string          `json:"saveSessionID"`
	SaveRevision  string          `json:"saveRevision"`
	CharacterID   int             `json:"characterID"`
	Active        bool            `json:"active"`
	Records       []StorageRecord `json:"records"`
	Total         int             `json:"total"`
	Page          int             `json:"page"`
	PageSize      int             `json:"pageSize"`
}

// GetStorage returns one page of the raw Storage Box records stored in one
// physical character slot of an existing session. Like the other character
// readers it reads the session's private snapshot through the codec only: it
// opens no file, writes nothing, changes no session and returns no snapshot
// byte. It calls no other getter and no endpoint.
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
// page and pageSize follow the paging convention of the other paging getters: a
// negative value is rejected, 0 means page 1 and 50 entries, there is no maximum
// page size, and a valid page beyond the total returns an empty, non-nil list
// with the real total.
//
// For an active slot the section is located dynamically: the confirmed anchor is
// searched inside that one slot, the projectile count is read at a fixed
// distance behind it, and the Storage Box starts behind the records that count
// declares plus the three fixed blocks in between. A missing anchor, a count
// above the accepted maximum and a section reaching past the end of the slot or
// of the snapshot are hard errors. There is no fallback position, no partial
// result and nothing is guessed.
func (engine *Engine) GetStorage(
	saveSessionID string,
	characterID int,
	containerSection string,
	page int,
	pageSize int,
) (CharacterStorage, error) {
	if saveSessionID == "" {
		return CharacterStorage{}, apperror.MissingField("saveSessionID")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterStorage{}, apperror.UnknownSaveSession(saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterStorage{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}
	switch containerSection {
	case "", StorageSectionCommon, StorageSectionKey:
	default:
		return CharacterStorage{}, fmt.Errorf(
			"containerSection must be %q, %q or empty; got %q",
			StorageSectionCommon, StorageSectionKey, containerSection)
	}
	if page < 0 {
		return CharacterStorage{}, fmt.Errorf("page must not be negative; got %d", page)
	}
	if pageSize < 0 {
		return CharacterStorage{}, fmt.Errorf("pageSize must not be negative; got %d", pageSize)
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = StorageDefaultPageSize
	}

	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return CharacterStorage{}, fmt.Errorf("cannot read activity of character %d: %w", characterID, err)
	}

	storage := CharacterStorage{
		SaveSessionID: saveSessionID,
		SaveRevision:  loaded.session.revisionString(),
		CharacterID:   characterID,
		Records:       []StorageRecord{},
		Page:          page,
		PageSize:      pageSize,
	}
	if flag[0] != userData10ActiveFlagValue {
		// An inactive slot is reported from its flag alone, so the residual
		// storage of a deleted character is never located or decoded.
		return storage, nil
	}

	records, err := readStorageRecords(loaded, characterID)
	if err != nil {
		return CharacterStorage{}, err
	}

	matches := records
	if containerSection != "" {
		matches = make([]StorageRecord, 0, len(records))
		for _, record := range records {
			if record.ContainerSection == containerSection {
				matches = append(matches, record)
			}
		}
	}

	storage.Active = true
	storage.Total = len(matches)
	storage.Records = storagePage(matches, page, pageSize)
	return storage, nil
}

// readStorageRecords locates the Storage Box of one active slot across the one
// dynamic length the save itself declares, decodes both of its physical sections
// and returns every non-empty record with its owned-item identity minted, in
// physical native order.
//
// It is the single source of truth for the anchor, the projectile count, the
// bounds checks, the section layout, the sentinels, the quantity mask, the
// physical index and the minting. GetStorage adds only the section filter and
// the paging on top, so containerSection, page and pageSize can never influence
// which identity a record gets. Both physical sections are always decoded and
// identified here.
//
// The caller must already hold Engine.mutex and must have established that the
// slot is active: an inactive slot is answered from its activity flag alone and
// its data is never searched.
func readStorageRecords(loaded *loadedSave, characterID int) ([]StorageRecord, error) {
	sectionAt, err := storageBoxSectionAt(loaded, characterID)
	if err != nil {
		return nil, err
	}
	section, err := loaded.snapshot.readAt(sectionAt, storageSectionSize)
	if err != nil {
		return nil, fmt.Errorf("cannot read storage of character %d: %w", characterID, err)
	}

	records := appendStorageRecords(make([]StorageRecord, 0),
		section[storageCommonAt:storageCommonAt+storageCommonSize], StorageSectionCommon)
	records = appendStorageRecords(records,
		section[storageKeyAt:storageKeyAt+storageKeySize], StorageSectionKey)
	for index := range records {
		records[index].OwnedItemID = loaded.session.mintOwnedItemID(ownedItemLocator{
			characterID:      characterID,
			container:        ownedContainerStorage,
			containerSection: records[index].ContainerSection,
			physicalIndex:    records[index].PhysicalIndex,
		})
	}
	return records, nil
}

// storageBoxSectionAt walks one active slot across the one dynamic length the
// save itself declares and returns the absolute offset of the first byte of its
// Storage Box.
//
// It is the single source of truth for the anchor of this container, the
// projectile count and the bounds of the section behind them:
// readStorageRecords reads the section from here, and the quantity setter
// derives the offset of one record from the same value, so a reader and a writer
// can never disagree about where a row lives.
//
// The caller must already hold Engine.mutex and must have established that the
// slot is active.
func storageBoxSectionAt(loaded *loadedSave, characterID int) (int64, error) {
	base, slotEnd := storageSlotBounds(loaded.session.platform, characterID)

	anchor, err := loaded.snapshot.indexIn(base, slotEnd-base, storageAnchor)
	if err != nil {
		return 0, fmt.Errorf("cannot search the storage of character %d: %w", characterID, err)
	}
	if anchor < 0 {
		return 0, fmt.Errorf("character %d carries no storage anchor", characterID)
	}

	countAt := anchor + storageProjectileCountOffset
	if countAt+4 > slotEnd {
		return 0, fmt.Errorf("projectile count of character %d lies outside its slot", characterID)
	}
	rawCount, err := loaded.snapshot.readAt(countAt, 4)
	if err != nil {
		return 0, fmt.Errorf("cannot read projectile count of character %d: %w", characterID, err)
	}
	// The count is widened to int64 before it is multiplied, so a declared
	// length can never wrap into a small, seemingly valid offset.
	count := int64(binary.LittleEndian.Uint32(rawCount))
	if count > storageMaxProjectileRecords {
		return 0, fmt.Errorf(
			"character %d declares %d projectile records, want at most %d",
			characterID, count, storageMaxProjectileRecords)
	}

	sectionAt := countAt + 4 + count*storageProjectileRecordSize + storageBlocksBeforeSection
	if sectionAt+storageSectionSize > slotEnd {
		return 0, fmt.Errorf("storage of character %d does not fit into its slot", characterID)
	}
	return sectionAt, nil
}

// appendStorageRecords decodes one physical section and appends its non-empty
// records in stored order. Only the two native sentinels count as absent; every
// other handle is kept exactly as stored, so an unknown value stays visible
// instead of being dropped or reinterpreted.
func appendStorageRecords(
	into []StorageRecord,
	records []byte,
	containerSection string,
) []StorageRecord {
	for index := 0; index*storageRecordSize < len(records); index++ {
		record := records[index*storageRecordSize:]
		handle := binary.LittleEndian.Uint32(record)
		if handle == storageEmptyHandle || handle == storageInvalidHandle {
			continue
		}
		into = append(into, StorageRecord{
			ContainerSection: containerSection,
			PhysicalIndex:    index,
			GaItemHandle:     handle,
			Quantity:         binary.LittleEndian.Uint32(record[4:]) & storageQuantityMask,
			AcquisitionIndex: binary.LittleEndian.Uint32(record[8:]),
		})
	}
	return into
}

// storagePage cuts one page out of the matches without changing their order. A
// page beyond the last one is a normal result: it returns an empty, non-nil
// list. The first index is derived by division instead of multiplication, so a
// large page never overflows before it is compared with the match count.
func storagePage(matches []StorageRecord, page, pageSize int) []StorageRecord {
	total := len(matches)
	if total == 0 || page-1 > (total-1)/pageSize {
		return []StorageRecord{}
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	return matches[start:end]
}

// storageSlotBounds selects the platform entry point of this getter. PC and PS4
// differ in the container only, so the platform files supply the bounds of the
// slot data and everything inside it is decoded identically.
func storageSlotBounds(platform Platform, characterID int) (int64, int64) {
	if platform == PlatformPS4 {
		return ps4StorageSlotBounds(characterID)
	}
	return pcStorageSlotBounds(characterID)
}

// nextStorageEffectiveBucket derives the effective bucket for Storage index
// allocation given a stored NextAcquisitionSortId bucket counter and the highest
// occupied bucket among retained or existing records.
func nextStorageEffectiveBucket(storedNextBucket uint32, maxOccupiedBucket uint64, hasOccupied bool) uint64 {
	effectiveBucket := uint64(storedNextBucket)
	if hasOccupied && maxOccupiedBucket+1 > effectiveBucket {
		effectiveBucket = maxOccupiedBucket + 1
	}
	if effectiveBucket < 1 {
		effectiveBucket = 1
	}
	return effectiveBucket
}

// nextStorageAcquisitionAndCounters derives the assigned acquisition index and
// the updated NextAcquisitionSortId for one deposit into common Storage.
//
// Native Storage indices are even with stride 2 starting at 2 (bucket 1).
// NextAcquisitionSortId holds the next free bucket (Index/2 + 1). NextEquipIndex
// is not derived here: it follows the physical layout, not the allocator, and is
// owned by storageNextEquipIndex.
func nextStorageAcquisitionAndCounters(
	storedNextAcquisition uint32,
	records []StorageRecord,
	characterID int,
) (assignedIndex uint32, newNextAcquisition uint32, err error) {
	var maxBucket uint64
	var hasBucket bool
	for _, record := range records {
		if record.ContainerSection != StorageSectionCommon || record.AcquisitionIndex >= 50000 {
			continue
		}
		bucket := uint64(record.AcquisitionIndex) / 2
		if !hasBucket || bucket > maxBucket {
			maxBucket = bucket
			hasBucket = true
		}
	}

	effectiveBucket := nextStorageEffectiveBucket(storedNextAcquisition, maxBucket, hasBucket)

	assigned := effectiveBucket * 2
	if assigned >= uint64(^uint32(0)) {
		return 0, 0, fmt.Errorf(
			"Storage acquisition index of character %d would overflow uint32", characterID)
	}

	nextAcq := effectiveBucket + 1
	if nextAcq > uint64(^uint32(0)) {
		return 0, 0, fmt.Errorf(
			"Storage NextAcquisitionSortId of character %d cannot be advanced", characterID)
	}

	return uint32(assigned), uint32(nextAcq), nil
}

// storageNextEquipIndex derives the Storage NextEquipIndex the game keeps after
// one record has been created in the common section: 128 plus the highest
// physically occupied row of that section, counted from 0.
//
// The counter follows the physical layout, never the record count and never the
// stored value: deleting a record zeroes it in place without closing the table,
// so holes are a normal state. A new record landing in a hole below the highest
// occupied row therefore leaves the counter unchanged, while a record extending
// the table raises it by one.
//
// common is the common section as it will look after the planned insertion, and
// insertedRow is the row the new record occupies. Passing both makes the result
// independent of whether the caller holds the pre-image (AddItemToStorage writes
// one row into it) or the already rotated image (MoveOwnedItemToStorage builds
// it), because insertedRow is occupied in either case.
//
// The result cannot overflow: the section holds storageCommonRecords rows, so
// the highest value is 128 + storageCommonRecords - 1.
func storageNextEquipIndex(common []byte, insertedRow int) uint32 {
	highest := insertedRow
	for row := storageCommonRecords - 1; row > highest; row-- {
		handle := binary.LittleEndian.Uint32(common[row*storageRecordSize:])
		if handle != storageEmptyHandle && handle != storageInvalidHandle {
			highest = row
			break
		}
	}
	return uint32(128 + highest)
}
