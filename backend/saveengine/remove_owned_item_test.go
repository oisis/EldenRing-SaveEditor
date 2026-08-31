package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// removeTestRecordAt restates where the twelve bytes of one row live,
// independently of the implementation, so a removal that addressed a position
// derived some other way fails here. The quantity is the second field of the
// row, so the row starts four bytes in front of the offset the quantity tests
// state.
func removeTestRecordAt(platform Platform, container, section string, index int) int64 {
	return quantityTestOffset(platform, container, section, index) - 4
}

// removeTestCountAt restates where the non-empty count of one physical section
// lives: immediately in front of the first record of that section.
func removeTestCountAt(platform Platform, container, section string) int64 {
	return removeTestRecordAt(platform, container, section, 0) - 4
}

// The confirmed reference representation, restated here independently of the
// implementation.
//
// The three equipment blocks of 22 little-endian uint32 fields sit in front of
// InventoryHeld, each behind one leading byte, and the last of them ends on the
// four-byte common count of the inventory section.
//
// EquipItemData sits behind InventoryHeld at the same constant distance
// quick_items.go and pouch_items.go state, and holds one interleaved
// {handle, row} pair per slot: ten Quick Items, then the four-byte active-quick
// value, then six Pouch slots.
const (
	removeTestEquipSlots     = 22
	removeTestEquipHandlesAt = inventoryTestCommonAt - 4 - removeTestEquipSlots*4
	removeTestEquipIndexesAt = removeTestEquipHandlesAt - removeTestEquipSlots*4 - 0x1C -
		removeTestEquipSlots*4
	removeTestEquipIndexBase = 0x180

	removeTestQuickSlots  = 10
	removeTestPouchSlots  = 6
	removeTestQuickPairAt = 0x9279
	removeTestPouchPairAt = removeTestQuickPairAt + removeTestQuickSlots*8 + 4
)

// removeTestAnchorAt restates where the confirmed slot anchor sits, measured
// back from the first common inventory row every other offset here is derived
// from.
func removeTestAnchorAt(platform Platform) int64 {
	return removeTestRecordAt(platform, ownedContainerInventory, InventorySectionCommon, 0) -
		inventoryTestCommonAt
}

// removeTestEquipAt restates where one field of the equipped-handle block or of
// the equipped-row block lives, measured from the same anchor every row offset
// above is measured from.
func removeTestEquipAt(platform Platform, blockAt int64, slot int) int64 {
	return removeTestAnchorAt(platform) + blockAt + int64(slot)*4
}

// removeTestPairFieldsAt restates where the handle field and the row field of
// one interleaved EquipItemData pair live.
func removeTestPairFieldsAt(platform Platform, blockAt int64, slot int) (int64, int64) {
	at := removeTestAnchorAt(platform) + blockAt + int64(slot)*8
	return at, at + 4
}

// removeTestReference is one writable {handle, row} reference pair of one of the
// three structures that can point at a physical InventoryHeld common row.
type removeTestReference struct {
	name     string
	handleAt int64
	rowAt    int64
}

// removeTestReferences are the three confirmed reference structures, each at one
// representative slot. The guard has to behave identically in all three.
func removeTestReferences(platform Platform) []removeTestReference {
	quickHandleAt, quickRowAt := removeTestPairFieldsAt(platform, removeTestQuickPairAt, 3)
	pouchHandleAt, pouchRowAt := removeTestPairFieldsAt(platform, removeTestPouchPairAt, 5)
	return []removeTestReference{
		{
			"equipment slot 1",
			removeTestEquipAt(platform, removeTestEquipHandlesAt, 1),
			removeTestEquipAt(platform, removeTestEquipIndexesAt, 1),
		},
		{"quick item 3", quickHandleAt, quickRowAt},
		{"pouch slot 5", pouchHandleAt, pouchRowAt},
	}
}

// removeTestBytes reads one range of the session's private snapshot exactly as
// it sits there.
func removeTestBytes(t *testing.T, engine *Engine, saveSessionID string, at int64, length int) []byte {
	t.Helper()

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	raw, err := engine.sessions[saveSessionID].snapshot.readAt(at, length)
	if err != nil {
		t.Fatalf("read 0x%X: %v", at, err)
	}
	return raw
}

// removeTestPut writes one uint32 into the private snapshot. The shared fixture
// leaves every section count at 0 and every handle resolvable, which are the
// cases most tests want; a test about another case states its own value here
// instead of changing the fixture every other test depends on.
func removeTestPut(t *testing.T, engine *Engine, saveSessionID string, at int64, value uint32) {
	t.Helper()

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if err := engine.sessions[saveSessionID].snapshot.writeAt(at, littleEndianUint32(value)); err != nil {
		t.Fatalf("write 0x%08X at 0x%X: %v", value, at, err)
	}
}

// removeTestIdentities reads one container through its own getter and keys the
// minted identities by physical row, which is also how a caller obtains an
// identity at all.
func removeTestIdentities(
	t *testing.T, engine *Engine, saveSessionID, container string, characterID int,
) map[string]string {
	t.Helper()

	if container == ownedContainerStorage {
		storage, err := engine.GetStorage(saveSessionID, characterID, "", 0, 0)
		if err != nil {
			t.Fatalf("GetStorage: %v", err)
		}
		return storageTestIdentitiesByRow(t, storage.Records)
	}
	inventory, err := engine.GetInventory(saveSessionID, characterID, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	return inventoryTestIdentitiesByRow(t, inventory.Records)
}

func removeTestCommonID(t *testing.T, engine *Engine, saveSessionID, container string) string {
	t.Helper()

	id := removeTestIdentities(t, engine, saveSessionID, container,
		ownedContainerTestSlot)["common#"+strconv.Itoa(ownedContainerTestCommonIndex)]
	if id == "" {
		t.Fatal("the container read never identified the common row")
	}
	return id
}

// One committed removal per container and per platform: the addressed row is
// cleared, the other row of the same item in the same container survives, the
// twin row in the other container survives, the revision advances by exactly
// one, the session becomes dirty and the identity used for the mutation is
// retired.
func TestRemoveOwnedItemCommitsInBothContainersOnBothPlatforms(t *testing.T) {
	cases := map[string]struct {
		platform      Platform
		container     string
		other         string
		commonSection string
		keySection    string
	}{
		"inventory on PC":  {PlatformPC, ownedContainerInventory, ownedContainerStorage, InventorySectionCommon, InventorySectionKey},
		"inventory on PS4": {PlatformPS4, ownedContainerInventory, ownedContainerStorage, InventorySectionCommon, InventorySectionKey},
		"storage on PC":    {PlatformPC, ownedContainerStorage, ownedContainerInventory, StorageSectionCommon, StorageSectionKey},
		"storage on PS4":   {PlatformPS4, ownedContainerStorage, ownedContainerInventory, StorageSectionCommon, StorageSectionKey},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			engine, saveSessionID := loadOwnedItemContainers(t, testCase.platform)
			// The other container carries a count of its own, so a removal that
			// lowered the wrong container's count is visible below.
			otherCountAt := removeTestCountAt(testCase.platform, testCase.other, testCase.commonSection)
			removeTestPut(t, engine, saveSessionID, otherCountAt, 5)
			// Both containers are read first, so both carry live identities and a
			// removal in one of them has to leave the other alone.
			otherIDs := removeTestIdentities(
				t, engine, saveSessionID, testCase.other, ownedContainerTestSlot)
			id := removeTestCommonID(t, engine, saveSessionID, testCase.container)

			result, err := engine.RemoveOwnedItem(saveSessionID, ownedContainerTestSlot, id, "0")
			if err != nil {
				t.Fatalf("RemoveOwnedItem: %v", err)
			}
			want := RemoveOwnedItemResult{
				SaveSessionID: saveSessionID,
				SaveRevision:  "1",
				OwnedItemID:   id,
				CharacterID:   ownedContainerTestSlot,
				GameID:        ownedContainerTestGameID,
			}
			if result != want {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			clearedAt := removeTestRecordAt(testCase.platform, testCase.container,
				testCase.commonSection, ownedContainerTestCommonIndex)
			removeTestWantCleared(t, engine, saveSessionID, testCase.container,
				clearedAt, ownedContainerTestCommonIndex)
			// The second row of the same item in the same container is a different
			// instance and stays exactly as it was.
			keptAt := removeTestRecordAt(testCase.platform, testCase.container,
				testCase.keySection, ownedContainerTestKeyIndex)
			removeTestWantRow(t, engine, saveSessionID, keptAt, ownedContainerTestRawKeyQty)
			// The row at the same coordinates in the other container is a different
			// record and is never searched, matched or cleared.
			otherAt := removeTestRecordAt(testCase.platform, testCase.other,
				testCase.commonSection, ownedContainerTestCommonIndex)
			removeTestWantRow(t, engine, saveSessionID, otherAt, ownedContainerTestRawQuantity)
			if got := removeTestCount(t, engine, saveSessionID, otherCountAt); got != 5 {
				t.Errorf("the other container's count = %d, want the untouched 5", got)
			}

			revision, dirty := quantityTestSession(t, engine, saveSessionID)
			if revision != "1" {
				t.Errorf("revision after one removal = %q, want \"1\"", revision)
			}
			if !dirty {
				t.Error("a committed removal left the session without unsaved changes")
			}

			// Every identity of the previous revision is retired, including the one
			// the removal was performed with and the untouched other container's.
			if _, err := engine.GetOwnedItem(saveSessionID, ownedContainerTestSlot, id); !errors.Is(
				err, errStaleOwnedItemID) {
				t.Errorf("the used ownedItemID resolved with %v, want errStaleOwnedItemID", err)
			}
			for row, otherID := range otherIDs {
				if _, err := engine.GetOwnedItem(
					saveSessionID, ownedContainerTestSlot, otherID); !errors.Is(err, errStaleOwnedItemID) {
					t.Errorf("%s of the other container resolved with %v, want errStaleOwnedItemID", row, err)
				}
			}

			// The record is gone from the container it lived in, and the other
			// container still reports both of its rows.
			if rows := removeTestIdentities(
				t, engine, saveSessionID, testCase.container, ownedContainerTestSlot); len(rows) != 1 {
				t.Errorf("the mutated container reports %d rows, want 1", len(rows))
			}
			if rows := removeTestIdentities(
				t, engine, saveSessionID, testCase.other, ownedContainerTestSlot); len(rows) != 2 {
				t.Errorf("the other container reports %d rows, want 2", len(rows))
			}
		})
	}
}

// removeTestWantCleared asserts that one row was cleared the way its own
// physical section is cleared: InventoryHeld keeps the physical row number in
// the third field, the Storage Box zeroes all twelve bytes.
func removeTestWantCleared(
	t *testing.T, engine *Engine, saveSessionID, container string, at int64, physicalIndex int,
) {
	t.Helper()

	want := make([]byte, 12)
	if container == ownedContainerInventory {
		binary.LittleEndian.PutUint32(want[8:], uint32(physicalIndex))
	}
	if raw := removeTestBytes(t, engine, saveSessionID, at, 12); !bytes.Equal(raw, want) {
		t.Errorf("removed %s row at 0x%X = % X, want % X", container, at, raw, want)
	}
}

// removeTestWantRow asserts that one row still holds the fixture record.
func removeTestWantRow(t *testing.T, engine *Engine, saveSessionID string, at int64, rawQuantity uint32) {
	t.Helper()

	raw := removeTestBytes(t, engine, saveSessionID, at, 12)
	want := make([]byte, 12)
	binary.LittleEndian.PutUint32(want, ownedContainerTestHandle)
	binary.LittleEndian.PutUint32(want[4:], rawQuantity)
	binary.LittleEndian.PutUint32(want[8:], ownedContainerTestAcquisition)
	if !bytes.Equal(raw, want) {
		t.Errorf("row at 0x%X = % X, want % X", at, raw, want)
	}
}

// The common count of the section the record lived in drops by exactly one, and
// no other count moves. A count that already reads 0 is left alone rather than
// wrapped around.
func TestRemoveOwnedItemMaintainsTheCommonCount(t *testing.T) {
	for _, container := range []string{ownedContainerInventory, ownedContainerStorage} {
		t.Run(container, func(t *testing.T) {
			commonSection, keySection := InventorySectionCommon, InventorySectionKey
			if container == ownedContainerStorage {
				commonSection, keySection = StorageSectionCommon, StorageSectionKey
			}
			commonCountAt := removeTestCountAt(PlatformPC, container, commonSection)
			keyCountAt := removeTestCountAt(PlatformPC, container, keySection)

			engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)
			removeTestPut(t, engine, saveSessionID, commonCountAt, 2)
			removeTestPut(t, engine, saveSessionID, keyCountAt, 1)
			id := removeTestCommonID(t, engine, saveSessionID, container)

			if _, err := engine.RemoveOwnedItem(saveSessionID, ownedContainerTestSlot, id, "0"); err != nil {
				t.Fatalf("RemoveOwnedItem: %v", err)
			}
			if got := removeTestCount(t, engine, saveSessionID, commonCountAt); got != 1 {
				t.Errorf("common count = %d, want 1", got)
			}
			if got := removeTestCount(t, engine, saveSessionID, keyCountAt); got != 1 {
				t.Errorf("key count = %d, want the untouched 1", got)
			}
		})
	}
}

// SaveForge 1.6.8 removed an InventoryHeld key record without touching the
// key_count header, and backend/core/remove_key_item_test.go still protects
// that. 2.0 inherits the confirmed behaviour: the row is cleared, both counts
// stay exactly where they were, and no native evidence in this project says
// otherwise.
func TestRemoveOwnedItemLeavesTheInventoryKeyCountUntouched(t *testing.T) {
	commonCountAt := removeTestCountAt(PlatformPC, ownedContainerInventory, InventorySectionCommon)
	keyCountAt := removeTestCountAt(PlatformPC, ownedContainerInventory, InventorySectionKey)

	engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)
	removeTestPut(t, engine, saveSessionID, commonCountAt, 2)
	removeTestPut(t, engine, saveSessionID, keyCountAt, 1)
	keyID := removeTestIdentities(t, engine, saveSessionID, ownedContainerInventory,
		ownedContainerTestSlot)["key#"+strconv.Itoa(ownedContainerTestKeyIndex)]
	if keyID == "" {
		t.Fatal("the container read never identified the key row")
	}

	if _, err := engine.RemoveOwnedItem(saveSessionID, ownedContainerTestSlot, keyID, "0"); err != nil {
		t.Fatalf("RemoveOwnedItem: %v", err)
	}
	if got := removeTestCount(t, engine, saveSessionID, keyCountAt); got != 1 {
		t.Errorf("key_count = %d, want the protected, untouched 1", got)
	}
	if got := removeTestCount(t, engine, saveSessionID, commonCountAt); got != 2 {
		t.Errorf("common count after a key removal = %d, want the untouched 2", got)
	}
	removeTestWantCleared(t, engine, saveSessionID, ownedContainerInventory,
		removeTestRecordAt(PlatformPC, ownedContainerInventory,
			InventorySectionKey, ownedContainerTestKeyIndex), ownedContainerTestKeyIndex)
}

// A cleared InventoryHeld row keeps its physical row number in the third field
// in both of its sections, while a cleared Storage Box row is zeroed whole. The
// two formats are not generalised into one write.
func TestRemoveOwnedItemClearsEachSectionItsOwnWay(t *testing.T) {
	cases := map[string]struct {
		container string
		section   string
		row       string
		index     int
	}{
		"inventory common": {ownedContainerInventory, InventorySectionCommon,
			"common#" + strconv.Itoa(ownedContainerTestCommonIndex), ownedContainerTestCommonIndex},
		"inventory key": {ownedContainerInventory, InventorySectionKey,
			"key#" + strconv.Itoa(ownedContainerTestKeyIndex), ownedContainerTestKeyIndex},
		"storage common": {ownedContainerStorage, StorageSectionCommon,
			"common#" + strconv.Itoa(ownedContainerTestCommonIndex), ownedContainerTestCommonIndex},
	}

	for name, testCase := range cases {
		for _, platform := range []Platform{PlatformPC, PlatformPS4} {
			t.Run(name+" on "+string(platform), func(t *testing.T) {
				engine, saveSessionID := loadOwnedItemContainers(t, platform)
				id := removeTestIdentities(t, engine, saveSessionID, testCase.container,
					ownedContainerTestSlot)[testCase.row]
				if id == "" {
					t.Fatalf("the container read never identified %s", testCase.row)
				}
				if _, err := engine.RemoveOwnedItem(
					saveSessionID, ownedContainerTestSlot, id, "0"); err != nil {
					t.Fatalf("RemoveOwnedItem: %v", err)
				}
				removeTestWantCleared(t, engine, saveSessionID, testCase.container,
					removeTestRecordAt(platform, testCase.container, testCase.section, testCase.index),
					testCase.index)
			})
		}
	}
}

// The Storage Box key section has no confirmed native write contract: SaveForge
// 1.6.8 never wrote it and its own specification records the semantics as
// unverified. The removal fails closed on both platforms and changes nothing at
// all — not the row, not either count, not the revision and not the registry.
func TestRemoveOwnedItemRejectsStorageKeyAsUnsupported(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine, saveSessionID := loadOwnedItemContainers(t, platform)
			commonCountAt := removeTestCountAt(platform, ownedContainerStorage, StorageSectionCommon)
			keyCountAt := removeTestCountAt(platform, ownedContainerStorage, StorageSectionKey)
			removeTestPut(t, engine, saveSessionID, commonCountAt, 2)
			removeTestPut(t, engine, saveSessionID, keyCountAt, 1)

			keyID := removeTestIdentities(t, engine, saveSessionID, ownedContainerStorage,
				ownedContainerTestSlot)["key#"+strconv.Itoa(ownedContainerTestKeyIndex)]
			if keyID == "" {
				t.Fatal("the container read never identified the storage key row")
			}

			_, err := engine.RemoveOwnedItem(saveSessionID, ownedContainerTestSlot, keyID, "0")
			if err == nil || !strings.Contains(err.Error(), "not supported") {
				t.Fatalf("error = %v, want the Storage key section refused as unsupported", err)
			}

			keyAt := removeTestRecordAt(platform, ownedContainerStorage,
				StorageSectionKey, ownedContainerTestKeyIndex)
			removeTestWantRow(t, engine, saveSessionID, keyAt, ownedContainerTestRawKeyQty)
			if got := removeTestCount(t, engine, saveSessionID, keyCountAt); got != 1 {
				t.Errorf("storage key count = %d, want the untouched 1", got)
			}
			if got := removeTestCount(t, engine, saveSessionID, commonCountAt); got != 2 {
				t.Errorf("storage common count = %d, want the untouched 2", got)
			}
			if revision, dirty := quantityTestSession(t, engine, saveSessionID); revision != "0" || dirty {
				t.Errorf("revision %q, unsavedChanges %v; want \"0\" and false", revision, dirty)
			}
			// The identity survives a rejection, because nothing it addressed moved.
			if _, err := engine.GetOwnedItem(saveSessionID, ownedContainerTestSlot, keyID); err != nil {
				t.Errorf("the storage key ownedItemID stopped resolving after a rejection: %v", err)
			}
		})
	}
}

// removeTestWantRejectionChangedNothing asserts the complete fail-closed
// contract of one rejected removal: the addressed row, both counts of its
// container, the reference pair that produced the rejection, the revision, the
// unsaved-changes flag and the identity registry all stay exactly as they were.
func removeTestWantRejectionChangedNothing(
	t *testing.T,
	engine *Engine,
	saveSessionID string,
	platform Platform,
	reference removeTestReference,
	wantHandle, wantRow uint32,
	id string,
) {
	t.Helper()

	removeTestWantRow(t, engine, saveSessionID,
		removeTestRecordAt(platform, ownedContainerInventory,
			InventorySectionCommon, ownedContainerTestCommonIndex),
		ownedContainerTestRawQuantity)
	commonCountAt := removeTestCountAt(platform, ownedContainerInventory, InventorySectionCommon)
	keyCountAt := removeTestCountAt(platform, ownedContainerInventory, InventorySectionKey)
	if got := removeTestCount(t, engine, saveSessionID, commonCountAt); got != 2 {
		t.Errorf("common count = %d, want the unchanged 2", got)
	}
	if got := removeTestCount(t, engine, saveSessionID, keyCountAt); got != 1 {
		t.Errorf("key count = %d, want the unchanged 1", got)
	}
	// The reference is evidence, never a target: both of its fields still hold
	// exactly what the test wrote.
	if got := binary.LittleEndian.Uint32(
		removeTestBytes(t, engine, saveSessionID, reference.handleAt, 4)); got != wantHandle {
		t.Errorf("%s handle = 0x%08X, want the untouched 0x%08X", reference.name, got, wantHandle)
	}
	if got := binary.LittleEndian.Uint32(
		removeTestBytes(t, engine, saveSessionID, reference.rowAt, 4)); got != wantRow {
		t.Errorf("%s row = 0x%08X, want the untouched 0x%08X", reference.name, got, wantRow)
	}
	if revision, dirty := quantityTestSession(t, engine, saveSessionID); revision != "0" || dirty {
		t.Errorf("revision %q, unsavedChanges %v; want \"0\" and false", revision, dirty)
	}
	// The identity survives a rejection, because nothing it addressed moved.
	if _, err := engine.GetOwnedItem(saveSessionID, ownedContainerTestSlot, id); err != nil {
		t.Errorf("the ownedItemID stopped resolving after a rejection: %v", err)
	}
}

// removeTestLoadWithCounts loads the shared fixture and gives both InventoryHeld
// counts a non-zero value, so a rejection that moved one of them is visible.
func removeTestLoadWithCounts(t *testing.T, platform Platform) (*Engine, string) {
	t.Helper()

	engine, saveSessionID := loadOwnedItemContainers(t, platform)
	removeTestPut(t, engine, saveSessionID,
		removeTestCountAt(platform, ownedContainerInventory, InventorySectionCommon), 2)
	removeTestPut(t, engine, saveSessionID,
		removeTestCountAt(platform, ownedContainerInventory, InventorySectionKey), 1)
	return engine, saveSessionID
}

// Three structures reference an owned instance by the pair
// {GaItem handle, 0x180 + physical InventoryHeld common row}: the 22 Equipment
// slots, the 10 Quick Items and the 6 Pouch slots. A pair that names the
// addressed row with that row's own handle is an exact reference and blocks the
// removal in all three, on both platforms. Nothing is unequipped and nothing is
// cascaded.
func TestRemoveOwnedItemRejectsAnExactReferencePair(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		for _, reference := range removeTestReferences(platform) {
			t.Run(reference.name+" on "+string(platform), func(t *testing.T) {
				engine, saveSessionID := removeTestLoadWithCounts(t, platform)
				const wantRow = removeTestEquipIndexBase + ownedContainerTestCommonIndex
				removeTestPut(t, engine, saveSessionID, reference.handleAt, ownedContainerTestHandle)
				removeTestPut(t, engine, saveSessionID, reference.rowAt, wantRow)

				id := removeTestCommonID(t, engine, saveSessionID, ownedContainerInventory)
				_, err := engine.RemoveOwnedItem(saveSessionID, ownedContainerTestSlot, id, "0")
				if err == nil || !strings.Contains(err.Error(), "unequip it first") {
					t.Fatalf("error = %v, want the active reference to be named", err)
				}
				removeTestWantRejectionChangedNothing(t, engine, saveSessionID, platform,
					reference, ownedContainerTestHandle, wantRow, id)
			})
		}
	}
}

// A pair that names the addressed row while carrying a different handle is an
// inconsistent reference. It is rejected fail-closed — never ignored, never
// repaired — in all three structures and on both platforms.
func TestRemoveOwnedItemRejectsAReferencedRowWithAnotherHandle(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		for _, reference := range removeTestReferences(platform) {
			t.Run(reference.name+" on "+string(platform), func(t *testing.T) {
				engine, saveSessionID := removeTestLoadWithCounts(t, platform)
				const wantHandle = uint32(0xB0009999)
				const wantRow = removeTestEquipIndexBase + ownedContainerTestCommonIndex
				removeTestPut(t, engine, saveSessionID, reference.handleAt, wantHandle)
				removeTestPut(t, engine, saveSessionID, reference.rowAt, wantRow)

				id := removeTestCommonID(t, engine, saveSessionID, ownedContainerInventory)
				_, err := engine.RemoveOwnedItem(saveSessionID, ownedContainerTestSlot, id, "0")
				if err == nil || !strings.Contains(err.Error(), "fails closed") {
					t.Fatalf("error = %v, want the inconsistent reference to fail closed", err)
				}
				if strings.Contains(err.Error(), "unequip it first") {
					t.Errorf("error = %v, want it distinguishable from an exact reference", err)
				}
				removeTestWantRejectionChangedNothing(t, engine, saveSessionID, platform,
					reference, wantHandle, wantRow, id)
			})
		}
	}
}

// A reference pair is exact, not fuzzy: a slot that references no row at all,
// one that references another row, and — decisively — one that carries this
// instance's own handle while referencing another row all leave the removal
// alone. `docs/owned-item-identity.md` L1 records that one handle is
// legitimately shared, so a matching handle at a different row names a different
// instance and may not block anything.
func TestRemoveOwnedItemAcceptsAnUnrelatedReferencePair(t *testing.T) {
	cases := map[string]struct{ handle, row uint32 }{
		"empty slot":             {0x00000000, 0xFFFFFFFF},
		"cleared slot":           {0xFFFFFFFF, 0xFFFFFFFF},
		"another row":            {0xB0009999, removeTestEquipIndexBase + ownedContainerTestCommonIndex + 1},
		"another handle":         {0xB0009999, removeTestEquipIndexBase + 99},
		"shared handle, own row": {ownedContainerTestHandle, removeTestEquipIndexBase + 99},
		"row below the base":     {ownedContainerTestHandle, removeTestEquipIndexBase - 1},
	}

	for name, testCase := range cases {
		for _, reference := range removeTestReferences(PlatformPC) {
			t.Run(name+" in "+reference.name, func(t *testing.T) {
				engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)
				removeTestPut(t, engine, saveSessionID, reference.handleAt, testCase.handle)
				removeTestPut(t, engine, saveSessionID, reference.rowAt, testCase.row)

				id := removeTestCommonID(t, engine, saveSessionID, ownedContainerInventory)
				if _, err := engine.RemoveOwnedItem(
					saveSessionID, ownedContainerTestSlot, id, "0"); err != nil {
					t.Fatalf("RemoveOwnedItem: %v", err)
				}
				removeTestWantCleared(t, engine, saveSessionID, ownedContainerInventory,
					removeTestRecordAt(PlatformPC, ownedContainerInventory,
						InventorySectionCommon, ownedContainerTestCommonIndex),
					ownedContainerTestCommonIndex)
			})
		}
	}
}

// removeTestSecondCommonRow is the second InventoryHeld common row the
// shared-handle tests add. It carries the same handle as the addressed row,
// which is exactly the layout `docs/owned-item-identity.md` L1 records as legal.
const removeTestSecondCommonRow = ownedContainerTestCommonIndex + 4

// Two InventoryHeld common rows may legitimately share one GaItem handle, and
// only the physical row separates the two instances. A reference to one of them
// blocks that one and only that one: the other row is removable while the
// reference stands.
func TestRemoveOwnedItemSeparatesTwoRowsSharingOneHandle(t *testing.T) {
	cases := map[string]struct {
		referencedRow int
		removedRow    int
		wantRemoved   bool
	}{
		"the unreferenced row is removed": {
			ownedContainerTestCommonIndex, removeTestSecondCommonRow, true},
		"the referenced row is refused": {
			ownedContainerTestCommonIndex, ownedContainerTestCommonIndex, false},
	}

	for name, testCase := range cases {
		for _, platform := range []Platform{PlatformPC, PlatformPS4} {
			t.Run(name+" on "+string(platform), func(t *testing.T) {
				engine, saveSessionID := loadOwnedItemContainers(t, platform)
				// The second row is written before the container is ever read, so both
				// rows are minted in the same revision.
				secondAt := removeTestRecordAt(platform, ownedContainerInventory,
					InventorySectionCommon, removeTestSecondCommonRow)
				removeTestPut(t, engine, saveSessionID, secondAt, ownedContainerTestHandle)
				removeTestPut(t, engine, saveSessionID, secondAt+4, ownedContainerTestRawQuantity)
				removeTestPut(t, engine, saveSessionID, secondAt+8, ownedContainerTestAcquisition)

				reference := removeTestReferences(platform)[0]
				removeTestPut(t, engine, saveSessionID, reference.handleAt, ownedContainerTestHandle)
				removeTestPut(t, engine, saveSessionID, reference.rowAt,
					removeTestEquipIndexBase+uint32(testCase.referencedRow))

				id := removeTestIdentities(t, engine, saveSessionID, ownedContainerInventory,
					ownedContainerTestSlot)["common#"+strconv.Itoa(testCase.removedRow)]
				if id == "" {
					t.Fatalf("the container read never identified common#%d", testCase.removedRow)
				}

				_, err := engine.RemoveOwnedItem(saveSessionID, ownedContainerTestSlot, id, "0")
				if testCase.wantRemoved {
					if err != nil {
						t.Fatalf("RemoveOwnedItem: %v", err)
					}
					removeTestWantCleared(t, engine, saveSessionID, ownedContainerInventory,
						secondAt, removeTestSecondCommonRow)
					// The referenced row is a different instance and is left alone.
					removeTestWantRow(t, engine, saveSessionID,
						removeTestRecordAt(platform, ownedContainerInventory,
							InventorySectionCommon, ownedContainerTestCommonIndex),
						ownedContainerTestRawQuantity)
					return
				}
				if err == nil || !strings.Contains(err.Error(), "unequip it first") {
					t.Fatalf("error = %v, want the referenced row to be refused", err)
				}
				removeTestWantRow(t, engine, saveSessionID,
					removeTestRecordAt(platform, ownedContainerInventory,
						InventorySectionCommon, ownedContainerTestCommonIndex),
					ownedContainerTestRawQuantity)
				removeTestWantRow(t, engine, saveSessionID, secondAt, ownedContainerTestRawQuantity)
			})
		}
	}
}

// The row fields of all three structures are counted in the InventoryHeld common
// section, so an Inventory key record and both Storage sections can never be
// named by one of them. Sharing the referenced handle may not make such a record
// look equipped: the guard reads nothing at all for them.
func TestRemoveOwnedItemNeverTreatsOtherSectionsAsReferenced(t *testing.T) {
	cases := map[string]struct {
		container string
		section   string
		row       string
		index     int
	}{
		"inventory key": {ownedContainerInventory, InventorySectionKey,
			"key#" + strconv.Itoa(ownedContainerTestKeyIndex), ownedContainerTestKeyIndex},
		"storage common": {ownedContainerStorage, StorageSectionCommon,
			"common#" + strconv.Itoa(ownedContainerTestCommonIndex), ownedContainerTestCommonIndex},
	}

	for name, testCase := range cases {
		for _, platform := range []Platform{PlatformPC, PlatformPS4} {
			t.Run(name+" on "+string(platform), func(t *testing.T) {
				engine, saveSessionID := loadOwnedItemContainers(t, platform)
				// Every reference structure carries a genuine reference to the
				// Inventory common row, which shares its handle with the record below.
				for _, reference := range removeTestReferences(platform) {
					removeTestPut(t, engine, saveSessionID, reference.handleAt, ownedContainerTestHandle)
					removeTestPut(t, engine, saveSessionID, reference.rowAt,
						removeTestEquipIndexBase+ownedContainerTestCommonIndex)
				}

				id := removeTestIdentities(t, engine, saveSessionID, testCase.container,
					ownedContainerTestSlot)[testCase.row]
				if id == "" {
					t.Fatalf("the container read never identified %s", testCase.row)
				}
				if _, err := engine.RemoveOwnedItem(
					saveSessionID, ownedContainerTestSlot, id, "0"); err != nil {
					t.Fatalf("RemoveOwnedItem: %v", err)
				}
				removeTestWantCleared(t, engine, saveSessionID, testCase.container,
					removeTestRecordAt(platform, testCase.container, testCase.section, testCase.index),
					testCase.index)
				// The referenced Inventory common row is a different instance and is
				// left exactly as it was.
				removeTestWantRow(t, engine, saveSessionID,
					removeTestRecordAt(platform, ownedContainerInventory,
						InventorySectionCommon, ownedContainerTestCommonIndex),
					ownedContainerTestRawQuantity)
			})
		}
	}
}

// A section count of 0 in a save that still holds the record is already
// inconsistent. The removal neither wraps it to 0xFFFFFFFF nor repairs the save
// nor refuses the request: it clears the record and leaves the count at 0.
func TestRemoveOwnedItemLeavesAZeroSectionCountAlone(t *testing.T) {
	engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)
	countAt := removeTestCountAt(PlatformPC, ownedContainerInventory, InventorySectionCommon)
	if got := removeTestCount(t, engine, saveSessionID, countAt); got != 0 {
		t.Fatalf("the fixture starts with a count of %d, want 0", got)
	}
	id := removeTestCommonID(t, engine, saveSessionID, ownedContainerInventory)

	if _, err := engine.RemoveOwnedItem(saveSessionID, ownedContainerTestSlot, id, "0"); err != nil {
		t.Fatalf("RemoveOwnedItem: %v", err)
	}
	if got := removeTestCount(t, engine, saveSessionID, countAt); got != 0 {
		t.Errorf("count = %d, want the unchanged 0", got)
	}
	clearedAt := removeTestRecordAt(PlatformPC, ownedContainerInventory,
		InventorySectionCommon, ownedContainerTestCommonIndex)
	removeTestWantCleared(t, engine, saveSessionID, ownedContainerInventory,
		clearedAt, ownedContainerTestCommonIndex)
}

func removeTestCount(t *testing.T, engine *Engine, saveSessionID string, at int64) uint32 {
	t.Helper()

	return binary.LittleEndian.Uint32(removeTestBytes(t, engine, saveSessionID, at, 4))
}

// Every rejection leaves the record, both counts, the revision, the
// unsaved-changes flag and the identity registry exactly as they were.
func TestRemoveOwnedItemRejectsWithoutChangingAnything(t *testing.T) {
	cases := map[string]struct {
		characterID      int
		ownedItemID      func(valid string) string
		expectedRevision string
	}{
		"empty identity":     {ownedContainerTestSlot, func(string) string { return "" }, "0"},
		"unknown identity":   {ownedContainerTestSlot, func(valid string) string { return valid + "-x" }, "0"},
		"fabricated prefix":  {ownedContainerTestSlot, func(string) string { return "oi-nobody-0-1" }, "0"},
		"another character":  {ownedContainerTestSlot + 1, func(valid string) string { return valid }, "0"},
		"malformed revision": {ownedContainerTestSlot, func(valid string) string { return valid }, "00"},
		"padded revision":    {ownedContainerTestSlot, func(valid string) string { return valid }, " 0"},
		"empty revision":     {ownedContainerTestSlot, func(valid string) string { return valid }, ""},
		"stale revision":     {ownedContainerTestSlot, func(valid string) string { return valid }, "7"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)
			countAt := removeTestCountAt(PlatformPC, ownedContainerInventory, InventorySectionCommon)
			removeTestPut(t, engine, saveSessionID, countAt, 2)
			id := removeTestCommonID(t, engine, saveSessionID, ownedContainerInventory)
			recordAt := removeTestRecordAt(PlatformPC, ownedContainerInventory,
				InventorySectionCommon, ownedContainerTestCommonIndex)

			result, err := engine.RemoveOwnedItem(saveSessionID, testCase.characterID,
				testCase.ownedItemID(id), testCase.expectedRevision)
			if err == nil {
				t.Fatalf("RemoveOwnedItem accepted %s: %+v", name, result)
			}
			if result != (RemoveOwnedItemResult{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}

			removeTestWantRow(t, engine, saveSessionID, recordAt, ownedContainerTestRawQuantity)
			if got := removeTestCount(t, engine, saveSessionID, countAt); got != 2 {
				t.Errorf("count = %d, want the unchanged 2", got)
			}
			revision, dirty := quantityTestSession(t, engine, saveSessionID)
			if revision != "0" {
				t.Errorf("revision = %q, want the unchanged \"0\"", revision)
			}
			if dirty {
				t.Error("a rejected removal reported unsaved changes")
			}
			// The identity of a rejected removal stays valid, because nothing it
			// addressed has changed.
			if _, err := engine.GetOwnedItem(saveSessionID, ownedContainerTestSlot, id); err != nil {
				t.Errorf("the ownedItemID stopped resolving after a rejection: %v", err)
			}
		})
	}
}

// A token is bound to the session that minted it. One from another session is
// unknown here, whatever it addresses there.
func TestRemoveOwnedItemRejectsAForeignSessionIdentity(t *testing.T) {
	engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)
	foreignEngine, foreignSession := loadOwnedItemContainers(t, PlatformPC)
	foreignID := removeTestCommonID(t, foreignEngine, foreignSession, ownedContainerInventory)
	recordAt := removeTestRecordAt(PlatformPC, ownedContainerInventory,
		InventorySectionCommon, ownedContainerTestCommonIndex)
	// The token is materialised in the target session too, so the rejection is
	// about the token's origin and not about an unread container.
	removeTestCommonID(t, engine, saveSessionID, ownedContainerInventory)

	_, err := engine.RemoveOwnedItem(saveSessionID, ownedContainerTestSlot, foreignID, "0")
	if !errors.Is(err, errUnknownOwnedItemID) {
		t.Fatalf("error = %v, want errUnknownOwnedItemID", err)
	}
	removeTestWantRow(t, engine, saveSessionID, recordAt, ownedContainerTestRawQuantity)
	if revision, dirty := quantityTestSession(t, engine, saveSessionID); revision != "0" || dirty {
		t.Errorf("revision %q, unsavedChanges %v; want \"0\" and false", revision, dirty)
	}
}

// A handle no GaItem record and no documented fallback can resolve is
// undecodable data, and undecodable data is never turned into a deletion.
func TestRemoveOwnedItemRejectsAnUnresolvableHandle(t *testing.T) {
	engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)
	recordAt := removeTestRecordAt(PlatformPC, ownedContainerInventory,
		InventorySectionCommon, ownedContainerTestCommonIndex)
	// A weapon handle without a GaItem record has no fallback, unlike the goods
	// handle the fixture normally carries.
	removeTestPut(t, engine, saveSessionID, recordAt, 0x80000123)
	id := removeTestCommonID(t, engine, saveSessionID, ownedContainerInventory)

	_, err := engine.RemoveOwnedItem(saveSessionID, ownedContainerTestSlot, id, "0")
	if err == nil || !strings.Contains(err.Error(), "has no record") {
		t.Fatalf("error = %v, want the unresolvable handle to be named", err)
	}
	if raw := removeTestBytes(t, engine, saveSessionID, recordAt, 4); binary.LittleEndian.Uint32(
		raw) != 0x80000123 {
		t.Errorf("handle = % X, want the unchanged 0x80000123", raw)
	}
	if revision, dirty := quantityTestSession(t, engine, saveSessionID); revision != "0" || dirty {
		t.Errorf("revision %q, unsavedChanges %v; want \"0\" and false", revision, dirty)
	}
}

// An inactive or residual slot mints no identity at all, so nothing in it can
// be addressed and nothing in it is searched or read.
func TestRemoveOwnedItemCannotReachAnInactiveSlot(t *testing.T) {
	engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)
	const residualSlot = ownedContainerTestSlot + 1

	inventory, err := engine.GetInventory(saveSessionID, residualSlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory of the residual slot: %v", err)
	}
	if inventory.Active || len(inventory.Records) != 0 {
		t.Fatalf("slot %d reports active %v with %d records, want an inactive empty slot",
			residualSlot, inventory.Active, len(inventory.Records))
	}

	id := removeTestCommonID(t, engine, saveSessionID, ownedContainerInventory)
	// The identity of the active slot cannot be redirected into the inactive one.
	if _, err := engine.RemoveOwnedItem(saveSessionID, residualSlot, id, "0"); err == nil {
		t.Fatal("RemoveOwnedItem accepted an identity of another slot")
	}
	if _, err := engine.RemoveOwnedItem(
		saveSessionID, residualSlot, "oi-anything-0-1", "0"); !errors.Is(err, errUnknownOwnedItemID) {
		t.Fatalf("error = %v, want errUnknownOwnedItemID", err)
	}
	if revision, dirty := quantityTestSession(t, engine, saveSessionID); revision != "0" || dirty {
		t.Errorf("revision %q, unsavedChanges %v; want \"0\" and false", revision, dirty)
	}
}

// The identity used once is gone afterwards, and a second removal with it is a
// stale identity rather than a second deletion.
func TestRemoveOwnedItemRetiresTheIdentityItUsed(t *testing.T) {
	engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)
	id := removeTestCommonID(t, engine, saveSessionID, ownedContainerInventory)

	if _, err := engine.RemoveOwnedItem(saveSessionID, ownedContainerTestSlot, id, "0"); err != nil {
		t.Fatalf("RemoveOwnedItem: %v", err)
	}
	if _, err := engine.RemoveOwnedItem(
		saveSessionID, ownedContainerTestSlot, id, "1"); !errors.Is(err, errStaleOwnedItemID) {
		t.Fatalf("second removal error = %v, want errStaleOwnedItemID", err)
	}
	if revision, _ := quantityTestSession(t, engine, saveSessionID); revision != "1" {
		t.Errorf("revision = %q, want \"1\": the rejected second removal may not advance it", revision)
	}
}

// A revision that was current once is not current after a commit, and the
// rejection names the revision the caller has to re-read.
func TestRemoveOwnedItemRejectsARevisionThatHasMovedOn(t *testing.T) {
	engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)
	id := removeTestCommonID(t, engine, saveSessionID, ownedContainerInventory)

	if _, err := engine.RemoveOwnedItem(saveSessionID, ownedContainerTestSlot, id, "0"); err != nil {
		t.Fatalf("first RemoveOwnedItem: %v", err)
	}
	keyID := removeTestIdentities(t, engine, saveSessionID, ownedContainerInventory,
		ownedContainerTestSlot)["key#"+strconv.Itoa(ownedContainerTestKeyIndex)]
	if keyID == "" {
		t.Fatal("the container read never identified the key row")
	}

	// The identity is current, only the revision is not.
	_, err := engine.RemoveOwnedItem(saveSessionID, ownedContainerTestSlot, keyID, "0")
	want := `expectedRevision "0" does not match the current saveRevision "1"`
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	keptAt := removeTestRecordAt(PlatformPC, ownedContainerInventory,
		InventorySectionKey, ownedContainerTestKeyIndex)
	removeTestWantRow(t, engine, saveSessionID, keptAt, ownedContainerTestRawKeyQty)
}

// The removal changes the private snapshot only. The file the session was loaded
// from stays byte-identical until a separate WriteSave persists the change into
// its own target.
func TestRemoveOwnedItemLeavesTheSourceFileToWriteSave(t *testing.T) {
	source := writeOwnedItemContainerFixture(t, PlatformPC)
	before, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read the fixture: %v", err)
	}

	engine := New()
	loaded, err := engine.LoadSave(source, string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	id := removeTestCommonID(t, engine, loaded.SaveSessionID, ownedContainerInventory)
	if _, err := engine.RemoveOwnedItem(
		loaded.SaveSessionID, ownedContainerTestSlot, id, "0"); err != nil {
		t.Fatalf("RemoveOwnedItem: %v", err)
	}

	after, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("re-read the fixture: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("the removal changed the file the session was loaded from")
	}

	target := filepath.Join(t.TempDir(), "written.sl2")
	if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
		t.Fatalf("WriteSave: %v", err)
	}
	if unchanged, err := os.ReadFile(source); err != nil || !bytes.Equal(before, unchanged) {
		t.Fatalf("WriteSave changed the source file (err %v)", err)
	}

	// The written save carries the removal: reloading it reports one row less.
	reloadEngine := New()
	reloaded, err := reloadEngine.LoadSave(target, string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave of the written save: %v", err)
	}
	rows := removeTestIdentities(t, reloadEngine, reloaded.SaveSessionID,
		ownedContainerInventory, ownedContainerTestSlot)
	if len(rows) != 1 {
		t.Fatalf("the written save holds %d inventory rows, want 1", len(rows))
	}
	if _, kept := rows["key#"+strconv.Itoa(ownedContainerTestKeyIndex)]; !kept {
		t.Errorf("the written save lost the key row instead of the common row: %v", rows)
	}
}

// removeTestPrevious is the twelve bytes of one occupied row, used by the write
// step's own unit tests: handle 0xB000272E, quantity 3, acquisition index 7.
var removeTestPrevious = []byte{0x2E, 0x27, 0x00, 0xB0, 0x03, 0x00, 0x00, 0x00, 0x07, 0x00, 0x00, 0x00}

// The write step is the only part of the removal that can fail after a byte has
// changed. A count header outside the snapshot proves the first rollback: the
// codec rejects the whole range before its first byte, so the count never moved
// and only the record has to come back.
func TestWriteOwnedItemRemovalRollsBackAFailedCountWrite(t *testing.T) {
	snapshot := &codec{data: append([]byte(nil), removeTestPrevious...)}

	err := writeOwnedItemRemoval(snapshot, 0, make([]byte, len(removeTestPrevious)),
		snapshot.length(), 4)
	if err == nil {
		t.Fatal("writeOwnedItemRemoval accepted a count outside the snapshot")
	}
	if !strings.Contains(err.Error(), "the record is unchanged") {
		t.Errorf("error = %v, want it to state that the record is unchanged", err)
	}
	if !bytes.Equal(snapshot.data, removeTestPrevious) {
		t.Errorf("record = % X, want the restored % X", snapshot.data, removeTestPrevious)
	}

	// The same call with a valid count position clears the record and lowers the
	// count, so the failure above is about the count write and nothing else.
	full := &codec{data: append(append([]byte(nil), removeTestPrevious...), 4, 0, 0, 0)}
	if err := writeOwnedItemRemoval(full, 0, make([]byte, len(removeTestPrevious)),
		int64(len(removeTestPrevious)), 4); err != nil {
		t.Fatalf("writeOwnedItemRemoval: %v", err)
	}
	if !bytes.Equal(full.data[:len(removeTestPrevious)], make([]byte, len(removeTestPrevious))) {
		t.Errorf("record = % X, want it cleared", full.data[:len(removeTestPrevious)])
	}
	if count := binary.LittleEndian.Uint32(full.data[len(removeTestPrevious):]); count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

// A failure that happens after the count was already lowered has to restore both
// writes, not just the record. The count header is placed inside the record
// here, so the accepted count write clobbers the cleared row and the
// verification behind it fails with both values already changed.
func TestWriteOwnedItemRemovalRollsBackTheCountItAlreadyLowered(t *testing.T) {
	snapshot := &codec{data: append([]byte(nil), removeTestPrevious...)}
	// The third field of the row doubles as the count header, so its stored value
	// — 7 — is the previous count the rollback has to put back.
	const countAt, count = int64(8), uint32(7)
	cleared := make([]byte, len(removeTestPrevious))
	binary.LittleEndian.PutUint32(cleared[8:], 1)

	err := writeOwnedItemRemoval(snapshot, 0, cleared, countAt, count)
	if err == nil {
		t.Fatal("writeOwnedItemRemoval reported success although the record was overwritten")
	}
	if !strings.Contains(err.Error(), "is unchanged") {
		t.Errorf("error = %v, want it to state that the record is unchanged", err)
	}
	if !bytes.Equal(snapshot.data, removeTestPrevious) {
		t.Errorf("snapshot = % X, want every changed byte restored to % X",
			snapshot.data, removeTestPrevious)
	}
	if stored := binary.LittleEndian.Uint32(snapshot.data[countAt:]); stored != count {
		t.Errorf("count = %d, want the restored %d", stored, count)
	}
}

// The InventoryHeld key section keeps its key_count header, so the write step
// takes no count position at all there and touches nothing but the row.
func TestWriteOwnedItemRemovalSkipsAnUnmaintainedCount(t *testing.T) {
	snapshot := &codec{data: append(append([]byte(nil), removeTestPrevious...), 4, 0, 0, 0)}
	cleared := make([]byte, len(removeTestPrevious))
	binary.LittleEndian.PutUint32(cleared[8:], 2)

	if err := writeOwnedItemRemoval(snapshot, 0, cleared, -1, 0); err != nil {
		t.Fatalf("writeOwnedItemRemoval: %v", err)
	}
	if !bytes.Equal(snapshot.data[:len(cleared)], cleared) {
		t.Errorf("record = % X, want % X", snapshot.data[:len(cleared)], cleared)
	}
	if count := binary.LittleEndian.Uint32(snapshot.data[len(cleared):]); count != 4 {
		t.Errorf("the header behind the row = %d, want the untouched 4", count)
	}
}

// The registry, the revision, the dirty flag and the snapshot are shared mutable
// state behind one mutex, so readers, the quantity setter and the removal must
// stay serialised and must never deadlock on each other.
func TestRemoveOwnedItemIsRaceFreeWithTheReaders(t *testing.T) {
	engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)

	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for round := 0; round < 25; round++ {
				inventory, err := engine.GetInventory(saveSessionID, ownedContainerTestSlot, "", 0, 0)
				if err != nil {
					t.Errorf("GetInventory: %v", err)
					return
				}
				if _, err := engine.GetStorage(saveSessionID, ownedContainerTestSlot, "", 0, 0); err != nil {
					t.Errorf("GetStorage: %v", err)
					return
				}
				if len(inventory.Records) == 0 {
					continue
				}
				id := inventory.Records[0].OwnedItemID
				// A concurrent commit may retire this identity or move the revision on
				// between the read and the call below, so the call is allowed to fail.
				// What may not happen is a race, a deadlock or a success that reports
				// something other than what was asked for.
				result, err := engine.RemoveOwnedItem(
					saveSessionID, ownedContainerTestSlot, id, inventory.SaveRevision)
				if err != nil {
					continue
				}
				if result.OwnedItemID != id || result.GameID != ownedContainerTestGameID {
					t.Errorf("result = %+v, want the removal of %q", result, id)
					return
				}
			}
		}()
	}
	workers.Wait()

	revision, dirty := quantityTestSession(t, engine, saveSessionID)
	if revision == "0" || !dirty {
		t.Fatalf("after the concurrent run revision = %q, unsavedChanges = %v; want a committed removal",
			revision, dirty)
	}
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	session := engine.sessions[saveSessionID].session
	if len(session.ownedByID) != len(session.ownedByLocator) {
		t.Fatalf("registry directions diverged: %d tokens, %d locators",
			len(session.ownedByID), len(session.ownedByLocator))
	}
}
