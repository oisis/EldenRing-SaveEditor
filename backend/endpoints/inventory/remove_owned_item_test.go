package inventory

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// This file covers only what the endpoint itself owns: the missing-dependency
// guard, the declared contract, and the pass-through of every value to
// SaveEngine, including the refusal of the unsupported Storage key section. The
// removal plan, the identity rules, the revision guard, the section counts, the
// equipment guard, the platform matrix and the rollback belong to
// backend/saveengine/remove_owned_item_test.go and are not repeated here.
//
// The two container fixtures of this package are reused. Both carry the same
// three records, so removing common#1 has to leave common#6 and key#1 alone.
const removeTestRow = "common#1"

// removeTestTarget loads the fixture of one container, reads it through the
// endpoint a caller would use to obtain an identity at all, and returns the
// engine, the session and the identity of the addressed row.
func removeTestTarget(
	t *testing.T, container string, catalog *gamecatalog.Catalog,
) (*saveengine.Engine, string, string) {
	t.Helper()

	if container == "storage" {
		engine, sessionID := loadGetStorageSession(t, "pc", true, getStorageAnchorAt)
		listed, err := GetStorage(engine, catalog, sessionID, getStorageSlot, "", 0, 0)
		if err != nil {
			t.Fatalf("GetStorage: %v", err)
		}
		return engine, sessionID, removeTestIdentity(t, getStorageIdentities(t, listed.Records))
	}

	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	listed, err := GetInventory(engine, catalog, sessionID, getInventorySlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	return engine, sessionID, removeTestIdentity(t, getInventoryIdentities(t, listed.Records))
}

func removeTestIdentity(t *testing.T, identities map[string]string) string {
	t.Helper()

	id := identities[removeTestRow]
	if id == "" {
		t.Fatalf("the container read never identified %s", removeTestRow)
	}
	return id
}

// removeTestRows re-reads the container and reports which physical rows it still
// holds, plus the session revision and the unsaved-changes flag.
func removeTestRows(
	t *testing.T, engine *saveengine.Engine, catalog *gamecatalog.Catalog,
	container, sessionID string,
) (map[string]string, string, bool) {
	t.Helper()

	var rows map[string]string
	var revision string
	if container == "storage" {
		listed, err := GetStorage(engine, catalog, sessionID, getStorageSlot, "", 0, 0)
		if err != nil {
			t.Fatalf("GetStorage: %v", err)
		}
		rows, revision = getStorageIdentities(t, listed.Records), listed.SaveRevision
	} else {
		listed, err := GetInventory(engine, catalog, sessionID, getInventorySlot, "", 0, 0)
		if err != nil {
			t.Fatalf("GetInventory: %v", err)
		}
		rows, revision = getInventoryIdentities(t, listed.Records), listed.SaveRevision
	}

	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	return rows, revision, info.UnsavedChanges
}

func removeTestSlot(container string) int {
	if container == "storage" {
		return getStorageSlot
	}
	return getInventorySlot
}

// A record of either container is removed through the endpoint, and only that
// one row disappears.
func TestRemoveOwnedItemCommitsInBothContainers(t *testing.T) {
	for _, container := range []string{"inventory", "storage"} {
		t.Run(container, func(t *testing.T) {
			catalog := inventoryCatalog(t)
			engine, sessionID, ownedItemID := removeTestTarget(t, container, catalog)

			result, err := RemoveOwnedItem(
				engine, sessionID, removeTestSlot(container), ownedItemID, "0")
			if err != nil {
				t.Fatalf("RemoveOwnedItem: %v", err)
			}
			assertMutationReceipt(t, result.MutationReceipt, sessionID,
				RemoveOwnedItemEndpointID, "1")
			// The receipt is pinned from the result because operationID names one
			// execution and cannot be predicted; every other member is asserted above.
			want := RemoveOwnedItemResult{
				MutationReceipt: result.MutationReceipt,
				OwnedItemID:     ownedItemID,
				CharacterID:     removeTestSlot(container),
				GameID:          0x4000272E,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			rows, revision, dirty := removeTestRows(t, engine, catalog, container, sessionID)
			if _, present := rows[removeTestRow]; present {
				t.Errorf("%s is still listed after the removal", removeTestRow)
			}
			for _, kept := range []string{"common#6", "key#1"} {
				if _, present := rows[kept]; !present {
					t.Errorf("the removal also dropped %s; rows = %v", kept, rows)
				}
			}
			if revision != "1" {
				t.Errorf("saveRevision = %q, want 1", revision)
			}
			if !dirty {
				t.Error("a committed removal reported no unsaved changes")
			}
		})
	}
}

// The endpoint owns no identity, revision or plan rule: it hands every value to
// SaveEngine unchanged, and a rejection there leaves the container, the revision
// and the unsaved-changes flag exactly as they were.
func TestRemoveOwnedItemPassesRejectionsThrough(t *testing.T) {
	cases := map[string]struct {
		characterID      int
		ownedItemID      func(valid string) string
		expectedRevision string
		message          string
	}{
		"empty identity": {getInventorySlot,
			func(string) string { return "" }, "0", "ownedItemID is required"},
		"unknown identity": {getInventorySlot,
			func(valid string) string { return valid + "-x" }, "0", "unknown ownedItemID"},
		"another character": {getInventorySlot + 1,
			func(valid string) string { return valid }, "0", "belongs to character"},
		"characterID out of range": {10,
			func(valid string) string { return valid }, "0", "outside the range 0..9"},
		"malformed revision": {getInventorySlot,
			func(valid string) string { return valid }, "00", "canonical decimal saveRevision"},
		"stale revision": {getInventorySlot,
			func(valid string) string { return valid }, "7", "does not match the current saveRevision"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			catalog := inventoryCatalog(t)
			engine, sessionID, ownedItemID := removeTestTarget(t, "inventory", catalog)

			result, err := RemoveOwnedItem(engine, sessionID, testCase.characterID,
				testCase.ownedItemID(ownedItemID), testCase.expectedRevision)
			if err == nil {
				t.Fatalf("RemoveOwnedItem accepted %s: %+v", name, result)
			}
			if !strings.Contains(err.Error(), testCase.message) {
				t.Errorf("error = %v, want it to contain %q", err, testCase.message)
			}
			if !reflect.DeepEqual(result, RemoveOwnedItemResult{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}

			rows, revision, dirty := removeTestRows(t, engine, catalog, "inventory", sessionID)
			if len(rows) != 3 || revision != "0" || dirty {
				t.Errorf("rejected removal left %d rows, revision %q, unsavedChanges %v;"+
					" want 3, \"0\", false", len(rows), revision, dirty)
			}
		})
	}
}

// The Storage Box key section has no confirmed native write contract, so a
// caller who addresses one of its records is refused at the public layer too,
// and the container keeps all three of its rows.
func TestRemoveOwnedItemRejectsAStorageKeyRecord(t *testing.T) {
	catalog := inventoryCatalog(t)
	engine, sessionID := loadGetStorageSession(t, "pc", true, getStorageAnchorAt)
	listed, err := GetStorage(engine, catalog, sessionID, getStorageSlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	ownedItemID := getStorageIdentities(t, listed.Records)["key#1"]
	if ownedItemID == "" {
		t.Fatal("the storage read never identified key#1")
	}

	result, err := RemoveOwnedItem(engine, sessionID, getStorageSlot, ownedItemID, "0")
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("error = %v, want the Storage key section refused as unsupported", err)
	}
	if !reflect.DeepEqual(result, RemoveOwnedItemResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}

	rows, revision, dirty := removeTestRows(t, engine, catalog, "storage", sessionID)
	if len(rows) != 3 || revision != "0" || dirty {
		t.Errorf("rejected removal left %d rows, revision %q, unsavedChanges %v;"+
			" want 3, \"0\", false", len(rows), revision, dirty)
	}
}

// A missing engine is rejected in the endpoint itself, before any session is
// touched. The endpoint needs no catalog, so there is no second dependency to
// reject.
func TestRemoveOwnedItemRejectsAMissingEngine(t *testing.T) {
	catalog := inventoryCatalog(t)
	engine, sessionID, ownedItemID := removeTestTarget(t, "inventory", catalog)

	result, err := RemoveOwnedItem(nil, sessionID, getInventorySlot, ownedItemID, "0")
	if err == nil || err.Error() != "save engine is not available" {
		t.Errorf("nil engine error = %v, want save engine is not available", err)
	}
	if !reflect.DeepEqual(result, RemoveOwnedItemResult{}) {
		t.Errorf("nil engine result = %+v, want the zero value", result)
	}

	rows, revision, dirty := removeTestRows(t, engine, catalog, "inventory", sessionID)
	if len(rows) != 3 || revision != "0" || dirty {
		t.Errorf("rejected removal left %d rows, revision %q, unsavedChanges %v;"+
			" want 3, \"0\", false", len(rows), revision, dirty)
	}
}

func TestRemoveOwnedItemContractDeclaresTheAcceptedVariables(t *testing.T) {
	if RemoveOwnedItemDefinition.SupportedResourceTypes != "ItemDocument" {
		t.Errorf("supported resource types = %q, want ItemDocument",
			RemoveOwnedItemDefinition.SupportedResourceTypes)
	}
	want := []string{"saveSessionID", "characterID", "ownedItemID", "expectedRevision"}
	if !reflect.DeepEqual(RemoveOwnedItemDefinition.SupportedResourceVariables, want) {
		t.Errorf("supported resource variables = %v, want %v",
			RemoveOwnedItemDefinition.SupportedResourceVariables, want)
	}
}
