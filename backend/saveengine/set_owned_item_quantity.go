package saveengine

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
)

// This file holds the first mutation of SaveForge 2.0: it sets the quantity of
// one existing physical record addressed by its opaque OwnedItemID. It creates
// no record, removes none, merges none, moves none and reorders none. It changes
// exactly the four quantity bytes of the addressed record inside the session's
// private snapshot, and it never touches the file the session was loaded from —
// there is no WriteSave yet.
//
// It is an internal SaveEngine boundary, not an endpoint. The public
// SetOwnedItemQuantity endpoint is still contract-only.

// ownedItemQuantityFlag is the high bit the game keeps on a stored quantity. The
// two container readers mask it off with their own constant because it is not
// part of the count; the writer is the only place that has to preserve it, so it
// states the bit itself here. Its complement is the whole value range a quantity
// may occupy.
const ownedItemQuantityFlag uint32 = 0x80000000

// SetOwnedItemQuantityResult reports one committed quantity change.
//
// SaveRevision is the revision the change committed under, which is the previous
// one plus exactly 1. OwnedItemID is echoed back exactly as supplied — it is the
// identifier the mutation was performed with and is already stale by the time
// this result exists, because the commit retired every identity of the previous
// revision. A caller that wants to address the record again reads it back
// through GetInventory or GetStorage under the new revision.
//
// Quantity is the masked value now stored in the record, which equals the
// requested quantity.
type SetOwnedItemQuantityResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	OwnedItemID   string `json:"ownedItemID"`
	CharacterID   int    `json:"characterID"`
	Quantity      uint32 `json:"quantity"`
}

// SetOwnedItemQuantity sets the stored quantity of the one physical record
// ownedItemID was minted for.
//
// saveSessionID is matched exactly, like everywhere else: it is never trimmed,
// normalised or guessed.
//
// expectedRevision is the revision the caller believes the session is at. It has
// to be a canonical decimal string — no sign, no prefix, no padding, no
// separator, no whitespace, and "0" is a valid value — and it is compared byte
// for byte against the session's current revision. A malformed value and a
// mismatched value are distinct errors, because the remedies differ, and the
// mismatch names the current revision so the caller can re-read without a second
// round trip. Neither changes a byte.
//
// expectedGameID is an anti-TOCTOU guard rather than an item selector. The
// endpoint above resolves the record against GameCatalog outside this lock in
// order to derive the two limits; this method re-resolves the addressed record's
// GaItem handle under the lock and rejects the request unless it still denotes
// exactly that game ID. The limits therefore always belong to the item they were
// read for.
//
// maxPerRecord and maxContainerTotal are enforced exactly as supplied and are
// never invented, defaulted, widened or clamped here. Choosing them — per stack,
// per inventory, per storage, Safe Mode or Chaos Mode — belongs to the endpoint
// that reads GameCatalog. A quantity above either limit is rejected; nothing is
// ever silently reduced to fit.
//
// quantity must be at least 1. Zero is an error: this method never removes a
// record, which is the job of a later RemoveOwnedItem.
//
// The whole mutation runs inside one critical section under the single
// process-wide Engine.mutex, taken exactly once. Every fallible check completes
// before the first byte changes, the write itself is four bytes wide and
// verified, and a failed verification restores the exact previous bytes and
// reports an error without advancing the revision, marking the session dirty or
// retiring an identity.
func (engine *Engine) SetOwnedItemQuantity(
	saveSessionID string,
	characterID int,
	ownedItemID string,
	quantity uint32,
	expectedRevision string,
	expectedGameID uint32,
	maxPerRecord uint32,
	maxContainerTotal uint32,
) (SetOwnedItemQuantityResult, error) {
	// These four arguments are checked before the session is touched because they
	// depend on nothing but themselves.
	if quantity == 0 {
		return SetOwnedItemQuantityResult{}, errors.New(
			"quantity must be at least 1; removing a record is a separate operation")
	}
	if quantity > ^ownedItemQuantityFlag {
		return SetOwnedItemQuantityResult{}, fmt.Errorf(
			"quantity %d exceeds the %d the record can store", quantity, ^ownedItemQuantityFlag)
	}
	if maxPerRecord == 0 || maxContainerTotal == 0 {
		return SetOwnedItemQuantityResult{}, fmt.Errorf(
			"maxPerRecord and maxContainerTotal must both be at least 1; got %d and %d",
			maxPerRecord, maxContainerTotal)
	}
	if quantity > maxPerRecord {
		return SetOwnedItemQuantityResult{}, fmt.Errorf(
			"quantity %d exceeds the limit of %d per record", quantity, maxPerRecord)
	}
	if !isCanonicalRevision(expectedRevision) {
		return SetOwnedItemQuantityResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}

	saveRevision, err := engine.commitRevision(saveSessionID, func(loaded *loadedSave) error {
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

		locator, err := loaded.session.resolveOwnedItemID(characterID, ownedItemID)
		if err != nil {
			return err
		}
		return writeOwnedItemQuantity(
			loaded, locator, ownedItemID, quantity, expectedGameID, maxContainerTotal)
	})
	if err != nil {
		return SetOwnedItemQuantityResult{}, err
	}

	return SetOwnedItemQuantityResult{
		SaveSessionID: saveSessionID,
		SaveRevision:  saveRevision,
		OwnedItemID:   ownedItemID,
		CharacterID:   characterID,
		Quantity:      quantity,
	}, nil
}

// writeOwnedItemQuantity performs the mutation itself: it reads exactly the
// container the token was minted in, proves that the addressed record is still
// the one the caller means, proves that the container total stays within its
// limit, and only then writes the four quantity bytes.
//
// Only the container recorded in the locator is read. There is no fallback into
// the other container, no second candidate record and no neighbouring row.
//
// Reading that container materialises its identities exactly as any getter does,
// and the token resolved a moment earlier can only exist because the same
// container was already read under this revision, so a rejected mutation leaves
// the registry as it found it.
//
// The caller must already hold Engine.mutex.
func writeOwnedItemQuantity(
	loaded *loadedSave,
	locator ownedItemLocator,
	ownedItemID string,
	quantity uint32,
	expectedGameID uint32,
	maxContainerTotal uint32,
) error {
	characterID := locator.characterID
	records, err := readOwnedRecords(loaded, characterID, locator.container)
	if err != nil {
		return err
	}

	target := -1
	for index, record := range records {
		if record.containerSection == locator.containerSection &&
			record.physicalIndex == locator.physicalIndex && record.ownedItemID == ownedItemID {
			target = index
			break
		}
	}
	if target < 0 {
		return fmt.Errorf("ownedItemID %q no longer addresses a record of character %d",
			ownedItemID, characterID)
	}

	byHandle, err := readGaItemMap(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return fmt.Errorf("cannot resolve items of character %d: %w", characterID, err)
	}
	gameID, err := resolveGaItemHandle(byHandle, records[target].gaItemHandle)
	if err != nil {
		return fmt.Errorf("ownedItemID %q: %w", ownedItemID, err)
	}
	if gameID != expectedGameID {
		return fmt.Errorf(
			"ownedItemID %q now denotes item 0x%08X, not the expected 0x%08X",
			ownedItemID, gameID, expectedGameID)
	}

	// The limit covers the whole physical container — both of its sections —
	// because the game counts what a character holds there, not what one row
	// holds. Nothing is merged, deduplicated, moved or reindexed to satisfy it:
	// the request is simply rejected when the container would end up over the
	// limit. Records are compared by resolved game ID rather than by raw handle,
	// so two rows of one item never escape the sum by carrying different handles.
	total := uint64(quantity)
	for index, record := range records {
		if index == target {
			continue
		}
		otherID, err := resolveGaItemHandle(byHandle, record.gaItemHandle)
		if err != nil {
			return fmt.Errorf("%s record %d of character %d: %w",
				locator.container, record.physicalIndex, characterID, err)
		}
		if otherID != gameID {
			continue
		}
		// Both operands are widened to uint64, and every quantity is a masked
		// 31-bit value, so the sum cannot wrap.
		total += uint64(record.quantity)
	}
	if total > uint64(maxContainerTotal) {
		return fmt.Errorf(
			"quantity %d would raise the %s total of item 0x%08X to %d, above the limit of %d",
			quantity, locator.container, gameID, total, maxContainerTotal)
	}

	offset, err := ownedItemQuantityOffset(loaded, locator)
	if err != nil {
		return err
	}
	oldRaw, err := loaded.snapshot.uint32At(offset)
	if err != nil {
		return fmt.Errorf("cannot read the quantity of ownedItemID %q: %w", ownedItemID, err)
	}
	// The stored value is read raw rather than through a reader's masked quantity,
	// because the high bit is not part of the count and has to survive the write
	// exactly as the game left it: never set here, never cleared here.
	newRaw := (oldRaw & ownedItemQuantityFlag) | quantity

	if err := loaded.snapshot.writeAt(offset, littleEndianUint32(newRaw)); err != nil {
		return fmt.Errorf("cannot write the quantity of ownedItemID %q: %w", ownedItemID, err)
	}
	written, err := loaded.snapshot.uint32At(offset)
	if err == nil && written == newRaw {
		return nil
	}

	// The write is four bytes wide, so the rollback is the exact previous four
	// bytes rather than a copy of the snapshot.
	if rollback := loaded.snapshot.writeAt(offset, littleEndianUint32(oldRaw)); rollback != nil {
		return fmt.Errorf(
			"the quantity of ownedItemID %q could not be verified and could not be restored: %w",
			ownedItemID, rollback)
	}
	restored, restoreErr := loaded.snapshot.uint32At(offset)
	if restoreErr != nil || restored != oldRaw {
		return fmt.Errorf(
			"the quantity of ownedItemID %q could not be verified and its rollback could not be confirmed",
			ownedItemID)
	}
	return fmt.Errorf("the quantity of ownedItemID %q was not stored; the record is unchanged", ownedItemID)
}

// ownedRecord is the handful of record fields this mutation needs, taken from
// whichever container the locator names. It exists so the matching and the
// container total are written once instead of once per container; it is private
// to this file, carries no offset and never leaves the package.
type ownedRecord struct {
	ownedItemID      string
	containerSection string
	physicalIndex    int
	gaItemHandle     uint32
	quantity         uint32
}

// readOwnedRecords reads exactly one container of one character through the
// shared reader of that container, so the anchors, bounds, section sizes,
// sentinels, quantity mask, physical indexes and minting keep their single
// owner.
//
// The caller must already hold Engine.mutex.
func readOwnedRecords(loaded *loadedSave, characterID int, container string) ([]ownedRecord, error) {
	switch container {
	case ownedContainerInventory:
		records, err := readInventoryRecords(loaded, characterID)
		if err != nil {
			return nil, err
		}
		owned := make([]ownedRecord, len(records))
		for index, record := range records {
			owned[index] = ownedRecord{record.OwnedItemID, record.ContainerSection,
				record.PhysicalIndex, record.GaItemHandle, record.Quantity}
		}
		return owned, nil
	case ownedContainerStorage:
		records, err := readStorageRecords(loaded, characterID)
		if err != nil {
			return nil, err
		}
		owned := make([]ownedRecord, len(records))
		for index, record := range records {
			owned[index] = ownedRecord{record.OwnedItemID, record.ContainerSection,
				record.PhysicalIndex, record.GaItemHandle, record.Quantity}
		}
		return owned, nil
	default:
		return nil, fmt.Errorf("unknown container %q", container)
	}
}

// ownedItemQuantityOffset reports where the four quantity bytes of one record
// live. The start of the container comes from the same helper its reader uses,
// so the two can never disagree, and the row is addressed with the container's
// own record size and section distance.
//
// The offset stays inside this package: it is not a field of InventoryRecord, of
// StorageRecord, of OwnedItem or of any result.
//
// The caller must already hold Engine.mutex.
func ownedItemQuantityOffset(loaded *loadedSave, locator ownedItemLocator) (int64, error) {
	switch locator.container {
	case ownedContainerInventory:
		sectionAt, err := inventoryHeldSectionAt(loaded, locator.characterID)
		if err != nil {
			return 0, err
		}
		row := int64(0)
		if locator.containerSection == InventorySectionKey {
			row = inventoryHeldKeyAt
		}
		return sectionAt + row + int64(locator.physicalIndex)*inventoryHeldRecordSize + 4, nil
	case ownedContainerStorage:
		sectionAt, err := storageBoxSectionAt(loaded, locator.characterID)
		if err != nil {
			return 0, err
		}
		row := int64(storageCommonAt)
		if locator.containerSection == StorageSectionKey {
			row = storageKeyAt
		}
		return sectionAt + row + int64(locator.physicalIndex)*storageRecordSize + 4, nil
	default:
		return 0, fmt.Errorf("unknown container %q", locator.container)
	}
}

// littleEndianUint32 renders one value the way every record field is stored.
func littleEndianUint32(value uint32) []byte {
	raw := make([]byte, 4)
	binary.LittleEndian.PutUint32(raw, value)
	return raw
}

// isCanonicalRevision reports whether value is exactly what revisionString
// produces for some revision: a non-empty decimal rendering of a uint64 with no
// sign, no prefix, no padding, no separator and no whitespace. "0" is canonical;
// "", "00", "+1", " 1" and "1 " are not.
//
// Round-tripping through the same pair of functions revisionString uses is the
// definition of canonical, so the shape rule cannot drift from the renderer. No
// number leaves this check: the revision itself is only ever compared as a
// string, byte for byte.
func isCanonicalRevision(value string) bool {
	parsed, err := strconv.ParseUint(value, 10, 64)
	return err == nil && strconv.FormatUint(parsed, 10) == value
}
