package saveengine

import (
	"errors"
	"sync"
	"testing"
)

// loadOwnedItemSession loads a synthetic PC container and returns the engine,
// the identifier of the fresh session and the private session behind it. The
// session is reached directly because these tests exercise package-private
// helpers that assume Engine.mutex is already held by their caller.
func loadOwnedItemSession(t *testing.T, name string) (*Engine, string, *Session) {
	t.Helper()

	path := writeFixture(t, name, pcHeader(), pcFixtureSize)
	engine := New()
	info, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	return engine, info.SaveSessionID, engine.sessions[info.SaveSessionID].session
}

// inventoryLocator and storageLocator address the same physical coordinates in
// the two containers, so a test can prove that the container is part of the
// identity.
func inventoryLocator(characterID int, containerSection string, physicalIndex int) ownedItemLocator {
	return ownedItemLocator{
		characterID:      characterID,
		container:        ownedContainerInventory,
		containerSection: containerSection,
		physicalIndex:    physicalIndex,
	}
}

func storageLocator(characterID int, containerSection string, physicalIndex int) ownedItemLocator {
	return ownedItemLocator{
		characterID:      characterID,
		container:        ownedContainerStorage,
		containerSection: containerSection,
		physicalIndex:    physicalIndex,
	}
}

func TestNewSessionStartsAtRevisionZero(t *testing.T) {
	_, _, session := loadOwnedItemSession(t, "pc.sl2")

	if session.revision != 0 {
		t.Fatalf("revision of a freshly loaded session = %d, want 0", session.revision)
	}
	if len(session.ownedByID) != 0 || len(session.ownedByLocator) != 0 {
		t.Fatalf("registry of a freshly loaded session = %d/%d entries, want empty",
			len(session.ownedByID), len(session.ownedByLocator))
	}
}

func TestMintOwnedItemIDIsIdempotentForOneLocator(t *testing.T) {
	_, _, session := loadOwnedItemSession(t, "pc.sl2")
	locator := inventoryLocator(0, InventorySectionCommon, 7)

	first := session.mintOwnedItemID(locator)
	second := session.mintOwnedItemID(locator)

	if first == "" {
		t.Fatal("mintOwnedItemID returned an empty token")
	}
	if first != second {
		t.Fatalf("re-minting the same locator = %q, want the first token %q", second, first)
	}
	if len(session.ownedByID) != 1 {
		t.Fatalf("registry holds %d tokens, want 1", len(session.ownedByID))
	}
}

func TestMintOwnedItemIDSeparatesDistinctLocators(t *testing.T) {
	_, _, session := loadOwnedItemSession(t, "pc.sl2")

	locators := map[string]ownedItemLocator{
		"inventory common 0":  inventoryLocator(0, InventorySectionCommon, 0),
		"inventory common 1":  inventoryLocator(0, InventorySectionCommon, 1),
		"inventory key 0":     inventoryLocator(0, InventorySectionKey, 0),
		"storage common 0":    storageLocator(0, StorageSectionCommon, 0),
		"storage key 0":       storageLocator(0, StorageSectionKey, 0),
		"character 3 in inv":  inventoryLocator(3, InventorySectionCommon, 0),
		"character 3 in stor": storageLocator(3, StorageSectionCommon, 0),
	}

	tokens := make(map[string]string, len(locators))
	for name, locator := range locators {
		token := session.mintOwnedItemID(locator)
		if owner, taken := tokens[token]; taken {
			t.Fatalf("%s reused the token of %s", name, owner)
		}
		tokens[token] = name
	}

	// The same coordinates in Inventory and in Storage are two physical records.
	inInventory := session.mintOwnedItemID(inventoryLocator(0, InventorySectionCommon, 0))
	inStorage := session.mintOwnedItemID(storageLocator(0, StorageSectionCommon, 0))
	if inInventory == inStorage {
		t.Fatalf("Inventory and Storage share the token %q at the same coordinates", inInventory)
	}
	if len(session.ownedByID) != len(locators) {
		t.Fatalf("registry holds %d tokens, want %d", len(session.ownedByID), len(locators))
	}
}

func TestResolveOwnedItemIDReturnsTheMintedRecord(t *testing.T) {
	_, _, session := loadOwnedItemSession(t, "pc.sl2")
	locator := storageLocator(2, StorageSectionKey, 11)

	resolved, err := session.resolveOwnedItemID(2, session.mintOwnedItemID(locator))
	if err != nil {
		t.Fatalf("resolveOwnedItemID: %v", err)
	}
	if resolved != locator {
		t.Fatalf("resolveOwnedItemID = %+v, want %+v", resolved, locator)
	}
}

func TestResolveOwnedItemIDRejectsMissingAndUnknownTokens(t *testing.T) {
	_, _, session := loadOwnedItemSession(t, "pc.sl2")
	minted := session.mintOwnedItemID(inventoryLocator(0, InventorySectionCommon, 0))

	if _, err := session.resolveOwnedItemID(0, ""); err == nil {
		t.Fatal("an empty ownedItemID resolved, want a rejection")
	}

	// A foreign string and a fabricated token of the current revision are both
	// unknown; neither may fall back to a physical lookup.
	for _, token := range []string{"whatever", minted + "0", session.currentOwnedItemIDPrefix() + "999"} {
		if _, err := session.resolveOwnedItemID(0, token); !errors.Is(err, errUnknownOwnedItemID) {
			t.Fatalf("resolveOwnedItemID(%q) error = %v, want errUnknownOwnedItemID", token, err)
		}
	}
}

func TestResolveOwnedItemIDRejectsTheWrongCharacter(t *testing.T) {
	_, _, session := loadOwnedItemSession(t, "pc.sl2")
	token := session.mintOwnedItemID(inventoryLocator(1, InventorySectionCommon, 4))

	if _, err := session.resolveOwnedItemID(2, token); err == nil {
		t.Fatal("a token of character 1 resolved for character 2, want a rejection")
	}
	if _, err := session.resolveOwnedItemID(1, token); err != nil {
		t.Fatalf("the token stayed valid for its own character: %v", err)
	}
}

func TestCommitRevisionAdvancesAndInvalidatesIdentities(t *testing.T) {
	engine, saveSessionID, session := loadOwnedItemSession(t, "pc.sl2")
	locator := inventoryLocator(0, InventorySectionCommon, 3)
	stale := session.mintOwnedItemID(locator)

	committed, err := engine.commitRevision(saveSessionID, func(*loadedSave) error { return nil })
	if err != nil {
		t.Fatalf("commitRevision: %v", err)
	}
	if committed != "1" {
		t.Fatalf("commitRevision returned revision %q, want \"1\"", committed)
	}

	if session.revision != 1 {
		t.Fatalf("revision after one commit = %d, want 1", session.revision)
	}
	if len(session.ownedByID) != 0 || len(session.ownedByLocator) != 0 {
		t.Fatalf("registry after a commit = %d/%d entries, want empty",
			len(session.ownedByID), len(session.ownedByLocator))
	}
	if _, err := session.resolveOwnedItemID(0, stale); !errors.Is(err, errStaleOwnedItemID) {
		t.Fatalf("resolving a token of the previous revision = %v, want errStaleOwnedItemID", err)
	}
	if fresh := session.mintOwnedItemID(locator); fresh == stale {
		t.Fatalf("re-minting after a commit returned the retired token %q", stale)
	}
}

func TestCommitRevisionLeavesEverythingUntouchedWhenTheCommitFails(t *testing.T) {
	engine, saveSessionID, session := loadOwnedItemSession(t, "pc.sl2")
	locator := inventoryLocator(0, InventorySectionCommon, 3)
	token := session.mintOwnedItemID(locator)

	rejected := errors.New("validation rejected the plan")
	committed, err := engine.commitRevision(saveSessionID, func(*loadedSave) error { return rejected })
	if !errors.Is(err, rejected) {
		t.Fatalf("commitRevision error = %v, want the commit error", err)
	}
	if committed != "" {
		t.Fatalf("a failed commit returned revision %q, want an empty one", committed)
	}

	if session.revision != 0 {
		t.Fatalf("revision after a failed commit = %d, want 0", session.revision)
	}
	if len(session.ownedByID) != 1 || len(session.ownedByLocator) != 1 {
		t.Fatalf("registry after a failed commit = %d/%d entries, want 1/1",
			len(session.ownedByID), len(session.ownedByLocator))
	}
	if _, err := session.resolveOwnedItemID(0, token); err != nil {
		t.Fatalf("a token issued before a failed commit stopped resolving: %v", err)
	}
	if again := session.mintOwnedItemID(locator); again != token {
		t.Fatalf("minting after a failed commit = %q, want the unchanged token %q", again, token)
	}
}

func TestOwnedItemIDIsRejectedByAnotherSession(t *testing.T) {
	_, _, first := loadOwnedItemSession(t, "first.sl2")
	_, _, second := loadOwnedItemSession(t, "second.sl2")
	locator := inventoryLocator(0, InventorySectionCommon, 0)

	foreign := first.mintOwnedItemID(locator)
	if _, err := second.resolveOwnedItemID(0, foreign); !errors.Is(err, errUnknownOwnedItemID) {
		t.Fatalf("a token of another session resolved with %v, want errUnknownOwnedItemID", err)
	}
	// The second session mints its own token for the same coordinates.
	if own := second.mintOwnedItemID(locator); own == foreign {
		t.Fatalf("two sessions minted the same token %q for the same locator", own)
	}
}

func TestOwnedItemIdentityIsSerialisedByTheEngineMutex(t *testing.T) {
	engine, saveSessionID, session := loadOwnedItemSession(t, "pc.sl2")

	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			for round := 0; round < 25; round++ {
				// The session helpers carry no lock of their own, so every direct
				// use takes the engine lock the production callers already hold.
				engine.mutex.Lock()
				token := session.mintOwnedItemID(inventoryLocator(worker, InventorySectionCommon, round))
				_, err := session.resolveOwnedItemID(worker, token)
				engine.mutex.Unlock()
				if err != nil {
					// Mint and resolve share one critical section, so a concurrent
					// commit can never slip between them.
					t.Errorf("resolve of a freshly minted token: %v", err)
					return
				}
				if round%10 == 0 {
					if _, err := engine.commitRevision(
						saveSessionID, func(*loadedSave) error { return nil }); err != nil {
						t.Errorf("commitRevision: %v", err)
						return
					}
				}
			}
		}(worker)
	}
	workers.Wait()

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if session.revision == 0 {
		t.Fatal("no commit advanced the revision")
	}
	if len(session.ownedByID) != len(session.ownedByLocator) {
		t.Fatalf("registry directions diverged: %d tokens, %d locators",
			len(session.ownedByID), len(session.ownedByLocator))
	}
}
