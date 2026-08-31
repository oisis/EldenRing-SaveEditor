package inventory

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func setInventoryOrderEndpointTarget(
	t *testing.T,
) (*saveengine.Engine, string, []InventoryRecord) {
	t.Helper()
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetInventoryFixture(t, "pc", true, getInventoryAnchorAt), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := GetInventory(
		engine, inventoryCatalog(t), loaded.SaveSessionID, getInventorySlot, "common", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	return engine, loaded.SaveSessionID, inventory.Records
}

func TestSetInventoryOrderUsesConfirmedCatalogCategories(t *testing.T) {
	engine, sessionID, records := setInventoryOrderEndpointTarget(t)
	result, err := SetInventoryOrder(
		engine, inventoryCatalog(t), sessionID, getInventorySlot,
		[]string{records[1].OwnedItemID}, "0")
	if err != nil {
		t.Fatalf("SetInventoryOrder: %v", err)
	}
	if result.SaveRevision != "1" || len(result.OrderedResources) != 1 ||
		result.OrderedResources[0].Key != "100704E0" ||
		!reflect.DeepEqual(result.AcquisitionIndices, []uint32{434}) {
		t.Errorf("result = %+v", result)
	}
}

func TestSetInventoryOrderRejectsUnsupportedCatalogCategory(t *testing.T) {
	engine, sessionID, records := setInventoryOrderEndpointTarget(t)
	result, err := SetInventoryOrder(
		engine, inventoryCatalog(t), sessionID, getInventorySlot,
		[]string{records[0].OwnedItemID}, "0")
	if err == nil || !strings.Contains(err.Error(), "not supported by Inventory order") {
		t.Fatalf("error = %v, want unsupported-category rejection", err)
	}
	if !reflect.DeepEqual(result, SetInventoryOrderResult{}) {
		t.Errorf("result = %+v, want zero value", result)
	}
}

func TestSetInventoryOrderDefinitionMatchesRuntimeContract(t *testing.T) {
	want := []string{
		"saveSessionID", "characterID", "orderedOwnedItemIDs", "expectedRevision",
	}
	if !reflect.DeepEqual(SetInventoryOrderDefinition.SupportedResourceVariables, want) {
		t.Errorf("supported variables = %v, want %v",
			SetInventoryOrderDefinition.SupportedResourceVariables, want)
	}
}
