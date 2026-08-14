package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

const (
	setStorageOrderTestSlot       = 2
	setStorageOrderTestKeyRecords = 0x80
)

func setStorageOrderFixture(platform Platform) addItemTestFixture {
	return addItemTestFixture{
		platform: platform,
		slot:     setStorageOrderTestSlot,
		common: []addItemTestRow{
			{index: 1, handle: addItemTestGoodsHandle, rawQuantity: 3, acquisition: 39},
		},
		storage: []addItemTestRow{
			{index: 1, handle: addItemTestGoodsHandle, rawQuantity: 3, acquisition: 440},
			{index: 4, handle: addItemTestTalismanHandle, rawQuantity: 1, acquisition: 460},
			{index: 7, handle: addItemTestOtherHandle, rawQuantity: 2, acquisition: 480},
		},
		storageKey: []addItemTestRow{
			{index: 1, handle: addItemTestGoodsHandle, rawQuantity: 1, acquisition: 500},
		},
		commonCount:     1,
		storageCount:    3,
		storageKeyCount: 1,
	}
}

func setStorageOrderFixtureCounters(
	t *testing.T, path string, platform Platform, slot int,
) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	base := addItemTestSlotBase(t, platform, slot) + addItemTestAnchorAt +
		addItemTestStorageAt + addItemTestStorageKeyAt +
		setStorageOrderTestKeyRecords*addItemTestRecordSize
	binary.LittleEndian.PutUint32(data[base:], 777)
	binary.LittleEndian.PutUint32(data[base+4:], 500)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func setStorageOrderTarget(
	t *testing.T,
	platform Platform,
) (*Engine, string, []StorageRecord, string) {
	t.Helper()
	engine := New()
	path := writeAddItemFixture(t, setStorageOrderFixture(platform))
	setStorageOrderFixtureCounters(t, path, platform, setStorageOrderTestSlot)
	loaded, err := engine.LoadSave(path, string(platform))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	storage, err := engine.GetStorage(
		loaded.SaveSessionID, setStorageOrderTestSlot, StorageSectionCommon, 0, 0)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, setStorageOrderTestSlot, InventorySectionCommon, 0, 0)
	if err != nil || len(inventory.Records) != 1 {
		t.Fatalf("GetInventory: %v, records=%d", err, len(inventory.Records))
	}
	return engine, loaded.SaveSessionID, storage.Records, inventory.Records[0].OwnedItemID
}

func TestSetStorageOrderWritesOnlyIndicesAndReloadsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine, sessionID, records, _ := setStorageOrderTarget(t, platform)
			orderedIDs := []string{
				records[2].OwnedItemID, records[1].OwnedItemID, records[0].OwnedItemID,
			}
			before := addItemTestSlotData(
				t, engine, sessionID, platform, setStorageOrderTestSlot)

			result, err := engine.SetStorageOrder(
				sessionID, setStorageOrderTestSlot, orderedIDs, "0",
				func(uint32) (bool, error) { return true, nil })
			if err != nil {
				t.Fatalf("SetStorageOrder: %v", err)
			}
			wantIDs := []uint32{addItemTestOtherID, addItemTestTalismanID, addItemTestGoodsID}
			wantIndices := []uint32{502, 504, 506}
			if result.SaveRevision != "1" || !reflect.DeepEqual(result.GameIDs, wantIDs) ||
				!reflect.DeepEqual(result.AcquisitionIndices, wantIndices) {
				t.Errorf("result = %+v", result)
			}

			after := addItemTestSlotData(
				t, engine, sessionID, platform, setStorageOrderTestSlot)
			expected := make([][2]int64, 0, 4)
			for _, physicalIndex := range []int{1, 4, 7} {
				at := int64(addItemTestStorageAt + addItemTestStorageCommonAt +
					physicalIndex*addItemTestRecordSize + 8)
				expected = append(expected, [2]int64{at, at + 4})
			}
			storageNextEquipAt := int64(addItemTestStorageAt + addItemTestStorageKeyAt +
				setStorageOrderTestKeyRecords*addItemTestRecordSize)
			storageNextAcqAt := storageNextEquipAt + 4
			expected = append(expected, [2]int64{storageNextAcqAt, storageNextAcqAt + 4})
			addItemTestAssertChanged(t, before, after, expected)
			if got := addItemTestUint32(after, storageNextEquipAt); got != 777 {
				t.Errorf("Storage NextEquipIndex = %d, want unchanged 777", got)
			}
			if got := addItemTestUint32(after, storageNextAcqAt); got != 507 {
				t.Errorf("Storage NextAcquisitionSortId = %d, want 507", got)
			}
			if got := addItemTestUint32(after,
				addItemTestStorageAt+addItemTestStorageKeyAt+addItemTestRecordSize+8); got != 500 {
				t.Errorf("Storage key acquisition index = %d, want unchanged 500", got)
			}

			target := filepath.Join(t.TempDir(), "storage-order.sl2")
			if _, err := engine.WriteSave(sessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			fresh := New()
			reloaded, err := fresh.LoadSave(target, string(platform))
			if err != nil {
				t.Fatalf("LoadSave after WriteSave: %v", err)
			}
			storage, err := fresh.GetStorage(
				reloaded.SaveSessionID, setStorageOrderTestSlot, StorageSectionCommon, 0, 0)
			if err != nil {
				t.Fatalf("GetStorage after reload: %v", err)
			}
			sort.Slice(storage.Records, func(left, right int) bool {
				return storage.Records[left].AcquisitionIndex < storage.Records[right].AcquisitionIndex
			})
			wantHandles := []uint32{
				addItemTestOtherHandle, addItemTestTalismanHandle, addItemTestGoodsHandle,
			}
			for index, record := range storage.Records {
				if record.GaItemHandle != wantHandles[index] {
					t.Errorf("reloaded record %d handle = 0x%08X, want 0x%08X",
						index, record.GaItemHandle, wantHandles[index])
				}
			}
		})
	}
}

func TestSetStorageOrderRejectsInvalidPermutationsWithoutMutation(t *testing.T) {
	for name, change := range map[string]func([]string, string) []string{
		"incomplete": func(ids []string, _ string) []string { return ids[:2] },
		"duplicate": func(ids []string, _ string) []string {
			return []string{ids[0], ids[0], ids[2]}
		},
		"inventory token": func(ids []string, inventoryID string) []string {
			return []string{inventoryID, ids[1], ids[2]}
		},
	} {
		t.Run(name, func(t *testing.T) {
			engine, sessionID, records, inventoryID := setStorageOrderTarget(t, PlatformPC)
			ids := []string{
				records[2].OwnedItemID, records[1].OwnedItemID, records[0].OwnedItemID,
			}
			before := addItemTestSlotData(
				t, engine, sessionID, PlatformPC, setStorageOrderTestSlot)
			_, err := engine.SetStorageOrder(
				sessionID, setStorageOrderTestSlot, change(ids, inventoryID), "0",
				func(uint32) (bool, error) { return true, nil })
			if err == nil {
				t.Fatal("invalid Storage order was accepted")
			}
			after := addItemTestSlotData(
				t, engine, sessionID, PlatformPC, setStorageOrderTestSlot)
			if !bytes.Equal(after, before) {
				t.Error("a rejected Storage order changed the slot")
			}
			if revision, dirty := addItemTestSessionState(
				t, engine, sessionID); revision != "0" || dirty {
				t.Errorf("rejected order left revision %q, dirty %v", revision, dirty)
			}
		})
	}
}
