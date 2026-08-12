package saveengine

import (
	"errors"
	"strconv"
	"sync"
	"testing"
)

// loadOwnedItemContainers loads the one fixture that carries both containers of
// one slot and returns the engine and the session identifier.
func loadOwnedItemContainers(t *testing.T, platform Platform) (*Engine, string) {
	t.Helper()

	engine := New()
	loaded, err := engine.LoadSave(writeOwnedItemContainerFixture(t, platform), string(platform))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

// ownedItemWant is the record both containers of the fixture carry at the two
// coordinates, so only the container itself may separate the two results.
func ownedItemWant(saveSessionID, container, section string, index int, id string) OwnedItem {
	return OwnedItem{
		SaveSessionID:    saveSessionID,
		SaveRevision:     "0",
		OwnedItemID:      id,
		CharacterID:      ownedContainerTestSlot,
		Container:        container,
		ContainerSection: section,
		PhysicalIndex:    index,
		GaItemHandle:     ownedContainerTestHandle,
		Quantity:         ownedContainerTestQuantity,
		AcquisitionIndex: ownedContainerTestAcquisition,
	}
}

// A token of either container resolves back to exactly the record it was minted
// for, on both platforms. The two containers hold the same handle, quantity and
// acquisition index at the same coordinates, so a getter that resolved into the
// wrong container would still have to report the wrong container name.
func TestGetOwnedItemResolvesBothContainersOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine, saveSessionID := loadOwnedItemContainers(t, platform)

			inventory, err := engine.GetInventory(saveSessionID, ownedContainerTestSlot, "", 0, 0)
			if err != nil {
				t.Fatalf("GetInventory: %v", err)
			}
			storage, err := engine.GetStorage(saveSessionID, ownedContainerTestSlot, "", 0, 0)
			if err != nil {
				t.Fatalf("GetStorage: %v", err)
			}
			inventoryIDs := inventoryTestIdentitiesByRow(t, inventory.Records)
			storageIDs := storageTestIdentitiesByRow(t, storage.Records)

			cases := map[string]struct {
				container string
				section   string
				index     int
				id        string
			}{
				"inventory common": {ownedContainerInventory, InventorySectionCommon,
					ownedContainerTestCommonIndex,
					inventoryIDs[InventorySectionCommon+"#"+strconv.Itoa(ownedContainerTestCommonIndex)]},
				"inventory key": {ownedContainerInventory, InventorySectionKey,
					ownedContainerTestKeyIndex,
					inventoryIDs[InventorySectionKey+"#"+strconv.Itoa(ownedContainerTestKeyIndex)]},
				"storage common": {ownedContainerStorage, StorageSectionCommon,
					ownedContainerTestCommonIndex,
					storageIDs[StorageSectionCommon+"#"+strconv.Itoa(ownedContainerTestCommonIndex)]},
				"storage key": {ownedContainerStorage, StorageSectionKey,
					ownedContainerTestKeyIndex,
					storageIDs[StorageSectionKey+"#"+strconv.Itoa(ownedContainerTestKeyIndex)]},
			}
			for name, testCase := range cases {
				t.Run(name, func(t *testing.T) {
					if testCase.id == "" {
						t.Fatal("the container read never identified this row")
					}
					item, err := engine.GetOwnedItem(saveSessionID, ownedContainerTestSlot, testCase.id)
					if err != nil {
						t.Fatalf("GetOwnedItem: %v", err)
					}
					want := ownedItemWant(saveSessionID,
						testCase.container, testCase.section, testCase.index, testCase.id)
					if item != want {
						t.Errorf("item = %+v, want %+v", item, want)
					}
				})
			}
		})
	}
}

// Resolving a Storage token may not materialise the Inventory identities of the
// same slot: the registry stays lazy per container.
func TestGetOwnedItemKeepsMaterialisationLazyPerContainer(t *testing.T) {
	engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)

	mintedCount := func() int {
		engine.mutex.Lock()
		defer engine.mutex.Unlock()
		return len(engine.sessions[saveSessionID].session.ownedByID)
	}

	storage, err := engine.GetStorage(saveSessionID, ownedContainerTestSlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	if minted := mintedCount(); minted != 2 {
		t.Fatalf("a Storage-only read minted %d identities, want 2", minted)
	}

	if _, err := engine.GetOwnedItem(saveSessionID, ownedContainerTestSlot,
		storage.Records[0].OwnedItemID); err != nil {
		t.Fatalf("GetOwnedItem: %v", err)
	}
	if minted := mintedCount(); minted != 2 {
		t.Fatalf("resolving a Storage token minted %d identities in total, want 2", minted)
	}
}

// The token is the only address. A record that is no longer where its token was
// minted for is a hard error, never a neighbouring row and never a zero-value
// success.
func TestGetOwnedItemRejectsAMissingPhysicalRecord(t *testing.T) {
	engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)

	// The fixture leaves this row at the native empty sentinel, so no read ever
	// mints it. Minting it directly is the smallest way to express a token whose
	// physical record does not exist.
	engine.mutex.Lock()
	orphan := engine.sessions[saveSessionID].session.mintOwnedItemID(
		inventoryLocator(ownedContainerTestSlot, InventorySectionCommon, 0x400))
	engine.mutex.Unlock()

	item, err := engine.GetOwnedItem(saveSessionID, ownedContainerTestSlot, orphan)
	if err == nil {
		t.Fatalf("GetOwnedItem accepted a token without a record: %+v", item)
	}
	if err.Error() != "ownedItemID "+strconv.Quote(orphan)+" no longer addresses a record of character 2" {
		t.Errorf("error = %q, want the missing-record error", err)
	}
	if item != (OwnedItem{}) {
		t.Errorf("item = %+v, want the zero value", item)
	}
}

func TestGetOwnedItemRejectsInvalidRequests(t *testing.T) {
	engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)
	inventory, err := engine.GetInventory(saveSessionID, ownedContainerTestSlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	valid := inventory.Records[0].OwnedItemID

	// A second session on its own file mints its own tokens; neither may be
	// resolved by the other.
	otherEngine, otherSessionID := loadOwnedItemContainers(t, PlatformPC)
	otherInventory, err := otherEngine.GetInventory(otherSessionID, ownedContainerTestSlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory of the second session: %v", err)
	}
	foreign := otherInventory.Records[0].OwnedItemID

	cases := map[string]struct {
		saveSessionID string
		characterID   int
		ownedItemID   string
		want          string
	}{
		"empty session":   {"", ownedContainerTestSlot, valid, "saveSessionID is required"},
		"unknown session": {"missing", ownedContainerTestSlot, valid, `unknown save session "missing"`},
		"characterID -1":  {saveSessionID, -1, valid, "characterID -1 is outside the range 0..9"},
		"characterID 10":  {saveSessionID, 10, valid, "characterID 10 is outside the range 0..9"},
		"empty ownedItemID": {saveSessionID, ownedContainerTestSlot, "",
			"ownedItemID is required"},
		"unknown ownedItemID": {saveSessionID, ownedContainerTestSlot, "whatever",
			"unknown ownedItemID"},
		"ownedItemID of another session": {saveSessionID, ownedContainerTestSlot, foreign,
			"unknown ownedItemID"},
		"ownedItemID of another character": {saveSessionID, 4, valid,
			"ownedItemID belongs to character 2, not to character 4"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			item, err := engine.GetOwnedItem(testCase.saveSessionID, testCase.characterID, testCase.ownedItemID)
			if err == nil {
				t.Fatalf("GetOwnedItem accepted %s: %+v", name, item)
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if item != (OwnedItem{}) {
				t.Errorf("item = %+v, want the zero value", item)
			}
		})
	}
}

// A token of the previous revision is rejected with an error distinguishable
// from "unknown", because the remedy is to re-read the container.
func TestGetOwnedItemRejectsAStaleToken(t *testing.T) {
	engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)
	inventory, err := engine.GetInventory(saveSessionID, ownedContainerTestSlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	stale := inventory.Records[0].OwnedItemID

	if err := engine.commitRevision(saveSessionID, func() error { return nil }); err != nil {
		t.Fatalf("commitRevision: %v", err)
	}

	item, err := engine.GetOwnedItem(saveSessionID, ownedContainerTestSlot, stale)
	if !errors.Is(err, errStaleOwnedItemID) {
		t.Fatalf("GetOwnedItem(stale) error = %v, want errStaleOwnedItemID", err)
	}
	if item != (OwnedItem{}) {
		t.Errorf("item = %+v, want the zero value", item)
	}

	// The next read of the new revision mints a fresh token for the same record.
	after, err := engine.GetInventory(saveSessionID, ownedContainerTestSlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory after the commit: %v", err)
	}
	fresh := after.Records[0].OwnedItemID
	if fresh == stale {
		t.Fatalf("the retired token %q was minted again", stale)
	}
	resolved, err := engine.GetOwnedItem(saveSessionID, ownedContainerTestSlot, fresh)
	if err != nil {
		t.Fatalf("GetOwnedItem(fresh): %v", err)
	}
	if resolved.SaveRevision != "1" {
		t.Errorf("saveRevision after one commit = %q, want \"1\"", resolved.SaveRevision)
	}
}

// The registry and the counter are shared mutable state behind Engine.mutex, so
// concurrent readers of one session must stay serialised.
func TestGetOwnedItemIsRaceFreeWithTheContainerReaders(t *testing.T) {
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
				storage, err := engine.GetStorage(saveSessionID, ownedContainerTestSlot, "", 0, 0)
				if err != nil {
					t.Errorf("GetStorage: %v", err)
					return
				}
				for _, id := range []string{inventory.Records[0].OwnedItemID, storage.Records[0].OwnedItemID} {
					item, err := engine.GetOwnedItem(saveSessionID, ownedContainerTestSlot, id)
					if err != nil {
						t.Errorf("GetOwnedItem: %v", err)
						return
					}
					if item.OwnedItemID != id {
						t.Errorf("resolved %q as %q", id, item.OwnedItemID)
						return
					}
				}
			}
		}()
	}
	workers.Wait()
}
