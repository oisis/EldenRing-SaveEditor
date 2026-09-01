package inventory

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// This file covers only what the endpoint itself owns: the catalog projection
// that produces the two limits, and the pass-through of everything else. The
// mutation, the revision check, the container total, the quantity bounds and the
// rollback belong to backend/saveengine/set_owned_item_quantity_test.go and are
// not repeated here.
//
// The two container fixtures of this package are reused, and the record both of
// them carry at common#1 is the one addressed throughout.
const (
	quantityTestGameID = 0x4000272E
	quantityTestRow    = "common#1"
)

// quantityTestCatalog rebuilds the loaded catalog with the addressed document
// rewritten, so a test states the catalog facts the endpoint reads instead of
// depending on whatever the shipped document happens to declare. The document is
// copied before it is changed, so the shared loader data stays untouched.
func quantityTestCatalog(t *testing.T, apply func(*schema.ItemDocument)) *gamecatalog.Catalog {
	t.Helper()

	base := inventoryCatalog(t)
	target, exists := base.ItemByGameID(quantityTestGameID)
	if !exists {
		t.Fatalf("game ID 0x%08X is absent from the catalog", uint32(quantityTestGameID))
	}

	resources := inventoryCatalogData.Resources()
	rewritten := false
	for index, resource := range resources {
		if resource.Kind != target.Kind || resource.Key != target.Key {
			continue
		}
		document := *resource.Item
		apply(&document)
		resources[index].Item = &document
		rewritten = true
		break
	}
	if !rewritten {
		t.Fatalf("resource %s/%s is absent from the loaded resources", target.Kind, target.Key)
	}

	catalog, err := gamecatalog.New(inventoryCatalogData.Manifest, resources)
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	return catalog
}

// quantityTestItem is the fully known shape the endpoint requires. Every test
// starts from it and states only the numbers, or the single fact, it is about.
//
// A stack capability that is enabled without rules, or with maxPerStack 0, is
// rejected by schema validation itself, so no validated catalog can carry one.
// The endpoint's guard against it is unreachable from here by construction.
func quantityTestItem(maxPerStack, maxInventory, maxStorage uint32) func(*schema.ItemDocument) {
	return func(document *schema.ItemDocument) {
		source := document.Capabilities.Stack.Provenance
		document.Capabilities.Stack = schema.Capability[schema.StackRules]{
			Known:         true,
			Enabled:       true,
			Rules:         &schema.StackRules{MaxPerStack: maxPerStack},
			Provenance:    source,
			RulesEvidence: []schema.Provenance{source},
		}
		document.Storage.RecordMode = schema.Fact[schema.RecordMode]{
			Known:      true,
			Value:      schema.RecordModeQuantityStack,
			Provenance: document.Storage.RecordMode.Provenance,
		}
		document.Storage.MaxInventory = schema.Fact[uint32]{
			Known:      true,
			Value:      maxInventory,
			Provenance: document.Storage.MaxInventory.Provenance,
		}
		document.Storage.MaxStorage = schema.Fact[uint32]{
			Known:      true,
			Value:      maxStorage,
			Provenance: document.Storage.MaxStorage.Provenance,
		}
	}
}

// quantityTestThen applies the known shape and then the single deviation one
// rejection case is about.
func quantityTestThen(
	maxPerStack, maxInventory, maxStorage uint32, deviate func(*schema.ItemDocument),
) func(*schema.ItemDocument) {
	known := quantityTestItem(maxPerStack, maxInventory, maxStorage)
	return func(document *schema.ItemDocument) {
		known(document)
		deviate(document)
	}
}

// quantityTestSlot is the character slot of one container fixture.
func quantityTestSlot(container string) int {
	if container == "storage" {
		return getStorageSlot
	}
	return getInventorySlot
}

// quantityTestTarget loads the fixture of one container, reads it through the
// endpoint a caller would use to obtain an identity at all, and returns the
// engine, the session and the identity of the addressed row.
func quantityTestTarget(
	t *testing.T, container string, catalog *gamecatalog.Catalog,
) (*saveengine.Engine, string, string) {
	t.Helper()

	if container == "storage" {
		engine, sessionID := loadGetStorageSession(t, "pc", true, getStorageAnchorAt)
		listed, err := GetStorage(engine, catalog, sessionID, getStorageSlot, "", 0, 0)
		if err != nil {
			t.Fatalf("GetStorage: %v", err)
		}
		return engine, sessionID, quantityTestIdentity(t, getStorageIdentities(t, listed.Records))
	}

	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	listed, err := GetInventory(engine, catalog, sessionID, getInventorySlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	return engine, sessionID, quantityTestIdentity(t, getInventoryIdentities(t, listed.Records))
}

func quantityTestIdentity(t *testing.T, identities map[string]string) string {
	t.Helper()

	id := identities[quantityTestRow]
	if id == "" {
		t.Fatalf("the container read never identified %s", quantityTestRow)
	}
	return id
}

// quantityTestState re-reads the addressed row and reports what a rejected
// mutation has to leave untouched: the stored quantity, the session revision and
// the unsaved-changes flag.
func quantityTestState(
	t *testing.T, engine *saveengine.Engine, catalog *gamecatalog.Catalog,
	container, sessionID string,
) (uint32, string, bool) {
	t.Helper()

	var revision string
	var quantity uint32
	found := false
	if container == "storage" {
		listed, err := GetStorage(engine, catalog, sessionID, getStorageSlot, "", 0, 0)
		if err != nil {
			t.Fatalf("GetStorage: %v", err)
		}
		revision = listed.SaveRevision
		for _, record := range listed.Records {
			if record.ContainerSection == "common" && record.PhysicalIndex == 1 {
				quantity, found = record.Quantity, true
			}
		}
	} else {
		listed, err := GetInventory(engine, catalog, sessionID, getInventorySlot, "", 0, 0)
		if err != nil {
			t.Fatalf("GetInventory: %v", err)
		}
		revision = listed.SaveRevision
		for _, record := range listed.Records {
			if record.ContainerSection == "common" && record.PhysicalIndex == 1 {
				quantity, found = record.Quantity, true
			}
		}
	}
	if !found {
		t.Fatalf("the %s read no longer contains %s", container, quantityTestRow)
	}

	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	return quantity, revision, info.UnsavedChanges
}

// A record of either container commits under the limits of its own container.
func TestSetOwnedItemQuantityCommitsInBothContainers(t *testing.T) {
	for _, container := range []string{"inventory", "storage"} {
		t.Run(container, func(t *testing.T) {
			catalog := quantityTestCatalog(t, quantityTestItem(99, 600, 600))
			engine, sessionID, ownedItemID := quantityTestTarget(t, container, catalog)

			result, err := SetOwnedItemQuantity(
				engine, catalog, sessionID, quantityTestSlot(container), ownedItemID, 42, "0")
			if err != nil {
				t.Fatalf("SetOwnedItemQuantity: %v", err)
			}
			assertMutationReceipt(t, result.MutationReceipt, sessionID,
				SetOwnedItemQuantityEndpointID, "1")
			// The receipt is pinned from the result because operationID names one
			// execution and cannot be predicted; every other member is asserted above.
			want := SetOwnedItemQuantityResult{
				MutationReceipt: result.MutationReceipt,
				OwnedItemID:     ownedItemID,
				CharacterID:     quantityTestSlot(container),
				Quantity:        42,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			quantity, revision, dirty := quantityTestState(t, engine, catalog, container, sessionID)
			if quantity != 42 {
				t.Errorf("stored quantity = %d, want 42", quantity)
			}
			if revision != "1" {
				t.Errorf("saveRevision = %q, want 1", revision)
			}
			if !dirty {
				t.Error("a committed mutation reported no unsaved changes")
			}
		})
	}
}

// The per-record limit is the smaller of the two, so a Storage record stays
// bounded by maxPerStack even though maxStorage is far larger.
func TestSetOwnedItemQuantityTakesTheSmallerOfStackAndContainerLimits(t *testing.T) {
	catalog := quantityTestCatalog(t, quantityTestItem(5, 600, 600))
	engine, sessionID, ownedItemID := quantityTestTarget(t, "storage", catalog)

	result, err := SetOwnedItemQuantity(
		engine, catalog, sessionID, getStorageSlot, ownedItemID, 6, "0")
	if err == nil {
		t.Fatalf("SetOwnedItemQuantity accepted 6 above the stack limit of 5: %+v", result)
	}
	if !strings.Contains(err.Error(), "exceeds the limit of 5 per record") {
		t.Errorf("error = %v, want the per-record limit of 5", err)
	}
	if !reflect.DeepEqual(result, SetOwnedItemQuantityResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}

	quantity, revision, dirty := quantityTestState(t, engine, catalog, "storage", sessionID)
	if quantity != 3 || revision != "0" || dirty {
		t.Errorf("rejected mutation left quantity %d, revision %q, unsavedChanges %v; want 3, \"0\", false",
			quantity, revision, dirty)
	}

	if _, err := SetOwnedItemQuantity(
		engine, catalog, sessionID, getStorageSlot, ownedItemID, 5, "0"); err != nil {
		t.Fatalf("SetOwnedItemQuantity rejected the stack limit itself: %v", err)
	}
}

// Unknown or unsupported catalog data rejects the whole request. No limit is
// defaulted, widened or invented, and the record is left exactly as it was.
func TestSetOwnedItemQuantityRejectsUnusableCatalogData(t *testing.T) {
	cases := map[string]struct {
		container string
		apply     func(*schema.ItemDocument)
		message   string
	}{
		"unknown stack capability": {"inventory", quantityTestThen(99, 600, 600,
			func(document *schema.ItemDocument) {
				document.Capabilities.Stack = schema.Capability[schema.StackRules]{
					Provenance: document.Capabilities.Stack.Provenance,
				}
			}), "unknown stack capability"},
		"disabled stack capability": {"inventory", quantityTestThen(99, 600, 600,
			func(document *schema.ItemDocument) {
				document.Capabilities.Stack = schema.Capability[schema.StackRules]{
					Known:      true,
					Provenance: document.Capabilities.Stack.Provenance,
				}
			}), "does not stack"},
		"separate instances": {"inventory", quantityTestThen(99, 600, 600,
			func(document *schema.ItemDocument) {
				document.Storage.RecordMode.Value = schema.RecordModeSeparateInstances
			}), "does not store a quantity in one record"},
		"unknown record mode": {"inventory", quantityTestThen(99, 600, 600,
			func(document *schema.ItemDocument) {
				document.Storage.RecordMode.Known = false
				document.Storage.RecordMode.Value = ""
			}), "does not store a quantity in one record"},
		"unknown inventory limit": {"inventory", quantityTestThen(99, 600, 600,
			func(document *schema.ItemDocument) {
				document.Storage.MaxInventory.Known = false
				document.Storage.MaxInventory.Value = 0
			}), "carries no inventory limit"},
		"zero inventory limit": {"inventory", quantityTestThen(99, 0, 600,
			func(*schema.ItemDocument) {}), "carries no inventory limit"},
		"unknown storage limit": {"storage", quantityTestThen(99, 600, 600,
			func(document *schema.ItemDocument) {
				document.Storage.MaxStorage.Known = false
				document.Storage.MaxStorage.Value = 0
			}), "carries no storage limit"},
		"zero storage limit": {"storage", quantityTestThen(99, 600, 0,
			func(*schema.ItemDocument) {}), "carries no storage limit"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			catalog := quantityTestCatalog(t, testCase.apply)
			engine, sessionID, ownedItemID := quantityTestTarget(t, testCase.container, catalog)

			result, err := SetOwnedItemQuantity(engine, catalog, sessionID,
				quantityTestSlot(testCase.container), ownedItemID, 42, "0")
			if err == nil {
				t.Fatalf("SetOwnedItemQuantity accepted the request: %+v", result)
			}
			if !strings.Contains(err.Error(), testCase.message) {
				t.Errorf("error = %v, want it to contain %q", err, testCase.message)
			}
			if !reflect.DeepEqual(result, SetOwnedItemQuantityResult{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}

			quantity, revision, dirty := quantityTestState(
				t, engine, catalog, testCase.container, sessionID)
			if quantity != 3 || revision != "0" || dirty {
				t.Errorf("rejected mutation left quantity %d, revision %q, unsavedChanges %v;"+
					" want 3, \"0\", false", quantity, revision, dirty)
			}
		})
	}
}

// A missing dependency is rejected in the endpoint itself, before any session is
// touched.
func TestSetOwnedItemQuantityRejectsMissingDependencies(t *testing.T) {
	catalog := quantityTestCatalog(t, quantityTestItem(99, 600, 600))
	engine, sessionID, ownedItemID := quantityTestTarget(t, "inventory", catalog)

	result, err := SetOwnedItemQuantity(
		nil, catalog, sessionID, getInventorySlot, ownedItemID, 42, "0")
	if err == nil || err.Error() != "save engine is not available" {
		t.Errorf("nil engine error = %v, want save engine is not available", err)
	}
	if !reflect.DeepEqual(result, SetOwnedItemQuantityResult{}) {
		t.Errorf("nil engine result = %+v, want the zero value", result)
	}

	result, err = SetOwnedItemQuantity(
		engine, nil, sessionID, getInventorySlot, ownedItemID, 42, "0")
	if err == nil || err.Error() != "game catalog is not available" {
		t.Errorf("nil catalog error = %v, want game catalog is not available", err)
	}
	if !reflect.DeepEqual(result, SetOwnedItemQuantityResult{}) {
		t.Errorf("nil catalog result = %+v, want the zero value", result)
	}

	quantity, revision, dirty := quantityTestState(t, engine, catalog, "inventory", sessionID)
	if quantity != 3 || revision != "0" || dirty {
		t.Errorf("rejected mutation left quantity %d, revision %q, unsavedChanges %v;"+
			" want 3, \"0\", false", quantity, revision, dirty)
	}
}

func TestSetOwnedItemQuantityContractDeclaresTheAcceptedVariables(t *testing.T) {
	if SetOwnedItemQuantityDefinition.SupportedResourceTypes != "ItemDocument z capability stack" {
		t.Errorf("supported resource types = %q, want ItemDocument z capability stack",
			SetOwnedItemQuantityDefinition.SupportedResourceTypes)
	}
	want := []string{"saveSessionID", "characterID", "ownedItemID", "quantity", "expectedRevision"}
	if !reflect.DeepEqual(SetOwnedItemQuantityDefinition.SupportedResourceVariables, want) {
		t.Errorf("supported resource variables = %v, want %v",
			SetOwnedItemQuantityDefinition.SupportedResourceVariables, want)
	}
}
