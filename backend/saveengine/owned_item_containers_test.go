package saveengine

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// One character holds InventoryHeld and the Storage Box in the same slot, and
// the two are different physical containers. The fixture below is the only one
// that carries both sections at once, so the getter layer can prove that a row
// of one container never borrows the identity of the row at the same
// coordinates in the other, and that reading one container mints nothing for the
// other.
const (
	ownedContainerTestSlot = 2

	// The anchor sits behind a full GaItem table, so the same fixture also serves
	// every reader that resolves a handle to a game ID. The table starts at
	// offset 0x20 of the slot data and ends immediately before the anchor, and
	// 5120 all-zero records are eight bytes each, so this is the first position
	// the anchor may occupy.
	ownedContainerTestAnchorAt = 0x20 + 5120*8

	// ownedContainerTestSlotVersion is above the legacy record-count break, so the
	// table is read with the current record count.
	ownedContainerTestSlotVersion = 83

	// The two rows every container in this fixture carries. They deliberately
	// hold the same coordinates, the same handle, the same effective quantity and
	// the same acquisition index in both containers, so nothing but the container
	// itself can separate their identities. The key row additionally carries the
	// native high bit on its stored quantity, which is not part of the count and
	// therefore masks off to the same value.
	ownedContainerTestCommonIndex = 1
	ownedContainerTestKeyIndex    = 2
	ownedContainerTestHandle      = 0xB000272E
	ownedContainerTestRawQuantity = 3
	ownedContainerTestRawKeyQty   = 0x80000000 | ownedContainerTestRawQuantity
	ownedContainerTestQuantity    = 3
	ownedContainerTestAcquisition = 7

	// ownedContainerTestGameID is what the fixture's handle resolves to. The
	// table carries no record for it, so a goods handle resolves through the
	// documented fallback.
	ownedContainerTestGameID = 0x4000272E
)

// writeOwnedItemContainerFixture builds a synthetic save of the given platform
// whose single active slot carries an InventoryHeld section and a Storage Box
// behind an empty declared projectile list. Both getters measure from the same
// confirmed anchor, so one anchor serves both. The two platforms differ only in
// the container around the slot, which is exactly what the platform split of the
// two readers claims.
func writeOwnedItemContainerFixture(t *testing.T, platform Platform) string {
	t.Helper()

	var data []byte
	var userData10Base, slotBase int64
	switch platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = pcUserData10DataOffset
		slotBase = inventoryTestPCSlotDataBase + ownedContainerTestSlot*inventoryTestPCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = ps4UserData10DataOffset
		slotBase = inventoryTestPS4SlotDataBase + ownedContainerTestSlot*inventoryTestPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", platform)
	}

	data[userData10Base+userData10ActiveFlagsOffset+ownedContainerTestSlot] = 1
	// The slot declares a version, so the GaItem table in front of the anchor is
	// read instead of rejected. Its records stay zeroed, which is how a handle
	// without an instance record is expressed.
	binary.LittleEndian.PutUint32(data[slotBase:], ownedContainerTestSlotVersion)
	copy(data[slotBase+ownedContainerTestAnchorAt:], inventoryTestAnchor)

	putRow := func(at int64, rawQuantity uint32) {
		binary.LittleEndian.PutUint32(data[slotBase+at:], ownedContainerTestHandle)
		binary.LittleEndian.PutUint32(data[slotBase+at+4:], rawQuantity)
		binary.LittleEndian.PutUint32(data[slotBase+at+8:], ownedContainerTestAcquisition)
	}

	// InventoryHeld starts at a constant distance behind the anchor.
	putRow(ownedContainerTestAnchorAt+inventoryTestCommonAt+
		ownedContainerTestCommonIndex*inventoryTestRecordSize, ownedContainerTestRawQuantity)
	putRow(ownedContainerTestAnchorAt+inventoryTestKeyAt+
		ownedContainerTestKeyIndex*inventoryTestRecordSize, ownedContainerTestRawKeyQty)

	// The Storage Box starts behind the declared projectile list, which this
	// fixture leaves at the zero the container is already filled with.
	storageAt := int64(ownedContainerTestAnchorAt) +
		storageTestProjectileCountAt + 4 + storageTestBlocksBefore
	putRow(storageAt+storageTestCommonAt+ownedContainerTestCommonIndex*storageTestRecordSize,
		ownedContainerTestRawQuantity)
	putRow(storageAt+storageTestKeyAt+ownedContainerTestKeyIndex*storageTestRecordSize,
		ownedContainerTestRawKeyQty)

	path := filepath.Join(t.TempDir(), "owned-item-containers.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestInventoryAndStorageNeverShareAnIdentity(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeOwnedItemContainerFixture(t, PlatformPC), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	mintedCount := func() int {
		engine.mutex.Lock()
		defer engine.mutex.Unlock()
		return len(engine.sessions[loaded.SaveSessionID].session.ownedByID)
	}

	inventory, err := engine.GetInventory(loaded.SaveSessionID, ownedContainerTestSlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	inventoryIDs := inventoryTestIdentitiesByRow(t, inventory.Records)
	if len(inventoryIDs) != 2 {
		t.Fatalf("inventory identified %d records, want 2", len(inventoryIDs))
	}
	// Materialisation is lazy and per container: the Inventory read may not have
	// minted anything for the Storage Box behind it.
	if minted := mintedCount(); minted != 2 {
		t.Fatalf("an Inventory-only read minted %d identities, want 2", minted)
	}

	storage, err := engine.GetStorage(loaded.SaveSessionID, ownedContainerTestSlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	storageIDs := storageTestIdentitiesByRow(t, storage.Records)
	if len(storageIDs) != 2 {
		t.Fatalf("storage identified %d records, want 2", len(storageIDs))
	}
	if minted := mintedCount(); minted != 4 {
		t.Fatalf("both containers together minted %d identities, want 4", minted)
	}

	for row, id := range inventoryIDs {
		if storageIDs[row] == id {
			t.Errorf("Inventory and Storage share the identity %q at %s", id, row)
		}
	}
	if inventory.SaveRevision != storage.SaveRevision {
		t.Errorf("the two containers reported revisions %q and %q, want the same one",
			inventory.SaveRevision, storage.SaveRevision)
	}

	// The later Storage read may not have disturbed the identities the earlier
	// Inventory read issued.
	again, err := engine.GetInventory(loaded.SaveSessionID, ownedContainerTestSlot, "", 0, 0)
	if err != nil {
		t.Fatalf("second GetInventory: %v", err)
	}
	for row, id := range inventoryTestIdentitiesByRow(t, again.Records) {
		if id != inventoryIDs[row] {
			t.Errorf("after the Storage read %s is identified as %q, want %q", row, id, inventoryIDs[row])
		}
	}
}
