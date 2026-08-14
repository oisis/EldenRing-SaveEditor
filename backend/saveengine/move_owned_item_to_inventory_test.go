package saveengine

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const moveInventoryTestSlot = 1

func moveInventoryTestFixture() addItemTestFixture {
	return addItemTestFixture{
		common: []addItemTestRow{
			{index: 1, handle: addItemTestOtherHandle, rawQuantity: 2, acquisition: 440},
			{index: 4, handle: addItemTestTalismanHandle, rawQuantity: 1, acquisition: 460},
		},
		storage: []addItemTestRow{{
			index: 3, handle: addItemTestGoodsHandle,
			rawQuantity: 0x80000003, acquisition: 77,
		}},
		commonCount: 2, storageCount: 1,
		nextEquipIndex: 777, nextAcquisition: 468,
	}
}

func moveInventoryTestTarget(
	t *testing.T,
	platform Platform,
	content addItemTestFixture,
) (*Engine, string, string) {
	t.Helper()
	content.platform = platform
	content.slot = moveInventoryTestSlot
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(platform))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	storage, err := engine.GetStorage(
		loaded.SaveSessionID, moveInventoryTestSlot, StorageSectionCommon, 0, 0)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	for _, record := range storage.Records {
		if record.PhysicalIndex == 3 {
			return engine, loaded.SaveSessionID, record.OwnedItemID
		}
	}
	t.Fatal("the fixture source row was not identified")
	return nil, "", ""
}

func TestMoveOwnedItemToInventoryWritesBothPlatformsAndReloads(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine, sessionID, ownedItemID := moveInventoryTestTarget(
				t, platform, moveInventoryTestFixture())

			result, err := engine.MoveOwnedItemToInventory(
				sessionID, moveInventoryTestSlot, ownedItemID, 1, "0",
				addItemTestGoodsID, 99)
			if err != nil {
				t.Fatalf("MoveOwnedItemToInventory: %v", err)
			}
			if result.SaveRevision != "1" || result.GameID != addItemTestGoodsID ||
				result.Quantity != 3 || result.TargetPosition != 1 || result.PhysicalIndex != 0 ||
				result.AcquisitionIndex != 460 {
				t.Errorf("result = %+v", result)
			}

			inventory, err := engine.GetInventory(
				sessionID, moveInventoryTestSlot, InventorySectionCommon, 0, 0)
			if err != nil {
				t.Fatalf("GetInventory after move: %v", err)
			}
			if len(inventory.Records) != 3 {
				t.Fatalf("Inventory has %d records, want 3", len(inventory.Records))
			}
			logical := append([]InventoryRecord(nil), inventory.Records...)
			sort.Slice(logical, func(left, right int) bool {
				return logical[left].AcquisitionIndex < logical[right].AcquisitionIndex
			})
			wantHandles := []uint32{
				addItemTestOtherHandle, addItemTestGoodsHandle, addItemTestTalismanHandle,
			}
			for index, record := range logical {
				if record.GaItemHandle != wantHandles[index] {
					t.Errorf("logical Inventory record %d handle = 0x%08X, want 0x%08X",
						index, record.GaItemHandle, wantHandles[index])
				}
			}

			storage, err := engine.GetStorage(
				sessionID, moveInventoryTestSlot, StorageSectionCommon, 0, 0)
			if err != nil {
				t.Fatalf("GetStorage after move: %v", err)
			}
			if len(storage.Records) != 0 {
				t.Errorf("Storage records = %+v, want empty", storage.Records)
			}

			base := addItemTestSlotBase(t, platform, moveInventoryTestSlot) + addItemTestAnchorAt
			movedAt := base + addItemTestCommonAt
			if got := binary.LittleEndian.Uint32(removeTestBytes(
				t, engine, sessionID, movedAt+4, 4)); got != 0x80000003 {
				t.Errorf("raw moved quantity = 0x%08X, want 0x80000003", got)
			}
			if got := binary.LittleEndian.Uint32(removeTestBytes(
				t, engine, sessionID, base+addItemTestCommonCountAt, 4)); got != 3 {
				t.Errorf("Inventory count = %d, want 3", got)
			}
			if got := binary.LittleEndian.Uint32(removeTestBytes(
				t, engine, sessionID, base+addItemTestStorageAt, 4)); got != 0 {
				t.Errorf("Storage count = %d, want 0", got)
			}
			if got := binary.LittleEndian.Uint32(removeTestBytes(
				t, engine, sessionID, base+addItemTestNextEquipAt, 4)); got != 777 {
				t.Errorf("Inventory NextEquipIndex = %d, want unchanged 777", got)
			}
			if got := binary.LittleEndian.Uint32(removeTestBytes(
				t, engine, sessionID, base+addItemTestNextAcqAt, 4)); got != 470 {
				t.Errorf("Inventory NextAcquisitionSortId = %d, want 470", got)
			}

			target := filepath.Join(t.TempDir(), "moved.sl2")
			if _, err := engine.WriteSave(sessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform))
			if err != nil {
				t.Fatalf("LoadSave after WriteSave: %v", err)
			}
			reloadedInventory, err := reloadedEngine.GetInventory(
				reloaded.SaveSessionID, moveInventoryTestSlot, InventorySectionCommon, 0, 0)
			if err != nil {
				t.Fatalf("GetInventory after reload: %v", err)
			}
			if len(reloadedInventory.Records) != 3 ||
				reloadedInventory.Records[0].GaItemHandle != addItemTestGoodsHandle {
				t.Errorf("reloaded Inventory = %+v", reloadedInventory.Records)
			}
		})
	}
}

func TestMoveOwnedItemToInventoryRejectsInvalidPlansWithoutMutation(t *testing.T) {
	for name, testCase := range map[string]struct {
		change         func(*addItemTestFixture)
		targetPosition int
		maxInventory   uint32
		want           string
	}{
		"position": {targetPosition: 3, maxInventory: 99, want: "outside the range"},
		"limit":    {targetPosition: 0, maxInventory: 2, want: "above its Inventory limit"},
		"duplicate acquisition": {
			change: func(content *addItemTestFixture) {
				content.common[1].acquisition = content.common[0].acquisition
			},
			targetPosition: 0, maxInventory: 99, want: "duplicate acquisition index",
		},
	} {
		t.Run(name, func(t *testing.T) {
			content := moveInventoryTestFixture()
			if testCase.change != nil {
				testCase.change(&content)
			}
			engine, sessionID, ownedItemID := moveInventoryTestTarget(t, PlatformPC, content)
			before := addItemTestSlotData(t, engine, sessionID, PlatformPC, moveInventoryTestSlot)
			_, err := engine.MoveOwnedItemToInventory(
				sessionID, moveInventoryTestSlot, ownedItemID, testCase.targetPosition, "0",
				addItemTestGoodsID, testCase.maxInventory)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
			if after := addItemTestSlotData(
				t, engine, sessionID, PlatformPC, moveInventoryTestSlot); !bytes.Equal(after, before) {
				t.Error("a rejected move changed the slot")
			}
		})
	}
}

func TestMoveOwnedItemToInventoryRejectsStorageKey(t *testing.T) {
	content := moveInventoryTestFixture()
	content.storage = nil
	content.storageCount = 0
	content.storageKey = []addItemTestRow{{
		index: 3, handle: addItemTestGoodsHandle, rawQuantity: 1, acquisition: 77,
	}}
	content.storageKeyCount = 1
	content.platform = PlatformPC
	content.slot = moveInventoryTestSlot
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	storage, err := engine.GetStorage(
		loaded.SaveSessionID, moveInventoryTestSlot, StorageSectionKey, 0, 0)
	if err != nil || len(storage.Records) != 1 {
		t.Fatalf("GetStorage key: %v, records=%d", err, len(storage.Records))
	}
	_, err = engine.MoveOwnedItemToInventory(
		loaded.SaveSessionID, moveInventoryTestSlot, storage.Records[0].OwnedItemID,
		0, "0", addItemTestGoodsID, 99)
	if err == nil || !strings.Contains(err.Error(), "Storage key record") {
		t.Fatalf("error = %v, want key-record rejection", err)
	}
}
