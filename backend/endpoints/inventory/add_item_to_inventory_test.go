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
// that decides the family, the routing, the record mode and the two limits, and
// the pass-through of everything else. The mutation, the revision check, the
// physical row, the counters, the GaItemData entry and the rollback belong to
// backend/saveengine/add_item_to_inventory_test.go and are not repeated here.
//
// The Inventory fixture of this package is reused unchanged. Its common row 1
// carries the goods handle of addItemTestEndpointGameID with quantity 3, so an
// add of that resource is a top-up and every other resource opens a new record.
const (
	addItemTestKind           = "item"
	addItemTestKey            = "4000272E"
	addItemTestEndpointGameID = 0x4000272E

	// A real talisman document, used unchanged: family talisman, category
	// talismans, separate instances, one per inventory.
	addItemTestTalismanKey    = "200003E8"
	addItemTestTalismanGameID = 0x200003E8

	// A real key-routed goods document, used unchanged.
	addItemTestKeyItemKey = "40002134"
	addItemTestMapKey     = "400021A3"

	// Two real documents that carry the goods game-ID prefix without being goods,
	// so only the family gate keeps them out.
	addItemTestSpellKey     = "40000FA0"
	addItemTestSpiritAshKey = "40030D40"
)

// addItemTestDocument states the fully known shape of a stacking goods resource.
// Every test starts from it and states only the numbers, or the single fact, it
// is about.
func addItemTestDocument(
	category string, maxInventory, maxPerStack uint32,
) func(*schema.ItemDocument) {
	return func(document *schema.ItemDocument) {
		document.Family = schema.Fact[schema.ItemFamily]{
			Known:      true,
			Value:      schema.ItemFamilyGoods,
			Provenance: document.Family.Provenance,
		}
		document.Category = schema.Fact[string]{
			Known:      true,
			Value:      category,
			Provenance: document.Category.Provenance,
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
		source := document.Capabilities.Stack.Provenance
		document.Capabilities.Stack = schema.Capability[schema.StackRules]{
			Known:         true,
			Enabled:       true,
			Rules:         &schema.StackRules{MaxPerStack: maxPerStack},
			Provenance:    source,
			RulesEvidence: []schema.Provenance{source},
		}
	}
}

// addItemTestThen applies the known shape and then the single deviation one
// rejection case is about.
func addItemTestThen(
	category string, maxInventory, maxPerStack uint32, deviate func(*schema.ItemDocument),
) func(*schema.ItemDocument) {
	known := addItemTestDocument(category, maxInventory, maxPerStack)
	return func(document *schema.ItemDocument) {
		known(document)
		deviate(document)
	}
}

// addItemTestState re-reads the addressed row and reports what a rejected
// mutation has to leave untouched: the stored quantity, the number of records,
// the session revision and the unsaved-changes flag.
func addItemTestState(
	t *testing.T, engine *saveengine.Engine, catalog *gamecatalog.Catalog, sessionID string,
) (uint32, int, string, bool) {
	t.Helper()

	listed, err := GetInventory(engine, catalog, sessionID, getInventorySlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	var quantity uint32
	found := false
	for _, record := range listed.Records {
		if record.ContainerSection == "common" && record.PhysicalIndex == 1 {
			quantity, found = record.Quantity, true
		}
	}
	if !found {
		t.Fatalf("the inventory read no longer contains common#1")
	}
	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	return quantity, listed.Total, listed.SaveRevision, info.UnsavedChanges
}

func TestAddItemToInventoryTopsUpThroughTheCatalogLimits(t *testing.T) {
	catalog := quantityTestCatalog(t, addItemTestDocument("tools", 600, 40))
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

	result, err := AddItemToInventory(
		engine, catalog, sessionID, getInventorySlot, addItemTestKind, addItemTestKey, nil, 5, "0")
	if err != nil {
		t.Fatalf("AddItemToInventory: %v", err)
	}

	assertMutationReceipt(t, result.MutationReceipt, sessionID, AddItemToInventoryEndpointID, "1")
	// The receipt is pinned from the result because operationID names one
	// execution and cannot be predicted; every other member is asserted above.
	want := AddItemToInventoryResult{
		MutationReceipt: result.MutationReceipt, CharacterID: getInventorySlot,
		GameID: addItemTestEndpointGameID, Added: 5, Quantity: 8, CreatedRecord: false,
		ContainerSection: "common", PhysicalIndex: 1,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("AddItemToInventory = %+v, want %+v", result, want)
	}

	// quantity is a delta, so the record holds the previous 3 plus the added 5
	// rather than the 5 that were requested.
	quantity, total, revision, dirty := addItemTestState(t, engine, catalog, sessionID)
	if quantity != 8 || total != 3 || revision != "1" || !dirty {
		t.Errorf("after the add common#1 holds %d of %d records at revision %q, dirty %v",
			quantity, total, revision, dirty)
	}
}

func TestAddItemToInventoryDerivesBothLimitsFromTheDocument(t *testing.T) {
	// The per-record limit is the smaller of the stack limit and the container
	// limit, and the container limit covers every record of the item. Each case
	// crosses exactly one of them.
	cases := []struct {
		name         string
		maxInventory uint32
		maxPerStack  uint32
		quantity     uint32
		wants        string
	}{
		{"stack limit", 600, 4, 2, "per record"},
		{"container limit below the stack limit", 4, 40, 2, "above the limit of 4"},
		{"stack limit above the container limit", 4, 40, 5, "exceeds the limit of 4 per record"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			catalog := quantityTestCatalog(t,
				addItemTestDocument("tools", testCase.maxInventory, testCase.maxPerStack))
			engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

			_, err := AddItemToInventory(engine, catalog, sessionID, getInventorySlot,
				addItemTestKind, addItemTestKey, nil, testCase.quantity, "0")
			if err == nil {
				t.Fatalf("the request was accepted, want a rejection mentioning %q", testCase.wants)
			}
			if !strings.Contains(err.Error(), testCase.wants) {
				t.Errorf("error %q does not mention %q", err, testCase.wants)
			}
			quantity, total, revision, dirty := addItemTestState(t, engine, catalog, sessionID)
			if quantity != 3 || total != 3 || revision != "0" || dirty {
				t.Errorf("the rejected add left common#1 at %d of %d records, revision %q, dirty %v",
					quantity, total, revision, dirty)
			}
		})
	}
}

func TestAddItemToInventoryCreatesOneRecordPerTalismanCopy(t *testing.T) {
	// The talisman document is used exactly as it ships: separate instances, one
	// per inventory. The first copy opens a record, and a second one crosses the
	// container limit the same document declares.
	catalog := inventoryCatalog(t)
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

	result, err := AddItemToInventory(engine, catalog, sessionID, getInventorySlot,
		addItemTestKind, addItemTestTalismanKey, nil, 1, "0")
	if err != nil {
		t.Fatalf("AddItemToInventory: %v", err)
	}
	if !result.CreatedRecord || result.Quantity != 1 ||
		result.GameID != addItemTestTalismanGameID {
		t.Errorf("AddItemToInventory = %+v, want a new record holding one talisman", result)
	}

	if _, err := AddItemToInventory(engine, catalog, sessionID, getInventorySlot,
		addItemTestKind, addItemTestTalismanKey, nil, 2, "1"); err == nil {
		t.Error("a separate-instances resource accepted a quantity above one")
	}
	if _, err := AddItemToInventory(engine, catalog, sessionID, getInventorySlot,
		addItemTestKind, addItemTestTalismanKey, nil, 1, "1"); err == nil {
		t.Error("a second copy was accepted above the declared inventory limit of one")
	}
}

// Three guards of this endpoint are unreachable from a validated catalog and
// are therefore not exercised here: an unknown family, a family whose
// family-specific document is missing, and a stack capability enabled without
// rules are all rejected by schema validation itself, so no catalog the endpoint
// can be handed carries one. They stay in the endpoint as the fail-closed
// default for a catalog assembled another way.
func TestAddItemToInventoryRejectsWhatTheCatalogDoesNotAllow(t *testing.T) {
	cases := []struct {
		name    string
		catalog func(*testing.T) *gamecatalog.Catalog
		key     string
		variant *uint32
		wants   string
		// A case that deliberately changes the fixture record's catalog game ID
		// re-reads the unchanged save through the original catalog after rejection.
		checkStateWithBaseCatalog bool
	}{
		{
			name:  "key-routed resource",
			key:   addItemTestKeyItemKey,
			wants: "does not distinguish common from key routing",
		},
		{
			// Maps are key_items resources that legacy kept in common. Rejecting this
			// real document proves that the category gate is deliberately broader than
			// the confirmed key-routed subset rather than an accidental ID list.
			name:  "common-routed map in the broad key category",
			key:   addItemTestMapKey,
			wants: "does not distinguish common from key routing",
		},
		{
			// A spell carries the same game-ID prefix as a goods resource, so only
			// the family gate keeps it out.
			name:  "spell behind the goods prefix",
			key:   addItemTestSpellKey,
			wants: "is of family",
		},
		{
			// So does a spirit ash.
			name:  "spirit ash behind the goods prefix",
			key:   addItemTestSpiritAshKey,
			wants: "is of family",
		},
		{
			name: "family and game ID prefix disagree",
			catalog: func(t *testing.T) *gamecatalog.Catalog {
				return quantityTestCatalog(t, addItemTestThen("tools", 600, 40,
					func(document *schema.ItemDocument) {
						document.GameID.Value = 0x2FFFFFFE
					}))
			},
			wants:                     "which disagree",
			checkStateWithBaseCatalog: true,
		},
		{
			name: "unknown category",
			catalog: func(t *testing.T) *gamecatalog.Catalog {
				return quantityTestCatalog(t, addItemTestThen("tools", 600, 40,
					func(document *schema.ItemDocument) { document.Category.Known = false }))
			},
			wants: "unknown category",
		},
		{
			name: "unknown inventory limit",
			catalog: func(t *testing.T) *gamecatalog.Catalog {
				return quantityTestCatalog(t, addItemTestThen("tools", 600, 40,
					func(document *schema.ItemDocument) { document.Storage.MaxInventory.Known = false }))
			},
			wants: "no inventory limit",
		},
		{
			name: "zero inventory limit",
			catalog: func(t *testing.T) *gamecatalog.Catalog {
				return quantityTestCatalog(t, addItemTestDocument("tools", 0, 40))
			},
			wants: "no inventory limit",
		},
		{
			name: "unknown record mode",
			catalog: func(t *testing.T) *gamecatalog.Catalog {
				return quantityTestCatalog(t, addItemTestThen("tools", 600, 40,
					func(document *schema.ItemDocument) { document.Storage.RecordMode.Known = false }))
			},
			wants: "unknown record mode",
		},
		{
			name: "unknown stack capability",
			catalog: func(t *testing.T) *gamecatalog.Catalog {
				return quantityTestCatalog(t, addItemTestThen("tools", 600, 40,
					func(document *schema.ItemDocument) {
						document.Capabilities.Stack = schema.Capability[schema.StackRules]{
							Provenance: document.Capabilities.Stack.Provenance,
						}
					}))
			},
			wants: "unknown stack capability",
		},
		{
			name: "quantity stack that does not stack",
			catalog: func(t *testing.T) *gamecatalog.Catalog {
				return quantityTestCatalog(t, addItemTestThen("tools", 600, 40,
					func(document *schema.ItemDocument) {
						source := document.Capabilities.Stack.Provenance
						document.Capabilities.Stack = schema.Capability[schema.StackRules]{
							Known: true, Enabled: false, Provenance: source,
						}
					}))
			},
			wants: "does not stack",
		},
		{
			name:  "unknown key",
			key:   "no-such-key",
			wants: "unknown resource key",
		},
		{
			name:    "unknown variant",
			variant: func() *uint32 { id := uint32(0xDEADBEEF); return &id }(),
			wants:   "unknown variant ID",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			catalog := inventoryCatalog(t)
			if testCase.catalog != nil {
				catalog = testCase.catalog(t)
			}
			key := addItemTestKey
			if testCase.key != "" {
				key = testCase.key
			}
			engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

			_, err := AddItemToInventory(engine, catalog, sessionID, getInventorySlot,
				addItemTestKind, key, testCase.variant, 1, "0")
			if err == nil {
				t.Fatalf("the request was accepted, want a rejection mentioning %q", testCase.wants)
			}
			if !strings.Contains(err.Error(), testCase.wants) {
				t.Errorf("error %q does not mention %q", err, testCase.wants)
			}
			stateCatalog := catalog
			if testCase.checkStateWithBaseCatalog {
				stateCatalog = inventoryCatalog(t)
			}
			quantity, total, revision, dirty := addItemTestState(t, engine, stateCatalog, sessionID)
			if quantity != 3 || total != 3 || revision != "0" || dirty {
				t.Errorf("the rejected add left common#1 at %d of %d records, revision %q, dirty %v",
					quantity, total, revision, dirty)
			}
		})
	}
}

func TestAddItemToInventoryRejectsAnUnknownKind(t *testing.T) {
	catalog := inventoryCatalog(t)
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

	if _, err := AddItemToInventory(engine, catalog, sessionID, getInventorySlot,
		"gesture", addItemTestKey, nil, 1, "0"); err == nil {
		t.Error("a kind that carries no item document was accepted")
	}
}

func TestAddItemToInventoryRequiresItsDependencies(t *testing.T) {
	catalog := inventoryCatalog(t)
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

	if _, err := AddItemToInventory(nil, catalog, sessionID, getInventorySlot,
		addItemTestKind, addItemTestKey, nil, 1, "0"); err == nil {
		t.Error("a missing save engine was accepted")
	}
	if _, err := AddItemToInventory(engine, nil, sessionID, getInventorySlot,
		addItemTestKind, addItemTestKey, nil, 1, "0"); err == nil {
		t.Error("a missing game catalog was accepted")
	}
}
