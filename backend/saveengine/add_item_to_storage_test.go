package saveengine

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
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
			loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(platform), "local")
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
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC), "local")
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
	if got := addItemTestUint32(after, row+8); got != 8 {
		t.Errorf("acquisition index = %d, want 8", got)
	}
	// The section keeps a record on physical row 3 and the new one lands in the
	// hole on row 0, so the highest occupied row stays 3: 128 + 3.
	if got := addItemTestUint32(after, counters); got != 131 {
		t.Errorf("NextEquipIndex = %d, want 131", got)
	}
	if got := addItemTestUint32(after, counters+4); got != 5 {
		t.Errorf("NextAcquisitionSortId = %d, want 5", got)
	}
	if got := addItemTestUint32(after, addItemTestGaItemDataAt); got != 2 {
		t.Errorf("GaItemData count = %d, want unchanged 2", got)
	}
	if got := addItemTestUint32(after, int64(addItemTestStorageAt+addItemTestStorageCommonAt)+
		3*addItemTestRecordSize); got != addItemTestOtherHandle {
		t.Errorf("existing Storage row changed handle to 0x%08X", got)
	}
}

func TestAddItemToStorageT330ShapeSingleAndAccumulationOnPCAndPS4(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			// T330 shape: 6 common records, max Index 12 (indices 2, 4, 6, 8, 10, 12),
			// NextEquipIndex 133, NextAcquisitionSortId 7.
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
				platform:     platform,
				slot:         2,
				storage:      rows,
				storageCount: 6,
			}

			engine := New()
			loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			setAddStorageTestCounters(t, engine, loaded.SaveSessionID, platform, content.slot, 133, 7)

			// Single add: expect assigned index 14, NextAcquisitionSortId 8, NextEquipIndex 134.
			res, err := engine.AddItemToStorage(
				loaded.SaveSessionID, content.slot, addItemTestGoodsID, 1, "0", false, 600)
			if err != nil {
				t.Fatalf("AddItemToStorage single: %v", err)
			}
			if !res.CreatedRecord || res.PhysicalIndex != 6 {
				t.Errorf("res = %+v, want new row at physical index 6", res)
			}

			after := addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, content.slot)
			counters := int64(addItemTestStorageAt + addItemTestStorageSize - 8)
			row6 := int64(addItemTestStorageAt + addItemTestStorageCommonAt + 6*addItemTestRecordSize)

			idx := addItemTestUint32(after, row6+8)
			if idx != 14 {
				t.Errorf("single add acquisition index = %d, want 14", idx)
			}
			if idx%2 != 0 {
				t.Errorf("single add acquisition index %d is not even", idx)
			}
			if equip := addItemTestUint32(after, counters); equip != 134 {
				t.Errorf("NextEquipIndex = %d, want 134", equip)
			}
			if acq := addItemTestUint32(after, counters+4); acq != 8 {
				t.Errorf("NextAcquisitionSortId = %d, want 8", acq)
			}

			// Accumulation: add 4 more items (5 additions total)
			// Expect assigned indices 16, 18, 20, 22, NextAcquisitionSortId 12, NextEquipIndex 138.
			wantIndices := []uint32{16, 18, 20, 22}
			for step, wantIdx := range wantIndices {
				rev := string(rune('1' + step))
				res, err := engine.AddItemToStorage(
					loaded.SaveSessionID, content.slot, addItemTestOtherID, 1, rev, true, 600)
				if err != nil {
					t.Fatalf("AddItemToStorage step %d (rev %s): %v", step+1, rev, err)
				}
				if !res.CreatedRecord || res.PhysicalIndex != 7+step {
					t.Errorf("step %d res = %+v, want physicalIndex %d", step+1, res, 7+step)
				}
				stepData := addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, content.slot)
				rowAt := int64(addItemTestStorageAt + addItemTestStorageCommonAt + int64(7+step)*addItemTestRecordSize)
				stepIdx := addItemTestUint32(stepData, rowAt+8)
				if stepIdx != wantIdx {
					t.Errorf("step %d acquisition index = %d, want %d", step+1, stepIdx, wantIdx)
				}
				if stepIdx%2 != 0 {
					t.Errorf("step %d acquisition index %d is not even", step+1, stepIdx)
				}
			}

			finalData := addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, content.slot)
			if finalEquip := addItemTestUint32(finalData, counters); finalEquip != 138 {
				t.Errorf("final NextEquipIndex = %d, want 138", finalEquip)
			}
			if finalAcq := addItemTestUint32(finalData, counters+4); finalAcq != 12 {
				t.Errorf("final NextAcquisitionSortId = %d, want 12", finalAcq)
			}
		})
	}
}

func TestAddItemToStorageTopUpRefreshesAcquisitionIndex(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := addItemTestFixture{
				platform: platform,
				slot:     2,
				storage: []addItemTestRow{{
					index: 4, handle: addItemTestGoodsHandle,
					rawQuantity: ownedItemQuantityFlag | 5, acquisition: 12,
				}},
				storageCount: 1,
				gaItemData:   []uint32{addItemTestGoodsID},
			}
			engine := New()
			loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			setAddStorageTestCounters(t, engine, loaded.SaveSessionID, platform, content.slot, 9, 14)
			before := addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, content.slot)

			result, err := engine.AddItemToStorage(
				loaded.SaveSessionID, content.slot, addItemTestGoodsID, 3, "0", false, 600)
			if err != nil {
				t.Fatalf("AddItemToStorage: %v", err)
			}
			if result.CreatedRecord || result.PhysicalIndex != 4 || result.Quantity != 8 ||
				result.SaveRevision != "1" || result.ContainerSection != StorageSectionCommon {
				t.Errorf("AddItemToStorage = %+v, want row 4 topped up to 8", result)
			}

			after := addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, content.slot)
			storageRow := int64(addItemTestStorageAt+addItemTestStorageCommonAt) + 4*addItemTestRecordSize
			quantityAt := storageRow + 4
			acquisitionAt := storageRow + 8
			countersAt := int64(addItemTestStorageAt + addItemTestStorageSize - 8)
			nextAcqAt := countersAt + 4

			// Top-up writes the new quantity, the refreshed AcquisitionIndex and the
			// advanced NextAcquisitionSortId; NextEquipIndex, count, GaItemData and handle
			// stay unchanged.
			addItemTestAssertChanged(t, before, after, [][2]int64{
				{quantityAt, quantityAt + 4},
				{acquisitionAt, acquisitionAt + 4},
				{nextAcqAt, nextAcqAt + 4},
			})

			if got := addItemTestUint32(after, storageRow); got != addItemTestGoodsHandle {
				t.Errorf("handle = 0x%08X, want 0x%08X", got, addItemTestGoodsHandle)
			}
			if got := addItemTestUint32(after, quantityAt); got != ownedItemQuantityFlag|8 {
				t.Errorf("raw quantity = 0x%08X, want preserved flag and 8", got)
			}
			// Stored next was 14, max bucket was 12/2 = 6. Effective bucket = max(14, 7, 1) = 14.
			// Assigned index = 14 * 2 = 28. NextAcquisitionSortId = 14 + 1 = 15.
			if got := addItemTestUint32(after, acquisitionAt); got != 28 {
				t.Errorf("acquisition index = %d, want 28", got)
			}
			if got := addItemTestUint32(after, countersAt); got != 9 {
				t.Errorf("NextEquipIndex = %d, want unchanged 9", got)
			}
			if got := addItemTestUint32(after, nextAcqAt); got != 15 {
				t.Errorf("NextAcquisitionSortId = %d, want 15", got)
			}
			if got := addItemTestUint32(after, addItemTestStorageAt); got != 1 {
				t.Errorf("Storage count = %d, want unchanged 1", got)
			}
			if got := addItemTestUint32(after, addItemTestGaItemDataAt); got != 1 {
				t.Errorf("GaItemData count = %d, want unchanged 1", got)
			}

			assertNoStorageBucketCollision(t, engine, loaded.SaveSessionID, content.slot)

			// Persist through WriteSave and verify identical state on fresh reload.
			target := filepath.Join(t.TempDir(), "topped_up.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave after WriteSave: %v", err)
			}
			reloadedStorage, err := reloadedEngine.GetStorage(
				reloaded.SaveSessionID, content.slot, StorageSectionCommon, 0, 0)
			if err != nil {
				t.Fatalf("GetStorage after reload: %v", err)
			}
			if len(reloadedStorage.Records) != 1 {
				t.Fatalf("reloaded Storage record count = %d, want 1", len(reloadedStorage.Records))
			}
			rec := reloadedStorage.Records[0]
			if rec.PhysicalIndex != 4 || rec.Quantity != 8 || rec.AcquisitionIndex != 28 ||
				rec.GaItemHandle != addItemTestGoodsHandle {
				t.Errorf("reloaded Storage record = %+v, want row 4 qty 8 acq 28 handle 0x%08X",
					rec, addItemTestGoodsHandle)
			}
			assertNoStorageBucketCollision(t, reloadedEngine, reloaded.SaveSessionID, content.slot)
		})
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
	topUpFixture := addItemTestFixture{
		platform: PlatformPC, slot: 2,
		storage: []addItemTestRow{{
			index: 0, handle: addItemTestGoodsHandle, rawQuantity: 580, acquisition: 5,
		}},
		storageCount: 1, gaItemData: []uint32{addItemTestGoodsID},
	}
	key := base
	key.storageKey = []addItemTestRow{{
		index: 0, handle: addItemTestGoodsHandle, rawQuantity: 1, acquisition: 3,
	}}
	key.storageKeyCount = 1

	for _, testCase := range []struct {
		name      string
		content   addItemTestFixture
		quantity  uint32
		revision  string
		limit     uint32
		nextEquip uint32
		nextAcq   uint32
		want      string
	}{
		{name: "Storage key record", content: key, quantity: 1, revision: "0", limit: 600, nextEquip: 7, nextAcq: 4,
			want: "Storage key record"},
		{name: "container limit", content: base, quantity: 601, revision: "0", limit: 600, nextEquip: 7, nextAcq: 4,
			want: "exceeds the limit"},
		{name: "top-up record limit", content: topUpFixture, quantity: 25, revision: "0", limit: 600, nextEquip: 7, nextAcq: 4,
			want: "above the limit"},
		{name: "stale revision", content: base, quantity: 1, revision: "1", limit: 600, nextEquip: 7, nextAcq: 4,
			want: "does not match"},
		{name: "allocator acq overflow", content: base, quantity: 1, revision: "0", limit: 600,
			nextEquip: 7, nextAcq: ^uint32(0),
			want: "overflow uint32"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeAddItemFixture(t, testCase.content), string(PlatformPC), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			setAddStorageTestCounters(
				t, engine, loaded.SaveSessionID, PlatformPC, testCase.content.slot, testCase.nextEquip, testCase.nextAcq)
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

// sparseStorageRows builds the physical common Storage of the confirmed Test 3
// fixture: rows 0..346 occupied except for the hole on row 315, every bucket
// distinct. Its NextEquipIndex is 128 + 346 = 474.
func sparseStorageRows() []addItemTestRow {
	rows := make([]addItemTestRow, 0, 346)
	for index := 0; index <= 346; index++ {
		if index == sparseStorageHole {
			continue
		}
		rows = append(rows, addItemTestRow{
			index:       index,
			handle:      addItemTestFillHandleBase + uint32(index),
			rawQuantity: 1,
			acquisition: uint32((index + 1) * 2),
		})
	}
	return rows
}

const (
	sparseStorageHole    = 315
	sparseStorageHighest = 346
	sparseStorageEquip   = 128 + sparseStorageHighest
)

// assertNoStorageBucketCollision proves the hard criterion of TODO 19.4: no two
// common Storage records share an acquisition bucket after the mutation.
func assertNoStorageBucketCollision(t *testing.T, engine *Engine, sessionID string, slot int) {
	t.Helper()
	storage, err := engine.GetStorage(sessionID, slot, StorageSectionCommon, 1, 1<<16)
	if err != nil {
		t.Fatalf("GetStorage: %v", err)
	}
	seen := make(map[uint32]int, len(storage.Records))
	for _, record := range storage.Records {
		bucket := record.AcquisitionIndex >> 1
		if other, clash := seen[bucket]; clash {
			t.Errorf("rows %d and %d share acquisition bucket %d",
				other, record.PhysicalIndex, bucket)
		}
		seen[bucket] = record.PhysicalIndex
	}
}

// storageCountersAt reports the absolute offset of the two trailing Storage
// counters of the fixture slot, so an assertion can read the raw four bytes.
func storageCountersAt(t *testing.T, platform Platform, slot int) int64 {
	t.Helper()
	return addItemTestSlotBase(t, platform, slot) + addItemTestAnchorAt +
		addItemTestStorageAt + addItemTestStorageSize - 8
}

// TestAddItemToStorageDerivesNextEquipIndexFromPhysicalLayout is the regression
// test of TODO 19.1. The counter is 128 plus the highest occupied physical row,
// so a record filling a hole below that row leaves it unchanged and only a
// record extending the table raises it. A single case cannot tell the two models
// apart, so both sides of the boundary are asserted on the same sparse fixture.
func TestAddItemToStorageDerivesNextEquipIndexFromPhysicalLayout(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := addItemTestFixture{
				platform:     platform,
				slot:         2,
				storage:      sparseStorageRows(),
				storageCount: uint32(sparseStorageHighest),
			}
			engine := New()
			loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			setAddStorageTestCounters(t, engine, loaded.SaveSessionID, platform, content.slot,
				sparseStorageEquip, uint32(sparseStorageHighest+2))
			counters := int64(addItemTestStorageAt + addItemTestStorageSize - 8)

			// The hole on row 315 is the first free row, so the new record lands
			// below the highest occupied row and the counter must not move.
			hole, err := engine.AddItemToStorage(
				loaded.SaveSessionID, content.slot, addItemTestGoodsID, 1, "0", false, 600)
			if err != nil {
				t.Fatalf("AddItemToStorage into the hole: %v", err)
			}
			if !hole.CreatedRecord || hole.PhysicalIndex != sparseStorageHole {
				t.Errorf("hole add = %+v, want a new record on row %d", hole, sparseStorageHole)
			}
			after := addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, content.slot)
			if got := addItemTestUint32(after, counters); got != sparseStorageEquip {
				t.Errorf("NextEquipIndex after the hole add = %d, want unchanged %d",
					got, sparseStorageEquip)
			}
			if raw := removeTestBytes(t, engine, loaded.SaveSessionID,
				storageCountersAt(t, platform, content.slot), 4); !bytes.Equal(
				raw, littleEndianUint32(sparseStorageEquip)) {
				t.Errorf("raw NextEquipIndex bytes = % X, want % X",
					raw, littleEndianUint32(sparseStorageEquip))
			}
			assertNoStorageBucketCollision(t, engine, loaded.SaveSessionID, content.slot)

			// The table is dense up to 346 now, so the next record extends it to
			// 347 and the counter follows exactly one step.
			extend, err := engine.AddItemToStorage(
				loaded.SaveSessionID, content.slot, addItemTestOtherID, 1, "1", false, 600)
			if err != nil {
				t.Fatalf("AddItemToStorage past the last row: %v", err)
			}
			if !extend.CreatedRecord || extend.PhysicalIndex != sparseStorageHighest+1 {
				t.Errorf("extending add = %+v, want a new record on row %d",
					extend, sparseStorageHighest+1)
			}
			after = addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, content.slot)
			if got := addItemTestUint32(after, counters); got != sparseStorageEquip+1 {
				t.Errorf("NextEquipIndex after the extending add = %d, want %d",
					got, sparseStorageEquip+1)
			}
			if raw := removeTestBytes(t, engine, loaded.SaveSessionID,
				storageCountersAt(t, platform, content.slot), 4); !bytes.Equal(
				raw, littleEndianUint32(sparseStorageEquip+1)) {
				t.Errorf("raw NextEquipIndex bytes = % X, want % X",
					raw, littleEndianUint32(sparseStorageEquip+1))
			}
			assertNoStorageBucketCollision(t, engine, loaded.SaveSessionID, content.slot)
		})
	}
}

// TestAddItemToStorageRecomputesStaleNextEquipIndex covers the counter the game
// leaves behind on deletion: it is never carried over, so even a saturated
// stored value is replaced by the value the physical layout dictates.
func TestAddItemToStorageRecomputesStaleNextEquipIndex(t *testing.T) {
	content := addItemTestFixture{
		platform: PlatformPC, slot: 2,
		storage: []addItemTestRow{{
			index: 0, handle: addItemTestOtherHandle, rawQuantity: 2, acquisition: 5,
		}},
		storageCount: 1, gaItemData: []uint32{addItemTestOtherID},
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	setAddStorageTestCounters(
		t, engine, loaded.SaveSessionID, PlatformPC, content.slot, ^uint32(0), 4)

	result, err := engine.AddItemToStorage(
		loaded.SaveSessionID, content.slot, addItemTestGoodsID, 1, "0", false, 600)
	if err != nil {
		t.Fatalf("AddItemToStorage: %v", err)
	}
	if !result.CreatedRecord || result.PhysicalIndex != 1 {
		t.Errorf("AddItemToStorage = %+v, want a new record on row 1", result)
	}
	after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)
	counters := int64(addItemTestStorageAt + addItemTestStorageSize - 8)
	if got := addItemTestUint32(after, counters); got != 129 {
		t.Errorf("NextEquipIndex = %d, want 129 derived from row 1", got)
	}
	if raw := removeTestBytes(t, engine, loaded.SaveSessionID,
		storageCountersAt(t, PlatformPC, content.slot), 4); !bytes.Equal(
		raw, littleEndianUint32(129)) {
		t.Errorf("raw NextEquipIndex bytes = % X, want % X", raw, littleEndianUint32(129))
	}
	assertNoStorageBucketCollision(t, engine, loaded.SaveSessionID, content.slot)
}

func TestAddItemToStorageRejectsExistingDuplicateRecords(t *testing.T) {
	content := addItemTestFixture{
		platform: PlatformPC, slot: 2,
		storage: []addItemTestRow{
			{index: 0, handle: addItemTestGoodsHandle, rawQuantity: 5, acquisition: 2},
			{index: 1, handle: addItemTestGoodsHandle, rawQuantity: 5, acquisition: 4},
		},
		storageCount: 2, gaItemData: []uint32{addItemTestGoodsID},
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot)

	_, err = engine.AddItemToStorage(
		loaded.SaveSessionID, content.slot, addItemTestGoodsID, 1, "0", false, 600)
	if err == nil || !strings.Contains(err.Error(), "already holds 2 duplicate records in Storage") {
		t.Fatalf("error = %v, want duplicate quantity_stack rejection", err)
	}
	if after := addItemTestSlotData(t, engine, loaded.SaveSessionID, PlatformPC, content.slot); !bytes.Equal(after, before) {
		t.Error("a rejected duplicate add changed the slot")
	}
}
