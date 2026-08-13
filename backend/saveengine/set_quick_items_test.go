package saveengine

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	setQuickInventoryAt                 = 505
	setQuickInventoryRowSize            = 12
	setQuickSlotVersion                 = 82
	setQuickAnchorAt                    = 0x10020
	setQuickSectionAt                   = 0x9279
	setQuickActiveAt                    = 0x50
	setQuickPouchAt                     = 0x54
	setQuickProjectileCountAt           = 0x931D
	setQuickProjectileRecordSize        = 8
	setQuickArmamentsTailAt             = 0x58
	setQuickPouchTailAt                 = 0x80
	setQuickInventoryEquipBase          = 0x180
	setQuickTestGameID1          uint32 = 0x40000064
	setQuickTestGameID2          uint32 = 0x40000065
	setQuickTestAccessory        uint32 = 0x20000064
)

func writeSetQuickItemsFixture(
	t *testing.T,
	platform Platform,
	slot int,
	owned []struct {
		gameID   uint32
		quantity uint32
	},
) (string, string) {
	t.Helper()

	path := writeQuickItemsFixture(t, quickItemsFixture{
		platform:    platform,
		slot:        slot,
		flag:        1,
		anchorAt:    setQuickAnchorAt,
		activeQuick: 4,
	})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var slotBase int64
	if platform == PlatformPS4 {
		slotBase = 0x70 + int64(slot)*0x280000
	} else {
		slotBase = 0x310 + int64(slot)*0x280010
	}
	binary.LittleEndian.PutUint32(data[slotBase:], setQuickSlotVersion)

	pairAt := slotBase + setQuickAnchorAt + setQuickSectionAt
	for i := 0; i < 10; i++ {
		binary.LittleEndian.PutUint32(data[pairAt+int64(i)*8:], 0)
		binary.LittleEndian.PutUint32(data[pairAt+int64(i)*8+4:], 0xFFFFFFFF)
	}
	binary.LittleEndian.PutUint32(data[pairAt+setQuickActiveAt:], 4)

	countAt := slotBase + setQuickAnchorAt + setQuickProjectileCountAt
	binary.LittleEndian.PutUint32(data[countAt:], 17)
	armamentsAt := countAt + 4 + 17*setQuickProjectileRecordSize
	for i := 0; i < 10; i++ {
		binary.LittleEndian.PutUint32(
			data[armamentsAt+setQuickArmamentsTailAt+int64(i)*4:], 0xFFFFFFFF)
	}

	inventoryAt := slotBase + setQuickAnchorAt + setQuickInventoryAt
	binary.LittleEndian.PutUint32(data[inventoryAt-4:], uint32(len(owned)))
	for i, item := range owned {
		handle, err := gaItemHandleForGameID(item.gameID)
		if err != nil {
			t.Fatalf("gaItemHandleForGameID(0x%08X): %v", item.gameID, err)
		}
		rowAt := inventoryAt + int64(i*setQuickInventoryRowSize)
		binary.LittleEndian.PutUint32(data[rowAt:], handle)
		binary.LittleEndian.PutUint32(data[rowAt+4:], item.quantity)
		binary.LittleEndian.PutUint32(data[rowAt+8:], uint32(i+1))
	}

	// Keep one valid Pouch reference to prove that Quick Items may use the same
	// goods record and that the Pouch representation remains untouched.
	if len(owned) > 0 {
		handle, err := gaItemHandleForGameID(owned[0].gameID)
		if err != nil {
			t.Fatalf("gaItemHandleForGameID: %v", err)
		}
		binary.LittleEndian.PutUint32(data[pairAt+setQuickPouchAt:], handle)
		binary.LittleEndian.PutUint32(
			data[pairAt+setQuickPouchAt+4:], setQuickInventoryEquipBase)
		binary.LittleEndian.PutUint32(
			data[armamentsAt+setQuickPouchTailAt:], owned[0].gameID)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}
	return path, string(platform)
}

func validateQuickTestGameID(gameID uint32) error {
	switch gameID {
	case setQuickTestGameID1, setQuickTestGameID2:
		return nil
	default:
		return fmt.Errorf("item 0x%08X cannot be used as a quick item", gameID)
	}
}

func TestSetQuickItemsWritesBothPlatformsAndReloads(t *testing.T) {
	for _, testCase := range []struct {
		platform Platform
		slot     int
	}{{PlatformPC, 0}, {PlatformPS4, 5}} {
		t.Run(string(testCase.platform), func(t *testing.T) {
			source, platform := writeSetQuickItemsFixture(t, testCase.platform, testCase.slot, []struct {
				gameID   uint32
				quantity uint32
			}{{setQuickTestGameID1, 3}, {setQuickTestGameID2, 2}})

			engine := New()
			loaded, err := engine.LoadSave(source, platform)
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			session := engine.sessions[loaded.SaveSessionID]

			inventoryBefore, err := engine.GetInventory(
				loaded.SaveSessionID, testCase.slot, InventorySectionCommon, 1, 50)
			if err != nil || len(inventoryBefore.Records) != 2 {
				t.Fatalf("GetInventory: %v, len=%d", err, len(inventoryBefore.Records))
			}
			pouchBefore, err := engine.GetPouchItems(loaded.SaveSessionID, testCase.slot)
			if err != nil {
				t.Fatalf("GetPouchItems before: %v", err)
			}
			before := append([]byte(nil), session.snapshot.data...)

			first := inventoryBefore.Records[0].OwnedItemID
			second := inventoryBefore.Records[1].OwnedItemID
			assignments := [10]*string{&first, nil, &second}
			result, err := engine.SetQuickItems(
				loaded.SaveSessionID, testCase.slot, assignments, "0", validateQuickTestGameID)
			if err != nil {
				t.Fatalf("SetQuickItems: %v", err)
			}
			if result.SaveRevision != "1" || result.GameIDs[0] != setQuickTestGameID1 ||
				result.GameIDs[1] != QuickItemEmptyGameID || result.GameIDs[2] != setQuickTestGameID2 {
				t.Fatalf("result = %+v", result)
			}

			quick, err := engine.GetQuickItems(loaded.SaveSessionID, testCase.slot)
			if err != nil {
				t.Fatalf("GetQuickItems: %v", err)
			}
			if quick.ActiveQuick != 4 {
				t.Errorf("activeQuick = %d, want preserved 4", quick.ActiveQuick)
			}
			if quick.Items[0].ItemID == 0 || quick.Items[0].EquipIndex != 0x180 ||
				quick.Items[1] != (QuickItemSlot{ItemID: 0, EquipIndex: 0xFFFFFFFF}) ||
				quick.Items[2].EquipIndex != 0x181 {
				t.Errorf("quick items = %+v", quick.Items)
			}

			pouchAfter, err := engine.GetPouchItems(loaded.SaveSessionID, testCase.slot)
			if err != nil {
				t.Fatalf("GetPouchItems after: %v", err)
			}
			if !reflect.DeepEqual(pouchBefore.Items, pouchAfter.Items) {
				t.Errorf("Pouch changed: before=%+v after=%+v", pouchBefore.Items, pouchAfter.Items)
			}
			inventoryAfter, err := engine.GetInventory(
				loaded.SaveSessionID, testCase.slot, InventorySectionCommon, 1, 50)
			if err != nil {
				t.Fatalf("GetInventory after: %v", err)
			}
			for i := range inventoryBefore.Records {
				beforeRecord := inventoryBefore.Records[i]
				afterRecord := inventoryAfter.Records[i]
				beforeRecord.OwnedItemID = ""
				afterRecord.OwnedItemID = ""
				if !reflect.DeepEqual(beforeRecord, afterRecord) {
					t.Errorf("inventory record %d changed: before=%+v after=%+v",
						i, beforeRecord, afterRecord)
				}
			}

			var slotBase int64
			if testCase.platform == PlatformPS4 {
				slotBase = 0x70 + int64(testCase.slot)*0x280000
			} else {
				slotBase = 0x310 + int64(testCase.slot)*0x280010
			}
			pairAt := slotBase + setQuickAnchorAt + setQuickSectionAt
			countAt := slotBase + setQuickAnchorAt + setQuickProjectileCountAt
			tailAt := countAt + 4 + 17*setQuickProjectileRecordSize + setQuickArmamentsTailAt
			for i := range before {
				if before[i] == session.snapshot.data[i] {
					continue
				}
				offset := int64(i)
				inPairs := offset >= pairAt && offset < pairAt+setQuickActiveAt
				inTail := offset >= tailAt && offset < tailAt+40
				if !inPairs && !inTail {
					t.Errorf("unexpected byte modified at offset 0x%X", offset)
				}
			}

			target := filepath.Join(t.TempDir(), "written-quick-items.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, platform)
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			reloadedQuick, err := reloadedEngine.GetQuickItems(reloaded.SaveSessionID, testCase.slot)
			if err != nil {
				t.Fatalf("GetQuickItems after reload: %v", err)
			}
			if !reflect.DeepEqual(reloadedQuick.Items, quick.Items) || reloadedQuick.ActiveQuick != 4 {
				t.Errorf("reloaded quick items differ: got=%+v want=%+v", reloadedQuick, quick)
			}
		})
	}
}

func TestSetQuickItemsRejectsInvalidPlansWithoutMutation(t *testing.T) {
	source, platform := writeSetQuickItemsFixture(t, PlatformPC, 0, []struct {
		gameID   uint32
		quantity uint32
	}{{setQuickTestGameID1, 1}, {setQuickTestGameID2, 0}, {setQuickTestAccessory, 1}})
	engine := New()
	loaded, err := engine.LoadSave(source, platform)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	session := engine.sessions[loaded.SaveSessionID]
	inventory, err := engine.GetInventory(loaded.SaveSessionID, 0, InventorySectionCommon, 1, 50)
	if err != nil || len(inventory.Records) != 3 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
	}
	valid := inventory.Records[0].OwnedItemID
	zeroQuantity := inventory.Records[1].OwnedItemID
	accessory := inventory.Records[2].OwnedItemID
	storage := session.session.mintOwnedItemID(ownedItemLocator{
		characterID: 0, container: ownedContainerStorage,
		containerSection: StorageSectionCommon, physicalIndex: 0,
	})
	before := append([]byte(nil), session.snapshot.data...)

	tests := []struct {
		name        string
		assignments [10]*string
		revision    string
		want        string
	}{
		{"stale revision", [10]*string{&valid}, "1", "does not match"},
		{"duplicate", [10]*string{&valid, &valid}, "0", "assigned to both"},
		{"zero quantity", [10]*string{&zeroQuantity}, "0", "0 quantity"},
		{"non-goods", [10]*string{&accessory}, "0", "cannot be used as a quick item"},
		{"storage", [10]*string{&storage}, "0", "storage"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := engine.SetQuickItems(
				loaded.SaveSessionID, 0, testCase.assignments, testCase.revision,
				validateQuickTestGameID)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want containing %q", err, testCase.want)
			}
			if result != (SetQuickItemsResult{}) {
				t.Errorf("result = %+v, want zero", result)
			}
			if !bytes.Equal(session.snapshot.data, before) {
				t.Fatal("rejected mutation changed snapshot")
			}
			if session.session.revisionString() != "0" {
				t.Fatalf("revision = %q, want 0", session.session.revisionString())
			}
		})
	}
}

func TestSetQuickItemsRejectsInconsistentExistingState(t *testing.T) {
	source, platform := writeSetQuickItemsFixture(t, PlatformPC, 0, []struct {
		gameID   uint32
		quantity uint32
	}{{setQuickTestGameID1, 1}})
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	pairAt := int64(0x310 + setQuickAnchorAt + setQuickSectionAt)
	handle, err := gaItemHandleForGameID(setQuickTestGameID1)
	if err != nil {
		t.Fatalf("gaItemHandleForGameID: %v", err)
	}
	binary.LittleEndian.PutUint32(data[pairAt:], handle)
	binary.LittleEndian.PutUint32(data[pairAt+4:], 0x180)
	if err := os.WriteFile(source, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	engine := New()
	loaded, err := engine.LoadSave(source, platform)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	session := engine.sessions[loaded.SaveSessionID]
	before := append([]byte(nil), session.snapshot.data...)
	_, err = engine.SetQuickItems(
		loaded.SaveSessionID, 0, [10]*string{}, "0", validateQuickTestGameID)
	if err == nil || !strings.Contains(err.Error(), "inconsistent existing save state") {
		t.Fatalf("error = %v, want inconsistent existing state", err)
	}
	if !bytes.Equal(session.snapshot.data, before) {
		t.Fatal("rejected mutation changed snapshot")
	}
}
