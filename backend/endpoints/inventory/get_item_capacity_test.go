package inventory

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestGetItemCapacityResolvesInventoryLimits(t *testing.T) {
	catalog := quantityTestCatalog(t, addItemTestDocument("tools", 600, 40))
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

	result, err := GetItemCapacity(
		engine, catalog, sessionID, getInventorySlot,
		saveengine.ItemCapacityDestinationInventory,
		addItemTestKind, addItemTestKey, nil, 5)
	if err != nil {
		t.Fatalf("GetItemCapacity: %v", err)
	}
	if result.Kind != schema.ResourceKindItem || result.Key != addItemTestKey ||
		result.GameID != addItemTestEndpointGameID || !result.CanFit ||
		result.CurrentQuantity != 3 || result.QuantityAfter != 8 ||
		result.MaxContainerQuantity != 600 || result.PhysicalRecordsRequired != 0 {
		t.Errorf("GetItemCapacity = %+v", result)
	}
	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if result.SaveRevision != "0" || info.UnsavedChanges {
		t.Errorf("getter changed revision/state: result=%+v info=%+v", result, info)
	}
}

func TestGetItemCapacityRejectsUnsupportedCatalogSemantics(t *testing.T) {
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	tests := []struct {
		name        string
		catalog     func(*schema.ItemDocument)
		destination string
		want        string
	}{
		{
			name: "flask shared limit",
			catalog: addItemTestThen("tools", 20, 20, func(item *schema.ItemDocument) {
				item.Subcategory = schema.Fact[string]{
					Known: true, Value: addItemFlasksSubcategory,
					Provenance: item.Subcategory.Provenance,
				}
			}),
			destination: saveengine.ItemCapacityDestinationInventory,
			want:        "shared charge limit",
		},
		{
			name: "non-depositable goods",
			catalog: addItemTestThen("tools", 600, 40, func(item *schema.ItemDocument) {
				item.Storage.MaxStorage = schema.Fact[uint32]{
					Known: true, Value: 600,
					Provenance: item.Storage.MaxStorage.Provenance,
				}
			}),
			destination: saveengine.ItemCapacityDestinationStorage,
			want:        "cannot be deposited",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			catalog := quantityTestCatalog(t, testCase.catalog)
			_, err := GetItemCapacity(
				engine, catalog, sessionID, getInventorySlot, testCase.destination,
				addItemTestKind, addItemTestKey, nil, 1)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("GetItemCapacity error = %v, want %q", err, testCase.want)
			}
		})
	}
}
