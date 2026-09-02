package saveengine

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// This file holds the second mutation of SaveForge 2.0: it removes the one
// existing physical record an opaque OwnedItemID was minted for. It creates no
// record, moves none, merges none, reorders none and reindexes none. It clears
// exactly the twelve bytes of the addressed record the way that record's own
// section is confirmed to be cleared, maintains the non-empty count of that
// section where the count is confirmed to move, and does all of it inside the
// session's private snapshot. It never writes a file itself; persistence belongs
// to WriteSave.
//
// It is the internal SaveEngine boundary used by the public RemoveOwnedItem
// endpoint.
//
// The four physical sections do not share one write, because they do not share
// one confirmed rule:
//
//   - InventoryHeld common: handle 0, quantity 0 and the physical row number in
//     the third field, and the common count of the section lowered by one.
//   - InventoryHeld key: the same three fields, and the key_count header left
//     exactly as it is. SaveForge 1.6.8 removed a key record without touching
//     that header, and the regression test protecting it
//     (backend/core/remove_key_item_test.go) is the only confirmed statement
//     about it this project has.
//   - Storage common: all twelve bytes zeroed, including the third field, and
//     the common count lowered by one.
//   - Storage key: rejected as unsupported. SaveForge 1.6.8 never wrote that
//     section, and its own specification (spec/10-storage.md) records its
//     semantics as unverified. A synthetic fixture is not evidence of a write
//     contract, so the removal fails closed instead of inventing one.
//
// What it deliberately does not touch:
//
//   - The GaItem table. Deleting a GaItem record means repacking a
//     variable-length section and shifting everything behind it, which is the
//     retired SaveForge 1.x rebuild path. Leaving the table untouched also means
//     no reference into it can dangle: a weapon that names an Ash of War handle,
//     and a handle a second record still uses, both keep resolving exactly as
//     before. The removal only requires that the handle of the record it is
//     about resolves at all, so undecodable data is never silently deleted.
//   - The trailing NextEquipIndex and NextAcquisitionSortId counters. They are
//     monotonic allocators of the game, not a population count, and SaveForge
//     1.5.8 and 1.6.8 left them alone on removal as well.
//   - Equipment, Quick Items and the Pouch themselves. An active reference is a
//     reason to reject the removal, never a reason to change the structure that
//     holds it: nothing is unequipped and nothing is cascaded here.

// The confirmed reference representation of one character slot, stated here for
// this mutation alone the way every other reader states the layout it depends
// on.
//
// Three blocks of equipmentSlotCount little-endian uint32 fields sit in front of
// InventoryHeld, each behind one leading byte of its serialized structure:
//
//	anchor + 0x00D0 + 1  EquipedItemIndex — physical inventory row + 0x180
//	anchor + 0x0144 + 1  EquipedItemsID   — the bare item ID
//	anchor + 0x019C + 1  ActiveEquipedItemsGa — the exact GaItem handle
//
// The third block ends on the four-byte common count of InventoryHeld, which is
// what inventoryHeldCommonOffset already measures, so the two distances below
// are derived from that one confirmed constant and cannot drift away from it.
//
// EquipItemData, behind InventoryHeld, holds the same reference as one
// interleaved pair per slot — the GaItem handle followed by the row field — for
// ten Quick Items and six Pouch slots. Its position, its slot counts and its
// record size are the ones quick_items.go and pouch_items.go already own, so
// this guard borrows those constants instead of restating them.
const (
	// removeEquipmentHandlesOffset is the distance from the anchor to the first
	// ActiveEquipedItemsGa field: the block ends where the common count begins.
	removeEquipmentHandlesOffset = inventoryHeldCommonOffset - 4 - equipmentSlotCount*4

	// removeEquipmentIndexesOffset is the distance from the anchor to the first
	// EquipedItemIndex field: the EquipedItemsID block and the 0x1C gap in front
	// of it separate the two.
	removeEquipmentIndexesOffset = removeEquipmentHandlesOffset - equipmentSlotCount*4 - 0x1C -
		equipmentSlotCount*4

	// removeReferenceInventoryRowBase is the base every row field is counted
	// from: the value is that base plus the physical row of the referenced record
	// in the common section of InventoryHeld.
	removeReferenceInventoryRowBase uint32 = 0x180

	// removeReferenceInvalidRow is the value a row field carries when it
	// references no row at all. Every other value below the base is equally
	// meaningless as a row and is skipped the same way.
	removeReferenceInvalidRow uint32 = 0xFFFFFFFF
)

// RemoveOwnedItemResult reports one committed removal.
//
// SaveRevision is the revision the removal committed under, which is the
// previous one plus exactly 1. OwnedItemID is echoed back exactly as supplied —
// it is the identifier the mutation was performed with and is already stale by
// the time this result exists, because the commit retired every identity of the
// previous revision, and because the record it addressed no longer exists.
//
// GameID is the save-side game ID the removed record's handle resolved to under
// the lock, so the caller learns what was removed without addressing the record
// again.
//
// The receipt the central commit path produced is embedded anonymously, so
// saveSessionID and saveRevision keep their previous JSON names and the three
// new members join them flat. Nothing here is reassembled from the kind, the
// session, the revision or a scope lookup.
type RemoveOwnedItemResult struct {
	MutationReceipt
	OwnedItemID string `json:"ownedItemID"`
	CharacterID int    `json:"characterID"`
	GameID      uint32 `json:"gameID"`
}

// RemoveOwnedItem removes the one physical record ownedItemID was minted for.
//
// saveSessionID is matched exactly, like everywhere else: it is never trimmed,
// normalised or guessed.
//
// ownedItemID is opaque and is never parsed. It is resolved through the identity
// registry only, so the record it names is the one physical row of the one
// character and the one container it was minted in. There is no lookup by game
// ID, no fallback into the other container and no second candidate row.
//
// expectedRevision follows the same contract as every other mutation: it has to
// be a canonical decimal string — no sign, no prefix, no padding, no separator,
// no whitespace, and "0" is a valid value — and it is compared byte for byte
// against the session's current revision. A malformed value and a mismatched
// value are distinct errors, and the mismatch names the current revision so the
// caller can re-read without a second round trip. Neither changes a byte.
//
// The whole mutation runs inside one critical section under the single
// process-wide Engine.mutex, taken exactly once. Every fallible check completes
// before the first byte changes: the section has to be one this project has a
// confirmed write contract for, the record still has to exist at the addressed
// coordinates, still has to carry that exact identity, its handle still has to
// resolve to a game ID, and no Equipment slot, Quick Item or Pouch slot of the
// character may reference its physical InventoryHeld common row. A write that
// cannot be verified restores the exact previous
// bytes of everything it changed and reports an error without advancing the
// revision, marking the session dirty or retiring an identity.
func (engine *Engine) RemoveOwnedItem(
	saveSessionID string,
	characterID int,
	ownedItemID string,
	expectedRevision string,
) (RemoveOwnedItemResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return RemoveOwnedItemResult{}, apperror.InvalidRevision(expectedRevision)
	}

	var gameID uint32
	committed, err := engine.commitCharacterRevision(saveSessionID, kindRemoveOwnedItem, characterID, func(loaded *loadedSave) error {
		if characterID < 0 || characterID >= characterSlotCount {
			return fmt.Errorf("characterID %d is outside the range 0..%d",
				characterID, characterSlotCount-1)
		}
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return apperror.RevisionConflict(expectedRevision, current)
		}

		locator, err := loaded.session.resolveOwnedItemID(characterID, ownedItemID)
		if err != nil {
			return err
		}
		gameID, err = removeOwnedItemRecord(loaded, locator, ownedItemID)
		return err
	})
	if err != nil {
		return RemoveOwnedItemResult{}, err
	}

	return RemoveOwnedItemResult{
		MutationReceipt: committed,
		OwnedItemID:     ownedItemID,
		CharacterID:     characterID,
		GameID:          gameID,
	}, nil
}

// removeOwnedItemRecord validates the complete removal plan and only then
// performs it. It reads exactly the container the token was minted in, through
// that container's own reader, so anchors, bounds, section sizes, sentinels,
// physical indexes and minting keep their single owner.
//
// The caller must already hold Engine.mutex.
func removeOwnedItemRecord(
	loaded *loadedSave,
	locator ownedItemLocator,
	ownedItemID string,
) (uint32, error) {
	gameID, removal, err := planOwnedItemRemovalWrite(loaded, locator, ownedItemID)
	if err != nil {
		return 0, err
	}
	if err := applyByteWrites(loaded.snapshot, removal.writes()); err != nil {
		return 0, fmt.Errorf("ownedItemID %q: %w", ownedItemID, err)
	}
	return gameID, nil
}

// plannedOwnedItemRemoval is one already-validated record clear and, where the
// native section owns one, the original count header that must be decremented.
// Batch repair uses the header separately so several removals from one section
// produce one correct count write instead of overlapping writes.
type plannedOwnedItemRemoval struct {
	record  byteWrite
	countAt int64
	count   uint32
}

func (removal plannedOwnedItemRemoval) writes() []byteWrite {
	writes := []byteWrite{removal.record}
	if removal.countAt >= 0 && removal.count > 0 {
		writes = append(writes, byteWrite{at: removal.countAt, data: littleEndianUint32(removal.count - 1)})
	}
	return writes
}

// planOwnedItemRemovalWrite validates one removal and prepares its physical
// writes without mutating the snapshot. It is shared by the standalone remover
// and the one-commit repair executor.
//
// The caller must already hold Engine.mutex.
func planOwnedItemRemovalWrite(
	loaded *loadedSave,
	locator ownedItemLocator,
	ownedItemID string,
) (uint32, plannedOwnedItemRemoval, error) {
	characterID := locator.characterID
	// The unsupported section is refused first, before anything is read: an
	// operation this project has no write contract for may not even look like it
	// started.
	plan, err := planOwnedItemRemoval(locator, ownedItemID)
	if err != nil {
		return 0, plannedOwnedItemRemoval{}, err
	}

	records, err := readOwnedRecords(loaded, characterID, locator.container)
	if err != nil {
		return 0, plannedOwnedItemRemoval{}, err
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
		return 0, plannedOwnedItemRemoval{}, fmt.Errorf("ownedItemID %q no longer addresses a record of character %d",
			ownedItemID, characterID)
	}

	// An unresolvable handle is a hard error rather than a removal: undecodable
	// data is never turned into a deletion, and the GaItem table this resolves
	// against is left exactly as it is.
	byHandle, err := readGaItemMap(loaded.snapshot, loaded.session.platform, characterID)
	if err != nil {
		return 0, plannedOwnedItemRemoval{}, fmt.Errorf("cannot resolve items of character %d: %w", characterID, err)
	}
	gameID, err := resolveGaItemHandle(byHandle, records[target].gaItemHandle)
	if err != nil {
		return 0, plannedOwnedItemRemoval{}, fmt.Errorf("ownedItemID %q: %w", ownedItemID, err)
	}

	if err := rejectReferencedOwnedItem(
		loaded, locator, records[target].gaItemHandle, ownedItemID); err != nil {
		return 0, plannedOwnedItemRemoval{}, err
	}

	recordAt, countAt, recordSize, err := ownedItemRowAndCountAt(loaded, locator)
	if err != nil {
		return 0, plannedOwnedItemRemoval{}, err
	}
	if int64(len(plan.cleared)) != recordSize {
		return 0, plannedOwnedItemRemoval{}, fmt.Errorf(
			"ownedItemID %q: the removal plan states %d bytes and the container stores %d",
			ownedItemID, len(plan.cleared), recordSize)
	}

	var count uint32
	if plan.maintainsCount {
		count, err = loaded.snapshot.uint32At(countAt)
		if err != nil {
			return 0, plannedOwnedItemRemoval{}, fmt.Errorf("cannot read the record count of ownedItemID %q: %w", ownedItemID, err)
		}
	} else {
		countAt = -1
	}

	return gameID, plannedOwnedItemRemoval{
		record:  byteWrite{at: recordAt, data: plan.cleared},
		countAt: countAt,
		count:   count,
	}, nil
}

// ownedItemRemovalPlan states what one removal writes into the addressed record
// and whether it maintains the non-empty count of that record's section. The two
// differ per physical section, so they are decided once, here, instead of being
// generalised into one write the native formats do not agree on.
type ownedItemRemovalPlan struct {
	cleared        []byte
	maintainsCount bool
}

// planOwnedItemRemoval derives the write of one removal from the section the
// record lives in, and refuses a section this project has no confirmed write
// contract for. It reads no save data at all: the plan depends on the locator
// only.
func planOwnedItemRemoval(locator ownedItemLocator, ownedItemID string) (ownedItemRemovalPlan, error) {
	switch {
	case locator.container == ownedContainerInventory:
		// Both InventoryHeld sections keep the physical row number in the third
		// field of a cleared record, which is what SaveForge 1.5.8 and 1.6.8 wrote
		// and what their tests still protect. Only the common section has a count
		// that moves with it: the key_count header stayed untouched there, and no
		// native evidence in this project says otherwise.
		cleared := make([]byte, inventoryHeldRecordSize)
		binary.LittleEndian.PutUint32(cleared[8:], uint32(locator.physicalIndex))
		return ownedItemRemovalPlan{
			cleared:        cleared,
			maintainsCount: locator.containerSection == InventorySectionCommon,
		}, nil
	case locator.container == ownedContainerStorage &&
		locator.containerSection == StorageSectionCommon:
		// The Storage Box is a different format with its own confirmed rule: the
		// whole record is zeroed, third field included.
		return ownedItemRemovalPlan{
			cleared:        make([]byte, storageRecordSize),
			maintainsCount: true,
		}, nil
	case locator.container == ownedContainerStorage &&
		locator.containerSection == StorageSectionKey:
		return ownedItemRemovalPlan{}, fmt.Errorf(
			"ownedItemID %q addresses a Storage key record, and removing one is not supported:"+
				" this project has no confirmed native write contract for that section", ownedItemID)
	default:
		return ownedItemRemovalPlan{}, fmt.Errorf("unknown container %q", locator.container)
	}
}

// ownedItemReference is one stored reference to a physical InventoryHeld common
// row: the GaItem handle the structure carries and the raw row field beside it.
// The structure name and the slot exist for the rejection message alone.
type ownedItemReference struct {
	structure string
	slot      int
	handle    uint32
	row       uint32
}

// rejectReferencedOwnedItem refuses to remove an instance an active structure of
// the character still points at.
//
// **What identifies the referenced instance is the physical InventoryHeld common
// row, not the handle.** `docs/owned-item-identity.md` L1 and "Variant B —
// GaItemHandle alone" record that one GaItem handle is legitimately shared by
// several physical records, in one container or split across Inventory and
// Storage. A guard that matched on the handle alone would therefore refuse to
// remove a record nothing references, and would contradict the rule that a
// removal deletes exactly one instance. Legacy's own reference check
// (`v1.6.8:backend/core/equipment_writer_test.go`, `nativeHandleMatches`)
// resolves the row first and only then compares the handle of that row, which is
// the same rule this guard applies.
//
// The guard therefore covers records of the InventoryHeld common section only,
// and returns immediately for every other section and container: the row fields
// below are counted in that one section, so an Inventory key record and both
// Storage sections can never be named by one of them, however their handle is
// shared.
//
// Three structures carry such a reference and all three are checked, because all
// three would otherwise be left pointing at a row this removal empties:
//
//	22 Equipment slots  — EquipedItemIndex beside ActiveEquipedItemsGa
//	10 Quick Items      — the EquipItemData pairs
//	 6 Pouch slots      — the EquipItemData pairs behind them
//
// For each pair:
//
//   - a pair naming the addressed row with that row's handle is an exact
//     reference to this instance and rejects the removal;
//   - a pair naming the addressed row with a different handle is an inconsistent
//     reference; it is rejected fail-closed rather than repaired or ignored;
//   - a pair naming another row never blocks the removal, even when it carries
//     the same handle: a shared handle at a different row is a reference to a
//     different instance.
//
// Nothing is unequipped, cleared or cascaded, and no byte of any of the three
// structures is written here.
//
// The caller must already hold Engine.mutex.
func rejectReferencedOwnedItem(
	loaded *loadedSave,
	locator ownedItemLocator,
	gaItemHandle uint32,
	ownedItemID string,
) error {
	if locator.container != ownedContainerInventory ||
		locator.containerSection != InventorySectionCommon {
		return nil
	}
	characterID := locator.characterID
	references, err := ownedItemRowReferences(loaded, characterID)
	if err != nil {
		return err
	}

	for _, reference := range references {
		if reference.row == removeReferenceInvalidRow ||
			reference.row < removeReferenceInventoryRowBase {
			continue
		}
		if int(reference.row-removeReferenceInventoryRowBase) != locator.physicalIndex {
			continue
		}
		if reference.handle == gaItemHandle {
			return fmt.Errorf(
				"ownedItemID %q is referenced by %s %d of character %d and is not removed;"+
					" unequip it first", ownedItemID, reference.structure, reference.slot, characterID)
		}
		return fmt.Errorf(
			"ownedItemID %q sits in the inventory row %s %d of character %d references, and that"+
				" reference carries the different handle 0x%08X; the removal fails closed rather"+
				" than emptying a referenced row", ownedItemID, reference.structure, reference.slot,
			characterID, reference.handle)
	}
	return nil
}

// ownedItemRowReferences reads every stored {handle, InventoryHeld common row}
// pair of one active slot: the 22 Equipment slots in front of InventoryHeld and
// the 10 Quick Item plus 6 Pouch pairs of EquipItemData behind it.
//
// It is the single place this mutation decodes a reference pair, so the guard
// above cannot end up applying one rule to Equipment and another to
// EquipItemData. Every position is measured from the same anchor the container
// reader used, and the two EquipItemData blocks reuse the offsets, slot counts
// and record size their own getters already own.
//
// The caller must already hold Engine.mutex and must have established that the
// slot is active.
func ownedItemRowReferences(loaded *loadedSave, characterID int) ([]ownedItemReference, error) {
	sectionAt, err := inventoryHeldSectionAt(loaded, characterID)
	if err != nil {
		return nil, err
	}
	anchor := sectionAt - inventoryHeldCommonOffset
	_, slotEnd := inventorySlotBounds(loaded.session.platform, characterID)
	// EquipItemData sits behind the section inventoryHeldSectionAt already
	// bounded, so its own end is the one distance that still has to fit.
	if anchor+pouchItemsSectionOffset+pouchItemsReadSize > slotEnd {
		return nil, fmt.Errorf(
			"the quick items and pouch of character %d do not fit into its slot", characterID)
	}

	handles, err := loaded.snapshot.readAt(anchor+removeEquipmentHandlesOffset, equipmentSlotCount*4)
	if err != nil {
		return nil, fmt.Errorf("cannot read the equipped handles of character %d: %w", characterID, err)
	}
	rows, err := loaded.snapshot.readAt(anchor+removeEquipmentIndexesOffset, equipmentSlotCount*4)
	if err != nil {
		return nil, fmt.Errorf("cannot read the equipped rows of character %d: %w", characterID, err)
	}

	references := make([]ownedItemReference, 0,
		equipmentSlotCount+quickItemSlotCount+pouchItemSlotCount)
	for slot := 0; slot < equipmentSlotCount; slot++ {
		references = append(references, ownedItemReference{
			structure: "equipment slot",
			slot:      slot,
			handle:    binary.LittleEndian.Uint32(handles[slot*4:]),
			row:       binary.LittleEndian.Uint32(rows[slot*4:]),
		})
	}

	references, err = appendEquipItemDataReferences(loaded, references, "quick item",
		anchor+quickItemsSectionOffset, quickItemSlotCount, quickItemRecordSize, characterID)
	if err != nil {
		return nil, err
	}
	return appendEquipItemDataReferences(loaded, references, "pouch slot",
		anchor+pouchItemsSectionOffset, pouchItemSlotCount, pouchItemRecordSize, characterID)
}

// appendEquipItemDataReferences decodes one interleaved EquipItemData block —
// the Quick Items or the Pouch — into reference pairs. Both blocks store the
// GaItem handle followed by the row field in every record, so they share this
// one decode instead of two copies of it.
func appendEquipItemDataReferences(
	loaded *loadedSave,
	into []ownedItemReference,
	structure string,
	at int64,
	count, size, characterID int,
) ([]ownedItemReference, error) {
	raw, err := loaded.snapshot.readAt(at, count*size)
	if err != nil {
		return nil, fmt.Errorf("cannot read the %s pairs of character %d: %w",
			structure, characterID, err)
	}
	for slot := 0; slot < count; slot++ {
		pair := raw[slot*size:]
		into = append(into, ownedItemReference{
			structure: structure,
			slot:      slot,
			handle:    binary.LittleEndian.Uint32(pair),
			row:       binary.LittleEndian.Uint32(pair[4:]),
		})
	}
	return into, nil
}

// writeOwnedItemRemoval is the write step: it clears the record, maintains the
// non-empty count of its section where that count moves at all, verifies both
// writes and restores everything it changed when either cannot be confirmed.
//
// countAt below zero means the section's count does not participate in this
// removal, which is the confirmed contract of the InventoryHeld key section: its
// key_count header is left exactly as it was.
//
// The count header of a participating section states how many non-empty records
// that section holds, so removing one record lowers it by one. A header that
// already reads 0 is left alone instead of wrapping to 0xFFFFFFFF: the save is
// then already inconsistent with its own content, and a removal is not the place
// to repair or to reject it. SaveForge 1.5.8 and 1.6.8 made the same choice on
// their common sections.
//
// The two writes are sixteen bytes together, so the rollback is those exact
// previous bytes rather than a copy of the snapshot. A rejected write changes
// nothing, because the codec bounds-checks a whole range before its first byte,
// so only a failure behind an accepted write has to put the count back too.
func writeOwnedItemRemoval(
	snapshot *codec,
	recordAt int64,
	cleared []byte,
	countAt int64,
	count uint32,
) error {
	previous, err := snapshot.readAt(recordAt, len(cleared))
	if err != nil {
		return fmt.Errorf("cannot read the record: %w", err)
	}
	maintainsCount := countAt >= 0 && count > 0
	// A section whose count does not participate has nothing to roll back either.
	rollbackCountAt := int64(-1)
	if maintainsCount {
		rollbackCountAt = countAt
	}

	if err := snapshot.writeAt(recordAt, cleared); err != nil {
		return fmt.Errorf("cannot clear the record: %w", err)
	}
	if maintainsCount {
		if err := snapshot.writeAt(countAt, littleEndianUint32(count-1)); err != nil {
			// The count write was rejected as a whole, so the count is still the
			// value that was read and only the record has to come back.
			return restoreOwnedItemRemoval(snapshot, recordAt, previous, -1, count,
				"the record count could not be written; the record is unchanged")
		}
	}

	written, err := snapshot.readAt(recordAt, len(cleared))
	if err != nil || !bytes.Equal(written, cleared) {
		return restoreOwnedItemRemoval(snapshot, recordAt, previous, rollbackCountAt, count,
			"the record was not cleared; it is unchanged")
	}
	if maintainsCount {
		stored, err := snapshot.uint32At(countAt)
		if err != nil || stored != count-1 {
			return restoreOwnedItemRemoval(snapshot, recordAt, previous, rollbackCountAt, count,
				"the record count could not be verified; the record is unchanged")
		}
	}
	return nil
}

// restoreOwnedItemRemoval puts the exact previous bytes of everything the
// removal changed back and confirms every one of them, so a failed mutation
// either leaves nothing changed or says that it could not.
//
// countAt below zero means the count was never changed and is not restored. The
// count is written before the record so that the record, which is the wider
// write, decides any range the two might share.
func restoreOwnedItemRemoval(
	snapshot *codec,
	recordAt int64,
	previous []byte,
	countAt int64,
	count uint32,
	reason string,
) error {
	if countAt >= 0 {
		if err := snapshot.writeAt(countAt, littleEndianUint32(count)); err != nil {
			return fmt.Errorf("%s could not be restored: %w", reason, err)
		}
	}
	if err := snapshot.writeAt(recordAt, previous); err != nil {
		return fmt.Errorf("%s could not be restored: %w", reason, err)
	}
	restored, err := snapshot.readAt(recordAt, len(previous))
	if err != nil || !bytes.Equal(restored, previous) {
		return fmt.Errorf("%s and its rollback could not be confirmed", reason)
	}
	if countAt >= 0 {
		stored, err := snapshot.uint32At(countAt)
		if err != nil || stored != count {
			return fmt.Errorf("%s and its rollback could not be confirmed", reason)
		}
	}
	return fmt.Errorf("%s", reason)
}

// ownedItemRowAndCountAt reports where the record of one locator starts, where
// the non-empty count of its physical section lives, and how wide one record of
// that container is. The start of the container comes from the same helper its
// reader uses, so a reader and a writer can never disagree about where a row
// lives, and this is the only place that maps a container to its record size and
// section distances.
//
// It reports positions only. Whether a removal actually maintains the count it
// points at is the removal plan's decision, not this helper's.
//
// The offsets stay inside this package: none is a field of InventoryRecord, of
// StorageRecord, of OwnedItem or of any result.
//
// The caller must already hold Engine.mutex.
func ownedItemRowAndCountAt(loaded *loadedSave, locator ownedItemLocator) (int64, int64, int64, error) {
	switch locator.container {
	case ownedContainerInventory:
		sectionAt, err := inventoryHeldSectionAt(loaded, locator.characterID)
		if err != nil {
			return 0, 0, 0, err
		}
		// The common count sits immediately in front of the first common record,
		// and the key count immediately in front of the first key record.
		row, countAt := int64(0), sectionAt-4
		if locator.containerSection == InventorySectionKey {
			row, countAt = inventoryHeldKeyAt, sectionAt+inventoryHeldCommonSize
		}
		return sectionAt + row + int64(locator.physicalIndex)*inventoryHeldRecordSize,
			countAt, inventoryHeldRecordSize, nil
	case ownedContainerStorage:
		sectionAt, err := storageBoxSectionAt(loaded, locator.characterID)
		if err != nil {
			return 0, 0, 0, err
		}
		// The Storage Box begins with its common count, so the section offset is
		// that header itself; the key count again sits in front of the key records.
		row, countAt := int64(storageCommonAt), sectionAt
		if locator.containerSection == StorageSectionKey {
			row, countAt = storageKeyAt, sectionAt+storageKeyAt-4
		}
		return sectionAt + row + int64(locator.physicalIndex)*storageRecordSize,
			countAt, storageRecordSize, nil
	default:
		return 0, 0, 0, fmt.Errorf("unknown container %q", locator.container)
	}
}
