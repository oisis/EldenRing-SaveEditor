package inventory

import (
	"encoding/binary"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func moveInventoryEndpointTarget(
	t *testing.T,
	physicalIndex int,
) (*saveengine.Engine, *gamecatalog.Catalog, string, string) {
	t.Helper()
	path := writeGetStorageFixture(t, "pc", true, getStorageAnchorAt)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Storage fixture: %v", err)
	}
	slotBase := int64(getStorageHeaderSize) + 0x10 + getStorageSlot*getStorageSlotBlockSize
	sectionAt := getStorageAnchorAt + int64(getStorageProjectileCountAt+4+
		getStorageProjectiles*getStorageProjectileStride+getStorageBlocksBefore)
	binary.LittleEndian.PutUint32(data[slotBase+sectionAt:], 2)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write Storage move fixture: %v", err)
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := inventoryCatalog(t)
	listed, err := GetStorage(engine, catalog, loaded.SaveSessionID, getStorageSlot, "common", 0, 0)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	for _, record := range listed.Records {
		if record.PhysicalIndex == physicalIndex {
			return engine, catalog, loaded.SaveSessionID, record.OwnedItemID
		}
	}
	t.Fatalf("common Storage row %d was not identified", physicalIndex)
	return nil, nil, "", ""
}

func TestMoveOwnedItemToInventoryUsesTheCatalogInventoryLimit(t *testing.T) {
	engine, catalog, sessionID, ownedItemID := moveInventoryEndpointTarget(t, 6)
	result, err := MoveOwnedItemToInventory(
		engine, catalog, sessionID, getStorageSlot, ownedItemID, 0, "0")
	if err != nil {
		t.Fatalf("MoveOwnedItemToInventory: %v", err)
	}
	if result.SaveRevision != "1" || result.GameID != 0x100704E0 ||
		result.Quantity != 1 || result.PhysicalIndex != 0 {
		t.Errorf("result = %+v", result)
	}

	inventory, err := GetInventory(
		engine, catalog, sessionID, getStorageSlot, "common", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	if len(inventory.Records) != 1 || inventory.Records[0].GameID != 0x100704E0 {
		t.Errorf("Inventory records = %+v", inventory.Records)
	}
}

func TestMoveOwnedItemToInventoryRejectsAmbiguousKeyCategory(t *testing.T) {
	engine, catalog, sessionID, ownedItemID := moveInventoryEndpointTarget(t, 1)
	result, err := MoveOwnedItemToInventory(
		engine, catalog, sessionID, getStorageSlot, ownedItemID, 0, "0")
	if err == nil || !strings.Contains(err.Error(), "key_items") {
		t.Fatalf("error = %v, want key-category rejection", err)
	}
	if !reflect.DeepEqual(result, MoveOwnedItemToInventoryResult{}) {
		t.Errorf("result = %+v, want zero value", result)
	}
}

func moveInventoryEndpointTargetWithCatalog(
	t *testing.T,
	catalog *gamecatalog.Catalog,
	physicalIndex int,
) (*saveengine.Engine, *gamecatalog.Catalog, string, string) {
	t.Helper()
	path := writeGetStorageFixture(t, "pc", true, getStorageAnchorAt)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Storage fixture: %v", err)
	}
	slotBase := int64(getStorageHeaderSize) + 0x10 + getStorageSlot*getStorageSlotBlockSize
	sectionAt := getStorageAnchorAt + int64(getStorageProjectileCountAt+4+
		getStorageProjectiles*getStorageProjectileStride+getStorageBlocksBefore)
	binary.LittleEndian.PutUint32(data[slotBase+sectionAt:], 2)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write Storage move fixture: %v", err)
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	listed, err := GetStorage(engine, catalog, loaded.SaveSessionID, getStorageSlot, "common", 0, 0)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	for _, record := range listed.Records {
		if record.PhysicalIndex == physicalIndex {
			return engine, catalog, loaded.SaveSessionID, record.OwnedItemID
		}
	}
	t.Fatalf("common Storage row %d was not identified", physicalIndex)
	return nil, nil, "", ""
}

func moveTestCatalog(t *testing.T, gameID uint32, mutate func(*schema.ItemDocument)) *gamecatalog.Catalog {
	t.Helper()
	target, exists := inventoryCatalog(t).ItemByGameID(gameID)
	if !exists {
		t.Fatalf("game ID 0x%08X is absent from the catalog", gameID)
	}

	resources := inventoryCatalogData.Resources()
	rewritten := false
	for index, resource := range resources {
		if resource.Kind != target.Kind || resource.Key != target.Key {
			continue
		}
		document := *resource.Item
		mutate(&document)
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

func TestMoveOwnedItemToInventoryRejectsUnknownRecordMode(t *testing.T) {
	catalog := moveTestCatalog(t, 0x100704E0, func(doc *schema.ItemDocument) {
		doc.Storage.RecordMode = schema.Fact[schema.RecordMode]{
			Known:      false,
			Provenance: doc.Storage.RecordMode.Provenance,
		}
	})
	engine, _, sessionID, ownedItemID := moveInventoryEndpointTargetWithCatalog(t, catalog, 6)

	result, err := MoveOwnedItemToInventory(
		engine, catalog, sessionID, getStorageSlot, ownedItemID, 0, "0")
	if err == nil || !strings.Contains(err.Error(), "unknown record mode") {
		t.Fatalf("error = %v, want unknown record mode rejection", err)
	}
	if !reflect.DeepEqual(result, MoveOwnedItemToInventoryResult{}) {
		t.Errorf("result = %+v, want zero value", result)
	}

	listed, err := GetStorage(engine, catalog, sessionID, getStorageSlot, "common", 0, 0)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if listed.SaveRevision != "0" || info.UnsavedChanges {
		t.Errorf("session state = revision %q, dirty %v; want revision 0, not dirty",
			listed.SaveRevision, info.UnsavedChanges)
	}
	found := false
	for _, record := range listed.Records {
		if record.OwnedItemID == ownedItemID && record.PhysicalIndex == 6 {
			found = true
			break
		}
	}
	if !found {
		t.Error("source Storage record was lost after rejected move")
	}
}

func TestMoveOwnedItemToInventoryContract(t *testing.T) {
	want := []string{
		"saveSessionID", "characterID", "ownedItemID", "targetPosition", "expectedRevision",
	}
	if !reflect.DeepEqual(MoveOwnedItemToInventoryDefinition.SupportedResourceVariables, want) {
		t.Errorf("supported variables = %v, want %v",
			MoveOwnedItemToInventoryDefinition.SupportedResourceVariables, want)
	}
}
