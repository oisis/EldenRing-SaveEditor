package saveengine

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"strings"
	"testing"
)

const moveStorageTestSlot = 1

func moveStorageTestTarget(
	t *testing.T,
	platform Platform,
	content addItemTestFixture,
) (*Engine, string, string) {
	t.Helper()
	content.platform = platform
	content.slot = moveStorageTestSlot
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(platform))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, moveStorageTestSlot, InventorySectionCommon, 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	for _, record := range inventory.Records {
		if record.PhysicalIndex == 2 {
			return engine, loaded.SaveSessionID, record.OwnedItemID
		}
	}
	t.Fatal("the fixture source row was not identified")
	return nil, "", ""
}

func moveStorageTestFixture() addItemTestFixture {
	return addItemTestFixture{
		common: []addItemTestRow{{
			index: 2, handle: addItemTestGoodsHandle,
			rawQuantity: 0x80000003, acquisition: 17,
		}},
		storage: []addItemTestRow{
			{index: 1, handle: addItemTestOtherHandle, rawQuantity: 1, acquisition: 440},
			{index: 4, handle: addItemTestTalismanHandle, rawQuantity: 1, acquisition: 450},
		},
		commonCount:  1,
		storageCount: 2,
	}
}

func TestMoveOwnedItemToStorageWritesBothPlatformsAndReloads(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine, sessionID, ownedItemID := moveStorageTestTarget(
				t, platform, moveStorageTestFixture())

			result, err := engine.MoveOwnedItemToStorage(
				sessionID, moveStorageTestSlot, ownedItemID, 1, "0",
				addItemTestGoodsID, 99)
			if err != nil {
				t.Fatalf("MoveOwnedItemToStorage: %v", err)
			}
			if result.SaveRevision != "1" || result.GameID != addItemTestGoodsID ||
				result.Quantity != 3 || result.TargetPosition != 1 || result.PhysicalIndex != 4 ||
				result.AcquisitionIndex != 452 {
				t.Errorf("result = %+v", result)
			}

			inventory, err := engine.GetInventory(
				sessionID, moveStorageTestSlot, InventorySectionCommon, 0, 0)
			if err != nil {
				t.Fatalf("GetInventory after move: %v", err)
			}
			if len(inventory.Records) != 0 {
				t.Errorf("Inventory records = %+v, want empty", inventory.Records)
			}
			storage, err := engine.GetStorage(
				sessionID, moveStorageTestSlot, StorageSectionCommon, 0, 0)
			if err != nil {
				t.Fatalf("GetStorage after move: %v", err)
			}
			if len(storage.Records) != 3 {
				t.Fatalf("Storage has %d records, want 3", len(storage.Records))
			}
			wantHandles := []uint32{
				addItemTestOtherHandle, addItemTestGoodsHandle, addItemTestTalismanHandle,
			}
			for index, record := range storage.Records {
				if record.GaItemHandle != wantHandles[index] {
					t.Errorf("Storage record %d handle = 0x%08X, want 0x%08X",
						index, record.GaItemHandle, wantHandles[index])
				}
			}
			if storage.Records[1].Quantity != 3 || storage.Records[1].AcquisitionIndex != 452 {
				t.Errorf("moved Storage record = %+v", storage.Records[1])
			}

			base := addItemTestSlotBase(t, platform, moveStorageTestSlot) + addItemTestAnchorAt
			if got := binary.LittleEndian.Uint32(removeTestBytes(
				t, engine, sessionID, base+addItemTestCommonCountAt, 4)); got != 0 {
				t.Errorf("Inventory count = %d, want 0", got)
			}
			if got := binary.LittleEndian.Uint32(removeTestBytes(
				t, engine, sessionID, base+addItemTestStorageAt, 4)); got != 3 {
				t.Errorf("Storage count = %d, want 3", got)
			}
			movedQuantityAt := base + addItemTestStorageAt + addItemTestStorageCommonAt +
				4*addItemTestRecordSize + 4
			if got := binary.LittleEndian.Uint32(removeTestBytes(
				t, engine, sessionID, movedQuantityAt, 4)); got != 0x80000003 {
				t.Errorf("raw moved quantity = 0x%08X, want 0x80000003", got)
			}
			storageNextEquipAt := base + addItemTestStorageAt +
				addItemTestStorageKeyAt + 0x80*addItemTestRecordSize
			if got := binary.LittleEndian.Uint32(removeTestBytes(
				t, engine, sessionID, storageNextEquipAt, 4)); got != 128 {
				t.Errorf("Storage NextEquipIndex = %d, want 128", got)
			}
			storageNextAcqAt := storageNextEquipAt + 4
			if got := binary.LittleEndian.Uint32(removeTestBytes(
				t, engine, sessionID, storageNextAcqAt, 4)); got != 227 {
				t.Errorf("Storage NextAcquisitionSortId = %d, want 227", got)
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
			reloadedStorage, err := reloadedEngine.GetStorage(
				reloaded.SaveSessionID, moveStorageTestSlot, StorageSectionCommon, 0, 0)
			if err != nil {
				t.Fatalf("GetStorage after reload: %v", err)
			}
			if len(reloadedStorage.Records) != 3 ||
				reloadedStorage.Records[1].GaItemHandle != addItemTestGoodsHandle {
				t.Errorf("reloaded Storage = %+v", reloadedStorage.Records)
			}
		})
	}
}

func TestMoveOwnedItemToStorageMatchesAddItemAllocatorCounters(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			// T330 shape in Storage (6 records, max index 12, equip 133, acq 7)
			rows := make([]addItemTestRow, 6)
			for i := 0; i < 6; i++ {
				rows[i] = addItemTestRow{
					index:       i,
					handle:      addItemTestFillHandleBase + uint32(i),
					rawQuantity: 1,
					acquisition: uint32((i + 1) * 2), // 2, 4, 6, 8, 10, 12
				}
			}
			content := addItemTestFixture{
				platform: platform,
				slot:     moveStorageTestSlot,
				common: []addItemTestRow{{
					index:       2,
					handle:      addItemTestGoodsHandle,
					rawQuantity: 0x80000001,
					acquisition: 17,
				}},
				storage:      rows,
				commonCount:  1,
				storageCount: 6,
			}

			engine, sessionID, ownedItemID := moveStorageTestTarget(t, platform, content)
			setAddStorageTestCounters(t, engine, sessionID, platform, moveStorageTestSlot, 133, 7)

			result, err := engine.MoveOwnedItemToStorage(
				sessionID, moveStorageTestSlot, ownedItemID, 6, "0", addItemTestGoodsID, 99)
			if err != nil {
				t.Fatalf("MoveOwnedItemToStorage: %v", err)
			}
			if result.AcquisitionIndex != 14 {
				t.Errorf("moved acquisitionIndex = %d, want 14", result.AcquisitionIndex)
			}
			if result.AcquisitionIndex%2 != 0 {
				t.Errorf("moved acquisitionIndex %d is not even", result.AcquisitionIndex)
			}

			base := addItemTestSlotBase(t, platform, moveStorageTestSlot) + addItemTestAnchorAt
			storageNextEquipAt := base + addItemTestStorageAt +
				addItemTestStorageKeyAt + 0x80*addItemTestRecordSize
			storageNextAcqAt := storageNextEquipAt + 4

			if equip := binary.LittleEndian.Uint32(removeTestBytes(
				t, engine, sessionID, storageNextEquipAt, 4)); equip != 134 {
				t.Errorf("Storage NextEquipIndex = %d, want 134", equip)
			}
			if acq := binary.LittleEndian.Uint32(removeTestBytes(
				t, engine, sessionID, storageNextAcqAt, 4)); acq != 8 {
				t.Errorf("Storage NextAcquisitionSortId = %d, want 8", acq)
			}
		})
	}
}

func TestMoveOwnedItemToStorageRejectsReferencesAndInvalidPlans(t *testing.T) {
	t.Run("active reference", func(t *testing.T) {
		engine, sessionID, ownedItemID := moveStorageTestTarget(
			t, PlatformPC, moveStorageTestFixture())
		base := addItemTestSlotBase(t, PlatformPC, moveStorageTestSlot) + addItemTestAnchorAt
		removeTestPut(t, engine, sessionID,
			base+removeTestEquipHandlesAt, addItemTestGoodsHandle)
		removeTestPut(t, engine, sessionID,
			base+removeTestEquipIndexesAt, removeTestEquipIndexBase+2)
		before := addItemTestSlotData(t, engine, sessionID, PlatformPC, moveStorageTestSlot)

		_, err := engine.MoveOwnedItemToStorage(
			sessionID, moveStorageTestSlot, ownedItemID, 0, "0", addItemTestGoodsID, 99)
		if err == nil || !strings.Contains(err.Error(), "unequip it first") {
			t.Fatalf("error = %v, want active-reference rejection", err)
		}
		if after := addItemTestSlotData(t, engine, sessionID, PlatformPC, moveStorageTestSlot); !bytes.Equal(after, before) {
			t.Error("a rejected referenced move changed the slot")
		}
	})

	for name, testCase := range map[string]struct {
		targetPosition int
		maxStorage     uint32
		want           string
	}{
		"position": {3, 99, "outside the range"},
		"limit":    {0, 2, "above its storage limit"},
	} {
		t.Run(name, func(t *testing.T) {
			engine, sessionID, ownedItemID := moveStorageTestTarget(
				t, PlatformPC, moveStorageTestFixture())
			before := addItemTestSlotData(t, engine, sessionID, PlatformPC, moveStorageTestSlot)
			_, err := engine.MoveOwnedItemToStorage(
				sessionID, moveStorageTestSlot, ownedItemID, testCase.targetPosition, "0",
				addItemTestGoodsID, testCase.maxStorage)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
			if after := addItemTestSlotData(t, engine, sessionID, PlatformPC, moveStorageTestSlot); !bytes.Equal(after, before) {
				t.Error("a rejected move changed the slot")
			}
		})
	}
}

func TestMoveOwnedItemToStorageRejectsInventoryKey(t *testing.T) {
	content := moveStorageTestFixture()
	content.common = nil
	content.commonCount = 0
	content.key = []addItemTestRow{{
		index: 2, handle: addItemTestGoodsHandle, rawQuantity: 1, acquisition: 17,
	}}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, addItemTestFixture{
		platform: PlatformPC,
		slot:     moveStorageTestSlot,
		key:      content.key,
	}), string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, moveStorageTestSlot, InventorySectionKey, 0, 0)
	if err != nil || len(inventory.Records) != 1 {
		t.Fatalf("GetInventory key: %v, records=%d", err, len(inventory.Records))
	}
	_, err = engine.MoveOwnedItemToStorage(
		loaded.SaveSessionID, moveStorageTestSlot, inventory.Records[0].OwnedItemID,
		0, "0", addItemTestGoodsID, 99)
	if err == nil || !strings.Contains(err.Error(), "key record") {
		t.Fatalf("error = %v, want key-record rejection", err)
	}
}
