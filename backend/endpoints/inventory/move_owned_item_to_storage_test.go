package inventory

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func moveStorageEndpointTarget(
	t *testing.T,
	physicalIndex int,
) (*saveengine.Engine, *gamecatalog.Catalog, string, string) {
	t.Helper()
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	catalog := inventoryCatalog(t)
	listed, err := GetInventory(engine, catalog, sessionID, getInventorySlot, "common", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	for _, record := range listed.Records {
		if record.PhysicalIndex == physicalIndex {
			return engine, catalog, sessionID, record.OwnedItemID
		}
	}
	t.Fatalf("common Inventory row %d was not identified", physicalIndex)
	return nil, nil, "", ""
}

func TestMoveOwnedItemToStorageUsesTheCatalogStorageLimit(t *testing.T) {
	engine, catalog, sessionID, ownedItemID := moveStorageEndpointTarget(t, 6)
	result, err := MoveOwnedItemToStorage(
		engine, catalog, sessionID, getInventorySlot,
		ownedItemID, 0, "0")
	if err != nil {
		t.Fatalf("MoveOwnedItemToStorage: %v", err)
	}
	if result.SaveRevision != "1" || result.GameID != 0x100704E0 ||
		result.Quantity != 1 || result.PhysicalIndex != 0 {
		t.Errorf("result = %+v", result)
	}

	storage, err := GetStorage(
		engine, catalog, sessionID, getInventorySlot, "common", 0, 0)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	if len(storage.Records) != 1 || storage.Records[0].GameID != 0x100704E0 {
		t.Errorf("Storage records = %+v", storage.Records)
	}
}

func TestMoveOwnedItemToStorageRejectsAnItemWithZeroStorageLimit(t *testing.T) {
	engine, catalog, sessionID, ownedItemID := moveStorageEndpointTarget(t, 1)
	result, err := MoveOwnedItemToStorage(
		engine, catalog, sessionID, getInventorySlot,
		ownedItemID, 0, "0")
	if err == nil || !strings.Contains(err.Error(), "cannot be stored") {
		t.Fatalf("error = %v, want storage-limit rejection", err)
	}
	if !reflect.DeepEqual(result, MoveOwnedItemToStorageResult{}) {
		t.Errorf("result = %+v, want zero value", result)
	}
}

func TestMoveOwnedItemToStorageRejectsUnknownRecordMode(t *testing.T) {
	catalog := moveTestCatalog(t, 0x100704E0, func(doc *schema.ItemDocument) {
		doc.Storage.RecordMode = schema.Fact[schema.RecordMode]{
			Known:      false,
			Provenance: doc.Storage.RecordMode.Provenance,
		}
	})
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	listed, err := GetInventory(engine, catalog, sessionID, getInventorySlot, "common", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	var ownedItemID string
	for _, record := range listed.Records {
		if record.PhysicalIndex == 6 {
			ownedItemID = record.OwnedItemID
			break
		}
	}
	if ownedItemID == "" {
		t.Fatal("common Inventory row 6 was not identified")
	}

	result, err := MoveOwnedItemToStorage(
		engine, catalog, sessionID, getInventorySlot, ownedItemID, 0, "0")
	if err == nil || !strings.Contains(err.Error(), "unknown record mode") {
		t.Fatalf("error = %v, want unknown record mode rejection", err)
	}
	if !reflect.DeepEqual(result, MoveOwnedItemToStorageResult{}) {
		t.Errorf("result = %+v, want zero value", result)
	}

	listedAfter, err := GetInventory(engine, catalog, sessionID, getInventorySlot, "common", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if listedAfter.SaveRevision != "0" || info.UnsavedChanges {
		t.Errorf("session state = revision %q, dirty %v; want revision 0, not dirty",
			listedAfter.SaveRevision, info.UnsavedChanges)
	}
	found := false
	for _, record := range listedAfter.Records {
		if record.OwnedItemID == ownedItemID && record.PhysicalIndex == 6 {
			found = true
			break
		}
	}
	if !found {
		t.Error("source Inventory record was lost after rejected move")
	}
}

func TestMoveOwnedItemToStorageContract(t *testing.T) {
	want := []string{
		"saveSessionID", "characterID", "ownedItemID", "targetPosition", "expectedRevision",
	}
	if !reflect.DeepEqual(MoveOwnedItemToStorageDefinition.SupportedResourceVariables, want) {
		t.Errorf("supported variables = %v, want %v",
			MoveOwnedItemToStorageDefinition.SupportedResourceVariables, want)
	}
}
