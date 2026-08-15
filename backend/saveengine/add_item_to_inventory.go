package saveengine

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// This file holds the third mutation of SaveForge 2.0: it adds one item to the
// common section of InventoryHeld. It moves no record, merges none, reorders
// none and reindexes none, it never writes the key section, the Storage Box,
// Equipment or any other structure, and it never allocates a record in the
// variable-length GaItem table. It changes the session's private snapshot only;
// persistence belongs to WriteSave.
//
// It is the internal SaveEngine boundary used by the public AddItemToInventory
// endpoint.
//
// Two shapes of add exist and they are decided by the caller, not here:
//
//   - A quantity stack tops up the first common record of the same item when one
//     exists and creates a new record otherwise. quantity is a delta: the stored
//     value becomes the existing value plus quantity, and nothing is ever
//     clamped to a limit.
//   - Separate instances always create a new record and always with quantity 1,
//     because each copy is its own physical record.
//
// What a genuinely new common record writes, and what a top-up deliberately does
// not:
//
//	record            handle, quantity and the acquisition index of the new row
//	common count      the four bytes in front of the first common record, plus 1
//	NextEquipIndex    the first trailing counter, plus 1
//	NextAcquisitionSortId  the second trailing counter, the new index plus 1
//	GaItemData        one active entry, only for an item the character does not
//	                  own a physical record of yet
//
// A top-up writes the four quantity bytes of the existing record and nothing
// else: the counters are allocators of the game rather than a population count,
// the section already holds the row, and the item already has its GaItemData
// entry. That is the contract SaveForge 1.5.8 and 1.6.8 shared byte for byte;
// only the meaning of the quantity argument differs, and that difference lives
// in this project's API rather than in the format.
//
// What it deliberately does not do:
//
//   - It never writes a key section. A key-routed item whose first record would
//     have to be created there is rejected by the endpoint above, and an item
//     that already holds an Inventory or Storage key record is rejected here,
//     so a second, wrongly routed copy can never appear in common.
//   - It never allocates a GaItem record, so weapons, armour and Ashes of War —
//     the families whose handle only exists as such a record — never reach this
//     mutation. Allocating one means repacking a variable-length section, which
//     is the retired SaveForge 1.x rebuild path.
//   - It never merges, moves or reindexes an existing row, and it never spills a
//     stack into a second record.

const (
	// addItemAcquisitionFloor is the lowest mark the acquisition allocator may
	// start from: the game reserves the indexes up to and including 432 for
	// equipment, and the mark itself sits two above that reservation.
	addItemAcquisitionFloor uint32 = 434

	// addItemMaxAcquisitionMark is the greatest stored mark for which parity
	// stabilisation, the assigned index and the following stored mark all fit in
	// uint32. A save above it cannot be advanced without wrapping.
	addItemMaxAcquisitionMark uint32 = ^uint32(0) - 3

	// The two trailing counters sit behind the key records, NextEquipIndex first.
	addItemNextEquipIndexOffset    = inventoryHeldSectionSize - inventoryHeldTrailingCounters
	addItemNextAcquisitionOffset   = inventoryHeldSectionSize - 4
	addItemCommonCountBackDistance = 4
)

// AddItemToInventoryResult reports one committed add.
//
// SaveRevision is the revision the add committed under, which is the previous
// one plus exactly 1.
//
// Added is the delta that was applied and Quantity is the value the record
// stores now: the two are equal for a new record and differ for a top-up.
// CreatedRecord tells the two apart without comparing them.
//
// ContainerSection and PhysicalIndex are the physical coordinates of the written
// row. The result deliberately carries no OwnedItemID: commitRevision retires
// every identity of the previous revision, so a token minted inside the mutation
// would be the only one alive under the new revision and the registry would be
// inconsistent with every other record. A caller that wants to address the row
// reads the container back under the new revision, exactly as it does after
// SetOwnedItemQuantity and RemoveOwnedItem, whose echoed tokens are stale for
// the same reason.
type AddItemToInventoryResult struct {
	SaveSessionID    string `json:"saveSessionID"`
	SaveRevision     string `json:"saveRevision"`
	CharacterID      int    `json:"characterID"`
	GameID           uint32 `json:"gameID"`
	Added            uint32 `json:"added"`
	Quantity         uint32 `json:"quantity"`
	CreatedRecord    bool   `json:"createdRecord"`
	ContainerSection string `json:"containerSection"`
	PhysicalIndex    int    `json:"physicalIndex"`
}

// AddItemToInventory adds quantity of gameID to the common section of the
// InventoryHeld section of one character.
//
// saveSessionID is matched exactly, like everywhere else: it is never trimmed,
// normalised or guessed.
//
// expectedRevision follows the same contract as every other mutation: it has to
// be a canonical decimal string — no sign, no prefix, no padding, no separator,
// no whitespace, and "0" is a valid value — and it is compared byte for byte
// against the session's current revision. A malformed value and a mismatched
// value are distinct errors, and the mismatch names the current revision so the
// caller can re-read without a second round trip. Neither changes a byte.
//
// gameID has to be one whose handle is derived from the ID itself. Everything
// else is rejected before a byte is read, because its handle only exists as a
// record in the variable-length GaItem table this mutation never allocates in.
//
// separateInstances selects the shape of the add and belongs to the endpoint
// that reads GameCatalog: true means every copy is its own record and quantity
// has to be exactly 1, false means the item stacks and quantity is a delta on
// the first common record of that item.
//
// maxPerRecord and maxContainerTotal are enforced exactly as supplied and are
// never invented, defaulted, widened or clamped here, exactly as
// SetOwnedItemQuantity enforces them. A resulting quantity above either limit is
// rejected; nothing is ever silently reduced to fit.
//
// The whole mutation runs inside one critical section under the single
// process-wide Engine.mutex, taken exactly once. Every fallible check completes
// before the first byte changes, every write of the plan is verified, and a
// write that cannot be verified restores the exact previous bytes of everything
// the plan changed and reports an error without advancing the revision, marking
// the session dirty or retiring an identity.
func (engine *Engine) AddItemToInventory(
	saveSessionID string,
	characterID int,
	gameID uint32,
	quantity uint32,
	expectedRevision string,
	separateInstances bool,
	maxPerRecord uint32,
	maxContainerTotal uint32,
) (AddItemToInventoryResult, error) {
	// These arguments are checked before the session is touched because they
	// depend on nothing but themselves.
	if quantity == 0 {
		return AddItemToInventoryResult{}, fmt.Errorf(
			"quantity must be at least 1; it is the amount added, not a target total")
	}
	if quantity > ^ownedItemQuantityFlag {
		return AddItemToInventoryResult{}, fmt.Errorf(
			"quantity %d exceeds the %d the record can store", quantity, ^ownedItemQuantityFlag)
	}
	if separateInstances && quantity != 1 {
		return AddItemToInventoryResult{}, fmt.Errorf(
			"item 0x%08X stores every copy in its own record, so quantity must be 1; got %d",
			gameID, quantity)
	}
	if maxPerRecord == 0 || maxContainerTotal == 0 {
		return AddItemToInventoryResult{}, fmt.Errorf(
			"maxPerRecord and maxContainerTotal must both be at least 1; got %d and %d",
			maxPerRecord, maxContainerTotal)
	}
	if quantity > maxPerRecord {
		return AddItemToInventoryResult{}, fmt.Errorf(
			"quantity %d exceeds the limit of %d per record", quantity, maxPerRecord)
	}
	if !isCanonicalRevision(expectedRevision) {
		return AddItemToInventoryResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	var outcome addedInventoryRecord
	saveRevision, err := engine.commitCharacterRevision(saveSessionID, opAddItemToInventory, characterID, func(loaded *loadedSave) error {
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

		var err error
		outcome, err = addItemToInventoryRecord(
			loaded, characterID, gameID, quantity, separateInstances, maxPerRecord, maxContainerTotal)
		return err
	})
	if err != nil {
		return AddItemToInventoryResult{}, err
	}

	return AddItemToInventoryResult{
		SaveSessionID:    saveSessionID,
		SaveRevision:     saveRevision,
		CharacterID:      characterID,
		GameID:           gameID,
		Added:            quantity,
		Quantity:         outcome.quantity,
		CreatedRecord:    outcome.created,
		ContainerSection: InventorySectionCommon,
		PhysicalIndex:    outcome.physicalIndex,
	}, nil
}

// addedInventoryRecord is what one committed add reports back about the row it
// wrote.
type addedInventoryRecord struct {
	physicalIndex int
	quantity      uint32
	created       bool
}

// addItemToInventoryRecord validates the complete plan and only then applies it.
// It reads both containers of the character through their own readers, so
// anchors, bounds, section sizes, sentinels, the quantity mask and the physical
// indexes keep their single owner.
//
// The caller must already hold Engine.mutex.
func addItemToInventoryRecord(
	loaded *loadedSave,
	characterID int,
	gameID uint32,
	quantity uint32,
	separateInstances bool,
	maxPerRecord uint32,
	maxContainerTotal uint32,
) (addedInventoryRecord, error) {
	// An item whose handle only exists as a GaItem record is refused first,
	// before anything is read: an operation this mutation has no write contract
	// for may not even look like it started.
	handle, err := gaItemHandleForGameID(gameID)
	if err != nil {
		return addedInventoryRecord{}, err
	}

	// An inactive slot — including a residual one, whose deleted character's
	// inventory is still in the file — is never written and never even located.
	flag, err := loaded.snapshot.readAt(
		userData10Base(loaded.session.platform)+userData10ActiveFlagsOffset+int64(characterID), 1)
	if err != nil {
		return addedInventoryRecord{}, fmt.Errorf(
			"cannot read activity of character %d: %w", characterID, err)
	}
	if flag[0] != userData10ActiveFlagValue {
		return addedInventoryRecord{}, fmt.Errorf(
			"character %d is not active and receives no item", characterID)
	}

	records, err := readInventoryRecords(loaded, characterID)
	if err != nil {
		return addedInventoryRecord{}, err
	}
	byHandle, err := readGaItemMap(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return addedInventoryRecord{}, fmt.Errorf(
			"cannot resolve items of character %d: %w", characterID, err)
	}

	// The container total covers both physical sections, because the game counts
	// what a character holds in InventoryHeld rather than what one row holds.
	// Records are compared by resolved game ID rather than by raw handle, so two
	// rows of one item never escape the sum by carrying different handles. An
	// unresolvable handle is a hard error: an add never proceeds past data this
	// engine cannot decode.
	target := -1
	total := uint64(quantity)
	for index, record := range records {
		recordID, err := resolveGaItemHandle(byHandle, record.GaItemHandle)
		if err != nil {
			return addedInventoryRecord{}, fmt.Errorf(
				"inventory record %d of character %d: %w", record.PhysicalIndex, characterID, err)
		}
		if recordID != gameID {
			continue
		}
		total += uint64(record.Quantity)
		if record.ContainerSection == InventorySectionKey {
			return addedInventoryRecord{}, fmt.Errorf(
				"item 0x%08X already holds a key record of character %d, and this mutation never"+
					" writes the key section", gameID, characterID)
		}
		if target < 0 && !separateInstances {
			target = index
		}
	}
	if total > uint64(maxContainerTotal) {
		return addedInventoryRecord{}, fmt.Errorf(
			"adding %d would raise the inventory total of item 0x%08X to %d, above the limit of %d",
			quantity, gameID, total, maxContainerTotal)
	}

	owned, err := ownsPhysicalRecord(loaded, characterID, records, handle, gameID)
	if err != nil {
		return addedInventoryRecord{}, err
	}
	if target >= 0 {
		return topUpInventoryRecord(
			loaded, characterID, records[target], gameID, quantity, maxPerRecord)
	}
	return createInventoryRecord(
		loaded, characterID, handle, gameID, quantity, owned)
}

// topUpInventoryRecord raises the quantity of the one existing common record of
// the item. It writes those four bytes and nothing else: no counter, no header
// and no GaItemData entry participates in a top-up.
//
// The caller must already hold Engine.mutex.
func topUpInventoryRecord(
	loaded *loadedSave,
	characterID int,
	record InventoryRecord,
	gameID uint32,
	quantity uint32,
	maxPerRecord uint32,
) (addedInventoryRecord, error) {
	// Both operands are widened, and a stored quantity is a masked 31-bit value,
	// so the sum cannot wrap before it is compared with the limit.
	stacked := uint64(record.Quantity) + uint64(quantity)
	if stacked > uint64(maxPerRecord) {
		return addedInventoryRecord{}, fmt.Errorf(
			"adding %d to the %d item 0x%08X already holds would store %d, above the limit of %d"+
				" per record", quantity, record.Quantity, gameID, stacked, maxPerRecord)
	}

	sectionAt, err := inventoryHeldSectionAt(loaded, characterID)
	if err != nil {
		return addedInventoryRecord{}, err
	}
	quantityAt := sectionAt + int64(record.PhysicalIndex)*inventoryHeldRecordSize + 4
	raw, err := loaded.snapshot.uint32At(quantityAt)
	if err != nil {
		return addedInventoryRecord{}, fmt.Errorf(
			"cannot read the quantity of inventory record %d of character %d: %w",
			record.PhysicalIndex, characterID, err)
	}
	// The stored value is read raw rather than through the reader's masked
	// quantity, because the high bit is not part of the count and has to survive
	// the write exactly as the game left it: never set here, never cleared here.
	updated := uint32(stacked)
	writes := []byteWrite{
		{at: quantityAt, data: littleEndianUint32((raw & ownedItemQuantityFlag) | updated)},
	}
	if err := applyByteWrites(loaded.snapshot, writes); err != nil {
		return addedInventoryRecord{}, fmt.Errorf("item 0x%08X: %w", gameID, err)
	}
	return addedInventoryRecord{
		physicalIndex: record.PhysicalIndex,
		quantity:      updated,
		created:       false,
	}, nil
}

// createInventoryRecord writes a genuinely new common record: the row itself,
// the common count in front of the section, both trailing allocators and, for an
// item the character owns no physical record of yet, one active GaItemData
// entry.
//
// The caller must already hold Engine.mutex.
func createInventoryRecord(
	loaded *loadedSave,
	characterID int,
	handle uint32,
	gameID uint32,
	quantity uint32,
	owned bool,
) (addedInventoryRecord, error) {
	result, writes, err := planInventoryRecordCreation(
		loaded, characterID, handle, gameID, quantity, owned)
	if err != nil {
		return addedInventoryRecord{}, err
	}
	if err := applyByteWrites(loaded.snapshot, writes); err != nil {
		return addedInventoryRecord{}, fmt.Errorf("item 0x%08X: %w", gameID, err)
	}
	return result, nil
}

// planInventoryRecordCreation validates and describes one new common record
// without changing the snapshot. It is shared by the public common-item add and
// domain mutations that must compose the same confirmed record write with
// additional state in one atomic byte plan.
//
// The caller must already hold Engine.mutex.
func planInventoryRecordCreation(
	loaded *loadedSave,
	characterID int,
	handle uint32,
	gameID uint32,
	quantity uint32,
	owned bool,
) (addedInventoryRecord, []byteWrite, error) {
	sectionAt, err := inventoryHeldSectionAt(loaded, characterID)
	if err != nil {
		return addedInventoryRecord{}, nil, err
	}

	row, err := firstFreeInventoryRow(loaded, sectionAt, characterID)
	if err != nil {
		return addedInventoryRecord{}, nil, err
	}

	commonCountAt := sectionAt - addItemCommonCountBackDistance
	commonCount, err := loaded.snapshot.uint32At(commonCountAt)
	if err != nil {
		return addedInventoryRecord{}, nil, fmt.Errorf(
			"cannot read the common item count of character %d: %w", characterID, err)
	}
	if commonCount >= inventoryHeldCommonRecords {
		return addedInventoryRecord{}, nil, fmt.Errorf(
			"the inventory of character %d declares %d of %d common records and receives no item",
			characterID, commonCount, inventoryHeldCommonRecords)
	}

	equipIndexAt := sectionAt + addItemNextEquipIndexOffset
	equipIndex, err := loaded.snapshot.uint32At(equipIndexAt)
	if err != nil {
		return addedInventoryRecord{}, nil, fmt.Errorf(
			"cannot read NextEquipIndex of character %d: %w", characterID, err)
	}
	if equipIndex == ^uint32(0) {
		return addedInventoryRecord{}, nil, fmt.Errorf(
			"NextEquipIndex of character %d cannot be advanced", characterID)
	}

	acquisitionAt := sectionAt + addItemNextAcquisitionOffset
	nextAcquisition, err := loaded.snapshot.uint32At(acquisitionAt)
	if err != nil {
		return addedInventoryRecord{}, nil, fmt.Errorf(
			"cannot read NextAcquisitionSortId of character %d: %w", characterID, err)
	}
	acquisitionIndex, err := nextAcquisitionIndex(nextAcquisition, characterID)
	if err != nil {
		return addedInventoryRecord{}, nil, err
	}

	// The GaItemData entry belongs to the first physical copy of an item ID only.
	// Ownership is read off the physical records of both containers, exactly as
	// SaveForge 1.5.8 and 1.6.8 read it, so a further copy of an already owned
	// item never creates a second entry.
	var gaItemData []byteWrite
	if !owned {
		gaItemData, err = planGaItemDataInsertion(loaded, characterID, gameID)
		if err != nil {
			return addedInventoryRecord{}, nil, err
		}
	}

	record := make([]byte, inventoryHeldRecordSize)
	copy(record, littleEndianUint32(handle))
	copy(record[4:], littleEndianUint32(quantity))
	copy(record[8:], littleEndianUint32(acquisitionIndex))

	writes := append([]byteWrite{
		{at: sectionAt + int64(row)*inventoryHeldRecordSize, data: record},
		{at: commonCountAt, data: littleEndianUint32(commonCount + 1)},
		{at: equipIndexAt, data: littleEndianUint32(equipIndex + 1)},
		{at: acquisitionAt, data: littleEndianUint32(acquisitionIndex + 1)},
	}, gaItemData...)
	return addedInventoryRecord{physicalIndex: row, quantity: quantity, created: true}, writes, nil
}

// firstFreeInventoryRow reports the lowest common row that carries one of the
// two native absent sentinels. The section is scanned in ascending physical
// order, so a new record always lands in the first gap the game left, and a full
// section is a rejection rather than an overwrite of an occupied row.
//
// The caller must already hold Engine.mutex.
func firstFreeInventoryRow(loaded *loadedSave, sectionAt int64, characterID int) (int, error) {
	common, err := loaded.snapshot.readAt(sectionAt, inventoryHeldCommonSize)
	if err != nil {
		return 0, fmt.Errorf("cannot read the inventory of character %d: %w", characterID, err)
	}
	for row := 0; row < inventoryHeldCommonRecords; row++ {
		handle := binary.LittleEndian.Uint32(common[row*inventoryHeldRecordSize:])
		if handle == inventoryHeldEmptyHandle || handle == inventoryHeldInvalidHandle {
			return row, nil
		}
	}
	return 0, fmt.Errorf(
		"the common inventory section of character %d holds no free record", characterID)
}

// ownsPhysicalRecord reports whether the character already holds a physical
// common-section record carrying handle, in InventoryHeld or the Storage Box.
// The caller has already rejected a matching Inventory key record. A matching
// Storage key record is rejected here rather than treated as ownership: this
// mutation may not use an unsupported location to suppress the GaItemData entry
// of the new common record. The Inventory records are the ones the caller has
// already read, so that container is not decoded twice.
//
// The caller must already hold Engine.mutex.
func ownsPhysicalRecord(
	loaded *loadedSave,
	characterID int,
	records []InventoryRecord,
	handle uint32,
	gameID uint32,
) (bool, error) {
	owned := false
	for _, record := range records {
		if record.GaItemHandle != handle {
			continue
		}
		owned = true
	}
	stored, err := readStorageRecords(loaded, characterID)
	if err != nil {
		return false, err
	}
	for _, record := range stored {
		if record.GaItemHandle != handle {
			continue
		}
		if record.ContainerSection == StorageSectionKey {
			return false, fmt.Errorf(
				"item 0x%08X already holds a storage key record of character %d, and this mutation"+
					" writes common records only", gameID, characterID)
		}
		owned = true
	}
	return owned, nil
}

// nextAcquisitionIndex derives the acquisition index of a new record from the
// stored allocator.
//
// NextAcquisitionSortId is a high-water mark rather than the index to assign:
// the game sorts by index >> 1, so the mark is stabilised to an even value and
// the new record takes the odd index one past it, which keeps consecutive adds
// in distinct sort buckets. A mark below the reserved equipment range is raised
// to the floor first. This is the rule SaveForge 1.5.8 and 1.6.8 shared byte for
// byte.
func nextAcquisitionIndex(stored uint32, characterID int) (uint32, error) {
	if stored > addItemMaxAcquisitionMark {
		return 0, fmt.Errorf(
			"NextAcquisitionSortId of character %d is %d and cannot be advanced",
			characterID, stored)
	}
	mark := stored
	if mark < addItemAcquisitionFloor {
		mark = addItemAcquisitionFloor
	}
	if mark%2 != 0 {
		mark++
	}
	return mark + 1, nil
}

// byteWrite is one range of a mutation plan: where it goes and what it stores.
// previous is filled in by applyByteWrites and is the rollback material of that
// one range.
type byteWrite struct {
	at       int64
	data     []byte
	previous []byte
}

// applyByteWrites performs a plan of non-overlapping ranges as one atomic step.
//
// It reads the previous bytes of every range before it writes the first one,
// verifies every range after the last one, and restores everything it managed to
// change — in reverse order, so the ranges come back the way they went out —
// when any write or any verification fails. A rejected write changes nothing at
// all, because the codec bounds-checks a whole range before its first byte.
//
// ponytail: a slice of ranges is the whole transaction machinery this mutation
// needs. Widen it only when a plan appears that ranges cannot express.
func applyByteWrites(snapshot *codec, writes []byteWrite) error {
	for index := range writes {
		previous, err := snapshot.readAt(writes[index].at, len(writes[index].data))
		if err != nil {
			return fmt.Errorf("cannot read the range the plan writes: %w", err)
		}
		writes[index].previous = previous
	}

	for index := range writes {
		if err := snapshot.writeAt(writes[index].at, writes[index].data); err != nil {
			return restoreByteWrites(snapshot, writes[:index],
				fmt.Sprintf("the plan could not be written: %v", err))
		}
	}
	for index := range writes {
		written, err := snapshot.readAt(writes[index].at, len(writes[index].data))
		if err != nil || !bytes.Equal(written, writes[index].data) {
			return restoreByteWrites(snapshot, writes, "the plan could not be verified")
		}
	}
	return nil
}

// restoreByteWrites puts the exact previous bytes of every applied range back
// and confirms every one of them, so a failed mutation either leaves nothing
// changed or says that it could not.
func restoreByteWrites(snapshot *codec, applied []byteWrite, reason string) error {
	for index := len(applied) - 1; index >= 0; index-- {
		if err := snapshot.writeAt(applied[index].at, applied[index].previous); err != nil {
			return fmt.Errorf("%s and could not be restored: %w", reason, err)
		}
	}
	for index := range applied {
		restored, err := snapshot.readAt(applied[index].at, len(applied[index].previous))
		if err != nil || !bytes.Equal(restored, applied[index].previous) {
			return fmt.Errorf("%s and its rollback could not be confirmed", reason)
		}
	}
	return fmt.Errorf("%s; the inventory is unchanged", reason)
}
