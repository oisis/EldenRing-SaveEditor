package saveengine

import (
	"encoding/binary"
	"strings"
	"testing"
)

func setAddStorageTestCounters(
	t *testing.T,
	engine *Engine,
	sessionID string,
	platform Platform,
	slot int,
	nextEquip uint32,
	nextAcquisition uint32,
) {
	t.Helper()
	at := addItemTestSlotBase(t, platform, slot) + addItemTestAnchorAt +
		addItemTestStorageAt + addItemTestStorageSize - 8
	raw := make([]byte, 8)
	binary.LittleEndian.PutUint32(raw, nextEquip)
	binary.LittleEndian.PutUint32(raw[4:], nextAcquisition)
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if err := engine.sessions[sessionID].snapshot.writeAt(at, raw); err != nil {
		t.Fatalf("write Storage counters: %v", err)
	}
}

func TestAddItemToStorageInitialisesEmptyStorageOnPCAndPS4(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := addItemTestFixture{
				platform:   platform,
				slot:       2,
				tailMarker: true,
			}
			engine := New()
			loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			setAddStorageTestCounters(
				t, engine, loaded.SaveSessionID, platform, content.slot, 0, 1)
			before := addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, content.slot)

			result, err := engine.AddItemToStorage(
				loaded.SaveSessionID, content.slot, addItemTestGoodsID, 3, "0", false, 600)
			if err != nil {
				t.Fatalf("AddItemToStorage: %v", err)
			}
			if !result.CreatedRecord || result.PhysicalIndex != 0 || result.Quantity != 3 ||
				result.SaveRevision != "1" || result.ContainerSection != StorageSectionCommon {
				t.Errorf("AddItemToStorage = %+v", result)
			}

			after := addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, content.slot)
			storageRow := int64(addItemTestStorageAt + addItemTestStorageCommonAt)
			storageCounters := int64(addItemTestStorageAt + addItemTestStorageSize - 8)
			addItemTestAssertChanged(t, before, after, [][2]int64{
				{addItemTestStorageAt, addItemTestStorageAt + 4},
				{storageRow, storageRow + addItemTestRecordSize},
				{storageCounters, storageCounters + 8},
				{addItemTestGaItemDataAt, addItemTestGaItemDataAt + 4},
				{addItemTestGaItemDataArrayAt, addItemTestGaItemDataArrayAt + addItemTestGaItemEntrySize},
			})
			if got := addItemTestUint32(after, storageRow); got != addItemTestGoodsHandle {
				t.Errorf("handle = 0x%08X, want 0x%08X", got, addItemTestGoodsHandle)
			}
			if got := addItemTestUint32(after, storageRow+8); got != 2 {
				t.Errorf("acquisition index = %d, want 2", got)
			}
			if got := addItemTestUint32(after, storageCounters); got != 128 {
				t.Errorf("NextEquipIndex = %d, want 128", got)
			}
			if got := addItemTestUint32(after, storageCounters+4); got != 2 {
				t.Errorf("NextAcquisitionSortId = %d, want 2", got)
			}
		})
	}
}

func TestAddItemToStorageUsesPopulatedStorageAllocator(t *testing.T) {
	content := addItemTestFixture{
		platform: PlatformPC,
		slot:     2,
		common: []addItemTestRow{{
			index: 0, handle: addItemTestGoodsHandle, rawQuantity: 1, acquisition: 435,
		}},
		storage: []addItemTestRow{{
			index: 3, handle: addItemTestOtherHandle, rawQuantity: 2, acquisition: 5,
		}},
		commonCount: 1, storageCount: 1,
		gaItemData: []uint32{addItemTestGoodsID, addItemTestOtherID},
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	setAddStorageTestCounters(t, engine, loaded.SaveSessionID, PlatformPC, content.slot, 7, 4)
	result, err := engine.AddItemToStorage(
		loaded.SaveSessionID, content.slot, addItemTestGoodsID, 4, "0", false, 600)
	if err != nil {
		t.Fatalf("AddItemToStorage: %v", err)
	}
	if !result.CreatedRecord || result.PhysicalIndex != 0 {
		t.Errorf("AddItemToStorage = %+v, want a new row 0", result)
	}
	after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)
	row := int64(addItemTestStorageAt + addItemTestStorageCommonAt)
	counters := int64(addItemTestStorageAt + addItemTestStorageSize - 8)
	if got := addItemTestUint32(after, row+8); got != 6 {
		t.Errorf("acquisition index = %d, want 6", got)
	}
	if got := addItemTestUint32(after, counters); got != 7 {
		t.Errorf("NextEquipIndex = %d, want preserved 7", got)
	}
	if got := addItemTestUint32(after, counters+4); got != 7 {
		t.Errorf("NextAcquisitionSortId = %d, want 7", got)
	}
	if got := addItemTestUint32(after, addItemTestGaItemDataAt); got != 2 {
		t.Errorf("GaItemData count = %d, want unchanged 2", got)
	}
	if got := addItemTestUint32(after, int64(addItemTestStorageAt+addItemTestStorageCommonAt)+
		3*addItemTestRecordSize); got != addItemTestOtherHandle {
		t.Errorf("existing Storage row changed handle to 0x%08X", got)
	}
}

func TestAddItemToStorageTopUpChangesOnlyQuantity(t *testing.T) {
	content := addItemTestFixture{
		platform: PlatformPC,
		slot:     2,
		storage: []addItemTestRow{{
			index: 4, handle: addItemTestGoodsHandle,
			rawQuantity: ownedItemQuantityFlag | 5, acquisition: 12,
		}},
		storageCount: 1,
		gaItemData:   []uint32{addItemTestGoodsID},
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	setAddStorageTestCounters(t, engine, loaded.SaveSessionID, PlatformPC, content.slot, 9, 14)
	before := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)

	result, err := engine.AddItemToStorage(
		loaded.SaveSessionID, content.slot, addItemTestGoodsID, 3, "0", false, 600)
	if err != nil {
		t.Fatalf("AddItemToStorage: %v", err)
	}
	if result.CreatedRecord || result.PhysicalIndex != 4 || result.Quantity != 8 {
		t.Errorf("AddItemToStorage = %+v, want row 4 topped up to 8", result)
	}
	after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)
	quantityAt := int64(addItemTestStorageAt+addItemTestStorageCommonAt) +
		4*addItemTestRecordSize + 4
	addItemTestAssertChanged(t, before, after, [][2]int64{{quantityAt, quantityAt + 4}})
	if got := addItemTestUint32(after, quantityAt); got != ownedItemQuantityFlag|8 {
		t.Errorf("raw quantity = 0x%08X, want preserved flag and 8", got)
	}
}

func TestAddItemToStorageRejectsBeforeMutation(t *testing.T) {
	base := addItemTestFixture{
		platform: PlatformPC, slot: 2,
		storage: []addItemTestRow{{
			index: 0, handle: addItemTestOtherHandle, rawQuantity: 2, acquisition: 5,
		}},
		storageCount: 1, gaItemData: []uint32{addItemTestOtherID},
	}
	key := base
	key.storageKey = []addItemTestRow{{
		index: 0, handle: addItemTestGoodsHandle, rawQuantity: 1, acquisition: 3,
	}}
	key.storageKeyCount = 1

	for _, testCase := range []struct {
		name     string
		content  addItemTestFixture
		quantity uint32
		revision string
		limit    uint32
		nextAcq  uint32
		want     string
	}{
		{name: "Storage key record", content: key, quantity: 1, revision: "0", limit: 600, nextAcq: 4,
			want: "Storage key record"},
		{name: "container limit", content: base, quantity: 601, revision: "0", limit: 600, nextAcq: 4,
			want: "exceeds the limit"},
		{name: "stale revision", content: base, quantity: 1, revision: "1", limit: 600, nextAcq: 4,
			want: "does not match"},
		{name: "allocator exhausted", content: base, quantity: 1, revision: "0", limit: 600,
			nextAcq: ^uint32(0),
			want:    "cannot be advanced"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeAddItemFixture(t, testCase.content), string(PlatformPC))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			setAddStorageTestCounters(
				t, engine, loaded.SaveSessionID, PlatformPC, testCase.content.slot, 7, testCase.nextAcq)
			before := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, testCase.content.slot)
			_, err = engine.AddItemToStorage(
				loaded.SaveSessionID, testCase.content.slot, addItemTestGoodsID, testCase.quantity,
				testCase.revision, false, testCase.limit)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("AddItemToStorage error = %v, want %q", err, testCase.want)
			}
			after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, testCase.content.slot)
			addItemTestAssertChanged(t, before, after, nil)
			if revision, dirty := addItemTestSessionState(
				t, engine, loaded.SaveSessionID); revision != "0" || dirty {
				t.Errorf("rejected add left revision %q, dirty %v", revision, dirty)
			}
		})
	}
}
