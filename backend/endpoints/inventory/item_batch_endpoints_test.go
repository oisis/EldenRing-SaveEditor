package inventory

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/safetyprofile"
)

// This file covers the public boundary of the batch mutations and of the
// container getter: the profile the host supplies, the atomicity the endpoint
// promises and the exact scopes one execution reports. The physical rows, the
// counters and the rollback belong to backend/saveengine and are not repeated
// here.

const chaosMode = string(safetyprofile.Chaos)

// batchTestBanRiskDocument is the stacking goods shape plus a known ban-risk
// fact, so the same request differs only by what the profile allows.
func batchTestBanRiskDocument(
	category string, maxInventory, maxPerStack uint32,
) func(*schema.ItemDocument) {
	return addItemTestThen(category, maxInventory, maxPerStack,
		func(document *schema.ItemDocument) {
			document.Safety.BanRisk = schema.Fact[bool]{
				Known:      true,
				Value:      true,
				Provenance: document.Safety.BanRisk.Provenance,
			}
		})
}

// An Inventory-only batch commits one revision and reports the container it
// actually wrote. Storage is absent from the scopes because no entry addressed
// it, so the caller never refreshes a container this execution left alone.
func TestAddItemsToContainersCommitsOneRevisionAndNamesOnlyTheContainerItWrote(t *testing.T) {
	catalog := quantityTestCatalog(t, addItemTestDocument("tools", 600, 40))
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

	result, err := AddItemsToContainers(
		engine, catalog, expandedLimits, sessionID, getInventorySlot,
		[]AddItemsRequestEntry{{
			Kind: addItemTestKind, Key: addItemTestKey, InventoryQuantity: 5,
		}},
		false, "0")
	if err != nil {
		t.Fatalf("AddItemsToContainers: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, sessionID, AddItemsToContainersEndpointID, "1")
	for _, scope := range result.ChangedScopes {
		if scope == "storage" {
			t.Errorf("an inventory-only batch reported the storage scope: %v", result.ChangedScopes)
		}
	}
	found := false
	for _, scope := range result.ChangedScopes {
		if scope == "inventory" {
			found = true
		}
	}
	if !found {
		t.Errorf("changedScopes = %v, want the inventory scope", result.ChangedScopes)
	}

	quantity, _, revision, dirty := addItemTestState(t, engine, catalog, sessionID)
	if quantity != 8 || revision != "1" || !dirty {
		t.Errorf("after the batch common#1 holds %d at revision %q, dirty %v; want 8, \"1\", true",
			quantity, revision, dirty)
	}
}

// A failing later entry leaves the session exactly as it was: the batch is
// rejected before the session is touched, so no revision, no history entry and
// no dirty flag can appear.
func TestAddItemsToContainersLeavesTheSessionUntouchedWhenALaterEntryFails(t *testing.T) {
	catalog := quantityTestCatalog(t, addItemTestDocument("tools", 600, 40))
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

	result, err := AddItemsToContainers(
		engine, catalog, expandedLimits, sessionID, getInventorySlot,
		[]AddItemsRequestEntry{
			{Kind: addItemTestKind, Key: addItemTestKey, InventoryQuantity: 5},
			{Kind: addItemTestKind, Key: "not-a-catalog-key", InventoryQuantity: 1},
		},
		false, "0")
	if err == nil {
		t.Fatalf("the batch was accepted with an unknown second entry: %+v", result)
	}
	if !strings.Contains(err.Error(), "items[1]") {
		t.Errorf("error = %v, want the rejected entry index", err)
	}
	if !reflect.DeepEqual(result, AddItemsToContainersResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}

	quantity, _, revision, dirty := addItemTestState(t, engine, catalog, sessionID)
	if quantity != 3 || revision != "0" || dirty {
		t.Errorf("the rejected batch left quantity %d, revision %q, unsavedChanges %v;"+
			" want 3, \"0\", false", quantity, revision, dirty)
	}
}

// The profile decides whether a ban-risk resource may be written at all, and
// the user's confirmation can never substitute for a profile that forbids it.
func TestAddItemsToContainersAppliesTheBanRiskRuleOfTheActiveProfile(t *testing.T) {
	cases := []struct {
		name           string
		profile        string
		confirmBanRisk bool
		accepted       bool
	}{
		{"safe refuses even with confirmation", safeMode, true, false},
		{"expanded limits refuses even with confirmation", expandedLimits, true, false},
		{"chaos still needs the confirmation", chaosMode, false, false},
		{"chaos with the confirmation", chaosMode, true, true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			catalog := quantityTestCatalog(t, batchTestBanRiskDocument("tools", 600, 40))
			engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

			result, err := AddItemsToContainers(
				engine, catalog, testCase.profile, sessionID, getInventorySlot,
				[]AddItemsRequestEntry{{
					Kind: addItemTestKind, Key: addItemTestKey, InventoryQuantity: 5,
				}},
				testCase.confirmBanRisk, "0")
			if testCase.accepted {
				if err != nil {
					t.Fatalf("AddItemsToContainers: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("a ban-risk addition was accepted: %+v", result)
			}
			if !reflect.DeepEqual(result, AddItemsToContainersResult{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
			quantity, _, revision, dirty := addItemTestState(t, engine, catalog, sessionID)
			if quantity != 3 || revision != "0" || dirty {
				t.Errorf("the refused addition left quantity %d, revision %q, unsavedChanges %v;"+
					" want 3, \"0\", false", quantity, revision, dirty)
			}
		})
	}
}

// The batch endpoints reject an unusable request before the session is read.
func TestBatchEndpointsRejectUnusableRequests(t *testing.T) {
	catalog := quantityTestCatalog(t, addItemTestDocument("tools", 600, 40))
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

	t.Run("AddItemsToContainers/unknown profile", func(t *testing.T) {
		result, err := AddItemsToContainers(engine, catalog, "", sessionID, getInventorySlot,
			[]AddItemsRequestEntry{{Kind: addItemTestKind, Key: addItemTestKey, InventoryQuantity: 1}},
			false, "0")
		if err == nil || !strings.Contains(err.Error(), "unknown safety profile") {
			t.Fatalf("error = %v, want an unknown safety profile", err)
		}
		assertZeroResult(t, err, result, AddItemsToContainersResult{})
	})
	t.Run("AddItemsToContainers/empty batch", func(t *testing.T) {
		result, err := AddItemsToContainers(engine, catalog, expandedLimits, sessionID,
			getInventorySlot, nil, false, "0")
		assertZeroResult(t, err, result, AddItemsToContainersResult{})
	})
	t.Run("AddItemsToContainers/entry addressing neither container", func(t *testing.T) {
		result, err := AddItemsToContainers(engine, catalog, expandedLimits, sessionID,
			getInventorySlot,
			[]AddItemsRequestEntry{{Kind: addItemTestKind, Key: addItemTestKey}}, false, "0")
		assertZeroResult(t, err, result, AddItemsToContainersResult{})
	})
	t.Run("MoveOwnedItemsToStorage/unknown profile", func(t *testing.T) {
		result, err := MoveOwnedItemsToStorage(engine, catalog, "", sessionID, getInventorySlot,
			[]string{"oi-not-a-record"}, "0")
		assertZeroResult(t, err, result, MoveOwnedItemsToStorageResult{})
	})
	t.Run("MoveOwnedItemsToInventory/unknown profile", func(t *testing.T) {
		result, err := MoveOwnedItemsToInventory(engine, catalog, "", sessionID, getInventorySlot,
			[]string{"oi-not-a-record"}, "0")
		assertZeroResult(t, err, result, MoveOwnedItemsToInventoryResult{})
	})
	t.Run("RemoveOwnedItems/no engine", func(t *testing.T) {
		result, err := RemoveOwnedItems(nil, sessionID, getInventorySlot,
			[]string{"oi-not-a-record"}, "0")
		assertZeroResult(t, err, result, RemoveOwnedItemsResult{})
	})
	t.Run("ReorderInventoryItems/no catalog", func(t *testing.T) {
		result, err := ReorderInventoryItems(engine, nil, sessionID, getInventorySlot,
			"oi-not-a-record", nil, 0, "0")
		assertZeroResult(t, err, result, ReorderInventoryItemsResult{})
	})

	// Nothing above may have moved the session.
	quantity, _, revision, dirty := addItemTestState(t, engine, catalog, sessionID)
	if quantity != 3 || revision != "0" || dirty {
		t.Errorf("a rejected request left quantity %d, revision %q, unsavedChanges %v;"+
			" want 3, \"0\", false", quantity, revision, dirty)
	}
}

// GetOwnedItems reports the container it was asked for together with the
// profile it resolved the limits under, and rejects a container or a profile it
// does not know.
func TestGetOwnedItemsReportsTheContainerUnderTheActiveProfile(t *testing.T) {
	catalog := quantityTestCatalog(t, addItemTestDocument("tools", 600, 40))
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

	result, err := GetOwnedItems(engine, catalog, safeMode, sessionID, getInventorySlot,
		OwnedItemsContainerInventory, "", "", "", false, nil, OwnedItemsSortContainer, 0, 0)
	if err != nil {
		t.Fatalf("GetOwnedItems: %v", err)
	}
	if result.SafetyProfile != safeMode {
		t.Errorf("safetyProfile = %q, want %q", result.SafetyProfile, safeMode)
	}
	if result.Container != OwnedItemsContainerInventory {
		t.Errorf("container = %q, want %q", result.Container, OwnedItemsContainerInventory)
	}
	if result.SaveRevision != "0" {
		t.Errorf("saveRevision = %q, want 0", result.SaveRevision)
	}
	if result.Total != len(result.Records) || result.Total == 0 {
		t.Errorf("total = %d with %d records, want one non-empty page",
			result.Total, len(result.Records))
	}

	if _, err := GetOwnedItems(engine, catalog, "", sessionID, getInventorySlot,
		OwnedItemsContainerInventory, "", "", "", false, nil,
		OwnedItemsSortContainer, 0, 0); err == nil {
		t.Error("GetOwnedItems accepted an unknown safety profile")
	}
	if _, err := GetOwnedItems(engine, catalog, safeMode, sessionID, getInventorySlot,
		"backpack", "", "", "", false, nil, OwnedItemsSortContainer, 0, 0); err == nil {
		t.Error("GetOwnedItems accepted an unknown container")
	}
}
