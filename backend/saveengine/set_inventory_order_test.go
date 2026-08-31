package saveengine

import (
	"bytes"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const setInventoryOrderTestSlot = 2

func setInventoryOrderFixture(platform Platform) addItemTestFixture {
	return addItemTestFixture{
		platform: platform,
		slot:     setInventoryOrderTestSlot,
		common: []addItemTestRow{
			{index: 1, handle: addItemTestGoodsHandle, rawQuantity: 0x80000003, acquisition: 440},
			{index: 4, handle: addItemTestTalismanHandle, rawQuantity: 1, acquisition: 460},
			{index: 7, handle: addItemTestOtherHandle, rawQuantity: 2, acquisition: 480},
		},
		key: []addItemTestRow{
			{index: 1, handle: addItemTestGoodsHandle, rawQuantity: 1, acquisition: 500},
		},
		storage: []addItemTestRow{
			{index: 3, handle: addItemTestOtherHandle, rawQuantity: 1, acquisition: 41},
		},
		commonCount:     3,
		storageCount:    1,
		nextEquipIndex:  777,
		nextAcquisition: 500,
	}
}

func setInventoryOrderTarget(
	t *testing.T,
	platform Platform,
) (*Engine, string, []InventoryRecord, string) {
	t.Helper()
	engine := New()
	loaded, err := engine.LoadSave(
		writeAddItemFixture(t, setInventoryOrderFixture(platform)), string(platform), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, setInventoryOrderTestSlot, InventorySectionCommon, 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	storage, err := engine.GetStorage(
		loaded.SaveSessionID, setInventoryOrderTestSlot, StorageSectionCommon, 0, 0)
	if err != nil || len(storage.Records) != 1 {
		t.Fatalf("GetStorage: %v, records=%d", err, len(storage.Records))
	}
	return engine, loaded.SaveSessionID, inventory.Records, storage.Records[0].OwnedItemID
}

func TestSetInventoryOrderWritesOnlyIndicesAndReloadsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine, sessionID, records, _ := setInventoryOrderTarget(t, platform)
			orderedIDs := []string{
				records[2].OwnedItemID, records[1].OwnedItemID, records[0].OwnedItemID,
			}
			before := addItemTestSlotData(
				t, engine, sessionID, platform, setInventoryOrderTestSlot)

			result, err := engine.SetInventoryOrder(
				sessionID, setInventoryOrderTestSlot, orderedIDs, "0",
				func(uint32) (bool, error) { return true, nil })
			if err != nil {
				t.Fatalf("SetInventoryOrder: %v", err)
			}
			wantIDs := []uint32{addItemTestOtherID, addItemTestTalismanID, addItemTestGoodsID}
			wantIndices := []uint32{502, 504, 506}
			if result.SaveRevision != "1" || !reflect.DeepEqual(result.GameIDs, wantIDs) ||
				!reflect.DeepEqual(result.AcquisitionIndices, wantIndices) {
				t.Errorf("result = %+v", result)
			}

			after := addItemTestSlotData(
				t, engine, sessionID, platform, setInventoryOrderTestSlot)
			expected := make([][2]int64, 0, 4)
			for _, physicalIndex := range []int{1, 4, 7} {
				at := int64(addItemTestCommonAt + physicalIndex*addItemTestRecordSize + 8)
				expected = append(expected, [2]int64{at, at + 4})
			}
			expected = append(expected,
				[2]int64{addItemTestNextAcqAt, addItemTestNextAcqAt + 4})
			addItemTestAssertChanged(t, before, after, expected)
			if got := addItemTestUint32(after, addItemTestNextEquipAt); got != 777 {
				t.Errorf("NextEquipIndex = %d, want unchanged 777", got)
			}
			if got := addItemTestUint32(after, addItemTestNextAcqAt); got != 507 {
				t.Errorf("NextAcquisitionSortId = %d, want 507", got)
			}
			if got := addItemTestUint32(after,
				addItemTestKeyAt+addItemTestRecordSize+8); got != 500 {
				t.Errorf("Inventory key acquisition index = %d, want unchanged 500", got)
			}

			target := filepath.Join(t.TempDir(), "inventory-order.sl2")
			if _, err := engine.WriteSave(sessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			fresh := New()
			reloaded, err := fresh.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave after WriteSave: %v", err)
			}
			inventory, err := fresh.GetInventory(
				reloaded.SaveSessionID, setInventoryOrderTestSlot, InventorySectionCommon, 0, 0)
			if err != nil {
				t.Fatalf("GetInventory after reload: %v", err)
			}
			sort.Slice(inventory.Records, func(left, right int) bool {
				return inventory.Records[left].AcquisitionIndex < inventory.Records[right].AcquisitionIndex
			})
			wantHandles := []uint32{
				addItemTestOtherHandle, addItemTestTalismanHandle, addItemTestGoodsHandle,
			}
			for index, record := range inventory.Records {
				if record.GaItemHandle != wantHandles[index] {
					t.Errorf("reloaded record %d handle = 0x%08X, want 0x%08X",
						index, record.GaItemHandle, wantHandles[index])
				}
			}
		})
	}
}

func TestSetInventoryOrderRejectsInvalidPermutationsWithoutMutation(t *testing.T) {
	for name, change := range map[string]func([]string, string) []string{
		"incomplete": func(ids []string, _ string) []string { return ids[:2] },
		"duplicate": func(ids []string, _ string) []string {
			return []string{ids[0], ids[0], ids[2]}
		},
		"storage token": func(ids []string, storageID string) []string {
			return []string{storageID, ids[1], ids[2]}
		},
	} {
		t.Run(name, func(t *testing.T) {
			engine, sessionID, records, storageID := setInventoryOrderTarget(t, PlatformPC)
			ids := []string{
				records[2].OwnedItemID, records[1].OwnedItemID, records[0].OwnedItemID,
			}
			before := addItemTestSlotData(
				t, engine, sessionID, PlatformPC, setInventoryOrderTestSlot)
			_, err := engine.SetInventoryOrder(
				sessionID, setInventoryOrderTestSlot, change(ids, storageID), "0",
				func(uint32) (bool, error) { return true, nil })
			if err == nil {
				t.Fatal("invalid Inventory order was accepted")
			}
			after := addItemTestSlotData(
				t, engine, sessionID, PlatformPC, setInventoryOrderTestSlot)
			if !bytes.Equal(after, before) {
				t.Error("a rejected Inventory order changed the slot")
			}
			if revision, dirty := addItemTestSessionState(
				t, engine, sessionID); revision != "0" || dirty {
				t.Errorf("rejected order left revision %q, dirty %v", revision, dirty)
			}
		})
	}
}

func TestPlanItemOrderIndicesRejectsTheUnsafeBoundary(t *testing.T) {
	indices, err := planItemOrderIndices(500, 1, map[uint32]struct{}{350: {}})
	if err != nil || !reflect.DeepEqual(indices, []uint32{702}) {
		t.Fatalf("indices = %v, error = %v, want [702]", indices, err)
	}

	_, err = planItemOrderIndices(9998, 2, nil)
	if err == nil || !strings.Contains(err.Error(), "10000") {
		t.Fatalf("error = %v, want unsafe-index rejection", err)
	}
}
