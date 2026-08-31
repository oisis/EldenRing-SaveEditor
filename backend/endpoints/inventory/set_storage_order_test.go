package inventory

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func setStorageOrderEndpointTarget(
	t *testing.T,
) (*saveengine.Engine, string, []StorageRecord) {
	t.Helper()
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetStorageFixture(t, "pc", true, getStorageAnchorAt), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	storage, err := GetStorage(
		engine, inventoryCatalog(t), loaded.SaveSessionID, getStorageSlot, "common", 0, 0)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	return engine, loaded.SaveSessionID, storage.Records
}

func TestSetStorageOrderUsesConfirmedCatalogCategories(t *testing.T) {
	engine, sessionID, records := setStorageOrderEndpointTarget(t)
	result, err := SetStorageOrder(
		engine, inventoryCatalog(t), sessionID, getStorageSlot,
		[]string{records[1].OwnedItemID}, "0")
	if err != nil {
		t.Fatalf("SetStorageOrder: %v", err)
	}
	if result.SaveRevision != "1" || len(result.OrderedResources) != 1 ||
		result.OrderedResources[0].Key != "100704E0" ||
		!reflect.DeepEqual(result.AcquisitionIndices, []uint32{14}) {
		t.Errorf("result = %+v", result)
	}
}

func TestSetStorageOrderRejectsUnsupportedCatalogCategory(t *testing.T) {
	engine, sessionID, records := setStorageOrderEndpointTarget(t)
	result, err := SetStorageOrder(
		engine, inventoryCatalog(t), sessionID, getStorageSlot,
		[]string{records[0].OwnedItemID}, "0")
	if err == nil || !strings.Contains(err.Error(), "not supported by Storage order") {
		t.Fatalf("error = %v, want unsupported-category rejection", err)
	}
	if !reflect.DeepEqual(result, SetStorageOrderResult{}) {
		t.Errorf("result = %+v, want zero value", result)
	}
}

func TestSetStorageOrderDefinitionMatchesRuntimeContract(t *testing.T) {
	want := []string{
		"saveSessionID", "characterID", "orderedOwnedItemIDs", "expectedRevision",
	}
	if !reflect.DeepEqual(SetStorageOrderDefinition.SupportedResourceVariables, want) {
		t.Errorf("supported variables = %v, want %v",
			SetStorageOrderDefinition.SupportedResourceVariables, want)
	}
}

func TestSetStorageOrderLeavesSetInventoryOrderUnaffected(t *testing.T) {
	// Guard: verify SetInventoryOrder continues to enforce the 434 floor.
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
		t.Errorf("SetInventoryOrder result = %+v, want floor 434 intact", result)
	}
}

func TestSetStorageOrderOnPS4(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetStorageFixture(t, "ps4", true, getStorageAnchorAt), "ps4", "local")
	if err != nil {
		t.Fatalf("LoadSave PS4: %v", err)
	}
	storage, err := GetStorage(
		engine, inventoryCatalog(t), loaded.SaveSessionID, getStorageSlot, "common", 0, 0)
	if err != nil {
		t.Fatalf("GetStorage PS4: %v", err)
	}
	result, err := SetStorageOrder(
		engine, inventoryCatalog(t), loaded.SaveSessionID, getStorageSlot,
		[]string{storage.Records[1].OwnedItemID}, "0")
	if err != nil {
		t.Fatalf("SetStorageOrder PS4: %v", err)
	}
	if result.SaveRevision != "1" || len(result.OrderedResources) != 1 ||
		result.OrderedResources[0].Key != "100704E0" ||
		!reflect.DeepEqual(result.AcquisitionIndices, []uint32{14}) {
		t.Errorf("PS4 result = %+v", result)
	}
}
