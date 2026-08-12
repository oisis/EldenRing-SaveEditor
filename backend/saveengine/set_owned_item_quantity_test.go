package saveengine

import (
	"errors"
	"strconv"
	"sync"
	"testing"
)

// The limits SaveEngine is told to enforce. They are supplied by the caller, so
// the tests state them literally and never expect the engine to invent one.
const (
	quantityTestMaxPerRecord      = 99
	quantityTestMaxContainerTotal = 600
)

// quantityTestOffset restates where the four quantity bytes of one row live,
// independently of the implementation, so a setter that wrote to a position
// derived some other way fails here. It is the literal sum of the fixture
// offsets the container tests already state.
func quantityTestOffset(platform Platform, container, section string, index int) int64 {
	var at int64
	switch platform {
	case PlatformPS4:
		at = inventoryTestPS4SlotDataBase + ownedContainerTestSlot*inventoryTestPS4SlotStride
	default:
		at = inventoryTestPCSlotDataBase + ownedContainerTestSlot*inventoryTestPCSlotStride
	}
	at += ownedContainerTestAnchorAt

	if container == ownedContainerStorage {
		at += storageTestProjectileCountAt + 4 + storageTestBlocksBefore
		if section == StorageSectionKey {
			at += storageTestKeyAt
		} else {
			at += storageTestCommonAt
		}
		return at + int64(index)*storageTestRecordSize + 4
	}
	if section == InventorySectionKey {
		at += inventoryTestKeyAt
	} else {
		at += inventoryTestCommonAt
	}
	return at + int64(index)*inventoryTestRecordSize + 4
}

// quantityTestRaw reads the stored quantity of one row exactly as it sits in the
// private snapshot, including the high bit the readers mask off.
func quantityTestRaw(t *testing.T, engine *Engine, saveSessionID string, at int64) uint32 {
	t.Helper()

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	raw, err := engine.sessions[saveSessionID].snapshot.uint32At(at)
	if err != nil {
		t.Fatalf("read the raw quantity at 0x%X: %v", at, err)
	}
	return raw
}

// quantityTestSession reports the revision and the unsaved-changes flag of one
// session, which together say whether a mutation committed.
func quantityTestSession(t *testing.T, engine *Engine, saveSessionID string) (string, bool) {
	t.Helper()

	info, err := engine.GetSessionInfo(saveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	return engine.sessions[saveSessionID].session.revisionString(), info.UnsavedChanges
}

// quantityTestTarget loads the shared two-container fixture and returns the
// engine, the session and the identity of the common row of one container. The
// container is read first, which is also what a caller of the future endpoint
// does to obtain an identity at all.
func quantityTestTarget(t *testing.T, platform Platform, container string) (*Engine, string, string) {
	t.Helper()

	engine, saveSessionID := loadOwnedItemContainers(t, platform)
	var identities map[string]string
	switch container {
	case ownedContainerStorage:
		storage, err := engine.GetStorage(saveSessionID, ownedContainerTestSlot, "", 0, 0)
		if err != nil {
			t.Fatalf("GetStorage: %v", err)
		}
		identities = storageTestIdentitiesByRow(t, storage.Records)
	default:
		inventory, err := engine.GetInventory(saveSessionID, ownedContainerTestSlot, "", 0, 0)
		if err != nil {
			t.Fatalf("GetInventory: %v", err)
		}
		identities = inventoryTestIdentitiesByRow(t, inventory.Records)
	}
	id := identities["common#"+strconv.Itoa(ownedContainerTestCommonIndex)]
	if id == "" {
		t.Fatal("the container read never identified the common row")
	}
	return engine, saveSessionID, id
}

// A freshly loaded session reports no unsaved changes; a committed mutation of
// its private snapshot reports them, and a rejected one does not.
func TestCommitRevisionOwnsTheUnsavedChangesFlag(t *testing.T) {
	engine, saveSessionID, _ := quantityTestTarget(t, PlatformPC, ownedContainerInventory)

	if _, dirty := quantityTestSession(t, engine, saveSessionID); dirty {
		t.Fatal("a freshly loaded session already reports unsaved changes")
	}

	rejected := errors.New("validation rejected the plan")
	if _, err := engine.commitRevision(
		saveSessionID, func(*loadedSave) error { return rejected }); !errors.Is(err, rejected) {
		t.Fatalf("commitRevision error = %v, want the commit error", err)
	}
	if _, dirty := quantityTestSession(t, engine, saveSessionID); dirty {
		t.Fatal("a failed commit reported unsaved changes")
	}

	if _, err := engine.commitRevision(saveSessionID, func(*loadedSave) error { return nil }); err != nil {
		t.Fatalf("commitRevision: %v", err)
	}
	if _, dirty := quantityTestSession(t, engine, saveSessionID); !dirty {
		t.Fatal("a successful commit reported no unsaved changes")
	}
}

// One committed mutation per container and per platform: the addressed row
// carries the new quantity, the row beside it is untouched, the revision
// advances by exactly one, the session becomes dirty, the identity used for the
// mutation is retired, and the next read returns the new value under a fresh
// identity.
func TestSetOwnedItemQuantityCommitsInBothContainers(t *testing.T) {
	cases := map[string]struct {
		platform      Platform
		container     string
		commonSection string
		keySection    string
	}{
		"inventory on PC": {PlatformPC, ownedContainerInventory,
			InventorySectionCommon, InventorySectionKey},
		"storage on PS4": {PlatformPS4, ownedContainerStorage,
			StorageSectionCommon, StorageSectionKey},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			engine, saveSessionID, id := quantityTestTarget(t, testCase.platform, testCase.container)
			const wanted = 17

			result, err := engine.SetOwnedItemQuantity(saveSessionID, ownedContainerTestSlot, id,
				wanted, "0", ownedContainerTestGameID,
				quantityTestMaxPerRecord, quantityTestMaxContainerTotal)
			if err != nil {
				t.Fatalf("SetOwnedItemQuantity: %v", err)
			}

			want := SetOwnedItemQuantityResult{
				SaveSessionID: saveSessionID,
				SaveRevision:  "1",
				OwnedItemID:   id,
				CharacterID:   ownedContainerTestSlot,
				Quantity:      wanted,
			}
			if result != want {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			// Exactly the addressed row changed, and it changed to the plain value:
			// the fixture stores this row without the high bit, so the write may not
			// invent one.
			changedAt := quantityTestOffset(testCase.platform, testCase.container,
				testCase.commonSection, ownedContainerTestCommonIndex)
			if raw := quantityTestRaw(t, engine, saveSessionID, changedAt); raw != wanted {
				t.Errorf("raw quantity of the changed row = 0x%08X, want 0x%08X", raw, uint32(wanted))
			}
			untouchedAt := quantityTestOffset(testCase.platform, testCase.container,
				testCase.keySection, ownedContainerTestKeyIndex)
			if raw := quantityTestRaw(t, engine, saveSessionID, untouchedAt); raw != ownedContainerTestRawKeyQty {
				t.Errorf("raw quantity of the untouched row = 0x%08X, want 0x%08X",
					raw, uint32(ownedContainerTestRawKeyQty))
			}

			revision, dirty := quantityTestSession(t, engine, saveSessionID)
			if revision != "1" {
				t.Errorf("revision after one mutation = %q, want \"1\"", revision)
			}
			if !dirty {
				t.Error("a committed mutation left the session without unsaved changes")
			}

			// The identity the mutation was performed with is retired by the commit.
			if _, err := engine.GetOwnedItem(saveSessionID, ownedContainerTestSlot, id); !errors.Is(
				err, errStaleOwnedItemID) {
				t.Errorf("the used ownedItemID resolved with %v, want errStaleOwnedItemID", err)
			}

			fresh, freshID := quantityTestReadCommonRow(t, engine, saveSessionID, testCase.container)
			if fresh != wanted {
				t.Errorf("quantity after the mutation = %d, want %d", fresh, wanted)
			}
			if freshID == id {
				t.Errorf("the retired identity %q was minted again", id)
			}
		})
	}
}

// quantityTestReadCommonRow re-reads the mutated row and returns its quantity
// and its current identity.
func quantityTestReadCommonRow(
	t *testing.T, engine *Engine, saveSessionID, container string,
) (uint32, string) {
	t.Helper()

	if container == ownedContainerStorage {
		storage, err := engine.GetStorage(saveSessionID, ownedContainerTestSlot, StorageSectionCommon, 0, 0)
		if err != nil {
			t.Fatalf("GetStorage after the mutation: %v", err)
		}
		return storage.Records[0].Quantity, storage.Records[0].OwnedItemID
	}
	inventory, err := engine.GetInventory(saveSessionID, ownedContainerTestSlot, InventorySectionCommon, 0, 0)
	if err != nil {
		t.Fatalf("GetInventory after the mutation: %v", err)
	}
	return inventory.Records[0].Quantity, inventory.Records[0].OwnedItemID
}

// The high bit of a stored quantity is not part of the count and is not the
// writer's to decide: a row that carries it keeps it, and the count beside it is
// replaced.
func TestSetOwnedItemQuantityPreservesTheHighBit(t *testing.T) {
	engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)
	inventory, err := engine.GetInventory(saveSessionID, ownedContainerTestSlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	id := inventoryTestIdentitiesByRow(t, inventory.Records)["key#"+strconv.Itoa(ownedContainerTestKeyIndex)]
	if id == "" {
		t.Fatal("the container read never identified the key row")
	}

	const wanted = 9
	if _, err := engine.SetOwnedItemQuantity(saveSessionID, ownedContainerTestSlot, id,
		wanted, "0", ownedContainerTestGameID,
		quantityTestMaxPerRecord, quantityTestMaxContainerTotal); err != nil {
		t.Fatalf("SetOwnedItemQuantity: %v", err)
	}

	at := quantityTestOffset(PlatformPC, ownedContainerInventory,
		InventorySectionKey, ownedContainerTestKeyIndex)
	if raw := quantityTestRaw(t, engine, saveSessionID, at); raw != 0x80000000|wanted {
		t.Errorf("raw quantity = 0x%08X, want 0x%08X", raw, uint32(0x80000000|wanted))
	}
}

// Every rejection leaves the record, the revision, the unsaved-changes flag and
// the identity registry exactly as they were. The cases below each protect a
// different rule, so none of them is a variation of another.
func TestSetOwnedItemQuantityRejectsWithoutChangingAnything(t *testing.T) {
	// The fixture holds two rows of the same item in one container — the common
	// row and the key row, three each — so the container total is a real sum and
	// not a restatement of the single record limit.
	cases := map[string]struct {
		quantity          uint32
		expectedRevision  string
		expectedGameID    uint32
		maxPerRecord      uint32
		maxContainerTotal uint32
	}{
		"quantity zero": {0, "0", ownedContainerTestGameID,
			quantityTestMaxPerRecord, quantityTestMaxContainerTotal},
		"above the record limit": {quantityTestMaxPerRecord + 1, "0", ownedContainerTestGameID,
			quantityTestMaxPerRecord, quantityTestMaxContainerTotal},
		"above the 31-bit field": {0x80000001, "0", ownedContainerTestGameID,
			0xFFFFFFFF, 0xFFFFFFFF},
		// 8 fits the record limit, but the key row of the same item already holds
		// three, so the container would end up at 11 against a limit of 10.
		"above the container total": {8, "0", ownedContainerTestGameID, quantityTestMaxPerRecord, 10},
		"malformed revision": {5, "00", ownedContainerTestGameID,
			quantityTestMaxPerRecord, quantityTestMaxContainerTotal},
		"padded revision": {5, " 0", ownedContainerTestGameID,
			quantityTestMaxPerRecord, quantityTestMaxContainerTotal},
		"empty revision": {5, "", ownedContainerTestGameID,
			quantityTestMaxPerRecord, quantityTestMaxContainerTotal},
		"stale revision": {5, "7", ownedContainerTestGameID,
			quantityTestMaxPerRecord, quantityTestMaxContainerTotal},
		"another item": {5, "0", ownedContainerTestGameID + 1,
			quantityTestMaxPerRecord, quantityTestMaxContainerTotal},
		"zero limits": {5, "0", ownedContainerTestGameID, 0, 0},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			engine, saveSessionID, id := quantityTestTarget(t, PlatformPC, ownedContainerInventory)
			at := quantityTestOffset(PlatformPC, ownedContainerInventory,
				InventorySectionCommon, ownedContainerTestCommonIndex)

			result, err := engine.SetOwnedItemQuantity(saveSessionID, ownedContainerTestSlot, id,
				testCase.quantity, testCase.expectedRevision, testCase.expectedGameID,
				testCase.maxPerRecord, testCase.maxContainerTotal)
			if err == nil {
				t.Fatalf("SetOwnedItemQuantity accepted %s: %+v", name, result)
			}
			if result != (SetOwnedItemQuantityResult{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}

			if raw := quantityTestRaw(t, engine, saveSessionID, at); raw != ownedContainerTestRawQuantity {
				t.Errorf("raw quantity = 0x%08X, want the unchanged 0x%08X",
					raw, uint32(ownedContainerTestRawQuantity))
			}
			revision, dirty := quantityTestSession(t, engine, saveSessionID)
			if revision != "0" {
				t.Errorf("revision = %q, want the unchanged \"0\"", revision)
			}
			if dirty {
				t.Error("a rejected mutation reported unsaved changes")
			}
			// The identity of a rejected mutation stays valid, because nothing it
			// addressed has changed.
			if _, err := engine.GetOwnedItem(saveSessionID, ownedContainerTestSlot, id); err != nil {
				t.Errorf("the ownedItemID stopped resolving after a rejection: %v", err)
			}
		})
	}
}

// A revision that was current once is not current after a commit, and the
// rejection names the revision the caller has to re-read.
func TestSetOwnedItemQuantityRejectsARevisionThatHasMovedOn(t *testing.T) {
	engine, saveSessionID, id := quantityTestTarget(t, PlatformPC, ownedContainerInventory)

	if _, err := engine.SetOwnedItemQuantity(saveSessionID, ownedContainerTestSlot, id,
		5, "0", ownedContainerTestGameID,
		quantityTestMaxPerRecord, quantityTestMaxContainerTotal); err != nil {
		t.Fatalf("first SetOwnedItemQuantity: %v", err)
	}

	inventory, err := engine.GetInventory(saveSessionID, ownedContainerTestSlot, InventorySectionCommon, 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	fresh := inventory.Records[0].OwnedItemID

	// The identity is current, only the revision is not.
	_, err = engine.SetOwnedItemQuantity(saveSessionID, ownedContainerTestSlot, fresh,
		6, "0", ownedContainerTestGameID,
		quantityTestMaxPerRecord, quantityTestMaxContainerTotal)
	if err == nil {
		t.Fatal("SetOwnedItemQuantity accepted the previous revision")
	}
	want := `expectedRevision "0" does not match the current saveRevision "1"`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
	at := quantityTestOffset(PlatformPC, ownedContainerInventory,
		InventorySectionCommon, ownedContainerTestCommonIndex)
	if raw := quantityTestRaw(t, engine, saveSessionID, at); raw != 5 {
		t.Errorf("raw quantity = 0x%08X, want the unchanged 0x%08X", raw, uint32(5))
	}
}

// The registry, the revision, the dirty flag and the snapshot are shared mutable
// state behind one mutex, so readers and the mutation must stay serialised and
// must never deadlock on each other.
func TestSetOwnedItemQuantityIsRaceFreeWithTheReaders(t *testing.T) {
	engine, saveSessionID := loadOwnedItemContainers(t, PlatformPC)

	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func(worker int) {
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
				id := inventory.Records[0].OwnedItemID
				// A concurrent commit may retire this identity or move the revision
				// on between the read and either call below, so both are allowed to
				// fail. What may not happen is a race, a deadlock or a success that
				// reports something other than what was asked for.
				_, _ = engine.GetOwnedItem(saveSessionID, ownedContainerTestSlot, id)

				wanted := uint32(1 + worker)
				result, err := engine.SetOwnedItemQuantity(saveSessionID, ownedContainerTestSlot, id,
					wanted, inventory.SaveRevision, ownedContainerTestGameID,
					quantityTestMaxPerRecord, quantityTestMaxContainerTotal)
				if err != nil {
					continue
				}
				if result.Quantity != wanted || result.OwnedItemID != id {
					t.Errorf("result = %+v, want quantity %d of %q", result, wanted, id)
					return
				}
			}
		}(worker)
	}
	workers.Wait()

	revision, dirty := quantityTestSession(t, engine, saveSessionID)
	if revision == "0" || !dirty {
		t.Fatalf("after the concurrent run revision = %q, unsavedChanges = %v; want a committed mutation",
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
