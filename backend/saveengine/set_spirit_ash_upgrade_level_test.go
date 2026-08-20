package saveengine

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"strings"
	"testing"
)

const (
	spiritAshTestCurrent  = uint32(0x40038A44)
	spiritAshTestTarget   = uint32(0x40038A4A)
	spiritAshTestHandle   = uint32(0xB0038A44)
	spiritAshTargetHandle = uint32(0xB0038A4A)
)

func TestSetSpiritAshUpgradeLevelUpdatesInventoryReferencesAndReloads(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			slot := 0
			if platform == PlatformPS4 {
				slot = 5
			}
			engine := New()
			loaded, err := engine.LoadSave(writeAddItemFixture(t, addItemTestFixture{
				platform: platform,
				slot:     slot,
				common: []addItemTestRow{{
					index: 0, handle: spiritAshTestHandle, rawQuantity: 1, acquisition: 7,
				}},
				commonCount: 1,
			}), string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			engine.mutex.Lock()
			snapshot := engine.sessions[loaded.SaveSessionID].snapshot
			slotBase := addItemTestSlotBase(t, platform, slot)
			anchor := slotBase + addItemTestAnchorAt
			quickPairAt := anchor + quickItemsSectionOffset
			pouchPairAt := anchor + pouchItemsSectionOffset
			dynamicAt := anchor + equipmentProjectileCountOffset + 4
			for _, pairAt := range []int64{quickPairAt, pouchPairAt} {
				if err := snapshot.writeAt(pairAt, littleEndianUint32(spiritAshTestHandle)); err != nil {
					engine.mutex.Unlock()
					t.Fatalf("write reference handle: %v", err)
				}
				if err := snapshot.writeAt(pairAt+4,
					littleEndianUint32(removeReferenceInventoryRowBase)); err != nil {
					engine.mutex.Unlock()
					t.Fatalf("write reference row: %v", err)
				}
			}
			for _, tailAt := range []int64{
				dynamicAt + quickItemsTailOffset,
				dynamicAt + pouchItemsTailOffset,
			} {
				if err := snapshot.writeAt(tailAt, littleEndianUint32(spiritAshTestCurrent)); err != nil {
					engine.mutex.Unlock()
					t.Fatalf("write reference ID: %v", err)
				}
			}
			matchmakingAt := anchor + statsMatchmakingWeaponLevelOffset
			if err := snapshot.writeAt(matchmakingAt, []byte{4}); err != nil {
				engine.mutex.Unlock()
				t.Fatalf("write initial matchmaking level: %v", err)
			}
			before := append([]byte(nil), snapshot.data...)
			engine.mutex.Unlock()

			inventory, err := engine.GetInventory(
				loaded.SaveSessionID, slot, InventorySectionCommon, 1, 50)
			if err != nil || len(inventory.Records) != 1 {
				t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
			}
			result, err := engine.SetSpiritAshUpgradeLevel(
				loaded.SaveSessionID, slot, inventory.Records[0].OwnedItemID, 10, "0",
				spiritAshTestCurrent, spiritAshTestTarget)
			if err != nil {
				t.Fatalf("SetSpiritAshUpgradeLevel: %v", err)
			}
			if result.SaveRevision != "1" || result.Container != "inventory" ||
				result.PreviousGameID != spiritAshTestCurrent || result.GameID != spiritAshTestTarget {
				t.Fatalf("result = %+v", result)
			}

			engine.mutex.Lock()
			after := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)
			engine.mutex.Unlock()
			inventoryAt := anchor + addItemTestCommonAt
			for name, at := range map[string]int64{
				"inventory handle": inventoryAt,
				"quick handle":     quickPairAt,
				"pouch handle":     pouchPairAt,
			} {
				if got := binary.LittleEndian.Uint32(after[at:]); got != spiritAshTargetHandle {
					t.Errorf("%s = 0x%08X", name, got)
				}
			}
			for name, at := range map[string]int64{
				"quick ID": dynamicAt + quickItemsTailOffset,
				"pouch ID": dynamicAt + pouchItemsTailOffset,
			} {
				if got := binary.LittleEndian.Uint32(after[at:]); got != spiritAshTestTarget {
					t.Errorf("%s = 0x%08X", name, got)
				}
			}
			gaItemDataAt := anchor + addItemTestGaItemDataAt
			if got := binary.LittleEndian.Uint32(after[gaItemDataAt:]); got != 1 {
				t.Fatalf("GaItemData count = %d", got)
			}
			if got := binary.LittleEndian.Uint32(after[gaItemDataAt+8:]); got != spiritAshTestTarget {
				t.Fatalf("GaItemData ID = 0x%08X", got)
			}

			allowed := [][2]int64{
				{inventoryAt, inventoryAt + 4},
				{quickPairAt, quickPairAt + 4},
				{pouchPairAt, pouchPairAt + 4},
				{dynamicAt + quickItemsTailOffset, dynamicAt + quickItemsTailOffset + 4},
				{dynamicAt + pouchItemsTailOffset, dynamicAt + pouchItemsTailOffset + 4},
				{gaItemDataAt, gaItemDataAt + 4},
				{gaItemDataAt + 8, gaItemDataAt + 16},
			}
			for offset := range before {
				if before[offset] == after[offset] {
					continue
				}
				inside := false
				for _, span := range allowed {
					inside = inside || int64(offset) >= span[0] && int64(offset) < span[1]
				}
				if !inside {
					t.Fatalf("unexpected byte modified at 0x%X", offset)
				}
			}

			target := filepath.Join(t.TempDir(), "spirit-ash-upgrade.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform))
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			reloadedInventory, err := reloadedEngine.GetInventory(
				reloaded.SaveSessionID, slot, InventorySectionCommon, 1, 50)
			if err != nil || reloadedInventory.Records[0].GaItemHandle != spiritAshTargetHandle {
				t.Fatalf("reloaded inventory: %v, records=%+v", err, reloadedInventory.Records)
			}

			reloadedEngine.mutex.Lock()
			reloadedAnchor, err := findStatsAnchor(
				reloadedEngine.sessions[reloaded.SaveSessionID].snapshot, platform, slot)
			if err != nil {
				reloadedEngine.mutex.Unlock()
				t.Fatalf("findStatsAnchor reloaded (%s): %v", platform, err)
			}
			reloadedMatchmaking, err := reloadedEngine.sessions[reloaded.SaveSessionID].snapshot.readAt(
				reloadedAnchor+statsMatchmakingWeaponLevelOffset, 1)
			reloadedEngine.mutex.Unlock()
			if err != nil {
				t.Fatalf("read matchmaking level after reload (%s): %v", platform, err)
			}
			if reloadedMatchmaking[0] != 4 {
				t.Errorf("reloaded %s matchmaking level after Spirit Ash +10 = %d, want preserved 4",
					platform, reloadedMatchmaking[0])
			}
		})
	}
}

func TestSetSpiritAshUpgradeLevelRejectsInconsistentInventoryReference(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, addItemTestFixture{
		platform: PlatformPC,
		common: []addItemTestRow{{
			index: 0, handle: spiritAshTestHandle, rawQuantity: 1, acquisition: 7,
		}},
		commonCount: 1,
	}), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, 0, InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	engine.mutex.Lock()
	snapshot := engine.sessions[loaded.SaveSessionID].snapshot
	pairAt := addItemTestSlotBase(t, PlatformPC, 0) + addItemTestAnchorAt + quickItemsSectionOffset
	if err := snapshot.writeAt(pairAt, littleEndianUint32(spiritAshTestHandle)); err != nil {
		engine.mutex.Unlock()
		t.Fatalf("write reference handle: %v", err)
	}
	if err := snapshot.writeAt(
		pairAt+4, littleEndianUint32(removeReferenceInventoryRowBase)); err != nil {
		engine.mutex.Unlock()
		t.Fatalf("write reference row: %v", err)
	}
	before := append([]byte(nil), snapshot.data...)
	engine.mutex.Unlock()

	_, err = engine.SetSpiritAshUpgradeLevel(
		loaded.SaveSessionID, 0, inventory.Records[0].OwnedItemID, 10, "0",
		spiritAshTestCurrent, spiritAshTestTarget)
	if err == nil || !strings.Contains(err.Error(), "inconsistent Spirit Ash representations") {
		t.Fatalf("error = %v", err)
	}
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if !bytes.Equal(before, engine.sessions[loaded.SaveSessionID].snapshot.data) ||
		engine.sessions[loaded.SaveSessionID].session.revisionString() != "0" {
		t.Fatal("rejected mutation changed snapshot or revision")
	}
}

func TestSetSpiritAshUpgradeLevelSupportsStorageCommon(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, addItemTestFixture{
		platform: PlatformPC,
		storage: []addItemTestRow{{
			index: 0, handle: spiritAshTestHandle, rawQuantity: 1, acquisition: 7,
		}},
		storageCount: 1,
	}), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	storage, err := engine.GetStorage(
		loaded.SaveSessionID, 0, StorageSectionCommon, 1, 50)
	if err != nil || len(storage.Records) != 1 {
		t.Fatalf("GetStorage: %v, len=%d", err, len(storage.Records))
	}
	result, err := engine.SetSpiritAshUpgradeLevel(
		loaded.SaveSessionID, 0, storage.Records[0].OwnedItemID, 10, "0",
		spiritAshTestCurrent, spiritAshTestTarget)
	if err != nil {
		t.Fatalf("SetSpiritAshUpgradeLevel: %v", err)
	}
	if result.Container != "storage" || result.GameID != spiritAshTestTarget {
		t.Fatalf("result = %+v", result)
	}
	updated, err := engine.GetStorage(
		loaded.SaveSessionID, 0, StorageSectionCommon, 1, 50)
	if err != nil || updated.Records[0].GaItemHandle != spiritAshTargetHandle {
		t.Fatalf("updated storage: %v, records=%+v", err, updated.Records)
	}
}
