package inventory

import (
	"encoding/binary"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
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
	loaded, err := engine.LoadSave(path, "pc")
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

func TestMoveOwnedItemToInventoryContract(t *testing.T) {
	want := []string{
		"saveSessionID", "characterID", "ownedItemID", "targetPosition", "expectedRevision",
	}
	if !reflect.DeepEqual(MoveOwnedItemToInventoryDefinition.SupportedResourceVariables, want) {
		t.Errorf("supported variables = %v, want %v",
			MoveOwnedItemToInventoryDefinition.SupportedResourceVariables, want)
	}
}
