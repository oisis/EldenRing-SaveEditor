package inventory

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func addStorageTestDocument(
	depositable bool,
	maxStorage uint32,
) func(*schema.ItemDocument) {
	return addItemTestThen("tools", 600, 40, func(document *schema.ItemDocument) {
		document.Storage.MaxStorage = schema.Fact[uint32]{
			Known: true, Value: maxStorage,
			Provenance: document.Storage.MaxStorage.Provenance,
		}
		goods := *document.Goods
		document.Goods = &goods
		document.Goods.IsDepositable = schema.Fact[bool]{
			Known: true, Value: depositable,
			Provenance: document.Goods.IsDepositable.Provenance,
		}
	})
}

func TestAddItemToStorageUsesCatalogStorageRules(t *testing.T) {
	catalog := quantityTestCatalog(t, addStorageTestDocument(true, 600))
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

	result, err := AddItemToStorage(
		engine, catalog, sessionID, getInventorySlot,
		addItemTestKind, addItemTestKey, nil, 5, "0")
	if err != nil {
		t.Fatalf("AddItemToStorage: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, sessionID, AddItemToStorageEndpointID, "1")
	// The receipt is pinned from the result because operationID names one
	// execution and cannot be predicted; every other member is asserted above.
	want := AddItemToStorageResult{
		MutationReceipt: result.MutationReceipt, CharacterID: getInventorySlot,
		GameID: addItemTestEndpointGameID, Added: 5, Quantity: 5, CreatedRecord: true,
		ContainerSection: saveengine.StorageSectionCommon, PhysicalIndex: 0,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("AddItemToStorage = %+v, want %+v", result, want)
	}

	stored, err := engine.GetStorage(sessionID, getInventorySlot, saveengine.StorageSectionCommon, 0, 0)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	if stored.SaveRevision != "1" || len(stored.Records) != 1 ||
		stored.Records[0].Quantity != 5 || stored.Records[0].GaItemHandle != 0xB000272E {
		t.Errorf("Storage after add = %+v", stored)
	}
}

func TestAddItemToStorageRejectsUnsupportedCatalogData(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		catalog func(*testing.T) *gamecatalog.Catalog
		want    string
	}{
		{
			name: "non-depositable goods",
			catalog: func(t *testing.T) *gamecatalog.Catalog {
				return quantityTestCatalog(t, addStorageTestDocument(false, 600))
			},
			want: "cannot be deposited",
		},
		{
			name: "unknown storage limit",
			catalog: func(t *testing.T) *gamecatalog.Catalog {
				return quantityTestCatalog(t, addItemTestThen("tools", 600, 40,
					func(document *schema.ItemDocument) {
						document.Storage.MaxStorage.Known = false
					}))
			},
			want: "no storage limit",
		},
		{
			name: "Flask shared limit",
			catalog: func(t *testing.T) *gamecatalog.Catalog {
				return quantityTestCatalog(t, addItemTestThen("tools", 600, 40,
					func(document *schema.ItemDocument) {
						document.Subcategory = schema.Fact[string]{
							Known: true, Value: addItemFlasksSubcategory,
							Provenance: document.Subcategory.Provenance,
						}
					}))
			},
			want: "shared charge limit",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
			_, err := AddItemToStorage(
				engine, testCase.catalog(t), sessionID, getInventorySlot,
				addItemTestKind, addItemTestKey, nil, 1, "0")
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("AddItemToStorage error = %v, want %q", err, testCase.want)
			}
			info, infoErr := engine.GetSessionInfo(sessionID)
			if infoErr != nil {
				t.Fatalf("GetSessionInfo: %v", infoErr)
			}
			stored, storageErr := engine.GetStorage(
				sessionID, getInventorySlot, saveengine.StorageSectionCommon, 0, 0)
			if storageErr != nil {
				t.Fatalf("GetStorage: %v", storageErr)
			}
			if stored.SaveRevision != "0" || len(stored.Records) != 0 || info.UnsavedChanges {
				t.Errorf("rejected add left revision %q, dirty %v",
					stored.SaveRevision, info.UnsavedChanges)
			}
		})
	}
}

func TestAddItemToStorageRequiresItsDependencies(t *testing.T) {
	catalog := quantityTestCatalog(t, addStorageTestDocument(true, 600))
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	if _, err := AddItemToStorage(
		nil, catalog, sessionID, getInventorySlot,
		addItemTestKind, addItemTestKey, nil, 1, "0"); err == nil {
		t.Error("a missing save engine was accepted")
	}
	if _, err := AddItemToStorage(
		engine, nil, sessionID, getInventorySlot,
		addItemTestKind, addItemTestKey, nil, 1, "0"); err == nil {
		t.Error("a missing game catalog was accepted")
	}
}
